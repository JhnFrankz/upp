package doctor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/adapters/official"
	"github.com/JhnFrankz/upp/internal/config"
	"github.com/JhnFrankz/upp/internal/lock"
	"github.com/JhnFrankz/upp/internal/platform"
)

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
	Adapters     []adapters.Adapter
	HTTPGet      func(ctx context.Context, url string) (int, error)
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
		Adapters:     official.AdaptersForCurrentPlatform(),
		HTTPGet:      defaultHTTPGet,
	}
}

var (
	runtimeGOOSFn = func() string { return runtime.GOOS }

	pathEnvMu      sync.RWMutex
	cachedPathEnv  string
	cachedPathDirs []string

	unixExts    = []string{""}
	windowsExts = []string{".exe", ".cmd", ".bat", ""}
)

func getPathDirs() []string {
	pathEnv := os.Getenv("PATH")
	if pathEnv == "" {
		return nil
	}
	pathEnvMu.RLock()
	if pathEnv == cachedPathEnv && cachedPathDirs != nil {
		dirs := cachedPathDirs
		pathEnvMu.RUnlock()
		return dirs
	}
	pathEnvMu.RUnlock()

	pathEnvMu.Lock()
	defer pathEnvMu.Unlock()
	if pathEnv == cachedPathEnv && cachedPathDirs != nil {
		return cachedPathDirs
	}
	cachedPathEnv = pathEnv
	rawDirs := filepath.SplitList(pathEnv)
	cachedPathDirs = make([]string, 0, len(rawDirs))
	seenDirs := make(map[string]struct{}, len(rawDirs))
	for _, d := range rawDirs {
		cleaned := filepath.Clean(d)
		if _, seen := seenDirs[cleaned]; !seen {
			seenDirs[cleaned] = struct{}{}
			cachedPathDirs = append(cachedPathDirs, cleaned)
		}
	}
	return cachedPathDirs
}

func defaultFindAllPaths(name string) []string {
	dirs := getPathDirs()
	if len(dirs) == 0 {
		return nil
	}
	found := make([]string, 0, 2)
	var foundFi []os.FileInfo
	seen := make(map[string]bool)

	exts := unixExts
	if runtimeGOOSFn() == "windows" {
		exts = windowsExts
	}

	sep := string(filepath.Separator)
	for _, dir := range dirs {
		if dir == "" {
			continue
		}
		hasSlash := strings.HasSuffix(dir, sep)
		for _, ext := range exts {
			var target string
			if hasSlash {
				if ext == "" {
					target = dir + name
				} else {
					target = dir + name + ext
				}
			} else {
				if ext == "" {
					target = dir + sep + name
				} else {
					target = dir + sep + name + ext
				}
			}
			fi, err := os.Stat(target)
			if err == nil && !fi.IsDir() {
				if runtimeGOOSFn() != "windows" && (fi.Mode().Perm()&0o111) == 0 {
					continue
				}
				clean := target
				if seen[clean] {
					break
				}
				evalPath := clean
				if ep, err := filepath.EvalSymlinks(target); err == nil {
					evalPath = ep
				}
				if seen[evalPath] {
					break
				}

				isSame := false
				for _, existingFi := range foundFi {
					if os.SameFile(fi, existingFi) {
						isSame = true
						break
					}
				}
				if isSame {
					seen[clean] = true
					seen[evalPath] = true
					break
				}

				seen[clean] = true
				seen[evalPath] = true
				foundFi = append(foundFi, fi)
				found = append(found, clean)
				break
			}
		}
	}
	return found
}

// defaultDoctorHTTPClient is reused across HTTP checks to enable connection pooling and keep-alives.
var defaultDoctorHTTPClient = &http.Client{
	Timeout: 3 * time.Second,
}

