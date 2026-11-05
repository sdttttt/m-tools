//go:build with_ffmpeg

// Bundle ffmpeg/ffprobe into the binary and extract them to a user cache
// directory on first run. Place the binaries in resources/bin before
// building: go build -tags with_ffmpeg
package mtools

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"

	"github.com/sdttttt/m-tools/ffmpeg"
)

//go:embed all:resources/bin
var bundledBin embed.FS

func init() {
	ffmpeg.UseBundledDir(extractBundledBin)
}

// extractBundledBin writes the embedded binaries to a stable cache dir and
// returns it.
func extractBundledBin() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "m-tools", "bin")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	err = fs.WalkDir(bundledBin, "resources/bin", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if filepath.Base(path) == "README.md" {
			return nil
		}
		data, err := bundledBin.ReadFile(path)
		if err != nil {
			return err
		}
		target := filepath.Join(dir, filepath.Base(path))
		if fi, err := os.Stat(target); err == nil && fi.Size() == int64(len(data)) {
			return nil // already extracted, same content
		}
		if err := os.WriteFile(target, data, 0o755); err != nil {
			return err
		}
		if runtime.GOOS != "windows" {
			if err := os.Chmod(target, 0o755); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("解压捆绑的 ffmpeg: %w", err)
	}
	return dir, nil
}
