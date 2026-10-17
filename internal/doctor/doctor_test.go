package doctor

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/config"
	"github.com/JhnFrankz/upp/internal/lock"
	"github.com/JhnFrankz/upp/internal/platform"
)

func TestHasErrorsAndWarnings(t *testing.T) {
	tests := []struct {
		name         string
		results      []CheckResult
		wantErrors   bool
		wantWarnings bool
	}{
		{
			name:         "empty",
			results:      nil,
			wantErrors:   false,
			wantWarnings: false,
		},
		{
			name: "all ok",
			results: []CheckResult{
				{Status: SeverityOK, Message: "All good"},
			},
			wantErrors:   false,
			wantWarnings: false,
		},
		{
			name: "has warning",
			results: []CheckResult{
				{Status: SeverityOK},
				{Status: SeverityWarn, Message: "Warning"},
			},
			wantErrors:   false,
			wantWarnings: true,
		},
		{
			name: "has error",
			results: []CheckResult{
				{Status: SeverityOK},
				{Status: SeverityError, Message: "Error"},
			},
			wantErrors:   true,
			wantWarnings: false,
		},
		{
			name: "has both",
			results: []CheckResult{
				{Status: SeverityWarn},
				{Status: SeverityError},
			},
			wantErrors:   true,
			wantWarnings: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasErrors(tt.results); got != tt.wantErrors {
				t.Errorf("HasErrors() = %v, want %v", got, tt.wantErrors)
			}
			if got := HasWarnings(tt.results); got != tt.wantWarnings {
				t.Errorf("HasWarnings() = %v, want %v", got, tt.wantWarnings)
			}
		})
	}
}

func baseTestDeps(t *testing.T) DoctorDeps {
	t.Helper()
	tmp := t.TempDir()
	cfgDir := filepath.Join(tmp, "config")
	cacheDir := filepath.Join(tmp, "cache")
	_ = os.MkdirAll(cfgDir, 0o755)
	_ = os.MkdirAll(cacheDir, 0o755)

	cfgPath := filepath.Join(cfgDir, "config.toml")
	_ = os.WriteFile(cfgPath, []byte("version = 1\n"), 0o644)

	return DoctorDeps{
		ConfigPath: func() (string, error) {
			return cfgPath, nil
		},
		ConfigDir: func() (string, error) {
			return cfgDir, nil
		},
		CacheDir: func() (string, error) {
			return cacheDir, nil
		},
		LoadConfig: func() (*config.Config, error) {
			return &config.Config{Version: 1, Tools: make(map[string]config.ToolConfig)}, nil
		},
		LockPath: func() (string, error) {
			return filepath.Join(cacheDir, "upp.lock"), nil
		},
		ProcessAlive: func(pid int) bool {
			return false
		},
		LookPath: func(name string) (string, error) {
			return "/usr/bin/" + name, nil
		},
		FindAllPaths: func(name string) []string {
			return []string{"/usr/bin/" + name}
		},
		Platform: platform.Platform{OS: platform.OSLinux, Arch: "x86_64"},
		Adapters: nil,
		HTTPGet: func(ctx context.Context, url string) (int, error) {
			return http.StatusOK, nil
		},
	}
}

func findResult(results []CheckResult, category, name string) *CheckResult {
	for i := range results {
		if (category == "" || results[i].Category == category) &&
			(name == "" || results[i].Name == name || strings.Contains(results[i].Name, name)) {
			return &results[i]
		}
	}
	return nil
}

