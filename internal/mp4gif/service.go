// Package mp4gif exposes the video→GIF tool to the page: dialogs, progress
// events, cancellation, preview and saving. All ffmpeg interaction lives in
// the ffmpeg module; this package only bridges it to IPC.
package mp4gif

import (
	"context"
	"errors"
	"sync"

	"github.com/sdttttt/m-tools/ffmpeg"
)

// Mp4Gif converts videos to animated GIFs. All ffmpeg interaction lives in
// the ffmpeg module; this service only bridges it to IPC: dialogs, progress
// events, cancel, preview and save-as.
type Service struct {
	mu     sync.Mutex // guards cancel
	cancel context.CancelFunc

	tmu    sync.Mutex // guards client
	client *ffmpeg.Client
}

// tools returns the ffmpeg client, resolving it on first use. A failed
// lookup is not cached, so installing ffmpeg later works without restart.
func (s *Service) tools() (*ffmpeg.Client, error) {
	s.tmu.Lock()
	defer s.tmu.Unlock()
	if s.client != nil {
		return s.client, nil
	}
	c, err := ffmpeg.New()
	if err != nil {
		return nil, err
	}
	s.client = c
	return c, nil
}

// begin registers cancel as the running job. Only one job runs at a time,
// which matches the page: a conversion or a preview, never both.
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

// CancelActive stops the running conversion or preview, if any. One job
// runs at a time, which is the natural model for a desktop toolbox.
func (s *Service) CancelActive() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		s.cancel()
	}
}
