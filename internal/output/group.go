package output

import (
	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/adapters/official"
)

// Group is a section of the list/selector output: a manager header line
// (empty for standalone tools) and the list rows under it.
//
// For a manager group, Header is the manager's display name and Items holds
// the manager's own row first followed by the tools it owns on the current
// platform. For the trailing standalone group, Header is empty and Items
// holds the standalone tool rows.
type Group struct {
	Header string
	Items  []ListEntry
}

// GroupByOwner buckets the given adapters into manager-grouped output rows in
// canonical discovery order: (1) manager headers first, in official.AllAdapters
// order (apt, brew, winget, scoop); (2) each manager's owned tools (KindTool
// whose Manager[os] resolves to that manager); (3) standalone tools (KindTool
// with no resolving owner on this platform).
//
// It is display-only: it never reorders or mutates the input, and a manager
// that was filtered out (not present in tools) never produces a phantom header
// — its owned tools fall through to the standalone group. For each adapter it
// builds a ListEntry exactly like the list command did (Detect → Check),
// so the returned groups carry Status/Version for rendering.
func GroupByOwner(tools []adapters.Adapter, osName string) []Group {
	entryByID := make(map[string]ListEntry, len(tools))
	presentManagers := make(map[string]bool)
	for _, a := range tools {
		entryByID[a.Name()] = listEntryFor(a)
		if a.Info().Kind == adapters.KindManager {
			presentManagers[a.Name()] = true
		}
	}

	ownerItems := make(map[string][]ListEntry)
	var standalone []ListEntry
	for _, a := range tools {
		if a.Info().Kind == adapters.KindManager {
			continue
		}
		entry := entryByID[a.Name()]
		if ownerID := ownerIDOf(a, osName); ownerID != "" && presentManagers[ownerID] {
			ownerItems[ownerID] = append(ownerItems[ownerID], entry)
		} else {
			standalone = append(standalone, entry)
		}
	}

	var groups []Group
	for _, m := range official.AllAdapters() {
		mi := m.Info()
		if mi.Kind != adapters.KindManager || !presentManagers[mi.ID] {
			continue
		}
		items := make([]ListEntry, 0, 1+len(ownerItems[mi.ID]))
		items = append(items, entryByID[mi.ID]) // the manager's own row leads its group
		items = append(items, ownerItems[mi.ID]...)
		groups = append(groups, Group{Header: mi.Name, Items: items})
	}
	if len(standalone) > 0 {
		groups = append(groups, Group{Items: standalone})
	}
	return groups
}

// GroupOrder returns the given adapters reordered into group order (manager
// rows first in canonical AllAdapters order, then their owned tools, then
// standalone tools) WITHOUT computing status. It is used by the interactive
// update board/selector so display order is grouped while each tool's status
// is computed exactly once by runChecks. Filtered-out managers never produce a
// phantom group: owned tools whose manager is absent fall to the standalone
// tail, preserving the flat --only/--skip round-trip.
func GroupOrder(tools []adapters.Adapter, osName string) []adapters.Adapter {
	presentManagers := make(map[string]bool)
	for _, a := range tools {
		if a.Info().Kind == adapters.KindManager {
			presentManagers[a.Name()] = true
		}
	}

	var ordered []adapters.Adapter
	for _, m := range official.AllAdapters() {
		mi := m.Info()
		if mi.Kind != adapters.KindManager || !presentManagers[mi.ID] {
			continue
		}
		for _, a := range tools {
			if a.Name() == mi.ID {
				ordered = append(ordered, a)
				break
			}
		}
		for _, a := range tools {
			if a.Info().Kind == adapters.KindManager {
				continue
			}
			if ownerIDOf(a, osName) == mi.ID {
				ordered = append(ordered, a)
			}
		}
	}

	for _, a := range tools {
		if a.Info().Kind == adapters.KindManager {
			continue
		}
		if ownerID := ownerIDOf(a, osName); ownerID != "" && presentManagers[ownerID] {
			continue
		}
		ordered = append(ordered, a)
	}
	return ordered
}

// OwnerGroupLabel returns the manager display label that owns a on osName, or
// "" when a is standalone or its owning manager is not among the given tools
// (so a filtered-out manager never creates a phantom group header in the
// selector). tools is the current run's adapter set used to decide presence.
func OwnerGroupLabel(a adapters.Adapter, osName string, tools []adapters.Adapter) string {
	presentManagers := make(map[string]bool)
	for _, t := range tools {
		if t.Info().Kind == adapters.KindManager {
			presentManagers[t.Name()] = true
		}
	}
	ownerID := ownerIDOf(a, osName)
	if ownerID == "" || !presentManagers[ownerID] {
		return ""
	}
	return managerDisplayName(ownerID)
}

// managerDisplayName resolves an owner manager ID to its display name, falling
// back to the ID itself for an unknown owner (custom-injected manager).
func managerDisplayName(ownerID string) string {
	if owner := official.AdapterByName(ownerID); owner != nil {
		return owner.Info().Name
	}
	return ownerID
}

// ownerIDOf returns the ID of the manager currently owning a on osName, or ""
// when a is standalone. An official tool reads its canonical Info().Manager
// map; a custom tool exposes its injected manager via ManagerAdapter.
func ownerIDOf(a adapters.Adapter, osName string) string {
	if custom, ok := a.(*adapters.CustomAdapter); ok {
		if m := custom.ManagerAdapter(); m != nil {
			return m.Name()
		}
		return ""
	}
	return a.Info().Manager[osName]
}

// listEntryFor builds the ListEntry for a single adapter, matching the list
// command's Detect → Check logic: an uninstalled tool is Skipped with no
// version; an installed tool is Current with its detected version (a failed
// Check leaves the version empty).
func listEntryFor(a adapters.Adapter) ListEntry {
	info := a.Info()
	status := StatusSkipped
	version := ""
	if a.Detect() {
		status = StatusCurrent
		if updateInfo, err := a.Check(); err == nil {
			version = updateInfo.CurrentVersion
		}
	}
	return ListEntry{
		ID:      info.ID,
		Name:    info.Name,
		Status:  status,
		Version: version,
	}
}