func TestDiagnose_StorageAndConfig(t *testing.T) {
	t.Run("valid config and writable directories", func(t *testing.T) {
		deps := baseTestDeps(t)
		results := Diagnose(context.Background(), deps)

		cfgRes := findResult(results, "Configuration & Storage", "Config File")
		if cfgRes == nil {
			t.Fatalf("expected Config File result, got results: %+v", results)
		}
		if cfgRes.Status != SeverityOK {
			t.Errorf("expected SeverityOK for valid config, got %v: %s", cfgRes.Status, cfgRes.Message)
		}

		cfgDirRes := findResult(results, "Configuration & Storage", "Config Directory")
		if cfgDirRes == nil || cfgDirRes.Status != SeverityOK {
			t.Errorf("expected SeverityOK for config dir, got %+v", cfgDirRes)
		}

		cacheDirRes := findResult(results, "Configuration & Storage", "Cache Directory")
		if cacheDirRes == nil || cacheDirRes.Status != SeverityOK {
			t.Errorf("expected SeverityOK for cache dir, got %+v", cacheDirRes)
		}
	})

	t.Run("missing config file", func(t *testing.T) {
		deps := baseTestDeps(t)
		deps.ConfigPath = func() (string, error) {
			return filepath.Join(t.TempDir(), "nonexistent", "config.toml"), nil
		}
		results := Diagnose(context.Background(), deps)

		cfgRes := findResult(results, "Configuration & Storage", "Config File")
		if cfgRes == nil {
			t.Fatal("missing Config File result")
		}
		if cfgRes.Status != SeverityWarn {
			t.Errorf("expected SeverityWarn for missing config, got %v", cfgRes.Status)
		}
		if !strings.Contains(cfgRes.FixHint, "upp init") {
			t.Errorf("expected fix hint to mention 'upp init', got %q", cfgRes.FixHint)
		}
	})

	t.Run("invalid config TOML", func(t *testing.T) {
		deps := baseTestDeps(t)
		deps.LoadConfig = func() (*config.Config, error) {
			return nil, errors.New("invalid TOML syntax")
		}
		results := Diagnose(context.Background(), deps)

		cfgRes := findResult(results, "Configuration & Storage", "Config File")
		if cfgRes == nil {
			t.Fatal("missing Config File result")
		}
		if cfgRes.Status != SeverityError {
			t.Errorf("expected SeverityError for invalid config, got %v", cfgRes.Status)
		}
	})

	t.Run("config path resolution error fix hint", func(t *testing.T) {
		deps := baseTestDeps(t)
		deps.ConfigPath = func() (string, error) {
			return "", errors.New("cannot determine home directory")
		}
		results := Diagnose(context.Background(), deps)

		cfgRes := findResult(results, "Configuration & Storage", "Config File")
		if cfgRes == nil {
			t.Fatal("missing Config File result")
		}
		if cfgRes.Status != SeverityError {
			t.Errorf("expected SeverityError, got %v", cfgRes.Status)
		}
		if strings.Contains(cfgRes.FixHint, "XDG_CONFIG_HOME") {
			t.Errorf("FixHint must not mention XDG_CONFIG_HOME per spec, got: %q", cfgRes.FixHint)
		}
		if !strings.Contains(cfgRes.FixHint, "HOME or APPDATA") {
			t.Errorf("FixHint must mention HOME or APPDATA, got: %q", cfgRes.FixHint)
		}
	})

	t.Run("unwritable config directory", func(t *testing.T) {
		deps := baseTestDeps(t)
		regularFile := filepath.Join(t.TempDir(), "blocking_file")
		if err := os.WriteFile(regularFile, []byte("block"), 0o644); err != nil {
			t.Fatal(err)
		}
		deps.ConfigDir = func() (string, error) {
			return filepath.Join(regularFile, "never_writable_subdir"), nil
		}
		results := Diagnose(context.Background(), deps)

		cfgDirRes := findResult(results, "Configuration & Storage", "Config Directory")
		if cfgDirRes == nil {
			t.Fatal("missing Config Directory result")
		}
		if cfgDirRes.Status != SeverityError {
			t.Errorf("expected SeverityError for unwritable config dir, got %v", cfgDirRes.Status)
		}
	})

	t.Run("non-existent config and cache directories are not created on disk (read-only)", func(t *testing.T) {
		deps := baseTestDeps(t)
		tmp := t.TempDir()
		missingCfgDir := filepath.Join(tmp, "nested", "config", "upp")
		missingCacheDir := filepath.Join(tmp, "nested", "cache", "upp")
		deps.ConfigDir = func() (string, error) { return missingCfgDir, nil }
		deps.CacheDir = func() (string, error) { return missingCacheDir, nil }

		results := Diagnose(context.Background(), deps)

		cfgDirRes := findResult(results, "Configuration & Storage", "Config Directory")
		if cfgDirRes == nil || cfgDirRes.Status != SeverityOK {
			t.Errorf("expected SeverityOK when ancestor of missing config dir is writable, got %+v", cfgDirRes)
		}
		cacheDirRes := findResult(results, "Configuration & Storage", "Cache Directory")
		if cacheDirRes == nil || cacheDirRes.Status != SeverityOK {
			t.Errorf("expected SeverityOK when ancestor of missing cache dir is writable, got %+v", cacheDirRes)
		}

		if _, err := os.Stat(filepath.Join(tmp, "nested")); !os.IsNotExist(err) {
			t.Errorf("Diagnose must not create missing directories on disk; stat err = %v", err)
		}
	})
}

