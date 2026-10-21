package doctor

import (
	"context"
	"os"
	"os/exec"
	"sync"

	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/config"
	"github.com/JhnFrankz/upp/internal/lock"
	"github.com/JhnFrankz/upp/internal/platform"
)

// Severity indicates the outcome of a diagnostic check.
type Severity int

const (
	SeverityOK Severity = iota
	SeverityWarn
	SeverityError
)

func (s Severity) String() string {
	switch s {
	case SeverityOK:
		return "OK"
	case SeverityWarn:
		return "WARN"
	case SeverityError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// CheckResult represents the outcome of a single diagnostic check.
type CheckResult struct {
	Category string // e.g. "Configuration & Storage", "Process Lock", "Package Managers", "Tool Paths", "Network"
	Name     string
	Status   Severity
	Message  string
	Detail   string
	FixHint  string
}

// HasErrors returns true if any check result has SeverityError status.
func HasErrors(results []CheckResult) bool {
	for _, r := range results {
		if r.Status == SeverityError {
			return true
		}
	}
	return false
}

// HasWarnings returns true if any check result has SeverityWarn status.
func HasWarnings(results []CheckResult) bool {
	for _, r := range results {
		if r.Status == SeverityWarn {
			return true
		}
	}
	return false
}

// DoctorDeps defines dependencies and environment hooks for diagnostics.
type DoctorDeps struct {
	ConfigPath   func() (string, error)
	ConfigDir    func() (string, error)
	CacheDir     func() (string, error)
	LoadConfig   func() (*config.Config, error)
	LockPath     func() (string, error)
	ProcessAlive func(pid int) bool
	TryLock      func(f *os.File) error
	LookPath     func(name string) (string, error)
	FindAllPaths func(name string) []string
	Platform     platform.Platform
	// Adapters contains platform tool adapters to diagnose. Injected by the CLI
	// or caller to keep package doctor decoupled from concrete official adapters.
	Adapters []adapters.Adapter
	HTTPGet  func(ctx context.Context, url string) (int, error)
}

// DefaultDoctorDeps returns default production dependencies.
func DefaultDoctorDeps() DoctorDeps {
	p, _ := platform.Detect()
	return DoctorDeps{
		ConfigPath:   config.ConfigPath,
		ConfigDir:    config.ConfigDir,
		CacheDir:     config.CacheDir,
		LoadConfig:   config.Load,
		LockPath:     lock.DefaultLockPath,
		ProcessAlive: defaultProcessAlive,
		TryLock:      lock.TryLock,
		LookPath:     exec.LookPath,
		FindAllPaths: defaultFindAllPaths,
		Platform:     p,
		Adapters:     nil,
		HTTPGet:      defaultHTTPGet,
	}
}

// Diagnose executes all diagnostic checks and returns the results.
func Diagnose(ctx context.Context, deps DoctorDeps) []CheckResult {
	if deps.ConfigPath == nil {
		deps.ConfigPath = config.ConfigPath
	}
	if deps.ConfigDir == nil {
		deps.ConfigDir = config.ConfigDir
	}
	if deps.CacheDir == nil {
		deps.CacheDir = config.CacheDir
	}
	if deps.LoadConfig == nil {
		deps.LoadConfig = config.Load
	}
	if deps.LockPath == nil {
		deps.LockPath = lock.DefaultLockPath
	}
	if deps.ProcessAlive == nil {
		deps.ProcessAlive = defaultProcessAlive
	}
	if deps.TryLock == nil {
		deps.TryLock = lock.TryLock
	}
	if deps.FindAllPaths == nil && deps.LookPath == nil {
		deps.FindAllPaths = defaultFindAllPaths
	}
	if deps.LookPath == nil {
		deps.LookPath = exec.LookPath
	}
	if deps.HTTPGet == nil {
		deps.HTTPGet = defaultHTTPGet
	}
	if deps.Platform.OS == "" {
		deps.Platform, _ = platform.Detect()
	}

	if deps.LoadConfig != nil {
		origLoad := deps.LoadConfig
		var (
			cachedCfg *config.Config
			cachedErr error
			loadOnce  sync.Once
		)
		deps.LoadConfig = func() (*config.Config, error) {
			loadOnce.Do(func() {
				cachedCfg, cachedErr = origLoad()
			})
			return cachedCfg, cachedErr
		}
	}

	estimatedCap := 16
	if len(deps.Adapters) > 0 {
		estimatedCap += len(deps.Adapters)
	}
	results := make([]CheckResult, 0, estimatedCap)
	if ctx.Err() != nil {
		return results
	}
	results = checkStorageAndConfig(results, deps)
	if ctx.Err() != nil {
		return results
	}
	results = checkProcessLock(results, deps)
	if ctx.Err() != nil {
		return results
	}
	results = checkPackageManagers(results, deps)
	if ctx.Err() != nil {
		return results
	}
	results = checkToolPaths(results, deps)
	if ctx.Err() != nil {
		return results
	}
	results = checkNetwork(ctx, results, deps)

	return results
}
