package official

import (
	"context"
	"fmt"
	"strings"

	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/security"
)

// ScoopAdapter manages Scoop packages on Windows.
type ScoopAdapter struct{}

var scoopStatusQuery coalescedQuery

func scoopStatusOutput(ctx context.Context) (string, error) {
	return scoopStatusQuery.run(ctx, func(c context.Context) (string, error) {
		return commandOutputErr(c, "scoop", "status")
	})
}

// scoopSelfUpdateCmd is scoop's real self-update command — declared by Info()
// and executed by Update() (design D2 single source of truth).
const (
	scoopSelfUpdateCmd         = "scoop update scoop"
	scoopPackageUpdateTemplate = "scoop update <pkg>"
)

func (a *ScoopAdapter) installedVersion(ctx context.Context, pkg string) string {
	return parseScoopListOutput(commandOutput(ctx, "scoop", "list", pkg), pkg)
}

func (a *ScoopAdapter) Name() string { return "scoop" }

func (a *ScoopAdapter) Detect() bool {
	return lookPath("scoop")
}

func (a *ScoopAdapter) Check(ctx context.Context) (adapters.UpdateInfo, error) {
	if !a.Detect() {
		return adapters.UpdateInfo{}, fmt.Errorf("%w: scoop is not installed", adapters.ErrNotInstalled)
	}

	// Self-only: scan `scoop status` output for scoop's own row to report a
	// real self-update availability. `scoop --version` is intentionally NOT
	// used — it reports a script commit hash rather than a usable version.
	// The parse is fail-closed: an absent or unparseable scoop row (found =
	// false, e.g. an old scoop or an unstable output shape) falls back to
	// current-only and reports no availability, no error.
	current := "unknown"
	latest := "unknown"
	out, err := scoopStatusOutput(ctx)
	if err != nil {
		return adapters.UpdateInfo{}, err
	}
	if out != "" {
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

func (a *ScoopAdapter) Update(ctx context.Context, dryRun bool) (adapters.Result, error) {
	if !a.Detect() {
		return adapters.Result{Success: false}, fmt.Errorf("%w: scoop is not installed", adapters.ErrNotInstalled)
	}

	before := "unknown"
	if out, err := scoopStatusOutput(ctx); err == nil && out != "" {
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
	_, stderr, err := runCmdArgsUpdate(ctx, "scoop", "update", "scoop")
	scoopStatusQuery.invalidate()
	if err != nil {
		return adapters.Result{
			Success: false,
			Before:  before,
			After:   before,
			Error:   fmt.Errorf("scoop update failed: %w", err),
			Stderr:  stderr,
		}, nil
	}

	if stderr != "" && strings.Contains(stderr, "ERROR") {
		return adapters.Result{
			Success: false,
			Before:  before,
			After:   before,
			Error:   fmt.Errorf("scoop update error: %s", truncate(stderr, 200)),
			Stderr:  stderr,
		}, nil
	}

	after := before
	if v := a.installedVersion(ctx, "scoop"); v != "" {
		after = v
	}

	return adapters.Result{
		Success: true,
		Before:  before,
		After:   after,
	}, nil
}

// CheckPackage reports the installed vs available version of an owned package under scoop.
func (a *ScoopAdapter) CheckPackage(ctx context.Context, pkg string) (adapters.UpdateInfo, error) {
	if !a.Detect() {
		return adapters.UpdateInfo{}, fmt.Errorf("%w: scoop is not installed", adapters.ErrNotInstalled)
	}
	if err := validatePackageName(pkg); err != nil {
		return adapters.UpdateInfo{}, err
	}

	current := "unknown"
	latest := "unknown"
	found := false
	out, err := scoopStatusOutput(ctx)
	if err != nil {
		return adapters.UpdateInfo{}, err
	}
	if out != "" {
		if cur, lat, ok := parseScoopPackageStatusOutput(out, pkg); ok {
			current = cur
			latest = lat
			found = true
		}
	}

	return adapters.UpdateInfo{
		CurrentVersion:  current,
		LatestVersion:   latest,
		UpdateAvailable: found && current != "unknown" && latest != "unknown" && current != latest,
	}, nil
}

// UpdatePackage runs the per-package update command for an owned tool under scoop.
func (a *ScoopAdapter) UpdatePackage(ctx context.Context, pkg string) (adapters.Result, error) {
	if !a.Detect() {
		return adapters.Result{Success: false}, fmt.Errorf("%w: scoop is not installed", adapters.ErrNotInstalled)
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

	_, stderr, err := runCmdArgsUpdate(ctx, "scoop", "update", pkg)
	scoopStatusQuery.invalidate()
	if err != nil {
		return adapters.Result{
			Success: false,
			Before:  before,
			After:   before,
			Error:   fmt.Errorf("scoop update failed: %w", err),
			Stderr:  stderr,
		}, nil
	}

	if stderr != "" && strings.Contains(stderr, "ERROR") {
		return adapters.Result{
			Success: false,
			Before:  before,
			After:   before,
			Error:   fmt.Errorf("scoop update error: %s", truncate(stderr, 200)),
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

var scoopInfo = adapters.ToolInfo{
	ID:                   "scoop",
	Name:                 "Scoop",
	Platforms:            platformsWindows,
	Trust:                security.TrustOfficial,
	UpdatePolicy:         adapters.PolicyAlwaysUpdate,
	Kind:                 adapters.KindManager,
	SelfUpdateCommand:    scoopSelfUpdateCmd,
	PackageUpdateCommand: scoopPackageUpdateTemplate,
}

func (a *ScoopAdapter) Info() adapters.ToolInfo {
	return scoopInfo
}
