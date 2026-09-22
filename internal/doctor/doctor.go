package doctor

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
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
		LookPath:     exec.LookPath,
		FindAllPaths: defaultFindAllPaths,
		Platform:     p,
		Adapters:     official.AdaptersForCurrentPlatform(),
		HTTPGet:      defaultHTTPGet,
	}
}

func defaultFindAllPaths(name string) []string {
	pathEnv := os.Getenv("PATH")
	if pathEnv == "" {
		return nil
	}
	dirs := filepath.SplitList(pathEnv)
	var found []string
	seen := make(map[string]bool)

	exts := []string{""}
	if runtime.GOOS == "windows" {
		exts = []string{".exe", ".cmd", ".bat", ""}
	}

	for _, dir := range dirs {
		if dir == "" {
			continue
		}
		for _, ext := range exts {
			target := filepath.Join(dir, name+ext)
			fi, err := os.Stat(target)
			if err == nil && !fi.IsDir() {
				if runtime.GOOS != "windows" && (fi.Mode().Perm()&0o111) == 0 {
					continue
				}
				clean := filepath.Clean(target)
				if !seen[clean] {
					seen[clean] = true
					found = append(found, clean)
				}
				break
			}
		}
	}
	return found
}

func defaultHTTPGet(ctx context.Context, url string) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("User-Agent", "upp")
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
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
	if deps.LookPath == nil {
		deps.LookPath = exec.LookPath
	}
	if deps.FindAllPaths == nil {
		deps.FindAllPaths = defaultFindAllPaths
	}
	if deps.HTTPGet == nil {
		deps.HTTPGet = defaultHTTPGet
	}

	var results []CheckResult
	results = append(results, checkStorageAndConfig(deps)...)
	results = append(results, checkProcessLock(deps)...)
	results = append(results, checkPackageManagers(deps)...)
	results = append(results, checkToolPaths(deps)...)
	results = append(results, checkNetwork(ctx, deps)...)

	return results
}

