package engine

import (
	"strings"

	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/platform"
)

// Resolve discovers, instantiates, and filters active tool adapters according to
// the engine platform, configuration, and filter parameters.
func (e *Engine) Resolve(filter Filter) ([]adapters.Adapter, error) {
	var source []adapters.Adapter

	if e.adapters != nil {
		source = e.adapters
	} else {
		var err error
		source, err = adapters.ResolveAdapters(e.cfg, e.osName)
		if err != nil {
			return nil, err
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

type onlyMatcher struct {
	enabled bool
	small   []string
	large   map[string]struct{}
}

func newOnlyMatcher(only []string) onlyMatcher {
	if len(only) == 0 {
		return onlyMatcher{}
	}
	if len(only) <= 3 {
		small := make([]string, 0, len(only))
		for _, name := range only {
			if trimmed := strings.TrimSpace(name); trimmed != "" {
				small = append(small, trimmed)
			}
		}
		return onlyMatcher{enabled: true, small: small}
	}
	large := make(map[string]struct{}, len(only))
	for _, name := range only {
		if trimmed := strings.TrimSpace(name); trimmed != "" {
			large[strings.ToLower(trimmed)] = struct{}{}
		}
	}
	return onlyMatcher{enabled: true, large: large}
}

func (m onlyMatcher) matches(name, id string) bool {
	if !m.enabled {
		return true
	}
	if m.large != nil {
		if _, ok := m.large[strings.ToLower(name)]; ok {
			return true
		}
		if id != "" && id != name {
			if _, ok := m.large[strings.ToLower(id)]; ok {
				return true
			}
		}
		return false
	}
	for _, want := range m.small {
		if strings.EqualFold(name, want) || strings.EqualFold(id, want) {
			return true
		}
	}
	return false
}

// FilterAdapters filters a slice of adapters by tool ID or Name using the given filter list.
func FilterAdapters(adapterList []adapters.Adapter, only []string) []adapters.Adapter {
	if len(only) == 0 {
		return adapterList
	}

	matcher := newOnlyMatcher(only)
	filtered := make([]adapters.Adapter, 0, len(only))
	for _, a := range adapterList {
		name := a.Name()
		if matcher.large != nil {
			if _, ok := matcher.large[strings.ToLower(name)]; ok {
				filtered = append(filtered, a)
				continue
			}
			// Only query Info() if name didn't match.
			info := a.Info()
			if info.ID != "" && info.ID != name {
				if _, ok := matcher.large[strings.ToLower(info.ID)]; ok {
					filtered = append(filtered, a)
				}
			}
			continue
		}
		if matcher.matches(name, a.Info().ID) {
			filtered = append(filtered, a)
		}
	}
	return filtered
}

func normalizeOSName(osName string) string {
	if norm, err := platform.NormalizeOS(osName); err == nil {
		return norm
	}
	return osName
}

func lookupOSMap(m map[string]string, osName, canonOS string) string {
	if m == nil {
		return ""
	}
	if v := m[osName]; v != "" {
		return v
	}
	if canonOS != "" && canonOS != osName {
		return m[canonOS]
	}
	return ""
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
			if allAdapters != nil {
				if owner := adapterByName(allAdapters, m.Info().ID); owner != nil {
					return owner
				}
				return adapterByName(allAdapters, m.Name())
			}
			return m
		}
	}
	canonOS := normalizeOSName(osName)

	if allAdapters != nil {
		if len(allAdapters) > 0 {
			info := a.Info()
			if ownerName := lookupOSMap(info.Manager, osName, canonOS); ownerName != "" {
				if owner := adapterByName(allAdapters, ownerName); owner != nil {
					return owner
				}
			}
		}
		return nil
	}

	if owner := adapters.ResolveOwner(a.Info().ID, canonOS); owner != nil {
		return owner
	}
	if owner := adapters.ResolveOwner(a.Name(), canonOS); owner != nil {
		return owner
	}
	if owner := adapters.ResolveOwner(a.Info().ID, osName); owner != nil {
		return owner
	}
	return adapters.ResolveOwner(a.Name(), osName)
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
	canonOS := normalizeOSName(osName)
	return lookupOSMap(a.Info().ManagerPackage, osName, canonOS)
}
