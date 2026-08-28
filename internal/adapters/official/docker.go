package official

import (
	"fmt"
	"runtime"

	"github.com/JhnFrankz/upp/internal/adapters"
)

// DockerAdapter manages Docker across platforms.
// Linux: apt, macOS: brew, Windows: winget.
type DockerAdapter struct{}

func (a *DockerAdapter) Name() string { return "docker" }

func (a *DockerAdapter) Detect() bool {
	return lookPath("docker")
}

func (a *DockerAdapter) Check() (adapters.UpdateInfo, error) {
	if !a.Detect() {
		return adapters.UpdateInfo{}, fmt.Errorf("docker is not installed")
	}

	// Delegated check path (WU2, spec Per-Owned-Tool Availability): an owned
	// tool's Check() reports the real update of its package under the
	// resolving manager, NOT the manager's own self check. docker is owned on
	// every supported platform, so it delegates to the manager's CheckPackage
	// for docker.ManagerPackage[platform] (e.g. `apt-cache policy docker-ce`,
	// `brew outdated --json docker`, `winget upgrade Docker.Docker`).
	// runtime.GOOS is translated to the platform key because the manager and
	// package maps are keyed by PLATFORM constants, not runtime.GOOS (darwin).
	platform := runtimeGOOSToPlatform(runtime.GOOS)
	if owner := ResolveOwner("docker", platform); owner != nil {
		if checker, ok := owner.(adapters.PackageChecker); ok {
			pkg := a.Info().ManagerPackage[platform]
			if pkg == "" {
				return adapters.UpdateInfo{}, fmt.Errorf("docker has no manager package on %s", runtime.GOOS)
			}
			return checker.CheckPackage(pkg)
		}
		return adapters.UpdateInfo{}, fmt.Errorf("docker's manager %s does not support per-package checks", runtime.GOOS)
	}

	return adapters.UpdateInfo{}, fmt.Errorf("docker has no resolving owner on %s", runtime.GOOS)
}

func (a *DockerAdapter) Update(dryRun bool) (adapters.Result, error) {
	if !a.Detect() {
		return adapters.Result{Success: false}, fmt.Errorf("docker is not installed")
	}

	// Delegated update path (WU2, spec Official Adapter Catalog / Resolved
	// Owner Update Delegation): an owned tool delegates to its resolving
	// manager rather than run its own hardcoded manager command. The manager's
	// Update() runs its self-only command — never an
	// "apt upgrade docker-ce ..." / "brew upgrade docker" invocation.
	// runtime.GOOS is translated to the platform key because ResolveOwner is
	// keyed by PLATFORM constants, not runtime.GOOS (darwin).
	if owner := ResolveOwner("docker", runtimeGOOSToPlatform(runtime.GOOS)); owner != nil {
		return owner.Update(dryRun)
	}

	// Unreachable in practice: docker is owned on every supported platform.
	// Fail-closed fallback if the ownership map ever regresses.
	return adapters.Result{
		Success: false,
		Error:   fmt.Errorf("docker has no resolving owner on %s", runtime.GOOS),
	}, nil
}

func (a *DockerAdapter) Info() adapters.ToolInfo {
	return adapters.ToolInfo{
		ID:             "docker",
		Name:           "Docker",
		Platforms:      []string{"linux", "macos", "windows"},
		Trust:          adapters.TrustOfficial,
		UpdatePolicy:   adapters.PolicyAlwaysUpdate,
		Kind:           adapters.KindTool,
		Manager:        map[string]string{"linux": "apt", "macos": "brew", "windows": "winget"},
		ManagerPackage: map[string]string{"linux": "docker-ce", "macos": "docker", "windows": "Docker.Docker"},
	}
}
