package doctor

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"

	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/catalog"
	"github.com/JhnFrankz/upp/internal/config"
	"github.com/JhnFrankz/upp/internal/platform"
)

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
		toolPlatforms = catalog.PlatformsFor(name)
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
	return platform.ExtractCommandBinary(command, fallbackName)
}

func isShellFunctionTool(name string) bool {
	return name == "nvm"
}

func findAdapter(toolAdapters []adapters.Adapter, name string) adapters.Adapter {
	var firstMatch adapters.Adapter
	for _, a := range toolAdapters {
		if a.Name() == name {
			if a.Detect() {
				return a
			}
			if firstMatch == nil {
				firstMatch = a
			}
		}
	}
	return firstMatch
}
