package cli

import (
	"fmt"

	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/engine"
	"github.com/JhnFrankz/upp/internal/output"
)

// runUpdateSequential processes each filtered adapter: for owned tools under a
// package manager, it checks per-package availability via PackageChecker and
// updates via PackageUpdater (with EnforceRisk: true); for standalone tools, it
// runs standard Check and Update. Per-tool errors are isolated.
func runUpdateSequential(s *UpdateSession, filteredAdapters []adapters.Adapter) error {
	total := len(filteredAdapters)
	if total == 0 {
		s.Renderer.UpdateSummary(output.Summary{Results: nil, DryRun: s.UpdateFlags.DryRun})
		return nil
	}

	if err := s.Ctx.Err(); err != nil {
		return err
	}

	checkAdapters := engine.PrepareCheckAdapters(filteredAdapters, s.OSName, s.allAdaptersArg()...)
	checkAdapters, err := s.AuthorizeChecks(checkAdapters)
	if err != nil {
		return err
	}
	outcomes, err := s.Engine.Check(s.Ctx, checkAdapters, nil)
	if err != nil {
		return err
	}

	plan, err := s.Engine.Plan(outcomes, engine.Filter{})
	if err != nil {
		return err
	}

	updatesByTool := make(map[string]engine.PlannedUpdate, len(plan.Updates))
	for _, u := range plan.Updates {
		id := u.ToolID
		if id == "" {
			id = u.ToolName
		}
		if id != "" {
			updatesByTool[id] = u
		}
	}

	results := make([]output.ToolResult, total)
	hasFailure := false

	for i, a := range filteredAdapters {
		if err := s.Ctx.Err(); err != nil {
			return err
		}
		info := a.Info()
		oc := outcomes[i]

		switch oc.Status {
		case engine.StatusSkipped, engine.StatusUnknown:
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
			u, isUpdate := updatesByTool[toolSelectionID(a)]
			if !isUpdate {
				results[i] = output.ToolResult{
					Name:    info.Name,
					Status:  output.StatusCurrent,
					Version: oc.CurrentVersion,
				}
				continue
			}

			// Dry run: show the planned action only, never execute it.
			if s.UpdateFlags.DryRun {
				if !s.GlobalFlags.Quiet && total > 1 {
					s.Renderer.Progress("Updating", i+1, total, info.Name)
				}
				if oc.UpdateAvailable {
					if oc.CurrentVersion != "" && oc.LatestVersion != "" {
						if oc.CurrentVersion == oc.LatestVersion {
							s.Renderer.DryRunPlanned(fmt.Sprintf("%s (%s → update available)", info.Name, oc.CurrentVersion))
						} else {
							s.Renderer.DryRunPlanned(fmt.Sprintf("%s (%s → %s)", info.Name, oc.CurrentVersion, oc.LatestVersion))
						}
					} else {
						s.Renderer.DryRunPlanned(info.Name)
					}
					results[i] = output.ToolResult{
						Name:    info.Name,
						Status:  output.StatusAvailable,
						Version: fmt.Sprintf("%s → %s", oc.CurrentVersion, oc.LatestVersion),
					}
				} else {
					// Reaching this branch means the row IS in plan.Updates
					// (isUpdate was true above), so a real run WILL execute its
					// update command: the row is eligible by PolicyAlwaysUpdate
					// even though Check reported no version delta. Rendering
					// StatusCurrent here claimed "up to date" for a tool the real
					// run would update, contradicting the interactive selector,
					// which lists the same row as pending (spec command-interface:
					// --dry-run MUST report what would be updated).
					s.Renderer.DryRunPlanned(info.Name)
					results[i] = output.ToolResult{
						Name:    info.Name,
						Status:  output.StatusAvailable,
						Version: oc.CurrentVersion,
					}
				}
				continue
			}

			// The shared executor owns confirm + risk + privilege parity with
			// the interactive path (design D2/D3); the plan's PlannedUpdate is
			// the source of policy, not a re-derivation here.
			res := s.ExecutePlannedUpdate(u, a, i+1, total)
			s.Renderer.ToolLine(res)
			results[i] = res
			if results[i].Status == output.StatusFailed {
				hasFailure = true
			}
		default:
			results[i] = output.ToolResult{
				Name:   info.Name,
				Status: output.StatusSkipped,
			}
		}
	}

	summary := output.Summary{
		Results: results,
		DryRun:  s.UpdateFlags.DryRun,
	}
	s.Renderer.UpdateSummary(summary)

	// CI mode: exit non-zero on failure
	if s.GlobalFlags.CI && hasFailure {
		return fmt.Errorf("update completed with failures")
	}

	return nil
}

