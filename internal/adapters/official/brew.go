package official

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/security"
)

// BrewAdapter manages Homebrew packages on Linux and macOS.
type BrewAdapter struct{}

var (
	brewOutdatedMu      sync.Mutex
	brewOutdatedQueries = map[string]*coalescedQuery{}
)

func brewOutdatedQueryFor(pkg string) *coalescedQuery {
	brewOutdatedMu.Lock()
	defer brewOutdatedMu.Unlock()
	q, ok := brewOutdatedQueries[pkg]
	if !ok {
		q = &coalescedQuery{}
		brewOutdatedQueries[pkg] = q
	}
	return q
}

func invalidateBrewOutdated(pkg string) {
	brewOutdatedMu.Lock()
	q := brewOutdatedQueries[pkg]
	brewOutdatedMu.Unlock()
	if q != nil {
		q.invalidate()
	}
}

func resetBrewOutdatedCache() {
	brewOutdatedMu.Lock()
	for _, q := range brewOutdatedQueries {
		q.invalidate()
	}
	brewOutdatedQueries = map[string]*coalescedQuery{}
	brewOutdatedMu.Unlock()
}

// brewSelfUpdateCmd is brew's real self-update command — declared by Info()
// and executed by Update() (design D2 single source of truth).
const brewSelfUpdateCmd = "brew update"

// brewPackageUpdateTemplate is brew's per-package update command template;
// the PackagePlaceholder is rendered per owned package by UpdatePackage().
const brewPackageUpdateTemplate = "brew upgrade <pkg>"

func (a *BrewAdapter) Name() string { return "brew" }

func (a *BrewAdapter) Detect() bool {
	return lookPath("brew")
}

func (a *BrewAdapter) Check(ctx context.Context) (adapters.UpdateInfo, error) {
	if !a.Detect() {
		return adapters.UpdateInfo{}, fmt.Errorf("%w: brew is not installed", adapters.ErrNotInstalled)
	}

	current := commandOutput(ctx, "brew", "--version")
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
func (a *BrewAdapter) CheckPackage(ctx context.Context, pkg string) (adapters.UpdateInfo, error) {
	if !a.Detect() {
		return adapters.UpdateInfo{}, fmt.Errorf("%w: brew is not installed", adapters.ErrNotInstalled)
	}
	if err := validatePackageName(pkg); err != nil {
		return adapters.UpdateInfo{}, err
	}
	stdout, err := brewOutdatedQueryFor(pkg).run(ctx, func(c context.Context) (string, error) {
		return commandOutputErr(c, "brew", "outdated", "--json", pkg)
	})
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

func (a *BrewAdapter) installedVersion(ctx context.Context, pkg string) string {
	return parseBrewListVersionsOutput(commandOutput(ctx, "brew", "list", "--versions", pkg), pkg)
}

// UpdatePackage runs the per-package update command for an owned tool under
// brew (e.g. `brew upgrade gh`), via brew's executor. This is the manager-group
// bulk path (design D3): it upgrades the owned FORMULA, NOT brew's self-only
// `brew update`. brew is a non-privileged (no sudo) manager, so the group
// update may auto-proceed (spec security-model "Non-sudo group proceeds").
func (a *BrewAdapter) UpdatePackage(ctx context.Context, pkg string) (adapters.Result, error) {
	if !a.Detect() {
		return adapters.Result{Success: false}, fmt.Errorf("%w: brew is not installed", adapters.ErrNotInstalled)
	}
	if err := validatePackageName(pkg); err != nil {
		return adapters.Result{Success: false, Error: err}, err
	}

	before := "unknown"
	if info, err := a.CheckPackage(ctx, pkg); err == nil && info.CurrentVersion != "" {
		before = info.CurrentVersion
	} else if v := a.installedVersion(ctx, pkg); v != "" {
		before = v
	}

	_, stderr, err := runCmdArgsUpdate(ctx, "brew", "upgrade", pkg)
	invalidateBrewOutdated(pkg)
	if err != nil {
		return adapters.Result{
			Success: false,
			Before:  before,
			After:   before,
			Error:   fmt.Errorf("brew upgrade failed: %w", err),
			Stderr:  stderr,
		}, nil
	}

	if stderr != "" && strings.Contains(stderr, "Error") {
		return adapters.Result{
			Success: false,
			Before:  before,
			After:   before,
			Error:   fmt.Errorf("brew upgrade error: %s", truncate(stderr, 200)),
			Stderr:  stderr,
		}, nil
	}

	after := before
	if v := a.installedVersion(ctx, pkg); v != "" {
		after = v
	}

	return adapters.Result{
		Success: true,
		Before:  before,
		After:   after,
	}, nil
}

func (a *BrewAdapter) Update(ctx context.Context, dryRun bool) (adapters.Result, error) {
	if !a.Detect() {
		return adapters.Result{Success: false}, fmt.Errorf("%w: brew is not installed", adapters.ErrNotInstalled)
	}

	before := commandOutput(ctx, "brew", "--version")
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
	_, stderr, err := runCmdArgsUpdate(ctx, "brew", "update")
	resetBrewOutdatedCache()
	if err != nil {
		return adapters.Result{
			Success: false,
			Before:  before,
			After:   before,
			Error:   fmt.Errorf("brew upgrade failed: %w", err),
			Stderr:  stderr,
		}, nil
	}

	if stderr != "" && strings.Contains(stderr, "Error") {
		return adapters.Result{
			Success: false,
			Before:  before,
			After:   before,
			Error:   fmt.Errorf("brew upgrade error: %s", truncate(stderr, 200)),
			Stderr:  stderr,
		}, nil
	}

	after := commandOutput(ctx, "brew", "--version")
	after = extractVersionFromString(after)

	return adapters.Result{
		Success: true,
		Before:  before,
		After:   after,
	}, nil
}

var brewInfo = adapters.ToolInfo{
	ID:                   "brew",
	Name:                 "Homebrew",
	Platforms:            platformsLinuxMacOS,
	Trust:                security.TrustOfficial,
	UpdatePolicy:         adapters.PolicyAlwaysUpdate,
	Kind:                 adapters.KindManager,
	SelfUpdateCommand:    brewSelfUpdateCmd,
	PackageUpdateCommand: brewPackageUpdateTemplate,
}

func (a *BrewAdapter) Info() adapters.ToolInfo {
	return brewInfo
}
