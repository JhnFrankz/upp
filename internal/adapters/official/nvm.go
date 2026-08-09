package official

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/JhnFrankz/upp/internal/adapters"
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

	// Get current node version via nvm.
	stdout := shellOutput("source ~/.nvm/nvm.sh 2>/dev/null && nvm current")
	current := strings.TrimSpace(stdout)
	if current == "" {
		current = "unknown"
	}

	// Get latest stable version.
	stdout = shellOutput("source ~/.nvm/nvm.sh 2>/dev/null && nvm ls-remote --lts | tail -1 | awk '{print $1}'")
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

	_, stderr, err := runCmd("source ~/.nvm/nvm.sh 2>/dev/null && nvm install stable")
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
		ID:        "nvm",
		Name:      "Node Version Manager",
		Platforms: []string{"linux", "macos", "windows"},
		Trust:     adapters.TrustTrusted,
	}
}

func (a *NVMAdapter) currentVersion() (string, error) {
	stdout := shellOutput("source ~/.nvm/nvm.sh 2>/dev/null && nvm current")
	v := strings.TrimSpace(stdout)
	if v == "" {
		return "unknown", nil
	}
	return v, nil
}