func TestDiagnose_ProcessLock(t *testing.T) {
	t.Run("no lock file present", func(t *testing.T) {
		deps := baseTestDeps(t)
		results := Diagnose(context.Background(), deps)

		lockRes := findResult(results, "Process Lock", "Process Lock")
		if lockRes == nil {
			t.Fatal("missing Process Lock result")
		}
		if lockRes.Status != SeverityOK {
			t.Errorf("expected SeverityOK when no lock file, got %v", lockRes.Status)
		}
	})

	t.Run("stale lock with dead PID", func(t *testing.T) {
		deps := baseTestDeps(t)
		tmp := t.TempDir()
		lockPath := filepath.Join(tmp, "upp.lock")
		_ = os.WriteFile(lockPath, []byte("999999\n"), 0o644)

		deps.LockPath = func() (string, error) { return lockPath, nil }
		deps.ProcessAlive = func(pid int) bool { return false }

		results := Diagnose(context.Background(), deps)
		lockRes := findResult(results, "Process Lock", "Process Lock")
		if lockRes == nil {
			t.Fatal("missing Process Lock result")
		}
		if lockRes.Status != SeverityWarn {
			t.Errorf("expected SeverityWarn for stale lock, got %v", lockRes.Status)
		}
		if !strings.Contains(lockRes.Message, "Stale") && !strings.Contains(lockRes.Message, "dead") {
			t.Errorf("expected message to mention stale/dead, got %q", lockRes.Message)
		}
		if !strings.Contains(lockRes.FixHint, "rm") && !strings.Contains(lockRes.FixHint, "Remove") {
			t.Errorf("expected FixHint to suggest removing lock, got %q", lockRes.FixHint)
		}
	})

	t.Run("active process holds lock via TryLock fake", func(t *testing.T) {
		deps := baseTestDeps(t)
		tmp := t.TempDir()
		lockPath := filepath.Join(tmp, "upp.lock")
		_ = os.WriteFile(lockPath, []byte("1234\n"), 0o644)

		deps.LockPath = func() (string, error) { return lockPath, nil }
		deps.ProcessAlive = func(pid int) bool { return pid == 1234 }
		deps.TryLock = func(f *os.File) error { return lock.ErrLocked }

		results := Diagnose(context.Background(), deps)
		lockRes := findResult(results, "Process Lock", "Process Lock")
		if lockRes == nil {
			t.Fatal("missing Process Lock result")
		}
		if lockRes.Status != SeverityWarn {
			t.Errorf("expected SeverityWarn for active lock held, got %v", lockRes.Status)
		}
		if !strings.Contains(lockRes.Message, "Lock held by active process") || !strings.Contains(lockRes.Message, "1234") {
			t.Errorf("expected message to mention active process and PID 1234, got %q", lockRes.Message)
		}
	})

	t.Run("active process holds lock via real acquire", func(t *testing.T) {
		deps := baseTestDeps(t)
		tmp := t.TempDir()
		lockPath := filepath.Join(tmp, "upp.lock")
		l, err := lock.Acquire(lockPath)
		if err != nil {
			t.Fatalf("failed to acquire lock: %v", err)
		}
		defer func() { _ = l.Release() }()

		deps.LockPath = func() (string, error) { return lockPath, nil }

		results := Diagnose(context.Background(), deps)
		lockRes := findResult(results, "Process Lock", "Process Lock")
		if lockRes == nil {
			t.Fatal("missing Process Lock result")
		}
		if lockRes.Status != SeverityWarn {
			t.Errorf("expected SeverityWarn for active lock held, got %v", lockRes.Status)
		}
		if !strings.Contains(lockRes.Message, "Lock held by active process") {
			t.Errorf("expected message to mention active process, got %q", lockRes.Message)
		}
	})

	t.Run("lock clean release removes file and doctor reports SeverityOK", func(t *testing.T) {
		deps := baseTestDeps(t)
		tmp := t.TempDir()
		lockPath := filepath.Join(tmp, "upp.lock")
		l, err := lock.Acquire(lockPath)
		if err != nil {
			t.Fatalf("failed to acquire lock: %v", err)
		}
		if err := l.Release(); err != nil {
			t.Fatalf("failed to release lock: %v", err)
		}

		deps.LockPath = func() (string, error) { return lockPath, nil }

		results := Diagnose(context.Background(), deps)
		lockRes := findResult(results, "Process Lock", "Process Lock")
		if lockRes == nil {
			t.Fatal("missing Process Lock result")
		}
		if lockRes.Status != SeverityOK {
			t.Errorf("expected SeverityOK after lock release, got %v", lockRes.Status)
		}
	})

	t.Run("stale lock with alive PID not holding lock", func(t *testing.T) {
		deps := baseTestDeps(t)
		tmp := t.TempDir()
		lockPath := filepath.Join(tmp, "upp.lock")
		_ = os.WriteFile(lockPath, []byte("5678\n"), 0o644)

		deps.LockPath = func() (string, error) { return lockPath, nil }
		deps.ProcessAlive = func(pid int) bool { return pid == 5678 }
		deps.TryLock = func(f *os.File) error { return nil }

		results := Diagnose(context.Background(), deps)
		lockRes := findResult(results, "Process Lock", "Process Lock")
		if lockRes == nil {
			t.Fatal("missing Process Lock result")
		}
		if lockRes.Status != SeverityWarn {
			t.Errorf("expected SeverityWarn for stale lock, got %v", lockRes.Status)
		}
		if !strings.Contains(lockRes.Message, "does not hold lock") {
			t.Errorf("expected message to mention does not hold lock, got %q", lockRes.Message)
		}
	})

	t.Run("tryLock unexpected error", func(t *testing.T) {
		deps := baseTestDeps(t)
		tmp := t.TempDir()
		lockPath := filepath.Join(tmp, "upp.lock")
		_ = os.WriteFile(lockPath, []byte("1234\n"), 0o644)

		deps.LockPath = func() (string, error) { return lockPath, nil }
		deps.TryLock = func(f *os.File) error { return errors.New("simulated I/O failure") }

		results := Diagnose(context.Background(), deps)
		lockRes := findResult(results, "Process Lock", "Process Lock")
		if lockRes == nil {
			t.Fatal("missing Process Lock result")
		}
		if lockRes.Status != SeverityWarn {
			t.Errorf("expected SeverityWarn for tryLock error, got %v", lockRes.Status)
		}
		if !strings.Contains(lockRes.Message, "Cannot test lock on file") {
			t.Errorf("expected message to mention Cannot test lock on file, got %q", lockRes.Message)
		}
	})

	t.Run("corrupted lock file", func(t *testing.T) {
		deps := baseTestDeps(t)
		tmp := t.TempDir()
		lockPath := filepath.Join(tmp, "upp.lock")
		_ = os.WriteFile(lockPath, []byte("not-a-number\n"), 0o644)

		deps.LockPath = func() (string, error) { return lockPath, nil }

		results := Diagnose(context.Background(), deps)
		lockRes := findResult(results, "Process Lock", "Process Lock")
		if lockRes == nil {
			t.Fatal("missing Process Lock result")
		}
		if lockRes.Status != SeverityWarn {
			t.Errorf("expected SeverityWarn for corrupted lock, got %v", lockRes.Status)
		}
	})
}

