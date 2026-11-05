// Package imgconv exposes the image format conversion tool to the page:
// pick images, probe them, convert between formats in memory and save the
// results. Decoders/encoders come from the standard library, golang.org/x/image
// and (on with_webp builds) the webp module; none of that leaks past this
// package — the page only sees IPC structs.
package imgconv

import (
	"context"
	"errors"
	"sync"

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
// (with_webp tag). The page calls it once on mount to disable webp options
// with a hint instead of a runtime error.
func (s *Service) Supported() bool {
	return webp.Supported()
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
