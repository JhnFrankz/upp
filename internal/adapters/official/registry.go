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

// AllAdapters returns every official adapter, regardless of platform.
// The caller is responsible for filtering by platform if needed.
func AllAdapters() []adapters.Adapter {
	return []adapters.Adapter{
		&AptAdapter{},
		&BrewAdapter{},
		&PacmanAdapter{},
		&WingetAdapter{},
		&ScoopAdapter{},
		&NVMAdapter{},
		&NpmAdapter{},
		&PnpmAdapter{},
		&BunAdapter{},
		&UvAdapter{},
		&GhAdapter{},
		&DockerAdapter{},
		&GoAdapter{},
		&OpenCodeAdapter{},
	}
}

// AdaptersForPlatform returns only the adapters relevant to the given OS.
// OS constants should come from the platform package (e.g., platform.OSLinux).
func AdaptersForPlatform(os string) []adapters.Adapter {
	switch os {
	case platform.OSLinux:
		return []adapters.Adapter{
			&AptAdapter{},
			&BrewAdapter{},
			&PacmanAdapter{},
			&NVMAdapter{},
			&NpmAdapter{},
			&PnpmAdapter{},
			&BunAdapter{},
			&UvAdapter{},
			&GhAdapter{},
			&DockerAdapter{},
			&GoAdapter{},
			&OpenCodeAdapter{},
		}
	case platform.OSMacOS:
		return []adapters.Adapter{
			&BrewAdapter{},
			&NVMAdapter{},
			&NpmAdapter{},
			&PnpmAdapter{},
			&BunAdapter{},
			&UvAdapter{},
			&GhAdapter{},
			&DockerAdapter{},
			&GoAdapter{},
			&OpenCodeAdapter{},
		}
	case platform.OSWindows:
		return []adapters.Adapter{
			&WingetAdapter{},
			&ScoopAdapter{},
			&NVMAdapter{},
			&NpmAdapter{},
			&PnpmAdapter{},
			&BunAdapter{},
			&UvAdapter{},
			&GhAdapter{},
			&DockerAdapter{},
			&GoAdapter{},
			&OpenCodeAdapter{},
		}
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
		return &AptAdapter{}
	case "brew":
		return &BrewAdapter{}
	case "bun":
		return &BunAdapter{}
	case "docker":
		return &DockerAdapter{}
	case "gh":
		return &GhAdapter{}
	case "go":
		return &GoAdapter{}
	case "npm":
		return &NpmAdapter{}
	case "nvm":
		return &NVMAdapter{}
	case "opencode":
		return &OpenCodeAdapter{}
	case "pacman":
		return &PacmanAdapter{}
	case "pnpm":
		return &PnpmAdapter{}
	case "scoop":
		return &ScoopAdapter{}
	case "uv":
		return &UvAdapter{}
	case "winget":
		return &WingetAdapter{}
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
