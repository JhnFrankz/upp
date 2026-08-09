package official

import (
	"fmt"
	"runtime"
	"strings"

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

	current := commandOutput("docker", "--version")
	current = extractVersion(current)

	return adapters.UpdateInfo{
		CurrentVersion:  current,
		LatestVersion:   current,
		UpdateAvailable: false,
	}, nil
}

func (a *DockerAdapter) Update(dryRun bool) (adapters.Result, error) {
	if !a.Detect() {
		return adapters.Result{Success: false}, fmt.Errorf("docker is not installed")
	}

	before := extractVersion(commandOutput("docker", "--version"))

	if dryRun {
		return adapters.Result{
			Success: true,
			Before:  before,
			After:   before,
		}, nil
	}

	var cmd string
	var privileges []string

	switch runtime.GOOS {
	case "linux":
		cmd = "sudo apt update -qq && sudo apt upgrade -y docker-ce docker-ce-cli containerd.io"
		privileges = []string{"sudo"}
	case "darwin":
		cmd = "brew upgrade docker"
	case "windows":
		cmd = "winget upgrade Docker.DockerDesktop --accept-source-agreements --accept-package-agreements"
	default:
		return adapters.Result{
			Success: false,
			Before:  before,
			After:   before,
			Error:   fmt.Errorf("unsupported platform: %s", runtime.GOOS),
		}, nil
	}

	_, stderr, err := runCmd(cmd)
	if err != nil {
		return adapters.Result{
			Success:    false,
			Before:     before,
			After:      before,
			Error:      fmt.Errorf("docker update failed: %w", err),
			Privileges: privileges,
		}, nil
	}

	if stderr != "" && (strings.Contains(stderr, "Error") || strings.Contains(stderr, "error") || strings.Contains(stderr, "E:")) {
		return adapters.Result{
			Success:    false,
			Before:     before,
			After:      before,
			Error:      fmt.Errorf("docker update error: %s", truncate(stderr, 200)),
			Privileges: privileges,
		}, nil
	}

	after := extractVersion(commandOutput("docker", "--version"))
	return adapters.Result{
		Success:    true,
		Before:     before,
		After:      after,
		Privileges: privileges,
	}, nil
}

func (a *DockerAdapter) Info() adapters.ToolInfo {
	return adapters.ToolInfo{
		ID:        "docker",
		Name:      "Docker",
		Platforms: []string{"linux", "macos", "windows"},
		Trust:     adapters.TrustTrusted,
	}
}
