package doctor

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

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
