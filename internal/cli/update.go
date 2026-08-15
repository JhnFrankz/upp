package cli

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/config"
	"github.com/JhnFrankz/upp/internal/output"
	"github.com/JhnFrankz/upp/internal/platform"
	"github.com/JhnFrankz/upp/internal/security"
)

// NewUpdateCommand creates the `upp update` command.
func NewUpdateCommand(gf *GlobalFlags) *cobra.Command {
	uf := &UpdateFlags{}

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Apply updates to enabled tools",
		Long:  "Process each enabled tool: detect, check, confirm, and update.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpdate(gf, uf, cliDeps.update)
		},
	}

	cmd.Flags().BoolVar(&uf.DryRun, "dry-run", false, "show planned actions without executing")

	return cmd
}

// updateDeps carries the injectable seam for runUpdate (design D5), mirroring
// checkDeps/selfUpdateDeps. The zero value uses the production adapter list
// builder.
type updateDeps struct {
	buildAdapterList func(cfg *config.Config, osName string) []adapters.Adapter
}

func runUpdate(gf *GlobalFlags, uf *UpdateFlags, deps updateDeps) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("cannot load config: %w", err)
	}

	p, err := platform.Detect()
	if err != nil {
		return fmt.Errorf("cannot detect platform: %w", err)
	}
	if deps.buildAdapterList == nil {
		deps.buildAdapterList = buildAdapterList
	}
	adapterList := deps.buildAdapterList(cfg, p.OS)

	toolIDs := adapterIDs(adapterList)
	onlyList, skipList := ParseFilter(gf.Only, gf.Skip)
	filteredIDs := FilterTools(toolIDs, onlyList, skipList, os.Stderr)

	adapterMap := adapterByID(adapterList)
	var filteredAdapters []adapters.Adapter
	for _, id := range filteredIDs {
		if a, ok := adapterMap[id]; ok {
			filteredAdapters = append(filteredAdapters, a)
		}
	}

	lang := "en"
	if cfg != nil {
		lang = cfg.Settings.Language
	}
	r := output.NewRendererWithLang(os.Stdout, gf.Quiet, lang)

	if uf.DryRun {
		r.DryRunHeader()
	}

	var results []output.ToolResult
	total := len(filteredAdapters)
	hasFailure := false

	for i, a := range filteredAdapters {
		info := a.Info()

		// Detect
		if !a.Detect() {
			results = append(results, output.ToolResult{
				Name:   info.Name,
				Status: output.StatusSkipped,
			})
			continue
		}

		// Progress
		if !gf.Quiet && total > 1 {
			r.Progress(i+1, total, info.Name)
		}

		// Check for updates
		updateInfo, err := a.Check()
		if err != nil {
			results = append(results, output.ToolResult{
				Name:   info.Name,
				Status: output.StatusFailed,
				Error:  timeoutErr(info.Name, "check", err),
			})
			hasFailure = true
			continue
		}

		// Dry run: just show planned action
		if uf.DryRun {
			if updateInfo.UpdateAvailable {
				r.DryRunPlanned(fmt.Sprintf("%s (%s → %s)", info.Name, updateInfo.CurrentVersion, updateInfo.LatestVersion))
				results = append(results, output.ToolResult{
					Name:    info.Name,
					Status:  output.StatusAvailable,
					Version: fmt.Sprintf("%s → %s", updateInfo.CurrentVersion, updateInfo.LatestVersion),
				})
			} else {
				results = append(results, output.ToolResult{
					Name:    info.Name,
					Status:  output.StatusCurrent,
					Version: updateInfo.CurrentVersion,
				})
			}
			continue
		}

		// Confirm if needed — always evaluate trust/risk, even in CI mode.
		// In CI mode, ConfirmAction returns ConfirmError for untrusted tools.
		riskCommand := info.Command
		if riskCommand == "" {
			// Official adapters don't expose a command; use the conventional one.
			riskCommand = info.Name + " update"
		}
		riskLevel := security.ClassifyCommand(riskCommand)
		decision := security.ConfirmAction(security.ConfirmConfig{
			ToolName:   info.Name,
			TrustLevel: info.Trust,
			RiskLevel:  riskLevel,
			Command:    info.Command,
			Privileges: info.Privileges,
			CI:         gf.CI,
		})

		switch decision {
		case security.ConfirmDeny:
			results = append(results, output.ToolResult{
				Name:   info.Name,
				Status: output.StatusSkipped,
			})
			continue
		case security.ConfirmError:
			results = append(results, output.ToolResult{
				Name:   info.Name,
				Status: output.StatusFailed,
				Error:  fmt.Errorf("CI mode: custom tool requires confirmation"),
			})
			hasFailure = true
			continue
		}

		// Gate: adapters declaring PolicyGated (apt, npm, pnpm, nvm)
		// update only when check() reported an update available (design
		// D2, spec Update Gating). Adapters declaring PolicyAlwaysUpdate
		// (brew, bun, docker, gh, go, opencode, winget, scoop, custom)
		// always run their update when requested.
		if info.UpdatePolicy == adapters.PolicyGated && !updateInfo.UpdateAvailable {
			results = append(results, output.ToolResult{
				Name:    info.Name,
				Status:  output.StatusCurrent,
				Version: updateInfo.CurrentVersion,
			})
			continue
		}

		// Update
		result, err := a.Update(false)
		if err != nil {
			results = append(results, output.ToolResult{
				Name:   info.Name,
				Status: output.StatusFailed,
				Error:  timeoutErr(info.Name, "update", err),
			})
			hasFailure = true
			continue
		}

		if result.Success {
			results = append(results, output.ToolResult{
				Name:    info.Name,
				Status:  output.StatusUpdated,
				Version: result.After,
			})
		} else {
			errMsg := result.Error
			if errMsg == nil {
				errMsg = fmt.Errorf("update failed")
			}
			results = append(results, output.ToolResult{
				Name:   info.Name,
				Status: output.StatusFailed,
				Error:  timeoutErr(info.Name, "update", errMsg),
			})
			hasFailure = true
		}
	}

	summary := output.Summary{
		Results: results,
		DryRun:  uf.DryRun,
	}
	r.UpdateSummary(summary)

	// CI mode: exit non-zero on failure
	if gf.CI && hasFailure {
		return fmt.Errorf("update completed with failures")
	}

	return nil
}

// timeoutErr maps a context deadline exceeded onto a structured error naming
// the tool, operation, and timeout limit (design D3, spec Subprocess
// Timeouts). Non-timeout errors pass through unchanged; the %w chain
// preserves errors.Is detection.
func timeoutErr(name, op string, err error) error {
	if !errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	limit := adapters.UpdateTimeout
	if op == "check" {
		limit = adapters.CheckTimeout
	}
	return fmt.Errorf("%s %s timed out after %s: %w", name, op, limit, err)
}
