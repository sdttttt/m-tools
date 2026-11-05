// m-tools: a personal toolbox desktop app for macOS and Windows,
// built with MyGo (webview) + SolidJS.
package main

import (
	"context"
	"encoding/base64"
	"errors"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo"
	mtools "github.com/sdttttt/m-tools"
	"github.com/sdttttt/m-tools/ffmpeg"
)

// The mtools import runs the root package's init hooks (bundled ffmpeg
// on a with_ffmpeg build) and provides shared app metadata.

// Version is the app version, kept in sync with mygo.config.ts.
const Version = "0.1.0"

// SystemInfo describes the machine the app runs on.
type SystemInfo struct {
	OS        string `json:"os"`
	Arch      string `json:"arch"`
	GoVersion string `json:"goVersion"`
	AppVer    string `json:"appVer"`
}

// System is callable from the frontend: `mygo generate` turns its methods
// into typed TypeScript functions in src/mygo.ts.
type System struct{}

// Info returns information about the host system and this app build.
func (System) Info() SystemInfo {
	return SystemInfo{
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
		GoVersion: runtime.Version(),
		AppVer:    Version,
	}
}

// windowColors mirror the page's --background tokens in webview/src/index.css.
const (
	windowColorLight = "#ffffff"
	windowColorDark  = "#09090b"
)

// SetWindowTheme syncs the native window chrome with the page theme mode
// (light | dark | system): the background fill shown until the page paints,
// and the title-bar buttons theme on macOS.
func (System) SetWindowTheme(ctx context.Context, mode string) error {
	win := mygo.CallerWindow(ctx)
	if win == nil {
		return nil
	}
	switch mode {
	case "light":
		_ = win.SetBackgroundColor(windowColorLight)
		win.SetWindowButtonsTheme(mygo.ThemeLight)
	case "dark":
		_ = win.SetBackgroundColor(windowColorDark)
		win.SetWindowButtonsTheme(mygo.ThemeDark)
	default:
		_ = win.SetBackgroundColor(
			"light-dark(" + windowColorLight + ", " + windowColorDark + ")")
		win.SetWindowButtonsTheme(mygo.ThemeSystem)
	}
	return nil
}

// mp4GifProgress streams conversion progress to the page. The payload is
// the ffmpeg module's GIFProgress (json: stage/percent).
var mp4GifProgress = mygo.NewEvent[ffmpeg.GIFProgress]("mp4gifProgress")

// ConvertRequest asks for one video→GIF conversion.
type ConvertRequest struct {
	Path        string  `json:"path"`
	StartSec    float64 `json:"startSec"`
	DurationSec float64 `json:"durationSec"`
	FPS         int     `json:"fps"`
	MaxWidth    int     `json:"maxWidth"`
	Quality     string  `json:"quality"` // high | balanced | compact
}

// ConvertResult reports a finished conversion.
type ConvertResult struct {
	OutputPath     string `json:"outputPath"`
	SizeBytes      int64  `json:"sizeBytes"`
	Width          int    `json:"width"`
	Height         int    `json:"height"`
	Frames         int    `json:"frames"`
	DurationMs     int64  `json:"durationMs"`
	PreviewDataURL string `json:"previewDataURL"`
}

// previewLimit caps how big a GIF may be inlined as a data URL for preview.
const previewLimit = 8 << 20 // 8 MiB

// Mp4Gif converts videos to animated GIFs. All ffmpeg interaction lives in
// the ffmpeg module; this service only bridges it to IPC: dialogs, progress
// events, cancel, preview and save-as.
type Mp4Gif struct {
	mu     sync.Mutex // guards cancel
	cancel context.CancelFunc

	tmu    sync.Mutex // guards client
	client *ffmpeg.Client
}

// tools returns the ffmpeg client, resolving it on first use. A failed
// lookup is not cached, so installing ffmpeg later works without restart.
func (s *Mp4Gif) tools() (*ffmpeg.Client, error) {
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

// PickVideo shows a native dialog and returns the chosen video path,
// or "" when canceled.
func (s *Mp4Gif) PickVideo() (string, error) {
	paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
		Title: "选择视频文件",
		Filters: []mygo.FileFilter{
			{Name: "视频", Extensions: []string{"mp4", "mov", "m4v", "mkv", "webm", "avi"}},
			{Name: "所有文件", Extensions: []string{"*"}},
		},
	})
	if err != nil {
		return "", err
	}
	if len(paths) == 0 {
		return "", nil
	}
	return paths[0], nil
}

// Probe reads metadata of a video file.
func (s *Mp4Gif) Probe(ctx context.Context, path string) (ffmpeg.VideoInfo, error) {
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
func (s *Mp4Gif) Thumbnails(ctx context.Context, path string) ([]string, error) {
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

// SuggestOutput returns a default GIF path next to the input file.
func (s *Mp4Gif) SuggestOutput(input string) string {
	ext := filepath.Ext(input)
	return strings.TrimSuffix(input, ext) + ".gif"
}

// Convert runs the conversion to completion and reports the result.
// Progress streams through the mp4gifProgress event while it runs.
func (s *Mp4Gif) Convert(ctx context.Context, req ConvertRequest) (ConvertResult, error) {
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
	s.mu.Lock()
	s.cancel = cancel
	s.mu.Unlock()
	defer func() {
		cancel()
		s.mu.Lock()
		s.cancel = nil
		s.mu.Unlock()
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

// CancelActive stops the running conversion, if any. One conversion runs at
// a time, which is the natural model for a desktop toolbox.
func (s *Mp4Gif) CancelActive() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		s.cancel()
	}
}

// SaveAs shows a native save dialog and copies the converted GIF there.
// Returns the destination path, or "" when canceled.
func (s *Mp4Gif) SaveAs(source, defaultName string) (string, error) {
	dest, err := mygo.Dialog.Save(mygo.SaveDialogOptions{
		Title:       "保存 GIF",
		DefaultPath: defaultName,
		Filters:     []mygo.FileFilter{{Name: "GIF 动图", Extensions: []string{"gif"}}},
	})
	if err != nil {
		return "", err
	}
	if dest == "" {
		return "", nil
	}
	if dest == source {
		return dest, nil
	}
	src, err := os.Open(source)
	if err != nil {
		return "", err
	}
	defer src.Close()
	dst, err := os.Create(dest)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		os.Remove(dest)
		return "", err
	}
	return dest, dst.Close()
}

// OpenFolder reveals the file's folder in the OS file manager.
func (s *Mp4Gif) OpenFolder(path string) error {
	dir := filepath.Dir(path)
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", dir)
	case "darwin":
		cmd = exec.Command("open", dir)
	default:
		cmd = exec.Command("xdg-open", dir)
	}
	return cmd.Start()
}

func main() {
	mygo.Bind(System{}, &Mp4Gif{})

	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:     mtools.AppName,
			Width:     960,
			Height:    640,
			MinWidth:  480,
			MinHeight: 360,
			// The page's --background tokens (webview/src/index.css); the
			// same pair SetWindowTheme applies when the user picks a mode.
			BackgroundColor: "light-dark(" + windowColorLight + ", " + windowColorDark + ")",
			// Opens where the user left it last time.
			StateKey: "main",
			// The frontend: devUrl during `mygo dev`, frontendDist
			// embedded by `mygo build` otherwise (see mygo.config.ts).
			URL: "/",
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
