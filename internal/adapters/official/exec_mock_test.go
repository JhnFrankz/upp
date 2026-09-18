package official

import (
	"fmt"
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
	shell           map[string]fakeResult
	cmdArgs         map[string]fakeResult
	lookPath        map[string]bool
	opencodeTag     string
	opencodeTagErr  error
	goBinaryPath    string
	goDevVersion    string
	goDevVersionErr error
}

// setExecFakes swaps the package exec seam variables (runCmdFn,
// runCmdArgsFn, lookPathFn) for the duration of the test and restores the
// real implementations via t.Cleanup. No real subprocess ever runs.
func setExecFakes(t *testing.T, f execFakes) {
	t.Helper()

	origRunCmd := runCmdFn
	origRunCmdArgs := runCmdArgsFn
	origLookPath := lookPathFn
	origOpencodeTag := opencodeLatestTagFn
	origGoBinaryPath := goBinaryPathFn
	origGoDevVersion := goDevVersionFn

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
		if r, ok := f.cmdArgs[name]; ok {
			return r.stdout, r.stderr, r.err
		}
		if name == "apt-cache" && len(args) == 2 && args[0] == "policy" {
			pkg := args[1]
			instKey := aptPolicyCmd(pkg, "Installed")
			candKey := aptPolicyCmd(pkg, "Candidate")
			if pkg == "apt" {
				instKey = aptInstalledCmd
				candKey = aptCandidateCmd
			}
			instRes := f.shell[instKey]
			candRes := f.shell[candKey]
			if instRes.err != nil {
				return "", "", instRes.err
			}
			if candRes.err != nil {
				return "", "", candRes.err
			}
			if instRes.stdout != "" || candRes.stdout != "" {
				return fmt.Sprintf("%s:\n  Installed: %s\n  Candidate: %s\n", pkg, instRes.stdout, candRes.stdout), "", nil
			}
		}
		if name == "pacman" && len(args) == 2 {
			pkg := args[1]
			switch args[0] {
			case "-Q":
				qKey := pacmanInstalledQueryCmd(pkg)
				if pkg == "pacman" {
					qKey = pacmanInstalledCmd
				}
				if r, ok := f.shell[qKey]; ok {
					return r.stdout, r.stderr, r.err
				}
			case "-Si":
				siKey := pacmanCandidateQueryCmd(pkg)
				if pkg == "pacman" {
					siKey = pacmanCandidateCmd
				}
				if r, ok := f.shell[siKey]; ok {
					if r.err != nil {
						return "", r.stderr, r.err
					}
					return fmt.Sprintf("Repository : extra\nName : %s\nVersion : %s\n", pkg, r.stdout), "", nil
				}
			}
		}
		return "", "", nil
	}
	lookPathFn = func(name string) bool {
		return f.lookPath[name]
	}
	opencodeLatestTagFn = func() (string, error) {
		if f.opencodeTag != "" || f.opencodeTagErr != nil {
			return f.opencodeTag, f.opencodeTagErr
		}
		return "", nil
	}
	goBinaryPathFn = func() string {
		if f.goBinaryPath != "" {
			return f.goBinaryPath
		}
		return "/usr/local/go/bin/go"
	}
	goDevVersionFn = func() (string, error) {
		if f.goDevVersion != "" || f.goDevVersionErr != nil {
			return f.goDevVersion, f.goDevVersionErr
		}
		return "", nil
	}

	t.Cleanup(func() {
		runCmdFn = origRunCmd
		runCmdArgsFn = origRunCmdArgs
		lookPathFn = origLookPath
		opencodeLatestTagFn = origOpencodeTag
		goBinaryPathFn = origGoBinaryPath
		goDevVersionFn = origGoDevVersion
	})
}
