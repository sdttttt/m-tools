//go:build windows

package ffmpeg

import (
	"os/exec"
	"syscall"
)

// createNoWindow is CreateProcess's CREATE_NO_WINDOW flag: the child is
// started without a console, so a GUI-subsystem app (the release build has
// -H=windowsgui) does not flash a console window for ffmpeg/ffprobe.
const createNoWindow = 0x08000000

func hideConsole(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.CreationFlags |= createNoWindow
}
