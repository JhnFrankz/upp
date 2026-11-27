package official

import (
	"context"

	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/security"
)

// DockerAdapter manages Docker across platforms.
// Linux: apt, macOS: brew, Windows: winget.
type DockerAdapter struct{}

func (a *DockerAdapter) Name() string { return "docker" }

func (a *DockerAdapter) Detect() bool {
	return dockerDelegated.detect()
}

func (a *DockerAdapter) CurrentVersion(ctx context.Context) (string, error) {
	return dockerDelegated.currentVersion(ctx)
}

func (a *DockerAdapter) Check(ctx context.Context) (adapters.UpdateInfo, error) {
	return dockerDelegated.check(ctx, a)
}

func (a *DockerAdapter) Update(ctx context.Context, dryRun bool) (adapters.Result, error) {
	return dockerDelegated.update(ctx, a, dryRun)
}

var (
	dockerManagerApt    = map[string]string{"linux": "apt", "macos": "brew", "windows": "winget"}
	dockerManagerPacman = map[string]string{"linux": "pacman", "macos": "brew", "windows": "winget"}
	dockerPkgApt        = map[string]string{"linux": "docker-ce", "macos": "docker", "windows": "Docker.Docker"}
	dockerPkgPacman     = map[string]string{"linux": "docker", "macos": "docker", "windows": "Docker.Docker"}

	dockerInfoApt = adapters.ToolInfo{
		ID:             "docker",
		Name:           "Docker",
		Platforms:      platformsLinuxMacOSWindows,
		Trust:          security.TrustOfficial,
		UpdatePolicy:   adapters.PolicyAlwaysUpdate,
		Kind:           adapters.KindTool,
		Manager:        dockerManagerApt,
		ManagerPackage: dockerPkgApt,
	}
	dockerInfoPacman = adapters.ToolInfo{
		ID:             "docker",
		Name:           "Docker",
		Platforms:      platformsLinuxMacOSWindows,
		Trust:          security.TrustOfficial,
		UpdatePolicy:   adapters.PolicyAlwaysUpdate,
		Kind:           adapters.KindTool,
		Manager:        dockerManagerPacman,
		ManagerPackage: dockerPkgPacman,
	}
	dockerDelegated = delegatedToolAdapter{
		id:         "docker",
		aptInfo:    dockerInfoApt,
		pacmanInfo: dockerInfoPacman,
	}
)

func (a *DockerAdapter) Info() adapters.ToolInfo {
	return dockerDelegated.info()
}
