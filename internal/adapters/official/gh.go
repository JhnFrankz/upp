package official

import (
	"context"
	"fmt"

	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/platform"
	"github.com/JhnFrankz/upp/internal/security"
)

// GhAdapter manages GitHub CLI across platforms.
// Linux: apt, macOS: brew, Windows: winget.
type GhAdapter struct{}

func (a *GhAdapter) Name() string { return "gh" }

func (a *GhAdapter) Detect() bool {
	return lookPath("gh")
}

func (a *GhAdapter) Check(ctx context.Context) (adapters.UpdateInfo, error) {
	if !a.Detect() {
		return adapters.UpdateInfo{}, fmt.Errorf("gh is not installed")
	}

	current := extractVersion(commandOutput(ctx, "gh", "--version"))

	// Delegated check path (WU2, spec Per-Owned-Tool Availability): an owned
	// tool's Check() reports the real update of its package under the
	// resolving manager, NOT the manager's own self check. gh is owned on
	// every supported platform, so it delegates to the manager's CheckPackage
	// for gh.ManagerPackage[platform] (e.g. `apt-cache policy gh`,
	// `brew outdated --json gh`, `winget upgrade`). runtime.GOOS is
	// translated to the platform key because the manager/package maps are
	// keyed by PLATFORM constants, not runtime.GOOS (darwin) — the
	// WU1-documented gotcha.
	plat, _ := platform.NormalizeOS(runtimeGOOSFn())
	if owner := ResolveOwner("gh", plat); owner != nil {
		if !owner.Detect() {
			return adapters.UpdateInfo{
				CurrentVersion:  current,
				LatestVersion:   current,
				UpdateAvailable: false,
			}, nil
		}
		if checker, ok := owner.(adapters.PackageChecker); ok {
			pkg := a.Info().ManagerPackage[plat]
			if pkg == "" {
				return adapters.UpdateInfo{}, fmt.Errorf("gh has no manager package on %s", runtimeGOOSFn())
			}
			return checker.CheckPackage(ctx, pkg)
		}
		return adapters.UpdateInfo{}, fmt.Errorf("gh's manager %s does not support per-package checks", runtimeGOOSFn())
	}

	return adapters.UpdateInfo{}, fmt.Errorf("gh has no resolving owner on %s", runtimeGOOSFn())
}

func (a *GhAdapter) Update(ctx context.Context, dryRun bool) (adapters.Result, error) {
	if !a.Detect() {
		return adapters.Result{Success: false}, fmt.Errorf("gh is not installed")
	}

	// Delegated update path: an owned tool delegates to its resolving manager's
	// PackageUpdater interface to upgrade its specific package name (e.g. `gh`),
	// rather than triggering manager self-update.
	plat, _ := platform.NormalizeOS(runtimeGOOSFn())
	if owner := ResolveOwner("gh", plat); owner != nil {
		if dryRun {
			return adapters.Result{Success: true}, nil
		}
		if updater, ok := owner.(adapters.PackageUpdater); ok {
			pkg := a.Info().ManagerPackage[plat]
			if pkg == "" {
				return adapters.Result{Success: false}, fmt.Errorf("gh has no manager package on %s", runtimeGOOSFn())
			}
			return updater.UpdatePackage(ctx, pkg)
		}
		return adapters.Result{Success: false}, fmt.Errorf("gh's manager %s does not support per-package updates", runtimeGOOSFn())
	}

	// Fail-closed fallback if ownership map ever regresses.
	return adapters.Result{
		Success: false,
		Error:   fmt.Errorf("gh has no resolving owner on %s", runtimeGOOSFn()),
	}, nil
}

func (a *GhAdapter) Info() adapters.ToolInfo {
	mgr := defaultLinuxManager()
	pkg := "gh"
	if mgr == "pacman" {
		pkg = "github-cli"
	}
	return adapters.ToolInfo{
		ID:             "gh",
		Name:           "GitHub CLI",
		Platforms:      []string{"linux", "macos", "windows"},
		Trust:          security.TrustOfficial,
		UpdatePolicy:   adapters.PolicyAlwaysUpdate,
		Kind:           adapters.KindTool,
		Manager:        map[string]string{"linux": mgr, "macos": "brew", "windows": "winget"},
		ManagerPackage: map[string]string{"linux": pkg, "macos": "gh", "windows": "gh"},
	}
}
