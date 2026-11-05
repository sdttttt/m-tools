//go:build !windows && !darwin

package reveal

import "os/exec"

func open(dir string) error {
	return exec.Command("xdg-open", dir).Start()
}
