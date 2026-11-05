package mp4gif

import (
	"context"
	"encoding/base64"
	"errors"
	"os"
	"time"

	"github.com/egoist/mygo"
	"github.com/sdttttt/m-tools/ffmpeg"
)

// Probe reads metadata of a video file.
func (s *Service) Probe(ctx context.Context, path string) (ffmpeg.VideoInfo, error) {
	client, err := s.tools()
	if err != nil {
		return ffmpeg.VideoInfo{}, err
	}
	return client.Probe(ctx, path)
}

// Thumbnail geometry for the trim timeline's filmstrip.
const (
	thumbnailCount = 12
	thumbnailWidth = 160
)

// Thumbnails returns evenly spaced stills of the video as JPEG data URLs,
// for the trim timeline's filmstrip.
func (s *Service) Thumbnails(ctx context.Context, path string) ([]string, error) {
	client, err := s.tools()
	if err != nil {
		return nil, err
	}
	info, err := client.Probe(ctx, path)
	if err != nil {
		return nil, err
	}
	thumbs, err := client.Thumbnails(ctx, path, thumbnailCount, thumbnailWidth, info.DurationSec)
	if err != nil {
		return nil, err
	}
	out := make([]string, len(thumbs))
	for i, data := range thumbs {
		out[i] = "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(data)
	}
	return out, nil
}

// Preview renders the clip range as a GIF data URL for an inline preview,
// honoring the page's quality choice but capping length, width and frame
// rate so it stays quick and small.
func (s *Service) Preview(ctx context.Context, req PreviewRequest) (PreviewResult, error) {
	client, err := s.tools()
	if err != nil {
		return PreviewResult{}, err
	}
	info, err := client.Probe(ctx, req.Path)
	if err != nil {
		return PreviewResult{}, err
	}

	runCtx, cancel := context.WithCancel(ctx)
	if err := s.begin(cancel); err != nil {
		cancel()
		return PreviewResult{}, err
	}
	defer func() {
		cancel()
		s.end()
	}()

	duration := req.DurationSec
	if duration <= 0 || duration > previewMaxSec {
		duration = previewMaxSec
	}
	width := req.MaxWidth
	if width <= 0 || width > previewMaxWidth {
		width = previewMaxWidth
	}
	fps := req.FPS
	if fps <= 0 || fps > previewMaxFPS {
		fps = previewMaxFPS
	}

	start := time.Now()
	res, err := client.ConvertGIF(runCtx, ffmpeg.GIFRequest{
		Input:       req.Path,
		StartSec:    req.StartSec,
		DurationSec: duration,
		TotalSec:    info.DurationSec,
		FPS:         fps,
		MaxWidth:    width,
		Quality:     ffmpeg.Quality(req.Quality),
		SourceFPS:   info.FPS,
		SourceWidth: info.Width,
	}, nil)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return PreviewResult{}, errors.New("预览已取消")
		}
		return PreviewResult{}, err
	}
	defer os.Remove(res.OutputPath)

	data, err := os.ReadFile(res.OutputPath)
	if err != nil {
		return PreviewResult{}, err
	}
	return PreviewResult{
		DataURL:     "data:image/gif;base64," + base64.StdEncoding.EncodeToString(data),
		Width:       res.Width,
		Height:      res.Height,
		Frames:      res.Frames,
		DurationSec: duration,
		SizeBytes:   res.SizeBytes,
		DurationMs:  time.Since(start).Milliseconds(),
	}, nil
}

// Convert runs the conversion to completion and reports the result.
// Progress streams through the mp4gifProgress event while it runs.
func (s *Service) Convert(ctx context.Context, req ConvertRequest) (ConvertResult, error) {
	client, err := s.tools()
	if err != nil {
		return ConvertResult{}, err
	}
	win := mygo.CallerWindow(ctx)
	info, err := client.Probe(ctx, req.Path)
	if err != nil {
		return ConvertResult{}, err
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

	gifReq := ffmpeg.GIFRequest{
		Input:       req.Path,
		StartSec:    req.StartSec,
		DurationSec: req.DurationSec,
		TotalSec:    info.DurationSec,
		FPS:         req.FPS,
		MaxWidth:    req.MaxWidth,
		Quality:     ffmpeg.Quality(req.Quality),
		SourceFPS:   info.FPS,
		SourceWidth: info.Width,
	}

	start := time.Now()
	res, err := client.ConvertGIF(runCtx, gifReq, func(p ffmpeg.GIFProgress) {
		if win != nil {
			_ = mp4GifProgress.Emit(win, p)
		}
	})
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return ConvertResult{}, errors.New("转换已取消")
		}
		return ConvertResult{}, err
	}

	preview := ""
	if res.SizeBytes <= previewLimit {
		if data, err := os.ReadFile(res.OutputPath); err == nil {
			preview = "data:image/gif;base64," + base64.StdEncoding.EncodeToString(data)
		}
	}

	return ConvertResult{
		OutputPath:     res.OutputPath,
		SizeBytes:      res.SizeBytes,
		Width:          res.Width,
		Height:         res.Height,
		Frames:         res.Frames,
		DurationMs:     time.Since(start).Milliseconds(),
		PreviewDataURL: preview,
	}, nil
}
