package cli

import (
	"io"
	"testing"

	"github.com/JhnFrankz/upp/internal/config"
	"github.com/JhnFrankz/upp/internal/platform"
)

func BenchmarkRunDashboard_WithConfig(b *testing.B) {
	gf := &GlobalFlags{}
	deps := dashboardDeps{
		configExists: func() bool { return true },
		loadConfig: func() (*config.Config, error) {
			return &config.Config{
				Tools: map[string]config.ToolConfig{
					"git": {Enabled: true},
					"go":  {Enabled: true},
				},
			}, nil
		},
		detectPlatform: func() (platform.Platform, error) {
			return platform.Platform{OS: "linux", Arch: "x86_64"}, nil
		},
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = runDashboard(gf, "v1.0.0", io.Discard, deps)
	}
}

func BenchmarkRunDashboard_NoConfig(b *testing.B) {
	gf := &GlobalFlags{}
	deps := dashboardDeps{
		configExists: func() bool { return false },
		detectPlatform: func() (platform.Platform, error) {
			return platform.Platform{OS: "linux", Arch: "x86_64"}, nil
		},
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = runDashboard(gf, "v1.0.0", io.Discard, deps)
	}
}
