package official

import (
	"fmt"
	"strings"

	"github.com/JhnFrankz/upp/internal/adapters"
)

// OpenCodeAdapter manages OpenCode on all platforms via the curl installer.
type OpenCodeAdapter struct{}

func (a *OpenCodeAdapter) Name() string { return "opencode" }

func (a *OpenCodeAdapter) Detect() bool {
	return lookPath("opencode")
}

func (a *OpenCodeAdapter) Check() (adapters.UpdateInfo, error) {
	if !a.Detect() {
		return adapters.UpdateInfo{}, fmt.Errorf("opencode is not installed")
	}

	current := commandOutput("opencode", "--version")
	current = extractVersion(current)

	return adapters.UpdateInfo{
		CurrentVersion:  current,
		LatestVersion:   current,
		UpdateAvailable: false,
	}, nil
}

func (a *OpenCodeAdapter) Update(dryRun bool) (adapters.Result, error) {
	if !a.Detect() {
		return adapters.Result{Success: false}, fmt.Errorf("opencode is not installed")
	}

	before := extractVersion(commandOutput("opencode", "--version"))

	if dryRun {
		return adapters.Result{
			Success: true,
			Before:  before,
			After:   before,
		}, nil
	}

	// OpenCode uses the same installer for updates and installs.
	cmd := "curl -fsSL https://opencode.ai/install | bash"

	_, stderr, err := runCmd(cmd)
	if err != nil {
		return adapters.Result{
			Success: false,
			Before:  before,
			After:   before,
			Error:   fmt.Errorf("opencode update failed: %w", err),
		}, nil
	}

	if stderr != "" && (strings.Contains(stderr, "error") || strings.Contains(stderr, "Error")) {
		return adapters.Result{
			Success: false,
			Before:  before,
			After:   before,
			Error:   fmt.Errorf("opencode update error: %s", truncate(stderr, 200)),
		}, nil
	}

	after := extractVersion(commandOutput("opencode", "--version"))
	return adapters.Result{
		Success: true,
		Before:  before,
		After:   after,
	}, nil
}

func (a *OpenCodeAdapter) Info() adapters.ToolInfo {
	return adapters.ToolInfo{
		ID:           "opencode",
		Name:         "OpenCode",
		Platforms:    []string{"linux", "macos", "windows"},
		Trust:        adapters.TrustOfficial,
		UpdatePolicy: adapters.PolicyAlwaysUpdate,
		Kind:         adapters.KindTool,
	}
}

// Ensure OpenCodeAdapter implements adapters.Adapter at compile time.
var _ adapters.Adapter = (*OpenCodeAdapter)(nil)
