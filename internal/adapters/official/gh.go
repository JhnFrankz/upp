package official

import (
	"fmt"
	"runtime"

	"github.com/JhnFrankz/upp/internal/adapters"
)

// GhAdapter manages GitHub CLI across platforms.
// Linux: apt, macOS: brew, Windows: winget.
type GhAdapter struct{}

func (a *GhAdapter) Name() string { return "gh" }

func (a *GhAdapter) Detect() bool {
	return lookPath("gh")
}

func (a *GhAdapter) Check() (adapters.UpdateInfo, error) {
	if !a.Detect() {
		return adapters.UpdateInfo{}, fmt.Errorf("gh is not installed")
	}

	// Delegated check path (WU2, spec Per-Owned-Tool Availability): an owned
	// tool's Check() reports the real update of its package under the
	// resolving manager, NOT the manager's own self check. gh is owned on
	// every supported platform, so it delegates to the manager's CheckPackage
	// for gh.ManagerPackage[platform] (e.g. `apt-cache policy gh`,
	// `brew outdated --json gh`, `winget upgrade gh`). runtime.GOOS is
	// translated to the platform key because the manager/package maps are
	// keyed by PLATFORM constants, not runtime.GOOS (darwin) — the
	// WU1-documented gotcha.
	platform := runtimeGOOSToPlatform(runtime.GOOS)
	if owner := ResolveOwner("gh", platform); owner != nil {
		if checker, ok := owner.(adapters.PackageChecker); ok {
			pkg := a.Info().ManagerPackage[platform]
			if pkg == "" {
				return adapters.UpdateInfo{}, fmt.Errorf("gh has no manager package on %s", runtime.GOOS)
			}
			return checker.CheckPackage(pkg)
		}
		return adapters.UpdateInfo{}, fmt.Errorf("gh's manager %s does not support per-package checks", runtime.GOOS)
	}

	return adapters.UpdateInfo{}, fmt.Errorf("gh has no resolving owner on %s", runtime.GOOS)
}

func (a *GhAdapter) Update(dryRun bool) (adapters.Result, error) {
	if !a.Detect() {
		return adapters.Result{Success: false}, fmt.Errorf("gh is not installed")
	}

	// Delegated update path (WU2, spec Official Adapter Catalog / Resolved
	// Owner Update Delegation): an owned tool delegates to its resolving
	// manager rather than run its own hardcoded manager command. The manager's
	// Update() runs its self-only command (apt self-only, brew update, winget
	// self-only) — never an "apt install gh" / "brew upgrade gh" invocation.
	// runtime.GOOS is translated to the platform key because ResolveOwner is
	// keyed by PLATFORM constants (linux/macos/windows), not runtime.GOOS
	// (darwin) — the WU1-documented gotcha.
	if owner := ResolveOwner("gh", runtimeGOOSToPlatform(runtime.GOOS)); owner != nil {
		return owner.Update(dryRun)
	}

	// Unreachable in practice: gh is owned on every supported platform. Kept
	// as a fail-closed fallback rather than silently returning an empty result
	// if the ownership map ever regresses.
	return adapters.Result{
		Success: false,
		Error:   fmt.Errorf("gh has no resolving owner on %s", runtime.GOOS),
	}, nil
}

func (a *GhAdapter) Info() adapters.ToolInfo {
	return adapters.ToolInfo{
		ID:             "gh",
		Name:           "GitHub CLI",
		Platforms:      []string{"linux", "macos", "windows"},
		Trust:          adapters.TrustOfficial,
		UpdatePolicy:   adapters.PolicyAlwaysUpdate,
		Kind:           adapters.KindTool,
		Manager:        map[string]string{"linux": "apt", "macos": "brew", "windows": "winget"},
		ManagerPackage: map[string]string{"linux": "gh", "macos": "gh", "windows": "gh"},
	}
}
