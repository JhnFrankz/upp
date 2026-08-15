// Package adapters provides the core interface and types for tool adapters.
package adapters

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"time"
)

// CustomAdapter implements Adapter for user-defined tools from config.
// Commands are executed via the platform shell.
type CustomAdapter struct {
	id       string
	command  string
	checkCmd string
	trusted  bool
}

// NewCustomAdapter creates a CustomAdapter from config values.
// Returns an error if the command is empty (required field).
func NewCustomAdapter(id, command, checkCmd string, trusted bool) (*CustomAdapter, error) {
	if command == "" {
		return nil, fmt.Errorf("custom tool %q: command is required", id)
	}
	return &CustomAdapter{
		id:       id,
		command:  command,
		checkCmd: checkCmd,
		trusted:  trusted,
	}, nil
}

func (c *CustomAdapter) Name() string { return c.id }

// Detect returns true if the base command exists on PATH.
func (c *CustomAdapter) Detect() bool {
	base := extractBaseCommand(c.command)
	_, err := exec.LookPath(base)
	return err == nil
}

// Check executes the check_cmd and parses version output.
func (c *CustomAdapter) Check() (UpdateInfo, error) {
	if c.checkCmd == "" {
		return UpdateInfo{}, nil
	}

	stdout, err := shellExecWithTimeout(c.checkCmd, CheckTimeout)
	if err != nil {
		return UpdateInfo{}, fmt.Errorf("check command failed for %s: %w", c.id, err)
	}

	latest := extractVersionFromOutput(stdout)

	return UpdateInfo{
		CurrentVersion:  latest,
		LatestVersion:   latest,
		UpdateAvailable: false,
	}, nil
}

// Update executes the tool's update command.
func (c *CustomAdapter) Update(dryRun bool) (Result, error) {
	if dryRun {
		return Result{
			Success: true,
			Before:  c.command,
			After:   c.command,
		}, nil
	}

	privileges := detectPrivileges(c.command)

	_, err := shellExec(c.command)
	if err != nil {
		return Result{
			Success:    false,
			Error:      fmt.Errorf("update command failed for %s: %w", c.id, err),
			Privileges: privileges,
		}, nil
	}

	return Result{
		Success:    true,
		Privileges: privileges,
	}, nil
}

func (c *CustomAdapter) Info() ToolInfo {
	trust := TrustCustomUntrusted
	if c.trusted {
		trust = TrustCustomTrusted
	}

	return ToolInfo{
		ID:           c.id,
		Name:         c.id,
		Platforms:    []string{"linux", "darwin", "windows"},
		Trust:        trust,
		UpdatePolicy: PolicyAlwaysUpdate,
		Command:      c.command,
		Privileges:   detectPrivileges(c.command),
	}
}

// IsTrusted returns whether the custom adapter is trusted via config.
func (c *CustomAdapter) IsTrusted() bool {
	return c.trusted
}

// extractBaseCommand gets the first token from a command string.
func extractBaseCommand(cmd string) string {
	fields := strings.Fields(cmd)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

// shellExec runs a command via the platform shell, bounded by UpdateTimeout.
func shellExec(command string) (string, error) {
	return shellExecWithTimeout(command, UpdateTimeout)
}

// shellExecWithTimeout runs a command via the platform shell and kills it —
// including its whole process group on Unix — once timeout expires, so
// pipeline/grandchild work (curl|tar, sudo apt, brew...) cannot outlive the
// deadline. The returned error is errors.Is-detectable as
// context.DeadlineExceeded. On Windows only the direct child is terminated.
// Delegates to the shared RunCommandWithTimeout implementation.
func shellExecWithTimeout(command string, timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd", "/C", command)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", command)
		// Own process group so the timeout can kill shell grandchildren too.
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	}

	stdout, _, err := RunCommandWithTimeout(ctx, cmd)
	return strings.TrimSpace(stdout), err
}

// extractVersionFromOutput extracts a version-like string from command output.
func extractVersionFromOutput(output string) string {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		version := findVersionInLine(line)
		if version != "" {
			return version
		}
	}
	return output
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
	start := 0
	for start < len(s) && ((s[start] >= 'a' && s[start] <= 'z') || (s[start] >= 'A' && s[start] <= 'Z')) {
		start++
	}
	if start >= len(s) {
		return false
	}
	if s[start] < '0' || s[start] > '9' {
		return false
	}
	dotFound := false
	for i := start; i < len(s); i++ {
		c := s[i]
		if c == '.' {
			dotFound = true
		} else if (c < '0' || c > '9') && c != '.' && c != '-' && c != '+' {
			break
		}
	}
	return dotFound
}

// detectPrivileges checks if a command requires elevated privileges.
func detectPrivileges(cmd string) []string {
	lower := strings.ToLower(cmd)
	var privs []string
	if strings.Contains(lower, "sudo") {
		privs = append(privs, "sudo")
	}
	if strings.Contains(lower, "runas") || strings.Contains(lower, "admin") {
		privs = append(privs, "admin")
	}
	return privs
}
