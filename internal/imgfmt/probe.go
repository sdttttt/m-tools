package imgfmt

import (
	"bytes"
	"fmt"
	"image"
	"os"
)

// webpMagic is the RIFF container prefix shared by all webp variants
// (lossy VP8, lossless VP8L, extended VP8X). `?` wildcards the size field.
const webpMagic = "RIFF????WEBP"

// Detect reports a file's canonical format id (png, jpg, gif, webp, bmp,
// tiff) by magic bytes, for callers that need the format without the full
// Probe (known=false means unrecognizable; the error carries a specific
// hint such as webp on a stub build).
func Detect(data []byte) (string, bool, error) {
	f, known, err := detect(data)
	if f == nil {
		return "", known, err
	}
	return f.name, known, err
}

// Probe sniffs a file's format by magic bytes and reads its dimensions.
// It never fails: unrecognizable or unsupported sources come back with a
// SkipReason so the page can show the row as "skipped" — including the
// webp-needs-with_webp hint on default builds.
func Probe(path string) Info {
	data, err := os.ReadFile(path)
	if err != nil {
		return Info{Path: path, SkipReason: "读取失败"}
	}
	info := Info{
		Path:      path,
		SizeBytes: int64(len(data)),
	}
	f, known, err := detect(data)
	if err != nil {
		info.SkipReason = err.Error()
		return info
	}
	if !known {
		info.SkipReason = "无法识别的图片格式"
		return info
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		info.Format = f.name
		info.SkipReason = "解码失败：" + err.Error()
		return info
	}
	info.Format = f.name
	info.Width, info.Height = cfg.Width, cfg.Height
	info.Supported = true
	info.HasAlpha = hasAlpha(data, f)
	return info
}

// detect returns (format, known, error): known=false with a nil error
// means "unrecognizable"; the error carries a specific hint (webp on a
// stub build, unknown RIFF container).
func detect(data []byte) (*format, bool, error) {
	check := func(f *format) bool {
		if len(data) < len(f.magic) {
			return false
		}
		for i, c := range []byte(f.magic) {
			if c != '?' && data[i] != c {
				return false
			}
		}
		return true
	}
	for _, f := range formats {
		if check(f) {
			return f, true, nil
		}
	}
	webpProbe := &format{name: "webp", magic: webpMagic}
	if check(webpProbe) {
		if f, ok := webpFormat(); ok {
			return f, true, nil
		}
		return nil, false, ErrUnsupported
	}
	if bytes.HasPrefix(data, []byte("RIFF")) {
		return nil, false, fmt.Errorf("不支持该 RIFF 容器格式")
	}
	// TIFF big-endian variant ("MM\x00*").
	if bytes.HasPrefix(data, []byte("MM\x00*")) {
		return formats["tiff"], true, nil
	}
	return nil, false, fmt.Errorf("无法识别的图片格式")
}

// hasAlpha reports whether decoding this source yields transparency that a
// JPEG target would have to flatten. It decodes fully (cheap relative to
// conversion) because DecodeConfig does not expose the color model
// uniformly for paletted formats.
func hasAlpha(data []byte, f *format) bool {
	if f.name == "jpg" {
		return false // baseline JPEG has no alpha channel
	}
	m, err := f.decode(bytes.NewReader(data))
	if err != nil {
		return false
	}
	return AlphaPresent(m)
}

// AlphaPresent scans an image for any pixel with transparency.
func AlphaPresent(m image.Image) bool {
	switch pm := m.(type) {
	case *image.Gray, *image.Gray16, *image.CMYK:
		return false
	case *image.Paletted:
		for _, c := range pm.Palette {
			if _, _, _, a := c.RGBA(); a < 0xffff {
				return true
			}
		}
		return false
	}
	b := m.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if _, _, _, a := m.At(x, y).RGBA(); a < 0xffff {
				return true
			}
		}
	}
	return false
}
