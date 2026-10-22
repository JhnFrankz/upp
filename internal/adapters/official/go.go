package official

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/platform"
	"github.com/JhnFrankz/upp/internal/security"
)

// GoAdapter manages Go across platforms.
// Linux: manual binary replace or apt/pacman delegation, macOS: brew, Windows: winget.
type GoAdapter struct{}

func (a *GoAdapter) Name() string { return "go" }

func (a *GoAdapter) Detect() bool {
	return lookPath("go")
}

func (a *GoAdapter) linuxOwner() (adapters.Adapter, string) {
	if runtimeGOOSFn() != "linux" {
		return nil, ""
	}
	bin := goBinaryPathFn()
	if bin == "/usr/bin/go" || strings.HasPrefix(bin, "/usr/bin/") || bin == "/bin/go" || strings.HasPrefix(bin, "/bin/") {
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
		return adapters.UpdateInfo{}, fmt.Errorf("%w: go is not installed", adapters.ErrNotInstalled)
	}

	current := commandOutput(ctx, "go", "version")
	current = extractGoVersion(current)

	plat, _ := platform.NormalizeOS(runtimeGOOSFn())
	if owner := resolveOwnerForAdapter(a, plat); owner != nil {
		return DelegatedCheck(ctx, a, current)
	}

	latest := current
	updateAvailable := false

	if runtimeGOOSFn() == "linux" && !isStandardGoPath(goBinaryPathFn()) {
		return adapters.UpdateInfo{
			CurrentVersion:  current,
			LatestVersion:   current,
			UpdateAvailable: false,
		}, nil
	}

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

// goLinuxUpdateCmd declares the structured privileged swap for manual Go updates on Linux.
const goLinuxUpdateCmd = "sudo mv /usr/local/go /usr/local/go.bak && sudo mv ... /usr/local/go && sudo rm -rf /usr/local/go.bak"

func isStandardGoPath(bin string) bool {
	if bin == "/usr/local/go/bin/go" {
		return true
	}
	if resolved, err := filepath.EvalSymlinks(bin); err == nil && resolved == "/usr/local/go/bin/go" {
		return true
	}
	return false
}

func hasCommandError(stderr string) bool {
	return stderr != "" && (strings.Contains(stderr, "Error") || strings.Contains(stderr, "error") || strings.Contains(stderr, "E:"))
}

func (a *GoAdapter) Update(ctx context.Context, dryRun bool) (adapters.Result, error) {
	if !a.Detect() {
		return adapters.Result{Success: false}, fmt.Errorf("%w: go is not installed", adapters.ErrNotInstalled)
	}

	plat, _ := platform.NormalizeOS(runtimeGOOSFn())
	if owner := resolveOwnerForAdapter(a, plat); owner != nil {
		return DelegatedUpdate(ctx, a, dryRun)
	}

	before := extractGoVersion(commandOutput(ctx, "go", "version"))

	if dryRun {
		return adapters.Result{
			Success: true,
			Before:  before,
			After:   before,
		}, nil
	}

	privileges := []string{"sudo"}

	if runtimeGOOSFn() != "linux" {
		return adapters.Result{
			Success: false,
			Before:  before,
			After:   before,
			Error:   fmt.Errorf("unsupported platform: %s", runtimeGOOSFn()),
		}, nil
	}

	bin := goBinaryPathFn()
	if !isStandardGoPath(bin) {
		return adapters.Result{
			Success:    false,
			Before:     before,
			After:      before,
			Error:      fmt.Errorf("cannot update go: active binary is at %q, but manual update only manages standard /usr/local/go installations", bin),
			Privileges: privileges,
		}, nil
	}

	rel, err := goReleaseFn(ctx, runtimeGOOSFn(), runtime.GOARCH)
	if err != nil {
		return adapters.Result{
			Success:    false,
			Before:     before,
			After:      before,
			Error:      fmt.Errorf("go update failed: %w", err),
			Privileges: privileges,
		}, nil
	}

	tmpDir, err := os.MkdirTemp("", "upp-go-update-*")
	if err != nil {
		return adapters.Result{
			Success:    false,
			Before:     before,
			After:      before,
			Error:      fmt.Errorf("go update failed: %w", err),
			Privileges: privileges,
		}, nil
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	archivePath := filepath.Join(tmpDir, rel.Filename)
	if err := goDownloadAndVerifyFn(ctx, rel, archivePath); err != nil {
		return adapters.Result{
			Success:    false,
			Before:     before,
			After:      before,
			Error:      fmt.Errorf("go update failed: %w", err),
			Privileges: privileges,
		}, nil
	}

	if err := goExtractTarballFn(archivePath, tmpDir); err != nil {
		return adapters.Result{
			Success:    false,
			Before:     before,
			After:      before,
			Error:      fmt.Errorf("go update failed: %w", err),
			Privileges: privileges,
		}, nil
	}

	stagedDir := filepath.Join(tmpDir, "go")
	backedUp := false
	if goTargetExistsFn() {
		_, _, _ = runCmdArgsUpdate(ctx, "sudo", "rm", "-rf", "/usr/local/go.bak")
		_, stderr, err := runCmdArgsUpdate(ctx, "sudo", "mv", "/usr/local/go", "/usr/local/go.bak")
		if err != nil || hasCommandError(stderr) {
			errMsg := err
			if errMsg == nil {
				errMsg = fmt.Errorf("%s", truncate(stderr, 200))
			}
			return adapters.Result{
				Success:    false,
				Before:     before,
				After:      before,
				Error:      fmt.Errorf("go update failed: %w", errMsg),
				Stderr:     stderr,
				Privileges: privileges,
			}, nil
		}
		backedUp = true
	}

	_, stderr, err := runCmdArgsUpdate(ctx, "sudo", "mv", stagedDir, "/usr/local/go")
	if err != nil || hasCommandError(stderr) {
		errMsg := err
		if errMsg == nil {
			errMsg = fmt.Errorf("%s", truncate(stderr, 200))
		}
		if backedUp {
			_, _, _ = runCmdArgsUpdate(ctx, "sudo", "mv", "/usr/local/go.bak", "/usr/local/go")
		}
		return adapters.Result{
			Success:    false,
			Before:     before,
			After:      before,
			Error:      fmt.Errorf("go update failed: %w", errMsg),
			Stderr:     stderr,
			Privileges: privileges,
		}, nil
	}

	if backedUp {
		_, _, _ = runCmdArgsUpdate(ctx, "sudo", "rm", "-rf", "/usr/local/go.bak")
	}

	after := extractGoVersion(commandOutput(ctx, "go", "version"))
	return adapters.Result{
		Success:    true,
		Before:     before,
		After:      after,
		Privileges: privileges,
	}, nil
}

var (
	goManager        = map[string]string{"macos": "brew", "windows": "winget"}
	goManagerPackage = map[string]string{"macos": "golang", "windows": "GoLang.Go"}
	goInfo           = adapters.ToolInfo{
		ID:             "go",
		Name:           "Go",
		Platforms:      platformsLinuxMacOSWindows,
		Trust:          security.TrustOfficial,
		UpdatePolicy:   adapters.PolicyAlwaysUpdate,
		Kind:           adapters.KindTool,
		Manager:        goManager,
		ManagerPackage: goManagerPackage,
		// Command is the exact string Update() runs on Linux, declared so the
		// plan's RiskCommand and the confirmation gate see what actually
		// executes. On macOS and Windows go is owned (Manager above), so the
		// plan takes the owning manager's command and never reads this field.
		Command: goLinuxUpdateCmd,
	}
)

func (a *GoAdapter) Info() adapters.ToolInfo {
	if runtimeGOOSFn() == "linux" {
		if owner, pkg := a.linuxOwner(); owner != nil {
			info := goInfo
			mgrMap := make(map[string]string, len(goManager)+1)
			for k, v := range goManager {
				mgrMap[k] = v
			}
			mgrMap["linux"] = owner.Name()
			info.Manager = mgrMap

			pkgMap := make(map[string]string, len(goManagerPackage)+1)
			for k, v := range goManagerPackage {
				pkgMap[k] = v
			}
			pkgMap["linux"] = pkg
			info.ManagerPackage = pkgMap

			info.UpdatePolicy = owner.Info().UpdatePolicy
			info.Command = ""
			return info
		}
	}
	return goInfo
}

// extractGoVersion extracts the version from "go version go1.22.0 linux/amd64".
func extractGoVersion(output string) string {
	// Format: "go version go1.22.0 linux/amd64"
	rem := output
	for {
		var field string
		field, rem = nextField(rem)
		if field == "" {
			break
		}
		cleaned := strings.Trim(field, "(),:;[]\"'")
		if strings.HasPrefix(cleaned, "go") && len(cleaned) > 2 && cleaned[2] >= '0' && cleaned[2] <= '9' && isVersionLike(cleaned[2:]) {
			return cleaned[2:] // strip "go" prefix
		}
	}
	return extractVersion(output)
}
