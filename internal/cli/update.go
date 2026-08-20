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

	cmd.Flags().BoolVarP(&uf.DryRun, "dry-run", "n", false, "show planned actions without executing")

	return cmd
}

// updateDeps carries the injectable seams for runUpdate (design D5),
// mirroring checkDeps/selfUpdateDeps. The zero value uses production
// behavior: the production adapter list builder, real TTY detection, and
// the real CheckboxSelector.
type updateDeps struct {
	buildAdapterList func(cfg *config.Config, osName string) []adapters.Adapter
	// stdinIsTTY reports whether stdin is a TTY — the interactive gate
	// (design D2: TTY && !ci && !quiet && !dry-run). Zero value = production
	// stdinIsTTY().
	stdinIsTTY func() bool
	// selector runs the pending-set checkbox selector (design D2/D9) and
	// returns the selected tool IDs plus whether the user canceled. Zero
	// value = production CheckboxSelector over os.Stdout/os.Stdin.
	selector func(pending []output.SelectOption) ([]string, bool)
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

	r := output.NewRendererVerbose(os.Stdout, gf.Quiet, gf.Verbose)

	if uf.DryRun {
		r.DryRunHeader()
	}

	// Interactive gate (design D2): selector only when stdin is a TTY and
	// --ci, --quiet, and --dry-run are all unset. Any other combination keeps
	// today's sequential behavior byte-identical.
	if deps.stdinIsTTY == nil {
		deps.stdinIsTTY = stdinIsTTY
	}
	if deps.stdinIsTTY() && !gf.CI && !gf.Quiet && !uf.DryRun {
		return runUpdateInteractive(gf, uf, deps, filteredAdapters, r)
	}

	return runUpdateSequential(gf, uf, filteredAdapters, r)
}

