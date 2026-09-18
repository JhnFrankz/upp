package official

import (
	"fmt"
	"net/http"
	"path"
	"strings"

	"github.com/JhnFrankz/upp/internal/adapters"
)

// opencodeLatestTagFn is the seam for fetching the latest OpenCode release tag.
// Swapped in tests via setExecFakes.
var opencodeLatestTagFn = fetchOpenCodeLatestTag

func fetchOpenCodeLatestTag() (string, error) {
	client := &http.Client{
		Timeout: adapters.CheckTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	req, err := http.NewRequest(http.MethodHead, "https://github.com/anomalyco/opencode/releases/latest", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "upp")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusFound ||
		resp.StatusCode == http.StatusMovedPermanently ||
		resp.StatusCode == http.StatusTemporaryRedirect ||
		resp.StatusCode == http.StatusSeeOther {
		loc := resp.Header.Get("Location")
		if loc != "" {
			return path.Base(loc), nil
		}
	}
	return "", fmt.Errorf("unexpected status %d or missing Location header", resp.StatusCode)
}

// OpenCodeAdapter manages OpenCode on all platforms.
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

	latest := current
	updateAvailable := false

	if tag, err := opencodeLatestTagFn(); err == nil && tag != "" {
		if v := extractVersion(tag); v != "" {
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

	cmd := "opencode update"

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
		Command:      "opencode update",
	}
}

// Ensure OpenCodeAdapter implements adapters.Adapter at compile time.
var _ adapters.Adapter = (*OpenCodeAdapter)(nil)
