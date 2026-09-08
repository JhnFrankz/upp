package engine

import (
	"sort"
	"strings"

	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/adapters/official"
	"github.com/JhnFrankz/upp/internal/platform"
)

// canonicalOS normalizes OS strings (e.g. "darwin" to platform.OSMacOS).
func canonicalOS(osName string) string {
	switch strings.ToLower(osName) {
	case "darwin", "macos":
		return platform.OSMacOS
	case "linux":
		return platform.OSLinux
	case "windows":
		return platform.OSWindows
	default:
		return osName
	}
}

// Resolve discovers, instantiates, and filters active tool adapters according to
// the engine platform, configuration, and filter parameters.
func (e *Engine) Resolve(filter Filter) ([]adapters.Adapter, error) {
	var source []adapters.Adapter

	if e.adapters != nil {
		source = e.adapters
	} else {
		canonOS := canonicalOS(e.osName)
		platformAdapters := official.AdaptersForPlatform(canonOS)
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

				a, err := adapters.NewCustomAdapter(id, custom.Command, custom.CheckCmd, custom.Trusted, managerArgs...)
				if err != nil {
					continue
				}
				source = append(source, a)
			}
		}
	}

	if len(filter.Only) > 0 {
		onlySet := make(map[string]struct{}, len(filter.Only))
		for _, name := range filter.Only {
			onlySet[strings.ToLower(strings.TrimSpace(name))] = struct{}{}
		}

		filtered := make([]adapters.Adapter, 0)
		for _, a := range source {
			nameLower := strings.ToLower(a.Name())
			idLower := strings.ToLower(a.Info().ID)
			if _, ok := onlySet[nameLower]; ok {
				filtered = append(filtered, a)
				continue
			}
			if _, ok := onlySet[idLower]; ok {
				filtered = append(filtered, a)
				continue
			}
		}
		return filtered, nil
	}

	if source == nil {
		return []adapters.Adapter{}, nil
	}
	return source, nil
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

// ResolvingOwner returns the manager adapter that owns the given adapter on the
// given OS, or nil when the adapter has no resolving owner (standalone).
func ResolvingOwner(a adapters.Adapter, osName string, allAdapters ...[]adapters.Adapter) adapters.Adapter {
	if a == nil {
		return nil
	}
	canonOS := canonicalOS(osName)

	if len(allAdapters) > 0 && allAdapters[0] != nil {
		info := a.Info()
		if info.Manager != nil {
			ownerName := info.Manager[osName]
			if ownerName == "" {
				ownerName = info.Manager[canonOS]
			}
			if ownerName != "" {
				if owner := adapterByName(allAdapters[0], ownerName); owner != nil {
					return owner
				}
			}
		}
	}

	if custom, ok := a.(*adapters.CustomAdapter); ok {
		if m := custom.ManagerAdapter(); m != nil {
			return m
		}
	}

	if owner := official.ResolveOwner(a.Name(), canonOS); owner != nil {
		return owner
	}
	return official.ResolveOwner(a.Name(), osName)
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
	canonOS := canonicalOS(osName)
	if pkg, ok := info.ManagerPackage[canonOS]; ok {
		return pkg
	}
	return ""
}

// UpdateCmdName returns the package-manager command token used to build the
// conventional risk command for a manager's owned-package command.
func UpdateCmdName(manager string) string {
	switch manager {
	case "apt":
		return "sudo apt install --only-upgrade"
	case "brew":
		return "brew upgrade"
	case "winget":
		return "winget upgrade"
	default:
		return manager + " upgrade"
	}
}
