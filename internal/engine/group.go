package engine

import (
	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/catalog"
)

// ToolGroup represents a logical group of tools: a manager header line
// and its associated adapters.
//
// For a manager group, Header is the manager's display name, Manager is
// the manager adapter itself, and Adapters holds the manager adapter first
// followed by the tools it owns on the current platform. For the trailing
// standalone group, Header is empty, Manager is nil, and Adapters holds
// the standalone tool adapters.
type ToolGroup struct {
	Header   string
	Manager  adapters.Adapter
	Adapters []adapters.Adapter
}

var canonicalManagerIDs = catalog.CanonicalManagerIDs()

// GroupByOwner buckets the given adapters into manager-grouped buckets in
// canonical discovery order: (1) manager groups first, in catalog.CanonicalManagerIDs()
// order (apt, brew, pacman, winget, scoop); (2) each manager's owned tools;
// (3) standalone tools (tools with no resolving owner on this platform or
// whose manager was filtered out).
//
// It is display-only and does not perform I/O: a manager that was filtered
// out (not present in tools) never produces a phantom header — its owned
// tools fall through to the standalone group.
func GroupByOwner(tools []adapters.Adapter, osName string, allAdapters ...[]adapters.Adapter) []ToolGroup {
	infos := make([]adapters.ToolInfo, len(tools))
	presentManagers := make(map[string]adapters.Adapter)
	for i, a := range tools {
		infos[i] = a.Info()
		if infos[i].Kind == adapters.KindManager {
			presentManagers[a.Name()] = a
			if id := infos[i].ID; id != "" {
				presentManagers[id] = a
			}
		}
	}

	if len(presentManagers) == 0 {
		return []ToolGroup{{
			Header:   "",
			Manager:  nil,
			Adapters: append([]adapters.Adapter(nil), tools...),
		}}
	}

	var allList []adapters.Adapter
	if len(allAdapters) > 0 {
		allList = allAdapters[0]
	}
	canonOS := normalizeOSName(osName)
	var idx adapterIndex
	if len(allList) > 0 {
		idx = buildAdapterIndex(allList)
	}

	ownerTools := make(map[string][]adapters.Adapter)
	var standalone []adapters.Adapter
	for i, a := range tools {
		if infos[i].Kind == adapters.KindManager {
			continue
		}
		ownerID := ownerIDForAdapter(a, infos[i], osName, canonOS, idx, allList == nil)
		if ownerID != "" && presentManagers[ownerID] != nil {
			ownerTools[ownerID] = append(ownerTools[ownerID], a)
		} else {
			standalone = append(standalone, a)
		}
	}

	groups := make([]ToolGroup, 0, len(canonicalManagerIDs)+1)
	visitedManagers := make(map[string]bool, len(canonicalManagerIDs)*2)

	// (1) Manager groups in canonical order
	for _, mgrID := range canonicalManagerIDs {
		mgrAdapter, present := presentManagers[mgrID]
		if !present || visitedManagers[mgrAdapter.Name()] {
			continue
		}
		mgrInfo := mgrAdapter.Info()
		visitedManagers[mgrAdapter.Name()] = true
		visitedManagers[mgrID] = true
		visitedManagers[mgrInfo.ID] = true

		owned := collectOwnedForManager(ownerTools, mgrID, mgrAdapter.Name())
		groups = appendManagerGroup(groups, mgrAdapter, mgrInfo, owned)
	}

	// Any manager in tools that wasn't in catalog.CanonicalManagerIDs()
	for i, a := range tools {
		info := infos[i]
		if info.Kind != adapters.KindManager || visitedManagers[a.Name()] {
			continue
		}
		visitedManagers[a.Name()] = true
		if info.ID != "" {
			visitedManagers[info.ID] = true
		}
		owned := collectOwnedForManager(ownerTools, info.ID, a.Name())
		groups = appendManagerGroup(groups, a, info, owned)
	}

	// (2) Standalone tools
	if len(standalone) > 0 {
		groups = append(groups, ToolGroup{
			Header:   "",
			Manager:  nil,
			Adapters: standalone,
		})
	}

	return groups
}

