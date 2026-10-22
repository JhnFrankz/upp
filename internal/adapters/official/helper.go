// Package official provides built-in adapter implementations for all supported tools.
package official

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"sync"

	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/platform"
	"github.com/JhnFrankz/upp/internal/version"
)

// runtimeGOOSFn is the seam for querying runtime.GOOS. Swapped in tests via setExecFakes.
var runtimeGOOSFn = func() string { return runtime.GOOS }

// resolveOwnerFn is the seam for resolving owner adapters.
var resolveOwnerFn = ResolveOwner

// Test seam (D1): package-level function variables swapped by tests via
// setExecFakes so adapters can be exercised hermetically without executing
// real subprocesses. Production behavior is preserved — the vars initialize
// to the real implementations and are only replaced inside tests. The public
// leaf functions (runCmd, runCmdArgs, lookPath) delegate to the vars, so both
// adapters and wrappers stay hermetic when a test swaps the seam.

var (
	runCmdFn = func(ctx context.Context, command string) (stdout, stderr string, err error) {
		if ctx == nil {
			ctx = context.Background()
		}
		ctx, cancel := context.WithTimeout(ctx, adapters.UpdateTimeout)
		defer cancel()

		var cmd *exec.Cmd
		if runtimeGOOSFn() == "windows" {
			cmd = exec.CommandContext(ctx, "cmd", "/C", command)
		} else {
			cmd = exec.CommandContext(ctx, "sh", "-c", command)
			// Own process group so the timeout can kill shell grandchildren
			// (curl|tar, sudo apt, brew...) too — the direct child alone is
			// not enough for the compound commands every adapter runs.
			adapters.Setpgid(cmd)
		}
		return adapters.RunCommandWithTimeout(ctx, cmd)
	}
	runCmdArgsFn = func(ctx context.Context, name string, args ...string) (stdout, stderr string, err error) {
		if ctx == nil {
			ctx = context.Background()
		}
		ctx, cancel := context.WithTimeout(ctx, adapters.CheckTimeout)
		defer cancel()

		var cmd *exec.Cmd
		if runtimeGOOSFn() == "windows" {
			cmd = exec.CommandContext(ctx, name, args...)
		} else {
			cmd = exec.CommandContext(ctx, name, args...)
			// Own process group so the timeout can kill descendants too
			// (npm/pnpm workers holding the pipes): the direct child alone
			// is not enough, exactly like runCmdFn's shell path.
			adapters.Setpgid(cmd)
		}
		return adapters.RunCommandWithTimeout(ctx, cmd)
	}
	runCmdArgsUpdateFn = func(ctx context.Context, name string, args ...string) (stdout, stderr string, err error) {
		if ctx == nil {
			ctx = context.Background()
		}
		ctx, cancel := context.WithTimeout(ctx, adapters.UpdateTimeout)
		defer cancel()

		var cmd *exec.Cmd
		if runtimeGOOSFn() == "windows" {
			cmd = exec.CommandContext(ctx, name, args...)
		} else {
			cmd = exec.CommandContext(ctx, name, args...)
			// Own process group so the timeout can kill descendants too
			// (workers holding the pipes): the direct child alone is
			// not enough, exactly like runCmdFn's shell path.
			adapters.Setpgid(cmd)
		}
		return adapters.RunCommandWithTimeout(ctx, cmd)
	}
	lookPathFn = func(name string) bool {
		_, err := exec.LookPath(name)
		return err == nil
	}
)

// runCmd executes a shell command and returns stdout, stderr, and any error.
// The command runs via the platform's default shell.
// Delegates to the runCmdFn seam variable.
func runCmd(ctx context.Context, command string) (stdout, stderr string, err error) {
	return runCmdFn(ctx, command)
}

// runCmdArgs executes a command with explicit arguments (no shell).
// Delegates to the runCmdArgsFn seam variable.
func runCmdArgs(ctx context.Context, name string, args ...string) (stdout, stderr string, err error) {
	return runCmdArgsFn(ctx, name, args...)
}

// runCmdArgsUpdate executes a command with explicit arguments (no shell) using UpdateTimeout.
// Delegates to the runCmdArgsUpdateFn seam variable.
func runCmdArgsUpdate(ctx context.Context, name string, args ...string) (stdout, stderr string, err error) {
	return runCmdArgsUpdateFn(ctx, name, args...)
}

