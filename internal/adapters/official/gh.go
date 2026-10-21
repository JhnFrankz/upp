package official

import (
	"context"
	"fmt"

	"github.com/JhnFrankz/upp/internal/adapters"
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
		return adapters.UpdateInfo{}, fmt.Errorf("%w: gh is not installed", adapters.ErrNotInstalled)
	}

	current := extractVersion(commandOutput(ctx, "gh", "--version"))
	return DelegatedCheck(ctx, a, current)
}

func (a *GhAdapter) Update(ctx context.Context, dryRun bool) (adapters.Result, error) {
	if !a.Detect() {
		return adapters.Result{Success: false}, fmt.Errorf("%w: gh is not installed", adapters.ErrNotInstalled)
	}

	return DelegatedUpdate(ctx, a, dryRun)
}

var (
	ghManagerApt    = map[string]string{"linux": "apt", "macos": "brew", "windows": "winget"}
	ghManagerPacman = map[string]string{"linux": "pacman", "macos": "brew", "windows": "winget"}
	ghPkgApt        = map[string]string{"linux": "gh", "macos": "gh", "windows": "GitHub.cli"}
	ghPkgPacman     = map[string]string{"linux": "github-cli", "macos": "gh", "windows": "GitHub.cli"}

	ghInfoApt = adapters.ToolInfo{
		ID:             "gh",
		Name:           "GitHub CLI",
		Platforms:      platformsLinuxMacOSWindows,
		Trust:          security.TrustOfficial,
		UpdatePolicy:   adapters.PolicyAlwaysUpdate,
		Kind:           adapters.KindTool,
		Manager:        ghManagerApt,
		ManagerPackage: ghPkgApt,
	}
	ghInfoPacman = adapters.ToolInfo{
		ID:             "gh",
		Name:           "GitHub CLI",
		Platforms:      platformsLinuxMacOSWindows,
		Trust:          security.TrustOfficial,
		UpdatePolicy:   adapters.PolicyAlwaysUpdate,
		Kind:           adapters.KindTool,
		Manager:        ghManagerPacman,
		ManagerPackage: ghPkgPacman,
	}
)

func (a *GhAdapter) Info() adapters.ToolInfo {
	if defaultLinuxManager() == "pacman" {
		return ghInfoPacman
	}
	return ghInfoApt
}
