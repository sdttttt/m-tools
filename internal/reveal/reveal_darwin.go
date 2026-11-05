//go:build darwin

package reveal

import "os/exec"

func open(dir string) error {
	return exec.Command("open", dir).Start()
}
