package official

import (
	"github.com/JhnFrankz/upp/internal/adapters"
)

// ResolveOwner returns the manager adapter that owns the given tool on the
// given OS, or nil when the tool has no resolving owner on that platform
// (standalone). It is a PURE function: it reads the tool adapter's own
// canonical ToolInfo.Manager map (platform -> owning manager ID) and does not
// mutate any state. A tool with no Manager[os] entry (e.g. go on Linux) stays
// standalone and returns nil, so its own adapter Update() path runs.
func ResolveOwner(tool, os string) adapters.Adapter {
	a := AdapterByName(tool)
	if a == nil {
		return nil
	}
	ownerID := a.Info().Manager[os] // nil map -> "" (safe on a nil map read)
	if ownerID == "" {
		return nil
	}
	return AdapterByName(ownerID)
}
