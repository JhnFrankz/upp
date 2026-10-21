package adapters

import (
	"sort"
	"sync"

	"github.com/JhnFrankz/upp/internal/config"
	"github.com/JhnFrankz/upp/internal/platform"
)

// Provider defines the interface for querying platform-specific official or built-in adapters.
type Provider interface {
	AdaptersForPlatform(os string) []Adapter
	AdapterByName(name string) Adapter
	ResolveOwner(tool, os string) Adapter
}

var (
	providerMu     sync.RWMutex
	globalProvider Provider
)

// RegisterProvider registers the official/global adapter provider.
func RegisterProvider(p Provider) {
	providerMu.Lock()
	defer providerMu.Unlock()
	globalProvider = p
}

// GlobalProvider returns the currently registered adapter provider, or nil if none is registered.
func GlobalProvider() Provider {
	providerMu.RLock()
	defer providerMu.RUnlock()
	return globalProvider
}

// AdaptersForPlatform returns platform adapters using the registered provider, or nil.
func AdaptersForPlatform(os string) []Adapter {
	providerMu.RLock()
	p := globalProvider
	providerMu.RUnlock()
	if p != nil {
		return p.AdaptersForPlatform(os)
	}
	return nil
}

// AdapterByName returns an adapter by tool ID using the registered provider, or nil.
func AdapterByName(name string) Adapter {
	providerMu.RLock()
	p := globalProvider
	providerMu.RUnlock()
	if p != nil {
		return p.AdapterByName(name)
	}
	return nil
}

// ResolveOwner resolves the owning manager using the registered provider, or nil.
func ResolveOwner(tool, os string) Adapter {
	providerMu.RLock()
	p := globalProvider
	providerMu.RUnlock()
	if p != nil {
		return p.ResolveOwner(tool, os)
	}
	return nil
}

// ResolveOption configures adapter resolution.
type ResolveOption func(*resolveOptions)

type resolveOptions struct {
	provider Provider
}

// WithProvider overrides the global provider for a specific ResolveAdapters call.
func WithProvider(p Provider) ResolveOption {
	return func(ro *resolveOptions) {
		ro.provider = p
	}
}

// ResolveAdapters discovers, instantiates, and filters active tool adapters according to
// the configuration and platform.
func ResolveAdapters(cfg *config.Config, osName string, opts ...ResolveOption) ([]Adapter, error) {
	ro := &resolveOptions{}
	for _, opt := range opts {
		opt(ro)
	}
	provider := ro.provider
	if provider == nil {
		providerMu.RLock()
		provider = globalProvider
		providerMu.RUnlock()
	}

	canonOS := osName
	if norm, err := platform.NormalizeOS(osName); err == nil {
		canonOS = norm
	}

	var platformAdapters []Adapter
	if provider != nil {
		platformAdapters = provider.AdaptersForPlatform(canonOS)
	}

	customCount := 0
	if cfg != nil {
		customCount = len(cfg.Custom)
	}

	source := make([]Adapter, 0, len(platformAdapters)+customCount)
	for _, a := range platformAdapters {
		info := a.Info()
		if cfg != nil && cfg.Tools != nil {
			toolCfg, exists := cfg.Tools[info.ID]
			if !exists {
				toolCfg, exists = cfg.Tools[a.Name()]
			}
			if exists && !toolCfg.Enabled {
				continue
			}
		}
		source = append(source, a)
	}

	if cfg != nil && len(cfg.Custom) > 0 {
		customIDs := make([]string, 0, len(cfg.Custom))
		for id := range cfg.Custom {
			customIDs = append(customIDs, id)
		}
		sort.Strings(customIDs)

		for _, id := range customIDs {
			custom := cfg.Custom[id]
			if cfg.Tools != nil {
				if toolCfg, exists := cfg.Tools[id]; exists && !toolCfg.Enabled {
					continue
				}
			}

			var managerArgs []Adapter
			if custom.Manager != "" {
				managerDisabled := false
				if cfg.Tools != nil {
					if toolCfg, exists := cfg.Tools[custom.Manager]; exists && !toolCfg.Enabled {
						managerDisabled = true
					}
				}
				if !managerDisabled && provider != nil {
					if mgr := provider.AdapterByName(custom.Manager); mgr != nil && mgr.Info().Kind == KindManager && mgr.Info().SupportsPlatform(canonOS) {
						managerArgs = append(managerArgs, mgr)
					}
				}
			}

			a, err := NewCustomAdapterWithPackage(id, custom.Command, custom.CheckCmd, custom.Package, custom.Trusted, managerArgs...)
			if err != nil {
				continue
			}
			source = append(source, a)
		}
	}

	if source == nil {
		return []Adapter{}, nil
	}
	return source, nil
}