// lookPath checks if a command exists on PATH.
// Returns true if found, false otherwise. Delegates to the lookPathFn seam.
func lookPath(name string) bool {
	return lookPathFn(name)
}

var (
	linuxManagerMu     sync.RWMutex
	cachedLinuxManager string
)

// defaultLinuxManager resolves the default Linux package manager for official tools.
// It returns "pacman" if apt is absent and pacman is present on PATH, otherwise "apt".
func defaultLinuxManager() string {
	if runtimeGOOSFn() != "linux" {
		return "apt"
	}
	linuxManagerMu.RLock()
	if cachedLinuxManager != "" {
		mgr := cachedLinuxManager
		linuxManagerMu.RUnlock()
		return mgr
	}
	linuxManagerMu.RUnlock()

	linuxManagerMu.Lock()
	defer linuxManagerMu.Unlock()
	if cachedLinuxManager != "" {
		return cachedLinuxManager
	}
	mgr := "apt"
	if !lookPathFn("apt") && lookPathFn("pacman") {
		mgr = "pacman"
	}
	cachedLinuxManager = mgr
	return mgr
}

// extractVersion attempts to extract a version string from command output.
// It looks for semver-like patterns (v1.2.3, 1.2.3, 1.2.3-rc1, etc.).
func extractVersion(output string) string {
	return version.ExtractVersionFromString(output)
}

// extractVersionFromString extracts a version from a version string or command output.
func extractVersionFromString(s string) string {
	return version.ExtractVersionFromString(s)
}

// isVersionLike returns true if the string looks like a version number.
func isVersionLike(s string) bool {
	return version.IsVersionLike(s)
}

// commandOutput runs a command and returns its trimmed stdout.
// Used for simple version commands where stderr is irrelevant.
func commandOutput(ctx context.Context, name string, args ...string) string {
	stdout, _, err := runCmdArgsFn(ctx, name, args...)
	if err != nil || stdout == "" {
		return ""
	}
	return strings.TrimSpace(stdout)
}

// shellOutput runs a shell command and returns its trimmed stdout.
func shellOutput(ctx context.Context, command string) string {
	stdout, _, err := runCmdFn(ctx, command)
	if err != nil || stdout == "" {
		return ""
	}
	return strings.TrimSpace(stdout)
}

// commandOutputErrFor runs a command and returns its trimmed stdout, labeling
// errors with the given tool name rather than the binary name.
func commandOutputErrFor(ctx context.Context, tool, name string, args ...string) (string, error) {
	stdout, stderr, err := runCmdArgsFn(ctx, name, args...)
	var retErr error
	if err != nil {
		retErr = commandFailureErr(tool, stderr, err)
	}
	if stdout == "" {
		return "", retErr
	}
	return strings.TrimSpace(stdout), retErr
}

// commandOutputErr runs a command and returns its trimmed stdout, or a
// structured failure when the subprocess fails (design D3). Delegates to the
// same runCmdArgsFn seam variable as commandOutput, so seam fakes keep
// working. stdout is preserved on failure: the npm/pnpm exit-1 convention
// (D4) needs it to decide availability.
func commandOutputErr(ctx context.Context, name string, args ...string) (string, error) {
	return commandOutputErrFor(ctx, name, name, args...)
}

// shellOutputErr runs a shell command and returns its trimmed stdout, or a
// structured failure when the subprocess fails (design D3). Delegates to the
// same runCmdFn seam variable as shellOutput, so seam fakes keep working.
// stdout is preserved on failure: the npm/pnpm exit-1 convention (D4) needs
// it to decide availability. The failure label is the explicit tool name,
// not the command's first token (a wrapper or shell builtin).
func shellOutputErr(ctx context.Context, command, tool string) (string, error) {
	stdout, stderr, err := runCmdFn(ctx, command)
	var retErr error
	if err != nil {
		retErr = commandFailureErr(tool, stderr, err)
	}
	if stdout == "" {
		return "", retErr
	}
	return strings.TrimSpace(stdout), retErr
}

// commandFailureErr builds the structured Check() failure message
// "<tool> check failed (exit N): <stderr excerpt>: %w". The exit code is
// extracted via errors.As(*exec.ExitError) and omitted when not extractable;
// the seam error stays %w-chained so timeout mapping
// (errors.Is(err, context.DeadlineExceeded), CLI timeoutErr) survives (D3).
func commandFailureErr(tool, stderr string, err error) error {
	var exitErr *exec.ExitError
	code := ""
	if errors.As(err, &exitErr) {
		code = fmt.Sprintf(" (exit %d)", exitErr.ExitCode())
	}
	if excerpt := truncate(strings.TrimSpace(stderr), 200); excerpt != "" {
		return fmt.Errorf("%s check failed%s: %s: %w", tool, code, excerpt, err)
	}
	return fmt.Errorf("%s check failed%s: %w", tool, code, err)
}

