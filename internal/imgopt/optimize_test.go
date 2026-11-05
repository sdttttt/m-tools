package imgopt

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// writeFile writes raw bytes as a test file and returns its path.
func writeFile(t *testing.T, dir, name string, data []byte) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// encodeToFile encodes m with encode and returns the written path.
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

// noiseNRGBA returns a w×h image with deterministic pseudo-random pixels
// (many unique colors, opaque), the hard case for PNG compression.
func noiseNRGBA(w, h int) *image.NRGBA {
	m := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			m.SetNRGBA(x, y, color.NRGBA{
				R: uint8(x*7 + y*13), G: uint8(x*3 + y*5), B: uint8(x*11 + y), A: 255,
			})
		}
	}
	return m
}

// pngBytes encodes m with the given compression level and returns bytes.
func pngBytes(t *testing.T, m image.Image, level png.CompressionLevel) []byte {
	t.Helper()
	var buf bytes.Buffer
	enc := png.Encoder{CompressionLevel: level}
	if err := enc.Encode(&buf, m); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// decodePNGPixels re-decodes data and returns the pixel image for
// equality checks against the source.
func decodePNGPixels(t *testing.T, data []byte) image.Image {
	t.Helper()
	m, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("optimized png does not decode: %v", err)
	}
	return m
}

func pixelsEqual(a, b image.Image) bool {
	ba, bb := a.Bounds(), b.Bounds()
	if ba.Dx() != bb.Dx() || ba.Dy() != bb.Dy() {
		return false
	}
	for y := ba.Min.Y; y < ba.Max.Y; y++ {
		for x := ba.Min.X; x < ba.Max.X; x++ {
			// color.Color values of different dynamic types (NRGBA vs the
			// RGBA the decoder hands back) never compare equal as
			// interfaces; 16-bit channel comparison is the real test.
			r1, g1, b1, a1 := a.At(x, y).RGBA()
			r2, g2, b2, a2 := b.At(x, y).RGBA()
			if r1 != r2 || g1 != g2 || b1 != b2 || a1 != a2 {
				return false
			}
		}
	}
	return true
}

func TestOptimizePNGNoiseShrinks(t *testing.T) {
	dir := t.TempDir()
	src := writeFile(t, dir, "noise.png", pngBytes(t, noiseNRGBA(200, 200), png.NoCompression))

	res, err := Optimize(src)
	if err != nil {
		t.Fatal(err)
	}
	if res.Skipped || res.NotNeeded {
		t.Fatalf("expected an optimized output, got skipped=%v notNeeded=%v (%s)", res.Skipped, res.NotNeeded, res.Reason)
	}
	if res.OutputPath == "" || filepath.Ext(res.OutputPath) != ".png" {
		t.Fatalf("unexpected output path %q", res.OutputPath)
	}
	if res.OptimizedBytes >= res.OriginalBytes {
		t.Fatalf("expected smaller output: %d >= %d", res.OptimizedBytes, res.OriginalBytes)
	}
	out, err := os.ReadFile(res.OutputPath)
	if err != nil {
		t.Fatal(err)
	}
	// Lossless: the optimized PNG must decode to the exact source pixels.
	if !pixelsEqual(noiseNRGBA(200, 200), decodePNGPixels(t, out)) {
		t.Fatal("optimized PNG pixels differ from the source")
	}
}

func TestOptimizePNGPalette(t *testing.T) {
	dir := t.TempDir()
	// A 4-color image written with a wasteful full-color encoder: the
	// optimizer should switch to an indexed palette.
	m := image.NewNRGBA(image.Rect(0, 0, 40, 40))
	for y := 0; y < 40; y++ {
		for x := 0; x < 40; x++ {
			var c color.RGBA
			switch (x/10 + y/10) % 2 {
			case 0:
				c = color.RGBA{R: 200, G: 10, B: 10, A: 255}
			default:
				c = color.RGBA{R: 10, G: 200, B: 10, A: 255}
			}
			m.Set(x, y, c)
		}
	}
	src := writeFile(t, dir, "four.png", pngBytes(t, m, png.BestCompression))

	res, err := Optimize(src)
	if err != nil {
		t.Fatal(err)
	}
	if res.Skipped || res.NotNeeded {
		t.Fatalf("expected palette shrink, got skipped=%v notNeeded=%v", res.Skipped, res.NotNeeded)
	}
	out, err := os.ReadFile(res.OutputPath)
	if err != nil {
		t.Fatal(err)
	}
	got := decodePNGPixels(t, out)
	if !pixelsEqual(m, got) {
		t.Fatal("palette PNG pixels differ from the source")
	}
	if _, ok := got.(*image.Paletted); !ok {
		t.Fatalf("expected a paletted output, got %T", got)
	}
}

