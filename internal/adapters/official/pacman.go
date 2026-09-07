package official

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/JhnFrankz/upp/internal/adapters"
)

var (
	_ adapters.Adapter        = (*PacmanAdapter)(nil)
	_ adapters.PackageChecker = (*PacmanAdapter)(nil)
	_ adapters.PackageUpdater = (*PacmanAdapter)(nil)
)

// PacmanAdapter manages Arch Linux packages.
type PacmanAdapter struct{}

func (a *PacmanAdapter) Name() string { return "pacman" }

func (a *PacmanAdapter) Detect() bool {
	return lookPath("pacman")
}

func (a *PacmanAdapter) Info() adapters.ToolInfo {
	return adapters.ToolInfo{
		ID:           "pacman",
		Name:         "Pacman Package Manager",
		Platforms:    []string{"linux"},
		Trust:        adapters.TrustOfficial,
		UpdatePolicy: adapters.PolicyGated,
		Kind:         adapters.KindManager,
	}
}

// Check queries for updates to pacman itself.
func (a *PacmanAdapter) Check() (adapters.UpdateInfo, error) {
	return a.CheckPackage("pacman")
}

// Update updates pacman itself.
func (a *PacmanAdapter) Update(dryRun bool) (adapters.Result, error) {
	return adapters.Result{}, nil
}

// CheckPackage reports the installed vs candidate version of a package under pacman.
func (a *PacmanAdapter) CheckPackage(pkg string) (adapters.UpdateInfo, error) {
	if !a.Detect() {
		return adapters.UpdateInfo{}, fmt.Errorf("pacman is not installed")
	}

	installedCmd := fmt.Sprintf("bash -o pipefail -c 'pacman -Q %s 2>/dev/null | awk \"{print \\$2}\"'", pkg)
	current := strings.TrimSpace(shellOutput(installedCmd))
	if current == "" {
		current = "unknown"
	}

	candidateCmd := fmt.Sprintf("bash -o pipefail -c 'pacman -Si %s 2>/dev/null | grep -E \"^Version\" | head -1 | awk \"{print \\$3}\"'", pkg)
	stdout, err := shellOutputErr(candidateCmd, "pacman")
	if err != nil {
		return adapters.UpdateInfo{}, err
	}
	latest := strings.TrimSpace(stdout)
	if idx := strings.IndexByte(latest, '\n'); idx != -1 {
		latest = strings.TrimSpace(latest[:idx])
	}
	if latest == "" {
		latest = "unknown"
	}

	updateAvailable := false
	if current != "unknown" && latest != "unknown" {
		updateAvailable = a.compareVersions(current, latest)
	}

	return adapters.UpdateInfo{
		CurrentVersion:  current,
		LatestVersion:   latest,
		UpdateAvailable: updateAvailable,
	}, nil
}

// compareVersions compares current and latest versions using vercmp if available,
// falling back to string inequality.
func (a *PacmanAdapter) compareVersions(current, latest string) bool {
	if current == latest {
		return false
	}
	if lookPath("vercmp") {
		out := strings.TrimSpace(commandOutput("vercmp", latest, current))
		if n, err := strconv.Atoi(out); err == nil {
			return n > 0
		}
	}
	return current != latest
}

// CurrentVersion returns the currently installed pacman version.
func (a *PacmanAdapter) CurrentVersion() (string, error) {
	stdout := shellOutput("bash -o pipefail -c 'pacman -Q pacman 2>/dev/null | awk \"{print \\$2}\"'")
	v := strings.TrimSpace(stdout)
	if v == "" {
		return "unknown", nil
	}
	return v, nil
}

// UpdatePackage updates a single package using pacman.
func (a *PacmanAdapter) UpdatePackage(pkg string) (adapters.Result, error) {
	return adapters.Result{}, nil
}
