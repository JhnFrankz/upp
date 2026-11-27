package engine

import (
	"context"

	"github.com/JhnFrankz/upp/internal/adapters"
)

// ownedCheckerAdapter delegates Check to a PackageChecker for an owned tool.
type ownedCheckerAdapter struct {
	adapters.Adapter
	checker adapters.PackageChecker
	pkg     string
}

func (o *ownedCheckerAdapter) Check(ctx context.Context) (adapters.UpdateInfo, error) {
	info, err := o.checker.CheckPackage(ctx, o.pkg)
	if err != nil {
		return adapters.UpdateInfo{}, err
	}
	if info != (adapters.UpdateInfo{}) {
		return info, nil
	}
	if vp, ok := o.Adapter.(interface {
		CurrentVersion(context.Context) (string, error)
	}); ok {
		cur, cerr := vp.CurrentVersion(ctx)
		if cerr != nil {
			return adapters.UpdateInfo{}, cerr
		}
		return adapters.UpdateInfo{
			CurrentVersion:  cur,
			LatestVersion:   cur,
			UpdateAvailable: false,
		}, nil
	}
	return o.Adapter.Check(ctx)
}

func (o *ownedCheckerAdapter) Unwrap() adapters.Adapter {
	return o.Adapter
}

// PrepareCheckAdapters wraps owned tools whose managers provide a PackageChecker.
func PrepareCheckAdapters(adapterList []adapters.Adapter, osName string, allAdapters ...[]adapters.Adapter) []adapters.Adapter {
	var list []adapters.Adapter
	if len(allAdapters) > 0 {
		list = allAdapters[0]
	}
	canonOS := normalizeOSName(osName)
	var idx adapterIndex
	if len(list) > 0 {
		idx = buildAdapterIndex(list)
	}

	result := make([]adapters.Adapter, len(adapterList))
	for i, a := range adapterList {
		if a == nil {
			result[i] = a
			continue
		}
		if _, ok := a.(*ownedCheckerAdapter); ok {
			result[i] = a
			continue
		}
		info := a.Info()
		if _, isCustom := a.(*adapters.CustomAdapter); !isCustom && len(info.Manager) == 0 {
			result[i] = a
			continue
		}
		var owner adapters.Adapter
		if list != nil {
			if idx != nil {
				owner = resolvingOwnerIndex(a, info, osName, canonOS, idx)
			}
		} else {
			owner = resolvingOwnerSlice(a, osName, nil)
		}
		if owner != nil && lookupOSMap(info.Manager, osName, canonOS) != "" {
			if checker, ok := owner.(adapters.PackageChecker); ok {
				if pkg := lookupOSMap(info.ManagerPackage, osName, canonOS); pkg != "" {
					result[i] = &ownedCheckerAdapter{
						Adapter: a,
						checker: checker,
						pkg:     pkg,
					}
					continue
				}
			}
		}
		result[i] = a
	}
	return result
}

// PrepareCheckAdapters wraps owned tools whose managers provide a PackageChecker,
// using the engine's configured OS and adapter list.
func (e *Engine) PrepareCheckAdapters(adapterList []adapters.Adapter) []adapters.Adapter {
	if all := e.allAdapters(); len(all) > 0 || e.adapters != nil {
		return PrepareCheckAdapters(adapterList, e.osName, all)
	}
	return PrepareCheckAdapters(adapterList, e.osName)
}
