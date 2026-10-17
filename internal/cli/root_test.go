package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/JhnFrankz/upp/internal/config"
	"github.com/JhnFrankz/upp/internal/platform"
)

func TestRunDashboard_NoConfig(t *testing.T) {
	var buf bytes.Buffer
	gf := &GlobalFlags{}
	deps := dashboardDeps{
		configExists: func() bool { return false },
		detectPlatform: func() (platform.Platform, error) {
			return platform.Platform{OS: "linux", Arch: "amd64"}, nil
		},
	}

	err := runDashboard(gf, "v0.2.0", &buf, deps)
	if err != nil {
		t.Fatalf("runDashboard failed: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "No configuration found") {
		t.Errorf("expected missing config message, got: %q", out)
	}
	if !strings.Contains(out, "upp init") {
		t.Errorf("expected prompt to run upp init, got: %q", out)
	}
}

func TestRunDashboard_WithConfig(t *testing.T) {
	var buf bytes.Buffer
	gf := &GlobalFlags{}
	deps := dashboardDeps{
		configExists: func() bool { return true },
		loadConfig: func() (*config.Config, error) {
			cfg := config.DefaultConfig()
			cfg.Tools["apt"] = config.ToolConfig{Enabled: true}
			cfg.Tools["npm"] = config.ToolConfig{Enabled: false}
			cfg.Custom["custom1"] = config.CustomTool{Command: "echo update"}
			return cfg, nil
		},
		detectPlatform: func() (platform.Platform, error) {
			return platform.Platform{OS: "linux", Arch: "amd64"}, nil
		},
	}

	err := runDashboard(gf, "v0.2.0", &buf, deps)
	if err != nil {
		t.Fatalf("runDashboard failed: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "upp v0.2.0 (linux/amd64)") {
		t.Errorf("expected version and platform banner, got:\n%s", out)
	}
	if !strings.Contains(out, "Tools:") {
		t.Errorf("expected tools count line, got:\n%s", out)
	}
	if !strings.Contains(out, "upp update -n") || !strings.Contains(out, "upp update") {
		t.Errorf("expected command guidance, got:\n%s", out)
	}
	if strings.Contains(out, "upp check") {
		t.Errorf("dashboard must not reference the removed 'upp check', got:\n%s", out)
	}
}

func TestRunDashboard_QuietSuppresses(t *testing.T) {
	var buf bytes.Buffer
	gf := &GlobalFlags{Quiet: true}
	deps := dashboardDeps{
		configExists: func() bool { return true },
		loadConfig: func() (*config.Config, error) {
			return config.DefaultConfig(), nil
		},
		detectPlatform: func() (platform.Platform, error) {
			return platform.Platform{OS: "linux", Arch: "amd64"}, nil
		},
	}

	err := runDashboard(gf, "v0.2.0", &buf, deps)
	if err != nil {
		t.Fatalf("runDashboard failed: %v", err)
	}

	if buf.Len() > 0 {
		t.Errorf("quiet mode should suppress dashboard output, got: %q", buf.String())
	}
}

func TestRunDashboard_CustomToolDisabledAndNoDoubleCount(t *testing.T) {
	var buf bytes.Buffer
	gf := &GlobalFlags{}
	deps := dashboardDeps{
		configExists: func() bool { return true },
		loadConfig: func() (*config.Config, error) {
			cfg := config.DefaultConfig()
			// Explicitly disable all official linux tools except apt
			for _, name := range []string{"brew", "pacman", "nvm", "npm", "pnpm", "bun", "uv", "gh", "docker", "go", "opencode"} {
				cfg.Tools[name] = config.ToolConfig{Enabled: false}
			}
			cfg.Tools["apt"] = config.ToolConfig{Enabled: true}
			// Custom tool 1: explicitly enabled in cfg.Tools (must not double-count)
			cfg.Custom["custom-enabled"] = config.CustomTool{Command: "echo 1"}
			cfg.Tools["custom-enabled"] = config.ToolConfig{Enabled: true}
			// Custom tool 2: explicitly disabled in cfg.Tools (must not count as enabled)
			cfg.Custom["custom-disabled"] = config.CustomTool{Command: "echo 2"}
			cfg.Tools["custom-disabled"] = config.ToolConfig{Enabled: false}
			// Custom tool 3: no entry in cfg.Tools (enabled by default)
			cfg.Custom["custom-default"] = config.CustomTool{Command: "echo 3"}
			return cfg, nil
		},
		detectPlatform: func() (platform.Platform, error) {
			return platform.Platform{OS: "linux", Arch: "amd64"}, nil
		},
	}

	if err := runDashboard(gf, "v0.2.0", &buf, deps); err != nil {
		t.Fatalf("runDashboard failed: %v", err)
	}

	out := buf.String()
	// 12 official linux tools + 3 custom tools = 15 available
	// 1 enabled official (apt) + 2 enabled custom (custom-enabled, custom-default) = 3 enabled
	if !strings.Contains(out, "Tools: 3 enabled (15 configured for platform)") {
		t.Errorf("expected 'Tools: 3 enabled (15 configured for platform)', got:\n%s", out)
	}
}