// isExitCode reports whether err is an *exec.ExitError with the given exit
// code — the npm/pnpm `outdated` convention where exit 1 means updates are
// available (design D4).
func isExitCode(err error, code int) bool {
	var exitErr *exec.ExitError
	return errors.As(err, &exitErr) && exitErr.ExitCode() == code
}

// resolveOwnerForAdapter resolves the manager adapter that owns the given tool
// on the given OS. It queries resolveOwnerFn(a.Name(), os), and falls back to looking up
// the manager ID declared in a.Info().Manager[os] via AdapterByName (e.g. for dynamic,
// unregistered, or mock adapters).
func resolveOwnerForAdapter(a adapters.Adapter, os string) adapters.Adapter {
	if a == nil {
		return nil
	}
	if owner := resolveOwnerFn(a.Name(), os); owner != nil {
		return owner
	}
	if mgrID := a.Info().Manager[os]; mgrID != "" {
		return AdapterByName(mgrID)
	}
	return nil
}

// DelegatedCheck checks for updates of an owned tool by delegating to its
// resolving package manager's PackageChecker interface.
func DelegatedCheck(ctx context.Context, a adapters.Adapter, current string) (adapters.UpdateInfo, error) {
	if a == nil {
		return adapters.UpdateInfo{}, fmt.Errorf("%w: adapter is nil", adapters.ErrNilAdapter)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	plat, _ := platform.NormalizeOS(runtimeGOOSFn())
	if owner := resolveOwnerForAdapter(a, plat); owner != nil {
		if !owner.Detect() {
			return adapters.UpdateInfo{
				CurrentVersion:  current,
				LatestVersion:   current,
				UpdateAvailable: false,
			}, nil
		}
		if checker, ok := owner.(adapters.PackageChecker); ok {
			pkg := a.Info().ManagerPackage[plat]
			if pkg == "" {
				return adapters.UpdateInfo{}, fmt.Errorf("%w: %s has no manager package on %s", adapters.ErrNoManagerPackage, a.Name(), runtimeGOOSFn())
			}
			return checker.CheckPackage(ctx, pkg)
		}
		return adapters.UpdateInfo{}, fmt.Errorf("%w: %s's manager %s does not support per-package checks", adapters.ErrUnsupportedPackageCheck, a.Name(), owner.Name())
	}

	return adapters.UpdateInfo{}, fmt.Errorf("%w: %s has no resolving owner on %s", adapters.ErrNoResolvingOwner, a.Name(), runtimeGOOSFn())
}

// DelegatedUpdate updates an owned tool by delegating to its resolving package
// manager's PackageUpdater interface.
func DelegatedUpdate(ctx context.Context, a adapters.Adapter, dryRun bool) (adapters.Result, error) {
	if a == nil {
		return adapters.Result{
			Success: false,
			Error:   fmt.Errorf("%w: adapter is nil", adapters.ErrNilAdapter),
		}, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	plat, _ := platform.NormalizeOS(runtimeGOOSFn())
	if owner := resolveOwnerForAdapter(a, plat); owner != nil {
		if dryRun {
			return adapters.Result{Success: true}, nil
		}
		if updater, ok := owner.(adapters.PackageUpdater); ok {
			pkg := a.Info().ManagerPackage[plat]
			if pkg == "" {
				return adapters.Result{Success: false}, fmt.Errorf("%w: %s has no manager package on %s", adapters.ErrNoManagerPackage, a.Name(), runtimeGOOSFn())
			}
			return updater.UpdatePackage(ctx, pkg)
		}
		return adapters.Result{Success: false}, fmt.Errorf("%w: %s's manager %s does not support per-package updates", adapters.ErrUnsupportedPackageUpdate, a.Name(), owner.Name())
	}

	// Fail-closed fallback if ownership map ever regresses.
	return adapters.Result{
		Success: false,
		Error:   fmt.Errorf("%w: %s has no resolving owner on %s", adapters.ErrNoResolvingOwner, a.Name(), runtimeGOOSFn()),
	}, nil
}
