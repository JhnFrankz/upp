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

			// Progress
			if !gf.Quiet && total > 1 {
				r.Progress("Updating", i+1, total, info.Name)
			}

			// Dry run: just show planned action
			if uf.DryRun {
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

			// Confirm if needed — always evaluate trust/risk, even in CI mode.
			riskLevel := security.ClassifyCommand(u.RiskCommand)
			decision := security.ConfirmAction(security.ConfirmConfig{
				ToolName:    info.Name,
				TrustLevel:  info.Trust,
				RiskLevel:   riskLevel,
				Command:     u.RiskCommand,
				Privileges:  info.Privileges,
				CI:          gf.CI,
				EnforceRisk: u.ManagerID != "",
			})

			switch decision {
			case security.ConfirmDeny:
				results[i] = output.ToolResult{
					Name:   info.Name,
					Status: output.StatusSkipped,
				}
				continue
			case security.ConfirmError:
				results[i] = output.ToolResult{
					Name:   info.Name,
					Status: output.StatusFailed,
					Error:  fmt.Errorf("CI mode: elevated risk requires confirmation"),
				}
				hasFailure = true
				continue
			}

			// Update: for owned tools with a PackageUpdater manager, delegate
			// to updater.UpdatePackage(pkg)
			owner := engine.ResolvingOwner(a, osName, allAdapters...)
			var result adapters.Result
			var updateErr error
			if owner != nil && a.Info().Manager != nil && a.Info().Manager[osName] != "" {
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
				results[i] = output.ToolResult{
					Name:   info.Name,
					Status: output.StatusFailed,
					Error:  engine.TimeoutErr(info.Name, "update", updateErr),
					Stderr: updateErr.Error(),
				}
				hasFailure = true
				continue
			}

			if result.Success {
				results[i] = output.ToolResult{
					Name:    info.Name,
					Status:  output.StatusUpdated,
					Version: result.After,
				}
			} else {
				errMsg := result.Error
				if errMsg == nil {
					errMsg = fmt.Errorf("update failed")
				}
				results[i] = output.ToolResult{
					Name:   info.Name,
					Status: output.StatusFailed,
					Error:  engine.TimeoutErr(info.Name, "update", errMsg),
					Stderr: errMsg.Error(),
				}
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
// a concurrent pre-check via eng.Check, a checkbox selector over the pending
// (StatusAvailable) tools, and the carried-outcome loop that updates only the user's selection.
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

	// Pending = tools with an update available, in group order.
	var pending []output.SelectOption
	for i, oc := range outcomes {
		if oc.Status == engine.StatusAvailable {
			name := oc.ToolName
			if name == "" {
				name = oc.ToolID
			}
			pending = append(pending, output.SelectOption{
				ID:      name,
				Label:   name,
				Version: fmt.Sprintf("%s → %s", oc.CurrentVersion, oc.LatestVersion),
				Group:   output.OwnerGroupLabel(grouped[i], osName, grouped),
			})
		}
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
	adapterMap := adapterByID(filteredAdapters)
	updateIndex := 0
	updateTotal := len(selected)
	for _, oc := range outcomes {
		name := oc.ToolName
		if name == "" {
			name = oc.ToolID
		}
		switch oc.Status {
		case engine.StatusAvailable:
			if _, ok := selectedSet[name]; !ok {
				continue // deselected: dropped, never processed
			}
			a, ok := adapterMap[name]
			if !ok {
				continue
			}
			updateIndex++
			hasFailure = processSelectedOutcome(gf, a, oc.RawUpdateInfo, updateIndex, updateTotal, r, &results, osName, allAdapters...) || hasFailure
		default:
			results = append(results, outcomeToToolResult(oc))
		}
	}

	r.UpdateSummary(output.Summary{Results: results, DryRun: uf.DryRun})

	// CI mode: exit non-zero on failure.
	if gf.CI && hasFailure {
		return fmt.Errorf("update completed with failures")
	}

	return nil
}

// processSelectedOutcome runs the per-tool confirm + policy gate + Update
// for one selected pending tool, appending its result.
func processSelectedOutcome(gf *GlobalFlags, a adapters.Adapter, updateInfo adapters.UpdateInfo, index, total int, r *output.Renderer, results *[]output.ToolResult, osName string, allAdapters ...[]adapters.Adapter) bool {
	info := a.Info()

	if !gf.Quiet && total > 1 {
		r.Progress("Updating", index, total, info.Name)
	}

	owner := engine.ResolvingOwner(a, osName, allAdapters...)

	var riskCommand string
	var enforceRisk bool
	if owner != nil {
		pkg := engine.OwnedPackage(a, osName)
		riskCommand = fmt.Sprintf("%s %s", engine.UpdateCmdName(owner.Name()), pkg)
		enforceRisk = true
	} else {
		riskCommand = info.Command
		if riskCommand == "" {
			riskCommand = info.Name + " update"
		}
	}
	riskLevel := security.ClassifyCommand(riskCommand)
	decision := security.ConfirmAction(security.ConfirmConfig{
		ToolName:    info.Name,
		TrustLevel:  info.Trust,
		RiskLevel:   riskLevel,
		Command:     riskCommand,
		Privileges:  info.Privileges,
		CI:          gf.CI,
		EnforceRisk: enforceRisk,
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
			Error:  fmt.Errorf("CI mode: elevated risk requires confirmation"),
		})
		return true
	}

	if engine.ResolveEffectiveUpdatePolicy(a, osName, allAdapters...) == adapters.PolicyGated && !updateInfo.UpdateAvailable {
		*results = append(*results, output.ToolResult{
			Name:    info.Name,
			Status:  output.StatusCurrent,
			Version: updateInfo.CurrentVersion,
		})
		return false
	}

	var result adapters.Result
	var err error
	if owner != nil && a.Info().Manager != nil && a.Info().Manager[osName] != "" {
		if updater, ok := owner.(adapters.PackageUpdater); ok {
			pkg := engine.OwnedPackage(a, osName)
			if pkg != "" {
				result, err = updater.UpdatePackage(pkg)
			} else {
				result, err = a.Update(false)
			}
		} else {
			result, err = a.Update(false)
		}
	} else {
		result, err = a.Update(false)
	}

	if err != nil {
		*results = append(*results, output.ToolResult{
			Name:   info.Name,
			Status: output.StatusFailed,
			Error:  engine.TimeoutErr(info.Name, "update", err),
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
		Error:  engine.TimeoutErr(info.Name, "update", errMsg),
		Stderr: errMsg.Error(),
	})
	return true
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
