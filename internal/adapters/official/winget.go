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

	// winget list outputs a table; parsing exact versions is complex.
	// For MVP, we report "unknown" and rely on the update command.
	_ = commandOutput("winget", "list")
	return adapters.UpdateInfo{
		CurrentVersion:  "unknown",
		LatestVersion:   "unknown",
		UpdateAvailable: true, // Assume updates may be available
	}, nil
}

func (a *WingetAdapter) Update(dryRun bool) (adapters.Result, error) {
	if !a.Detect() {
		return adapters.Result{Success: false}, fmt.Errorf("winget is not installed")
	}

	if dryRun {
		return adapters.Result{
			Success: true,
			Before:  "unknown",
			After:   "unknown",
		}, nil
	}

	_, stderr, err := runCmd("winget upgrade --all --accept-source-agreements --accept-package-agreements")
	if err != nil {
		return adapters.Result{
			Success: false,
			Before:  "unknown",
			After:   "unknown",
			Error:   fmt.Errorf("winget upgrade failed: %w", err),
		}, nil
	}

	if stderr != "" && strings.Contains(stderr, "Error") {
		return adapters.Result{
			Success: false,
			Before:  "unknown",
			After:   "unknown",
			Error:   fmt.Errorf("winget upgrade error: %s", truncate(stderr, 200)),
		}, nil
	}

	return adapters.Result{
		Success: true,
		Before:  "unknown",
		After:   "unknown",
	}, nil
}

func (a *WingetAdapter) Info() adapters.ToolInfo {
	return adapters.ToolInfo{
		ID:           "winget",
		Name:         "Windows Package Manager",
		Platforms:    []string{"windows"},
		Trust:        adapters.TrustOfficial,
		UpdatePolicy: adapters.PolicyAlwaysUpdate,
	}
}
