package official

import (
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
	return adapters.UpdateInfo{}, nil
}

// Update updates pacman itself.
func (a *PacmanAdapter) Update(dryRun bool) (adapters.Result, error) {
	return adapters.Result{}, nil
}

// CheckPackage reports the installed vs candidate version of a package under pacman.
func (a *PacmanAdapter) CheckPackage(pkg string) (adapters.UpdateInfo, error) {
	return adapters.UpdateInfo{}, nil
}

// UpdatePackage updates a single package using pacman.
func (a *PacmanAdapter) UpdatePackage(pkg string) (adapters.Result, error) {
	return adapters.Result{}, nil
}
