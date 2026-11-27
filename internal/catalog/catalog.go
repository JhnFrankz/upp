// Package catalog provides static tool metadata, manager classifications, and
// platform compatibility tables with zero dependencies on execution, networking,
// or archive packages.
package catalog

import "github.com/JhnFrankz/upp/internal/platform"

// ToolSpec describes static metadata for an official tool.
type ToolSpec struct {
	ID        string
	IsManager bool
	Platforms []string
}

var (
	PlatformsLinux             = []string{platform.OSLinux}
	PlatformsLinuxMacOS        = []string{platform.OSLinux, platform.OSMacOS}
	PlatformsWindows           = []string{platform.OSWindows}
	PlatformsLinuxMacOSWindows = []string{platform.OSLinux, platform.OSMacOS, platform.OSWindows}

	toolSpecs = []ToolSpec{
		{ID: "apt", IsManager: true, Platforms: PlatformsLinux},
		{ID: "brew", IsManager: true, Platforms: PlatformsLinuxMacOS},
		{ID: "pacman", IsManager: true, Platforms: PlatformsLinux},
		{ID: "winget", IsManager: true, Platforms: PlatformsWindows},
		{ID: "scoop", IsManager: true, Platforms: PlatformsWindows},
		{ID: "nvm", Platforms: PlatformsLinuxMacOS},
		{ID: "npm", Platforms: PlatformsLinuxMacOSWindows},
		{ID: "pnpm", Platforms: PlatformsLinuxMacOSWindows},
		{ID: "bun", Platforms: PlatformsLinuxMacOSWindows},
		{ID: "uv", Platforms: PlatformsLinuxMacOSWindows},
		{ID: "gh", Platforms: PlatformsLinuxMacOSWindows},
		{ID: "docker", Platforms: PlatformsLinuxMacOSWindows},
		{ID: "go", Platforms: PlatformsLinuxMacOSWindows},
		{ID: "opencode", Platforms: PlatformsLinuxMacOSWindows},
	}

	toolSpecByID        map[string]ToolSpec
	canonicalManagerIDs []string
	linuxToolNames      []string
	macosToolNames      []string
	windowsToolNames    []string
)

func init() {
	toolSpecByID = make(map[string]ToolSpec, len(toolSpecs))
	for _, spec := range toolSpecs {
		toolSpecByID[spec.ID] = spec
		if spec.IsManager {
			canonicalManagerIDs = append(canonicalManagerIDs, spec.ID)
		}
		for _, osName := range spec.Platforms {
			switch osName {
			case platform.OSLinux:
				linuxToolNames = append(linuxToolNames, spec.ID)
			case platform.OSMacOS:
				macosToolNames = append(macosToolNames, spec.ID)
			case platform.OSWindows:
				windowsToolNames = append(windowsToolNames, spec.ID)
			}
		}
	}
}

// CanonicalManagerIDs returns a copy of the canonical official manager IDs in discovery order.
func CanonicalManagerIDs() []string {
	return append([]string(nil), canonicalManagerIDs...)
}

// ToolNamesForPlatform returns the slice of tool IDs supported on the given OS without heap allocations.
func ToolNamesForPlatform(os string) []string {
	switch os {
	case platform.OSLinux:
		return linuxToolNames
	case platform.OSMacOS:
		return macosToolNames
	case platform.OSWindows:
		return windowsToolNames
	default:
		return nil
	}
}

// IsOfficial reports whether the given tool ID belongs to an official adapter.
func IsOfficial(id string) bool {
	_, ok := toolSpecByID[id]
	return ok
}

// IsManager reports whether the given tool ID belongs to a declared manager-kind
// official adapter (apt, brew, pacman, winget, scoop).
func IsManager(id string) bool {
	spec, ok := toolSpecByID[id]
	return ok && spec.IsManager
}

// PlatformsFor returns the slice of platform names supported by the official
// adapter with the given tool ID, or nil if the tool is unknown.
func PlatformsFor(id string) []string {
	spec, ok := toolSpecByID[id]
	if !ok {
		return nil
	}
	return spec.Platforms
}