func defaultHTTPGet(ctx context.Context, url string) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("User-Agent", "upp")
	resp, err := defaultDoctorHTTPClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer func() {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64*1024))
		_ = resp.Body.Close()
	}()
	return resp.StatusCode, nil
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
	if deps.Adapters == nil {
		deps.Adapters = official.AdaptersForCurrentPlatform()
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

func checkStorageAndConfig(results []CheckResult, deps DoctorDeps) []CheckResult {
	// 1. Config file
	cfgPath, err := deps.ConfigPath()
	if err != nil {
		results = append(results, CheckResult{
			Category: "Configuration & Storage",
			Name:     "Config File",
			Status:   SeverityError,
			Message:  fmt.Sprintf("Failed to resolve config path: %v", err),
			FixHint:  "Ensure environment variables (HOME or APPDATA) are set properly.",
		})
	} else {
		fi, statErr := os.Stat(cfgPath)
		if statErr != nil {
			if os.IsNotExist(statErr) {
				results = append(results, CheckResult{
					Category: "Configuration & Storage",
					Name:     "Config File",
					Status:   SeverityWarn,
					Message:  "Configuration file does not exist",
					Detail:   cfgPath,
					FixHint:  "Run 'upp init' to generate a default configuration file.",
				})
			} else {
				results = append(results, CheckResult{
					Category: "Configuration & Storage",
					Name:     "Config File",
					Status:   SeverityError,
					Message:  fmt.Sprintf("Cannot access config file: %v", statErr),
					Detail:   cfgPath,
					FixHint:  fmt.Sprintf("Ensure permissions allow reading %s", cfgPath),
				})
			}
		} else if fi.IsDir() {
			results = append(results, CheckResult{
				Category: "Configuration & Storage",
				Name:     "Config File",
				Status:   SeverityError,
				Message:  "Config path is a directory, expected a file",
				Detail:   cfgPath,
				FixHint:  "Remove directory or move config.toml inside.",
			})
		} else {
			_, loadErr := deps.LoadConfig()
			if loadErr != nil {
				results = append(results, CheckResult{
					Category: "Configuration & Storage",
					Name:     "Config File",
					Status:   SeverityError,
					Message:  "Configuration file is invalid",
					Detail:   loadErr.Error(),
					FixHint:  fmt.Sprintf("Fix TOML syntax or structural errors in %s", cfgPath),
				})
			} else {
				results = append(results, CheckResult{
					Category: "Configuration & Storage",
					Name:     "Config File",
					Status:   SeverityOK,
					Message:  "Configuration file is valid",
					Detail:   cfgPath,
				})
			}
		}
	}

	// 2. Config directory
	cfgDir, err := deps.ConfigDir()
	if err != nil {
		results = append(results, CheckResult{
			Category: "Configuration & Storage",
			Name:     "Config Directory",
			Status:   SeverityError,
			Message:  fmt.Sprintf("Failed to resolve config directory: %v", err),
		})
	} else {
		exists, writeErr := checkDirWritable(cfgDir)
		if writeErr != nil {
			msg := "Config directory is not writable"
			hint := fmt.Sprintf("Ensure %s has write permissions.", cfgDir)
			if !exists {
				msg = "Config directory cannot be created or is not writable"
				hint = fmt.Sprintf("Ensure %s exists and has write permissions.", cfgDir)
			}
			results = append(results, CheckResult{
				Category: "Configuration & Storage",
				Name:     "Config Directory",
				Status:   SeverityError,
				Message:  msg,
				Detail:   writeErr.Error(),
				FixHint:  hint,
			})
		} else {
			results = append(results, CheckResult{
				Category: "Configuration & Storage",
				Name:     "Config Directory",
				Status:   SeverityOK,
				Message:  "Config directory is writable",
				Detail:   cfgDir,
			})
		}
	}

	// 3. Cache directory
	cacheDir, err := deps.CacheDir()
	if err != nil {
		results = append(results, CheckResult{
			Category: "Configuration & Storage",
			Name:     "Cache Directory",
			Status:   SeverityWarn,
			Message:  fmt.Sprintf("Failed to resolve cache directory: %v", err),
		})
	} else {
		exists, writeErr := checkDirWritable(cacheDir)
		if writeErr != nil {
			msg := "Cache directory is not writable"
			hint := fmt.Sprintf("Ensure %s has write permissions.", cacheDir)
			if !exists {
				msg = "Cache directory cannot be created or is not writable"
				hint = fmt.Sprintf("Ensure %s exists and has write permissions.", cacheDir)
			}
			results = append(results, CheckResult{
				Category: "Configuration & Storage",
				Name:     "Cache Directory",
				Status:   SeverityWarn,
				Message:  msg,
				Detail:   writeErr.Error(),
				FixHint:  hint,
			})
		} else {
			results = append(results, CheckResult{
				Category: "Configuration & Storage",
				Name:     "Cache Directory",
				Status:   SeverityOK,
				Message:  "Cache directory is writable",
				Detail:   cacheDir,
			})
		}
	}

	return results
}

func checkDirWritable(targetDir string) (bool, error) {
	trimmed := strings.TrimSpace(targetDir)
	if trimmed == "" {
		return false, errors.New("empty directory path")
	}
	fi, statErr := os.Stat(trimmed)
	if statErr == nil {
		if !fi.IsDir() {
			return false, fmt.Errorf("%s is not a directory", trimmed)
		}
		return true, probeDirWrite(trimmed)
	}
	if !os.IsNotExist(statErr) {
		return false, statErr
	}

	cur := filepath.Clean(trimmed)
	if abs, absErr := filepath.Abs(cur); absErr == nil {
		cur = abs
	}
	for {
		parent := filepath.Dir(cur)
		if parent == cur {
			return false, statErr
		}
		cur = parent
		pfi, perr := os.Stat(cur)
		if perr == nil {
			if !pfi.IsDir() {
				return false, fmt.Errorf("%s is not a directory", cur)
			}
			return false, probeDirWrite(cur)
		}
		if !os.IsNotExist(perr) {
			return false, perr
		}
	}
}

func checkProcessLock(results []CheckResult, deps DoctorDeps) []CheckResult {
	lockPath, err := deps.LockPath()
	if err != nil {
		return append(results, CheckResult{
			Category: "Process Lock",
			Name:     "Process Lock",
			Status:   SeverityWarn,
			Message:  fmt.Sprintf("Failed to resolve lock path: %v", err),
		})
	}

	data, err := os.ReadFile(lockPath)
	if err != nil {
		if os.IsNotExist(err) {
			return append(results, CheckResult{
				Category: "Process Lock",
				Name:     "Process Lock",
				Status:   SeverityOK,
				Message:  "No active process lock",
				Detail:   lockPath,
			})
		}
		return append(results, CheckResult{
			Category: "Process Lock",
			Name:     "Process Lock",
			Status:   SeverityWarn,
			Message:  fmt.Sprintf("Cannot read lock file: %v", err),
			Detail:   lockPath,
			FixHint:  fmt.Sprintf("Remove inaccessible lock file: rm %s", lockPath),
		})
	}

	s := strings.TrimSpace(string(data))
	pid, perr := strconv.Atoi(s)
	if perr != nil || pid <= 0 {
		return append(results, CheckResult{
			Category: "Process Lock",
			Name:     "Process Lock",
			Status:   SeverityWarn,
			Message:  "Corrupted lock file found",
			Detail:   lockPath,
			FixHint:  fmt.Sprintf("Remove corrupted lock file: rm %s", lockPath),
		})
	}

	f, oerr := os.OpenFile(lockPath, os.O_RDWR, 0o600)
	if oerr != nil {
		return append(results, CheckResult{
			Category: "Process Lock",
			Name:     "Process Lock",
			Status:   SeverityWarn,
			Message:  fmt.Sprintf("Cannot open lock file: %v", oerr),
			Detail:   lockPath,
			FixHint:  fmt.Sprintf("Remove inaccessible lock file: rm %s", lockPath),
		})
	}

	tryLock := deps.TryLock
	if tryLock == nil {
		tryLock = lock.TryLock
	}

	lockErr := tryLock(f)
	if errors.Is(lockErr, lock.ErrLocked) {
		_ = f.Close()
		return append(results, CheckResult{
			Category: "Process Lock",
			Name:     "Process Lock",
			Status:   SeverityWarn,
			Message:  fmt.Sprintf("Lock held by active process (PID: %d)", pid),
			Detail:   lockPath,
			FixHint:  fmt.Sprintf("Wait for running upp process (PID %d) to finish or terminate it if stuck.", pid),
		})
	}
	if lockErr != nil {
		_ = f.Close()
		return append(results, CheckResult{
			Category: "Process Lock",
			Name:     "Process Lock",
			Status:   SeverityWarn,
			Message:  fmt.Sprintf("Cannot test lock on file: %v", lockErr),
			Detail:   lockPath,
			FixHint:  fmt.Sprintf("Check permissions or remove lock file: rm %s", lockPath),
		})
	}

	_ = lock.Unlock(f)
	_ = f.Close()

	if !deps.ProcessAlive(pid) {
		return append(results, CheckResult{
			Category: "Process Lock",
			Name:     "Process Lock",
			Status:   SeverityWarn,
			Message:  fmt.Sprintf("Stale lock file detected (PID: %d is dead)", pid),
			Detail:   lockPath,
			FixHint:  fmt.Sprintf("Remove stale lock file: rm %s", lockPath),
		})
	}

	return append(results, CheckResult{
		Category: "Process Lock",
		Name:     "Process Lock",
		Status:   SeverityWarn,
		Message:  fmt.Sprintf("Stale lock file detected (PID: %d does not hold lock)", pid),
		Detail:   lockPath,
		FixHint:  fmt.Sprintf("Remove stale lock file: rm %s", lockPath),
	})
}

type pmCheck struct {
	name    string
	binName string
}

var pmCandidates = []pmCheck{
	{"brew", "brew"},
	{"apt", "apt"},
	{"pacman", "pacman"},
	{"dnf", "dnf"},
	{"zypper", "zypper"},
	{"winget", "winget"},
	{"scoop", "scoop"},
	{"choco", "choco"},
	{"nix", "nix"},
}

func checkPackageManagers(results []CheckResult, deps DoctorDeps) []CheckResult {
	found := make([]string, 0, 4)
	for _, pm := range pmCandidates {
		_, err := deps.LookPath(pm.binName)
		if err == nil {
			found = append(found, pm.name)
		} else if pm.name == "apt" {
			if _, gerr := deps.LookPath("apt-get"); gerr == nil {
				found = append(found, pm.name)
			}
		}
	}

	if len(found) == 0 {
		return append(results, CheckResult{
			Category: "Package Managers",
			Name:     "Package Managers",
			Status:   SeverityWarn,
			Message:  "No package manager found in PATH",
			FixHint:  "Install a supported package manager for your platform (e.g. brew, apt, pacman, winget, scoop).",
		})
	}

	if deps.Platform.OS == platform.OSLinux {
		hasBrew := false
		hasSystem := false
		for _, m := range found {
			if m == "brew" {
				hasBrew = true
			}
			if m == "apt" || m == "pacman" || m == "dnf" || m == "zypper" {
				hasSystem = true
			}
		}
		if hasBrew && hasSystem {
			return append(results, CheckResult{
				Category: "Package Managers",
				Name:     "Package Managers",
				Status:   SeverityWarn,
				Message:  fmt.Sprintf("Multiple package managers detected (%s) on Linux", strings.Join(found, ", ")),
				Detail:   "Both Homebrew (brew) and a system package manager are installed. Tools may be duplicated or managed inconsistently.",
				FixHint:  "Ensure tools are managed consistently to avoid duplicate installations or PATH conflicts.",
			})
		}
	}

	return append(results, CheckResult{
		Category: "Package Managers",
		Name:     "Package Managers",
		Status:   SeverityOK,
		Message:  fmt.Sprintf("Detected package manager(s): %s", strings.Join(found, ", ")),
	})
}

func checkToolPaths(results []CheckResult, deps DoctorDeps) []CheckResult {
	initCap := len(deps.Adapters)
	var (
		cfg    *config.Config
		cfgErr error
	)
	if deps.LoadConfig != nil {
		cfg, cfgErr = deps.LoadConfig()
		if cfgErr == nil && cfg != nil {
			initCap += len(cfg.Tools) + len(cfg.Custom)
		}
	}

	toolsToCheck := make(map[string]bool, initCap)
	configuredTools := make(map[string]bool, initCap)
	var customBinaries map[string]string

	if cfgErr == nil && cfg != nil {
		for name, tCfg := range cfg.Tools {
			if !tCfg.Enabled || tCfg.Defaulted {
				continue
			}
			_, isCustom := cfg.Custom[name]
			if !isToolSupportedOnPlatform(name, tCfg, isCustom, deps.Platform.OS) {
				continue
			}
			toolsToCheck[name] = true
			configuredTools[name] = true
		}
		if len(cfg.Custom) > 0 {
			customBinaries = make(map[string]string, len(cfg.Custom))
			for name, custom := range cfg.Custom {
				var tCfg config.ToolConfig
				if cfg.Tools != nil {
					var exists bool
					tCfg, exists = cfg.Tools[name]
					if exists && !tCfg.Enabled {
						continue
					}
				}
				if !isToolSupportedOnPlatform(name, tCfg, true, deps.Platform.OS) {
					continue
				}
				toolsToCheck[name] = true
				configuredTools[name] = true
				customBinaries[name] = extractCustomBinary(custom.Command, name)
			}
		}
	}

	for _, a := range deps.Adapters {
		if a.Detect() {
			toolsToCheck[a.Name()] = true
		}
	}

	names := make([]string, 0, len(toolsToCheck))
	for name := range toolsToCheck {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		if isShellFunctionTool(name) {
			if a := findAdapter(deps.Adapters, name); a != nil && a.Detect() {
				results = append(results, CheckResult{
					Category: "Tool Paths",
					Name:     name,
					Status:   SeverityOK,
					Message:  fmt.Sprintf("%s detected via environment", name),
				})
				continue
			}
		}

		probeName := name
		if bin, ok := customBinaries[name]; ok && bin != "" {
			probeName = bin
		}

		paths := lookupToolPaths(deps, probeName)
		if len(paths) == 0 && strings.ContainsAny(probeName, `/\`) {
			if fi, err := os.Stat(probeName); err == nil && !fi.IsDir() {
				if runtimeGOOSFn() == "windows" || (fi.Mode().Perm()&0o111) != 0 {
					paths = []string{probeName}
				}
			}
		}
		if len(paths) == 0 && name == "apt" {
			paths = lookupToolPaths(deps, "apt-get")
			if len(paths) == 0 {
				if a := findAdapter(deps.Adapters, name); a != nil && a.Detect() {
					results = append(results, CheckResult{
						Category: "Tool Paths",
						Name:     name,
						Status:   SeverityOK,
						Message:  fmt.Sprintf("%s detected via environment", name),
					})
					continue
				}
			}
		}

		if len(paths) > 1 {
			results = append(results, CheckResult{
				Category: "Tool Paths",
				Name:     name,
				Status:   SeverityWarn,
				Message:  fmt.Sprintf("Tool %q is shadowed by multiple installations", name),
				Detail:   fmt.Sprintf("Active: %s\nShadowed: %s", paths[0], strings.Join(paths[1:], ", ")),
				FixHint:  fmt.Sprintf("Check your PATH order or remove duplicate %s installations.", name),
			})
		} else if len(paths) == 1 {
			results = append(results, CheckResult{
				Category: "Tool Paths",
				Name:     name,
				Status:   SeverityOK,
				Message:  fmt.Sprintf("%s found at %s", name, paths[0]),
				Detail:   paths[0],
			})
		} else {
			if configuredTools[name] {
				results = append(results, CheckResult{
					Category: "Tool Paths",
					Name:     name,
					Status:   SeverityWarn,
					Message:  fmt.Sprintf("Tool %q is configured but not found in PATH", name),
					FixHint:  fmt.Sprintf("Ensure %s is installed and added to PATH.", probeName),
				})
			}
		}
	}

	return results
}

func isToolSupportedOnPlatform(name string, tCfg config.ToolConfig, isCustom bool, currentOS string) bool {
	if currentOS == "" {
		return true
	}
	canonOS := currentOS
	if norm, err := platform.NormalizeOS(currentOS); err == nil {
		canonOS = norm
	}

	toolPlatforms := tCfg.Platforms
	if len(toolPlatforms) == 0 && !isCustom {
		toolPlatforms = official.PlatformsFor(name)
	}
	if len(toolPlatforms) == 0 {
		return true
	}
	for _, p := range toolPlatforms {
		if p == canonOS || p == currentOS {
			return true
		}
		if norm, err := platform.NormalizeOS(p); err == nil && norm == canonOS {
			return true
		}
	}
	return false
}

func lookupToolPaths(deps DoctorDeps, name string) []string {
	if deps.FindAllPaths != nil {
		return deps.FindAllPaths(name)
	}
	if deps.LookPath != nil {
		if p, err := deps.LookPath(name); err == nil && p != "" {
			return []string{p}
		}
	}
	return nil
}

func extractCustomBinary(command, fallbackName string) string {
	tokens := tokenizeShellCommand(command)
	if len(tokens) == 0 {
		return fallbackName
	}

	for i := 0; i < len(tokens); i++ {
		tok := tokens[i]
		if tok == "" || isEnvVarAssignment(tok) {
			continue
		}

		baseLower := normalizedWrapperName(tok)
		if baseLower == "env" {
			for i+1 < len(tokens) {
				next := tokens[i+1]
				if next == "--" {
					i++
					break
				}
				if next == "-u" || next == "--unset" || next == "-C" || next == "--chdir" {
					i += 2
					continue
				}
				if next == "-S" || next == "--split-string" {
					if i+2 < len(tokens) && i+3 >= len(tokens) {
						if inner := extractCustomBinary(tokens[i+2], ""); inner != "" {
							return inner
						}
					}
					i += 2
					continue
				}
				if strings.HasPrefix(next, "-") {
					i++
					continue
				}
				break
			}
			continue
		}

		if baseLower == "su" {
			for i+1 < len(tokens) {
				next := tokens[i+1]
				if next == "--" {
					i++
					break
				}
				if next == "-c" || next == "--command" || (strings.HasPrefix(next, "-") && !strings.HasPrefix(next, "--") && strings.HasSuffix(next, "c")) {
					if i+2 < len(tokens) {
						if inner := extractCustomBinary(tokens[i+2], ""); inner != "" {
							return inner
						}
					}
					i += 2
					continue
				}
				if strings.HasPrefix(next, "--command=") {
					if inner := extractCustomBinary(strings.TrimPrefix(next, "--command="), ""); inner != "" {
						return inner
					}
					i++
					continue
				}
				if isPrivilegeFlagWithArg(baseLower, next) {
					i += 2
					continue
				}
				if strings.HasPrefix(next, "-") || hasSuCommandFlagAhead(tokens[i+2:]) {
					i++
					continue
				}
				break
			}
			continue
		}

		if isPrivilegeWrapper(baseLower) {
			for i+1 < len(tokens) {
				next := tokens[i+1]
				if next == "--" {
					i++
					break
				}
				if isPrivilegeFlagWithArg(baseLower, next) {
					i += 2
					continue
				}
				if strings.HasPrefix(next, "-") || (baseLower == "runas" && strings.HasPrefix(next, "/")) {
					i++
					continue
				}
				break
			}
			continue
		}

		if isShellWrapper(baseLower) {
			if inner := extractShellCommandTarget(baseLower, tokens[i+1:]); inner != "" {
				return inner
			}
		}

		return tok
	}

	return fallbackName
}

func tokenizeShellCommand(command string) []string {
	var tokens []string
	var cur strings.Builder
	inToken := false
	var quote byte

	for i := 0; i < len(command); i++ {
		ch := command[i]
		if quote != 0 {
			if ch == quote {
				quote = 0
				continue
			}
			if quote == '"' && ch == '\\' && i+1 < len(command) {
				next := command[i+1]
				if next == '"' || next == '\\' || next == '$' || next == '`' {
					cur.WriteByte(next)
					i++
					continue
				}
			}
			cur.WriteByte(ch)
			continue
		}

		if ch == '\'' || ch == '"' {
			quote = ch
			inToken = true
			continue
		}

		if ch == '\\' && i+1 < len(command) {
			next := command[i+1]
			if next == ' ' || next == '\t' || next == '"' || next == '\'' || next == ';' || next == '|' || next == '&' {
				cur.WriteByte(next)
				inToken = true
				i++
				continue
			}
		}

		if ch == ' ' || ch == '\t' || ch == '\r' {
			if inToken {
				tokens = append(tokens, cur.String())
				cur.Reset()
				inToken = false
			}
			continue
		}

		if ch == ';' || ch == '|' || ch == '&' || ch == '\n' {
			if inToken {
				tokens = append(tokens, cur.String())
				cur.Reset()
				inToken = false
			}
			if len(tokens) > 0 {
				break
			}
			continue
		}

		cur.WriteByte(ch)
		inToken = true
	}

	if inToken {
		tokens = append(tokens, cur.String())
	}
	return tokens
}

