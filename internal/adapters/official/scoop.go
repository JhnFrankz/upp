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

	// Self-only: scan `scoop status` output for scoop's own row to report a
	// real self-update availability. `scoop --version` is intentionally NOT
	// used — it reports a script commit hash rather than a usable version.
	// The parse is fail-closed: an absent or unparseable scoop row (found =
	// false, e.g. an old scoop or an unstable output shape) falls back to
	// current-only and reports no availability, no error.
	current := "unknown"
	latest := "unknown"
	if out := commandOutput("scoop", "status"); out != "" {
		if cur, lat, ok := parseScoopStatusOutput(out); ok {
			current = cur
			latest = lat
		}
	}

	return adapters.UpdateInfo{
		CurrentVersion:  current,
		LatestVersion:   latest,
		UpdateAvailable: current != "unknown" && latest != "unknown" && current != latest,
	}, nil
}

func (a *ScoopAdapter) Update(dryRun bool) (adapters.Result, error) {
	if !a.Detect() {
		return adapters.Result{Success: false}, fmt.Errorf("scoop is not installed")
	}

	before := "unknown"
	if out := commandOutput("scoop", "status"); out != "" {
		if cur, _, ok := parseScoopStatusOutput(out); ok && cur != "" {
			before = cur
		}
	}

	if dryRun {
		return adapters.Result{
			Success: true,
			Before:  before,
			After:   before,
		}, nil
	}

	// Self-only: `scoop update scoop` upgrades Scoop itself, never the
	// packages it manages. A bulk `scoop update *` (which updates every
	// app) is intentionally avoided — self-only semantics per point 4.
	_, stderr, err := runCmd("scoop update scoop")
	if err != nil {
		return adapters.Result{
			Success: false,
			Before:  before,
			After:   before,
			Error:   fmt.Errorf("scoop update failed: %w", err),
		}, nil
	}

	if stderr != "" && strings.Contains(stderr, "ERROR") {
		return adapters.Result{
			Success: false,
			Before:  before,
			After:   before,
			Error:   fmt.Errorf("scoop update error: %s", truncate(stderr, 200)),
		}, nil
	}

	after := before
	if out := commandOutput("scoop", "status"); out != "" {
		if cur, _, ok := parseScoopStatusOutput(out); ok && cur != "" {
			after = cur
		}
	}

	return adapters.Result{
		Success: true,
		Before:  before,
		After:   after,
	}, nil
}

func (a *ScoopAdapter) Info() adapters.ToolInfo {
	return adapters.ToolInfo{
		ID:           "scoop",
		Name:         "Scoop",
		Platforms:    []string{"windows"},
		Trust:        adapters.TrustOfficial,
		UpdatePolicy: adapters.PolicyAlwaysUpdate,
	}
}
