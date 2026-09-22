package engine

import (
	"strings"

	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/adapters/official"
	"github.com/JhnFrankz/upp/internal/security"
)

// ResolveEffectiveUpdatePolicy returns the effective UpdatePolicy that governs
// whether an adapter update is executed. For an owned tool, the owner's policy
// governs and the tool's own declared policy is inert. Otherwise, the adapter's
// own declared policy applies.
func ResolveEffectiveUpdatePolicy(a adapters.Adapter, osName string, allAdapters ...[]adapters.Adapter) adapters.UpdatePolicy {
	if a == nil {
		return adapters.PolicyGated
	}
	if owner := ResolvingOwner(a, osName, allAdapters...); owner != nil {
		return owner.Info().UpdatePolicy
	}
	return a.Info().UpdatePolicy
}

// findAdapter searches adapterList for an adapter whose Name or ID matches toolID or toolName.
func findAdapter(adapterList []adapters.Adapter, toolID, toolName string) adapters.Adapter {
	for _, a := range adapterList {
		info := a.Info()
		if info.ID == toolID || a.Name() == toolID || (toolName != "" && (info.ID == toolName || a.Name() == toolName)) {
			return a
		}
	}
	return nil
}

// Plan formulates an executable UpdatePlan from check outcomes and filter options.
func (e *Engine) Plan(outcomes []CheckOutcome, filter Filter) (UpdatePlan, error) {
	plan := UpdatePlan{
		Updates: make([]PlannedUpdate, 0),
		Skipped: make([]CheckOutcome, 0),
		Current: make([]CheckOutcome, 0),
		Failed:  make([]CheckOutcome, 0),
	}

	if len(outcomes) == 0 {
		return plan, nil
	}

	var onlySet map[string]struct{}
	if len(filter.Only) > 0 {
		onlySet = make(map[string]struct{}, len(filter.Only))
		for _, name := range filter.Only {
			onlySet[strings.ToLower(strings.TrimSpace(name))] = struct{}{}
		}
	}

	var allAdapters []adapters.Adapter
	if e.adapters != nil {
		allAdapters = e.adapters
	} else {
		allAdapters, _ = e.Resolve(Filter{})
	}

	for _, oc := range outcomes {
		if len(onlySet) > 0 {
			nameLower := strings.ToLower(oc.ToolName)
			idLower := strings.ToLower(oc.ToolID)
			_, byName := onlySet[nameLower]
			_, byID := onlySet[idLower]
			if !byName && !byID {
				continue
			}
		}

		switch oc.Status {
		case StatusFailed:
			plan.Failed = append(plan.Failed, oc)
		case StatusSkipped, StatusUnknown:
			plan.Skipped = append(plan.Skipped, oc)
		case StatusAvailable, StatusCurrent:
			a := findAdapter(allAdapters, oc.ToolID, oc.ToolName)
			if a == nil {
				a = official.AdapterByName(oc.ToolID)
				if a == nil && oc.ToolName != "" {
					a = official.AdapterByName(oc.ToolName)
				}
			}

			owner := ResolvingOwner(a, e.osName, allAdapters)
			policy := ResolveEffectiveUpdatePolicy(a, e.osName, allAdapters)

			isEligible := false
			if policy == adapters.PolicyAlwaysUpdate {
				isEligible = true
			} else if oc.UpdateAvailable || filter.All {
				isEligible = true
			}

			if !isEligible {
				plan.Current = append(plan.Current, oc)
				continue
			}

			toolID := oc.ToolID
			toolName := oc.ToolName
			trust := security.TrustOfficial
			var privileges []string
			var command string
			var kind adapters.Kind
			var selfCommand string

			var info adapters.ToolInfo
			if a != nil {
				info = a.Info()
				if info.ID != "" {
					toolID = info.ID
				}
				if info.Name != "" {
					toolName = info.Name
				}
				trust = info.Trust
				privileges = info.Privileges
				command = info.Command
				kind = info.Kind
				selfCommand = info.SelfUpdateCommand
			}

			var managerID string
			var packageName string
			var riskCmd string

			switch {
			case owner != nil:
				ownerInfo := owner.Info()
				managerID = ownerInfo.ID
				if managerID == "" {
					managerID = owner.Name()
				}
				packageName = OwnedPackage(a, e.osName)
				if len(privileges) == 0 && len(ownerInfo.Privileges) > 0 {
					privileges = ownerInfo.Privileges
				}
				switch {
				case packageName != "" && ownerInfo.PackageUpdateCommand != "":
					// Owned-package row: the manager's real per-package command
					// (design D3), rendered from the same declaration the
					// manager's UpdatePackage() executes.
					riskCmd = adapters.RenderPackageCommand(ownerInfo.PackageUpdateCommand, packageName)
				case info.Manager == nil || ownerInfo.SelfUpdateCommand != "":
					// Custom manager-delegated row (or fallback when manager has no per-package command):
					// custom.Update() delegates to owner.Update(), so the manager's
					// real self-update command is what actually executes (design D3).
					riskCmd = ownerInfo.SelfUpdateCommand
				default:
					// Official owned tool with no declared package (unreachable
					// today): keep the standalone fallback rule.
					riskCmd = standaloneRiskCommand(command, toolName)
				}
			case kind == adapters.KindManager:
				// Manager self-row: the manager's real self-update command
				// (design D3). Mark the row manager-scoped iff the manager
				// declares privileges, so the gate's EnforceRisk path applies to
				// privileged managers (pacman) only — apt/brew/winget keep their
				// byte-identical auto-proceed decision (design D4).
				riskCmd = selfCommand
				if len(privileges) > 0 {
					managerID = toolID
				}
			default:
				// Standalone row: the declared command, or the "<name> update"
				// fallback (unchanged).
				riskCmd = standaloneRiskCommand(command, toolName)
			}

			if len(privileges) == 0 {
				privileges = security.DetectPrivileges(riskCmd)
			}

			plan.Updates = append(plan.Updates, PlannedUpdate{
				ToolID:         toolID,
				ToolName:       toolName,
				ManagerID:      managerID,
				PackageName:    packageName,
				UpdatePolicy:   policy,
				CurrentVersion: oc.CurrentVersion,
				LatestVersion:  oc.LatestVersion,
				RiskCommand:    riskCmd,
				Privileges:     privileges,
				Trust:          trust,
			})
		default:
			plan.Current = append(plan.Current, oc)
		}
	}

	return plan, nil
}

// standaloneRiskCommand returns a standalone row's risk command: the adapter's
// declared command when present, otherwise the "<name> update" fallback. It is
// also the fallback for an official owned row whose manager declares no
// per-package command (unreachable today).
func standaloneRiskCommand(command, toolName string) string {
	if command != "" {
		return command
	}
	return toolName + " update"
}