func TestDiagnose_PackageManagers(t *testing.T) {
	t.Run("single package manager detected on Linux", func(t *testing.T) {
		deps := baseTestDeps(t)
		deps.Platform = platform.Platform{OS: platform.OSLinux, Arch: "x86_64"}
		deps.LookPath = func(name string) (string, error) {
			if name == "apt" || name == "apt-get" {
				return "/usr/bin/apt", nil
			}
			return "", os.ErrNotExist
		}

		results := Diagnose(context.Background(), deps)
		pmRes := findResult(results, "Package Managers", "Package Managers")
		if pmRes == nil {
			t.Fatal("missing Package Managers result")
		}
		if pmRes.Status != SeverityOK {
			t.Errorf("expected SeverityOK for single manager, got %v: %s", pmRes.Status, pmRes.Message)
		}
	})

	t.Run("Linux with brew and apt conflict warning", func(t *testing.T) {
		deps := baseTestDeps(t)
		deps.Platform = platform.Platform{OS: platform.OSLinux, Arch: "x86_64"}
		deps.LookPath = func(name string) (string, error) {
			if name == "brew" {
				return "/home/linuxbrew/.linuxbrew/bin/brew", nil
			}
			if name == "apt" || name == "apt-get" {
				return "/usr/bin/apt", nil
			}
			return "", os.ErrNotExist
		}

		results := Diagnose(context.Background(), deps)
		pmRes := findResult(results, "Package Managers", "Package Managers")
		if pmRes == nil {
			t.Fatal("missing Package Managers result")
		}
		if pmRes.Status != SeverityWarn {
			t.Errorf("expected SeverityWarn for brew + apt conflict on Linux, got %v", pmRes.Status)
		}
		if !strings.Contains(pmRes.Message, "brew") || !strings.Contains(pmRes.Message, "apt") {
			t.Errorf("expected message to mention brew and apt, got %q", pmRes.Message)
		}
	})

	t.Run("no package manager detected", func(t *testing.T) {
		deps := baseTestDeps(t)
		deps.LookPath = func(name string) (string, error) {
			return "", os.ErrNotExist
		}

		results := Diagnose(context.Background(), deps)
		pmRes := findResult(results, "Package Managers", "Package Managers")
		if pmRes == nil {
			t.Fatal("missing Package Managers result")
		}
		if pmRes.Status != SeverityWarn {
			t.Errorf("expected SeverityWarn when no manager detected, got %v", pmRes.Status)
		}
	})
}

// mockAdapter is a minimal Adapter for tests.
type mockAdapter struct {
	name      string
	installed bool
}

func (m *mockAdapter) Name() string { return m.name }
func (m *mockAdapter) Detect() bool { return m.installed }
func (m *mockAdapter) Check(ctx context.Context) (adapters.UpdateInfo, error) {
	return adapters.UpdateInfo{}, nil
}
func (m *mockAdapter) Update(ctx context.Context, dryRun bool) (adapters.Result, error) {
	return adapters.Result{}, nil
}
func (m *mockAdapter) Info() adapters.ToolInfo {
	return adapters.ToolInfo{ID: m.name, Name: m.name}
}

