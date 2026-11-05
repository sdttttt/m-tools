// Package imgopt is the lossless image optimization tool: it shrinks file
// size without touching pixels (PNG palette/re-compression, JPEG metadata
// stripping, TIFF deflate, static-GIF/WebP re-encode) and never writes an
// output larger than the source. Output files carry a -min suffix next to
// the source; animated sources (APNG, multi-frame GIF, animated WebP) and
// formats with no compression headroom (BMP) are skipped with a reason.
package imgopt

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"image/png"
	"os"

	tiff "golang.org/x/image/tiff"

	"github.com/sdttttt/m-tools/internal/imgfmt"
	"github.com/sdttttt/m-tools/webp"
)

// Result reports one optimization attempt. Skipped=true rows carry a
// Reason and never produced a file; NotNeeded=true rows found nothing to
// shave (output would be >= the source) and also wrote nothing.
type Result struct {
	OutputPath     string
	Format         string
	OriginalBytes  int64
	OptimizedBytes int64
	NotNeeded      bool
	Skipped        bool
	Reason         string
	Preview        []byte
}

// ErrSkipped marks a source the tool cannot optimize (animated image, BMP,
// undecodable bytes); Reason carries the user-facing text.
var ErrSkipped = errors.New("skipped")

// Optimize optimizes one image file losslessly and writes the -min output
// next to it when the result is strictly smaller than the source. ctx
// cancellation stops the work between stages.
func Optimize(path string) (Result, error) {
	return optimize(context.Background(), path)
}

// optimize is the ctx-aware core of Optimize: read, detect, dispatch,
// compare, write. The service layer wraps it with the job guard.
func optimize(ctx context.Context, path string) (Result, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Result{}, err
	}
	res := Result{OriginalBytes: int64(len(data))}

	format, known, err := imgfmt.Detect(data)
	// Unrecognizable bytes surface as a skip row, not a hard error.
	if err != nil || !known {
		res.Skipped = true
		if errors.Is(err, imgfmt.ErrUnsupported) {
			res.Reason = err.Error()
		} else {
			res.Reason = "无法识别的图片格式"
		}
		return res, nil
	}
	res.Format = format

	if err := ctx.Err(); err != nil {
		return Result{}, err
	}

	var out []byte
	switch format {
	case "png":
		out, err = optimizePNG(data)
	case "jpg":
		out, err = optimizeJPEG(data)
	case "tiff":
		out, err = optimizeTIFF(data)
	case "gif":
		out, err = optimizeGIF(data)
	case "webp":
		out, err = optimizeWebP(data)
	case "bmp":
		res.Skipped = true
		res.Reason = "BMP 是无压缩格式，没有可优化的空间"
		return res, nil
	default:
		res.Skipped = true
		res.Reason = "无法识别的图片格式"
		return res, nil
	}
	if err != nil {
		if errors.Is(err, ErrSkipped) {
			res.Skipped = true
			res.Reason = errText(err)
			return res, nil
		}
		return Result{}, err
	}

	// Guardrail: never write an output that is not strictly smaller.
	if len(out) >= len(data) {
		res.NotNeeded = true
		res.OptimizedBytes = int64(len(data))
		return res, nil
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}

	ext := "." + format
	outputPath, err := imgfmt.OutputFor(path, ext)
	if err != nil {
		return Result{}, err
	}
	// OutputFor avoids clashes with -1/-2 suffixes; a previous -min run is
	// a clash like any other, so re-running never overwrites earlier work.
	if err := os.WriteFile(outputPath, out, 0o644); err != nil {
		return Result{}, err
	}

	res.OutputPath = outputPath
	res.OptimizedBytes = int64(len(out))
	if len(out) <= imgfmt.PreviewLimit {
		res.Preview = out
	}
	return res, nil
}

// errText unwraps the user-facing message from a skip-wrapped error.
func errText(err error) string {
	skipped := &skipError{}
	if errors.As(err, &skipped) {
		return skipped.msg
	}
	return err.Error()
}

