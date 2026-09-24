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
	"testing"

	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/config"
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

	t.Run("active process holds lock", func(t *testing.T) {
		deps := baseTestDeps(t)
		tmp := t.TempDir()
		lockPath := filepath.Join(tmp, "upp.lock")
		_ = os.WriteFile(lockPath, []byte("1234\n"), 0o644)

		deps.LockPath = func() (string, error) { return lockPath, nil }
		deps.ProcessAlive = func(pid int) bool { return pid == 1234 }

		results := Diagnose(context.Background(), deps)
		lockRes := findResult(results, "Process Lock", "Process Lock")
		if lockRes == nil {
			t.Fatal("missing Process Lock result")
		}
		if lockRes.Status != SeverityWarn {
			t.Errorf("expected SeverityWarn for active lock held, got %v", lockRes.Status)
		}
		if !strings.Contains(lockRes.Message, "1234") {
			t.Errorf("expected message to mention PID 1234, got %q", lockRes.Message)
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
