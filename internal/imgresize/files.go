package imgresize

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo"
	"github.com/sdttttt/m-tools/internal/reveal"
)

// pickFilters lists every format this tool can read; webp rides along so
// users can still pick webp files on stub builds (the row then reports the
// missing codec instead of hiding the format entirely).
var pickFilters = []mygo.FileFilter{
	{Name: "图片", Extensions: []string{"png", "jpg", "jpeg", "gif", "webp", "bmp", "tif", "tiff"}},
	{Name: "所有文件", Extensions: []string{"*"}},
}

// PickImages shows a native multi-select dialog and returns the chosen
// paths, or nil when canceled.
func (s *Service) PickImages() ([]string, error) {
	paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
		Title:    "选择图片",
		Filters:  pickFilters,
		Multiple: true,
	})
	if err != nil {
		return nil, err
	}
	return paths, nil
}

// SaveAs shows a native save dialog and copies the resized image there.
// Returns the destination path, or "" when canceled.
func (s *Service) SaveAs(source, defaultName string) (string, error) {
	ext := filepath.Ext(defaultName)
	filterName, filterExt := saveFilterFor(ext)
	dest, err := mygo.Dialog.Save(mygo.SaveDialogOptions{
		Title:       "保存图片",
		DefaultPath: defaultName,
		Filters: []mygo.FileFilter{
			{Name: filterName, Extensions: []string{filterExt}},
			{Name: "所有文件", Extensions: []string{"*"}},
		},
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

// saveFilterFor picks the dialog filter for an output extension.
func saveFilterFor(ext string) (string, string) {
	switch strings.ToLower(strings.TrimPrefix(ext, ".")) {
	case "jpg", "jpeg":
		return "JPEG 图片", "jpg"
	case "png":
		return "PNG 图片", "png"
	case "gif":
		return "GIF 图片", "gif"
	case "webp":
		return "WebP 图片", "webp"
	case "bmp":
		return "BMP 图片", "bmp"
	case "tif", "tiff":
		return "TIFF 图片", "tiff"
	}
	return "图片", strings.TrimPrefix(ext, ".")
}

// OpenFolder reveals the file's folder in the OS file manager.
func (s *Service) OpenFolder(path string) error {
	return reveal.Open(filepath.Dir(path))
}
