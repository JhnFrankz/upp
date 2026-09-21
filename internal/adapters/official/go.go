package official

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"runtime"
	"strings"

	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/security"
)

// goBinaryPathFn is the seam for locating the go binary on Linux.
// Swapped in tests via setExecFakes.
var goBinaryPathFn = func() string {
	p, err := exec.LookPath("go")
	if err != nil {
		return ""
	}
	return p
}

// goDevVersionFn is the seam for querying the latest version from go.dev.
// Swapped in tests via setExecFakes.
var goDevVersionFn = fetchGoDevVersion

func fetchGoDevVersion(ctx context.Context) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	client := &http.Client{
		Timeout: adapters.CheckTimeout,
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://go.dev/VERSION?m=text", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "upp")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status %d from go.dev", resp.StatusCode)
	}
	buf := make([]byte, 512)
	n, _ := resp.Body.Read(buf)
	line := strings.TrimSpace(strings.Split(string(buf[:n]), "\n")[0])
	return line, nil
}

// GoAdapter manages Go across platforms.
// Linux: manual binary replace or apt/pacman delegation, macOS: brew, Windows: winget.
type GoAdapter struct{}

func (a *GoAdapter) Name() string { return "go" }

func (a *GoAdapter) Detect() bool {
	return lookPath("go")
}

func (a *GoAdapter) linuxOwner() (adapters.Adapter, string) {
	if runtime.GOOS != "linux" {
		return nil, ""
	}
	bin := goBinaryPathFn()
	if bin == "/usr/bin/go" || strings.HasPrefix(bin, "/usr/bin/") {
		if lookPath("apt") {
			return AdapterByName("apt"), "golang-go"
		}
		if lookPath("pacman") {
			return AdapterByName("pacman"), "go"
		}
	}
	return nil, ""
}

func (a *GoAdapter) Check(ctx context.Context) (adapters.UpdateInfo, error) {
	if !a.Detect() {
		return adapters.UpdateInfo{}, fmt.Errorf("go is not installed")
	}

	// Delegated check path: go is owned by brew on macOS and winget on Windows.
	// On Linux, if go is installed in /usr/bin under apt or pacman, delegate to
	// that package manager; if manual (e.g. /usr/local/go), check go.dev directly.
	platform := runtimeGOOSToPlatform(runtime.GOOS)
	owner := ResolveOwner("go", platform)
	pkg := a.Info().ManagerPackage[platform]
	if owner == nil && runtime.GOOS == "linux" {
		owner, pkg = a.linuxOwner()
	}

	if owner != nil {
		if checker, ok := owner.(adapters.PackageChecker); ok {
			if pkg == "" {
				return adapters.UpdateInfo{}, fmt.Errorf("go has no manager package on %s", runtime.GOOS)
			}
			return checker.CheckPackage(ctx, pkg)
		}
		return adapters.UpdateInfo{}, fmt.Errorf("go's manager %s does not support per-package checks", runtime.GOOS)
	}

	current := commandOutput(ctx, "go", "version")
	current = extractGoVersion(current)

	latest := current
	updateAvailable := false

	if raw, err := goDevVersionFn(ctx); err == nil && raw != "" {
		if v := extractGoVersion(raw); v != "" {
			latest = v
			updateAvailable = current != "" && semverCompare(current, latest)
		}
	}

	return adapters.UpdateInfo{
		CurrentVersion:  current,
		LatestVersion:   latest,
		UpdateAvailable: updateAvailable,
	}, nil
}

func (a *GoAdapter) Update(ctx context.Context, dryRun bool) (adapters.Result, error) {
	if !a.Detect() {
		return adapters.Result{Success: false}, fmt.Errorf("go is not installed")
	}

	platform := runtimeGOOSToPlatform(runtime.GOOS)
	owner := ResolveOwner("go", platform)
	pkg := a.Info().ManagerPackage[platform]
	if owner == nil && runtime.GOOS == "linux" {
		owner, pkg = a.linuxOwner()
	}

	if owner != nil {
		if dryRun {
			return adapters.Result{Success: true}, nil
		}
		if updater, ok := owner.(adapters.PackageUpdater); ok {
			if pkg == "" {
				return adapters.Result{Success: false}, fmt.Errorf("go has no manager package on %s", runtime.GOOS)
			}
			return updater.UpdatePackage(ctx, pkg)
		}
		return adapters.Result{Success: false}, fmt.Errorf("go's manager %s does not support per-package updates", runtime.GOOS)
	}

	before := extractGoVersion(commandOutput(ctx, "go", "version"))

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
		// Manual binary update: purge existing installation, then download and extract latest from go.dev.
		cmd = "sudo rm -rf /usr/local/go && curl -fsSL " + goTarballURL(runtime.GOARCH) + " | sudo tar -C /usr/local -xzf -"
		privileges = []string{"sudo"}
	default:
		return adapters.Result{
			Success: false,
			Before:  before,
			After:   before,
			Error:   fmt.Errorf("unsupported platform: %s", runtime.GOOS),
		}, nil
	}

	_, stderr, err := runCmd(ctx, cmd)
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

	after := extractGoVersion(commandOutput(ctx, "go", "version"))
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
		Trust:          security.TrustOfficial,
		UpdatePolicy:   adapters.PolicyAlwaysUpdate,
		Kind:           adapters.KindTool,
		Manager:        map[string]string{"macos": "brew", "windows": "winget"},
		ManagerPackage: map[string]string{"macos": "golang", "windows": "GoLang.Go"},
		// Command is the exact string Update() runs on Linux, declared so the
		// plan's RiskCommand and the confirmation gate see what actually
		// executes. On macOS and Windows go is owned (Manager above), so the
		// plan takes the owning manager's command and never reads this field.
		Command: "sudo rm -rf /usr/local/go && curl -fsSL " + goTarballURL(runtime.GOARCH) + " | sudo tar -C /usr/local -xzf -",
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