func checkStorageAndConfig(deps DoctorDeps) []CheckResult {
	var results []CheckResult

	// 1. Config file
	cfgPath, err := deps.ConfigPath()
	if err != nil {
		results = append(results, CheckResult{
			Category: "Configuration & Storage",
			Name:     "Config File",
			Status:   SeverityError,
			Message:  fmt.Sprintf("Failed to resolve config path: %v", err),
			FixHint:  "Ensure environment variables (HOME, APPDATA, XDG_CONFIG_HOME) are set properly.",
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
		if mkErr := os.MkdirAll(cfgDir, 0o755); mkErr != nil {
			results = append(results, CheckResult{
				Category: "Configuration & Storage",
				Name:     "Config Directory",
				Status:   SeverityError,
				Message:  "Config directory cannot be created or is not writable",
				Detail:   mkErr.Error(),
				FixHint:  fmt.Sprintf("Ensure %s exists and has write permissions.", cfgDir),
			})
		} else {
			tmpFile, writeErr := os.CreateTemp(cfgDir, ".upp-doctor-test-*")
			if writeErr != nil {
				results = append(results, CheckResult{
					Category: "Configuration & Storage",
					Name:     "Config Directory",
					Status:   SeverityError,
					Message:  "Config directory is not writable",
					Detail:   writeErr.Error(),
					FixHint:  fmt.Sprintf("Ensure %s has write permissions.", cfgDir),
				})
			} else {
				tmpName := tmpFile.Name()
				_ = tmpFile.Close()
				_ = os.Remove(tmpName)
				results = append(results, CheckResult{
					Category: "Configuration & Storage",
					Name:     "Config Directory",
					Status:   SeverityOK,
					Message:  "Config directory is writable",
					Detail:   cfgDir,
				})
			}
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
		if mkErr := os.MkdirAll(cacheDir, 0o755); mkErr != nil {
			results = append(results, CheckResult{
				Category: "Configuration & Storage",
				Name:     "Cache Directory",
				Status:   SeverityWarn,
				Message:  "Cache directory cannot be created or is not writable",
				Detail:   mkErr.Error(),
				FixHint:  fmt.Sprintf("Ensure %s exists and has write permissions.", cacheDir),
			})
		} else {
			tmpFile, writeErr := os.CreateTemp(cacheDir, ".upp-doctor-test-*")
			if writeErr != nil {
				results = append(results, CheckResult{
					Category: "Configuration & Storage",
					Name:     "Cache Directory",
					Status:   SeverityWarn,
					Message:  "Cache directory is not writable",
					Detail:   writeErr.Error(),
					FixHint:  fmt.Sprintf("Ensure %s has write permissions.", cacheDir),
				})
			} else {
				tmpName := tmpFile.Name()
				_ = tmpFile.Close()
				_ = os.Remove(tmpName)
				results = append(results, CheckResult{
					Category: "Configuration & Storage",
					Name:     "Cache Directory",
					Status:   SeverityOK,
					Message:  "Cache directory is writable",
					Detail:   cacheDir,
				})
			}
		}
	}

	return results
}

func checkProcessLock(deps DoctorDeps) []CheckResult {
	lockPath, err := deps.LockPath()
	if err != nil {
		return []CheckResult{
			{
				Category: "Process Lock",
				Name:     "Process Lock",
				Status:   SeverityWarn,
				Message:  fmt.Sprintf("Failed to resolve lock path: %v", err),
			},
		}
	}

	data, err := os.ReadFile(lockPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []CheckResult{
				{
					Category: "Process Lock",
					Name:     "Process Lock",
					Status:   SeverityOK,
					Message:  "No active process lock",
					Detail:   lockPath,
				},
			}
		}
		return []CheckResult{
			{
				Category: "Process Lock",
				Name:     "Process Lock",
				Status:   SeverityWarn,
				Message:  fmt.Sprintf("Cannot read lock file: %v", err),
				Detail:   lockPath,
				FixHint:  fmt.Sprintf("Remove inaccessible lock file: rm %s", lockPath),
			},
		}
	}

	s := strings.TrimSpace(string(data))
	pid, perr := strconv.Atoi(s)
	if perr != nil || pid <= 0 {
		return []CheckResult{
			{
				Category: "Process Lock",
				Name:     "Process Lock",
				Status:   SeverityWarn,
				Message:  "Corrupted lock file found",
				Detail:   lockPath,
				FixHint:  fmt.Sprintf("Remove corrupted lock file: rm %s", lockPath),
			},
		}
	}

	if deps.ProcessAlive(pid) {
		return []CheckResult{
			{
				Category: "Process Lock",
				Name:     "Process Lock",
				Status:   SeverityWarn,
				Message:  fmt.Sprintf("Lock held by active process (PID: %d)", pid),
				Detail:   lockPath,
				FixHint:  fmt.Sprintf("Wait for running upp process (PID %d) to finish or terminate it if stuck.", pid),
			},
		}
	}

	return []CheckResult{
		{
			Category: "Process Lock",
			Name:     "Process Lock",
			Status:   SeverityWarn,
			Message:  fmt.Sprintf("Stale lock file detected (PID: %d is dead)", pid),
			Detail:   lockPath,
			FixHint:  fmt.Sprintf("Remove stale lock file: rm %s", lockPath),
		},
	}
}

func checkPackageManagers(deps DoctorDeps) []CheckResult {
	type pmCheck struct {
		name    string
		binName string
	}
	candidates := []pmCheck{
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

	var found []string
	for _, pm := range candidates {
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
		return []CheckResult{
			{
				Category: "Package Managers",
				Name:     "Package Managers",
				Status:   SeverityWarn,
				Message:  "No package manager found in PATH",
				FixHint:  "Install a supported package manager for your platform (e.g. brew, apt, pacman, winget, scoop).",
			},
		}
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
			return []CheckResult{
				{
					Category: "Package Managers",
					Name:     "Package Managers",
					Status:   SeverityWarn,
					Message:  fmt.Sprintf("Multiple package managers detected (%s) on Linux", strings.Join(found, ", ")),
					Detail:   "Both Homebrew (brew) and a system package manager are installed. Tools may be duplicated or managed inconsistently.",
					FixHint:  "Ensure tools are managed consistently to avoid duplicate installations or PATH conflicts.",
				},
			}
		}
	}

	return []CheckResult{
		{
			Category: "Package Managers",
			Name:     "Package Managers",
			Status:   SeverityOK,
			Message:  fmt.Sprintf("Detected package manager(s): %s", strings.Join(found, ", ")),
		},
	}
}

func checkToolPaths(deps DoctorDeps) []CheckResult {
	var results []CheckResult

	toolsToCheck := make(map[string]bool)
	configuredTools := make(map[string]bool)

	if deps.LoadConfig != nil {
		if cfg, err := deps.LoadConfig(); err == nil && cfg != nil {
			for name, tCfg := range cfg.Tools {
				if tCfg.Enabled {
					toolsToCheck[name] = true
					configuredTools[name] = true
				}
			}
			for name := range cfg.Custom {
				toolsToCheck[name] = true
				configuredTools[name] = true
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
		var paths []string
		if deps.FindAllPaths != nil {
			paths = deps.FindAllPaths(name)
		} else {
			p, err := deps.LookPath(name)
			if err == nil && p != "" {
				paths = []string{p}
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
					FixHint:  fmt.Sprintf("Ensure %s is installed and added to PATH.", name),
				})
			}
		}
	}

	return results
}

func checkNetwork(ctx context.Context, deps DoctorDeps) []CheckResult {
	targets := []struct {
		name string
		url  string
	}{
		{"api.github.com", "https://api.github.com"},
		{"go.dev", "https://go.dev"},
	}

	var results []CheckResult
	for _, t := range targets {
		status, err := deps.HTTPGet(ctx, t.url)
		if err != nil || status < 200 || status >= 400 {
			msg := fmt.Sprintf("Failed to reach %s", t.url)
			if err != nil {
				msg = fmt.Sprintf("Failed to reach %s: %v", t.url, err)
			} else {
				msg = fmt.Sprintf("Failed to reach %s (HTTP %d)", t.url, status)
			}
			results = append(results, CheckResult{
				Category: "Network",
				Name:     t.name,
				Status:   SeverityWarn,
				Message:  msg,
				Detail:   t.url,
				FixHint:  "Check your internet connection, DNS, or proxy settings.",
			})
		} else {
			results = append(results, CheckResult{
				Category: "Network",
				Name:     t.name,
				Status:   SeverityOK,
				Message:  fmt.Sprintf("Reachable (status %d)", status),
				Detail:   t.url,
			})
		}
	}
	return results
}
