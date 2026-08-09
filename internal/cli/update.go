package cli

import (
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
			return runUpdate(gf, uf)
		},
	}

	cmd.Flags().BoolVar(&uf.DryRun, "dry-run", false, "show planned actions without executing")

	return cmd
}

func runUpdate(gf *GlobalFlags, uf *UpdateFlags) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("cannot load config: %w", err)
	}

	p := platform.Detect()
	adapterList := buildAdapterList(cfg, p.OS)

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

	r := output.NewRenderer(os.Stdout, gf.Quiet)

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
				Error:  err,
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
		riskLevel := security.ClassifyCommand(info.Name + " update")
		decision := security.ConfirmAction(security.ConfirmConfig{
			ToolName:   info.Name,
			TrustLevel: trustLevelString(info.Trust),
			RiskLevel:  riskLevel,
			Command:    info.Name + " update",
			CI:         gf.CI,
			Trusted:    info.Trust == adapters.TrustTrusted,
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
				Error:  fmt.Errorf("CI mode: untrusted custom tool requires confirmation"),
			})
			hasFailure = true
			continue
		}

		// Update
		result, err := a.Update(false)
		if err != nil {
			results = append(results, output.ToolResult{
				Name:   info.Name,
				Status: output.StatusFailed,
				Error:  err,
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
				Error:  errMsg,
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

func trustLevelString(level adapters.TrustLevel) string {
	if level == adapters.TrustTrusted {
		return "official"
	}
	return "custom"
}
