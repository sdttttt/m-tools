package imgconv

import (
	"github.com/sdttttt/m-tools/internal/imgfile"
)

// PickImages shows a native multi-select dialog and returns the chosen
// paths, or nil when canceled.
func (s *Service) PickImages() ([]string, error) {
	return imgfile.PickImages()
}

// SaveAs shows a native save dialog and copies the converted image there.
// Returns the destination path, or "" when canceled.
func (s *Service) SaveAs(source, defaultName string) (string, error) {
	return imgfile.SaveAs(source, defaultName)
}

// OpenFolder reveals the file's folder in the OS file manager.
func (s *Service) OpenFolder(path string) error {
	return imgfile.OpenFolder(path)
}
