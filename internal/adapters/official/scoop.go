package official

import (
	"fmt"
	"strings"

	"github.com/JhnFrankz/upp/internal/adapters"
)

// ScoopAdapter manages Scoop packages on Windows.
type ScoopAdapter struct{}

func (a *ScoopAdapter) Name() string { return "scoop" }

func (a *ScoopAdapter) Detect() bool {
	return lookPath("scoop")
}

func (a *ScoopAdapter) Check() (adapters.UpdateInfo, error) {
	if !a.Detect() {
		return adapters.UpdateInfo{}, fmt.Errorf("scoop is not installed")
	}

	// scoop status outputs installed vs latest versions.
	// For MVP, we report "unknown" and rely on the update command.
	_ = commandOutput("scoop", "status")
	return adapters.UpdateInfo{
		CurrentVersion:  "unknown",
		LatestVersion:   "unknown",
		UpdateAvailable: true, // Assume updates may be available
	}, nil
}

func (a *ScoopAdapter) Update(dryRun bool) (adapters.Result, error) {
	if !a.Detect() {
		return adapters.Result{Success: false}, fmt.Errorf("scoop is not installed")
	}

	if dryRun {
		return adapters.Result{
			Success: true,
			Before:  "unknown",
			After:   "unknown",
		}, nil
	}

	_, stderr, err := runCmd("scoop update *")
	if err != nil {
		return adapters.Result{
			Success: false,
			Before:  "unknown",
			After:   "unknown",
			Error:   fmt.Errorf("scoop update failed: %w", err),
		}, nil
	}

	if stderr != "" && strings.Contains(stderr, "ERROR") {
		return adapters.Result{
			Success: false,
			Before:  "unknown",
			After:   "unknown",
			Error:   fmt.Errorf("scoop update error: %s", truncate(stderr, 200)),
		}, nil
	}

	return adapters.Result{
		Success: true,
		Before:  "unknown",
		After:   "unknown",
	}, nil
}

func (a *ScoopAdapter) Info() adapters.ToolInfo {
	return adapters.ToolInfo{
		ID:        "scoop",
		Name:      "Scoop",
		Platforms: []string{"windows"},
		Trust:     adapters.TrustOfficial,
	}
}
