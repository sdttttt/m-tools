package ffmpeg

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestNormalizeGIF(t *testing.T) {
	got := normalizeGIF(GIFRequest{FPS: 99, MaxWidth: 50, Quality: "nope", StartSec: -1, DurationSec: -3})
	if got.FPS != 30 {
		t.Errorf("FPS clamp: got %d, want 30", got.FPS)
	}
	if got.MaxWidth != 100 {
		t.Errorf("MaxWidth clamp: got %d, want 100", got.MaxWidth)
	}
	if got.Quality != QualityBalanced {
		t.Errorf("Quality default: got %q, want %q", got.Quality, QualityBalanced)
	}
	if got.StartSec != 0 || got.DurationSec != 0 {
		t.Errorf("trim clamp: got start=%v duration=%v, want 0/0", got.StartSec, got.DurationSec)
	}

	got = normalizeGIF(GIFRequest{})
	if got.FPS != 15 || got.MaxWidth != 480 || got.Quality != QualityBalanced {
		t.Errorf("defaults: got fps=%d width=%d quality=%q, want 15/480/balanced",
			got.FPS, got.MaxWidth, got.Quality)
	}
}

func TestNormalizeGIFSourceCaps(t *testing.T) {
	got := normalizeGIF(GIFRequest{FPS: 30, MaxWidth: 1920, SourceFPS: 12.5, SourceWidth: 640})
	if got.FPS != 12 {
		t.Errorf("FPS source cap: got %d, want 12", got.FPS)
	}
	if got.MaxWidth != 640 {
		t.Errorf("MaxWidth source cap: got %d, want 640", got.MaxWidth)
	}
	// Requests below the source pass through.
	got = normalizeGIF(GIFRequest{FPS: 8, MaxWidth: 320, SourceFPS: 24, SourceWidth: 1080})
	if got.FPS != 8 || got.MaxWidth != 320 {
		t.Errorf("below-source passthrough: got fps=%d width=%d, want 8/320", got.FPS, got.MaxWidth)
	}
}

func TestGifFilterChain(t *testing.T) {
	got := gifFilterChain(normalizeGIF(GIFRequest{FPS: 12, MaxWidth: 320}))
	want := "fps=12,scale=320:-2:flags=lanczos"
	if got != want {
		t.Errorf("gifFilterChain: got %q, want %q", got, want)
	}
}

