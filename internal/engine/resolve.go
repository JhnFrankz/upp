package engine

import (
	"sort"
	"strings"

	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/adapters/official"
	"github.com/JhnFrankz/upp/internal/platform"
)

// Resolve discovers, instantiates, and filters active tool adapters according to
// the engine platform, configuration, and filter parameters.
func (e *Engine) Resolve(filter Filter) ([]adapters.Adapter, error) {
	var source []adapters.Adapter

	if e.adapters != nil {
		source = e.adapters
	} else {
		canonOS := e.osName
		if norm, err := platform.NormalizeOS(e.osName); err == nil {
			canonOS = norm
		}
		platformAdapters := official.AdaptersForPlatform(canonOS)
		customCount := 0
		if e.cfg != nil {
			customCount = len(e.cfg.Custom)
		}
		source = make([]adapters.Adapter, 0, len(platformAdapters)+customCount)
		for _, a := range platformAdapters {
			info := a.Info()
			if e.cfg != nil && e.cfg.Tools != nil {
				toolCfg, exists := e.cfg.Tools[info.ID]
				if !exists {
					toolCfg, exists = e.cfg.Tools[a.Name()]
				}
				if exists && !toolCfg.Enabled {
					continue
				}
			}
			source = append(source, a)
		}

		if e.cfg != nil && len(e.cfg.Custom) > 0 {
			customIDs := make([]string, 0, len(e.cfg.Custom))
			for id := range e.cfg.Custom {
				customIDs = append(customIDs, id)
			}
			sort.Strings(customIDs)

			for _, id := range customIDs {
				custom := e.cfg.Custom[id]
				if e.cfg.Tools != nil {
					if toolCfg, exists := e.cfg.Tools[id]; exists && !toolCfg.Enabled {
						continue
					}
				}

				var managerArgs []adapters.Adapter
				if custom.Manager != "" {
					if mgr := official.AdapterByName(custom.Manager); mgr != nil && mgr.Info().Kind == adapters.KindManager {
						managerArgs = append(managerArgs, mgr)
					}
				}

				a, err := adapters.NewCustomAdapterWithPackage(id, custom.Command, custom.CheckCmd, custom.Package, custom.Trusted, managerArgs...)
				if err != nil {
					continue
				}
				source = append(source, a)
			}
		}
	}

	if len(filter.Only) > 0 {
		return FilterAdapters(source, filter.Only), nil
	}

	if source == nil {
		return []adapters.Adapter{}, nil
	}
	return source, nil
}

// FilterAdapters filters a slice of adapters by tool ID or Name using the given filter list.
func FilterAdapters(adapterList []adapters.Adapter, only []string) []adapters.Adapter {
	if len(only) == 0 {
		return adapterList
	}

	if len(only) <= 3 {
		filtered := make([]adapters.Adapter, 0, len(only))
		for _, a := range adapterList {
			name := a.Name()
			id := a.Info().ID
			for _, want := range only {
				if strings.EqualFold(name, want) || strings.EqualFold(id, want) {
					filtered = append(filtered, a)
					break
				}
			}
		}
		return filtered
	}

	onlySet := make(map[string]struct{}, len(only))
	for _, name := range only {
		onlySet[strings.ToLower(strings.TrimSpace(name))] = struct{}{}
	}

	filtered := make([]adapters.Adapter, 0, len(only))
	for _, a := range adapterList {
		nameLower := strings.ToLower(a.Name())
		if _, ok := onlySet[nameLower]; ok {
			filtered = append(filtered, a)
			continue
		}
		// Only query Info() if name didn't match:
		info := a.Info()
		if info.ID != "" && info.ID != a.Name() {
			if _, ok := onlySet[strings.ToLower(info.ID)]; ok {
				filtered = append(filtered, a)
				continue
			}
		}
	}
	return filtered
}

// adapterByName finds an adapter by Name or ID in the provided adapter list.
func adapterByName(adapterList []adapters.Adapter, name string) adapters.Adapter {
	for _, a := range adapterList {
		if a.Name() == name || a.Info().ID == name {
			return a
		}
	}
	return nil
}

// resolvingOwnerSlice resolves the owning manager using a direct slice without variadic allocation.
func resolvingOwnerSlice(a adapters.Adapter, osName string, allAdapters []adapters.Adapter) adapters.Adapter {
	if a == nil {
		return nil
	}
	if custom, ok := a.(*adapters.CustomAdapter); ok {
		if m := custom.ManagerAdapter(); m != nil {
			return m
		}
	}
	canonOS := osName
	if norm, err := platform.NormalizeOS(osName); err == nil {
		canonOS = norm
	}

	if allAdapters != nil {
		if len(allAdapters) > 0 {
			info := a.Info()
			if info.Manager != nil {
				ownerName := info.Manager[osName]
				if ownerName == "" {
					ownerName = info.Manager[canonOS]
				}
				if ownerName != "" {
					if owner := adapterByName(allAdapters, ownerName); owner != nil {
						return owner
					}
				}
			}
		}
		return nil
	}

	if owner := official.ResolveOwner(a.Name(), canonOS); owner != nil {
		return owner
	}
	return official.ResolveOwner(a.Name(), osName)
}

// ResolvingOwner returns the manager adapter that owns the given adapter on the
// given OS, or nil when the adapter has no resolving owner (standalone).
func ResolvingOwner(a adapters.Adapter, osName string, allAdapters ...[]adapters.Adapter) adapters.Adapter {
	var list []adapters.Adapter
	if len(allAdapters) > 0 {
		list = allAdapters[0]
	}
	return resolvingOwnerSlice(a, osName, list)
}

// OwnedPackage returns the package name under the resolving manager for an
// owned tool on osName, or "" when none is declared.
func OwnedPackage(a adapters.Adapter, osName string) string {
	if a == nil {
		return ""
	}
	info := a.Info()
	if info.ManagerPackage == nil {
		return ""
	}
	if pkg, ok := info.ManagerPackage[osName]; ok && pkg != "" {
		return pkg
	}
	canonOS := osName
	if norm, err := platform.NormalizeOS(osName); err == nil {
		canonOS = norm
	}
	if pkg, ok := info.ManagerPackage[canonOS]; ok {
		return pkg
	}
	return ""
}
