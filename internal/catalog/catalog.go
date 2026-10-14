// Package catalog provides static tool metadata, manager classifications, and
// platform compatibility tables with zero dependencies on execution, networking,
// or archive packages.
package catalog

import "github.com/JhnFrankz/upp/internal/platform"

var (
	allOfficialToolNames = []string{
		"apt", "brew", "bun", "docker", "gh", "go", "npm",
		"nvm", "opencode", "pacman", "pnpm", "scoop", "uv", "winget",
	}

	linuxToolNames = []string{
		"apt", "brew", "pacman", "nvm", "npm",
		"pnpm", "bun", "uv", "gh", "docker",
		"go", "opencode",
	}

	macosToolNames = []string{
		"brew", "nvm", "npm", "pnpm", "bun",
		"uv", "gh", "docker", "go", "opencode",
	}

	windowsToolNames = []string{
		"winget", "scoop", "npm", "pnpm",
		"bun", "uv", "gh", "docker", "go",
		"opencode",
	}

	PlatformsLinux             = []string{platform.OSLinux}
	PlatformsLinuxMacOS        = []string{platform.OSLinux, platform.OSMacOS}
	PlatformsWindows           = []string{platform.OSWindows}
	PlatformsLinuxMacOSWindows = []string{platform.OSLinux, platform.OSMacOS, platform.OSWindows}
)

// AllToolNames returns a copy of all official tool IDs.
func AllToolNames() []string {
	return append([]string(nil), allOfficialToolNames...)
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
	switch id {
	case "apt", "brew", "bun", "docker", "gh", "go", "npm", "nvm", "opencode", "pacman", "pnpm", "scoop", "uv", "winget":
		return true
	default:
		return false
	}
}

// IsManager reports whether the given tool ID belongs to a declared manager-kind
// official adapter (apt, brew, pacman, winget, scoop).
func IsManager(id string) bool {
	switch id {
	case "apt", "brew", "pacman", "winget", "scoop":
		return true
	default:
		return false
	}
}

// PlatformsFor returns the slice of platform names supported by the official
// adapter with the given tool ID, or nil if the tool is unknown.
func PlatformsFor(id string) []string {
	switch id {
	case "apt", "pacman":
		return PlatformsLinux
	case "brew", "nvm":
		return PlatformsLinuxMacOS
	case "winget", "scoop":
		return PlatformsWindows
	case "bun", "docker", "gh", "go", "npm", "opencode", "pnpm", "uv":
		return PlatformsLinuxMacOSWindows
	default:
		return nil
	}
}
