package output

import (
	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/engine"
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

// PresentGroups maps domain ToolGroup slices and their corresponding CheckOutcome
// results into presentation Group items with status and version populated,
// without performing any command execution or I/O.
func PresentGroups(toolGroups []engine.ToolGroup, outcomes []engine.CheckOutcome) []Group {
	outcomeByID := make(map[string]engine.CheckOutcome, len(outcomes))
	for _, oc := range outcomes {
		if oc.ToolID != "" {
			outcomeByID[oc.ToolID] = oc
		}
		if oc.ToolName != "" {
			outcomeByID[oc.ToolName] = oc
		}
	}

	groups := make([]Group, 0, len(toolGroups))
	for _, tg := range toolGroups {
		items := make([]ListEntry, 0, len(tg.Adapters))
		for _, a := range tg.Adapters {
			info := a.Info()
			id := info.ID
			if id == "" {
				id = a.Name()
			}
			name := info.Name
			if name == "" {
				name = a.Name()
			}

			oc, ok := outcomeByID[id]
			if !ok {
				oc, ok = outcomeByID[a.Name()]
			}

			status := StatusSkipped
			version := ""
			if ok {
				switch oc.Status {
				case engine.StatusAvailable:
					status = StatusAvailable
					version = oc.CurrentVersion
				case engine.StatusCurrent:
					status = StatusCurrent
					version = oc.CurrentVersion
				case engine.StatusSkipped, engine.StatusUnknown:
					status = StatusSkipped
				case engine.StatusFailed:
					status = StatusFailed
				default:
					status = StatusSkipped
				}
			}

			items = append(items, ListEntry{
				ID:      id,
				Name:    name,
				Status:  status,
				Version: version,
			})
		}
		groups = append(groups, Group{
			Header: tg.Header,
			Items:  items,
		})
	}
	return groups
}

// GroupByOwner is a backward-compatible presentation helper that delegates grouping
// to engine.GroupByOwner without subprocess I/O.
func GroupByOwner(tools []adapters.Adapter, osName string) []Group {
	return PresentGroups(engine.GroupByOwner(tools, osName), nil)
}

// GroupOrder delegates to engine.GroupOrder.
// Deprecated: use engine.GroupOrder.
func GroupOrder(tools []adapters.Adapter, osName string, allAdapters ...[]adapters.Adapter) []adapters.Adapter {
	return engine.GroupOrder(tools, osName, allAdapters...)
}

// OwnerGroupLabel delegates to engine.OwnerGroupLabel.
// Deprecated: use engine.OwnerGroupLabel.
func OwnerGroupLabel(a adapters.Adapter, osName string, tools []adapters.Adapter, allAdapters ...[]adapters.Adapter) string {
	return engine.OwnerGroupLabel(a, osName, tools, allAdapters...)
}