func TestDiagnose_ToolPaths(t *testing.T) {
	t.Run("tool single installation", func(t *testing.T) {
		deps := baseTestDeps(t)
		deps.Adapters = []adapters.Adapter{
			&mockAdapter{name: "go", installed: true},
		}
		deps.FindAllPaths = func(name string) []string {
			if name == "go" {
				return []string{"/usr/local/go/bin/go"}
			}
			return nil
		}
		deps.LookPath = func(name string) (string, error) {
			if name == "go" {
				return "/usr/local/go/bin/go", nil
			}
			return "", os.ErrNotExist
		}

		results := Diagnose(context.Background(), deps)
		toolRes := findResult(results, "Tool Paths", "go")
		if toolRes == nil {
			t.Fatal("missing Tool Paths result for go")
		}
		if toolRes.Status != SeverityOK {
			t.Errorf("expected SeverityOK for unshadowed tool, got %v: %s", toolRes.Status, toolRes.Message)
		}
	})

	t.Run("tool shadowed installation", func(t *testing.T) {
		deps := baseTestDeps(t)
		deps.Adapters = []adapters.Adapter{
			&mockAdapter{name: "node", installed: true},
		}
		deps.FindAllPaths = func(name string) []string {
			if name == "node" {
				return []string{"/home/user/.nvm/versions/node/v20/bin/node", "/usr/bin/node"}
			}
			return nil
		}

		results := Diagnose(context.Background(), deps)
		toolRes := findResult(results, "Tool Paths", "node")
		if toolRes == nil {
			t.Fatal("missing Tool Paths result for node")
		}
		if toolRes.Status != SeverityWarn {
			t.Errorf("expected SeverityWarn for shadowed tool, got %v", toolRes.Status)
		}
		if !strings.Contains(toolRes.Message, "shadowed") {
			t.Errorf("expected message to mention shadowed, got %q", toolRes.Message)
		}
		if !strings.Contains(toolRes.Detail, "/usr/bin/node") {
			t.Errorf("expected Detail to show secondary path, got %q", toolRes.Detail)
		}
	})

	t.Run("configured tool missing from PATH", func(t *testing.T) {
		deps := baseTestDeps(t)
		deps.LoadConfig = func() (*config.Config, error) {
			return &config.Config{
				Version: 1,
				Tools: map[string]config.ToolConfig{
					"uv": {Enabled: true},
				},
			}, nil
		}
		deps.FindAllPaths = func(name string) []string {
			return nil
		}
		deps.LookPath = func(name string) (string, error) {
			return "", os.ErrNotExist
		}

		results := Diagnose(context.Background(), deps)
		toolRes := findResult(results, "Tool Paths", "uv")
		if toolRes == nil {
			t.Fatal("missing Tool Paths result for missing uv tool")
		}
		if toolRes.Status != SeverityWarn {
			t.Errorf("expected SeverityWarn for missing tool, got %v", toolRes.Status)
		}
	})

	t.Run("defaulted catalog tools from ApplyDefaults do not warn when uninstalled", func(t *testing.T) {
		deps := baseTestDeps(t)
		deps.Adapters = []adapters.Adapter{
			&mockAdapter{name: "uv", installed: false},
			&mockAdapter{name: "bun", installed: false},
		}
		deps.LoadConfig = func() (*config.Config, error) {
			cfg := config.DefaultConfig()
			cfg.Tools["uv"] = config.ToolConfig{Enabled: true}
			config.ApplyDefaults(cfg)
			return cfg, nil
		}
		deps.FindAllPaths = func(name string) []string { return nil }
		deps.LookPath = func(name string) (string, error) { return "", os.ErrNotExist }

		results := Diagnose(context.Background(), deps)
		if uvRes := findResult(results, "Tool Paths", "uv"); uvRes == nil || uvRes.Status != SeverityWarn {
			t.Errorf("expected explicit tool uv to warn when missing, got %+v", uvRes)
		}
		if bunRes := findResult(results, "Tool Paths", "bun"); bunRes != nil {
			t.Errorf("expected ApplyDefaults-injected uninstalled tool bun not to warn, got %+v", bunRes)
		}
	})

	t.Run("disabled custom tool via tools section is skipped", func(t *testing.T) {
		deps := baseTestDeps(t)
		deps.Adapters = []adapters.Adapter{}
		deps.LoadConfig = func() (*config.Config, error) {
			return &config.Config{
				Version: 1,
				Tools: map[string]config.ToolConfig{
					"disabled-custom": {Enabled: false},
				},
				Custom: map[string]config.CustomTool{
					"disabled-custom": {Command: "nonexistent-bin --update"},
				},
			}, nil
		}
		deps.FindAllPaths = func(name string) []string { return nil }
		deps.LookPath = func(name string) (string, error) { return "", os.ErrNotExist }

		results := Diagnose(context.Background(), deps)
		if res := findResult(results, "Tool Paths", "disabled-custom"); res != nil {
			t.Errorf("expected disabled custom tool not to be checked or warned, got %+v", res)
		}
	})

	t.Run("custom tool probes executable extracted from command", func(t *testing.T) {
		deps := baseTestDeps(t)
		deps.Adapters = []adapters.Adapter{}
		deps.LoadConfig = func() (*config.Config, error) {
			return &config.Config{
				Version: 1,
				Tools:   map[string]config.ToolConfig{},
				Custom: map[string]config.CustomTool{
					"my-rust-updater": {Command: "FOO=1 sudo -u root rustup update"},
				},
			}, nil
		}
		deps.FindAllPaths = func(name string) []string {
			if name == "rustup" {
				return []string{"/home/user/.cargo/bin/rustup"}
			}
			return nil
		}
		deps.LookPath = func(name string) (string, error) {
			if name == "rustup" {
				return "/home/user/.cargo/bin/rustup", nil
			}
			return "", os.ErrNotExist
		}

		results := Diagnose(context.Background(), deps)
		res := findResult(results, "Tool Paths", "my-rust-updater")
		if res == nil {
			t.Fatal("missing Tool Paths result for my-rust-updater")
		}
		if res.Status != SeverityOK {
			t.Errorf("expected SeverityOK when custom command binary 'rustup' is in PATH, got %v: %s", res.Status, res.Message)
		}
	})

	t.Run("apt avoids false-positive missing warning when apt-get present or Detect succeeds", func(t *testing.T) {
		deps := baseTestDeps(t)
		deps.Adapters = []adapters.Adapter{
			&mockAdapter{name: "apt", installed: true},
		}
		deps.LoadConfig = func() (*config.Config, error) {
			return &config.Config{
				Version: 1,
				Tools: map[string]config.ToolConfig{
					"apt": {Enabled: true},
				},
			}, nil
		}
		deps.FindAllPaths = func(name string) []string {
			if name == "apt-get" {
				return []string{"/usr/bin/apt-get"}
			}
			return nil
		}
		deps.LookPath = func(name string) (string, error) {
			if name == "apt-get" {
				return "/usr/bin/apt-get", nil
			}
			return "", os.ErrNotExist
		}

		results := Diagnose(context.Background(), deps)
		res := findResult(results, "Tool Paths", "apt")
		if res == nil {
			t.Fatal("missing Tool Paths result for apt")
		}
		if res.Status != SeverityOK {
			t.Errorf("expected SeverityOK for apt when apt-get/Detect succeeds, got %v: %s", res.Status, res.Message)
		}
	})
}

func TestDiagnose_Network(t *testing.T) {
	t.Run("network reachable", func(t *testing.T) {
		deps := baseTestDeps(t)
		deps.HTTPGet = func(ctx context.Context, url string) (int, error) {
			return http.StatusOK, nil
		}

		results := Diagnose(context.Background(), deps)
		ghRes := findResult(results, "Network", "api.github.com")
		if ghRes == nil {
			t.Fatal("missing GitHub API network result")
		}
		if ghRes.Status != SeverityOK {
			t.Errorf("expected SeverityOK for reachable github, got %v", ghRes.Status)
		}

		goRes := findResult(results, "Network", "go.dev")
		if goRes == nil {
			t.Fatal("missing go.dev network result")
		}
		if goRes.Status != SeverityOK {
			t.Errorf("expected SeverityOK for reachable go.dev, got %v", goRes.Status)
		}
	})

	t.Run("network unreachable", func(t *testing.T) {
		deps := baseTestDeps(t)
		deps.HTTPGet = func(ctx context.Context, url string) (int, error) {
			return 0, fmt.Errorf("connection refused to %s", url)
		}

		results := Diagnose(context.Background(), deps)
		ghRes := findResult(results, "Network", "api.github.com")
		if ghRes == nil {
			t.Fatal("missing GitHub API network result")
		}
		if ghRes.Status != SeverityWarn {
			t.Errorf("expected SeverityWarn for unreachable network, got %v", ghRes.Status)
		}
		if ghRes.FixHint == "" {
			t.Errorf("expected FixHint for network error, got empty")
		}
	})
}

