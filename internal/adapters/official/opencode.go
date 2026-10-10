package official

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"

	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/security"
)

// defaultOpenCodeHTTPClient is the shared HTTP client used for checking OpenCode release redirects.
var defaultOpenCodeHTTPClient = &http.Client{
	Timeout: adapters.CheckTimeout,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

// opencodeLatestTagFn is the seam for fetching the latest OpenCode release tag.
// Swapped in tests via setExecFakes.
var opencodeLatestTagFn = fetchOpenCodeLatestTag

func fetchOpenCodeLatestTag(ctx context.Context) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, "https://github.com/anomalyco/opencode/releases/latest", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "upp")
	resp, err := defaultOpenCodeHTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1024))
		_ = resp.Body.Close()
	}()

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

func (a *OpenCodeAdapter) Check(ctx context.Context) (adapters.UpdateInfo, error) {
	if !a.Detect() {
		return adapters.UpdateInfo{}, fmt.Errorf("opencode is not installed")
	}

	current := commandOutput(ctx, "opencode", "--version")
	current = extractVersion(current)

	latest := current
	updateAvailable := false

	if tag, err := opencodeLatestTagFn(ctx); err == nil && tag != "" {
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

func (a *OpenCodeAdapter) Update(ctx context.Context, dryRun bool) (adapters.Result, error) {
	if !a.Detect() {
		return adapters.Result{Success: false}, fmt.Errorf("opencode is not installed")
	}

	before := extractVersion(commandOutput(ctx, "opencode", "--version"))

	if dryRun {
		return adapters.Result{
			Success: true,
			Before:  before,
			After:   before,
		}, nil
	}

	_, stderr, err := runCmdArgsUpdate(ctx, "opencode", "update")
	if err != nil {
		return adapters.Result{
			Success: false,
			Before:  before,
			After:   before,
			Error:   fmt.Errorf("opencode update failed: %w", err),
			Stderr:  stderr,
		}, nil
	}

	if stderr != "" && (strings.Contains(stderr, "error") || strings.Contains(stderr, "Error")) {
		return adapters.Result{
			Success: false,
			Before:  before,
			After:   before,
			Error:   fmt.Errorf("opencode update error: %s", truncate(stderr, 200)),
			Stderr:  stderr,
		}, nil
	}

	after := extractVersion(commandOutput(ctx, "opencode", "--version"))
	return adapters.Result{
		Success: true,
		Before:  before,
		After:   after,
	}, nil
}

var opencodeInfo = adapters.ToolInfo{
	ID:           "opencode",
	Name:         "OpenCode",
	Platforms:    platformsLinuxMacOSWindows,
	Trust:        security.TrustOfficial,
	UpdatePolicy: adapters.PolicyAlwaysUpdate,
	Kind:         adapters.KindTool,
	Command:      "opencode update",
}

func (a *OpenCodeAdapter) Info() adapters.ToolInfo {
	return opencodeInfo
}

// Ensure OpenCodeAdapter implements adapters.Adapter at compile time.
var _ adapters.Adapter = (*OpenCodeAdapter)(nil)
