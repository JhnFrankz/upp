// Package official provides built-in adapter implementations for all supported tools.
package official

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/JhnFrankz/upp/internal/adapters"
)

// Test seam (D1): package-level function variables swapped by tests via
// setExecFakes so adapters can be exercised hermetically without executing
// real subprocesses. Production behavior is preserved — the vars initialize
// to the real implementations and are only replaced inside tests. The public
// leaf functions (runCmd, runCmdArgs, lookPath) delegate to the vars, so both
// adapters and wrappers stay hermetic when a test swaps the seam.
var (
	runCmdFn = func(command string) (stdout, stderr string, err error) {
		ctx, cancel := context.WithTimeout(context.Background(), adapters.UpdateTimeout)
		defer cancel()

		var cmd *exec.Cmd
		if runtime.GOOS == "windows" {
			cmd = exec.CommandContext(ctx, "cmd", "/C", command)
		} else {
			cmd = exec.CommandContext(ctx, "sh", "-c", command)
			// Own process group so the timeout can kill shell grandchildren
			// (curl|tar, sudo apt, brew...) too — the direct child alone is
			// not enough for the compound commands every adapter runs.
			setpgid(cmd)
		}
		return adapters.RunCommandWithTimeout(ctx, cmd)
	}
	runCmdArgsFn = func(name string, args ...string) (stdout, stderr string, err error) {
		ctx, cancel := context.WithTimeout(context.Background(), adapters.CheckTimeout)
		defer cancel()

		var cmd *exec.Cmd
		if runtime.GOOS == "windows" {
			cmd = exec.CommandContext(ctx, name, args...)
		} else {
			cmd = exec.CommandContext(ctx, name, args...)
			// Own process group so the timeout can kill descendants too
			// (npm/pnpm workers holding the pipes): the direct child alone
			// is not enough, exactly like runCmdFn's shell path.
			setpgid(cmd)
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
func runCmd(command string) (stdout, stderr string, err error) {
	return runCmdFn(command)
}

// runCmdArgs executes a command with explicit arguments (no shell).
// Delegates to the runCmdArgsFn seam variable.
func runCmdArgs(name string, args ...string) (stdout, stderr string, err error) {
	return runCmdArgsFn(name, args...)
}

// lookPath checks if a command exists on PATH.
// Returns true if found, false otherwise. Delegates to the lookPathFn seam.
func lookPath(name string) bool {
	return lookPathFn(name)
}

// extractVersion attempts to extract a version string from command output.
// It looks for semver-like patterns (v1.2.3, 1.2.3, 1.2.3-rc1, etc.).
func extractVersion(output string) string {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Try to find a version-like string in the line.
		version := findVersionInLine(line)
		if version != "" {
			return version
		}
	}
	return ""
}

// findVersionInLine scans a line for a version-like token.
func findVersionInLine(line string) string {
	fields := strings.Fields(line)
	for _, field := range fields {
		cleaned := strings.Trim(field, "(),:;")
		if isVersionLike(cleaned) {
			return cleaned
		}
	}
	return ""
}

// isVersionLike returns true if the string looks like a version number.
func isVersionLike(s string) bool {
	if s == "" {
		return false
	}
	// Strip leading letters (e.g., "go" in "go1.22.0").
	start := 0
	for start < len(s) && ((s[start] >= 'a' && s[start] <= 'z') || (s[start] >= 'A' && s[start] <= 'Z')) {
		start++
	}
	if start >= len(s) {
		return false
	}
	// Must start with a digit after stripping letters.
	if s[start] < '0' || s[start] > '9' {
		return false
	}
	// Must contain at least one dot.
	dotFound := false
	for i := start; i < len(s); i++ {
		c := s[i]
		if c == '.' {
			dotFound = true
		} else if (c < '0' || c > '9') && c != '.' && c != '-' && c != '+' {
			// Allow digits, dots, hyphens, plus — stop at any letter (pre-release tags)
			break
		}
	}
	return dotFound
}

// commandOutput runs a command and returns its trimmed stdout.
// Used for simple version commands where stderr is irrelevant.
func commandOutput(name string, args ...string) string {
	stdout, _, err := runCmdArgsFn(name, args...)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(stdout)
}

// shellOutput runs a shell command and returns its trimmed stdout.
func shellOutput(command string) string {
	stdout, _, err := runCmdFn(command)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(stdout)
}

// commandOutputErr runs a command and returns its trimmed stdout, or a
// structured failure when the subprocess fails (design D3). Delegates to the
// same runCmdArgsFn seam variable as commandOutput, so seam fakes keep
// working. stdout is preserved on failure: the npm/pnpm exit-1 convention
// (D4) needs it to decide availability.
func commandOutputErr(name string, args ...string) (string, error) {
	stdout, stderr, err := runCmdArgsFn(name, args...)
	if err != nil {
		return strings.TrimSpace(stdout), commandFailureErr(name, stderr, err)
	}
	return strings.TrimSpace(stdout), nil
}

// shellOutputErr runs a shell command and returns its trimmed stdout, or a
// structured failure when the subprocess fails (design D3). Delegates to the
// same runCmdFn seam variable as shellOutput, so seam fakes keep working.
// stdout is preserved on failure: the npm/pnpm exit-1 convention (D4) needs
// it to decide availability. The failure label is the explicit tool name,
// not the command's first token (a wrapper or shell builtin).
func shellOutputErr(command, tool string) (string, error) {
	stdout, stderr, err := runCmdFn(command)
	if err != nil {
		return strings.TrimSpace(stdout), commandFailureErr(tool, stderr, err)
	}
	return strings.TrimSpace(stdout), nil
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

// formatUpdateCmd returns a human-readable description of the update command.
func formatUpdateCmd(cmd string) string {
	return fmt.Sprintf("exec: %s", cmd)
}

// wingetSelfID is the manifest Id that identifies Windows Package Manager
// itself in `winget upgrade` (no args) output. winget's self-update package
// is the App Installer, whose Id is "Microsoft.AppInstaller"; the "winget"
// display label and the "winget" Source column are not unique enough to anchor
// on (the display label is field 0 and the Source column shares the same
// string), so the manifest Id is the unambiguous key.
const wingetSelfID = "Microsoft.AppInstaller"

// parseWingetUpgradeOutput scans `winget upgrade` (no args) output for the
// winget self row and returns (current, latest, found). It is a PURE,
// fail-closed function: unparseable output or a missing self row yields
// found=false and no error, so an old winget (< 1.6, which lists no self row)
// reports "availability unavailable gracefully". The leading-v on a version
// (e.g. "v1.8.2311") is tolerated — the string is returned unchanged, since
// the winget versions genuinely carry the leading v.
func parseWingetUpgradeOutput(out string) (current, latest string, found bool) {
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		// Locate the Id field; the two fields after it are Current and
		// Latest. Anchor on the manifest Id so a wrong key at the display
		// Name or Source column cannot mis-align the version positions.
		for i := 0; i < len(fields); i++ {
			if !strings.EqualFold(fields[i], wingetSelfID) {
				continue
			}
			// Id, Current, Latest must all be present in the row.
			if i+2 < len(fields) {
				return fields[i+1], fields[i+2], true
			}
			return "", "", false
		}
	}
	return "", "", false
}
