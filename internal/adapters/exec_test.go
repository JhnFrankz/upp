package adapters

import (
	"context"
	"errors"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"
)

// TestTrustLevel_String covers the human-readable trust label mapping. It pins
// the labels consumers (output renderer) depend on and the fail-closed
// "unknown" fallback for out-of-range values.
func TestTrustLevel_String(t *testing.T) {
	tests := []struct {
		name  string
		level TrustLevel
		want  string
	}{
		{"official", TrustOfficial, "official"},
		{"custom-trusted", TrustCustomTrusted, "custom-trusted"},
		{"custom-untrusted", TrustCustomUntrusted, "custom-untrusted"},
		{"unknown fallback", TrustLevel(99), "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.level.String(); got != tt.want {
				t.Errorf("TrustLevel(%d).String() = %q, want %q", tt.level, tt.want, got)
			}
		})
	}
}

// TestRunCommandWithTimeout_Success covers the happy path: stdout and stderr
// are captured, no error is returned for a quick command.
func TestRunCommandWithTimeout_Success(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell success path skipped on windows")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", "echo hello; echo oops >&2")
	stdout, stderr, err := RunCommandWithTimeout(ctx, cmd)
	if err != nil {
		t.Fatalf("RunCommandWithTimeout() error = %v", err)
	}
	if strings.TrimSpace(stdout) != "hello" {
		t.Errorf("stdout = %q, want %q", stdout, "hello")
	}
	if strings.TrimSpace(stderr) != "oops" {
		t.Errorf("stderr = %q, want %q", stderr, "oops")
	}
}

// TestRunCommandWithTimeout_StartError covers the cmd.Start failure path:
// the error is returned and outputs stay empty.
func TestRunCommandWithTimeout_StartError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "definitely-not-a-real-binary-xyz-123")
	_, _, err := RunCommandWithTimeout(ctx, cmd)
	if err == nil {
		t.Fatal("RunCommandWithTimeout() error = nil, want start error")
	}
}

// TestRunCommandWithTimeout_KillsOnDeadline proves the deadline branch of
// RunCommandWithTimeout: a hung command is killed once the context expires
// and the returned error stays errors.Is-detectable as
// context.DeadlineExceeded. (Reaping is verified by the grandchild test and
// by the official timeout tests; reading cmd.ProcessState from outside would
// race with the internal Wait channel.)
func TestRunCommandWithTimeout_KillsOnDeadline(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("process-group kill verification requires unix")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", "sleep 30")
	setpgid(cmd)

	_, _, err := RunCommandWithTimeout(ctx, cmd)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("RunCommandWithTimeout() error = %v, want errors.Is(err, context.DeadlineExceeded)", err)
	}
}

// TestRunCommandWithTimeout_GroupKillReachesGrandchild proves the process-group
// kill reaches shell grandchildren, not just the direct shell child: after the
// deadline, a unique background marker started by the shell must be gone.
func TestRunCommandWithTimeout_GroupKillReachesGrandchild(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("process-group kill verification requires linux")
	}
	if _, err := exec.LookPath("pgrep"); err != nil {
		t.Skip("pgrep not available")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	marker := "sleep 30.41" // unique marker; only this test runs it
	cmd := exec.CommandContext(ctx, "sh", "-c", marker+" & wait")
	setpgid(cmd)

	_, _, err := RunCommandWithTimeout(ctx, cmd)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("RunCommandWithTimeout() error = %v, want DeadlineExceeded", err)
	}
	waitForGroupKillMarkerGone(t, marker)
}

// TestDefaultShellExecWithTimeout_TrimsOutput covers the default seam body
// directly: it runs a real quick command and verifies the captured stdout is
// trimmed (the seam trims whitespace before returning).
func TestDefaultShellExecWithTimeout_TrimsOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell output path skipped on windows")
	}

	out, err := defaultShellExecWithTimeout("echo hello", 5*time.Second)
	if err != nil {
		t.Fatalf("defaultShellExecWithTimeout() error = %v", err)
	}
	if out != "hello" {
		t.Errorf("defaultShellExecWithTimeout() = %q, want %q", out, "hello")
	}
}

// TestDefaultShellExecWithTimeout_KillsOnDeadline proves the default seam body
// kills a hung shell command once the timeout expires.
func TestDefaultShellExecWithTimeout_KillsOnDeadline(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("process-group kill verification requires unix")
	}

	_, err := defaultShellExecWithTimeout("sleep 30", 100*time.Millisecond)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("defaultShellExecWithTimeout() error = %v, want errors.Is(err, context.DeadlineExceeded)", err)
	}
}

// waitForGroupKillMarkerGone polls pgrep until the marker process is gone.
// pgrep exits non-zero when nothing matches, which is the success signal.
func waitForGroupKillMarkerGone(t *testing.T, marker string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		out, err := exec.Command("pgrep", "-f", marker).Output()
		if err != nil || len(out) == 0 {
			return // pgrep found nothing: the marker is gone
		}
		if time.Now().After(deadline) {
			t.Errorf("orphaned grandchild survived the timeout: %s", strings.TrimSpace(string(out)))
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
}
