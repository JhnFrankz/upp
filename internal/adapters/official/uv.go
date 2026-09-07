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

	selfUpdateAvailable := false
	stdout, stderr, err := runCmdArgs("uv", "self", "update", "--dry-run")
	if err != nil {
		if !isExternalManagerError(err, stdout+" "+stderr) {
			return adapters.UpdateInfo{}, commandFailureErr("uv", stderr, err)
		}
		selfUpdateAvailable = false
	} else {
		selfUpdateAvailable = parseUvSelfUpdateOutput(stdout + " " + stderr)
	}

	stdout, stderr, err = runCmdArgs("uv", "tool", "list", "--outdated")
	if err != nil {
		return adapters.UpdateInfo{}, commandFailureErr("uv", stderr, err)
	}
	toolsUpdateAvailable := parseUvToolListOutdatedOutput(stdout)

	updateAvailable := selfUpdateAvailable || toolsUpdateAvailable

	return adapters.UpdateInfo{
		CurrentVersion:  current,
		LatestVersion:   current,
		UpdateAvailable: updateAvailable,
	}, nil
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
	return adapters.Result{}, nil
}
