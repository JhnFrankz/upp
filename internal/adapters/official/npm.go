package official

import (
	"fmt"
	"strings"

	"github.com/JhnFrankz/upp/internal/adapters"
)

// NpmAdapter manages npm global packages on all platforms.
type NpmAdapter struct{}

func (a *NpmAdapter) Name() string { return "npm" }

func (a *NpmAdapter) Detect() bool {
	return lookPath("npm")
}

func (a *NpmAdapter) Check() (adapters.UpdateInfo, error) {
	if !a.Detect() {
		return adapters.UpdateInfo{}, fmt.Errorf("npm is not installed")
	}

	current := commandOutput("npm", "--version")
	current = strings.TrimSpace(current)
	if current == "" {
		current = "unknown"
	}

	// Check for npm updates. Run npm directly (no shell wrapper): the
	// runCmdArgsFn seam bounds the check with CheckTimeout, which is
	// portable (no GNU `timeout` dependency — macOS) and keeps stderr for
	// the failure excerpt. npm exits 1 when outdated packages exist — a
	// valid detection only when stdout carries the outdated table; exit 1
	// with empty stdout is an operational failure (EACCES, unreachable
	// registry), and any other non-zero exit is a structured failure.
	stdout, err := commandOutputErr("npm", "outdated", "-g", "--depth=0")
	if err != nil {
		if !isExitCode(err, 1) || strings.TrimSpace(stdout) == "" {
			return adapters.UpdateInfo{}, err
		}
	}
	updateAvailable := strings.TrimSpace(stdout) != ""

	return adapters.UpdateInfo{
		CurrentVersion:  current,
		LatestVersion:   current, // npm doesn't easily expose its own latest version
		UpdateAvailable: updateAvailable,
	}, nil
}

func (a *NpmAdapter) Update(dryRun bool) (adapters.Result, error) {
	if !a.Detect() {
		return adapters.Result{Success: false}, fmt.Errorf("npm is not installed")
	}

	before := commandOutput("npm", "--version")
	before = strings.TrimSpace(before)

	if dryRun {
		return adapters.Result{
			Success: true,
			Before:  before,
			After:   before,
		}, nil
	}

	_, stderr, err := runCmd("npm update -g")
	if err != nil {
		return adapters.Result{
			Success: false,
			Before:  before,
			After:   before,
			Error:   fmt.Errorf("npm update failed: %w", err),
		}, nil
	}

	if stderr != "" && strings.Contains(stderr, "ERR!") {
		return adapters.Result{
			Success: false,
			Before:  before,
			After:   before,
			Error:   fmt.Errorf("npm update error: %s", truncate(stderr, 200)),
		}, nil
	}

	after := commandOutput("npm", "--version")
	after = strings.TrimSpace(after)

	return adapters.Result{
		Success: true,
		Before:  before,
		After:   after,
	}, nil
}

func (a *NpmAdapter) Info() adapters.ToolInfo {
	return adapters.ToolInfo{
		ID:           "npm",
		Name:         "npm",
		Platforms:    []string{"linux", "macos", "windows"},
		Trust:        adapters.TrustOfficial,
		UpdatePolicy: adapters.PolicyGated,
	}
}
