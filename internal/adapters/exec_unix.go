//go:build !windows

package adapters

import (
	"os/exec"
	"syscall"
)

// setpgid marks cmd to run in its own process group on Unix so a timeout
// can kill shell grandchildren too. Windows has no process-group primitive;
// the build-tag split keeps syscall.SysProcAttr.Setpgid out of Windows builds.
func setpgid(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// killProcessGroup kills the command's whole process group on Unix. On
// Windows the group-kill primitive does not exist; the caller's context kill
// terminates the direct child (and, for CommandContext, the process tree via
// job objects) — see the windows stub.
func killProcessGroup(cmd *exec.Cmd) {
	if cmd.Process != nil {
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
}
