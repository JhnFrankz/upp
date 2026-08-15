package official

import (
	"strings"
	"testing"
)

// fakeResult is the canned output of a mocked command.
type fakeResult struct {
	stdout string
	stderr string
	err    error
}

// execFakes holds per-command fakes keyed the same way the production
// helpers key their lookups: shell by full command string, cmdArgs by
// binary name (or "name arg1 arg2..." for a specific invocation), lookPath
// by binary name.
type execFakes struct {
	shell    map[string]fakeResult
	cmdArgs  map[string]fakeResult
	lookPath map[string]bool
}

// setExecFakes swaps the package exec seam variables (runCmdFn,
// runCmdArgsFn, lookPathFn) for the duration of the test and restores the
// real implementations via t.Cleanup. No real subprocess ever runs.
func setExecFakes(t *testing.T, f execFakes) {
	t.Helper()

	origRunCmd := runCmdFn
	origRunCmdArgs := runCmdArgsFn
	origLookPath := lookPathFn

	runCmdFn = func(command string) (stdout, stderr string, err error) {
		r := f.shell[command]
		return r.stdout, r.stderr, r.err
	}
	runCmdArgsFn = func(name string, args ...string) (stdout, stderr string, err error) {
		// Prefer an invocation-specific key ("name arg1 arg2..."), falling
		// back to the binary-name key for callers that only fake by name.
		key := name
		if len(args) > 0 {
			key = name + " " + strings.Join(args, " ")
		}
		if r, ok := f.cmdArgs[key]; ok {
			return r.stdout, r.stderr, r.err
		}
		r := f.cmdArgs[name]
		return r.stdout, r.stderr, r.err
	}
	lookPathFn = func(name string) bool {
		return f.lookPath[name]
	}

	t.Cleanup(func() {
		runCmdFn = origRunCmd
		runCmdArgsFn = origRunCmdArgs
		lookPathFn = origLookPath
	})
}