// skipError marks a per-format skip with a user-facing reason.
type skipError struct{ msg string }

func (e *skipError) Error() string { return e.msg }

func (e *skipError) Is(target error) bool { return target == ErrSkipped }

// ErrSkip wraps a reason into a skip signal for the format dispatch.
func skipReason(msg string) error { return &skipError{msg: msg} }

// optimizePNG re-encodes a PNG at BestCompression, switching to an indexed
// palette when the image has at most 256 unique opaque-or-binary-alpha
// colors. The pixel values never change; only the encoding does.
func optimizePNG(data []byte) ([]byte, error) {
	if isAPNG(data) {
		return nil, skipReason("APNG 动图不支持优化")
	}
	m, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("解码失败：%w", err)
	}
	if pal, ok := palettedCandidate(m); ok {
		var buf bytes.Buffer
		if err := png.Encode(&buf, pal); err == nil {
			return buf.Bytes(), nil
		}
		// Palette encoding failed (should not happen after the checks);
		// fall through to the full-color path.
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, m); err != nil {
		return nil, fmt.Errorf("编码失败：%w", err)
	}
	return buf.Bytes(), nil
}

// isAPNG reports whether data carries the APNG animation chunk. Plain
// decoders would silently drop every frame after the first.
func isAPNG(data []byte) bool {
	return bytes.Contains(data, []byte("acTL"))
}

// palettedCandidate converts m to a *image.Paletted when the whole image
// is representable losslessly with at most 256 colors: every pixel's color
// must appear in the palette, alpha must be fully opaque or fully
// transparent (antialiased edges would shimmer). The returned image's
// pixels are verified against the source; any mismatch returns false.
func palettedCandidate(m image.Image) (*image.Paletted, bool) {
	type rgb struct{ r, g, b uint8 }
	seen := make(map[rgb]int, 256) // key -> palette index
	var palette []color.Color
	b := m.Bounds()
	// Pass 1: collect up to 256 unique colors (transparent collapses to a
	// single slot; distinct transparent RGB values stay identical anyway).
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, a := m.At(x, y).RGBA()
			if a != 0xffff && a != 0 {
				return nil, false // partial alpha: quantizing would change pixels
			}
			if a == 0 {
				if _, dup := seen[rgb{}]; !dup {
					if len(palette) == 256 {
						return nil, false
					}
					seen[rgb{}] = len(palette)
					palette = append(palette, color.RGBA{0, 0, 0, 0})
				}
				continue
			}
			key := rgb{uint8(r >> 8), uint8(g >> 8), uint8(bl >> 8)}
			if _, dup := seen[key]; !dup {
				if len(palette) == 256 {
					return nil, false
				}
				seen[key] = len(palette)
				palette = append(palette, color.RGBA{key.r, key.g, key.b, 0xff})
			}
		}
	}
	dst := image.NewPaletted(image.Rect(0, 0, b.Dx(), b.Dy()), palette)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, a := m.At(x, y).RGBA()
			var idx int
			if a == 0 {
				idx = seen[rgb{}]
			} else {
				idx = seen[rgb{uint8(r >> 8), uint8(g >> 8), uint8(bl >> 8)}]
			}
			dst.SetColorIndex(x, y, uint8(idx))
		}
	}
	// Verify: every source pixel must equal its palette entry after the
	// round trip, otherwise the palette path is not lossless.
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if dst.At(x, y) != m.At(x, y) {
				return nil, false
			}
		}
	}
	return dst, true
}

