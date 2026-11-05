package ffmpeg

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"image/gif"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// Quality selects the GIF palette strategy.
type Quality string

const (
	QualityHigh     Quality = "high"     // diff-based palette, full dithering
	QualityBalanced Quality = "balanced" // diff-based palette, default dithering
	QualityCompact  Quality = "compact"  // full palette, ordered dithering, smaller files
)

// Stage names reported through GIFProgress.Stage.
type Stage string

const (
	StagePalette Stage = "palette"
	StageEncode  Stage = "encode"
)

// GIFRequest asks for one video→GIF conversion.
type GIFRequest struct {
	Input       string  // source video path
	Output      string  // target GIF path; empty → temp file, caller owns its lifecycle
	StartSec    float64 // trim start; 0 = from the beginning
	DurationSec float64 // trim length; 0 = to the end
	TotalSec    float64 // full source duration, only used for progress
	FPS         int     // 0 → 15, clamped to 5..30, capped by SourceFPS
	MaxWidth    int     // 0 → 480, clamped to 100..1920, capped by SourceWidth
	Quality     Quality // invalid/empty → QualityBalanced
	SourceFPS   float64 // source frame rate; when > 0, FPS never exceeds it
	SourceWidth int     // source width; when > 0, MaxWidth never exceeds it (no upscaling)
}

// GIFProgress reports conversion progress; Percent is the overall 0..100.
type GIFProgress struct {
	Stage   Stage   `json:"stage"`
	Percent float64 `json:"percent"`
}

// GIFResult reports a finished conversion and validates the output.
type GIFResult struct {
	OutputPath string `json:"outputPath"`
	Width      int    `json:"width"`
	Height     int    `json:"height"`
	Frames     int    `json:"frames"`
	Loops      bool   `json:"loops"` // loops forever
	SizeBytes  int64  `json:"sizeBytes"`
}

// ConvertGIF runs the two-pass palette pipeline (palettegen, then
// paletteuse) and returns the validated GIF. onProgress, when non-nil,
// receives overall progress updates.
func (c *Client) ConvertGIF(ctx context.Context, req GIFRequest, onProgress func(GIFProgress)) (GIFResult, error) {
	if req.Input == "" {
		return GIFResult{}, errors.New("GIFRequest.Input 为空")
	}
	opts := normalizeGIF(req)

	palette, err := os.CreateTemp("", "m-tools-palette-*.png")
	if err != nil {
		return GIFResult{}, fmt.Errorf("创建临时调色板文件: %w", err)
	}
	palettePath := palette.Name()
	palette.Close()
	defer os.Remove(palettePath)

	output := opts.Output
	if output == "" {
		tmp, err := os.CreateTemp("", "m-tools-*.gif")
		if err != nil {
			return GIFResult{}, fmt.Errorf("创建临时输出文件: %w", err)
		}
		tmp.Close()
		output = tmp.Name()
	}

	trim := trimArgs(opts)
	chain := gifFilterChain(opts)

	// Pass 1: analyze a global palette (weighted ~30% of the total work).
	paletteFilter := chain + ",palettegen"
	if opts.Quality != QualityCompact {
		paletteFilter += "=stats_mode=diff"
	}
	pass1 := append([]string{"-y", "-v", "error", "-progress", "pipe:1", "-nostats"},
		trim...)
	pass1 = append(pass1, "-i", opts.Input, "-vf", paletteFilter, palettePath)
	if err := runFFmpeg(ctx, c.ffmpeg, pass1, stageReporter(onProgress, StagePalette, 0, 30, opts)); err != nil {
		return GIFResult{}, err
	}

	// Pass 2: apply the palette (weighted ~70%).
	dither := "sierra2_4a"
	if opts.Quality == QualityCompact {
		dither = "bayer:bayer_scale=5"
	}
	complexFilter := fmt.Sprintf("%s[x];[x][1:v]paletteuse=%s", chain, dither)
	pass2 := append([]string{"-y", "-v", "error", "-progress", "pipe:1", "-nostats"},
		trim...)
	pass2 = append(pass2, "-i", opts.Input, "-i", palettePath,
		"-filter_complex", complexFilter, "-loop", "0", "-f", "gif", output)
	if err := runFFmpeg(ctx, c.ffmpeg, pass2, stageReporter(onProgress, StageEncode, 30, 100, opts)); err != nil {
		return GIFResult{}, err
	}

	meta, err := readGIFMeta(output)
	if err != nil {
		return GIFResult{}, fmt.Errorf("校验输出 GIF: %w", err)
	}
	meta.OutputPath = output
	return meta, nil
}

