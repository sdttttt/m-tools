package imgfmt

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

	"github.com/sdttttt/m-tools/webp"
)

// solid returns an 8x8 NRGBA filled with one color.
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

// testWebPBytes produces webp bytes for probe tests: a real encode on
// with_webp builds, a bare RIFF/WEBP container header otherwise (the stub
// path reports the missing codec before any decode is attempted).
func testWebPBytes(t *testing.T) []byte {
	t.Helper()
	if webp.Supported() {
		var buf bytes.Buffer
		if err := webp.Encode(&buf, solid(color.RGBA{B: 255, A: 255}), 90); err != nil {
			t.Fatalf("webp.Encode: %v", err)
		}
		return buf.Bytes()
	}
	hdr := make([]byte, 32)
	copy(hdr, "RIFF")
	putLE32(hdr[4:], 24)
	copy(hdr[8:], "WEBP")
	copy(hdr[12:], "VP8 ")
	putLE32(hdr[16:], 16)
	return hdr
}

func putLE32(b []byte, v uint32) {
	b[0] = byte(v)
	b[1] = byte(v >> 8)
	b[2] = byte(v >> 16)
	b[3] = byte(v >> 24)
}

func TestProbeFormats(t *testing.T) {
	dir := t.TempDir()

	pngPath := writeTestPNG(t, dir, "a.png", color.RGBA{R: 255, A: 255})
	info := Probe(pngPath)
	if info.Format != "png" || !info.Supported || info.Width != 8 || info.Height != 8 {
		t.Fatalf("png probe = %+v", info)
	}
	if info.HasAlpha {
		t.Fatal("opaque png reported HasAlpha")
	}

	info = Probe(writeTestJPEG(t, dir, "b.jpg"))
	if info.Format != "jpg" || !info.Supported || info.HasAlpha {
		t.Fatalf("jpg probe = %+v", info)
	}

	info = Probe(writeTestAnimatedGIF(t, dir, "c.gif", color.RGBA{R: 255, A: 255}, color.RGBA{B: 255, A: 255}))
	if info.Format != "gif" || !info.Supported {
		t.Fatalf("gif probe = %+v", info)
	}

	// A webp file: supported builds probe it, stub builds report the tag hint.
	webpPath := filepath.Join(dir, "d.webp")
	if err := os.WriteFile(webpPath, testWebPBytes(t), 0o644); err != nil {
		t.Fatal(err)
	}
	info = Probe(webpPath)
	if webp.Supported() {
		if info.Format != "webp" || !info.Supported {
			t.Fatalf("webp probe = %+v", info)
		}
	} else {
		if info.Supported {
			t.Fatal("stub build reported webp as supported")
		}
		if info.SkipReason != ErrUnsupported.Error() {
			t.Fatalf("webp skip reason = %q", info.SkipReason)
		}
	}

	// Garbage: unsupported, but Probe never errors — the page shows rows.
	junk := filepath.Join(dir, "e.png")
	if err := os.WriteFile(junk, []byte("not an image at all"), 0o644); err != nil {
		t.Fatal(err)
	}
	info = Probe(junk)
	if info.Supported || info.SkipReason == "" {
		t.Fatalf("junk probe = %+v", info)
	}
}

