package adapters

import (
	"bytes"
	"context"
	"errors"
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
func defaultShellExecWithTimeout(command string, timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd", "/C", command)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", command)
		// Own process group so the timeout can kill shell grandchildren too.
		setpgid(cmd)
	}

	stdout, _, err := RunCommandWithTimeout(ctx, cmd)
	return strings.TrimSpace(stdout), err
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
//     signalled (no PID-reuse window).
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

	if err := cmd.Start(); err != nil {
		return stdoutBuf.String(), stderrBuf.String(), err
	}

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	select {
	case err := <-done:
		if err != nil && ctx.Err() != nil {
			// The command finished (or closed its pipes) exactly as the
			// caller's context deadline fired. On a deadline-class Wait error
			// the child may still have live descendants holding the pipes:
			// the caller's context deadline (context.DeadlineExceeded) raced
			// the child's successful exit, or WaitDelay expired because the
			// child closed its pipes but descendants keep them open — kill
			// the group so they cannot outlive the timeout. Any other
			// done-branch error means the process is already reaped, so no
			// kill runs (a killed PID would be a stale-PID hazard).
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, exec.ErrWaitDelay) {
				killProcessGroup(cmd)
			}
			// Classify deterministically as a timeout on both paths, matching
			// the pre-select behavior (err != nil && ctx.Err() != nil).
			return stdoutBuf.String(), stderrBuf.String(), fmt.Errorf("%w: %v", ctx.Err(), err)
		}
		return stdoutBuf.String(), stderrBuf.String(), err
	case <-ctx.Done():
		// Wait has not returned, so the process is still ours: kill the whole
		// group so descendants cannot keep holding the pipes (hanging Wait)
		// or keep mutating state after the reported timeout.
		killProcessGroup(cmd)
		<-done // reap; WaitDelay bounds the wait if a descendant holds the pipes
		// Go's exec.Wait returns the raw exit error, so chain the deadline
		// error to stay errors.Is(err, context.DeadlineExceeded)-detectable.
		return stdoutBuf.String(), stderrBuf.String(), fmt.Errorf("%w: %v", ctx.Err(), "killed after timeout")
	}
}
