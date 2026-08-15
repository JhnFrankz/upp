package official

import (
	"fmt"
	"strings"

	"github.com/JhnFrankz/upp/internal/adapters"
)

// BunAdapter manages Bun runtime on all platforms.
type BunAdapter struct{}

func (a *BunAdapter) Name() string { return "bun" }

func (a *BunAdapter) Detect() bool {
	return lookPath("bun")
}

func (a *BunAdapter) Check() (adapters.UpdateInfo, error) {
	if !a.Detect() {
		return adapters.UpdateInfo{}, fmt.Errorf("bun is not installed")
	}

	current := commandOutput("bun", "--version")
	current = strings.TrimSpace(current)
	if current == "" {
		current = "unknown"
	}

	return adapters.UpdateInfo{
		CurrentVersion:  current,
		LatestVersion:   current, // bun upgrade handles version resolution internally
		UpdateAvailable: false,
	}, nil
}

func (a *BunAdapter) Update(dryRun bool) (adapters.Result, error) {
	if !a.Detect() {
		return adapters.Result{Success: false}, fmt.Errorf("bun is not installed")
	}

	before := commandOutput("bun", "--version")
	before = strings.TrimSpace(before)

	if dryRun {
		return adapters.Result{
			Success: true,
			Before:  before,
			After:   before,
		}, nil
	}

	_, stderr, err := runCmd("bun upgrade")
	if err != nil {
		return adapters.Result{
			Success: false,
			Before:  before,
			After:   before,
			Error:   fmt.Errorf("bun upgrade failed: %w", err),
		}, nil
	}

	if stderr != "" && strings.Contains(stderr, "error") {
		return adapters.Result{
			Success: false,
			Before:  before,
			After:   before,
			Error:   fmt.Errorf("bun upgrade error: %s", truncate(stderr, 200)),
		}, nil
	}

	after := commandOutput("bun", "--version")
	after = strings.TrimSpace(after)

	return adapters.Result{
		Success: true,
		Before:  before,
		After:   after,
	}, nil
}

func (a *BunAdapter) Info() adapters.ToolInfo {
	return adapters.ToolInfo{
		ID:           "bun",
		Name:         "Bun",
		Platforms:    []string{"linux", "macos", "windows"},
		Trust:        adapters.TrustOfficial,
		UpdatePolicy: adapters.PolicyAlwaysUpdate,
	}
}
