package ffmpeg

import (
	"bytes"
	"context"
	"image/jpeg"
	"testing"
)

// TestThumbnailsEndToEnd synthesizes a test clip and checks the filmstrip
// extraction. Skipped when ffmpeg is unavailable.
func TestThumbnailsEndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("short mode")
	}
	client, err := New()
	if err != nil {
		t.Skipf("ffmpeg unavailable: %v", err)
	}

	dir := t.TempDir()
	input := dir + "/input.mp4"
	makeTestVideo(t, client, input)

	thumbs, err := client.Thumbnails(context.Background(), input, 6, 64, 2)
	if err != nil {
		t.Fatalf("Thumbnails: %v", err)
	}
	if len(thumbs) < 4 || len(thumbs) > 6 {
		t.Fatalf("frame count: got %d, want ~6", len(thumbs))
	}
	for i, data := range thumbs {
		cfg, err := jpeg.DecodeConfig(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("frame %d is not a JPEG: %v", i, err)
		}
		if cfg.Width != 64 {
			t.Errorf("frame %d width: got %d, want 64", i, cfg.Width)
		}
	}

	if _, err := client.Thumbnails(context.Background(), input, 6, 64, 0); err == nil {
		t.Error("zero duration: want error, got nil")
	}
	if _, err := client.Thumbnails(context.Background(), "", 6, 64, 2); err == nil {
		t.Error("empty input: want error, got nil")
	}
}
