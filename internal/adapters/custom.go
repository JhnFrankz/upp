// Package adapters provides the core interface and types for tool adapters.
package adapters

import (
	"fmt"
	"strings"
	"time"
)

// CustomAdapter implements Adapter for user-defined tools from config.
// Commands are executed via the platform shell.
type CustomAdapter struct {
	id       string
	command  string
	checkCmd string
	trusted  bool
	manager  Adapter // resolving owner adapter (nil = standalone custom tool)
}

// NewCustomAdapter creates a CustomAdapter from config values.
// Returns an error if the command is empty (required field).
//
// manager is the optional resolving owner adapter (WU2, spec Official Adapter
// Catalog / Resolved Owner Update Delegation). A custom tool MAY declare an
// owning manager (config `manager` field, threaded through buildAdapterList):
// when present, Update() delegates to the manager rather than running the
// tool's own command, and the manager's UpdatePolicy governs the CLI gate
// (the custom tool's own declared policy is INERT on the delegated path).
// Passing no manager (or nil) keeps the custom tool standalone. The variadic
// form preserves backward compatibility with existing 4-arg callers.
func NewCustomAdapter(id, command, checkCmd string, trusted bool, manager ...Adapter) (*CustomAdapter, error) {
	if command == "" {
		return nil, fmt.Errorf("custom tool %q: command is required", id)
	}
	ca := &CustomAdapter{
		id:       id,
		command:  command,
		checkCmd: checkCmd,
		trusted:  trusted,
	}
	if len(manager) > 0 {
		ca.manager = manager[0]
	}
	return ca, nil
}

func (c *CustomAdapter) Name() string { return c.id }

// Detect returns true if the base command exists on PATH.
func (c *CustomAdapter) Detect() bool {
	base := extractBaseCommand(c.command)
	if base == "" {
		return false
	}
	_, err := lookPathFn(base)
	return err == nil
}

// Check executes the check_cmd and parses version output.
func (c *CustomAdapter) Check() (UpdateInfo, error) {
	if c.checkCmd == "" {
		return UpdateInfo{}, nil
	}

	if !c.Detect() {
		return UpdateInfo{}, fmt.Errorf("tool %q is not installed (binary %q not found on PATH)", c.id, extractBaseCommand(c.command))
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
	// Delegated update path (WU2, spec Resolved Owner Update Delegation): a
	// custom tool with a resolving owner manager delegates to the manager's
	// Update() instead of running its own command. The manager's self-update
	// (and its command/privileges) governs; the custom tool's own command is
	// never invoked on the delegated path.
	if c.manager != nil {
		return c.manager.Update(dryRun)
	}

	privileges := detectPrivileges(c.command)

	if !c.Detect() {
		return Result{
			Success:    false,
			Error:      fmt.Errorf("tool %q is not installed (binary %q not found on PATH)", c.id, extractBaseCommand(c.command)),
			Privileges: privileges,
		}, nil
	}

	if dryRun {
		return Result{
			Success:    true,
			Before:     c.command,
			After:      c.command,
			Privileges: privileges,
		}, nil
	}

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

	info := ToolInfo{
		ID:           c.id,
		Name:         c.id,
		Platforms:    []string{"linux", "darwin", "windows"},
		Trust:        trust,
		UpdatePolicy: PolicyAlwaysUpdate,
		Kind:         KindTool,
		Command:      c.command,
		Privileges:   detectPrivileges(c.command),
	}
	if c.manager != nil {
		// An owned custom tool is not a manager itself and must not gate
		// independently: the manager's UpdatePolicy governs on the delegated
		// path (spec Update Gating). The declared policy here is INERT for the
		// CLI gate, which resolves the effective policy from the owner.
		info.Kind = KindTool
	}
	return info
}

// ManagerAdapter returns the injected resolving owner adapter a custom tool
// delegates to, or nil when the custom tool is standalone. The CLI gate uses
// it to derive the effective UpdatePolicy on the delegated path (the custom
// tool's own declared policy is INERT there, spec Update Gating).
func (c *CustomAdapter) ManagerAdapter() Adapter {
	return c.manager
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

// shellExecWithTimeout delegates to the shellExecWithTimeoutFn seam variable.
func shellExecWithTimeout(command string, timeout time.Duration) (string, error) {
	return shellExecWithTimeoutFn(command, timeout)
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
