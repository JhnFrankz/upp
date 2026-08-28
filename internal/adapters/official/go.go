package official

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/JhnFrankz/upp/internal/adapters"
)

// GoAdapter manages Go across platforms.
// Linux: manual binary replace, macOS: brew, Windows: winget.
type GoAdapter struct{}

func (a *GoAdapter) Name() string { return "go" }

func (a *GoAdapter) Detect() bool {
	return lookPath("go")
}

func (a *GoAdapter) Check() (adapters.UpdateInfo, error) {
	if !a.Detect() {
		return adapters.UpdateInfo{}, fmt.Errorf("go is not installed")
	}

	// Delegated check path (WU2, spec Per-Owned-Tool Availability): go is
	// owned by brew on macOS and winget on Windows, so its Check() reports the
	// real update of its package under the resolving manager (e.g.
	// `brew outdated --json golang`, `winget upgrade GoLang.Go`) there. On
	// Linux go has NO resolving owner (standalone manual binary replace), so
	// ResolveOwner returns nil and the standalone `go version` path below runs
	// — matching its Update() branch. runtime.GOOS is translated to the
	// platform key because the manager/package maps are keyed by PLATFORM
	// constants, not runtime.GOOS (darwin).
	platform := runtimeGOOSToPlatform(runtime.GOOS)
	if owner := ResolveOwner("go", platform); owner != nil {
		if checker, ok := owner.(adapters.PackageChecker); ok {
			pkg := a.Info().ManagerPackage[platform]
			if pkg == "" {
				return adapters.UpdateInfo{}, fmt.Errorf("go has no manager package on %s", runtime.GOOS)
			}
			return checker.CheckPackage(pkg)
		}
		return adapters.UpdateInfo{}, fmt.Errorf("go's manager %s does not support per-package checks", runtime.GOOS)
	}

	current := commandOutput("go", "version")
	current = extractGoVersion(current)

	return adapters.UpdateInfo{
		CurrentVersion:  current,
		LatestVersion:   current,
		UpdateAvailable: false,
	}, nil
}

func (a *GoAdapter) Update(dryRun bool) (adapters.Result, error) {
	if !a.Detect() {
		return adapters.Result{Success: false}, fmt.Errorf("go is not installed")
	}

	// Delegated update path (WU2, spec Official Adapter Catalog / Resolved
	// Owner Update Delegation): go is owned by brew on macOS and winget on
	// Windows, so it delegates to the resolving manager there. On Linux go has
	// NO resolving owner (standalone manual binary replace), so ResolveOwner
	// returns nil and the standalone path below runs. runtime.GOOS is
	// translated to the platform key because ResolveOwner is keyed by PLATFORM
	// constants, not runtime.GOOS (darwin).
	if owner := ResolveOwner("go", runtimeGOOSToPlatform(runtime.GOOS)); owner != nil {
		return owner.Update(dryRun)
	}

	before := extractGoVersion(commandOutput("go", "version"))

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
		// Manual binary update: download latest from go.dev.
		cmd = "curl -fsSL " + goTarballURL(runtime.GOARCH) + " | sudo tar -C /usr/local -xzf -"
		privileges = []string{"sudo"}
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
			Error:      fmt.Errorf("go update failed: %w", err),
			Privileges: privileges,
		}, nil
	}

	if stderr != "" && (strings.Contains(stderr, "Error") || strings.Contains(stderr, "error") || strings.Contains(stderr, "E:")) {
		return adapters.Result{
			Success:    false,
			Before:     before,
			After:      before,
			Error:      fmt.Errorf("go update error: %s", truncate(stderr, 200)),
			Privileges: privileges,
		}, nil
	}

	after := extractGoVersion(commandOutput("go", "version"))
	return adapters.Result{
		Success:    true,
		Before:     before,
		After:      after,
		Privileges: privileges,
	}, nil
}

func (a *GoAdapter) Info() adapters.ToolInfo {
	return adapters.ToolInfo{
		ID:             "go",
		Name:           "Go",
		Platforms:      []string{"linux", "macos", "windows"},
		Trust:          adapters.TrustOfficial,
		UpdatePolicy:   adapters.PolicyAlwaysUpdate,
		Kind:           adapters.KindTool,
		Manager:        map[string]string{"macos": "brew", "windows": "winget"},
		ManagerPackage: map[string]string{"macos": "golang", "windows": "GoLang.Go"},
	}
}

// goTarballURL returns the go.dev Linux tarball URL for the given
// architecture, so downloads match the running process instead of
// hardcoding amd64.
func goTarballURL(goarch string) string {
	return fmt.Sprintf("https://go.dev/dl/$(curl -fsSL https://go.dev/VERSION?m=text | head -1).linux-%s.tar.gz", goarch)
}

// extractGoVersion extracts the version from "go version go1.22.0 linux/amd64".
func extractGoVersion(output string) string {
	// Format: "go version go1.22.0 linux/amd64"
	fields := strings.Fields(output)
	for _, field := range fields {
		if strings.HasPrefix(field, "go") && len(field) > 2 {
			return field[2:] // strip "go" prefix
		}
	}
	return extractVersion(output)
}
