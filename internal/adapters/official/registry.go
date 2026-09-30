package official

import (
	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/platform"
)

// OwnerMetadataSummary summarizes the ownership model across all official
// adapters: how many declare KindManager and how many declare KindTool, plus
// the total. It is derived from the canonical adapter Info() Kind, so it stays
// in sync with the declared model (spec Manager Owned-Tool Cardinality /
// design registry convenience) without a separate hardcoded count.
type OwnerMetadataSummary struct {
	Total    int
	Managers int
	Tools    int
}

// OwnerMetadata returns the per-Kind counts across AllAdapters.
func OwnerMetadata() OwnerMetadataSummary {
	meta := OwnerMetadataSummary{Total: len(AllAdapters())}
	for _, a := range AllAdapters() {
		if a.Info().Kind == adapters.KindManager {
			meta.Managers++
		} else {
			meta.Tools++
		}
	}
	return meta
}

var (
	aptAdapter      = &AptAdapter{}
	brewAdapter     = &BrewAdapter{}
	bunAdapter      = &BunAdapter{}
	dockerAdapter   = &DockerAdapter{}
	ghAdapter       = &GhAdapter{}
	goAdapter       = &GoAdapter{}
	npmAdapter      = &NpmAdapter{}
	nvmAdapter      = &NVMAdapter{}
	openCodeAdapter = &OpenCodeAdapter{}
	pacmanAdapter   = &PacmanAdapter{}
	pnpmAdapter     = &PnpmAdapter{}
	scoopAdapter    = &ScoopAdapter{}
	uvAdapter       = &UvAdapter{}
	wingetAdapter   = &WingetAdapter{}

	allAdapters = []adapters.Adapter{
		aptAdapter, brewAdapter, pacmanAdapter, wingetAdapter, scoopAdapter,
		nvmAdapter, npmAdapter, pnpmAdapter, bunAdapter, uvAdapter,
		ghAdapter, dockerAdapter, goAdapter, openCodeAdapter,
	}

	linuxAdapters = []adapters.Adapter{
		aptAdapter, brewAdapter, pacmanAdapter, nvmAdapter, npmAdapter,
		pnpmAdapter, bunAdapter, uvAdapter, ghAdapter, dockerAdapter,
		goAdapter, openCodeAdapter,
	}
	macosAdapters = []adapters.Adapter{
		brewAdapter, nvmAdapter, npmAdapter, pnpmAdapter, bunAdapter,
		uvAdapter, ghAdapter, dockerAdapter, goAdapter, openCodeAdapter,
	}
	windowsAdapters = []adapters.Adapter{
		wingetAdapter, scoopAdapter, nvmAdapter, npmAdapter, pnpmAdapter,
		bunAdapter, uvAdapter, ghAdapter, dockerAdapter, goAdapter,
		openCodeAdapter,
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
		"winget", "scoop", "nvm", "npm", "pnpm",
		"bun", "uv", "gh", "docker", "go",
		"opencode",
	}
)

// AllAdapters returns every official adapter, regardless of platform.
// The caller is responsible for filtering by platform if needed.
func AllAdapters() []adapters.Adapter {
	return append([]adapters.Adapter(nil), allAdapters...)
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

// AdaptersForPlatform returns only the adapters relevant to the given OS.
// OS constants should come from the platform package (e.g., platform.OSLinux).
func AdaptersForPlatform(os string) []adapters.Adapter {
	switch os {
	case platform.OSLinux:
		return append([]adapters.Adapter(nil), linuxAdapters...)
	case platform.OSMacOS:
		return append([]adapters.Adapter(nil), macosAdapters...)
	case platform.OSWindows:
		return append([]adapters.Adapter(nil), windowsAdapters...)
	default:
		return nil
	}
}

// AdaptersForCurrentPlatform returns adapters for the detected runtime OS.
func AdaptersForCurrentPlatform() []adapters.Adapter {
	p, err := platform.Detect()
	if err != nil {
		return nil // unsupported platform
	}
	return AdaptersForPlatform(p.OS)
}

// AdapterByName returns the adapter with the given tool ID, or nil if not found.
func AdapterByName(id string) adapters.Adapter {
	switch id {
	case "apt":
		return aptAdapter
	case "brew":
		return brewAdapter
	case "bun":
		return bunAdapter
	case "docker":
		return dockerAdapter
	case "gh":
		return ghAdapter
	case "go":
		return goAdapter
	case "npm":
		return npmAdapter
	case "nvm":
		return nvmAdapter
	case "opencode":
		return openCodeAdapter
	case "pacman":
		return pacmanAdapter
	case "pnpm":
		return pnpmAdapter
	case "scoop":
		return scoopAdapter
	case "uv":
		return uvAdapter
	case "winget":
		return wingetAdapter
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

var (
	platformsLinux             = []string{platform.OSLinux}
	platformsLinuxMacOS        = []string{platform.OSLinux, platform.OSMacOS}
	platformsWindows           = []string{platform.OSWindows}
	platformsLinuxMacOSWindows = []string{platform.OSLinux, platform.OSMacOS, platform.OSWindows}
)

// PlatformsFor returns the slice of platform names supported by the official
// adapter with the given tool ID, or nil if the tool is unknown.
func PlatformsFor(id string) []string {
	switch id {
	case "apt", "pacman":
		return platformsLinux
	case "brew":
		return platformsLinuxMacOS
	case "winget", "scoop":
		return platformsWindows
	case "bun", "docker", "gh", "go", "npm", "nvm", "opencode", "pnpm", "uv":
		return platformsLinuxMacOSWindows
	default:
		return nil
	}
}