func TestParseRatio(t *testing.T) {
	cases := []struct {
		in   string
		want float64
	}{
		{"30000/1001", 29.97002997002997},
		{"15", 15},
		{"0/0", 0},
		{"", 0},
	}
	for _, c := range cases {
		if got := parseRatio(c.in); (c.want == 0 && got != 0) || (c.want != 0 && got != c.want) {
			t.Errorf("parseRatio(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

// TestConvertGIFEndToEnd runs a real conversion against ffmpeg from PATH (or
// $MTOOLS_FFMPEG_DIR): it synthesizes a 2s test video, converts it and
// validates the GIF result. Skipped when ffmpeg is unavailable.
func TestConvertGIFEndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("short mode")
	}
	client, err := New()
	if err != nil {
		t.Skipf("ffmpeg unavailable: %v", err)
	}

	dir := t.TempDir()
	input := filepath.Join(dir, "input.mp4")
	makeTestVideo(t, client, input)

	info, err := client.Probe(context.Background(), input)
	if err != nil {
		t.Fatalf("Probe: %v", err)
	}
	if info.Width != 320 || info.Height != 240 {
		t.Errorf("Probe size: got %dx%d, want 320x240", info.Width, info.Height)
	}

	var last GIFProgress
	output := filepath.Join(dir, "out.gif")
	res, err := client.ConvertGIF(context.Background(), GIFRequest{
		Input:       input,
		Output:      output,
		TotalSec:    info.DurationSec,
		FPS:         10,
		MaxWidth:    160,
		Quality:     QualityHigh,
		StartSec:    0.5,
		DurationSec: 1,
	}, func(p GIFProgress) { last = p })
	if err != nil {
		t.Fatalf("ConvertGIF: %v", err)
	}
	if res.OutputPath != output {
		t.Fatalf("ConvertGIF output: got %q, want %q", res.OutputPath, output)
	}
	if last.Percent != 100 {
		t.Errorf("final progress: got %.1f, want 100", last.Percent)
	}
	if last.Stage != StageEncode {
		t.Errorf("final stage: got %q, want %q", last.Stage, StageEncode)
	}
	if res.Width != 160 {
		t.Errorf("result size: %dx%d, want width 160", res.Width, res.Height)
	}
	// 1s trimmed at fps=10 → ~10 frames; be lenient for rounding.
	if res.Frames < 5 || res.Frames > 15 {
		t.Errorf("frame count: got %d, want ~10", res.Frames)
	}
	if !res.Loops {
		t.Error("GIF should loop forever (-loop 0)")
	}
	if res.SizeBytes == 0 {
		t.Error("result size bytes: got 0")
	}

	// Empty Input is rejected before any process is started.
	if _, err := client.ConvertGIF(context.Background(), GIFRequest{}, nil); err == nil {
		t.Error("empty Input: want error, got nil")
	}

	// Default output path lands in a temp file owned by the caller.
	tmpRes, err := client.ConvertGIF(context.Background(), GIFRequest{Input: input, FPS: 5}, nil)
	if err != nil {
		t.Fatalf("ConvertGIF (default output): %v", err)
	}
	if fi, err := os.Stat(tmpRes.OutputPath); err != nil || fi.Size() == 0 {
		t.Errorf("temp output missing/empty: %v", err)
	}
	os.Remove(tmpRes.OutputPath)
}

// TestConvertGIFTempOutputCleanedOnFailure converts an input ffmpeg cannot
// decode and checks the temp output ConvertGIF created for itself is removed
// along with the failure. Only files that appear during the failed call
// count, so leftovers from other runs on this shared temp dir don't matter.
func TestConvertGIFTempOutputCleanedOnFailure(t *testing.T) {
	if testing.Short() {
		t.Skip("short mode")
	}
	client, err := New()
	if err != nil {
		t.Skipf("ffmpeg unavailable: %v", err)
	}

	before := map[string]bool{}
	matches, err := filepath.Glob(filepath.Join(os.TempDir(), "m-tools-*.gif"))
	if err != nil {
		t.Fatalf("glob temp outputs: %v", err)
	}
	for _, m := range matches {
		before[m] = true
	}

	dir := t.TempDir()
	input := filepath.Join(dir, "input.mp4")
	if err := os.WriteFile(input, []byte("not a video"), 0o644); err != nil {
		t.Fatalf("write bogus input: %v", err)
	}

	if _, err := client.ConvertGIF(context.Background(), GIFRequest{Input: input}, nil); err == nil {
		t.Fatal("bogus input: want error, got nil")
	}
	if _, err := os.Stat(input); err != nil {
		t.Fatalf("bogus input itself was touched: %v", err)
	}

	matches, err = filepath.Glob(filepath.Join(os.TempDir(), "m-tools-*.gif"))
	if err != nil {
		t.Fatalf("glob temp outputs: %v", err)
	}
	for _, m := range matches {
		if !before[m] {
			t.Errorf("leftover temp output after failure: %s", m)
		}
	}
}

// makeTestVideo synthesizes a 320x240 10fps 2s testsrc clip with ffmpeg.
func makeTestVideo(t *testing.T, client *Client, output string) {
	t.Helper()
	cmd := exec.Command(client.ffmpeg, "-y", "-v", "error",
		"-f", "lavfi", "-i", "testsrc=duration=2:size=320x240:rate=10",
		"-pix_fmt", "yuv420p", output)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("synthesize test video: %v\n%s", err, out)
	}
}
