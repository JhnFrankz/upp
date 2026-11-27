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
	results = checkWritableDir(results, "Config Directory", "Config directory", deps.ConfigDir, SeverityError)

	// 3. Cache directory
	results = checkWritableDir(results, "Cache Directory", "Cache directory", deps.CacheDir, SeverityWarn)

	return results
}

func checkWritableDir(results []CheckResult, name, label string, resolveDir func() (string, error), failSeverity Severity) []CheckResult {
	dir, err := resolveDir()
	if err != nil {
		return append(results, CheckResult{
			Category: "Configuration & Storage",
			Name:     name,
			Status:   failSeverity,
			Message:  fmt.Sprintf("Failed to resolve %s: %v", strings.ToLower(label), err),
		})
	}
	exists, writeErr := checkDirWritable(dir)
	if writeErr != nil {
		msg := fmt.Sprintf("%s is not writable", label)
		hint := fmt.Sprintf("Ensure %s has write permissions.", dir)
		if !exists {
			msg = fmt.Sprintf("%s cannot be created or is not writable", label)
			hint = fmt.Sprintf("Ensure %s exists and has write permissions.", dir)
		}
		return append(results, CheckResult{
			Category: "Configuration & Storage",
			Name:     name,
			Status:   failSeverity,
			Message:  msg,
			Detail:   writeErr.Error(),
			FixHint:  hint,
		})
	}
	return append(results, CheckResult{
		Category: "Configuration & Storage",
		Name:     name,
		Status:   SeverityOK,
		Message:  fmt.Sprintf("%s is writable", label),
		Detail:   dir,
	})
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
