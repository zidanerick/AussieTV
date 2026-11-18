//go:build windows

package ffmpegexec

import (
	"os/exec"
	"syscall"
)

// Configure avoids showing a new console window each time the tray app
// launches FFmpeg for an MPEG-TS stream.
func Configure(cmd *exec.Cmd) {
	if cmd != nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	}
}
