package official

import (
	"context"
	"fmt"
	"strings"

	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/security"
)

// UvAdapter manages Astral's uv package and tool manager on all platforms.
type UvAdapter struct{}

var _ adapters.Adapter = (*UvAdapter)(nil)

func (a *UvAdapter) Name() string { return "uv" }

func (a *UvAdapter) Detect() bool {
	return lookPath("uv")
}

var uvInfo = adapters.ToolInfo{
	ID:           "uv",
	Name:         "uv",
	Platforms:    platformsLinuxMacOSWindows,
	Trust:        security.TrustOfficial,
	UpdatePolicy: adapters.PolicyGated,
	Kind:         adapters.KindTool,
	Command:      "uv self update",
}

func (a *UvAdapter) Info() adapters.ToolInfo {
	return uvInfo
}

func (a *UvAdapter) Check(ctx context.Context) (adapters.UpdateInfo, error) {
	if !a.Detect() {
		return adapters.UpdateInfo{}, fmt.Errorf("uv is not installed")
	}

	rawVersion, _, _ := runCmdArgs(ctx, "uv", "--version")
	current := extractUvVersion(rawVersion)
	if current == "" {
		current = "unknown"
	}

	latest := current
	selfUpdateAvailable := false
	stdout, stderr, err := runCmdArgs(ctx, "uv", "self", "update", "--dry-run")
	if err != nil {
		if !isExternalManagerError(err, stdout+" "+stderr) {
			return adapters.UpdateInfo{}, commandFailureErr("uv", stderr, err)
		}
		selfUpdateAvailable = false
	} else {
		combined := stdout + " " + stderr
		selfUpdateAvailable = parseUvSelfUpdateOutput(combined)
		if selfUpdateAvailable {
			latest = extractUvLatestVersion(combined, current)
		}
	}

	stdout, stderr, err = runCmdArgs(ctx, "uv", "tool", "list", "--outdated")
	if err != nil {
		return adapters.UpdateInfo{}, commandFailureErr("uv", stderr, err)
	}
	toolsUpdateAvailable := parseUvToolListOutdatedOutput(stdout)

	updateAvailable := selfUpdateAvailable || toolsUpdateAvailable

	return adapters.UpdateInfo{
		CurrentVersion:  current,
		LatestVersion:   latest,
		UpdateAvailable: updateAvailable,
	}, nil
}

func extractUvLatestVersion(output, current string) string {
	rem := output
	for len(rem) > 0 {
		var line string
		line, rem, _ = strings.Cut(rem, "\n")
		rest := line
		for {
			var field string
			field, rest = nextField(rest)
			if field == "" {
				break
			}
			if strings.EqualFold(field, "to") {
				next, _ := nextField(rest)
				if next != "" {
					target := strings.Trim(next, "(),:;\"'")
					if isVersionLike(target) {
						return target
					}
				}
			}
		}
	}
	rem = output
	for len(rem) > 0 {
		var line string
		line, rem, _ = strings.Cut(rem, "\n")
		var candidate string
		rest := line
		for {
			var field string
			field, rest = nextField(rest)
			if field == "" {
				break
			}
			cleaned := strings.Trim(field, "(),:;\"'")
			if isVersionLike(cleaned) && cleaned != current {
				candidate = cleaned
			}
		}
		if candidate != "" {
			return candidate
		}
	}
	return current
}

func extractUvVersion(raw string) string {
	return extractVersion(raw)
}

func isExternalManagerError(err error, output string) bool {
	if err == nil {
		return false
	}
	isCode2 := isExitCode(err, 2) ||
		strings.Contains(err.Error(), "exit status 2") ||
		strings.Contains(err.Error(), "(exit 2)")
	if !isCode2 {
		return false
	}
	return containsFold(output, "external package manager") || containsFold(err.Error(), "external package manager")
}

func parseUvSelfUpdateOutput(output string) bool {
	trimmed := strings.TrimSpace(output)
	if trimmed == "" {
		return false
	}
	if containsFold(trimmed, "up to date") {
		return false
	}
	return containsFold(trimmed, "would update") ||
		containsFold(trimmed, "new version") ||
		containsFold(trimmed, "updating")
}

func parseUvToolListOutdatedOutput(output string) bool {
	trimmed := strings.TrimSpace(output)
	if trimmed == "" {
		return false
	}
	if containsFold(trimmed, "no tools installed") || containsFold(trimmed, "no outdated tools") {
		return false
	}
	return true
}

func (a *UvAdapter) Update(ctx context.Context, dryRun bool) (adapters.Result, error) {
	if !a.Detect() {
		return adapters.Result{Success: false}, fmt.Errorf("uv is not installed")
	}

	before := extractUvVersion(commandOutput(ctx, "uv", "--version"))
	if before == "" {
		before = "unknown"
	}

	if dryRun {
		return adapters.Result{
			Success: true,
			Before:  before,
			After:   before,
		}, nil
	}

	stdout, stderr, err := runCmdArgsUpdate(ctx, "uv", "self", "update")
	if err != nil {
		if !isExternalManagerError(err, stdout+" "+stderr) {
			return adapters.Result{
				Success: false,
				Before:  before,
				After:   before,
				Error:   fmt.Errorf("uv self update failed: %w", err),
				Stderr:  stderr,
			}, nil
		}
	}

	_, toolStderr, err := runCmdArgsUpdate(ctx, "uv", "tool", "upgrade", "--all")
	if err != nil {
		return adapters.Result{
			Success: false,
			Before:  before,
			After:   before,
			Error:   fmt.Errorf("uv tool upgrade failed: %w", err),
			Stderr:  toolStderr,
		}, nil
	}

	after := extractUvVersion(commandOutput(ctx, "uv", "--version"))
	if after == "" {
		after = before
	}

	return adapters.Result{
		Success: true,
		Before:  before,
		After:   after,
	}, nil
}
