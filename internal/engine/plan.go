package engine

import (
	"fmt"
	"strings"

	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/adapters/official"
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
		case StatusSkipped:
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
			trust := adapters.TrustOfficial
			var privileges []string
			var command string

			if a != nil {
				info := a.Info()
				if info.ID != "" {
					toolID = info.ID
				}
				if info.Name != "" {
					toolName = info.Name
				}
				trust = info.Trust
				privileges = info.Privileges
				command = info.Command
			}

			var managerID string
			var packageName string
			var riskCmd string

			if owner != nil {
				managerID = owner.Info().ID
				if managerID == "" {
					managerID = owner.Name()
				}
				packageName = OwnedPackage(a, e.osName)
				if len(privileges) == 0 && len(owner.Info().Privileges) > 0 {
					privileges = owner.Info().Privileges
				}
				if packageName != "" {
					riskCmd = fmt.Sprintf("%s %s", UpdateCmdName(owner.Name()), packageName)
				} else {
					riskCmd = fmt.Sprintf("%s %s", UpdateCmdName(owner.Name()), toolName)
				}
			} else {
				if command != "" {
					riskCmd = command
				} else {
					riskCmd = toolName + " update"
				}
			}

			if len(privileges) == 0 {
				privileges = detectPrivileges(riskCmd)
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

// detectPrivileges inspects a command string for privilege escalation tokens.
func detectPrivileges(cmd string) []string {
	lower := strings.ToLower(cmd)
	var privs []string
	if strings.Contains(lower, "sudo") {
		privs = append(privs, "sudo")
	}
	if strings.Contains(lower, "runas") || strings.Contains(lower, "admin") {
		privs = append(privs, "admin")
	}
	return privs
}
