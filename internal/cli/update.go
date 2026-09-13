package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/config"
	"github.com/JhnFrankz/upp/internal/engine"
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

// ownedCheckerAdapter delegates Check to a PackageChecker for an owned tool.
type ownedCheckerAdapter struct {
	adapters.Adapter
	checker adapters.PackageChecker
	pkg     string
}

func (o *ownedCheckerAdapter) Check() (adapters.UpdateInfo, error) {
	info, err := o.checker.CheckPackage(o.pkg)
	if err != nil {
		return adapters.UpdateInfo{}, err
	}
	if info != (adapters.UpdateInfo{}) {
		return info, nil
	}
	return o.Adapter.Check()
}

// prepareCheckAdapters wraps owned tools whose managers provide a PackageChecker.
func prepareCheckAdapters(adapterList []adapters.Adapter, osName string, allAdapters ...[]adapters.Adapter) []adapters.Adapter {
	result := make([]adapters.Adapter, len(adapterList))
	for i, a := range adapterList {
		owner := engine.ResolvingOwner(a, osName, allAdapters...)
		if owner != nil && a.Info().Manager != nil && a.Info().Manager[osName] != "" {
			if checker, ok := owner.(adapters.PackageChecker); ok {
				pkg := engine.OwnedPackage(a, osName)
				if pkg != "" {
					result[i] = &ownedCheckerAdapter{
						Adapter: a,
						checker: checker,
						pkg:     pkg,
					}
					continue
				}
			}
		}
		result[i] = a
	}
	return result
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

	var allAdapters []adapters.Adapter
	var opts []engine.Option
	if deps.buildAdapterList != nil {
		allAdapters = deps.buildAdapterList(cfg, p.OS)
		opts = append(opts, engine.WithAdapters(allAdapters))
	}
	eng := engine.New(cfg, p.OS, opts...)
	if allAdapters == nil {
		allAdapters, _ = eng.Resolve(engine.Filter{})
	}

	onlyList := ParseFilter(gf.Only)
	if len(onlyList) > 0 {
		FilterTools(adapterIDs(allAdapters), onlyList, os.Stderr)
	}

	filteredAdapters, err := eng.Resolve(engine.Filter{Only: onlyList})
	if err != nil {
		return fmt.Errorf("cannot resolve tools: %w", err)
	}

	r := output.NewRendererVerbose(os.Stdout, gf.Quiet, gf.Verbose)

	if uf.DryRun {
		r.DryRunHeader()
	}

	// Interactive gate (design D2): selector only when stdin is a TTY and
	// --ci, --quiet, and --dry-run are all unset. Any other combination keeps
	// sequential behavior.
	if deps.stdinIsTTY == nil {
		deps.stdinIsTTY = stdinIsTTY
	}
	if deps.stdinIsTTY() && !gf.CI && !gf.Quiet && !uf.DryRun {
		return runUpdateInteractive(gf, uf, deps, filteredAdapters, r, p.OS, eng, allAdapters)
	}

	return runUpdateSequential(gf, uf, filteredAdapters, r, p.OS, eng, allAdapters)
}

