package ffmpeg

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// VideoInfo describes a video file as reported by ffprobe.
type VideoInfo struct {
	Path        string  `json:"path"`
	DurationSec float64 `json:"durationSec"`
	Width       int     `json:"width"`
	Height      int     `json:"height"`
	FPS         float64 `json:"fps"`
	VideoCodec  string  `json:"videoCodec"`
	PixFmt      string  `json:"pixFmt"`
	SizeBytes   int64   `json:"sizeBytes"`
}

type probeOutput struct {
	Streams []struct {
		Width        int    `json:"width"`
		Height       int    `json:"height"`
		CodecName    string `json:"codec_name"`
		PixFmt       string `json:"pix_fmt"`
		AvgFrameRate string `json:"avg_frame_rate"`
		RFrameRate   string `json:"r_frame_rate"`
		Duration     string `json:"duration"`
	} `json:"streams"`
	Format struct {
		Duration string `json:"duration"`
		Size     string `json:"size"`
	} `json:"format"`
}

// Probe reads metadata for the first video stream of path.
func (c *Client) Probe(ctx context.Context, path string) (VideoInfo, error) {
	cmd := exec.CommandContext(ctx, c.ffprobe,
		"-v", "error",
		"-print_format", "json",
		"-select_streams", "v:0",
		"-show_streams", "-show_format",
		path,
	)
	var out, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &stderr
	if err := cmd.Run(); err != nil {
		return VideoInfo{}, fmt.Errorf("ffprobe: %v\n%s", err, tail(stderr.String()))
	}

	var po probeOutput
	if err := json.Unmarshal(out.Bytes(), &po); err != nil {
		return VideoInfo{}, fmt.Errorf("ffprobe 输出解析失败: %w", err)
	}
	if len(po.Streams) == 0 {
		return VideoInfo{}, fmt.Errorf("文件中没有视频流: %s", path)
	}
	st := po.Streams[0]

	info := VideoInfo{
		Path:       path,
		Width:      st.Width,
		Height:     st.Height,
		VideoCodec: st.CodecName,
		PixFmt:     st.PixFmt,
		FPS:        parseRatio(st.AvgFrameRate),
	}
	if info.FPS == 0 {
		info.FPS = parseRatio(st.RFrameRate)
	}
	info.DurationSec = parseFloat(st.Duration)
	if info.DurationSec == 0 {
		info.DurationSec = parseFloat(po.Format.Duration)
	}
	info.SizeBytes = int64(parseFloat(po.Format.Size))
	if info.SizeBytes == 0 {
		if fi, err := os.Stat(path); err == nil {
			info.SizeBytes = fi.Size()
		}
	}
	return info, nil
}

// parseRatio parses an ffmpeg rate such as "30000/1001" or "15".
func parseRatio(s string) float64 {
	num, den, ok := strings.Cut(s, "/")
	n := parseFloat(num)
	if !ok {
		return n
	}
	d := parseFloat(den)
	if d == 0 {
		return 0
	}
	return n / d
}

func parseFloat(s string) float64 {
	v, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return v
}
