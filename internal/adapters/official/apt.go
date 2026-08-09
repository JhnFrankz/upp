package official

import (
	"fmt"
	"strings"

	"github.com/JhnFrankz/upp/internal/adapters"
)

// AptAdapter manages APT packages on Linux.
type AptAdapter struct{}

func (a *AptAdapter) Name() string { return "apt" }

func (a *AptAdapter) Detect() bool {
	return lookPath("apt")
}

func (a *AptAdapter) Check() (adapters.UpdateInfo, error) {
	if !a.Detect() {
		return adapters.UpdateInfo{}, fmt.Errorf("apt is not installed")
	}

	// Get installed version.
	stdout := shellOutput("apt-cache policy apt 2>/dev/null | grep 'Installed:' | awk '{print $2}'")
	current := strings.TrimSpace(stdout)
	if current == "" || current == "(none)" {
		current = "unknown"
	}

	// Get latest version.
	stdout = shellOutput("apt-cache policy apt 2>/dev/null | grep 'Candidate:' | awk '{print $2}'")
	latest := strings.TrimSpace(stdout)
	if latest == "" {
		latest = "unknown"
	}

	updateAvailable := current != "unknown" && latest != "unknown" && current != latest

	return adapters.UpdateInfo{
		CurrentVersion:  current,
		LatestVersion:   latest,
		UpdateAvailable: updateAvailable,
	}, nil
}

func (a *AptAdapter) Update(dryRun bool) (adapters.Result, error) {
	if !a.Detect() {
		return adapters.Result{Success: false}, fmt.Errorf("apt is not installed")
	}

	before, _ := a.CurrentVersion()

	if dryRun {
		return adapters.Result{
			Success: true,
			Before:  before,
			After:   before,
		}, nil
	}

	_, stderr, err := runCmd("sudo apt update -qq && sudo apt upgrade -y -qq")
	if err != nil {
		return adapters.Result{
			Success:    false,
			Before:     before,
			After:      before,
			Error:      fmt.Errorf("apt upgrade failed: %w", err),
			Privileges: []string{"sudo"},
		}, nil
	}

	if stderr != "" && strings.Contains(stderr, "E:") {
		return adapters.Result{
			Success:    false,
			Before:     before,
			After:      before,
			Error:      fmt.Errorf("apt upgrade error: %s", truncate(stderr, 200)),
			Privileges: []string{"sudo"},
		}, nil
	}

	after, _ := a.CurrentVersion()
	return adapters.Result{
		Success:    true,
		Before:     before,
		After:      after,
		Privileges: []string{"sudo"},
	}, nil
}

func (a *AptAdapter) Info() adapters.ToolInfo {
	return adapters.ToolInfo{
		ID:        "apt",
		Name:      "APT Package Manager",
		Platforms: []string{"linux"},
		Trust:     adapters.TrustTrusted,
	}
}

// CurrentVersion returns the currently installed apt version.
func (a *AptAdapter) CurrentVersion() (string, error) {
	stdout := shellOutput("apt-cache policy apt 2>/dev/null | grep 'Installed:' | awk '{print $2}'")
	v := strings.TrimSpace(stdout)
	if v == "" || v == "(none)" {
		return "unknown", nil
	}
	return v, nil
}

// truncate shortens a string to maxLen characters, appending "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
