//go:build windows

package mp4gif

import "os/exec"

// reveal opens dir in the OS file manager.
func reveal(dir string) error {
	return exec.Command("explorer", dir).Start()
}