func TestDefaultFindAllPaths_SymlinkAndHardlinkDeduplication(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink tests require special privileges on Windows")
	}

	t.Run("symlink across directories deduplicated", func(t *testing.T) {
		tmp := t.TempDir()
		binDir := filepath.Join(tmp, "bin")
		usrBinDir := filepath.Join(tmp, "usr_bin")
		if err := os.MkdirAll(binDir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(usrBinDir, 0o755); err != nil {
			t.Fatal(err)
		}

		targetBin := filepath.Join(usrBinDir, "tool")
		if err := os.WriteFile(targetBin, []byte("#!/bin/sh\necho hi\n"), 0o755); err != nil {
			t.Fatal(err)
		}
		symlinkBin := filepath.Join(binDir, "tool")
		if err := os.Symlink(targetBin, symlinkBin); err != nil {
			t.Fatal(err)
		}

		t.Setenv("PATH", binDir+string(filepath.ListSeparator)+usrBinDir)

		found := defaultFindAllPaths("tool")
		if len(found) != 1 {
			t.Fatalf("expected 1 deduplicated path, got %d: %v", len(found), found)
		}

		// Diagnose integration check: should not trigger shadowed warning
		deps := baseTestDeps(t)
		deps.FindAllPaths = defaultFindAllPaths
		deps.LookPath = func(name string) (string, error) {
			if name == "tool" {
				return found[0], nil
			}
			return "", os.ErrNotExist
		}
		deps.Adapters = []adapters.Adapter{
			&mockAdapter{name: "tool", installed: true},
		}

		results := Diagnose(context.Background(), deps)
		res := findResult(results, "Tool Paths", "tool")
		if res == nil {
			t.Fatal("missing Tool Paths result for tool")
		}
		if res.Status != SeverityOK {
			t.Errorf("expected SeverityOK (no shadowing), got %v: %s", res.Status, res.Message)
		}
	})

	t.Run("hardlink across directories deduplicated", func(t *testing.T) {
		tmp := t.TempDir()
		dir1 := filepath.Join(tmp, "dir1")
		dir2 := filepath.Join(tmp, "dir2")
		if err := os.MkdirAll(dir1, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(dir2, 0o755); err != nil {
			t.Fatal(err)
		}

		bin1 := filepath.Join(dir1, "tool")
		if err := os.WriteFile(bin1, []byte("#!/bin/sh\necho same\n"), 0o755); err != nil {
			t.Fatal(err)
		}
		bin2 := filepath.Join(dir2, "tool")
		if err := os.Link(bin1, bin2); err != nil {
			t.Fatal(err)
		}

		t.Setenv("PATH", dir1+string(filepath.ListSeparator)+dir2)

		found := defaultFindAllPaths("tool")
		if len(found) != 1 {
			t.Fatalf("expected 1 deduplicated path for hardlinks, got %d: %v", len(found), found)
		}
	})

	t.Run("distinct binaries both returned and shadowed status triggered", func(t *testing.T) {
		tmp := t.TempDir()
		dir1 := filepath.Join(tmp, "dir1")
		dir2 := filepath.Join(tmp, "dir2")
		if err := os.MkdirAll(dir1, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(dir2, 0o755); err != nil {
			t.Fatal(err)
		}

		bin1 := filepath.Join(dir1, "tool")
		if err := os.WriteFile(bin1, []byte("#!/bin/sh\necho v1\n"), 0o755); err != nil {
			t.Fatal(err)
		}
		bin2 := filepath.Join(dir2, "tool")
		if err := os.WriteFile(bin2, []byte("#!/bin/sh\necho v2\n"), 0o755); err != nil {
			t.Fatal(err)
		}

		t.Setenv("PATH", dir1+string(filepath.ListSeparator)+dir2)

		found := defaultFindAllPaths("tool")
		if len(found) != 2 {
			t.Fatalf("expected 2 distinct paths, got %d: %v", len(found), found)
		}

		// Diagnose integration check: should trigger shadowed warning
		deps := baseTestDeps(t)
		deps.FindAllPaths = defaultFindAllPaths
		deps.LookPath = func(name string) (string, error) {
			if name == "tool" {
				return found[0], nil
			}
			return "", os.ErrNotExist
		}
		deps.Adapters = []adapters.Adapter{
			&mockAdapter{name: "tool", installed: true},
		}

		results := Diagnose(context.Background(), deps)
		res := findResult(results, "Tool Paths", "tool")
		if res == nil {
			t.Fatal("missing Tool Paths result for tool")
		}
		if res.Status != SeverityWarn {
			t.Errorf("expected SeverityWarn (shadowed), got %v: %s", res.Status, res.Message)
		}
		if !strings.Contains(res.Message, "shadowed") {
			t.Errorf("expected shadowed message, got %q", res.Message)
		}
	})
}

func TestDoctor_NetworkParallel(t *testing.T) {
	deps := DoctorDeps{
		HTTPGet: func(ctx context.Context, url string) (int, error) {
			time.Sleep(50 * time.Millisecond)
			return 200, nil
		},
	}

	start := time.Now()
	results := checkNetwork(context.Background(), nil, deps)
	duration := time.Since(start)

	if duration >= 85*time.Millisecond {
		t.Errorf("checkNetwork took %v, expected < 85ms (indicating parallel execution)", duration)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Name != "api.github.com" {
		t.Errorf("results[0].Name = %q, want api.github.com", results[0].Name)
	}
	if results[1].Name != "go.dev" {
		t.Errorf("results[1].Name = %q, want go.dev", results[1].Name)
	}
	for i, res := range results {
		if res.Status != SeverityOK {
			t.Errorf("results[%d].Status = %v, want SeverityOK", i, res.Status)
		}
	}
}

func TestDoctor_Network_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	deps := DoctorDeps{
		HTTPGet: func(ctx context.Context, url string) (int, error) {
			t.Errorf("HTTPGet called unexpectedly for %s with canceled context", url)
			return 200, nil
		},
	}

	start := time.Now()
	results := checkNetwork(ctx, nil, deps)
	duration := time.Since(start)

	if duration >= 50*time.Millisecond {
		t.Errorf("checkNetwork with canceled context took %v, expected immediate abort", duration)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	for i, res := range results {
		if res.Status != SeverityWarn {
			t.Errorf("results[%d].Status = %v, want SeverityWarn", i, res.Status)
		}
		if !strings.Contains(res.Message, "aborted") && !strings.Contains(res.Message, "canceled") {
			t.Errorf("results[%d].Message = %q, want abort/canceled message", i, res.Message)
		}
	}
}

func TestDiagnose_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	invoked := false
	deps := baseTestDeps(t)
	deps.ConfigPath = func() (string, error) {
		invoked = true
		return "", errors.New("should not be called")
	}

	results := Diagnose(ctx, deps)
	if len(results) != 0 {
		t.Fatalf("Diagnose with canceled context returned %d results, want 0", len(results))
	}
	if invoked {
		t.Fatal("Diagnose with canceled context should return early without executing check phases")
	}
}

func TestGetPathDirs_CachingAndInvalidation(t *testing.T) {
	t.Setenv("PATH", "")
	if got := getPathDirs(); got != nil {
		t.Errorf("getPathDirs() on empty PATH = %v, want nil", got)
	}

	testPath := "/bin" + string(filepath.ListSeparator) + "/usr/bin"
	t.Setenv("PATH", testPath)
	dirs1 := getPathDirs()
	want0 := filepath.Clean("/bin")
	want1 := filepath.Clean("/usr/bin")
	if len(dirs1) != 2 || dirs1[0] != want0 || dirs1[1] != want1 {
		t.Fatalf("unexpected getPathDirs() = %v, want [%s %s]", dirs1, want0, want1)
	}

	dirs2 := getPathDirs()
	if len(dirs2) != len(dirs1) || &dirs1[0] != &dirs2[0] {
		t.Errorf("expected cached slice pointer to be reused, got %p vs %p", dirs1, dirs2)
	}

	newPath := "/opt/bin"
	t.Setenv("PATH", newPath)
	dirs3 := getPathDirs()
	want3 := filepath.Clean(newPath)
	if len(dirs3) != 1 || dirs3[0] != want3 {
		t.Fatalf("expected invalidated cache with new path, got %v, want [%s]", dirs3, want3)
	}
}

func TestGetPathDirs_Deduplication(t *testing.T) {
	sep := string(filepath.ListSeparator)
	// Includes duplicates and uncleaned paths (trailing slashes)
	dupPath := "/usr/bin" + sep + "/bin" + sep + "/usr/bin/" + sep + "/bin" + sep + "/opt/bin"
	t.Setenv("PATH", dupPath)

	dirs := getPathDirs()
	want := []string{filepath.Clean("/usr/bin"), filepath.Clean("/bin"), filepath.Clean("/opt/bin")}
	if len(dirs) != len(want) {
		t.Fatalf("got %d dirs (%v), want %d (%v)", len(dirs), dirs, len(want), want)
	}
	for i, w := range want {
		if dirs[i] != w {
			t.Errorf("dirs[%d] = %q, want %q", i, dirs[i], w)
		}
	}
}

func TestGetPathDirs_Concurrent(t *testing.T) {
	orig := os.Getenv("PATH")
	defer func() {
		_ = os.Setenv("PATH", orig)
	}()

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				_ = getPathDirs()
			}
		}(i)
	}
	wg.Wait()
}

