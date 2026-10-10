package config

import (
	"io"
	"testing"
)

func BenchmarkValidate_Default(b *testing.B) {
	cfg := DefaultConfig()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = Validate(cfg, io.Discard)
	}
}

func BenchmarkValidate_WithTools(b *testing.B) {
	cfg := DefaultConfig()
	toolNames := []string{
		"apt", "brew", "pacman", "winget", "scoop",
		"nvm", "npm", "pnpm", "bun", "uv",
		"gh", "docker", "go", "opencode", "my-custom",
	}
	for _, name := range toolNames {
		cfg.Tools[name] = ToolConfig{Enabled: true}
	}
	cfg.Custom["my-custom"] = CustomTool{
		Command: "custom-upgrade",
		Trusted: true,
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = Validate(cfg, io.Discard)
	}
}

func BenchmarkApplyDefaults(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		cfg := DefaultConfig()
		ApplyDefaults(cfg)
	}
}

func BenchmarkConfigPath(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = ConfigPath()
	}
}
