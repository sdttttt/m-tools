package webp

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

// drawRect returns a 2x2 RGBA image: an opaque and a half-transparent pixel,
// so lossy alpha paths have something to carry.
func drawRect() *image.NRGBA {
	m := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	m.Set(0, 0, color.RGBA{R: 255, A: 255})
	m.Set(1, 0, color.RGBA{R: 0, G: 255, A: 128})
	m.Set(0, 1, color.RGBA{B: 255, A: 255})
	m.Set(1, 1, color.RGBA{A: 255})
	return m
}

func encodePNG(t *testing.T, m image.Image) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, m); err != nil {
		t.Fatalf("png.Encode: %v", err)
	}
	return buf.Bytes()
}

func TestSupported(t *testing.T) {
	if Supported() != webpBuilt {
		t.Fatalf("Supported() = %v, want %v", Supported(), webpBuilt)
	}
}

func TestEncodeDecodeRoundTrip(t *testing.T) {
	if !Supported() {
		t.Skip("webp codec not compiled in (build without -tags with_webp)")
	}
	src := drawRect()

	var buf bytes.Buffer
	if err := Encode(&buf, src, 90); err != nil {
		t.Fatalf("Encode: %v", err)
	}

	cfg, err := DecodeConfig(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("DecodeConfig: %v", err)
	}
	if cfg.Width != 2 || cfg.Height != 2 {
		t.Fatalf("DecodeConfig = %dx%d, want 2x2", cfg.Width, cfg.Height)
	}

	img, err := Decode(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	bounds := img.Bounds()
	if bounds.Dx() != 2 || bounds.Dy() != 2 {
		t.Fatalf("Decode bounds = %v, want 2x2", bounds)
	}

	// Lossy at quality 90: colors drift, alpha direction is what matters.
	px := color.NRGBAModel.Convert(img.At(0, 0)).(color.NRGBA)
	if px.A != 255 {
		t.Fatalf("opaque pixel alpha = %d, want 255", px.A)
	}
	if px := color.NRGBAModel.Convert(img.At(1, 1)).(color.NRGBA); px.A != 255 {
		t.Fatalf("opaque pixel alpha = %d, want 255", px.A)
	}
}

func TestDecodeGarbage(t *testing.T) {
	if !Supported() {
		t.Skip("webp codec not compiled in")
	}
	if _, err := Decode(bytes.NewReader(encodePNG(t, drawRect()))); err == nil {
		t.Fatal("Decode(png bytes) succeeded, want error")
	}
}

func TestStubErrors(t *testing.T) {
	if Supported() {
		t.Skip("webp codec compiled in; stub errors not reachable")
	}
	if _, err := Decode(bytes.NewReader(encodePNG(t, drawRect()))); err == nil {
		t.Fatal("stub Decode succeeded, want ErrUnsupported")
	}
	var buf bytes.Buffer
	if err := Encode(&buf, drawRect(), 90); err == nil {
		t.Fatal("stub Encode succeeded, want ErrUnsupported")
	}
	if _, err := DecodeConfig(bytes.NewReader([]byte("RIFF"))); err == nil {
		t.Fatal("stub DecodeConfig succeeded, want ErrUnsupported")
	}
	Register() // must stay a no-op, not panic
}
