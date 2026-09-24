package official

import (
	"context"
	"fmt"
	"strings"

	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/security"
)

// BunAdapter manages Bun runtime on all platforms.
type BunAdapter struct{}

func (a *BunAdapter) Name() string { return "bun" }

func (a *BunAdapter) Detect() bool {
	return lookPath("bun")
}

func (a *BunAdapter) Check(ctx context.Context) (adapters.UpdateInfo, error) {
	if !a.Detect() {
		return adapters.UpdateInfo{}, fmt.Errorf("bun is not installed")
	}

	current := commandOutput(ctx, "bun", "--version")
	current = strings.TrimSpace(current)
	if current == "" {
		current = "unknown"
	}

	return adapters.UpdateInfo{
		CurrentVersion:  current,
		LatestVersion:   current, // bun upgrade handles version resolution internally
		UpdateAvailable: false,
	}, nil
}

func (a *BunAdapter) Update(ctx context.Context, dryRun bool) (adapters.Result, error) {
	if !a.Detect() {
		return adapters.Result{Success: false}, fmt.Errorf("bun is not installed")
	}

	before := commandOutput(ctx, "bun", "--version")
	before = strings.TrimSpace(before)

	if dryRun {
		return adapters.Result{
			Success: true,
			Before:  before,
			After:   before,
		}, nil
	}

	_, stderr, err := runCmdArgsUpdate(ctx, "bun", "upgrade")
	if err != nil {
		return adapters.Result{
			Success: false,
			Before:  before,
			After:   before,
			Error:   fmt.Errorf("bun upgrade failed: %w", err),
			Stderr:  stderr,
		}, nil
	}

	if stderr != "" && strings.Contains(stderr, "error") {
		return adapters.Result{
			Success: false,
			Before:  before,
			After:   before,
			Error:   fmt.Errorf("bun upgrade error: %s", truncate(stderr, 200)),
			Stderr:  stderr,
		}, nil
	}

	after := commandOutput(ctx, "bun", "--version")
	after = strings.TrimSpace(after)

	return adapters.Result{
		Success: true,
		Before:  before,
		After:   after,
	}, nil
}

func (a *BunAdapter) Info() adapters.ToolInfo {
	return adapters.ToolInfo{
		ID:           "bun",
		Name:         "Bun",
		Platforms:    []string{"linux", "macos", "windows"},
		Trust:        security.TrustOfficial,
		UpdatePolicy: adapters.PolicyAlwaysUpdate,
		Kind:         adapters.KindTool,
		// Command is the exact string Update() runs, declared so the plan's
		// RiskCommand and the confirmation gate see what actually executes.
		Command: "bun upgrade",
	}
}
