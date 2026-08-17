//go:build windows

package official

import "os/exec"

// setpgid is a no-op on Windows: the process-group primitive does not exist,
// so the timeout relies on the context kill, which terminates the direct child
// (and, for CommandContext, the process tree via job objects).
func setpgid(*exec.Cmd) {}
