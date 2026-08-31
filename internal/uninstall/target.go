package uninstall

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// TargetType identifies the category of an uninstallation target.
type TargetType string

const (
	TargetBinary TargetType = "binary"
	TargetBackup TargetType = "backup"
	TargetConfig TargetType = "config"
	TargetCache  TargetType = "cache"
)

// Target represents a specific path to be removed during uninstallation.
type Target struct {
	Type   TargetType
	Path   string
	Exists bool
}

// DiscoverTargets resolves and collects all uninstallation targets:
// 1. The canonical path of the running upp binary (after resolving symlinks).
// 2. Any historical backup binaries matching {binary}.backup.* in the binary directory.
// 3. The configuration directory (if non-empty).
// 4. The cache directory (if non-empty).
func DiscoverTargets(execPath, configDir, cacheDir string) ([]Target, error) {
	if execPath == "" {
		return nil, fmt.Errorf("uninstall: executable path is empty")
	}

	resolved, err := filepath.EvalSymlinks(execPath)
	if err != nil {
		return nil, fmt.Errorf("uninstall: cannot resolve binary path %s: %w", execPath, err)
	}

	var targets []Target

	// 1. Main binary target
	binStat, err := os.Stat(resolved)
	binExists := err == nil && !binStat.IsDir()
	targets = append(targets, Target{
		Type:   TargetBinary,
		Path:   resolved,
		Exists: binExists,
	})

	// 2. Historical backups in the binary directory
	binDir := filepath.Dir(resolved)
	binName := filepath.Base(resolved)
	backupPrefix := binName + ".backup."

	entries, err := os.ReadDir(binDir)
	if err == nil {
		for _, entry := range entries {
			name := entry.Name()
			if strings.HasPrefix(name, backupPrefix) {
				backupPath := filepath.Join(binDir, name)
				targets = append(targets, Target{
					Type:   TargetBackup,
					Path:   backupPath,
					Exists: true,
				})
			}
		}
	}

	// 3. Configuration directory
	if configDir != "" {
		stat, err := os.Stat(configDir)
		exists := err == nil && stat.IsDir()
		targets = append(targets, Target{
			Type:   TargetConfig,
			Path:   configDir,
			Exists: exists,
		})
	}

	// 4. Cache directory
	if cacheDir != "" && cacheDir != configDir {
		stat, err := os.Stat(cacheDir)
		exists := err == nil && stat.IsDir()
		targets = append(targets, Target{
			Type:   TargetCache,
			Path:   cacheDir,
			Exists: exists,
		})
	}

	return targets, nil
}
