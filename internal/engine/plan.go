package engine

import (
	"strings"

	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/security"
)

type adapterIndex map[string]adapters.Adapter

func buildAdapterIndex(adapterList []adapters.Adapter) adapterIndex {
	idx := make(adapterIndex, len(adapterList)*3)
	for _, a := range adapterList {
		indexAdapter(idx, a, true)
	}
	return idx
}

func setAdapterIndex(idx adapterIndex, key string, a adapters.Adapter, overwrite bool) {
	if overwrite {
		idx[key] = a
	} else if _, exists := idx[key]; !exists {
		idx[key] = a
	}
}

func indexAdapter(idx adapterIndex, a adapters.Adapter, overwrite bool) {
	if a == nil {
		return
	}
	info := a.Info()
	id := info.ID
	name := a.Name()

	var idLower string
	if id != "" {
		setAdapterIndex(idx, id, a, overwrite)
		idLower = strings.ToLower(id)
		if idLower != id {
			setAdapterIndex(idx, idLower, a, overwrite)
		}
	}

	if name != "" && name != id && name != idLower {
		setAdapterIndex(idx, name, a, overwrite)
		nameLower := strings.ToLower(name)
		if nameLower != name && nameLower != id && nameLower != idLower {
			setAdapterIndex(idx, nameLower, a, overwrite)
		}
	}
}

func (idx adapterIndex) find(toolID, toolName string) adapters.Adapter {
	if a, ok := idx[toolID]; ok {
		return a
	}
	if a, ok := idx[strings.ToLower(toolID)]; ok {
		return a
	}
	if toolName != "" {
		if a, ok := idx[toolName]; ok {
			return a
		}
		if a, ok := idx[strings.ToLower(toolName)]; ok {
			return a
		}
	}
	return nil
}

// resolvingOwnerIndex resolves the owning manager using the O(1) adapterIndex.
func resolvingOwnerIndex(a adapters.Adapter, info adapters.ToolInfo, osName, canonOS string, idx adapterIndex) adapters.Adapter {
	if a == nil {
		return nil
	}
	if custom, ok := a.(*adapters.CustomAdapter); ok {
		if m := custom.ManagerAdapter(); m != nil {
			return idx.find(m.Info().ID, m.Name())
		}
	}

	if ownerName := lookupOSMap(info.Manager, osName, canonOS); ownerName != "" {
		return idx.find(ownerName, ownerName)
	}
	return nil
}

// Plan formulates an executable UpdatePlan from check outcomes and filter options.
func (e *Engine) Plan(outcomes []CheckOutcome, filter Filter) (UpdatePlan, error) {
	n := len(outcomes)
	plan := UpdatePlan{
		Updates: make([]PlannedUpdate, 0, n),
		Skipped: make([]CheckOutcome, 0, n/4),
		Current: make([]CheckOutcome, 0, n),
		Failed:  make([]CheckOutcome, 0, n/4),
	}

	if len(outcomes) == 0 {
		return plan, nil
	}

	matcher := newOnlyMatcher(filter.Only)
	idx := buildAdapterIndex(e.allAdapters())
	canonOS := normalizeOSName(e.osName)

	for _, oc := range outcomes {
		if !matcher.matches(oc.ToolName, oc.ToolID) {
			continue
		}

		switch oc.Status {
		case StatusFailed:
			plan.Failed = append(plan.Failed, oc)
		case StatusSkipped, StatusUnknown:
			plan.Skipped = append(plan.Skipped, oc)
		case StatusAvailable, StatusCurrent:
			if planned, eligible := buildPlannedUpdate(oc, filter, idx, e.osName, canonOS); eligible {
				plan.Updates = append(plan.Updates, planned)
			} else {
				plan.Current = append(plan.Current, oc)
			}
		default:
			plan.Current = append(plan.Current, oc)
		}
	}

	return plan, nil
}

func buildPlannedUpdate(oc CheckOutcome, filter Filter, idx adapterIndex, osName, canonOS string) (PlannedUpdate, bool) {
	a := idx.find(oc.ToolID, oc.ToolName)
	var info adapters.ToolInfo
	if a != nil {
		info = a.Info()
	}

	owner := resolvingOwnerIndex(a, info, osName, canonOS, idx)
	policy := adapters.PolicyGated
	if a != nil {
		policy = info.UpdatePolicy
		if owner != nil {
			policy = owner.Info().UpdatePolicy
		}
	}

	if policy != adapters.PolicyAlwaysUpdate && !oc.UpdateAvailable && !filter.All {
		return PlannedUpdate{}, false
	}

	toolID, toolName := oc.ToolID, oc.ToolName
	trust := security.TrustCustomUntrusted
	var privileges []string
	var command string
	var kind adapters.Kind
	var selfCommand string

	if a != nil {
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

	managerID, packageName, riskCmd, privileges := resolveRiskDetails(
		toolID, toolName, command, selfCommand, kind, privileges, info, owner, osName, canonOS,
	)

	return PlannedUpdate{
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
	}, true
}

func resolveRiskDetails(
	toolID, toolName, command, selfCommand string,
	kind adapters.Kind,
	privileges []string,
	info adapters.ToolInfo,
	owner adapters.Adapter,
	osName, canonOS string,
) (managerID, packageName, riskCmd string, resolvedPrivileges []string) {
	resolvedPrivileges = privileges

	switch {
	case owner != nil:
		ownerInfo := owner.Info()
		managerID = ownerInfo.ID
		if managerID == "" {
			managerID = owner.Name()
		}
		packageName = lookupOSMap(info.ManagerPackage, osName, canonOS)
		if len(resolvedPrivileges) == 0 && len(ownerInfo.Privileges) > 0 {
			resolvedPrivileges = ownerInfo.Privileges
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
		if len(resolvedPrivileges) > 0 {
			managerID = toolID
		}
	default:
		// Standalone row: the declared command, or the "<name> update"
		// fallback (unchanged).
		riskCmd = standaloneRiskCommand(command, toolName)
	}

	if len(resolvedPrivileges) == 0 {
		resolvedPrivileges = security.DetectPrivileges(riskCmd)
	}
	return managerID, packageName, riskCmd, resolvedPrivileges
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
