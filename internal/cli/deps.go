package cli

import (
	"fmt"
	"io"

	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/config"
	"github.com/JhnFrankz/upp/internal/engine"
	"github.com/JhnFrankz/upp/internal/lock"
	"github.com/JhnFrankz/upp/internal/platform"
)

// cliDeps is the package-level injection point for the cobra entry points.
// Each New*Command RunE body passes the matching field down to its run*
// function, whose deps struct nil-defaults every nil seam to the production
// implementation — the zero value IS production behavior (design D2, mirrors
// setExecFakes in official/helper.go).
//
// Sequential-only: this package has no t.Parallel tests, so mutation from
// tests (setCLIDeps) needs no synchronization. Adding t.Parallel tests
// requires a mutex or per-command dep construction.
var cliDeps = struct {
	dashboard  dashboardDeps
	update     updateDeps
	list       listDeps
	selfUpdate selfUpdateDeps
	uninstall  uninstallDeps
	init       initDeps
	doctor     doctorDeps
}{
	init: initDeps{
		acquireLock: defaultAcquireLock,
	},
}

func defaultAcquireLock() (*lock.Lock, error) {
	path, err := lock.DefaultLockPath()
	if err != nil {
		return nil, err
	}
	return lock.Acquire(path)
}

// acquireOptionalLock acquires the process lock via fn (defaulting to
// defaultAcquireLock when fn is nil) and returns a release function safe to
// defer unconditionally.
func acquireOptionalLock(fn func() (*lock.Lock, error)) (func(), error) {
	if fn == nil {
		fn = defaultAcquireLock
	}
	l, err := fn()
	if err != nil {
		return nil, err
	}
	if l == nil {
		return func() {}, nil
	}
	return func() { _ = l.Release() }, nil
}

type resolvedSession struct {
	osName           string
	eng              *engine.Engine
	allAdapters      []adapters.Adapter
	filteredAdapters []adapters.Adapter
	onlyList         []string
}

func resolveSessionAdapters(
	gf *GlobalFlags,
	buildAdapterList func(*config.Config, string) []adapters.Adapter,
	errOut io.Writer,
) (resolvedSession, error) {
	cfg, err := config.Load()
	if err != nil {
		return resolvedSession{}, fmt.Errorf("cannot load config: %w", err)
	}

	p, err := platform.Detect()
	if err != nil {
		return resolvedSession{}, fmt.Errorf("cannot detect platform: %w", err)
	}

	var allAdapters []adapters.Adapter
	if buildAdapterList != nil {
		allAdapters = buildAdapterList(cfg, p.OS)
	} else {
		allAdapters, err = adapters.ResolveAdapters(cfg, p.OS)
		if err != nil {
			return resolvedSession{}, fmt.Errorf("cannot resolve tools: %w", err)
		}
	}
	eng := engine.New(cfg, p.OS, engine.WithAdapters(allAdapters))

	onlyList := ParseFilter(gf.Only)
	if len(onlyList) > 0 {
		FilterTools(adapterIDs(allAdapters), onlyList, errOut)
	}

	filteredAdapters := allAdapters
	if len(onlyList) > 0 {
		filteredAdapters = engine.FilterAdapters(allAdapters, onlyList)
	}

	return resolvedSession{
		osName:           p.OS,
		eng:              eng,
		allAdapters:      allAdapters,
		filteredAdapters: filteredAdapters,
		onlyList:         onlyList,
	}, nil
}