// normalizedGIF is GIFRequest with defaults applied and values clamped.
type normalizedGIF struct {
	GIFRequest
	FPS      int
	MaxWidth int
	Quality  Quality
}

func normalizeGIF(req GIFRequest) normalizedGIF {
	fps := req.FPS
	if fps <= 0 {
		fps = 15
	}
	fps = clampInt(fps, 5, 30)

	width := req.MaxWidth
	if width <= 0 {
		width = 480
	}
	width = clampInt(width, 100, 1920)

	// Never exceed the source: duplicated frames and upscaling only grow
	// the file or blur the picture.
	if req.SourceFPS > 0 && fps > int(req.SourceFPS) {
		fps = max(5, int(req.SourceFPS))
	}
	if req.SourceWidth > 0 && width > req.SourceWidth {
		width = req.SourceWidth
	}

	quality := req.Quality
	switch quality {
	case QualityHigh, QualityBalanced, QualityCompact:
	default:
		quality = QualityBalanced
	}

	if req.StartSec < 0 {
		req.StartSec = 0
	}
	if req.DurationSec < 0 {
		req.DurationSec = 0
	}
	return normalizedGIF{GIFRequest: req, FPS: fps, MaxWidth: width, Quality: quality}
}

// gifFilterChain builds the shared video filter chain (fps + scale).
func gifFilterChain(opts normalizedGIF) string {
	return fmt.Sprintf("fps=%d,scale=%d:-2:flags=lanczos", opts.FPS, opts.MaxWidth)
}

// trimArgs returns the -ss/-t arguments for the configured trim window.
func trimArgs(opts normalizedGIF) []string {
	var args []string
	if opts.StartSec > 0 {
		args = append(args, "-ss", secs(opts.StartSec))
	}
	if opts.DurationSec > 0 {
		args = append(args, "-t", secs(opts.DurationSec))
	}
	return args
}

// stageReporter maps one ffmpeg pass's 0..1 progress into an overall range.
func stageReporter(onProgress func(GIFProgress), stage Stage, from, to float64, opts normalizedGIF) func(key, val string) {
	if onProgress == nil {
		return nil
	}
	total := opts.DurationSec
	if total <= 0 {
		total = opts.TotalSec
	}
	return func(key, val string) {
		switch key {
		case "out_time_us", "out_time_ms": // both are microseconds despite the name
			if total <= 0 {
				return
			}
			us, err := strconv.ParseFloat(val, 64)
			if err != nil {
				return
			}
			pct := clamp(us/1e6/total, 0, 1)
			onProgress(GIFProgress{Stage: stage, Percent: from + pct*(to-from)})
		case "progress":
			if val == "end" {
				onProgress(GIFProgress{Stage: stage, Percent: to})
			}
		}
	}
}

// readGIFMeta decodes enough of path to validate it and report dimensions,
// frame count and loop behavior.
func readGIFMeta(path string) (GIFResult, error) {
	fi, err := os.Open(path)
	if err != nil {
		return GIFResult{}, err
	}
	defer fi.Close()
	g, err := gif.DecodeAll(fi)
	if err != nil {
		return GIFResult{}, err
	}
	st, err := fi.Stat()
	if err != nil {
		return GIFResult{}, err
	}
	return GIFResult{
		Width:     g.Config.Width,
		Height:    g.Config.Height,
		Frames:    len(g.Image),
		Loops:     g.LoopCount == 0, // 0 means loop forever (ffmpeg -loop 0)
		SizeBytes: st.Size(),
	}, nil
}

func runFFmpeg(ctx context.Context, ffmpeg string, args []string, onLine func(key, val string)) error {
	cmd := exec.CommandContext(ctx, ffmpeg, args...)
	hideConsole(cmd)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("启动 ffmpeg: %w", err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动 ffmpeg: %w", err)
	}
	if onLine != nil {
		sc := bufio.NewScanner(stdout)
		for sc.Scan() {
			if k, v, ok := strings.Cut(sc.Text(), "="); ok {
				onLine(strings.TrimSpace(k), strings.TrimSpace(v))
			}
		}
	}
	if err := cmd.Wait(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("ffmpeg: %v\n%s", err, tail(stderr.String()))
	}
	return nil
}

func secs(v float64) string {
	return strconv.FormatFloat(v, 'f', 3, 64)
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func tail(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 800 {
		return s[len(s)-800:]
	}
	return s
}
