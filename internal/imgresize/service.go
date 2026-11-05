// Package imgresize exposes the image size adjustment tool to the page:
// pick images, set an explicit pixel size (or a percentage of the source)
// and optionally convert to another format in the same pass. Resizing
// shares the imgfmt pipeline; this service adds the resize-specific math
// (percent vs exact size, aspect locking) and the same one-job-at-a-time
// semantics as the other tools.
package imgresize

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/sdttttt/m-tools/internal/imgfmt"
	"github.com/sdttttt/m-tools/webp"
)

func init() {
	// Registers webp with image.Decode on with_webp builds; a no-op
	// otherwise (see imgconv's init for the reasoning).
	webp.Register()
}

// maxPercent caps the percentage input so a typo cannot ask for a
// gigapixel image; 400% of a photo is already generous.
const maxPercent = 400

// Service is the image size adjustment tool. One job at a time, matching
// the other toolbox services.
type Service struct {
	mu     sync.Mutex // guards cancel
	cancel context.CancelFunc
}

// Supported reports whether WebP decode/encode is compiled into this build
// (with_webp tag).
func (s *Service) Supported() bool {
	return imgfmt.Supported()
}

// TargetNames lists every format the page can offer as an output format;
// WebP appears only on with_webp builds.
func (s *Service) TargetNames() []string {
	return imgfmt.TargetNames()
}

// Probe reads metadata of one picked image. It never errors: unsupported
// sources come back flagged so the page can show them as skipped rows.
func (s *Service) Probe(path string) (imgfmt.Info, error) {
	return imgfmt.Probe(path), nil
}

// begin/end/CancelActive guard the single running job.
func (s *Service) begin(cancel context.CancelFunc) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		return errors.New("已有任务在运行，请先取消")
	}
	s.cancel = cancel
	return nil
}

func (s *Service) end() {
	s.mu.Lock()
	s.cancel = nil
	s.mu.Unlock()
}

// CancelActive stops the running resize, if any.
func (s *Service) CancelActive() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		s.cancel()
	}
}

// resolveSize computes the output pixel size from the request: either an
// explicit width/height, or a percentage of the source size (srcW×srcH).
// When lock is set and only one axis is given explicitly, the other
// follows the source aspect ratio. Percent mode always keeps the ratio.
func resolveSize(mode string, width, height, percent int, lock bool, srcW, srcH int) (int, int, error) {
	if mode == "percent" {
		if percent <= 0 || percent > maxPercent {
			return 0, 0, fmt.Errorf("比例需在 1–%d 之间", maxPercent)
		}
		width = srcW * percent / 100
		height = srcH * percent / 100
	} else {
		if width < 0 || height < 0 {
			return 0, 0, fmt.Errorf("宽高不能为负数")
		}
		if lock && (width == 0) != (height == 0) {
			// One axis given, aspect locked: derive the other from the
			// source ratio. A derived axis of 0 (degenerate source) stays 0
			// and keeps the source size.
			if width > 0 {
				height = roundRatio(srcH, srcW, width)
			} else if height > 0 {
				width = roundRatio(srcW, srcH, height)
			}
		}
	}
	if width <= 0 || height <= 0 {
		width, height = 0, 0
	}
	return width, height, nil
}

// roundRatio scales base by factor/base with rounding (integer ratio math,
// so 300×200 at width 901 → 601, not 600 from stacked truncation).
func roundRatio(base, factor, value int) int {
	if factor <= 0 {
		return 0
	}
	return (base*value + factor/2) / factor
}

// Resize adjusts one image's size (and optionally its format) and writes
// the result next to the source file. ctx cancellation (CancelActive)
// stops the current image.
func (s *Service) Resize(ctx context.Context, req ResizeRequest) (ResizeResult, error) {
	if !imgfmt.HasTarget(req.TargetFormat) {
		if req.TargetFormat == "webp" {
			return ResizeResult{}, imgfmt.ErrUnsupported
		}
		return ResizeResult{}, fmt.Errorf("不支持的目标格式 %q", req.TargetFormat)
	}
	info := imgfmt.Probe(req.Path)
	if !info.Supported {
		if info.SkipReason == imgfmt.ErrUnsupported.Error() {
			return ResizeResult{}, imgfmt.ErrUnsupported
		}
		return ResizeResult{}, fmt.Errorf("%s", info.SkipReason)
	}

	width, height, err := resolveSize(req.Mode, req.Width, req.Height, req.Percent, req.Lock, info.Width, info.Height)
	if err != nil {
		return ResizeResult{}, err
	}

	runCtx, cancel := context.WithCancel(ctx)
	if err := s.begin(cancel); err != nil {
		cancel()
		return ResizeResult{}, err
	}
	defer func() {
		cancel()
		s.end()
	}()

	start := time.Now()
	res, err := imgfmt.Run(runCtx, imgfmt.Spec{
		Path:         req.Path,
		TargetFormat: req.TargetFormat,
		Quality:      req.Quality,
		Width:        width,
		Height:       height,
	})
	if err != nil {
		return ResizeResult{}, err
	}

	out := ResizeResult{
		OutputPath: res.OutputPath,
		Format:     res.Format,
		SizeBytes:  res.SizeBytes,
		Width:      res.Width,
		Height:     res.Height,
		DurationMs: time.Since(start).Milliseconds(),
	}
	if res.PNGBytes != nil {
		out.PreviewDataURL = "data:image/" + imgfmt.PreviewMime(res.Format) + ";base64," +
			base64.StdEncoding.EncodeToString(res.PNGBytes)
	}
	return out, nil
}
