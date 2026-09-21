package official

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/security"
)

var (
	_ adapters.Adapter        = (*PacmanAdapter)(nil)
	_ adapters.PackageChecker = (*PacmanAdapter)(nil)
	_ adapters.PackageUpdater = (*PacmanAdapter)(nil)
)

// PacmanAdapter manages Arch Linux packages.
type PacmanAdapter struct{}

// pacmanSelfUpdateCmd is pacman's real self-update command — declared by
// Info() and executed by Update() (design D2 single source of truth).
const pacmanSelfUpdateCmd = "sudo pacman -S --noconfirm pacman"

// pacmanPackageUpdateTemplate is pacman's per-package update command
// template; the PackagePlaceholder is rendered per owned package by
// UpdatePackage().
const pacmanPackageUpdateTemplate = "sudo pacman -S --noconfirm <pkg>"

func (a *PacmanAdapter) Name() string { return "pacman" }

func (a *PacmanAdapter) Detect() bool {
	return lookPath("pacman")
}

func (a *PacmanAdapter) Info() adapters.ToolInfo {
	return adapters.ToolInfo{
		ID:                   "pacman",
		Name:                 "Pacman Package Manager",
		Platforms:            []string{"linux"},
		Trust:                security.TrustOfficial,
		UpdatePolicy:         adapters.PolicyGated,
		Kind:                 adapters.KindManager,
		Privileges:           []string{"sudo"},
		SelfUpdateCommand:    pacmanSelfUpdateCmd,
		PackageUpdateCommand: pacmanPackageUpdateTemplate,
	}
}

// Check queries for updates to pacman itself.
func (a *PacmanAdapter) Check(ctx context.Context) (adapters.UpdateInfo, error) {
	return a.CheckPackage(ctx, "pacman")
}

// Update updates pacman itself.
func (a *PacmanAdapter) Update(ctx context.Context, dryRun bool) (adapters.Result, error) {
	if !a.Detect() {
		return adapters.Result{Success: false}, fmt.Errorf("pacman is not installed")
	}

	before, _ := a.CurrentVersion(ctx)
	if dryRun {
		return adapters.Result{
			Success: true,
			Before:  before,
			After:   before,
		}, nil
	}

	_, stderr, err := runCmdArgsUpdate(ctx, "sudo", "pacman", "-S", "--noconfirm", "pacman")
	if err != nil {
		return adapters.Result{
			Success:    false,
			Before:     before,
			After:      before,
			Error:      fmt.Errorf("pacman update failed: %w", err),
			Privileges: []string{"sudo"},
		}, nil
	}

	if stderr != "" && strings.Contains(stderr, "error:") {
		return adapters.Result{
			Success:    false,
			Before:     before,
			After:      before,
			Error:      fmt.Errorf("pacman update error: %s", truncate(stderr, 200)),
			Privileges: []string{"sudo"},
		}, nil
	}

	after, _ := a.CurrentVersion(ctx)
	return adapters.Result{
		Success:    true,
		Before:     before,
		After:      after,
		Privileges: []string{"sudo"},
	}, nil
}

// parsePacmanQOutput extracts the package version from `pacman -Q <pkg>` output.
func parsePacmanQOutput(out string) string {
	fields := strings.Fields(out)
	if len(fields) >= 2 {
		return fields[1]
	}
	if len(fields) == 1 {
		return fields[0]
	}
	return "unknown"
}

// parsePacmanSiOutput extracts the candidate version from `pacman -Si <pkg>` output.
func parsePacmanSiOutput(out string) string {
	for _, line := range strings.Split(out, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "Version") {
			parts := strings.SplitN(trimmed, ":", 2)
			if len(parts) == 2 {
				val := strings.TrimSpace(parts[1])
				if val != "" {
					return val
				}
			}
		}
	}
	return "unknown"
}

// CheckPackage reports the installed vs candidate version of a package under pacman.
func (a *PacmanAdapter) CheckPackage(ctx context.Context, pkg string) (adapters.UpdateInfo, error) {
	if !a.Detect() {
		return adapters.UpdateInfo{}, fmt.Errorf("pacman is not installed")
	}

	qOut := commandOutput(ctx, "pacman", "-Q", pkg)
	current := parsePacmanQOutput(qOut)

	siOut, err := commandOutputErr(ctx, "pacman", "-Si", pkg)
	if err != nil {
		return adapters.UpdateInfo{}, err
	}
	latest := parsePacmanSiOutput(siOut)

	updateAvailable := false
	if current != "unknown" && latest != "unknown" {
		updateAvailable = a.compareVersions(ctx, current, latest)
	}

	return adapters.UpdateInfo{
		CurrentVersion:  current,
		LatestVersion:   latest,
		UpdateAvailable: updateAvailable,
	}, nil
}

// compareVersions compares current and latest versions using vercmp if available,
// falling back to string inequality.
func (a *PacmanAdapter) compareVersions(ctx context.Context, current, latest string) bool {
	if current == latest {
		return false
	}
	if lookPath("vercmp") {
		out := strings.TrimSpace(commandOutput(ctx, "vercmp", latest, current))
		if n, err := strconv.Atoi(out); err == nil {
			return n > 0
		}
	}
	return current != latest
}

// CurrentVersion returns the currently installed pacman version.
func (a *PacmanAdapter) CurrentVersion(ctx context.Context) (string, error) {
	out := commandOutput(ctx, "pacman", "-Q", "pacman")
	v := parsePacmanQOutput(out)
	return v, nil
}

// UpdatePackage updates a single package using pacman.
func (a *PacmanAdapter) UpdatePackage(ctx context.Context, pkg string) (adapters.Result, error) {
	if !a.Detect() {
		return adapters.Result{Success: false}, fmt.Errorf("pacman is not installed")
	}

	before, _ := a.CurrentVersion(ctx)
	_, stderr, err := runCmdArgsUpdate(ctx, "sudo", "pacman", "-S", "--noconfirm", pkg)
	if err != nil {
		return adapters.Result{
			Success:    false,
			Before:     before,
			After:      before,
			Error:      fmt.Errorf("pacman update failed: %w", err),
			Privileges: []string{"sudo"},
		}, nil
	}

	if stderr != "" && strings.Contains(stderr, "error:") {
		return adapters.Result{
			Success:    false,
			Before:     before,
			After:      before,
			Error:      fmt.Errorf("pacman update error: %s", truncate(stderr, 200)),
			Privileges: []string{"sudo"},
		}, nil
	}

	after, _ := a.CurrentVersion(ctx)
	return adapters.Result{
		Success:    true,
		Before:     before,
		After:      after,
		Privileges: []string{"sudo"},
	}, nil
}