// runUpdateSequential processes each filtered adapter: for owned tools under a
// package manager, it checks per-package availability via PackageChecker and
// updates via PackageUpdater (with EnforceRisk: true); for standalone tools, it
// runs standard Check and Update. Per-tool errors are isolated.
//
// osName is the canonical platform key (platform.OSLinux/OSMacOS/OSWindows).
func runUpdateSequential(gf *GlobalFlags, uf *UpdateFlags, filteredAdapters []adapters.Adapter, r *output.Renderer, osName string, eng *engine.Engine, allAdapters ...[]adapters.Adapter) error {
	total := len(filteredAdapters)
	if total == 0 {
		r.UpdateSummary(output.Summary{Results: nil, DryRun: uf.DryRun})
		return nil
	}

	if eng == nil {
		var opts []engine.Option
		if len(allAdapters) > 0 && allAdapters[0] != nil {
			opts = append(opts, engine.WithAdapters(allAdapters[0]))
		}
		eng = engine.New(nil, osName, opts...)
	}

	checkAdapters := prepareCheckAdapters(filteredAdapters, osName, allAdapters...)
	outcomes, err := eng.Check(context.Background(), checkAdapters, nil)
	if err != nil {
		return err
	}

	plan, err := eng.Plan(outcomes, engine.Filter{})
	if err != nil {
		return err
	}

	updatesByTool := make(map[string]engine.PlannedUpdate, len(plan.Updates))
	for _, u := range plan.Updates {
		updatesByTool[u.ToolName] = u
		updatesByTool[u.ToolID] = u
	}

	results := make([]output.ToolResult, total)
	hasFailure := false

	for i, a := range filteredAdapters {
		info := a.Info()
		oc := outcomes[i]

		switch oc.Status {
		case engine.StatusSkipped:
			results[i] = output.ToolResult{
				Name:   info.Name,
				Status: output.StatusSkipped,
			}
		case engine.StatusFailed:
			results[i] = output.ToolResult{
				Name:   info.Name,
				Status: output.StatusFailed,
				Error:  oc.Err,
				Stderr: oc.Stderr,
			}
			hasFailure = true
		case engine.StatusCurrent, engine.StatusAvailable:
			u, isUpdate := updatesByTool[info.Name]
			if !isUpdate {
				u, isUpdate = updatesByTool[info.ID]
			}
			if !isUpdate {
				results[i] = output.ToolResult{
					Name:    info.Name,
					Status:  output.StatusCurrent,
					Version: oc.CurrentVersion,
				}
				continue
			}

			// Dry run: show the planned action only, never execute it.
			if uf.DryRun {
				if !gf.Quiet && total > 1 {
					r.Progress("Updating", i+1, total, info.Name)
				}
				if oc.UpdateAvailable {
					r.DryRunPlanned(fmt.Sprintf("%s (%s → %s)", info.Name, oc.CurrentVersion, oc.LatestVersion))
					results[i] = output.ToolResult{
						Name:    info.Name,
						Status:  output.StatusAvailable,
						Version: fmt.Sprintf("%s → %s", oc.CurrentVersion, oc.LatestVersion),
					}
				} else {
					results[i] = output.ToolResult{
						Name:    info.Name,
						Status:  output.StatusCurrent,
						Version: oc.CurrentVersion,
					}
				}
				continue
			}

			// The shared executor owns confirm + risk + privilege parity with
			// the interactive path (design D2/D3); the plan's PlannedUpdate is
			// the source of policy, not a re-derivation here.
			results[i] = executePlannedUpdate(gf, u, a, i+1, total, r, osName, allAdapters...)
			if results[i].Status == output.StatusFailed {
				hasFailure = true
			}
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
// the tool, operation, and timeout limit.
func timeoutErr(name, op string, err error) error {
	return engine.TimeoutErr(name, op, err)
}

// runUpdateInteractive implements the TTY update flow:
// a concurrent pre-check via eng.Check, a checkbox selector over the
// plan-derived pending set (engine.Plan, so PolicyAlwaysUpdate tools appear
// even when current), and the carried-outcome loop that updates the user's
// selection while reporting every deselected pending tool distinctly.
func runUpdateInteractive(gf *GlobalFlags, uf *UpdateFlags, deps updateDeps, filteredAdapters []adapters.Adapter, r *output.Renderer, osName string, eng *engine.Engine, allAdapters ...[]adapters.Adapter) error {
	if eng == nil {
		var opts []engine.Option
		if len(allAdapters) > 0 && allAdapters[0] != nil {
			opts = append(opts, engine.WithAdapters(allAdapters[0]))
		}
		eng = engine.New(nil, osName, opts...)
	}

	grouped := output.GroupOrder(filteredAdapters, osName)
	names := make([]string, len(grouped))
	for i, a := range grouped {
		names[i] = a.Info().Name
	}
	board := output.NewCheckBoard(os.Stdout, r.Color(), names)
	board.Start()
	outcomes, _ := eng.Check(context.Background(), grouped, func(prog engine.CheckProgress) {
		board.Complete(prog.Index, outcomeToToolResult(prog.Outcome))
	})
	board.Finish()

	// The engine's plan is the single source of truth for identity, pending
	// eligibility, and execution metadata for the interactive path (design
	// D2/D3/D5, PC1): the pending set IS plan.Updates, so PolicyAlwaysUpdate
	// tools (e.g. brew, bun) appear even when currently up to date. Option IDs
	// are the canonical stable identity; labels are display-only (spec
	// Canonical Tool Selection Identity).
	plan, err := eng.Plan(outcomes, engine.Filter{})
	if err != nil {
		return err
	}
	planByID := make(map[string]engine.PlannedUpdate, len(plan.Updates))
	for _, u := range plan.Updates {
		planByID[u.ToolID] = u
	}
	adapterMap := adapterByID(filteredAdapters)

	pending := make([]output.SelectOption, 0, len(plan.Updates))
	for _, u := range plan.Updates {
		label := u.ToolName
		if label == "" {
			label = u.ToolID
		}
		var group string
		if a, ok := adapterMap[u.ToolID]; ok {
			group = output.OwnerGroupLabel(a, osName, grouped)
		}
		pending = append(pending, output.SelectOption{
			ID:      u.ToolID,
			Label:   label,
			Version: fmt.Sprintf("%s → %s", u.CurrentVersion, u.LatestVersion),
			Group:   group,
		})
	}

	// No pending updates → skip the selector, show the normal summary
	if len(pending) == 0 {
		results := make([]output.ToolResult, len(outcomes))
		for i, oc := range outcomes {
			results[i] = outcomeToToolResult(oc)
		}
		r.UpdateSummary(output.Summary{Results: results, DryRun: uf.DryRun})
		return nil
	}

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
		r.UpdateCancelled()
		return nil
	}

	selectedSet := make(map[string]struct{}, len(selected))
	for _, id := range selected {
		selectedSet[id] = struct{}{}
	}

	var results []output.ToolResult
	hasFailure := false
	updateIndex := 0
	updateTotal := len(selected)
	for _, oc := range outcomes {
		id := outcomeSelectionID(oc)
		display := oc.ToolName
		if display == "" {
			display = id
		}
		p, isPending := planByID[id]
		if !isPending {
			// Not planned → not selectable: carry its check outcome (current,
			// skipped, or failed) unchanged.
			results = append(results, outcomeToToolResult(oc))
			continue
		}
		if _, selectedNow := selectedSet[id]; !selectedNow {
			// Pending but not selected: report it under the distinct deselected
			// status, never silently drop it (spec Deselected Pending Tools
			// Reporting, PC2).
			results = append(results, output.ToolResult{
				Name:   display,
				Status: output.StatusDeselected,
			})
			continue
		}
		a, ok := adapterMap[id]
		if !ok {
			// An option that resolves to no adapter is an explicit failure,
			// never a silent drop (spec Unresolvable selection).
			results = append(results, output.ToolResult{
				Name:   display,
				Status: output.StatusFailed,
				Error:  fmt.Errorf("selected tool %q could not be resolved to an adapter", id),
			})
			hasFailure = true
			continue
		}
		updateIndex++
		res := executePlannedUpdate(gf, p, a, updateIndex, updateTotal, r, osName, allAdapters...)
		if res.Status == output.StatusFailed {
			hasFailure = true
		}
		results = append(results, res)
	}

	// A selected ID that matched no outcome/plan is unresolvable: surface it as
	// an explicit failure rather than silently ignoring the selection (spec
	// Unresolvable selection).
	for _, id := range selected {
		if _, ok := planByID[id]; !ok {
			results = append(results, output.ToolResult{
				Name:   id,
				Status: output.StatusFailed,
				Error:  fmt.Errorf("selected tool %q could not be resolved to an adapter", id),
			})
			hasFailure = true
		}
	}

	r.UpdateSummary(output.Summary{Results: results, DryRun: uf.DryRun})

	// CI mode: exit non-zero on failure.
	if gf.CI && hasFailure {
		return fmt.Errorf("update completed with failures")
	}

	return nil
}

// executePlannedUpdate is the single executor shared by the sequential and
// interactive update paths (design D2). It runs the confirmation gate and the
// update for one engine.PlannedUpdate, returning that tool's ToolResult. Risk
// command, privileges, trust, and the EnforceRisk policy all come from the
// plan — never re-derived here — so the two paths cannot diverge (design D3).
// A non-empty ManagerID means the row is an owned-package update whose real
// command risk must decide even for TrustOfficial tools.
func executePlannedUpdate(gf *GlobalFlags, p engine.PlannedUpdate, a adapters.Adapter, index, total int, r *output.Renderer, osName string, allAdapters ...[]adapters.Adapter) output.ToolResult {
	info := a.Info()

	if !gf.Quiet && total > 1 {
		r.Progress("Updating", index, total, info.Name)
	}

	riskLevel := security.ClassifyCommand(p.RiskCommand)
	decision := security.ConfirmAction(security.ConfirmConfig{
		ToolName:    p.ToolName,
		TrustLevel:  p.Trust,
		RiskLevel:   riskLevel,
		Command:     p.RiskCommand,
		Privileges:  p.Privileges,
		CI:          gf.CI,
		EnforceRisk: p.ManagerID != "",
	})

	switch decision {
	case security.ConfirmDeny:
		return output.ToolResult{
			Name:   p.ToolName,
			Status: output.StatusSkipped,
		}
	case security.ConfirmError:
		return output.ToolResult{
			Name:   p.ToolName,
			Status: output.StatusFailed,
			Error:  fmt.Errorf("CI mode: elevated risk requires confirmation"),
		}
	}

	// Owned tools with a PackageUpdater manager delegate to
	// updater.UpdatePackage(pkg); standalone tools run their own Update.
	owner := engine.ResolvingOwner(a, osName, allAdapters...)
	var result adapters.Result
	var updateErr error
	if owner != nil && info.Manager != nil && info.Manager[osName] != "" {
		if updater, ok := owner.(adapters.PackageUpdater); ok {
			pkg := engine.OwnedPackage(a, osName)
			if pkg != "" {
				result, updateErr = updater.UpdatePackage(pkg)
			} else {
				result, updateErr = a.Update(false)
			}
		} else {
			result, updateErr = a.Update(false)
		}
	} else {
		result, updateErr = a.Update(false)
	}

	if updateErr != nil {
		return output.ToolResult{
			Name:   p.ToolName,
			Status: output.StatusFailed,
			Error:  engine.TimeoutErr(p.ToolName, "update", updateErr),
			Stderr: updateErr.Error(),
		}
	}

	if result.Success {
		return output.ToolResult{
			Name:    p.ToolName,
			Status:  output.StatusUpdated,
			Version: result.After,
		}
	}

	errMsg := result.Error
	if errMsg == nil {
		errMsg = fmt.Errorf("update failed")
	}
	return output.ToolResult{
		Name:   p.ToolName,
		Status: output.StatusFailed,
		Error:  engine.TimeoutErr(p.ToolName, "update", errMsg),
		Stderr: errMsg.Error(),
	}
}

// outcomeSelectionID returns the canonical selection identity carried by a
// check outcome. The engine sets ToolID to adapters.ToolInfo.ID, falling back
// to Adapter.Name() — exactly toolSelectionID — so an outcome's ID resolves
// against adapterByID's key.
func outcomeSelectionID(oc engine.CheckOutcome) string {
	if oc.ToolID != "" {
		return oc.ToolID
	}
	return oc.ToolName
}

// resolveEffectiveUpdatePolicy returns the UpdatePolicy that governs whether
// an adapter's Update() runs.
func resolveEffectiveUpdatePolicy(a adapters.Adapter, osName string, allAdapters ...[]adapters.Adapter) adapters.UpdatePolicy {
	return engine.ResolveEffectiveUpdatePolicy(a, osName, allAdapters...)
}

// resolvingOwner returns the manager adapter that owns the given adapter on
// the given OS, or nil when the adapter has no resolving owner (standalone).
func resolvingOwner(a adapters.Adapter, osName string, allAdapters ...[]adapters.Adapter) adapters.Adapter {
	return engine.ResolvingOwner(a, osName, allAdapters...)
}
