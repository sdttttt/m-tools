package imgconv

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image"
	"image/draw"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	xdraw "golang.org/x/image/draw"
)

// Convert converts one image to the requested format and writes the result
// next to the source file. The page drives batches by calling this once
// per row; ctx cancellation (CancelActive) stops the current image.
func (s *Service) Convert(ctx context.Context, req ConvertRequest) (ConvertResult, error) {
	target, ok := formats[req.TargetFormat]
	if !ok {
		if req.TargetFormat == "webp" {
			return ConvertResult{}, errWebpUnsupported
		}
		return ConvertResult{}, fmt.Errorf("不支持的目标格式 %q", req.TargetFormat)
	}

	runCtx, cancel := context.WithCancel(ctx)
	if err := s.begin(cancel); err != nil {
		cancel()
		return ConvertResult{}, err
	}
	defer func() {
		cancel()
		s.end()
	}()

	start := time.Now()
	data, err := os.ReadFile(req.Path)
	if err != nil {
		return ConvertResult{}, err
	}

	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		// A webp source on a stub build fails inside image.Decode; give
		// the tag hint instead of the raw decoder error.
		if f, known, perr := probeFormat(data); perr == nil && !known {
			return ConvertResult{}, perr
		} else if f == nil && perr != nil {
			return ConvertResult{}, perr
		} else if f != nil && f.name == "webp" {
			return ConvertResult{}, errWebpUnsupported
		}
		return ConvertResult{}, fmt.Errorf("解码失败：%w", err)
	}
	if err := runCtx.Err(); err != nil {
		return ConvertResult{}, err
	}

	out := scale(src, req.MaxWidth)
	if target.name == "jpg" && alphaPresent(out) {
		out = flattenWhite(out)
	}

	quality := qualityLevels["balanced"]
	if q, ok := qualityLevels[req.Quality]; ok {
		quality = q
	}

	var buf bytes.Buffer
	if err := target.encode(&buf, out, quality); err != nil {
		return ConvertResult{}, fmt.Errorf("编码失败：%w", err)
	}
	if err := runCtx.Err(); err != nil {
		return ConvertResult{}, err
	}

	outputPath, err := outputFor(req.Path, target.ext)
	if err != nil {
		return ConvertResult{}, err
	}
	if err := os.WriteFile(outputPath, buf.Bytes(), 0o644); err != nil {
		return ConvertResult{}, err
	}

	res := ConvertResult{
		OutputPath: outputPath,
		Format:     target.name,
		SizeBytes:  int64(buf.Len()),
		Width:      out.Bounds().Dx(),
		Height:     out.Bounds().Dy(),
		DurationMs: time.Since(start).Milliseconds(),
	}
	if buf.Len() <= previewLimit {
		res.PreviewDataURL = "data:image/" + previewMime(target.name) + ";base64," +
			base64.StdEncoding.EncodeToString(buf.Bytes())
	}
	return res, nil
}

// previewMime maps format ids to the mime subtype used in data URLs
// (jpg's subtype is jpeg).
func previewMime(name string) string {
	if name == "jpg" {
		return "jpeg"
	}
	return name
}

// scale shrinks m so its width is at most maxWidth, keeping the aspect
// ratio. 0 or a width above the source keeps m untouched; never upscales.
func scale(m image.Image, maxWidth int) image.Image {
	if maxWidth <= 0 || m.Bounds().Dx() <= maxWidth {
		return m
	}
	h := m.Bounds().Dy() * maxWidth / m.Bounds().Dx()
	dst := image.NewNRGBA(image.Rect(0, 0, maxWidth, max(h, 1)))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), m, m.Bounds(), xdraw.Over, nil)
	return dst
}

// flattenWhite composites m over opaque white, for opaque targets like
// JPEG. Exported files must not depend on the window theme, so white.
func flattenWhite(m image.Image) image.Image {
	b := m.Bounds()
	dst := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(dst, dst.Bounds(), &image.Uniform{image.White}, image.Point{}, draw.Src)
	draw.Draw(dst, dst.Bounds(), m, b.Min, draw.Over)
	return dst
}

// outputFor returns the default output path: next to the source, same stem,
// new extension; a name clash adds -1, -2… before the extension.
func outputFor(src, ext string) (string, error) {
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
