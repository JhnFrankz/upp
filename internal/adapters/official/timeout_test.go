package official

import (
	"context"
	"errors"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/JhnFrankz/upp/internal/adapters"
)

// TestUpdate_TimeoutErrorPropagates verifies that a DeadlineExceeded error
// produced at the exec seam surfaces in the adapter Result.Error chain and
// stays errors.Is-detectable once the command runs inside a WithTimeout
// context.
func TestUpdate_TimeoutErrorPropagates(t *testing.T) {
	tests := []struct {
		name    string
		newAdpt func() adapters.Adapter
		fakes   execFakes
	}{
		{
			name:    "apt",
			newAdpt: func() adapters.Adapter { return &AptAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"apt": true},
				shell: map[string]fakeResult{
					aptInstalledCmd: {stdout: "2.4.0"},
					aptUpdateCmd:    {err: context.DeadlineExceeded},
				},
			},
		},
		{
			name:    "brew",
			newAdpt: func() adapters.Adapter { return &BrewAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"brew": true},
				cmdArgs:  map[string]fakeResult{"brew": {stdout: "Homebrew 4.1.0"}},
				shell:    map[string]fakeResult{brewUpdateCmd: {err: context.DeadlineExceeded}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setExecFakes(t, tt.fakes)

			result, err := tt.newAdpt().Update(false)
			if err != nil {
				t.Fatalf("Update() unexpected error: %v", err)
			}
			if result.Success {
				t.Error("Update() Success = true, want false on timeout")
			}
			if !errors.Is(result.Error, context.DeadlineExceeded) {
				t.Errorf("Result.Error = %v, want errors.Is(err, context.DeadlineExceeded)", result.Error)
			}
		})
	}
}

// TestRunCmd_UpdateTimeoutKills proves the runCmdFn default seam body really
// kills a hung shell command once the UpdateTimeout context expires. It runs
// the production seam with no fakes: `sleep 2` must be terminated at ~100ms.
func TestRunCmd_UpdateTimeoutKills(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping shell test on windows")
	}

	orig := adapters.UpdateTimeout
	adapters.UpdateTimeout = 100 * time.Millisecond
	t.Cleanup(func() { adapters.UpdateTimeout = orig })

	_, _, err := runCmd("sleep 2")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("runCmd() error = %v, want errors.Is(err, context.DeadlineExceeded)", err)
	}
}

// TestRunCmdArgs_CheckTimeoutKills proves the runCmdArgsFn default seam body
// kills a hung command once the CheckTimeout context expires.
func TestRunCmdArgs_CheckTimeoutKills(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows: sleep is not a cmd.exe builtin")
	}

	orig := adapters.CheckTimeout
	adapters.CheckTimeout = 100 * time.Millisecond
	t.Cleanup(func() { adapters.CheckTimeout = orig })

	_, _, err := runCmdArgs("sleep", "2")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("runCmdArgs() error = %v, want errors.Is(err, context.DeadlineExceeded)", err)
	}
}

// TestRunCmdArgs_GroupKillProvesGrandchildrenDie verifies that the
// process-group kill also reaches descendants of the direct-exec seam (the
// npm/pnpm check path): after the timeout, a unique background marker
// process spawned by the direct child must no longer exist.
func TestRunCmdArgs_GroupKillProvesGrandchildrenDie(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("process-group kill verification requires linux")
	}
	if _, err := exec.LookPath("pgrep"); err != nil {
		t.Skip("pgrep not available")
	}

	orig := adapters.CheckTimeout
	adapters.CheckTimeout = 150 * time.Millisecond
	t.Cleanup(func() { adapters.CheckTimeout = orig })

	marker := "sleep 28.91" // unique marker; only this test ever runs it
	_, _, err := runCmdArgs("sh", "-c", marker+" & wait")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("runCmdArgs() error = %v, want DeadlineExceeded", err)
	}

	// Poll until the SIGKILL has landed; no fixed wait on the green path.
	waitForMarkerGone(t, marker)
}

// TestRunCmd_GroupKillProvesGrandchildrenDie verifies that the process-group
// kill reaches shell grandchildren (pipelines/&& chains), not just the direct
// shell child: after the timeout, a unique background marker process started
// by the shell must no longer exist.
func TestRunCmd_GroupKillProvesGrandchildrenDie(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("process-group kill verification requires linux")
	}
	if _, err := exec.LookPath("pgrep"); err != nil {
		t.Skip("pgrep not available")
	}

	orig := adapters.UpdateTimeout
	adapters.UpdateTimeout = 150 * time.Millisecond
	t.Cleanup(func() { adapters.UpdateTimeout = orig })

	marker := "sleep 29.17" // unique marker; only this test ever runs it
	_, _, err := runCmd("sh -c '" + marker + " & wait'")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("runCmd() error = %v, want DeadlineExceeded", err)
	}

	// Poll until the SIGKILL has landed; no fixed wait on the green path.
	waitForMarkerGone(t, marker)
}

// waitForMarkerGone polls pgrep until the marker process is gone, failing the
// test if it still survives after a 2s deadline. pgrep exits non-zero when
// nothing matches, which is the success signal.
func waitForMarkerGone(t *testing.T, marker string) {
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
