//go:build windows

package adapters

import (
	"os/exec"
	"strconv"
)

// Setpgid is a no-op on Windows: the process-group primitive does not exist,
// so the timeout relies on the context kill, which terminates the direct child
// (and, for CommandContext, the process tree via job objects).
func Setpgid(*exec.Cmd) {}

func setpgid(cmd *exec.Cmd) {
	Setpgid(cmd)
}

// killProcessGroup terminates the process tree on Windows using taskkill.
func killProcessGroup(cmd *exec.Cmd) {
	if cmd != nil && cmd.Process != nil && cmd.Process.Pid > 0 {
		killCmd := exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(cmd.Process.Pid))
		_ = killCmd.Run()
	}
}
