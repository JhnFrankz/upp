package adapters

import (
	"errors"
	"fmt"
	"os/exec"
	"reflect"
	"testing"
	"time"
)

// fakeResult is the canned output of a mocked command.
type fakeResult struct {
	stdout string
	err    error
}

// execFakes holds per-command and lookPath fakes.
type execFakes struct {
	shell    map[string]fakeResult
	lookPath map[string]bool
}

// setExecFakes swaps the package exec seam variables (shellExecWithTimeoutFn,
// lookPathFn) for the duration of the test and restores the real implementations
// via t.Cleanup. No real subprocess ever runs.
func setExecFakes(t *testing.T, f execFakes) {
	t.Helper()

	origShellExecWithTimeout := shellExecWithTimeoutFn
	origLookPath := lookPathFn

	shellExecWithTimeoutFn = func(command string, timeout time.Duration) (string, error) {
		if f.shell != nil {
			if r, ok := f.shell[command]; ok {
				return r.stdout, r.err
			}
		}
		return "", fmt.Errorf("command not mocked: %s", command)
	}

	lookPathFn = func(name string) (string, error) {
		if f.lookPath != nil {
			if ok, present := f.lookPath[name]; present {
				if ok {
					return "/fake/bin/" + name, nil
				}
				return "", &exec.Error{Name: name, Err: exec.ErrNotFound}
			}
		}
		return "", &exec.Error{Name: name, Err: exec.ErrNotFound}
	}

	t.Cleanup(func() {
		shellExecWithTimeoutFn = origShellExecWithTimeout
		lookPathFn = origLookPath
	})
}

func TestExecFakes_Isolation(t *testing.T) {
	origShell := reflect.ValueOf(shellExecWithTimeoutFn).Pointer()
	origLookPath := reflect.ValueOf(lookPathFn).Pointer()

	t.Run("isolated", func(t *testing.T) {
		setExecFakes(t, execFakes{
			shell: map[string]fakeResult{
				"mock-cmd": {stdout: "mocked output", err: nil},
				"err-cmd":  {stdout: "", err: errors.New("exec error")},
			},
			lookPath: map[string]bool{
				"mock-bin": true,
				"miss-bin": false,
			},
		})

		// Test intercepted shellExecWithTimeoutFn
		out, err := shellExecWithTimeoutFn("mock-cmd", 1*time.Second)
		if err != nil || out != "mocked output" {
			t.Errorf("shellExecWithTimeoutFn(mock-cmd) = (%q, %v), want (%q, nil)", out, err, "mocked output")
		}

		_, err = shellExecWithTimeoutFn("err-cmd", 1*time.Second)
		if err == nil || err.Error() != "exec error" {
			t.Errorf("shellExecWithTimeoutFn(err-cmd) err = %v, want 'exec error'", err)
		}

		// Test intercepted lookPathFn
		path, err := lookPathFn("mock-bin")
		if err != nil || path == "" {
			t.Errorf("lookPathFn(mock-bin) = (%q, %v), want present path", path, err)
		}

		_, err = lookPathFn("miss-bin")
		if err == nil {
			t.Errorf("lookPathFn(miss-bin) expected error, got nil")
		}
	})

	// Verify restored after subtest cleanup
	currShell := reflect.ValueOf(shellExecWithTimeoutFn).Pointer()
	currLookPath := reflect.ValueOf(lookPathFn).Pointer()

	if currShell != origShell {
		t.Errorf("shellExecWithTimeoutFn not restored after Cleanup: got %v, want %v", currShell, origShell)
	}
	if currLookPath != origLookPath {
		t.Errorf("lookPathFn not restored after Cleanup: got %v, want %v", currLookPath, origLookPath)
	}
}
