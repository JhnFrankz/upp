package official

import (
	"context"
	"fmt"

	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/security"
)

// DockerAdapter manages Docker across platforms.
// Linux: apt, macOS: brew, Windows: winget.
type DockerAdapter struct{}

func (a *DockerAdapter) Name() string { return "docker" }

func (a *DockerAdapter) Detect() bool {
	return lookPath("docker")
}

func (a *DockerAdapter) Check(ctx context.Context) (adapters.UpdateInfo, error) {
	if !a.Detect() {
		return adapters.UpdateInfo{}, fmt.Errorf("%w: docker is not installed", adapters.ErrNotInstalled)
	}

	current := extractVersion(commandOutput(ctx, "docker", "--version"))
	return DelegatedCheck(ctx, a, current)
}

func (a *DockerAdapter) Update(ctx context.Context, dryRun bool) (adapters.Result, error) {
	if !a.Detect() {
		return adapters.Result{Success: false}, fmt.Errorf("%w: docker is not installed", adapters.ErrNotInstalled)
	}

	return DelegatedUpdate(ctx, a, dryRun)
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
)

func (a *DockerAdapter) Info() adapters.ToolInfo {
	if defaultLinuxManager() == "pacman" {
		return dockerInfoPacman
	}
	return dockerInfoApt
}
