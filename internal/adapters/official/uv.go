package official

import (
	"fmt"
	"strings"

	"github.com/JhnFrankz/upp/internal/adapters"
)

// UvAdapter manages Astral's uv package and tool manager on all platforms.
type UvAdapter struct{}

var _ adapters.Adapter = (*UvAdapter)(nil)

func (a *UvAdapter) Name() string { return "uv" }

func (a *UvAdapter) Detect() bool {
	return lookPath("uv")
}

func (a *UvAdapter) Info() adapters.ToolInfo {
	return adapters.ToolInfo{
		ID:           "uv",
		Name:         "uv",
		Platforms:    []string{"linux", "macos", "windows"},
		Trust:        adapters.TrustOfficial,
		UpdatePolicy: adapters.PolicyGated,
		Kind:         adapters.KindTool,
		// Command is the exact string Update() runs, declared so the plan's
		// RiskCommand and the confirmation gate see what actually executes.
		// Update() also runs "uv tool upgrade --all", which upgrades the tools
		// uv manages rather than uv itself; Command names the self-update.
		Command: "uv self update",
	}
}

func (a *UvAdapter) Check() (adapters.UpdateInfo, error) {
	if !a.Detect() {
		return adapters.UpdateInfo{}, fmt.Errorf("uv is not installed")
	}

	rawVersion, _, _ := runCmdArgs("uv", "--version")
	current := extractUvVersion(rawVersion)
	if current == "" {
		current = "unknown"
	}

	latest := current
	selfUpdateAvailable := false
	stdout, stderr, err := runCmdArgs("uv", "self", "update", "--dry-run")
	if err != nil {
		if !isExternalManagerError(err, stdout+" "+stderr) {
			return adapters.UpdateInfo{}, commandFailureErr("uv", stderr, err)
		}
		selfUpdateAvailable = false
	} else {
		combined := stdout + " " + stderr
		selfUpdateAvailable = parseUvSelfUpdateOutput(combined)
		if selfUpdateAvailable {
			latest = extractUvLatestVersion(combined, current)
		}
	}

	stdout, stderr, err = runCmdArgs("uv", "tool", "list", "--outdated")
	if err != nil {
		return adapters.UpdateInfo{}, commandFailureErr("uv", stderr, err)
	}
	toolsUpdateAvailable := parseUvToolListOutdatedOutput(stdout)

	updateAvailable := selfUpdateAvailable || toolsUpdateAvailable

	return adapters.UpdateInfo{
		CurrentVersion:  current,
		LatestVersion:   latest,
		UpdateAvailable: updateAvailable,
	}, nil
}

func extractUvLatestVersion(output, current string) string {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		for i, f := range fields {
			if strings.EqualFold(f, "to") && i+1 < len(fields) {
				target := strings.Trim(fields[i+1], "(),:;\"'")
				if isVersionLike(target) {
					return target
				}
			}
		}
	}
	for _, line := range lines {
		fields := strings.Fields(line)
		for i := len(fields) - 1; i >= 0; i-- {
			cleaned := strings.Trim(fields[i], "(),:;\"'")
			if isVersionLike(cleaned) && cleaned != current {
				return cleaned
			}
		}
	}
	return current
}

func extractUvVersion(raw string) string {
	return extractVersion(raw)
}

func isExternalManagerError(err error, output string) bool {
	if err == nil {
		return false
	}
	isCode2 := isExitCode(err, 2) ||
		strings.Contains(err.Error(), "exit status 2") ||
		strings.Contains(err.Error(), "(exit 2)")
	if !isCode2 {
		return false
	}
	combined := output + " " + err.Error()
	return strings.Contains(strings.ToLower(combined), "external package manager")
}

func parseUvSelfUpdateOutput(output string) bool {
	lower := strings.ToLower(output)
	if strings.TrimSpace(lower) == "" {
		return false
	}
	if strings.Contains(lower, "up to date") {
		return false
	}
	return strings.Contains(lower, "would update") ||
		strings.Contains(lower, "new version") ||
		strings.Contains(lower, "updating")
}

func parseUvToolListOutdatedOutput(output string) bool {
	trimmed := strings.TrimSpace(output)
	if trimmed == "" {
		return false
	}
	lower := strings.ToLower(trimmed)
	if strings.Contains(lower, "no tools installed") || strings.Contains(lower, "no outdated tools") {
		return false
	}
	return true
}

func (a *UvAdapter) Update(dryRun bool) (adapters.Result, error) {
	if !a.Detect() {
		return adapters.Result{Success: false}, fmt.Errorf("uv is not installed")
	}

	before := extractUvVersion(commandOutput("uv", "--version"))
	if before == "" {
		before = "unknown"
	}

	if dryRun {
		return adapters.Result{
			Success: true,
			Before:  before,
			After:   before,
		}, nil
	}

	stdout, stderr, err := runCmd("uv self update")
	if err != nil {
		if !isExternalManagerError(err, stdout+" "+stderr) {
			return adapters.Result{
				Success: false,
				Before:  before,
				After:   before,
				Error:   fmt.Errorf("uv self update failed: %w", err),
			}, nil
		}
	}

	_, _, err = runCmd("uv tool upgrade --all")
	if err != nil {
		return adapters.Result{
			Success: false,
			Before:  before,
			After:   before,
			Error:   fmt.Errorf("uv tool upgrade failed: %w", err),
		}, nil
	}

	after := extractUvVersion(commandOutput("uv", "--version"))
	if after == "" {
		after = before
	}

	return adapters.Result{
		Success: true,
		Before:  before,
		After:   after,
	}, nil
}
