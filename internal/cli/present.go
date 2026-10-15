package cli

import (
	"github.com/JhnFrankz/upp/internal/doctor"
	"github.com/JhnFrankz/upp/internal/engine"
	"github.com/JhnFrankz/upp/internal/output"
	"github.com/JhnFrankz/upp/internal/uninstall"
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

// presentDoctorResults maps doctor domain models into presentation view models and renders them.
func presentDoctorResults(r *output.Renderer, results []doctor.CheckResult, quiet, verbose bool) {
	items := make([]output.DiagnosticResult, len(results))
	for i, res := range results {
		var sev output.DiagnosticSeverity
		switch res.Status {
		case doctor.SeverityOK:
			sev = output.DiagnosticSeverityOK
		case doctor.SeverityWarn:
			sev = output.DiagnosticSeverityWarn
		case doctor.SeverityError:
			sev = output.DiagnosticSeverityError
		default:
			sev = output.DiagnosticSeverityWarn
		}
		items[i] = output.DiagnosticResult{
			Name:     res.Name,
			Status:   sev,
			Category: res.Category,
			Message:  res.Message,
			Detail:   res.Detail,
			FixHint:  res.FixHint,
		}
	}
	r.DoctorResults(items, quiet, verbose)
}

// presentUninstallPlan maps uninstall domain models into presentation view models and renders them.
func presentUninstallPlan(r *output.Renderer, targets []uninstall.Target) {
	items := make([]output.TargetItem, len(targets))
	for i, t := range targets {
		items[i] = output.TargetItem{
			Type:   string(t.Type),
			Path:   t.Path,
			Exists: t.Exists,
		}
	}
	r.UninstallPlan(items)
}