func TestPipelineMatrix(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()

	src := writeTestPNG(t, dir, "img.png", color.RGBA{R: 255, G: 0, B: 0, A: 255})

	// Every target round-trips the source size (no resize axes given).
	for _, target := range []string{"jpg", "png", "gif", "bmp", "tiff"} {
		res, err := Run(ctx, Spec{
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
		if res.PNGBytes == nil {
			t.Fatalf("png→%s preview bytes missing", target)
		}
		os.Remove(res.OutputPath)
	}

	// Unknown target.
	if _, err := Run(ctx, Spec{Path: src, TargetFormat: "heic"}); err == nil {
		t.Fatal("unknown target accepted")
	}

	// JPEG source to png (decode via table, re-encode lossless).
	jpg := writeTestJPEG(t, dir, "photo.jpg")
	res, err := Run(ctx, Spec{Path: jpg, TargetFormat: "png"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Format != "png" || res.OutputPath != filepath.Join(dir, "photo.png") {
		t.Fatalf("jpg→png = %+v", res)
	}
}

func TestPipelineAlphaFlattenedForJPEG(t *testing.T) {
	dir := t.TempDir()

	// RGBA source: top-left opaque red, rest fully transparent.
	m := solid(color.NRGBA{A: 0})
	m.Set(0, 0, color.NRGBA{R: 255, A: 255})
	p := encodeToFile(t, dir, "alpha.png", m, func(f *os.File, img image.Image) error {
		return png.Encode(f, img)
	})

	info := Probe(p)
	if !info.HasAlpha {
		t.Fatal("transparent png not flagged HasAlpha")
	}

	res, err := Run(context.Background(), Spec{Path: p, TargetFormat: "jpg"})
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

func TestPipelineAnimatedGIFTakesFirstFrame(t *testing.T) {
	dir := t.TempDir()
	gifPath := writeTestAnimatedGIF(t, dir, "anim.gif",
		color.RGBA{R: 255, A: 255}, color.RGBA{B: 255, A: 255})

	res, err := Run(context.Background(), Spec{Path: gifPath, TargetFormat: "png"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Width != 8 || res.Height != 8 {
		t.Fatalf("anim→png size = %dx%d", res.Width, res.Height)
	}
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

func TestResizeUpAndDown(t *testing.T) {
	dir := t.TempDir()
	src := writeTestPNG(t, dir, "big.png", color.RGBA{G: 255, A: 255})
	ctx := context.Background()

	// Downscale to an exact size.
	res, err := Run(ctx, Spec{Path: src, TargetFormat: "png", Width: 4, Height: 4})
	if err != nil {
		t.Fatal(err)
	}
	if res.Width != 4 || res.Height != 4 {
		t.Fatalf("downscale size = %dx%d, want 4x4", res.Width, res.Height)
	}

	// Upscale is allowed by the shared pipeline (the resize tool decides).
	res, err = Run(ctx, Spec{Path: src, TargetFormat: "png", Width: 16, Height: 16})
	if err != nil {
		t.Fatal(err)
	}
	if res.Width != 16 || res.Height != 16 {
		t.Fatalf("upscale size = %dx%d, want 16x16", res.Width, res.Height)
	}

	// Zero axes keep the source size.
	res, err = Run(ctx, Spec{Path: src, TargetFormat: "png"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Width != 8 || res.Height != 8 {
		t.Fatalf("keep-size = %dx%d, want 8x8", res.Width, res.Height)
	}
}

func TestResizePartialAxes(t *testing.T) {
	// Non-square source: 8x4. The pipeline itself does no aspect math —
	// each zero axis just keeps the source size (the resize tool derives
	// ratios before calling Run).
	m := image.NewNRGBA(image.Rect(0, 0, 8, 4))
	dir := t.TempDir()
	src := encodeToFile(t, dir, "wide.png", m, func(f *os.File, img image.Image) error {
		return png.Encode(f, img)
	})

	res, err := Run(context.Background(), Spec{Path: src, TargetFormat: "png", Width: 4})
	if err != nil {
		t.Fatal(err)
	}
	if res.Width != 4 || res.Height != 4 {
		t.Fatalf("width-only = %dx%d, want 4x4 (height kept)", res.Width, res.Height)
	}

	res, err = Run(context.Background(), Spec{Path: src, TargetFormat: "png", Height: 8})
	if err != nil {
		t.Fatal(err)
	}
	if res.Width != 8 || res.Height != 8 {
		t.Fatalf("height-only = %dx%d, want 8x8 (width kept)", res.Width, res.Height)
	}
}

func TestOutputNameClashSuffix(t *testing.T) {
	dir := t.TempDir()
	existing := filepath.Join(dir, "img.jpg")
	if err := os.WriteFile(existing, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := OutputFor(filepath.Join(dir, "img.png"), ".jpg")
	if err != nil {
		t.Fatal(err)
	}
	if got != filepath.Join(dir, "img-1.jpg") {
		t.Fatalf("OutputFor clash = %q", got)
	}
}

func TestCancelStopsPipeline(t *testing.T) {
	dir := t.TempDir()
	src := writeTestPNG(t, dir, "c.png", color.RGBA{R: 255, A: 255})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Run(ctx, Spec{Path: src, TargetFormat: "png"}); err == nil {
		t.Fatal("Run succeeded with canceled context")
	}
}

func TestStubDecodeRejectsWebp(t *testing.T) {
	if webp.Supported() {
		t.Skip("webp codec compiled in; stub behavior not reachable")
	}
	if _, _, err := image.Decode(bytes.NewReader(testWebPBytes(t))); err == nil {
		t.Fatal("image.Decode accepted webp bytes on a stub build")
	}
}

func TestTargetNamesIncludeWebpOnlyWhenSupported(t *testing.T) {
	names := TargetNames()
	has := false
	for _, n := range names {
		if n == "webp" {
			has = true
		}
	}
	if has != webp.Supported() {
		t.Fatalf("TargetNames webp = %v, Supported = %v", has, webp.Supported())
	}
	for _, n := range []string{"jpg", "png", "gif", "bmp", "tiff"} {
		if !HasTarget(n) {
			t.Fatalf("HasTarget(%q) = false", n)
		}
	}
}
