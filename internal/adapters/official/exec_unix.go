//go:build !windows

package official

import (
	"os/exec"
	"syscall"
)

// setpgid marks cmd to run in its own process group on Unix so a timeout
// can kill shell grandchildren too (curl|tar, sudo apt, brew, npm/pnpm
// workers). Windows has no process-group primitive; the build-tag split keeps
// syscall.SysProcAttr.Setpgid out of Windows builds.
func setpgid(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}
