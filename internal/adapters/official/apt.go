package official

import (
	"fmt"
	"strings"

	"github.com/JhnFrankz/upp/internal/adapters"
)

// AptAdapter manages APT packages on Linux.
type AptAdapter struct{}

// aptSelfUpdateCmd is apt's real self-update command — declared by Info() and
// executed by Update() (design D2 single source of truth).
const aptSelfUpdateCmd = "sudo apt install --only-upgrade apt"

// aptPackageUpdateTemplate is apt's per-package update command template; the
// PackagePlaceholder is rendered per owned package by UpdatePackage().
const aptPackageUpdateTemplate = "sudo apt install --only-upgrade <pkg>"

func (a *AptAdapter) Name() string { return "apt" }

func (a *AptAdapter) Detect() bool {
	return lookPath("apt")
}

func (a *AptAdapter) Check() (adapters.UpdateInfo, error) {
	if !a.Detect() {
		return adapters.UpdateInfo{}, fmt.Errorf("apt is not installed")
	}
	return a.CheckPackage("apt")
}

// parseAptPolicyOutput parses the stdout of `apt-cache policy <pkg>`.
func parseAptPolicyOutput(out string) (string, string) {
	var current, latest string
	for _, line := range strings.Split(out, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "Installed:") {
			parts := strings.Fields(trimmed)
			if len(parts) >= 2 {
				current = parts[1]
			}
		} else if strings.HasPrefix(trimmed, "Candidate:") {
			parts := strings.Fields(trimmed)
			if len(parts) >= 2 {
				latest = parts[1]
			}
		}
	}
	if current == "" || current == "(none)" {
		current = "unknown"
	}
	if latest == "" {
		latest = "unknown"
	}
	return current, latest
}

// CheckPackage reports the installed vs candidate version of an owned package
// (e.g. `gh`, `docker-ce`) under apt, so an owned tool's delegated Check() and
// the manager-group bulk path know a real update exists (design D2). It
// queries `apt-cache policy <pkg>` directly via structured execution, not
// through bash or awk.
func (a *AptAdapter) CheckPackage(pkg string) (adapters.UpdateInfo, error) {
	stdout, err := commandOutputErrFor("apt", "apt-cache", "policy", pkg)
	if err != nil {
		return adapters.UpdateInfo{}, err
	}
	current, latest := parseAptPolicyOutput(stdout)
	updateAvailable := current != "unknown" && latest != "unknown" && current != latest

	return adapters.UpdateInfo{
		CurrentVersion:  current,
		LatestVersion:   latest,
		UpdateAvailable: updateAvailable,
	}, nil
}

// UpdatePackage runs the per-package update command for an owned tool under
// apt (e.g. `sudo apt install --only-upgrade gh`), via apt's privileged
// (sudo) executor. This is the manager-group bulk path (design D3): it
// upgrades the owned PACKAGE, NOT apt's self-only row. It is the sudo-gated
// mutating counterpart to CheckPackage's read-only availability query.
func (a *AptAdapter) UpdatePackage(pkg string) (adapters.Result, error) {
	if !a.Detect() {
		return adapters.Result{Success: false}, fmt.Errorf("apt is not installed")
	}

	before, _ := a.CurrentVersion()
	_, stderr, err := runCmd(adapters.RenderPackageCommand(aptPackageUpdateTemplate, pkg))
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

	// Self-only: `sudo apt install --only-upgrade apt` upgrades the APT
	// package manager itself, never the packages it manages (a full
	// `apt upgrade` is intentionally avoided). Stays sudo-gated: the row means
	// "apt package stale" (distro-managed, often intentional). Check() stays
	// root-free and reports real Installed vs Candidate availability.
	_, stderr, err := runCmd(aptSelfUpdateCmd)
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
		ID:                   "apt",
		Name:                 "APT Package Manager",
		Platforms:            []string{"linux"},
		Trust:                adapters.TrustOfficial,
		UpdatePolicy:         adapters.PolicyGated,
		Kind:                 adapters.KindManager,
		SelfUpdateCommand:    aptSelfUpdateCmd,
		PackageUpdateCommand: aptPackageUpdateTemplate,
	}
}

// CurrentVersion returns the currently installed apt version.
func (a *AptAdapter) CurrentVersion() (string, error) {
	stdout := shellOutput("bash -o pipefail -c 'apt-cache policy apt 2>/dev/null | grep \"Installed:\" | awk \"{print \\$2}\"'")
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
