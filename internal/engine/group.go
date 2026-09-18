package engine

import (
	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/adapters/official"
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

// GroupByOwner buckets the given adapters into manager-grouped buckets in
// canonical discovery order: (1) manager groups first, in official.AllAdapters
// order (apt, brew, winget, scoop, pacman); (2) each manager's owned tools;
// (3) standalone tools (tools with no resolving owner on this platform or
// whose manager was filtered out).
//
// It is display-only and does not perform I/O: a manager that was filtered
// out (not present in tools) never produces a phantom header — its owned
// tools fall through to the standalone group.
func GroupByOwner(tools []adapters.Adapter, osName string, allAdapters ...[]adapters.Adapter) []ToolGroup {
	presentManagers := make(map[string]adapters.Adapter)
	for _, a := range tools {
		if a.Info().Kind == adapters.KindManager {
			presentManagers[a.Name()] = a
			if id := a.Info().ID; id != "" {
				presentManagers[id] = a
			}
		}
	}

	ownerTools := make(map[string][]adapters.Adapter)
	var standalone []adapters.Adapter
	for _, a := range tools {
		if a.Info().Kind == adapters.KindManager {
			continue
		}
		ownerID := ownerIDOf(a, osName, allAdapters...)
		if ownerID != "" && presentManagers[ownerID] != nil {
			ownerTools[ownerID] = append(ownerTools[ownerID], a)
		} else {
			standalone = append(standalone, a)
		}
	}

	var groups []ToolGroup
	visitedManagers := make(map[string]bool)

	// (1) Manager groups in canonical official.AllAdapters() order
	for _, m := range official.AllAdapters() {
		mi := m.Info()
		if mi.Kind != adapters.KindManager {
			continue
		}
		mgrAdapter, present := presentManagers[mi.ID]
		if !present {
			mgrAdapter, present = presentManagers[m.Name()]
		}
		if !present {
			continue
		}
		if visitedManagers[mgrAdapter.Name()] {
			continue
		}
		visitedManagers[mgrAdapter.Name()] = true
		visitedManagers[mi.ID] = true
		visitedManagers[m.Name()] = true

		owned := ownerTools[mi.ID]
		if mgrAdapter.Name() != mi.ID && len(ownerTools[mgrAdapter.Name()]) > 0 {
			owned = append(owned, ownerTools[mgrAdapter.Name()]...)
		}

		header := managerDisplayName(mi.ID, allAdapters...)
		adaps := make([]adapters.Adapter, 0, 1+len(owned))
		adaps = append(adaps, mgrAdapter)
		adaps = append(adaps, owned...)

		groups = append(groups, ToolGroup{
			Header:   header,
			Manager:  mgrAdapter,
			Adapters: adaps,
		})
	}

	// Any manager in tools that wasn't in official.AllAdapters()
	for _, a := range tools {
		if a.Info().Kind != adapters.KindManager || visitedManagers[a.Name()] {
			continue
		}
		visitedManagers[a.Name()] = true
		owned := ownerTools[a.Name()]
		if id := a.Info().ID; id != "" && id != a.Name() {
			visitedManagers[id] = true
			if len(ownerTools[id]) > 0 {
				owned = append(owned, ownerTools[id]...)
			}
		}
		header := managerDisplayName(a.Name(), allAdapters...)
		adaps := make([]adapters.Adapter, 0, 1+len(owned))
		adaps = append(adaps, a)
		adaps = append(adaps, owned...)
		groups = append(groups, ToolGroup{
			Header:   header,
			Manager:  a,
			Adapters: adaps,
		})
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

// GroupOrder returns the given adapters reordered into group order (manager
// rows first in canonical AllAdapters order, then their owned tools, then
// standalone tools).
func GroupOrder(tools []adapters.Adapter, osName string, allAdapters ...[]adapters.Adapter) []adapters.Adapter {
	groups := GroupByOwner(tools, osName, allAdapters...)
	var ordered []adapters.Adapter
	for _, g := range groups {
		ordered = append(ordered, g.Adapters...)
	}
	return ordered
}

// OwnerGroupLabel returns the manager display label that owns a on osName, or
// "" when a is standalone or its owning manager is not among the given tools
// (so a filtered-out manager never creates a phantom group header).
// tools is the current run's adapter set used to decide presence.
func OwnerGroupLabel(a adapters.Adapter, osName string, tools []adapters.Adapter, allAdapters ...[]adapters.Adapter) string {
	if a == nil || a.Info().Kind == adapters.KindManager {
		return ""
	}
	presentManagers := make(map[string]bool)
	for _, t := range tools {
		if t.Info().Kind == adapters.KindManager {
			presentManagers[t.Name()] = true
			if id := t.Info().ID; id != "" {
				presentManagers[id] = true
			}
		}
	}
	ownerID := ownerIDOf(a, osName, allAdapters...)
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
	if owner := official.AdapterByName(ownerID); owner != nil {
		return owner.Info().Name
	}
	return ownerID
}

// ownerIDOf returns the ID of the manager currently owning a on osName, or ""
// when a is standalone. An official tool reads its canonical Info().Manager
// map; a custom tool exposes its injected manager via ManagerAdapter.
func ownerIDOf(a adapters.Adapter, osName string, allAdapters ...[]adapters.Adapter) string {
	if a == nil {
		return ""
	}
	if custom, ok := a.(*adapters.CustomAdapter); ok {
		if m := custom.ManagerAdapter(); m != nil {
			return m.Name()
		}
		return ""
	}
	canonOS := canonicalOS(osName)
	if a.Info().Manager != nil {
		if id := a.Info().Manager[osName]; id != "" {
			return id
		}
		if id := a.Info().Manager[canonOS]; id != "" {
			return id
		}
	}
	if owner := ResolvingOwner(a, osName, allAdapters...); owner != nil {
		return owner.Name()
	}
	return ""
}
