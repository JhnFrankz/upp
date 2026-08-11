package official

import (
	"fmt"
	"runtime"
	"strings"

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

	current := commandOutput("gh", "--version")
	current = extractVersion(current)

	return adapters.UpdateInfo{
		CurrentVersion:  current,
		LatestVersion:   current,
		UpdateAvailable: false,
	}, nil
}

func (a *GhAdapter) Update(dryRun bool) (adapters.Result, error) {
	if !a.Detect() {
		return adapters.Result{Success: false}, fmt.Errorf("gh is not installed")
	}

	before := extractVersion(commandOutput("gh", "--version"))

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
		cmd = "sudo apt update -qq && sudo apt install -y gh"
		privileges = []string{"sudo"}
	case "darwin":
		cmd = "brew upgrade gh"
	case "windows":
		cmd = "winget upgrade gh --accept-source-agreements --accept-package-agreements"
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
			Error:      fmt.Errorf("gh update failed: %w", err),
			Privileges: privileges,
		}, nil
	}

	if stderr != "" && (strings.Contains(stderr, "Error") || strings.Contains(stderr, "error") || strings.Contains(stderr, "E:")) {
		return adapters.Result{
			Success:    false,
			Before:     before,
			After:      before,
			Error:      fmt.Errorf("gh update error: %s", truncate(stderr, 200)),
			Privileges: privileges,
		}, nil
	}

	after := extractVersion(commandOutput("gh", "--version"))
	return adapters.Result{
		Success:    true,
		Before:     before,
		After:      after,
		Privileges: privileges,
	}, nil
}

func (a *GhAdapter) Info() adapters.ToolInfo {
	return adapters.ToolInfo{
		ID:        "gh",
		Name:      "GitHub CLI",
		Platforms: []string{"linux", "macos", "windows"},
		Trust:     adapters.TrustOfficial,
	}
}