func normalizedWrapperName(tok string) string {
	base := tok
	if idx := strings.LastIndexAny(base, `/\`); idx >= 0 {
		base = base[idx+1:]
	}
	baseLower := strings.ToLower(base)
	return strings.TrimSuffix(baseLower, ".exe")
}

func hasSuCommandFlagAhead(rest []string) bool {
	for _, t := range rest {
		if t == "-c" || t == "--command" || strings.HasPrefix(t, "--command=") ||
			(strings.HasPrefix(t, "-") && !strings.HasPrefix(t, "--") && strings.HasSuffix(t, "c")) {
			return true
		}
	}
	return false
}

func isShellWrapper(baseLower string) bool {
	switch baseLower {
	case "sh", "bash", "zsh", "dash", "ksh", "cmd", "powershell", "pwsh":
		return true
	default:
		return false
	}
}

func extractShellCommandTarget(shell string, args []string) string {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch shell {
		case "cmd":
			lower := strings.ToLower(arg)
			if (lower == "/c" || lower == "/k") && i+1 < len(args) {
				inner := extractCustomBinary(args[i+1], "")
				if inner != "" && !isShellBuiltin(normalizedWrapperName(inner)) {
					return inner
				}
				return ""
			}
		case "powershell", "pwsh":
			lower := strings.ToLower(arg)
			if (lower == "-c" || lower == "-command") && i+1 < len(args) {
				inner := extractCustomBinary(args[i+1], "")
				if inner != "" && !isShellBuiltin(normalizedWrapperName(inner)) {
					return inner
				}
				return ""
			}
		default:
			if arg == "-c" || (strings.HasPrefix(arg, "-") && !strings.HasPrefix(arg, "--") && strings.ContainsRune(arg[1:], 'c')) {
				if i+1 < len(args) {
					inner := extractCustomBinary(args[i+1], "")
					if inner != "" && !isShellBuiltin(normalizedWrapperName(inner)) {
						return inner
					}
				}
				return ""
			}
			if !strings.HasPrefix(arg, "-") {
				return ""
			}
		}
	}
	return ""
}

func isShellBuiltin(name string) bool {
	switch name {
	case "echo", "printf", "true", "false", "cd", "pwd", "export", "unset", "set", "exit", "return", "test", "[", ":", "eval", "exec", "source", ".":
		return true
	default:
		return false
	}
}

func isEnvVarAssignment(tok string) bool {
	idx := strings.IndexByte(tok, '=')
	if idx <= 0 {
		return false
	}
	key := tok[:idx]
	for i := 0; i < len(key); i++ {
		if !isEnvKeyChar(key[i], i == 0) {
			return false
		}
	}
	return true
}

func isEnvKeyChar(c byte, first bool) bool {
	if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_' {
		return true
	}
	return !first && c >= '0' && c <= '9'
}

func isPrivilegeWrapper(baseLower string) bool {
	switch baseLower {
	case "sudo", "doas", "pkexec", "su", "runas":
		return true
	default:
		return false
	}
}

func isPrivilegeFlagWithArg(wrapper, flag string) bool {
	if strings.ContainsRune(flag, '=') {
		return false
	}
	switch wrapper {
	case "sudo":
		switch flag {
		case "-u", "--user", "-g", "--group", "-C", "--close-from", "-D", "--chdir", "-R", "--chroot", "-T", "--command-timeout", "-h", "--host", "-p", "--prompt", "-r", "--role", "-t", "--type", "-U", "--other-user":
			return true
		}
		if strings.HasPrefix(flag, "-") && !strings.HasPrefix(flag, "--") && len(flag) > 2 {
			switch flag[len(flag)-1] {
			case 'u', 'g', 'C', 'D', 'R', 'T', 'h', 'p', 'r', 't', 'U':
				return true
			}
		}
	case "doas":
		switch flag {
		case "-u", "-C", "-a":
			return true
		}
		if strings.HasPrefix(flag, "-") && !strings.HasPrefix(flag, "--") && len(flag) > 2 {
			switch flag[len(flag)-1] {
			case 'u', 'C', 'a':
				return true
			}
		}
	case "pkexec":
		if flag == "--user" {
			return true
		}
	case "su":
		switch flag {
		case "-s", "--shell", "-g", "--group", "-G", "--supp-group":
			return true
		}
	}
	return false
}

func isShellFunctionTool(name string) bool {
	return name == "nvm"
}

func findAdapter(adapters []adapters.Adapter, name string) adapters.Adapter {
	for _, a := range adapters {
		if a.Name() == name {
			return a
		}
	}
	return nil
}

func checkNetwork(ctx context.Context, results []CheckResult, deps DoctorDeps) []CheckResult {
	targets := []struct {
		name string
		url  string
	}{
		{"api.github.com", "https://api.github.com"},
		{"go.dev", "https://go.dev"},
	}

	netResults := make([]CheckResult, len(targets))
	var wg sync.WaitGroup

	for i, t := range targets {
		wg.Add(1)
		go func(idx int, target struct{ name, url string }) {
			defer wg.Done()

			if ctx.Err() != nil {
				netResults[idx] = CheckResult{
					Category: "Network",
					Name:     target.name,
					Status:   SeverityWarn,
					Message:  fmt.Sprintf("Check aborted: %v", ctx.Err()),
					Detail:   target.url,
				}
				return
			}

			status, err := deps.HTTPGet(ctx, target.url)
			if err != nil || status < 200 || status >= 400 {
				var msg string
				if err != nil {
					msg = fmt.Sprintf("Failed to reach %s: %v", target.url, err)
				} else {
					msg = fmt.Sprintf("Failed to reach %s (HTTP %d)", target.url, status)
				}
				netResults[idx] = CheckResult{
					Category: "Network",
					Name:     target.name,
					Status:   SeverityWarn,
					Message:  msg,
					Detail:   target.url,
					FixHint:  "Check your internet connection, DNS, or proxy settings.",
				}
			} else {
				netResults[idx] = CheckResult{
					Category: "Network",
					Name:     target.name,
					Status:   SeverityOK,
					Message:  fmt.Sprintf("Reachable (status %d)", status),
					Detail:   target.url,
				}
			}
		}(i, t)
	}

	wg.Wait()
	return append(results, netResults...)
}
