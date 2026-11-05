package imgfmt

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/draw"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	xdraw "golang.org/x/image/draw"
)

// Spec is one pipeline request shared by the image services.
type Spec struct {
	Path         string
	TargetFormat string
	Quality      string // high | balanced | compact; JPEG/WebP only
	// Width/Height are the target pixel size; 0 on either axis means
	// "keep the source size on that axis". Upscaling is allowed — the
	// resize tool decides its own clamping policy.
	Width  int
	Height int
}

// Result reports one finished pipeline run.
type Result struct {
	OutputPath string
	Format     string
	SizeBytes  int64
	Width      int
	Height     int
	PNGBytes   []byte // converted image bytes; nil when > PreviewLimit
}

// PreviewLimit caps how big a converted image may be inlined as a data URL.
const PreviewLimit = 8 << 20 // 8 MiB

// Run executes the full pipeline for one file: read, decode, resize,
// flatten, encode, write next to the source. ctx cancellation stops it
// between stages; quality defaults to balanced when empty/unknown.
func Run(ctx context.Context, sp Spec) (Result, error) {
	target, ok := resolveTarget(sp.TargetFormat)
	if !ok {
		return Result{}, fmt.Errorf("不支持的目标格式 %q", sp.TargetFormat)
	}

	data, err := os.ReadFile(sp.Path)
	if err != nil {
		return Result{}, err
	}

	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		// A webp source on a stub build fails inside image.Decode; give
		// the tag hint instead of the raw decoder error.
		if f, known, perr := detect(data); perr == nil && !known {
			return Result{}, perr
		} else if f == nil && perr != nil {
			return Result{}, perr
		} else if f != nil && f.name == "webp" {
			return Result{}, ErrUnsupported
		}
		return Result{}, fmt.Errorf("解码失败：%w", err)
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}

	out := Resize(src, sp.Width, sp.Height)
	if target.name == "jpg" && AlphaPresent(out) {
		out = FlattenWhite(out)
	}

	quality, ok := QualityLevels[sp.Quality]
	if !ok {
		quality = QualityLevels["balanced"]
	}

	var buf bytes.Buffer
	if err := target.encode(&buf, out, quality); err != nil {
		return Result{}, fmt.Errorf("编码失败：%w", err)
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}

	outputPath, err := OutputFor(sp.Path, target.ext)
	if err != nil {
		return Result{}, err
	}
	if err := os.WriteFile(outputPath, buf.Bytes(), 0o644); err != nil {
		return Result{}, err
	}

	res := Result{
		OutputPath: outputPath,
		Format:     target.name,
		SizeBytes:  int64(buf.Len()),
		Width:      out.Bounds().Dx(),
		Height:     out.Bounds().Dy(),
	}
	if buf.Len() <= PreviewLimit {
		res.PNGBytes = buf.Bytes()
	}
	return res, nil
}

// resolveTarget maps a format id to its table entry (including webp on
// with_webp builds).
func resolveTarget(id string) (*format, bool) {
	if id == "webp" {
		return webpFormat()
	}
	f, ok := formats[id]
	return f, ok
}

// Resize scales m to exactly width×height (aspect is the caller's
// responsibility). An axis left at 0 keeps the source size on that axis;
// matching the source size skips the resample entirely. Upscaling is
// allowed: CatmullRom keeps enlarged images reasonable.
func Resize(m image.Image, width, height int) image.Image {
	b := m.Bounds()
	w, h := width, height
	if w <= 0 {
		w = b.Dx()
	}
	if h <= 0 {
		h = b.Dy()
	}
	if w == b.Dx() && h == b.Dy() {
		return m
	}
	dst := image.NewNRGBA(image.Rect(0, 0, w, h))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), m, b, xdraw.Over, nil)
	return dst
}

// FlattenWhite composites m over opaque white, for opaque targets like
// JPEG. Exported files must not depend on the window theme, so white.
func FlattenWhite(m image.Image) image.Image {
	b := m.Bounds()
	dst := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(dst, dst.Bounds(), &image.Uniform{image.White}, image.Point{}, draw.Src)
	draw.Draw(dst, dst.Bounds(), m, b.Min, draw.Over)
	return dst
}

// OutputFor returns the default output path: next to the source, same
// stem, new extension; a name clash adds -1, -2… before the extension.
func OutputFor(src, ext string) (string, error) {
	dir := filepath.Dir(src)
	stem := strings.TrimSuffix(filepath.Base(src), filepath.Ext(src))
	for i := 0; ; i++ {
		name := stem + ext
		if i > 0 {
			name = stem + "-" + strconv.Itoa(i) + ext
		}
		p := filepath.Join(dir, name)
		if _, err := os.Stat(p); os.IsNotExist(err) {
			return p, nil
		} else if err != nil {
			return "", err
		}
	}
}
