package adapters

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// execReapDelay bounds how long Wait waits for the command's output pipes to
// close after the command itself has exited. A descendant holding the pipes
// (e.g. an npm/pnpm worker) must not be able to block the caller forever.
const execReapDelay = 5 * time.Second

// Test seams (D1): package-level function variables swapped by tests via
// setExecFakes so CustomAdapter can be exercised hermetically without
// executing real subprocesses. Production behavior is preserved — the vars
// initialize to real implementations and are only replaced inside tests.
var (
	shellExecWithTimeoutFn = defaultShellExecWithTimeout
	lookPathFn             = exec.LookPath
)

// defaultShellExecWithTimeout runs a command via the platform shell and kills it —
// including its whole process group on Unix — once timeout expires, so
// pipeline/grandchild work (curl|tar, sudo apt, brew...) cannot outlive the
// deadline. The returned error is errors.Is-detectable as
// context.DeadlineExceeded. On Windows only the direct child is terminated.
// Delegates to the shared RunCommandWithTimeout implementation.
func defaultShellExecWithTimeout(ctx context.Context, command string, timeout time.Duration) (string, string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd", "/C", command)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", command)
		// Own process group so the timeout can kill shell grandchildren too.
		setpgid(cmd)
	}

	stdout, stderr, err := RunCommandWithTimeout(ctx, cmd)
	return strings.TrimSpace(stdout), strings.TrimSpace(stderr), err
}

// RunCommandWithTimeout runs a started command under ctx, killing the whole
// process group (Unix) when the context expires, and returns its stdout,
// stderr and error. It is the single implementation of the exec-timeout
// pattern used by the three seams (custom shellExecWithTimeout, official
// runCmdFn/runCmdArgsFn).
//
// Guarantees:
//   - On timeout the returned error is errors.Is(err, context.DeadlineExceeded)-
//     detectable, deterministically — including when the child exits at the
//     same instant the deadline fires (the pre-shared select raced on the
//     both-ready case).
//   - The process group is killed only while the command may still be alive:
//     in the ctx.Done branch (Wait has not returned) and on ErrWaitDelay
//     (the child closed its pipes but descendants may still run). A completed
//     Wait is never followed by a group kill, so a reaped PID cannot be
//     signaled (no PID-reuse window).
//   - cmd.WaitDelay is set here so a descendant holding the output pipes
//     cannot hang Wait past execReapDelay.
//
// The command must be built with Setpgid on Unix (exec.CommandContext does
// not create a process group by itself) for the group kill to reach
// descendants; callers that skip Setpgid still get a bounded direct-child
// kill via the context.
func RunCommandWithTimeout(ctx context.Context, cmd *exec.Cmd) (stdout, stderr string, err error) {
	var stdoutBuf, stderrBuf bytes.Buffer

	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf
	cmd.WaitDelay = execReapDelay

	cmd.Cancel = func() error {
		killProcessGroup(cmd)
		if cmd.Process != nil {
			return cmd.Process.Kill()
		}
		return nil
	}

	err = cmd.Run()
	if err != nil && ctx.Err() != nil {
		return stdoutBuf.String(), stderrBuf.String(), fmt.Errorf("%w: %v", ctx.Err(), err)
	}
	if err != nil {
		return stdoutBuf.String(), stderrBuf.String(), err
	}
	return stdoutBuf.String(), stderrBuf.String(), nil
}
