package official

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
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
	goRelease       *GoRelease
	goReleaseErr    error
	goDownloadErr   error
	goExtractErr    error
	goTargetExists  *bool
	recordedCmds    *[][]string
}

// setExecFakes swaps the package exec seam variables (runCmdFn,
// runCmdArgsFn, lookPathFn) for the duration of the test and restores the
// real implementations via t.Cleanup. No real subprocess ever runs.
func setExecFakes(t *testing.T, f execFakes) {
	t.Helper()

	origRunCmd := runCmdFn
	origRunCmdArgs := runCmdArgsFn
	origRunCmdArgsUpdate := runCmdArgsUpdateFn
	origLookPath := lookPathFn
	origOpencodeTag := opencodeLatestTagFn
	origGoBinaryPath := goBinaryPathFn
	origGoDevVersion := goDevVersionFn
	origGoRelease := goReleaseFn
	origGoDownloadAndVerify := goDownloadAndVerifyFn
	origGoExtractTarball := goExtractTarballFn
	origGoTargetExists := goTargetExistsFn

	runCmdFn = func(ctx context.Context, command string) (stdout, stderr string, err error) {
		r := f.shell[command]
		return r.stdout, r.stderr, r.err
	}
	runCmdArgsUpdateFn = func(ctx context.Context, name string, args ...string) (stdout, stderr string, err error) {
		key := name
		if len(args) > 0 {
			key = name + " " + strings.Join(args, " ")
		}
		if f.recordedCmds != nil {
			*f.recordedCmds = append(*f.recordedCmds, append([]string{name}, args...))
		}
		if r, ok := f.cmdArgs[key]; ok {
			return r.stdout, r.stderr, r.err
		}
		if r, ok := f.shell[key]; ok {
			return r.stdout, r.stderr, r.err
		}
		if name == "sudo" && len(args) == 3 && args[0] == "mv" && args[2] == "/usr/local/go" && args[1] != "/usr/local/go.bak" {
			if r, ok := f.cmdArgs["sudo mv staged /usr/local/go"]; ok {
				return r.stdout, r.stderr, r.err
			}
			if r, ok := f.cmdArgs["sudo mv ... /usr/local/go"]; ok {
				return r.stdout, r.stderr, r.err
			}
			if r, ok := f.shell["sudo mv staged /usr/local/go"]; ok {
				return r.stdout, r.stderr, r.err
			}
		}
		if r, ok := f.cmdArgs[name]; ok {
			return r.stdout, r.stderr, r.err
		}
		return "", "", nil
	}
	runCmdArgsFn = func(ctx context.Context, name string, args ...string) (stdout, stderr string, err error) {
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
	opencodeLatestTagFn = func(ctx context.Context) (string, error) {
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
	goDevVersionFn = func(ctx context.Context) (string, error) {
		if f.goDevVersion != "" || f.goDevVersionErr != nil {
			return f.goDevVersion, f.goDevVersionErr
		}
		return "", nil
	}
	if f.goRelease != nil || f.goReleaseErr != nil {
		goReleaseFn = func(ctx context.Context, goos, goarch string) (GoRelease, error) {
			if f.goReleaseErr != nil {
				return GoRelease{}, f.goReleaseErr
			}
			return *f.goRelease, nil
		}
	} else {
		goReleaseFn = func(ctx context.Context, goos, goarch string) (GoRelease, error) {
			return GoRelease{
				Version:  "go1.22.1",
				Filename: fmt.Sprintf("go1.22.1.%s-%s.tar.gz", goos, goarch),
				SHA256:   "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
				URL:      fmt.Sprintf("https://go.dev/dl/go1.22.1.%s-%s.tar.gz", goos, goarch),
			}, nil
		}
	}
	goDownloadAndVerifyFn = func(ctx context.Context, rel GoRelease, destPath string) error {
		if f.goDownloadErr != nil {
			return f.goDownloadErr
		}
		return nil
	}
	goExtractTarballFn = func(archivePath, destDir string) error {
		if f.goExtractErr != nil {
			return f.goExtractErr
		}
		_ = os.MkdirAll(filepath.Join(destDir, "go", "bin"), 0755)
		return nil
	}
	goTargetExistsFn = func() bool {
		if f.goTargetExists != nil {
			return *f.goTargetExists
		}
		return true
	}

	t.Cleanup(func() {
		runCmdFn = origRunCmd
		runCmdArgsFn = origRunCmdArgs
		runCmdArgsUpdateFn = origRunCmdArgsUpdate
		lookPathFn = origLookPath
		opencodeLatestTagFn = origOpencodeTag
		goBinaryPathFn = origGoBinaryPath
		goDevVersionFn = origGoDevVersion
		goReleaseFn = origGoRelease
		goDownloadAndVerifyFn = origGoDownloadAndVerify
		goExtractTarballFn = origGoExtractTarball
		goTargetExistsFn = origGoTargetExists
	})
}
