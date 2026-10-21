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
