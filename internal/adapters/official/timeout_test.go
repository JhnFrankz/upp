package official

import (
	"context"
	"errors"
	"runtime"
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
