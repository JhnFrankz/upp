package official

import (
	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/platform"
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

// runtimeGOOSToPlatform maps the runtime.GOOS value to upp's canonical
// platform key (platform constants). This is the WU1-documented gotcha:
// ResolveOwner is keyed by PLATFORM constants (linux/macos/windows), while
// runtime.GOOS returns "darwin" on macOS. Translating here guarantees the
// delegated Update() resolves the owner correctly on every OS.
func runtimeGOOSToPlatform(goos string) string {
	switch goos {
	case "darwin":
		return platform.OSMacOS
	case "linux":
		return platform.OSLinux
	case "windows":
		return platform.OSWindows
	default:
		return goos
	}
}
