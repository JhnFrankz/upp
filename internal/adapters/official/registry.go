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
		&WingetAdapter{},
		&ScoopAdapter{},
		&NVMAdapter{},
		&NpmAdapter{},
		&PnpmAdapter{},
		&BunAdapter{},
		&GhAdapter{},
		&DockerAdapter{},
		&GoAdapter{},
		&OpenCodeAdapter{},
	}
}

// AdaptersForPlatform returns only the adapters relevant to the given OS.
// OS constants should come from the platform package (e.g., platform.OSLinux).
func AdaptersForPlatform(os string) []adapters.Adapter {
	all := AllAdapters()
	var filtered []adapters.Adapter
	for _, a := range all {
		info := a.Info()
		for _, p := range info.Platforms {
			if p == os {
				filtered = append(filtered, a)
				break
			}
		}
	}
	return filtered
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
	for _, a := range AllAdapters() {
		if a.Name() == id {
			return a
		}
	}
	return nil
}