func TestDiagnose_LoadConfigMemoized(t *testing.T) {
	deps := baseTestDeps(t)
	loadCount := 0
	deps.LoadConfig = func() (*config.Config, error) {
		loadCount++
		return &config.Config{
			Version: 1,
			Tools: map[string]config.ToolConfig{
				"git": {Enabled: true},
			},
		}, nil
	}

	results := Diagnose(context.Background(), deps)
	if len(results) == 0 {
		t.Fatalf("expected diagnostic results, got empty")
	}

	if loadCount != 1 {
		t.Fatalf("expected deps.LoadConfig to be called exactly once, got %d", loadCount)
	}
}

type mockEnvToolAdapter struct {
	name     string
	detected bool
}

func (m *mockEnvToolAdapter) Name() string { return m.name }
func (m *mockEnvToolAdapter) Detect() bool { return m.detected }
func (m *mockEnvToolAdapter) Check(context.Context) (adapters.UpdateInfo, error) {
	return adapters.UpdateInfo{}, nil
}
func (m *mockEnvToolAdapter) Update(context.Context, bool) (adapters.Result, error) {
	return adapters.Result{}, nil
}
func (m *mockEnvToolAdapter) Info() adapters.ToolInfo {
	return adapters.ToolInfo{ID: m.name, Name: m.name}
}

