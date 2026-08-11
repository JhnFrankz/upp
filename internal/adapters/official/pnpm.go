package official

import (
	"fmt"
	"strings"

	"github.com/JhnFrankz/upp/internal/adapters"
)

// PnpmAdapter manages pnpm global packages on all platforms.
// Includes corruption recovery: if `pnpm update -g` fails, it attempts
// to remove and reinstall the global store.
type PnpmAdapter struct{}

func (a *PnpmAdapter) Name() string { return "pnpm" }

func (a *PnpmAdapter) Detect() bool {
	return lookPath("pnpm")
}

func (a *PnpmAdapter) Check() (adapters.UpdateInfo, error) {
	if !a.Detect() {
		return adapters.UpdateInfo{}, fmt.Errorf("pnpm is not installed")
	}

	current := commandOutput("pnpm", "--version")
	current = strings.TrimSpace(current)
	if current == "" {
		current = "unknown"
	}

	// Check for outdated packages (non-blocking, timeout-safe)
	stdout := shellOutput("timeout 15 pnpm outdated -g 2>/dev/null || true")
	updateAvailable := strings.Contains(stdout, "│") && !strings.Contains(stdout, "Package")

	return adapters.UpdateInfo{
		CurrentVersion:  current,
		LatestVersion:   current, // pnpm doesn't easily expose its own latest version
		UpdateAvailable: updateAvailable,
	}, nil
}

func (a *PnpmAdapter) Update(dryRun bool) (adapters.Result, error) {
	if !a.Detect() {
		return adapters.Result{Success: false}, fmt.Errorf("pnpm is not installed")
	}

	before := commandOutput("pnpm", "--version")
	before = strings.TrimSpace(before)

	if dryRun {
		return adapters.Result{
			Success: true,
			Before:  before,
			After:   before,
		}, nil
	}

	// First attempt: standard update.
	_, stderr, err := runCmd("pnpm update -g")
	if err == nil {
		after := commandOutput("pnpm", "--version")
		after = strings.TrimSpace(after)
		return adapters.Result{
			Success: true,
			Before:  before,
			After:   after,
		}, nil
	}

	// Corruption recovery: remove global store and retry.
	if stderr != "" && (strings.Contains(stderr, "corrupt") || strings.Contains(stderr, "ENOENT") || strings.Contains(stderr, "EACCES")) {
		_, _, _ = runCmd("pnpm store prune 2>/dev/null")
		_, stderr2, err2 := runCmd("pnpm update -g")
		if err2 == nil {
			after := commandOutput("pnpm", "--version")
			after = strings.TrimSpace(after)
			return adapters.Result{
				Success: true,
				Before:  before,
				After:   after,
			}, nil
		}
		return adapters.Result{
			Success: false,
			Before:  before,
			After:   before,
			Error:   fmt.Errorf("pnpm update failed (even after recovery): %w (stderr: %s)", err2, truncate(stderr2, 200)),
		}, nil
	}

	return adapters.Result{
		Success: false,
		Before:  before,
		After:   before,
		Error:   fmt.Errorf("pnpm update failed: %w", err),
	}, nil
}

func (a *PnpmAdapter) Info() adapters.ToolInfo {
	return adapters.ToolInfo{
		ID:        "pnpm",
		Name:      "pnpm",
		Platforms: []string{"linux", "macos", "windows"},
		Trust:     adapters.TrustOfficial,
	}
}
