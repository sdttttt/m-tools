//go:build windows

package ffmpeg

import (
	"os/exec"
	"testing"
)

func TestHideConsole(t *testing.T) {
	cmd := exec.Command("ffmpeg")
	hideConsole(cmd)
	if cmd.SysProcAttr == nil {
		t.Fatal("SysProcAttr is nil")
	}
	if cmd.SysProcAttr.CreationFlags&createNoWindow == 0 {
		t.Errorf("CreationFlags = %#x, want CREATE_NO_WINDOW (%#x) set",
			cmd.SysProcAttr.CreationFlags, createNoWindow)
	}
}
