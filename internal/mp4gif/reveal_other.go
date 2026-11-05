//go:build !windows && !darwin

package mp4gif

import "os/exec"

// reveal opens dir in the OS file manager.
func reveal(dir string) error {
	return exec.Command("xdg-open", dir).Start()
}
