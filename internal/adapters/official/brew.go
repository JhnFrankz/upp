package official

import (
	"fmt"
	"strings"

	"github.com/JhnFrankz/upp/internal/adapters"
)

// BrewAdapter manages Homebrew packages on Linux and macOS.
type BrewAdapter struct{}

func (a *BrewAdapter) Name() string { return "brew" }

func (a *BrewAdapter) Detect() bool {
	return lookPath("brew")
}

func (a *BrewAdapter) Check() (adapters.UpdateInfo, error) {
	if !a.Detect() {
		return adapters.UpdateInfo{}, fmt.Errorf("brew is not installed")
	}

	current := commandOutput("brew", "--version")
	current = extractVersionFromString(current)

	// brew doesn't have a direct "latest version" command for itself,
	// but outdated packages can be checked.
	latest := current // Assume current is latest for brew itself.

	return adapters.UpdateInfo{
		CurrentVersion:  current,
		LatestVersion:   latest,
		UpdateAvailable: false, // brew self-updates via `brew update`
	}, nil
}

// CheckPackage reports the installed vs latest version of an owned package
// (e.g. `gh`, `docker`, `golang`) under brew, so an owned tool's delegated
// Check() and the manager-group bulk path know a real update exists (design
// D2). It runs `brew outdated --json <pkg>` and parses the JSON array. brew
// is an AlwaysUpdate manager, so this is still the real availability signal:
// a brew formula present in the outdated JSON array has a newer version.
func (a *BrewAdapter) CheckPackage(pkg string) (adapters.UpdateInfo, error) {
	stdout, err := commandOutputErr("brew", "outdated", "--json", pkg)
	if err != nil {
		return adapters.UpdateInfo{}, err
	}
	current, latest, found := parseBrewOutdatedJSON(stdout)
	return adapters.UpdateInfo{
		CurrentVersion:  current,
		LatestVersion:   latest,
		UpdateAvailable: found,
	}, nil
}

func (a *BrewAdapter) Update(dryRun bool) (adapters.Result, error) {
	if !a.Detect() {
		return adapters.Result{Success: false}, fmt.Errorf("brew is not installed")
	}

	before := commandOutput("brew", "--version")
	before = extractVersionFromString(before)

	if dryRun {
		return adapters.Result{
			Success: true,
			Before:  before,
			After:   before,
		}, nil
	}

	// Self-only: `brew update` refreshes Homebrew's git metadata (a mutating
	// operation, so it must never run inside Check) but does NOT bump the
	// versions of the packages brew manages. `brew upgrade brew` is
	// intentionally avoided — it is non-canonical and is a known portable-ruby
	// footgun (Homebrew's ruby shims make it error-prone).
	_, stderr, err := runCmd("brew update")
	if err != nil {
		return adapters.Result{
			Success: false,
			Before:  before,
			After:   before,
			Error:   fmt.Errorf("brew upgrade failed: %w", err),
		}, nil
	}

	if stderr != "" && strings.Contains(stderr, "Error") {
		return adapters.Result{
			Success: false,
			Before:  before,
			After:   before,
			Error:   fmt.Errorf("brew upgrade error: %s", truncate(stderr, 200)),
		}, nil
	}

	after := commandOutput("brew", "--version")
	after = extractVersionFromString(after)

	return adapters.Result{
		Success: true,
		Before:  before,
		After:   after,
	}, nil
}

func (a *BrewAdapter) Info() adapters.ToolInfo {
	return adapters.ToolInfo{
		ID:           "brew",
		Name:         "Homebrew",
		Platforms:    []string{"linux", "macos"},
		Trust:        adapters.TrustOfficial,
		UpdatePolicy: adapters.PolicyAlwaysUpdate,
		Kind:         adapters.KindManager,
	}
}

// extractVersionFromString extracts a version from a "brew X.Y.Z" string.
func extractVersionFromString(s string) string {
	s = strings.TrimSpace(s)
	// "Homebrew 4.1.0" → "4.1.0"
	fields := strings.Fields(s)
	for _, field := range fields {
		if isVersionLike(field) {
			return field
		}
	}
	return s
}