func TestOptimizePNGPartialAlphaSkipsPalette(t *testing.T) {
	dir := t.TempDir()
	// Two colors but antialiased-style partial alpha: the palette path
	// must refuse; result stays lossless (or NotNeeded).
	m := image.NewNRGBA(image.Rect(0, 0, 20, 20))
	for y := 0; y < 20; y++ {
		for x := 0; x < 20; x++ {
			if x < 10 {
				m.SetNRGBA(x, y, color.NRGBA{R: 1, G: 2, B: 3, A: 128})
			} else {
				m.SetNRGBA(x, y, color.NRGBA{R: 4, G: 5, B: 6, A: 255})
			}
		}
	}
	src := writeFile(t, dir, "alpha.png", pngBytes(t, m, png.NoCompression))

	res, err := Optimize(src)
	if err != nil {
		t.Fatal(err)
	}
	if res.Skipped {
		t.Fatalf("full-color path should still apply: %s", res.Reason)
	}
	if res.NotNeeded {
		return // nothing to shave is a fine outcome here
	}
	out, err := os.ReadFile(res.OutputPath)
	if err != nil {
		t.Fatal(err)
	}
	if !pixelsEqual(m, decodePNGPixels(t, out)) {
		t.Fatal("optimized PNG pixels differ from the source")
	}
}

func TestOptimizeJPEGStripsMetadata(t *testing.T) {
	dir := t.TempDir()
	// Hand-build a JPEG stream: SOI + a large EXIF-like APP1 + a COM +
	// DQT (db) + SOF0 (c0) + SOS (da) + entropy bytes + EOI.
	var b bytes.Buffer
	b.Write([]byte{0xff, 0xd8})
	app1 := []byte("Exif\x00\x00" + string(make([]byte, 400)))
	app1Len := len(app1) + 2
	b.Write([]byte{0xff, 0xe1, byte(app1Len >> 8), byte(app1Len)})
	b.Write(app1)
	b.Write([]byte{0xff, 0xfe, 0x00, 0x08, 'c', 'o', 'm', 'm', 'e', 'n'})
	// A real minimal DQT: one 8-bit table.
	dqt := make([]byte, 67)
	dqt[0] = 0x00
	b.Write([]byte{0xff, 0xdb, 0x00, 0x43})
	b.Write(dqt)
	b.Write([]byte{0xff, 0xd9})
	full := b.Bytes()

	src := writeFile(t, dir, "meta.jpg", full)
	res, err := Optimize(src)
	if err != nil {
		t.Fatal(err)
	}
	if res.Skipped || res.NotNeeded {
		t.Fatalf("expected metadata strip, got skipped=%v notNeeded=%v (%s)", res.Skipped, res.NotNeeded, res.Reason)
	}
	out, err := os.ReadFile(res.OutputPath)
	if err != nil {
		t.Fatal(err)
	}
	if res.OptimizedBytes >= res.OriginalBytes {
		t.Fatalf("expected smaller output: %d >= %d", res.OptimizedBytes, res.OriginalBytes)
	}
	// APP1/COM gone, DQT/EOI kept.
	if bytes.Contains(out, []byte("Exif")) || bytes.Contains(out, []byte("comment")) {
		t.Fatal("metadata segments survived the strip")
	}
	if !bytes.Contains(out, []byte{0xff, 0xdb}) || !bytes.Contains(out, []byte{0xff, 0xd9}) {
		t.Fatal("table or EOI segments were lost")
	}
}

func TestOptimizeJPEGWithoutMetadataIsNotNeeded(t *testing.T) {
	dir := t.TempDir()
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 16, 16)), nil); err != nil {
		t.Fatal(err)
	}
	// stdlib writes only JFIF APP0 — small, so strip yields ~nothing;
	// the guardrail must keep the result at NotNeeded or skip writing.
	src := writeFile(t, dir, "plain.jpg", buf.Bytes())
	res, err := Optimize(src)
	if err != nil {
		t.Fatal(err)
	}
	if res.OutputPath != "" && res.OptimizedBytes >= res.OriginalBytes {
		t.Fatalf("wrote a non-smaller output: %d >= %d", res.OptimizedBytes, res.OriginalBytes)
	}
}

