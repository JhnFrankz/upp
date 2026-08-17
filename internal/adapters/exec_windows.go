//go:build windows

package adapters

import "os/exec"

// setpgid is a no-op on Windows: the process-group primitive does not exist,
// so the timeout relies on the context kill, which terminates the direct child
// (and, for CommandContext, the process tree via job objects).
func setpgid(*exec.Cmd) {}

// killProcessGroup is a no-op on Windows; see setpgid.
func killProcessGroup(*exec.Cmd) {}
