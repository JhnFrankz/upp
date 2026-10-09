package engine

import (
	"context"

	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/platform"
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
	return o.Adapter.Check(ctx)
}

// PrepareCheckAdapters wraps owned tools whose managers provide a PackageChecker.
func PrepareCheckAdapters(adapterList []adapters.Adapter, osName string, allAdapters ...[]adapters.Adapter) []adapters.Adapter {
	var list []adapters.Adapter
	if len(allAdapters) > 0 {
		list = allAdapters[0]
	}
	canonOS := osName
	if norm, err := platform.NormalizeOS(osName); err == nil {
		canonOS = norm
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
		owner := resolvingOwnerSlice(a, osName, list)
		if owner != nil && info.Manager != nil && (info.Manager[osName] != "" || info.Manager[canonOS] != "") {
			if checker, ok := owner.(adapters.PackageChecker); ok {
				pkg := OwnedPackage(a, osName)
				if pkg != "" {
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
	if e.adapters != nil {
		return PrepareCheckAdapters(adapterList, e.osName, e.adapters)
	}
	allAdapters, _ := e.Resolve(Filter{})
	if len(allAdapters) > 0 {
		return PrepareCheckAdapters(adapterList, e.osName, allAdapters)
	}
	return PrepareCheckAdapters(adapterList, e.osName)
}
