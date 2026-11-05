package mp4gif

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo"
)

// PickVideo shows a native dialog and returns the chosen video path,
// or "" when canceled.
func (s *Service) PickVideo() (string, error) {
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

// SuggestOutput returns a default GIF path next to the input file.
func (s *Service) SuggestOutput(input string) string {
	ext := filepath.Ext(input)
	return strings.TrimSuffix(input, ext) + ".gif"
}

// SaveAs shows a native save dialog and copies the converted GIF there.
// Returns the destination path, or "" when canceled.
func (s *Service) SaveAs(source, defaultName string) (string, error) {
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
func (s *Service) OpenFolder(path string) error {
	return reveal(filepath.Dir(path))
}
