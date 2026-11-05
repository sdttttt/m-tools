//go:build windows

package reveal

import "os/exec"

func open(dir string) error {
	return exec.Command("explorer", dir).Start()
}