// optimizeJPEG strips metadata segments (APPn, COM) from a JPEG byte
// stream without re-encoding: pixels, quantization tables and Huffman
// tables are untouched, so the output decodes to the exact same image.
func optimizeJPEG(data []byte) ([]byte, error) {
	if len(data) < 4 || data[0] != 0xff || data[1] != 0xd8 {
		return nil, fmt.Errorf("解码失败：非法 JPEG")
	}
	var buf bytes.Buffer
	buf.Write(data[:2]) // SOI
	pos := 2
	for pos < len(data) {
		// Byte-stuffing means real markers start with FF followed by a
		// non-zero, non-FF byte; scan to the next candidate.
		if data[pos] != 0xff {
			pos++
			continue
		}
		if pos+1 < len(data) && data[pos+1] == 0xff {
			pos++
			continue
		}
		if pos+1 >= len(data) {
			break
		}
		marker := data[pos+1]
		// Standalone markers without payload: RSTn (D0-D7) and TEM; EOI
		// (D9) and SOS (DA) end the metadata region.
		if marker == 0xd9 || marker == 0xda || marker == 0x01 || (marker >= 0xd0 && marker <= 0xd7) {
			// Copy everything from here on (entropy-coded data included)
			// verbatim; nothing after SOS carries removable metadata.
			buf.Write(data[pos:])
			pos = len(data)
			break
		}
		if marker == 0xff || marker == 0x00 {
			pos += 2
			continue
		}
		if pos+4 > len(data) {
			// Truncated length field: copy the rest verbatim, keep the
			// file decodable rather than "fixing" it.
			buf.Write(data[pos:])
			break
		}
		segLen := int(data[pos+2])<<8 | int(data[pos+3])
		isMeta := (marker >= 0xe0 && marker <= 0xef) || marker == 0xfe
		if isMeta && segLen >= 2 && pos+2+segLen <= len(data) {
			pos += 2 + segLen
			continue
		}
		end := pos + 2 + segLen
		if end > len(data) {
			buf.Write(data[pos:])
			break
		}
		buf.Write(data[pos : pos+2+segLen])
		pos = end
	}
	return buf.Bytes(), nil
}

// optimizeTIFF re-encodes a TIFF with Deflate compression. Pixels are
// decoded and re-encoded unchanged; only the storage shrinks.
func optimizeTIFF(data []byte) ([]byte, error) {
	m, err := tiff.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("解码失败：%w", err)
	}
	var buf bytes.Buffer
	if err := tiff.Encode(&buf, m, &tiff.Options{Compression: tiff.Deflate}); err != nil {
		return nil, fmt.Errorf("编码失败：%w", err)
	}
	return buf.Bytes(), nil
}

// optimizeGIF re-encodes a single-frame GIF. Multi-frame GIFs are skipped:
// decoding through the single-frame path would drop frames.
func optimizeGIF(data []byte) ([]byte, error) {
	g, err := gif.DecodeAll(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("解码失败：%w", err)
	}
	if len(g.Image) > 1 {
		return nil, skipReason("GIF 动图不支持优化")
	}
	var buf bytes.Buffer
	if err := gif.Encode(&buf, g.Image[0], nil); err != nil {
		return nil, fmt.Errorf("编码失败：%w", err)
	}
	return buf.Bytes(), nil
}

// optimizeWebP re-encodes a static WebP losslessly (VP8L). Animated WebP
// (VP8X with the animation flag) is skipped; default builds cannot read
// webp at all and report the build hint instead.
func optimizeWebP(data []byte) ([]byte, error) {
	if !webp.Supported() {
		return nil, imgfmt.ErrUnsupported
	}
	if isAnimatedWebP(data) {
		return nil, skipReason("WebP 动图不支持优化")
	}
	m, err := webp.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("解码失败：%w", err)
	}
	out, err := webp.EncodeLossless(m)
	if err != nil {
		return nil, fmt.Errorf("编码失败：%w", err)
	}
	return out, nil
}

// isAnimatedWebP reports whether the RIFF/WEBP container is the extended
// (VP8X) variant with the animation flag set (byte 1 of the VP8X flags,
// bit 1). Lossy (VP8) and still lossless (VP8L) files have no VP8X chunk.
func isAnimatedWebP(data []byte) bool {
	// Layout: RIFF(4) size(4) WEBP(4) chunk fourcc(4) chunk size(4)...
	if len(data) < 20 || string(data[8:12]) != "WEBP" {
		return false
	}
	if string(data[12:16]) != "VP8X" {
		return false
	}
	if len(data) < 17 {
		return false
	}
	return data[16]&0x02 != 0
}