func TestDiagnose_ShellFunctionEnvironmentTool(t *testing.T) {
	deps := baseTestDeps(t)
	deps.FindAllPaths = func(string) []string { return nil }
	deps.LookPath = func(string) (string, error) { return "", errors.New("not found in PATH") }
	deps.LoadConfig = func() (*config.Config, error) {
		return &config.Config{
			Version: 1,
			Tools: map[string]config.ToolConfig{
				"nvm": {Enabled: true},
			},
		}, nil
	}
	deps.Adapters = []adapters.Adapter{
		&mockEnvToolAdapter{name: "nvm", detected: true},
	}

	results := Diagnose(context.Background(), deps)
	res := findResult(results, "Tool Paths", "nvm")
	if res == nil {
		t.Fatal("expected result for nvm in Tool Paths")
	}
	if res.Status != SeverityOK {
		t.Errorf("expected SeverityOK for environment detected tool, got %v (msg: %s)", res.Status, res.Message)
	}
	if !strings.Contains(res.Message, "detected via environment") {
		t.Errorf("expected message to mention detected via environment, got %q", res.Message)
	}
}

func TestExtractCustomBinary(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		fallback string
		want     string
	}{
		{
			name:     "empty command uses fallback",
			command:  "   ",
			fallback: "my-tool",
			want:     "my-tool",
		},
		{
			name:     "simple binary and args",
			command:  "rustup update",
			fallback: "my-rust",
			want:     "rustup",
		},
		{
			name:     "leading env var assignments",
			command:  "FOO=bar BAZ_1=qux mybin --update",
			fallback: "custom",
			want:     "mybin",
		},
		{
			name:     "env wrapper with flags and assignments",
			command:  "/usr/bin/env -i -u OLD_VAR FOO=1 target-bin upgrade",
			fallback: "custom",
			want:     "target-bin",
		},
		{
			name:     "sudo with user and preserve-env flags",
			command:  "sudo -u root -E apt-get update",
			fallback: "custom",
			want:     "apt-get",
		},
		{
			name:     "doas with user flag",
			command:  "doas -u admin pkg-tool upgrade",
			fallback: "custom",
			want:     "pkg-tool",
		},
		{
			name:     "pkexec with user flag",
			command:  "pkexec --user root fwupdmgr update",
			fallback: "custom",
			want:     "fwupdmgr",
		},
		{
			name:     "runas on windows",
			command:  "runas /user:Administrator updater.exe /silent",
			fallback: "custom",
			want:     "updater.exe",
		},
		{
			name:     "compound command with trailing semicolon",
			command:  "mytool; echo done",
			fallback: "custom",
			want:     "mytool",
		},
		{
			name:     "only env vars falls back to name",
			command:  "FOO=bar BAZ=1",
			fallback: "fallback-bin",
			want:     "fallback-bin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extractCustomBinary(tt.command, tt.fallback); got != tt.want {
				t.Errorf("extractCustomBinary(%q, %q) = %q, want %q", tt.command, tt.fallback, got, tt.want)
			}
		})
	}
}

func TestCheckDirWritable_EdgeCases(t *testing.T) {
	t.Run("empty directory path fails", func(t *testing.T) {
		exists, err := checkDirWritable("   ")
		if exists || err == nil {
			t.Errorf("expected (false, err) for empty path, got (%v, %v)", exists, err)
		}
	})

	t.Run("target path is regular file", func(t *testing.T) {
		tmp := t.TempDir()
		f := filepath.Join(tmp, "file")
		if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		exists, err := checkDirWritable(f)
		if exists || err == nil {
			t.Errorf("expected (false, err) when target is a file, got (%v, %v)", exists, err)
		}
	})

	t.Run("read-only existing directory on unix", func(t *testing.T) {
		if runtime.GOOS == "windows" || os.Geteuid() == 0 {
			t.Skip("chmod 0555 writability test applies to non-root unix")
		}
		tmp := t.TempDir()
		roDir := filepath.Join(tmp, "ro")
		if err := os.MkdirAll(roDir, 0o555); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(roDir, 0o755) })

		exists, err := checkDirWritable(roDir)
		if !exists || err == nil {
			t.Errorf("expected (true, err) for existing read-only dir, got (%v, %v)", exists, err)
		}

		missingChild := filepath.Join(roDir, "upp", "nested")
		existsChild, errChild := checkDirWritable(missingChild)
		if existsChild || errChild == nil {
			t.Errorf("expected (false, err) for missing child under read-only ancestor, got (%v, %v)", existsChild, errChild)
		}
		if _, statErr := os.Stat(filepath.Join(roDir, "upp")); !os.IsNotExist(statErr) {
			t.Errorf("checkDirWritable must not create child directory, statErr = %v", statErr)
		}
	})
}

func TestDiagnose_AptDetectFallbackWithoutAptGetInLookPath(t *testing.T) {
	deps := baseTestDeps(t)
	deps.Adapters = []adapters.Adapter{
		&mockAdapter{name: "apt", installed: true},
	}
	deps.LoadConfig = func() (*config.Config, error) {
		return &config.Config{
			Version: 1,
			Tools: map[string]config.ToolConfig{
				"apt": {Enabled: true},
			},
		}, nil
	}
	deps.FindAllPaths = func(string) []string { return nil }
	deps.LookPath = func(string) (string, error) { return "", os.ErrNotExist }

	results := Diagnose(context.Background(), deps)
	res := findResult(results, "Tool Paths", "apt")
	if res == nil {
		t.Fatal("expected Tool Paths result for apt")
	}
	if res.Status != SeverityOK {
		t.Errorf("expected SeverityOK when apt Detect() succeeds, got %v: %s", res.Status, res.Message)
	}
}
