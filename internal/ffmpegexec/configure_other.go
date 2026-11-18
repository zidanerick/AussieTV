//go:build !windows

package ffmpegexec

import "os/exec"

func Configure(cmd *exec.Cmd) {}
