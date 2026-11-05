// Package imgconv exposes the image format conversion tool to the page:
// pick images, probe them, convert between formats in memory and save the
// results. The format registry and the decode→flatten→encode pipeline live
// in internal/imgfmt; this service adds the conversion job semantics —
// batch rows driven by the page, one Convert at a time, no resizing.
package imgconv

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
	// otherwise, so image.Decode keeps rejecting webp bytes instead of
	// failing deep inside an unsupported codec path.
	webp.Register()
}

// Service is the image conversion tool. Conversions run one at a time (the
// page drives a batch as a sequence of single Convert calls), guarded by
// the same begin/end pattern as the mp4gif service.
type Service struct {
	mu     sync.Mutex // guards cancel
	cancel context.CancelFunc
}

// Supported reports whether WebP decode/encode is compiled into this build
// (with_webp tag). The page calls it once on mount to hide webp options
// with a hint instead of a runtime error.
func (s *Service) Supported() bool {
	return imgfmt.Supported()
}

// TargetNames lists every format the page can offer as a conversion
// target; WebP appears only on with_webp builds.
func (s *Service) TargetNames() []string {
	return imgfmt.TargetNames()
}

// Probe reads metadata of one picked image. It never errors: unsupported
// sources come back flagged so the page can show them as skipped rows.
func (s *Service) Probe(path string) (imgfmt.Info, error) {
	return imgfmt.Probe(path), nil
}

// begin registers cancel as the running job. Only one job runs at a time,
// which matches the page: one image converting, never two.
func (s *Service) begin(cancel context.CancelFunc) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		return errors.New("已有任务在运行，请先取消")
	}
	s.cancel = cancel
	return nil
}

// end clears the running job.
func (s *Service) end() {
	s.mu.Lock()
	s.cancel = nil
	s.mu.Unlock()
}

// CancelActive stops the running conversion, if any.
func (s *Service) CancelActive() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		s.cancel()
	}
}

// Convert converts one image to the requested format at its original size
// and writes the result next to the source file. The page drives batches
// by calling this once per row; ctx cancellation (CancelActive) stops the
// current image.
func (s *Service) Convert(ctx context.Context, req ConvertRequest) (ConvertResult, error) {
	if !imgfmt.HasTarget(req.TargetFormat) {
		if req.TargetFormat == "webp" {
			return ConvertResult{}, imgfmt.ErrUnsupported
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
	res, err := imgfmt.Run(runCtx, imgfmt.Spec{
		Path:         req.Path,
		TargetFormat: req.TargetFormat,
		Quality:      req.Quality,
	})
	if err != nil {
		return ConvertResult{}, err
	}

	out := ConvertResult{
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
