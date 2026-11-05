package ffmpeg

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
)

// EnvDir overrides every other lookup step with a directory holding
// ffmpeg/ffprobe; useful for development and tests.
const EnvDir = "MTOOLS_FFMPEG_DIR"

var (
	mu       sync.Mutex
	resolver func() (string, error) // registered via UseBundledDir
)

// UseBundledDir registers a function that materializes ffmpeg binaries
// bundled at build time and returns the directory holding them. The host
// application calls this from an init when built with its bundling build
// tag; without it the package falls back to environment, sibling bin
// directory and PATH lookups.
func UseBundledDir(f func() (string, error)) {
	mu.Lock()
	defer mu.Unlock()
	resolver = f
}

// findTools returns paths to the ffmpeg and ffprobe executables.
func findTools() (ffmpeg, ffprobe string, err error) {
	if dir := os.Getenv(EnvDir); dir != "" {
		f, p, ok := findInDir(dir)
		if !ok {
			return "", "", fmt.Errorf("%s=%s: 未找到 ffmpeg/ffprobe", EnvDir, dir)
		}
		return f, p, nil
	}

	mu.Lock()
	r := resolver
	mu.Unlock()
	if r != nil {
		if dir, err := r(); err == nil {
			if f, p, ok := findInDir(dir); ok {
				return f, p, nil
			}
		}
	}

	if exe, err := os.Executable(); err == nil {
		if f, p, ok := findInDir(filepath.Join(filepath.Dir(exe), "bin")); ok {
			return f, p, nil
		}
	}

	f, err := exec.LookPath(binName("ffmpeg"))
	if err != nil {
		return "", "", errors.New("未找到 ffmpeg：请安装后加入 PATH，或使用应用的捆绑版本")
	}
	p, err := exec.LookPath(binName("ffprobe"))
	if err != nil {
		return "", "", errors.New("未找到 ffprobe：请安装完整的 ffmpeg（含 ffprobe）")
	}
	return f, p, nil
}

func findInDir(dir string) (ffmpeg, ffprobe string, ok bool) {
	f, err1 := exec.LookPath(filepath.Join(dir, binName("ffmpeg")))
	p, err2 := exec.LookPath(filepath.Join(dir, binName("ffprobe")))
	return f, p, err1 == nil && err2 == nil
}

func binName(base string) string {
	if runtime.GOOS == "windows" {
		return base + ".exe"
	}
	return base
}
