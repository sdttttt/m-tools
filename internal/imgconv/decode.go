package imgconv

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"os"

	"github.com/sdttttt/m-tools/webp"
)

// webpSupported mirrors webp.Supported(); kept as a local name so format.go
// and decode.go read uniformly.
func webpSupported() bool { return webp.Supported() }

// errWebpUnsupported is the friendly face of webp.ErrUnsupported for IPC.
var errWebpUnsupported = errors.New("webp 支持未编译：需使用 -tags with_webp 构建的版本")

// webpMagic is the RIFF container prefix shared by all webp variants
// (lossy VP8, lossless VP8L, extended VP8X). `?` wildcards the size field.
const webpMagic = "RIFF????WEBP"

// qualityLevels maps the page's quality choice to JPEG/WebP quality 0-100.
var qualityLevels = map[string]int{
	"high":     92,
	"balanced": 82,
	"compact":  62,
}

// previewLimit caps how big a converted image may be inlined as a data URL.
const previewLimit = 8 << 20 // 8 MiB

// probeFormat sniffs a file's format by magic bytes; it exists so webp can
// be reported as "needs with_webp" instead of a generic decode failure on
// default builds. Returns (format, known, error): known=false with a nil
// error means "unrecognizable"; the error carries a specific hint (webp on
// a stub build, unknown RIFF container).
func probeFormat(data []byte) (*format, bool, error) {
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
		return nil, false, errWebpUnsupported
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

// Probe reads metadata of one picked image.
func (s *Service) Probe(path string) (ImageInfo, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ImageInfo{}, err
	}
	info := ImageInfo{
		Path:      path,
		SizeBytes: int64(len(data)),
	}
	f, ok, err := probeFormat(data)
	if err != nil {
		info.SkipReason = err.Error()
		return info, nil
	}
	if !ok {
		info.SkipReason = "无法识别的图片格式"
		return info, nil
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		info.Format = f.name
		info.SkipReason = "解码失败：" + err.Error()
		return info, nil
	}
	info.Format = f.name
	info.Width, info.Height = cfg.Width, cfg.Height
	info.Supported = true
	info.HasAlpha = hasAlpha(data, f)
	return info, nil
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
	return alphaPresent(m)
}

// alphaPresent scans an image for any pixel with transparency.
func alphaPresent(m image.Image) bool {
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
