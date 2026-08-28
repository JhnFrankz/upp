package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/adapters/official"
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
	// Opt-in group bulk flags (spec command-interface Opt-in flags / bulk-update
	// Opt-In Group Bulk Trigger): --manager <mgr> and its alias --update-group
	// <mgr>. Either binds uf.Manager, which routes runUpdate to the
	// manager-group bulk path. Present but inert in the default path (a bare
	// `upp update` leaves Manager empty — bulk default is a later increment).
	cmd.Flags().StringVar(&uf.Manager, "manager", "", "run a manager-group bulk update of <mgr>'s owned tools (opt-in)")
	cmd.Flags().StringVar(&uf.Manager, "update-group", "", "alias for --manager: run a manager-group bulk update of <mgr>'s owned tools (opt-in)")

	return cmd
}

// updateDeps carries the injectable seams for runUpdate (design D5),
// mirroring selfUpdateDeps. The zero value uses production
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

	// Opt-in group bulk path (design D3): when --manager / --update-group is
	// set, route to runUpdateGroup and bypass the standard per-tool path
	// entirely. A bare `upp update` leaves uf.Manager empty, so the default
	// path is unchanged (spec bulk-update "Default unchanged").
	if uf.Manager != "" {
		return runUpdateGroup(gf, uf, adapterList, p.OS)
	}

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
		return runUpdateInteractive(gf, uf, deps, filteredAdapters, r, p.OS)
	}

	return runUpdateSequential(gf, uf, filteredAdapters, r, p.OS)
}

