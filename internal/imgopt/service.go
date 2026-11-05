package imgopt

import (
	"context"
	"encoding/base64"
	"errors"
	"sync"
	"time"

	"github.com/sdttttt/m-tools/internal/imgfmt"
)

// Service is the lossless image optimization tool. Optimizations run one
// at a time (the page drives a batch as a sequence of single Optimize
// calls), guarded by the same begin/end pattern as the other image tools.
type Service struct {
	mu     sync.Mutex // guards cancel
	cancel context.CancelFunc
}

// Supported reports whether WebP decode/encode is compiled into this build
// (with_webp tag). WebP sources are skipped with a hint on default builds.
func (s *Service) Supported() bool {
	return imgfmt.Supported()
}

// Probe reads metadata of one picked image. It never errors: unsupported
// sources come back flagged so the page can show them as skipped rows.
func (s *Service) Probe(path string) (imgfmt.Info, error) {
	return imgfmt.Probe(path), nil
}

// begin registers cancel as the running job. Only one job runs at a time,
// which matches the page: one image optimizing, never two.
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

// CancelActive stops the running optimization, if any.
func (s *Service) CancelActive() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		s.cancel()
	}
}

// Optimize optimizes one image losslessly (pixels stay identical) and
// writes the result as a -min file next to the source. When nothing can
// be shaved the result comes back with NotNeeded set and no file written;
// animated sources and BMP files come back with SkipReason. ctx
// cancellation (CancelActive) stops the current image between stages.
func (s *Service) Optimize(ctx context.Context, req OptimizeRequest) (OptimizeResult, error) {
	runCtx, cancel := context.WithCancel(ctx)
	if err := s.begin(cancel); err != nil {
		cancel()
		return OptimizeResult{}, err
	}
	defer func() {
		cancel()
		s.end()
	}()

	start := time.Now()
	res, err := optimize(runCtx, req.Path)
	if err != nil {
		return OptimizeResult{}, err
	}

	out := OptimizeResult{
		OutputPath:     res.OutputPath,
		Format:         res.Format,
		OriginalBytes:  res.OriginalBytes,
		OptimizedBytes: res.OptimizedBytes,
		DurationMs:     time.Since(start).Milliseconds(),
		NotNeeded:      res.NotNeeded,
		SkipReason:     res.Reason,
	}
	if res.Preview != nil {
		out.PreviewDataURL = "data:image/" + imgfmt.PreviewMime(res.Format) + ";base64," +
			base64.StdEncoding.EncodeToString(res.Preview)
	}
	return out, nil
}
