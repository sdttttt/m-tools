package imgconv

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// solid returns a 8x8 NRGBA filled with one color.
func solid(c color.Color) *image.NRGBA {
	m := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			m.Set(x, y, c)
		}
	}
	return m
}

func encodeToFile(t *testing.T, dir, name string, m image.Image, encode func(*os.File, image.Image) error) string {
	t.Helper()
	p := filepath.Join(dir, name)
	f, err := os.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := encode(f, m); err != nil {
		t.Fatal(err)
	}
	return p
}

func writeTestPNG(t *testing.T, dir, name string, c color.Color) string {
	return encodeToFile(t, dir, name, solid(c), func(f *os.File, m image.Image) error {
		return png.Encode(f, m)
	})
}

func writeTestJPEG(t *testing.T, dir, name string) string {
	return encodeToFile(t, dir, name, solid(color.RGBA{R: 255, G: 0, B: 0, A: 255}), func(f *os.File, m image.Image) error {
		return jpeg.Encode(f, m, nil)
	})
}

// writeTestAnimatedGIF writes a two-frame GIF (distinct frame colors) and
// returns its path.
func writeTestAnimatedGIF(t *testing.T, dir, name string, frames ...color.RGBA) string {
	t.Helper()
	out := &gif.GIF{}
	for _, c := range frames {
		palette := color.Palette{c, color.Black}
		m := image.NewPaletted(image.Rect(0, 0, 8, 8), palette)
		for y := 0; y < 8; y++ {
			for x := 0; x < 8; x++ {
				m.SetColorIndex(x, y, 0)
			}
		}
		out.Image = append(out.Image, m)
		out.Delay = append(out.Delay, 10)
	}
	p := filepath.Join(dir, name)
	f, err := os.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := gif.EncodeAll(f, out); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestProbeFormats(t *testing.T) {
	dir := t.TempDir()
	s := &Service{}

	pngPath := writeTestPNG(t, dir, "a.png", color.RGBA{R: 255, A: 255})
	info, err := s.Probe(pngPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Format != "png" || !info.Supported || info.Width != 8 || info.Height != 8 {
		t.Fatalf("png probe = %+v", info)
	}
	if info.HasAlpha {
		t.Fatal("opaque png reported HasAlpha")
	}

	jpgPath := writeTestJPEG(t, dir, "b.jpg")
	info, err = s.Probe(jpgPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Format != "jpg" || !info.Supported || info.HasAlpha {
		t.Fatalf("jpg probe = %+v", info)
	}

	gifPath := writeTestAnimatedGIF(t, dir, "c.gif", color.RGBA{R: 255, A: 255}, color.RGBA{B: 255, A: 255})
	info, err = s.Probe(gifPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Format != "gif" || !info.Supported {
		t.Fatalf("gif probe = %+v", info)
	}

	// A webp file: supported builds probe it, stub builds report the tag hint.
	webpPath := filepath.Join(dir, "d.webp")
	if err := os.WriteFile(webpPath, testWebPBytes(t), 0o644); err != nil {
		t.Fatal(err)
	}
	info, err = s.Probe(webpPath)
	if err != nil {
		t.Fatal(err)
	}
	if webpSupported() {
		if info.Format != "webp" || !info.Supported {
			t.Fatalf("webp probe = %+v", info)
		}
	} else {
		if info.Supported {
			t.Fatal("stub build reported webp as supported")
		}
		if info.SkipReason != errWebpUnsupported.Error() {
			t.Fatalf("webp skip reason = %q", info.SkipReason)
		}
	}

	// Garbage: unsupported, but Probe never errors — the page shows rows.
	junk := filepath.Join(dir, "e.png")
	if err := os.WriteFile(junk, []byte("not an image at all"), 0o644); err != nil {
		t.Fatal(err)
	}
	info, err = s.Probe(junk)
	if err != nil {
		t.Fatal(err)
	}
	if info.Supported || info.SkipReason == "" {
		t.Fatalf("junk probe = %+v", info)
	}
}

func TestConvertMatrix(t *testing.T) {
	dir := t.TempDir()
	s := &Service{}
	ctx := context.Background()

	src := writeTestPNG(t, dir, "img.png", color.RGBA{R: 255, G: 0, B: 0, A: 255})

	// Every target round-trips the source size.
	for _, target := range []string{"jpg", "png", "gif", "bmp", "tiff"} {
		res, err := s.Convert(ctx, ConvertRequest{
			Path: src, TargetFormat: target, Quality: "high",
		})
		if err != nil {
			t.Fatalf("png→%s: %v", target, err)
		}
		if res.Width != 8 || res.Height != 8 {
			t.Fatalf("png→%s size = %dx%d", target, res.Width, res.Height)
		}
		if res.SizeBytes == 0 || res.OutputPath == "" {
			t.Fatalf("png→%s result = %+v", target, res)
		}
		if _, err := os.Stat(res.OutputPath); err != nil {
			t.Fatalf("png→%s output missing: %v", target, err)
		}
		os.Remove(res.OutputPath)
	}

	// Unknown target.
	if _, err := s.Convert(ctx, ConvertRequest{Path: src, TargetFormat: "heic"}); err == nil {
		t.Fatal("unknown target accepted")
	}

	// JPEG source to png (decode via table, re-encode lossless).
	jpg := writeTestJPEG(t, dir, "photo.jpg")
	res, err := s.Convert(ctx, ConvertRequest{Path: jpg, TargetFormat: "png"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Format != "png" || res.OutputPath != filepath.Join(dir, "photo.png") {
		t.Fatalf("jpg→png = %+v", res)
	}
}

func TestConvertAlphaFlattenedForJPEG(t *testing.T) {
	dir := t.TempDir()
	s := &Service{}

	// RGBA source: top-left opaque red, rest fully transparent.
	m := solid(color.NRGBA{A: 0})
	m.Set(0, 0, color.NRGBA{R: 255, A: 255})
	p := encodeToFile(t, dir, "alpha.png", m, func(f *os.File, img image.Image) error {
		return png.Encode(f, img)
	})

	info, err := s.Probe(p)
	if err != nil {
		t.Fatal(err)
	}
	if !info.HasAlpha {
		t.Fatal("transparent png not flagged HasAlpha")
	}

	res, err := s.Convert(context.Background(), ConvertRequest{Path: p, TargetFormat: "jpg"})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(res.OutputPath)
	if err != nil {
		t.Fatal(err)
	}
	out, err := jpeg.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	// The transparent area must have become opaque white, not black.
	r, g, b, a := out.At(4, 4).RGBA()
	if a != 0xffff || r < 0xf000 || g < 0xf000 || b < 0xf000 {
		t.Fatalf("flattened pixel = (%d,%d,%d,%d), want opaque white", r, g, b, a)
	}
}

func TestConvertAnimatedGIFTakesFirstFrame(t *testing.T) {
	dir := t.TempDir()
	s := &Service{}
	gifPath := writeTestAnimatedGIF(t, dir, "anim.gif",
		color.RGBA{R: 255, A: 255}, color.RGBA{B: 255, A: 255})

	res, err := s.Convert(context.Background(), ConvertRequest{Path: gifPath, TargetFormat: "png"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Width != 8 || res.Height != 8 {
		t.Fatalf("anim→png size = %dx%d", res.Width, res.Height)
	}
	// Exactly one frame of output (the png holds the first frame's color).
	data, err := os.ReadFile(res.OutputPath)
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	r, _, _, _ := img.At(0, 0).RGBA()
	if r == 0 {
		t.Fatal("first frame lost: pixel came out black")
	}
}

func TestConvertScalesDownOnly(t *testing.T) {
	dir := t.TempDir()
	s := &Service{}
	ctx := context.Background()
	src := writeTestPNG(t, dir, "big.png", color.RGBA{G: 255, A: 255})

	res, err := s.Convert(ctx, ConvertRequest{Path: src, TargetFormat: "png", MaxWidth: 4})
	if err != nil {
		t.Fatal(err)
	}
	if res.Width != 4 || res.Height != 4 {
		t.Fatalf("scaled size = %dx%d, want 4x4", res.Width, res.Height)
	}

	res, err = s.Convert(ctx, ConvertRequest{Path: src, TargetFormat: "png", MaxWidth: 64})
	if err != nil {
		t.Fatal(err)
	}
	if res.Width != 8 {
		t.Fatalf("upscaled to %d, want 8", res.Width)
	}
}

func TestOutputNameClashSuffix(t *testing.T) {
	dir := t.TempDir()
	existing := filepath.Join(dir, "img.jpg")
	if err := os.WriteFile(existing, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := outputFor(filepath.Join(dir, "img.png"), ".jpg")
	if err != nil {
		t.Fatal(err)
	}
	if got != filepath.Join(dir, "img-1.jpg") {
		t.Fatalf("outputFor clash = %q", got)
	}
}

func TestCancelActive(t *testing.T) {
	dir := t.TempDir()
	s := &Service{}
	src := writeTestPNG(t, dir, "c.png", color.RGBA{R: 255, A: 255})

	s.CancelActive() // no job running: must not panic

	// A canceled context makes Convert bail before writing output.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := s.Convert(ctx, ConvertRequest{Path: src, TargetFormat: "png"}); err == nil {
		t.Fatal("Convert succeeded with canceled context")
	}
}

func TestEncodeHelpersRoundTrip(t *testing.T) {
	dir := t.TempDir()
	src := writeTestPNG(t, dir, "rt.png", color.RGBA{R: 255, A: 255})
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	m, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := encodeGIF(&buf, m, 0); err != nil {
		t.Fatal(err)
	}
	if got, err := gif.Decode(bytes.NewReader(buf.Bytes())); err != nil || got.Bounds().Dx() != 8 {
		t.Fatalf("gif round-trip: %v, %v", got, err)
	}
	buf.Reset()
	if err := encodeBMP(&buf, m, 0); err != nil {
		t.Fatal(err)
	}
	buf.Reset()
	if err := encodeTIFF(&buf, m, 0); err != nil {
		t.Fatal(err)
	}
}
