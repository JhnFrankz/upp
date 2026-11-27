package official

import (
	"context"

	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/security"
)

// GhAdapter manages GitHub CLI across platforms.
// Linux: apt, macOS: brew, Windows: winget.
type GhAdapter struct{}

func (a *GhAdapter) Name() string { return "gh" }

func (a *GhAdapter) Detect() bool {
	return ghDelegated.detect()
}

func (a *GhAdapter) CurrentVersion(ctx context.Context) (string, error) {
	return ghDelegated.currentVersion(ctx)
}

func (a *GhAdapter) Check(ctx context.Context) (adapters.UpdateInfo, error) {
	return ghDelegated.check(ctx, a)
}

func (a *GhAdapter) Update(ctx context.Context, dryRun bool) (adapters.Result, error) {
	return ghDelegated.update(ctx, a, dryRun)
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
	ghDelegated = delegatedToolAdapter{
		id:         "gh",
		aptInfo:    ghInfoApt,
		pacmanInfo: ghInfoPacman,
	}
)

func (a *GhAdapter) Info() adapters.ToolInfo {
	return ghDelegated.info()
}