func collectOwnedForManager(ownerTools map[string][]adapters.Adapter, id, name string) []adapters.Adapter {
	if id == "" || id == name {
		return ownerTools[name]
	}
	primary := ownerTools[id]
	secondary := ownerTools[name]
	if len(secondary) == 0 {
		return primary
	}
	owned := make([]adapters.Adapter, 0, len(primary)+len(secondary))
	owned = append(owned, primary...)
	owned = append(owned, secondary...)
	return owned
}

func appendManagerGroup(groups []ToolGroup, mgr adapters.Adapter, mgrInfo adapters.ToolInfo, owned []adapters.Adapter) []ToolGroup {
	header := mgrInfo.Name
	if header == "" {
		header = mgr.Name()
	}
	adaps := make([]adapters.Adapter, 0, 1+len(owned))
	adaps = append(adaps, mgr)
	adaps = append(adaps, owned...)
	return append(groups, ToolGroup{
		Header:   header,
		Manager:  mgr,
		Adapters: adaps,
	})
}

// GroupOrder returns the given adapters reordered into group order (manager
// rows first in canonical AllAdapters order, then their owned tools, then
// standalone tools).
func GroupOrder(tools []adapters.Adapter, osName string, allAdapters ...[]adapters.Adapter) []adapters.Adapter {
	groups := GroupByOwner(tools, osName, allAdapters...)
	ordered := make([]adapters.Adapter, 0, len(tools))
	for _, g := range groups {
		ordered = append(ordered, g.Adapters...)
	}
	return ordered
}

// OwnerGroupLabelWithManagers returns the manager display label that owns a on osName,
// using a precomputed map of present manager names and IDs.
func OwnerGroupLabelWithManagers(a adapters.Adapter, osName string, presentManagers map[string]bool, allAdapters ...[]adapters.Adapter) string {
	if a == nil || a.Info().Kind == adapters.KindManager {
		return ""
	}
	var list []adapters.Adapter
	if len(allAdapters) > 0 {
		list = allAdapters[0]
	}
	ownerID := ownerIDOfSlice(a, osName, list)
	if ownerID == "" || !presentManagers[ownerID] {
		return ""
	}
	return managerDisplayName(ownerID, allAdapters...)
}

// managerDisplayName resolves an owner manager ID to its display name, falling
// back to the ID itself for an unknown owner.
func managerDisplayName(ownerID string, allAdapters ...[]adapters.Adapter) string {
	if len(allAdapters) > 0 && allAdapters[0] != nil {
		if owner := adapterByName(allAdapters[0], ownerID); owner != nil {
			return owner.Info().Name
		}
	}
	if owner := adapters.AdapterByName(ownerID); owner != nil {
		return owner.Info().Name
	}
	return ownerID
}

func ownerIDOfSlice(a adapters.Adapter, osName string, allAdapters []adapters.Adapter) string {
	if a == nil {
		return ""
	}
	canonOS := normalizeOSName(osName)
	var idx adapterIndex
	if len(allAdapters) > 0 {
		idx = buildAdapterIndex(allAdapters)
	}
	return ownerIDForAdapter(a, a.Info(), osName, canonOS, idx, allAdapters == nil)
}

func ownerIDForAdapter(a adapters.Adapter, info adapters.ToolInfo, osName, canonOS string, idx adapterIndex, allowGlobalFallback bool) string {
	if a == nil {
		return ""
	}
	if custom, ok := a.(*adapters.CustomAdapter); ok {
		if m := custom.ManagerAdapter(); m != nil {
			return m.Name()
		}
		return ""
	}
	if id := lookupOSMap(info.Manager, osName, canonOS); id != "" {
		return id
	}
	if idx != nil {
		if owner := resolvingOwnerIndex(a, info, osName, canonOS, idx); owner != nil {
			return owner.Name()
		}
		return ""
	}
	if allowGlobalFallback {
		if owner := resolvingOwnerSlice(a, osName, nil); owner != nil {
			return owner.Name()
		}
	}
	return ""
}