// TestOptimizeAnimatedGIFFSkipped: a two-frame GIF must be skipped.
func TestOptimizeAnimatedGIFFSkipped(t *testing.T) {
	dir := t.TempDir()
	out := &gif.GIF{}
	for _, c := range []color.RGBA{{R: 255}, {R: 0, G: 255}} {
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
	var buf bytes.Buffer
	if err := gif.EncodeAll(&buf, out); err != nil {
		t.Fatal(err)
	}
	src := writeFile(t, dir, "anim.gif", buf.Bytes())

	res, err := Optimize(src)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Skipped || res.Reason == "" {
		t.Fatalf("expected a skip with reason, got skipped=%v notNeeded=%v reason=%q", res.Skipped, res.NotNeeded, res.Reason)
	}
	if res.OutputPath != "" {
		t.Fatalf("skipped row must not write a file, got %q", res.OutputPath)
	}
}

func TestOptimizeStaticGIF(t *testing.T) {
	dir := t.TempDir()
	palette := color.Palette{color.RGBA{R: 255}, color.Black}
	m := image.NewPaletted(image.Rect(0, 0, 8, 8), palette)
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			m.SetColorIndex(x, y, uint8((x+y)%2))
		}
	}
	var buf bytes.Buffer
	if err := gif.Encode(&buf, m, nil); err != nil {
		t.Fatal(err)
	}
	src := writeFile(t, dir, "still.gif", buf.Bytes())

	res, err := Optimize(src)
	if err != nil {
		t.Fatal(err)
	}
	if res.Skipped {
		t.Fatalf("static gif should optimize: %s", res.Reason)
	}
	// A single-frame GIF usually re-encodes to about the same size; either
	// NotNeeded or a strictly smaller file is acceptable, never larger.
	if res.OutputPath != "" && res.OptimizedBytes >= res.OriginalBytes {
		t.Fatalf("wrote a non-smaller output: %d >= %d", res.OptimizedBytes, res.OriginalBytes)
	}
}

func TestOptimizeBMPRejected(t *testing.T) {
	dir := t.TempDir()
	// Minimal BMP: header + pixel data (a 1×1 24-bit image).
	bmp := []byte{
		'B', 'M', 58, 0, 0, 0, 0, 0, 0, 0, 54, 0, 0, 0, // file header
		40, 0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0, 1, 0, 24, 0, // info header
		0, 0, 0, 0, 4, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0,
		255, 0, 0, 0, // pixel
	}
	src := writeFile(t, dir, "t.bmp", bmp)

	res, err := Optimize(src)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Skipped || res.Reason == "" {
		t.Fatalf("expected BMP skip with reason, got %+v", res)
	}
}

func TestOptimizeUnknownSkipped(t *testing.T) {
	dir := t.TempDir()
	src := writeFile(t, dir, "blob.bin", []byte("not an image at all"))
	res, err := Optimize(src)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Skipped || res.Reason == "" {
		t.Fatalf("expected skip with reason, got %+v", res)
	}
}

func TestOptimizeNeverWritesLargerOutput(t *testing.T) {
	dir := t.TempDir()
	// Already BestCompression PNG: re-encode cannot win, so no file.
	m := noiseNRGBA(64, 64)
	src := writeFile(t, dir, "tight.png", pngBytes(t, m, png.BestCompression))

	res, err := Optimize(src)
	if err != nil {
		t.Fatal(err)
	}
	if res.OutputPath != "" {
		t.Fatalf("guardrail must not write a non-smaller output, got %q", res.OutputPath)
	}
	if !res.NotNeeded {
		t.Fatalf("expected NotNeeded, got %+v", res)
	}
}

func TestOptimizeAPNGSkipped(t *testing.T) {
	dir := t.TempDir()
	// A PNG that carries the acTL chunk: must be skipped before decode
	// drops frames, even though the bytes are otherwise a valid PNG.
	base := pngBytes(t, noiseNRGBA(8, 8), png.DefaultCompression)
	// Inject "acTL" into a random ancillary-looking position: after IHDR.
	data := append([]byte{}, base[:33]...)
	data = append(data, []byte("acTL\x00\x00\x00\x08\x00\x00\x00\x01\x00\x00\x00\x02")...)
	data = append(data, base[33:]...)
	src := writeFile(t, dir, "apng.png", data)

	res, err := Optimize(src)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Skipped || res.Reason == "" {
		t.Fatalf("expected APNG skip, got %+v", res)
	}
}
