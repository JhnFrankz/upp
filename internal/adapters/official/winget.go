package official

import (
	"context"
	"fmt"

	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/security"
)

// WingetAdapter manages Windows Package Manager packages.
type WingetAdapter struct{}

var wingetUpgradeQuery coalescedQuery

func wingetUpgradeOutput(ctx context.Context) (string, error) {
	return wingetUpgradeQuery.run(ctx, func(c context.Context) (string, error) {
		return commandOutputErr(c, "winget", "upgrade")
	})
}

// wingetSelfUpdateCmd is winget's real self-update command — declared by
// Info() and executed by Update() (design D2 single source of truth).
const wingetSelfUpdateCmd = "winget upgrade winget"

// wingetPackageUpdateTemplate is winget's per-package update command
// template; the PackagePlaceholder is rendered per owned package by
// UpdatePackage().
const wingetPackageUpdateTemplate = "winget upgrade <pkg>"

func (a *WingetAdapter) Name() string { return "winget" }

func (a *WingetAdapter) Detect() bool {
	return lookPath("winget")
}

func (a *WingetAdapter) Check(ctx context.Context) (adapters.UpdateInfo, error) {
	if !a.Detect() {
		return adapters.UpdateInfo{}, fmt.Errorf("%w: winget is not installed", adapters.ErrNotInstalled)
	}

	// Self-only: report winget's own version from `winget --version`, then
	// scan `winget upgrade` (no args) for winget's own row to report a real
	// self-update availability. The version extraction tolerates a leading v
	// (e.g. "v1.8.2311") through isVersionLike/extractVersionFromString.
	current := commandOutput(ctx, "winget", "--version")
	current = extractVersionFromString(current)
	if current == "" {
		current = "unknown"
	}

	latest := current
	found := false
	if out, err := wingetUpgradeOutput(ctx); err == nil && out != "" {
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

// CheckPackage reports the installed vs available version of an owned package
// (e.g. `gh`, `Docker.Docker`, `GoLang.Go`) under winget, so an owned tool's
// delegated Check() and the manager-group bulk path know a real update exists
// (design D2). It runs the no-argument `winget upgrade` listing, which is the
// READ-ONLY form that merely lists available upgrades; `winget upgrade <query>`
// is the MUTATING upgrade command and MUST NEVER be used on this path. The
// parser anchors on the supplied package Id in that listing. Like winget.Check,
// a package that lists no row is reported current (fail-closed: no phantom
// update).
func (a *WingetAdapter) CheckPackage(ctx context.Context, pkg string) (adapters.UpdateInfo, error) {
	if !a.Detect() {
		return adapters.UpdateInfo{}, fmt.Errorf("%w: winget is not installed", adapters.ErrNotInstalled)
	}
	if err := validatePackageName(pkg); err != nil {
		return adapters.UpdateInfo{}, err
	}
	stdout, err := wingetUpgradeOutput(ctx)
	if err != nil {
		return adapters.UpdateInfo{}, err
	}
	current, latest, found := parseWingetPackageUpgradeOutput(stdout, pkg)
	return adapters.UpdateInfo{
		CurrentVersion:  current,
		LatestVersion:   latest,
		UpdateAvailable: found && current != latest,
	}, nil
}

func (a *WingetAdapter) installedVersion(ctx context.Context, pkg string) string {
	return parseWingetPackageListOutput(commandOutput(ctx, "winget", "list", pkg), pkg)
}

// UpdatePackage runs the per-package update command for an owned tool under
// winget (e.g. `winget upgrade gh`), via winget's executor. This is the
// manager-group bulk path (design D3): it upgrades the owned PACKAGE, NOT
// winget's self-only `winget upgrade winget`. winget is a non-privileged
// manager, so the group update may auto-proceed.
func (a *WingetAdapter) UpdatePackage(ctx context.Context, pkg string) (adapters.Result, error) {
	if !a.Detect() {
		return adapters.Result{Success: false}, fmt.Errorf("%w: winget is not installed", adapters.ErrNotInstalled)
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

	return executeUpdateCmd(ctx, updateExecConfig{
		label:   "winget upgrade",
		before:  before,
		cmd:     "winget",
		args:    []string{"upgrade", pkg},
		onDone:  wingetUpgradeQuery.invalidate,
		afterFn: func() string { return a.installedVersion(ctx, pkg) },
	})
}

func (a *WingetAdapter) Update(ctx context.Context, dryRun bool) (adapters.Result, error) {
	if !a.Detect() {
		return adapters.Result{Success: false}, fmt.Errorf("%w: winget is not installed", adapters.ErrNotInstalled)
	}

	before := commandOutput(ctx, "winget", "--version")
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
	return executeUpdateCmd(ctx, updateExecConfig{
		label:  "winget upgrade",
		before: before,
		cmd:    "winget",
		args:   []string{"upgrade", "winget"},
		onDone: wingetUpgradeQuery.invalidate,
		afterFn: func() string {
			if after := extractVersionFromString(commandOutput(ctx, "winget", "--version")); after != "" {
				return after
			}
			return "unknown"
		},
	})
}

var wingetInfo = adapters.ToolInfo{
	ID:                   "winget",
	Name:                 "Windows Package Manager",
	Platforms:            platformsWindows,
	Trust:                security.TrustOfficial,
	UpdatePolicy:         adapters.PolicyAlwaysUpdate,
	Kind:                 adapters.KindManager,
	SelfUpdateCommand:    wingetSelfUpdateCmd,
	PackageUpdateCommand: wingetPackageUpdateTemplate,
}

func (a *WingetAdapter) Info() adapters.ToolInfo {
	return wingetInfo
}