// runUpdateSequential is today's update loop, unchanged: per tool it runs
// Detect, Check, ConfirmAction, policy gate, and Update, then renders the
// summary. It is byte-identical to the pre-Phase-3 sequential behavior —
// the interactive gate routes TTY runs away from it, and every bypass path
// (non-TTY, --ci, --quiet, --dry-run) still lands here.
//
// osName is the canonical platform key (platform.OSLinux/OSMacOS/OSWindows)
// used to resolve an owned tool's effective UpdatePolicy from its manager on
// the delegated path (WU2, spec Update Gating).
func runUpdateSequential(gf *GlobalFlags, uf *UpdateFlags, filteredAdapters []adapters.Adapter, r *output.Renderer, osName string) error {
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
		// (brew, bun, opencode, winget, scoop, custom) always run their
		// update when requested. On the delegated path, the gate uses the
		// MANAGER's effective policy: an owned tool (docker, gh, go) inherits
		// its managing adapter's UpdatePolicy, and the owned tool's own
		// declared policy is INERT (spec Update Gating).
		if resolveEffectiveUpdatePolicy(a, osName) == adapters.PolicyGated && !updateInfo.UpdateAvailable {
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
// a concurrent pre-check via runChecks (completion callback seam; the live
// CheckBoard lands in Unit 3), a checkbox selector over the pending
// (StatusAvailable) tools,
// and the carried-outcome loop that updates only the user's selection. A
// cancel shows the fixed message and exits 0; the selector is skipped
// entirely when nothing is pending (spec ux-patterns "No pending updates").
//
// osName is the canonical platform key used to resolve an owned tool's
// effective UpdatePolicy from its manager on the delegated path (WU2).
func runUpdateInteractive(gf *GlobalFlags, uf *UpdateFlags, deps updateDeps, filteredAdapters []adapters.Adapter, r *output.Renderer, osName string) error {
	// Pre-check: concurrent Detect + Check over the filtered set. The
	// outcomes carry updateInfo so the loop below never re-calls Check()
	// (design D4). The live CheckBoard renders the pre-check (spec
	// ux-patterns Live Check Board): rows are built in canonical filtered
	// order, painted before the pool starts, flipped once per completion
	// through the onResult seam, and settled before the selector renders.
	// Color follows the renderer's single TTY detection (D5); without color
	// the board falls back to one plain line per completion.
	//
	// Grouping (design render/wiring): the filtered set is reordered into
	// group order (manager rows first, then their owned tools, then
	// standalone tools) so the board rows and the selector options appear
	// grouped by ownership. Display-only: the reorder never changes per-tool
	// completion or the filtered set membership/IDs. For the hermetic fake
	// adapters in tests (all standalone) this is a no-op that preserves
	// byte-identical behavior.
	grouped := output.GroupOrder(filteredAdapters, osName)
	names := make([]string, len(grouped))
	for i, a := range grouped {
		names[i] = a.Info().Name
	}
	board := output.NewCheckBoard(os.Stdout, r.Color(), names)
	board.Start()
	outcomes := runChecks(grouped, func(index int, oc checkOutcome) {
		board.Complete(index, oc.result)
	})
	board.Finish()

	// Pending = tools with an update available, in group order (D9).
	var pending []output.SelectOption
	for i, oc := range outcomes {
		if oc.result.Status == output.StatusAvailable {
			pending = append(pending, output.SelectOption{
				ID:      oc.result.Name,
				Label:   oc.result.Name,
				Version: oc.result.Version,
				Group:   output.OwnerGroupLabel(grouped[i], osName, grouped),
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
			hasFailure = processSelectedOutcome(gf, a, oc.updateInfo, updateIndex, updateTotal, r, &results, osName) || hasFailure
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
func processSelectedOutcome(gf *GlobalFlags, a adapters.Adapter, updateInfo adapters.UpdateInfo, index, total int, r *output.Renderer, results *[]output.ToolResult, osName string) bool {
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
	// (brew, bun, opencode, winget, scoop, custom) always run their
	// update when requested. On the delegated path, the gate uses the
	// MANAGER's effective policy: an owned tool inherits its managing
	// adapter's UpdatePolicy, and the owned tool's own declared policy is
	// INERT (spec Update Gating).
	if resolveEffectiveUpdatePolicy(a, osName) == adapters.PolicyGated && !updateInfo.UpdateAvailable {
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

// runUpdateGroup implements the opt-in manager-group bulk path (spec
// bulk-update, design D3): `upp update --manager <mgr>` / `--update-group
// <mgr>` enumerates the manager's resolving owned tools on the current
// platform, drops any --skip-ed, checks per-package availability, gates on the
// manager's UpdatePolicy, confirms by REAL command risk (EnforceRisk, D4), and
// runs each owned tool's per-manager package command via the manager's
// privileged executor. It returns a manager-group bulk summary.
//
// The manager's own self-only row is NEVER conflated with the owned-tool group
// batch (spec bulk-update "Manager self separate"): apt's own self path stays
// apt.Update; the group path upgrades gh/docker via their package commands.
//
// osName is the canonical platform key (platform.OSLinux/OSMacOS/OSWindows).
func runUpdateGroup(gf *GlobalFlags, uf *UpdateFlags, adapterList []adapters.Adapter, osName string) error {
	manager := adapterByName(adapterList, uf.Manager)
	if manager == nil {
		return fmt.Errorf("manager %q not found in the current platform", uf.Manager)
	}
	if manager.Info().Kind != adapters.KindManager {
		return fmt.Errorf("%q is not a manager", uf.Manager)
	}

	// Enumerate the manager's owned tools (KindTool whose resolving owner on
	// this platform is the manager), then drop --skip-ed.
	_, skipList := ParseFilter(gf.Only, gf.Skip)
	var owned []adapters.Adapter
	for _, a := range adapterList {
		if a.Info().Kind != adapters.KindTool {
			continue
		}
		owner := resolvingOwner(a, osName)
		if owner == nil || owner.Name() != uf.Manager {
			continue
		}
		owned = append(owned, a)
	}
	owned = filterGroupSkip(owned, skipList)

	r := output.NewRendererVerbose(os.Stdout, gf.Quiet, gf.Verbose)

	// Per-package availability (spec bulk-update Per-Package Availability
	// Check). A tool whose package has no update (or whose check fails) is
	// reported current / skipped — it is NOT updated.
	var tools []groupTool
	for _, a := range owned {
		pkg := ownedPackage(a, osName)
		if pkg == "" {
			// Fail-closed: no declared package → skipped from the batch, never
			// guessed (design Migration/Rollout).
			continue
		}
		checker, ok := manager.(adapters.PackageChecker)
		if !ok {
			return fmt.Errorf("manager %q does not support per-package checks", uf.Manager)
		}
		info, err := checker.CheckPackage(pkg)
		t := groupTool{adapter: a, pkg: pkg}
		if err != nil {
			t.failed = true
		} else {
			t.info = info
		}
		tools = append(tools, t)
	}

	// Gate (design D5): the group's gate IS the manager's UpdatePolicy. A
	// PolicyGated manager (apt) gates the group on availability: each owned
	// tool updates only when its package has an update, and the group is
	// blocked (all reported current) when no owned package is available. A
	// PolicyAlwaysUpdate manager (brew, winget, scoop) runs its group when
	// requested — owned tools update regardless of the check result (spec
	// tool-adapter "Owned inherits always").
	managerPolicy := resolveEffectiveUpdatePolicy(manager, osName)

	// Batch preview (spec ux-patterns Opt-In Flag UX 'Batch rendered'): show
	// the planned manager-group batch BEFORE executing — each owned tool and
	// its planned state, plus whether the batch is gated.
	var batchTools []output.GroupBatchTool
	for _, t := range tools {
		info := t.adapter.Info()
		bt := output.GroupBatchTool{Name: info.Name}
		if t.failed {
			bt.CheckFailed = true
		} else if managerPolicy == adapters.PolicyGated && !t.info.UpdateAvailable {
			// Gated manager with no package update → planned current.
		} else if t.info.UpdateAvailable {
			bt.UpdateAvailable = true
			bt.Version = t.info.CurrentVersion + " → " + t.info.LatestVersion
		}
		batchTools = append(batchTools, bt)
	}
	if len(batchTools) > 0 || managerPolicy == adapters.PolicyAlwaysUpdate {
		r.GroupBatchPreview(output.GroupBatchPreview{
			Manager: manager.Info().Name,
			Gated:   managerPolicy == adapters.PolicyGated,
			Tools:   batchTools,
		})
	}

	// Group bulk summary, built in canonical enumeration order (spec
	// ux-patterns Group bulk summary).
	var results []output.ToolResult
	hasFailure := false
	for _, t := range tools {
		info := t.adapter.Info()
		if t.failed {
			results = append(results, output.ToolResult{
				Name:   info.Name,
				Status: output.StatusFailed,
			})
			hasFailure = true
			continue
		}
		// Per-Package Availability (spec bulk-update): a Gated manager's owned
		// tool whose package has no update is reported current, never updated.
		// An AlwaysUpdate manager's owned tool is NOT availability-blocked — it
		// proceeds to confirm + update (single-adapter parity).
		if managerPolicy == adapters.PolicyGated && !t.info.UpdateAvailable {
			results = append(results, output.ToolResult{
				Name:    info.Name,
				Status:  output.StatusCurrent,
				Version: t.info.CurrentVersion,
			})
			continue
		}

		// Dry-run: show the planned package update without executing it (spec
		// ux-patterns "Group dry-run": a pending owned tool reports "would
		// update"). Dry-run fires BEFORE the confirm block so a sudo group in
		// --dry-run never prompts nor mutates.
		if uf.DryRun {
			results = append(results, output.ToolResult{
				Name:    info.Name,
				Status:  output.StatusAvailable,
				Version: fmt.Sprintf("%s → %s", t.info.CurrentVersion, t.info.LatestVersion),
			})
			continue
		}

		// Confirm by REAL command risk (design D4, spec security-model): an
		// owned tool is TrustOfficial, but a sudo-heavy package command
		// (apt) MUST prompt / fail in --ci despite that.
		riskCommand := fmt.Sprintf("%s %s", updateCmdName(uf.Manager), t.pkg)
		riskLevel := security.ClassifyCommand(riskCommand)
		decision := security.ConfirmAction(security.ConfirmConfig{
			ToolName:    info.Name,
			TrustLevel:  info.Trust,
			RiskLevel:   riskLevel,
			Command:     riskCommand,
			Privileges:  info.Privileges,
			CI:          gf.CI,
			EnforceRisk: true,
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
				Error:  fmt.Errorf("CI mode: group update requires confirmation"),
			})
			hasFailure = true
			continue
		}

		// Run the owned tool's per-manager package command via the manager's
		// privileged executor (PackageUpdater).
		updater, ok := manager.(adapters.PackageUpdater)
		if !ok {
			results = append(results, output.ToolResult{
				Name:   info.Name,
				Status: output.StatusFailed,
				Error:  fmt.Errorf("manager %q does not support per-package updates", uf.Manager),
			})
			hasFailure = true
			continue
		}
		result, err := updater.UpdatePackage(t.pkg)
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

	r.GroupBulkSummary(output.GroupBulkSummary{
		Manager: manager.Info().Name,
		DryRun:  uf.DryRun,
		Results: results,
	})

	// --ci: a high-risk sudo group update that needed confirmation fails
	// non-zero (spec security-model "--ci sudo group fails").
	if gf.CI && hasFailure {
		return fmt.Errorf("group update completed with failures")
	}

	return nil
}

// groupTool is one owned tool in a manager-group bulk batch, carrying its
// package name, its per-package availability, and whether the availability
// check failed (a failed check is never "available" nor current — it is
// reported failed).
type groupTool struct {
	adapter adapters.Adapter
	pkg     string
	info    adapters.UpdateInfo
	failed  bool
}

// adapterByName returns the adapter in the list with the given name, or nil.
func adapterByName(adapterList []adapters.Adapter, name string) adapters.Adapter {
	for _, a := range adapterList {
		if a.Name() == name {
			return a
		}
	}
	return nil
}

// filterGroupSkip drops any owned tool named in the --skip list. It matches by
// tool ID (case-insensitive), consistent with the per-tool filter. Unlike
// FilterTools it does not warn about unknown names (the group batch is
// derived from the ownership model, not user-provided the selector set).
func filterGroupSkip(owned []adapters.Adapter, skipList []string) []adapters.Adapter {
	if len(skipList) == 0 {
		return owned
	}
	skipSet := make(map[string]bool, len(skipList))
	for _, s := range skipList {
		skipSet[strings.ToLower(s)] = true
	}
	var result []adapters.Adapter
	for _, a := range owned {
		if skipSet[strings.ToLower(a.Name())] {
			continue
		}
		result = append(result, a)
	}
	return result
}

// ownedPackage returns the package name under the resolving manager for an
// owned tool on osName, or "" when none is declared (fail-closed: absent
// entries skip the group batch, never guessed).
func ownedPackage(a adapters.Adapter, osName string) string {
	return a.Info().ManagerPackage[osName]
}

// updateCmdName returns the package-manager command token used to build the
// conventional risk command for a manager's owned-package command.
func updateCmdName(manager string) string {
	switch manager {
	case "apt":
		return "sudo apt install --only-upgrade"
	case "brew":
		return "brew upgrade"
	case "winget":
		return "winget upgrade"
	default:
		return manager + " upgrade"
	}
}

// resolveEffectiveUpdatePolicy returns the UpdatePolicy that governs whether
// an adapter's Update() runs. For an owned tool (an official tool with a
// resolving manager on the given OS, or a custom tool carrying an injected
// manager adapter) the MANAGER's policy governs — the owned tool's own
// declared policy is INERT on the delegated path (spec Update Gating).
// Otherwise the adapter's own declared policy applies. osName is the canonical
// platform key (platform.OSLinux/OSMacOS/OSWindows), NOT runtime.GOOS (which
// returns "darwin" on macOS) — the WU1-documented gotcha.
func resolveEffectiveUpdatePolicy(a adapters.Adapter, osName string) adapters.UpdatePolicy {
	if owner := resolvingOwner(a, osName); owner != nil {
		return owner.Info().UpdatePolicy
	}
	return a.Info().UpdatePolicy
}

// resolvingOwner returns the manager adapter that owns the given adapter on
// the given OS, or nil when the adapter has no resolving owner (standalone).
// A custom tool exposes its injected manager via ManagerAdapter; an official
// tool resolves through official.ResolveOwner (keyed by platform constant).
func resolvingOwner(a adapters.Adapter, osName string) adapters.Adapter {
	if custom, ok := a.(*adapters.CustomAdapter); ok {
		if m := custom.ManagerAdapter(); m != nil {
			return m
		}
	}
	return official.ResolveOwner(a.Name(), osName)
}
