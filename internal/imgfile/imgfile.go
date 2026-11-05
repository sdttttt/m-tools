// Package imgfile holds the file-interaction bits shared by the image
// tools: the pick/save dialog filters, the SaveAs copy flow and the
// reveal-in-folder action. Tool services keep their own IPC-facing
// wrappers so each generated client documents its own methods.
package imgfile

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo"
	"github.com/sdttttt/m-tools/internal/reveal"
)

// Filters lists every image format the tools can read; webp rides along so
// users can still pick webp files on stub builds (the row then reports the
// missing codec instead of hiding the format entirely).
func Filters() []mygo.FileFilter {
	return []mygo.FileFilter{
		{Name: "图片", Extensions: []string{"png", "jpg", "jpeg", "gif", "webp", "bmp", "tif", "tiff"}},
		{Name: "所有文件", Extensions: []string{"*"}},
	}
}

// PickImages shows a native multi-select dialog and returns the chosen
// paths, or nil when canceled.
func PickImages() ([]string, error) {
	paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
		Title:    "选择图片",
		Filters:  Filters(),
		Multiple: true,
	})
	if err != nil {
		return nil, err
	}
	return paths, nil
}

// SaveAs shows a native save dialog and copies source there. Returns the
// destination path, or "" when canceled.
func SaveAs(source, defaultName string) (string, error) {
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
	return copyFile(source, dest)
}

// copyFile copies source to dest, removing a partial dest on error.
func copyFile(source, dest string) (string, error) {
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
func OpenFolder(path string) error {
	return reveal.Open(filepath.Dir(path))
}
