package official

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/platform"
	"github.com/JhnFrankz/upp/internal/security"
)

// GoRelease represents the metadata of an official Go release archive.
type GoRelease struct {
	Version  string
	Filename string
	SHA256   string
	URL      string
}

type goDevFile struct {
	Filename string `json:"filename"`
	OS       string `json:"os"`
	Arch     string `json:"arch"`
	Version  string `json:"version"`
	SHA256   string `json:"sha256"`
	Size     int64  `json:"size"`
	Kind     string `json:"kind"`
}

type goDevRelease struct {
	Version string      `json:"version"`
	Stable  bool        `json:"stable"`
	Files   []goDevFile `json:"files"`
}

// goReleaseURL is the endpoint for Go release metadata.
var goReleaseURL = "https://go.dev/dl/?mode=json"

// goReleaseFn is the seam for fetching Go release metadata.
// Swapped in tests via setExecFakes.
var goReleaseFn = fetchGoRelease

// goDownloadAndVerifyFn is the seam for downloading and verifying a Go archive.
// Swapped in tests via setExecFakes.
var goDownloadAndVerifyFn = downloadAndVerifyGo

// goExtractTarballFn is the seam for extracting a Go archive.
// Swapped in tests via setExecFakes.
var goExtractTarballFn = extractGoTarball

// goTargetExistsFn is the seam for checking if /usr/local/go exists before backing up.
// Swapped in tests via setExecFakes.
var goTargetExistsFn = func() bool {
	_, err := os.Stat("/usr/local/go")
	return err == nil
}

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

func fetchGoRelease(ctx context.Context, goos, goarch string) (GoRelease, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	client := &http.Client{
		Timeout: adapters.CheckTimeout,
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, goReleaseURL, nil)
	if err != nil {
		return GoRelease{}, err
	}
	req.Header.Set("User-Agent", "upp")
	resp, err := client.Do(req)
	if err != nil {
		return GoRelease{}, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return GoRelease{}, fmt.Errorf("unexpected status %d from go.dev", resp.StatusCode)
	}

	var releases []goDevRelease
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return GoRelease{}, fmt.Errorf("failed to decode go releases: %w", err)
	}

	for _, r := range releases {
		if !r.Stable {
			continue
		}
		for _, f := range r.Files {
			if f.OS == goos && f.Arch == goarch && f.Kind == "archive" {
				return GoRelease{
					Version:  r.Version,
					Filename: f.Filename,
					SHA256:   f.SHA256,
					URL:      "https://go.dev/dl/" + f.Filename,
				}, nil
			}
		}
	}

	return GoRelease{}, fmt.Errorf("no stable Go release found for %s/%s", goos, goarch)
}

func downloadAndVerifyGo(ctx context.Context, rel GoRelease, destPath string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	client := &http.Client{
		Timeout: adapters.UpdateTimeout,
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rel.URL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "upp")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s failed: HTTP %d", rel.URL, resp.StatusCode)
	}

	f, err := os.Create(destPath)
	if err != nil {
		return err
	}

	hasher := sha256.New()
	mw := io.MultiWriter(f, hasher)

	_, copyErr := io.Copy(mw, resp.Body)
	closeErr := f.Close()

	if copyErr != nil {
		_ = os.Remove(destPath)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(destPath)
		return closeErr
	}

	actualHash := fmt.Sprintf("%x", hasher.Sum(nil))
	if !strings.EqualFold(actualHash, rel.SHA256) {
		_ = os.Remove(destPath)
		return fmt.Errorf("checksum mismatch for %s: got %s, want %s", rel.Filename, actualHash, rel.SHA256)
	}

	return nil
}

func extractGoTarball(archivePath, destDir string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer func() { _ = gz.Close() }()

	tr := tar.NewReader(gz)
	cleanDest := filepath.Clean(destDir)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar entry: %w", err)
		}

		cleanName := filepath.Clean(hdr.Name)
		if strings.HasPrefix(cleanName, "..") || filepath.IsAbs(cleanName) {
			return fmt.Errorf("illegal archive entry path: %s", hdr.Name)
		}

		targetPath := filepath.Join(cleanDest, cleanName)
		if !strings.HasPrefix(targetPath, cleanDest+string(filepath.Separator)) && targetPath != cleanDest {
			return fmt.Errorf("archive entry escapes destination: %s", hdr.Name)
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return err
			}
			mode := hdr.FileInfo().Mode().Perm()
			if mode == 0 {
				mode = 0644
			}
			outFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
			if err != nil {
				return err
			}
			if _, err := io.Copy(outFile, tr); err != nil {
				_ = outFile.Close()
				return err
			}
			if err := outFile.Close(); err != nil {
				return err
			}
		case tar.TypeSymlink:
			if strings.HasPrefix(hdr.Linkname, "/") || strings.Contains(hdr.Linkname, "..") {
				return fmt.Errorf("unsupported or unsafe symlink target: %s", hdr.Linkname)
			}
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return err
			}
			_ = os.Remove(targetPath)
			if err := os.Symlink(hdr.Linkname, targetPath); err != nil {
				return err
			}
		default:
			// Ignore other header types
		}
	}
	return nil
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
	plat, _ := platform.NormalizeOS(runtime.GOOS)
	owner := ResolveOwner("go", plat)
	pkg := a.Info().ManagerPackage[plat]
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
		return adapters.Result{Success: false}, fmt.Errorf("go is not installed")
	}

	plat, _ := platform.NormalizeOS(runtime.GOOS)
	owner := ResolveOwner("go", plat)
	pkg := a.Info().ManagerPackage[plat]
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

	privileges := []string{"sudo"}

	if runtime.GOOS != "linux" {
		return adapters.Result{
			Success: false,
			Before:  before,
			After:   before,
			Error:   fmt.Errorf("unsupported platform: %s", runtime.GOOS),
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

	rel, err := goReleaseFn(ctx, runtime.GOOS, runtime.GOARCH)
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
		Command: goLinuxUpdateCmd,
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
