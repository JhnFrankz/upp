package official

import (
	"fmt"
	"strings"

	"github.com/JhnFrankz/upp/internal/adapters"
)

// WingetAdapter manages Windows Package Manager packages.
type WingetAdapter struct{}

func (a *WingetAdapter) Name() string { return "winget" }

func (a *WingetAdapter) Detect() bool {
	return lookPath("winget")
}

func (a *WingetAdapter) Check() (adapters.UpdateInfo, error) {
	if !a.Detect() {
		return adapters.UpdateInfo{}, fmt.Errorf("winget is not installed")
	}

	// Self-only: report winget's own version from `winget --version`, then
	// scan `winget upgrade` (no args) for winget's own row to report a real
	// self-update availability. The version extraction tolerates a leading v
	// (e.g. "v1.8.2311") through isVersionLike/extractVersionFromString.
	current := commandOutput("winget", "--version")
	current = extractVersionFromString(current)
	if current == "" {
		current = "unknown"
	}

	latest := current
	found := false
	if out := commandOutput("winget", "upgrade"); out != "" {
		if _, lat, ok := parseWingetUpgradeOutput(out); ok && lat != "" {
			latest = lat
			found = true
		}
	}

	// winget < 1.6 lists no self row (found=false) → availability unavailable
	// gracefully, no error. AlwaysUpdate policy still runs the update when
	// requested, but Check() reports current (no phantom upgrade).
	return adapters.UpdateInfo{
		CurrentVersion:  current,
		LatestVersion:   latest,
		UpdateAvailable: found && current != latest,
	}, nil
}

func (a *WingetAdapter) Update(dryRun bool) (adapters.Result, error) {
	if !a.Detect() {
		return adapters.Result{Success: false}, fmt.Errorf("winget is not installed")
	}

	before := commandOutput("winget", "--version")
	before = extractVersionFromString(before)
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

	// Self-only: `winget upgrade winget` upgrades Windows Package Manager
	// itself (equiv. Microsoft.AppInstaller), never the packages it manages.
	// A bulk `winget upgrade --all` is intentionally avoided.
	_, stderr, err := runCmd("winget upgrade winget")
	if err != nil {
		return adapters.Result{
			Success: false,
			Before:  before,
			After:   before,
			Error:   fmt.Errorf("winget upgrade failed: %w", err),
		}, nil
	}

	if stderr != "" && strings.Contains(stderr, "Error") {
		return adapters.Result{
			Success: false,
			Before:  before,
			After:   before,
			Error:   fmt.Errorf("winget upgrade error: %s", truncate(stderr, 200)),
		}, nil
	}

	after := commandOutput("winget", "--version")
	after = extractVersionFromString(after)
	if after == "" {
		after = "unknown"
	}

	return adapters.Result{
		Success: true,
		Before:  before,
		After:   after,
	}, nil
}

func (a *WingetAdapter) Info() adapters.ToolInfo {
	return adapters.ToolInfo{
		ID:           "winget",
		Name:         "Windows Package Manager",
		Platforms:    []string{"windows"},
		Trust:        adapters.TrustOfficial,
		UpdatePolicy: adapters.PolicyAlwaysUpdate,
		Kind:         adapters.KindManager,
	}
}