// runUpdateSequential is today's update loop, unchanged: per tool it runs
// Detect, Check, ConfirmAction, policy gate, and Update, then renders the
// summary. It is byte-identical to the pre-Phase-3 sequential behavior —
// the interactive gate routes TTY runs away from it, and every bypass path
// (non-TTY, --ci, --quiet, --dry-run) still lands here.
func runUpdateSequential(gf *GlobalFlags, uf *UpdateFlags, filteredAdapters []adapters.Adapter, r *output.Renderer) error {
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
			r.Progress("Updating", i+1, total, info.Name)
		}

		// Check for updates
		updateInfo, err := a.Check()
		if err != nil {
			results = append(results, output.ToolResult{
				Name:   info.Name,
				Status: output.StatusFailed,
				Error:  timeoutErr(info.Name, "check", err),
				Stderr: err.Error(),
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
				Stderr: err.Error(),
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
				Stderr: errMsg.Error(),
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

// runUpdateInteractive implements the TTY update flow (design D2/D4/D5/D7):
// a concurrent pre-check via runChecks (which renders "Checking X/Y"
// progress), a checkbox selector over the pending (StatusAvailable) tools,
// and the carried-outcome loop that updates only the user's selection. A
// cancel shows the fixed message and exits 0; the selector is skipped
// entirely when nothing is pending (spec ux-patterns "No pending updates").
func runUpdateInteractive(gf *GlobalFlags, uf *UpdateFlags, deps updateDeps, filteredAdapters []adapters.Adapter, r *output.Renderer) error {
	// Pre-check: concurrent Detect + Check over the filtered set. The
	// outcomes carry updateInfo so the loop below never re-calls Check()
	// (design D4). "Checking X/Y" progress lines render here, before the
	// selector — interactive-path tests include them deliberately (D5).
	outcomes := runChecks(filteredAdapters, r, gf.Quiet, true)

	// Pending = tools with an update available, in input order (D9).
	var pending []output.SelectOption
	for _, oc := range outcomes {
		if oc.result.Status == output.StatusAvailable {
			pending = append(pending, output.SelectOption{
				ID:      oc.result.Name,
				Label:   oc.result.Name,
				Version: oc.result.Version,
			})
		}
	}

	// No pending updates → skip the selector, show the normal summary
	// (spec ux-patterns "No pending updates").
	if len(pending) == 0 {
		results := make([]output.ToolResult, len(outcomes))
		for i, oc := range outcomes {
			results[i] = oc.result
		}
		r.UpdateSummary(output.Summary{Results: results, DryRun: uf.DryRun})
		return nil
	}

	// Selector seam (design D2): zero value = production CheckboxSelector
	// reading os.Stdin in raw mode.
	if deps.selector == nil {
		deps.selector = func(opts []output.SelectOption) ([]string, bool) {
			res, err := output.NewCheckboxSelector(os.Stdout, os.Stdin, opts).Run()
			if err != nil {
				return nil, true
			}
			return res.Selected, res.Canceled
		}
	}
	selected, canceled := deps.selector(pending)
	if canceled {
		// Design D8: fixed cancel message, nothing updated, exit 0.
		r.UpdateCancelled()
		return nil
	}

	selectedSet := make(map[string]struct{}, len(selected))
	for _, id := range selected {
		selectedSet[id] = struct{}{}
	}

	// Carried-outcome loop over ALL outcomes (design D4/D6): Skipped/Failed
	// append as-is; Available tools are updated only when selected, and
	// deselected pending tools are dropped — the summary counts reflect the
	// executed selection, never the pending set. Always-update tools without
	// a reported update (brew et al.) are NOT force-updated in interactive
	// TTY runs (design D7): only the pending selection is processed.
	var results []output.ToolResult
	hasFailure := false
	adapterMap := adapterByID(filteredAdapters)
	updateIndex := 0
	updateTotal := len(selected)
	for _, oc := range outcomes {
		switch oc.result.Status {
		case output.StatusAvailable:
			if _, ok := selectedSet[oc.result.Name]; !ok {
				continue // deselected: dropped, never processed (D6)
			}
			a, ok := adapterMap[oc.result.Name]
			if !ok {
				continue
			}
			updateIndex++
			// The carried outcome feeds the confirm + policy gate — no second
			// Check() (D4). Per-tool ConfirmAction still runs for selected
			// tools: the selector is a user-choice UI, NOT a security
			// confirmation (spec ux-patterns).
			hasFailure = processSelectedOutcome(gf, a, oc.updateInfo, updateIndex, updateTotal, r, &results) || hasFailure
		default:
			// Skipped/Failed/Current append as-is, byte-identical to the
			// sequential summary.
			results = append(results, oc.result)
		}
	}

	r.UpdateSummary(output.Summary{Results: results, DryRun: uf.DryRun})

	// CI mode: exit non-zero on failure (byte-identical to sequential).
	if gf.CI && hasFailure {
		return fmt.Errorf("update completed with failures")
	}

	return nil
}

// processSelectedOutcome runs the per-tool confirm + policy gate + Update
// for one selected pending tool, appending its result. The updateInfo comes
// from the carried pre-check outcome — Check() is never re-invoked (design
// D4). It returns whether the tool failed, for the CI failure aggregation.
func processSelectedOutcome(gf *GlobalFlags, a adapters.Adapter, updateInfo adapters.UpdateInfo, index, total int, r *output.Renderer, results *[]output.ToolResult) bool {
	info := a.Info()

	// Progress, mirroring the sequential loop (index/total over the
	// executed selection).
	if !gf.Quiet && total > 1 {
		r.Progress("Updating", index, total, info.Name)
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
		*results = append(*results, output.ToolResult{
			Name:   info.Name,
			Status: output.StatusSkipped,
		})
		return false
	case security.ConfirmError:
		*results = append(*results, output.ToolResult{
			Name:   info.Name,
			Status: output.StatusFailed,
			Error:  fmt.Errorf("CI mode: custom tool requires confirmation"),
		})
		return true
	}

	// Gate: adapters declaring PolicyGated (apt, npm, pnpm, nvm)
	// update only when check() reported an update available (design
	// D2, spec Update Gating). Adapters declaring PolicyAlwaysUpdate
	// (brew, bun, docker, gh, go, opencode, winget, scoop, custom)
	// always run their update when requested.
	if info.UpdatePolicy == adapters.PolicyGated && !updateInfo.UpdateAvailable {
		*results = append(*results, output.ToolResult{
			Name:    info.Name,
			Status:  output.StatusCurrent,
			Version: updateInfo.CurrentVersion,
		})
		return false
	}

	// Update
	result, err := a.Update(false)
	if err != nil {
		*results = append(*results, output.ToolResult{
			Name:   info.Name,
			Status: output.StatusFailed,
			Error:  timeoutErr(info.Name, "update", err),
			Stderr: err.Error(),
		})
		return true
	}

	if result.Success {
		*results = append(*results, output.ToolResult{
			Name:    info.Name,
			Status:  output.StatusUpdated,
			Version: result.After,
		})
		return false
	}

	errMsg := result.Error
	if errMsg == nil {
		errMsg = fmt.Errorf("update failed")
	}
	*results = append(*results, output.ToolResult{
		Name:   info.Name,
		Status: output.StatusFailed,
		Error:  timeoutErr(info.Name, "update", errMsg),
		Stderr: errMsg.Error(),
	})
	return true
}