// runUpdateInteractive implements the TTY update flow:
// a concurrent pre-check via eng.Check, a checkbox selector over the
// plan-derived pending set (engine.Plan, so PolicyAlwaysUpdate tools appear
// even when current), and the carried-outcome loop that updates the user's
// selection while reporting every deselected pending tool distinctly.
func runUpdateInteractive(s *UpdateSession, deps updateDeps, filteredAdapters []adapters.Adapter) error {
	if err := s.Ctx.Err(); err != nil {
		return err
	}

	grouped := engine.GroupOrder(filteredAdapters, s.OSName, s.allAdaptersArg()...)
	checkAdapters := engine.PrepareCheckAdapters(grouped, s.OSName, s.allAdaptersArg()...)
	authorized, err := s.AuthorizeChecks(checkAdapters)
	if err != nil {
		return err
	}
	checkAdapters = authorized
	names := make([]string, len(checkAdapters))
	for i, a := range checkAdapters {
		names[i] = a.Info().Name
	}
	board := output.NewCheckBoard(s.Out, s.Renderer.Color(), names)
	board.Start()
	outcomes, _ := s.Engine.Check(s.Ctx, checkAdapters, func(prog engine.CheckProgress) {
		board.Complete(prog.Index, outcomeToToolResult(&prog.Outcome))
	})
	board.Finish()

	if err := s.Ctx.Err(); err != nil {
		return err
	}

	// The engine's plan is the single source of truth for identity, pending
	// eligibility, and execution metadata for the interactive path (design
	// D2/D3/D5, PC1): the pending set IS plan.Updates, so PolicyAlwaysUpdate
	// tools (e.g. brew, bun) appear even when currently up to date. Option IDs
	// are the canonical stable identity; labels are display-only (spec
	// Canonical Tool Selection Identity).
	plan, err := s.Engine.Plan(outcomes, engine.Filter{})
	if err != nil {
		return err
	}
	planByID := make(map[string]engine.PlannedUpdate, len(plan.Updates))
	for _, u := range plan.Updates {
		planByID[u.ToolID] = u
	}
	adapterMap := adapterByID(filteredAdapters)

	presentManagers := make(map[string]bool)
	for _, t := range grouped {
		if t.Info().Kind == adapters.KindManager {
			presentManagers[t.Name()] = true
			if id := t.Info().ID; id != "" {
				presentManagers[id] = true
			}
		}
	}

	pending := make([]output.SelectOption, 0, len(plan.Updates))
	for _, u := range plan.Updates {
		label := u.ToolName
		if label == "" {
			label = u.ToolID
		}
		var group string
		if a, ok := adapterMap[u.ToolID]; ok && a.Info().Kind != adapters.KindManager {
			if u.ManagerID != "" && presentManagers[u.ManagerID] {
				if mgr, ok := adapterMap[u.ManagerID]; ok {
					group = mgr.Info().Name
				}
			}
			if group == "" {
				group = engine.OwnerGroupLabelWithManagers(a, s.OSName, presentManagers, s.allAdaptersArg()...)
			}
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
		for i := range outcomes {
			results[i] = outcomeToToolResult(&outcomes[i])
		}
		s.Renderer.UpdateSummary(output.Summary{Results: results, DryRun: s.UpdateFlags.DryRun})
		return nil
	}

	if deps.selector == nil {
		deps.selector = func(opts []output.SelectOption) ([]string, bool) {
			res, err := output.NewCheckboxSelector(s.Out, s.In, opts).Run()
			if err != nil {
				return nil, true
			}
			return res.Selected, res.Canceled
		}
	}
	selected, canceled := deps.selector(pending)
	if canceled {
		s.Renderer.UpdateCancelled()
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
	for i := range outcomes {
		oc := &outcomes[i]
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
		if err := s.Ctx.Err(); err != nil {
			return err
		}
		updateIndex++
		res := s.ExecutePlannedUpdate(p, a, updateIndex, updateTotal)
		s.Renderer.ToolLine(res)
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

	s.Renderer.UpdateSummary(output.Summary{Results: results, DryRun: s.UpdateFlags.DryRun})

	// CI mode: exit non-zero on failure.
	if s.GlobalFlags.CI && hasFailure {
		return fmt.Errorf("update completed with failures")
	}

	return nil
}

// RunSequential delegates to runUpdateSequential.
func (s *UpdateSession) RunSequential(filteredAdapters []adapters.Adapter) error {
	return runUpdateSequential(s, filteredAdapters)
}

// RunInteractive delegates to runUpdateInteractive.
func (s *UpdateSession) RunInteractive(deps updateDeps, filteredAdapters []adapters.Adapter) error {
	return runUpdateInteractive(s, deps, filteredAdapters)
}

// outcomeSelectionID returns the canonical selection identity carried by a
// check outcome. The engine sets ToolID to adapters.ToolInfo.ID, falling back
// to Adapter.Name() — exactly toolSelectionID — so an outcome's ID resolves
// against adapterByID's key.
func outcomeSelectionID(oc *engine.CheckOutcome) string {
	if oc == nil {
		return ""
	}
	if oc.ToolID != "" {
		return oc.ToolID
	}
	return oc.ToolName
}
