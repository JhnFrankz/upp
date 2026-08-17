package official

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/selfupdate"
)

// NVMAdapter manages Node Version Manager on all platforms.
type NVMAdapter struct{}

func (a *NVMAdapter) Name() string { return "nvm" }

func (a *NVMAdapter) Detect() bool {
	// nvm is a shell function, not a binary — exec.LookPath won't find it.
	// Check for NVM_DIR env var or the nvm directory in ~/.nvm.
	if dir := os.Getenv("NVM_DIR"); dir != "" {
		if _, err := os.Stat(filepath.Join(dir, "nvm.sh")); err == nil {
			return true
		}
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	nvmDir := filepath.Join(home, ".nvm")
	if _, err := os.Stat(filepath.Join(nvmDir, "nvm.sh")); err == nil {
		return true
	}

	return false
}

func (a *NVMAdapter) Check() (adapters.UpdateInfo, error) {
	if !a.Detect() {
		return adapters.UpdateInfo{}, fmt.Errorf("nvm is not installed")
	}

	// Get current node version via nvm. nvm is bash-only, so run through
	// bash explicitly (POSIX sh — dash/ash — lacks `source`); honour
	// NVM_DIR installs with a fallback to ~/.nvm.
	stdout, err := shellOutputErr("bash -c 'source \"${NVM_DIR:-$HOME/.nvm}/nvm.sh\" >/dev/null 2>&1 && nvm current'", "nvm")
	if err != nil {
		return adapters.UpdateInfo{}, err
	}
	current := strings.TrimSpace(stdout)
	if current == "" {
		current = "unknown"
	}

	// Get latest stable version. pipefail makes a failed nvm ls-remote
	// surface as an error instead of an empty awk result. On multi-LTS
	// setups nvm appends an alias line ("lts/* -> lts/jod (-> v22.3.0)")
	// after the real versions; grep filters to actual vX.Y.Z lines so the
	// last one is the newest LTS — tail -1 alone would pick the alias and
	// silently miss real updates.
	stdout, err = shellOutputErr("bash -o pipefail -c 'source \"${NVM_DIR:-$HOME/.nvm}/nvm.sh\" >/dev/null 2>&1 && nvm ls-remote --lts | grep -E \"^[[:space:]]*v[0-9]\" | tail -1 | awk \"{print \\$1}\"'", "nvm")
	if err != nil {
		return adapters.UpdateInfo{}, err
	}
	latest := strings.TrimSpace(stdout)
	if latest == "" {
		latest = "unknown"
	}

	updateAvailable := current != "unknown" && latest != "unknown" && semverCompare(current, latest)

	return adapters.UpdateInfo{
		CurrentVersion:  current,
		LatestVersion:   latest,
		UpdateAvailable: updateAvailable,
	}, nil
}

// semverCompare reports whether an update is available from cur to latest
// using semantic version comparison (D7). A leading "v" prefix on either
// version is tolerated; both sides must parse as versions — a "dev" build
// or any unparseable value (e.g. the "stable" alias) fails closed to false
// with no error, so a newer current never reports a phantom downgrade and
// unknown versions never claim an update based on string inequality.
func semverCompare(cur, latest string) bool {
	c, err := selfupdate.Parse(normalizeVersion(cur))
	if err != nil || c.Dev {
		return false
	}
	l, err := selfupdate.Parse(normalizeVersion(latest))
	if err != nil || l.Dev {
		return false
	}
	return c.Compare(l) < 0
}

// normalizeVersion adds the "v" prefix selfupdate.Parse requires, tolerating
// raw semver strings like "20.11.0" (nvm outputs both shapes).
func normalizeVersion(s string) string {
	if strings.HasPrefix(s, "v") {
		return s
	}
	return "v" + s
}

func (a *NVMAdapter) Update(dryRun bool) (adapters.Result, error) {
	if !a.Detect() {
		return adapters.Result{Success: false}, fmt.Errorf("nvm is not installed")
	}

	before, _ := a.currentVersion()

	if dryRun {
		return adapters.Result{
			Success: true,
			Before:  before,
			After:   before,
		}, nil
	}

	_, stderr, err := runCmd("bash -c 'source \"${NVM_DIR:-$HOME/.nvm}/nvm.sh\" >/dev/null 2>&1 && nvm install stable'")
	if err != nil {
		return adapters.Result{
			Success: false,
			Before:  before,
			After:   before,
			Error:   fmt.Errorf("nvm install stable failed: %w", err),
		}, nil
	}

	if stderr != "" && strings.Contains(stderr, "error") {
		return adapters.Result{
			Success: false,
			Before:  before,
			After:   before,
			Error:   fmt.Errorf("nvm install error: %s", truncate(stderr, 200)),
		}, nil
	}

	after, _ := a.currentVersion()
	return adapters.Result{
		Success: true,
		Before:  before,
		After:   after,
	}, nil
}

func (a *NVMAdapter) Info() adapters.ToolInfo {
	return adapters.ToolInfo{
		ID:           "nvm",
		Name:         "Node Version Manager",
		Platforms:    []string{"linux", "macos", "windows"},
		Trust:        adapters.TrustOfficial,
		UpdatePolicy: adapters.PolicyGated,
	}
}

func (a *NVMAdapter) currentVersion() (string, error) {
	stdout := shellOutput("bash -c 'source \"${NVM_DIR:-$HOME/.nvm}/nvm.sh\" >/dev/null 2>&1 && nvm current'")
	v := strings.TrimSpace(stdout)
	if v == "" {
		return "unknown", nil
	}
	return v, nil
}
