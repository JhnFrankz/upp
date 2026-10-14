package cli

import (
	"github.com/JhnFrankz/upp/internal/engine"
	"github.com/JhnFrankz/upp/internal/output"
)

// presentGroups maps domain ToolGroup slices and their corresponding CheckOutcome
// results into presentation output.Group items with status and version populated,
// without performing any command execution or I/O.
func presentGroups(toolGroups []engine.ToolGroup, outcomes []engine.CheckOutcome) []output.Group {
	outcomeByID := make(map[string]*engine.CheckOutcome, len(outcomes)*2)
	for i := range outcomes {
		oc := &outcomes[i]
		if oc.ToolID != "" {
			outcomeByID[oc.ToolID] = oc
		}
		if oc.ToolName != "" {
			outcomeByID[oc.ToolName] = oc
		}
	}

	groups := make([]output.Group, 0, len(toolGroups))
	for _, tg := range toolGroups {
		items := make([]output.ListEntry, 0, len(tg.Adapters))
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

			status := output.StatusSkipped
			version := ""
			if ok {
				switch oc.Status {
				case engine.StatusAvailable:
					status = output.StatusAvailable
					version = oc.CurrentVersion
				case engine.StatusCurrent:
					status = output.StatusCurrent
					version = oc.CurrentVersion
				case engine.StatusSkipped, engine.StatusUnknown:
					status = output.StatusSkipped
				case engine.StatusFailed:
					status = output.StatusFailed
				default:
					status = output.StatusSkipped
				}
			}

			items = append(items, output.ListEntry{
				ID:      id,
				Name:    name,
				Status:  status,
				Version: version,
			})
		}
		groups = append(groups, output.Group{
			Header: tg.Header,
			Items:  items,
		})
	}
	return groups
}
