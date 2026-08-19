package cli

import (
	"fmt"
	"io"

	"github.com/JhnFrankz/upp/internal/config"
	"github.com/JhnFrankz/upp/internal/output"
	"github.com/JhnFrankz/upp/internal/platform"
)

type dashboardDeps struct {
	configExists   func() bool
	loadConfig     func() (*config.Config, error)
	detectPlatform func() (platform.Platform, error)
}

func runDashboard(gf *GlobalFlags, version string, w io.Writer, deps dashboardDeps) error {
	if deps.configExists == nil {
		deps.configExists = config.Exists
	}
	if deps.loadConfig == nil {
		deps.loadConfig = config.Load
	}
	if deps.detectPlatform == nil {
		deps.detectPlatform = platform.Detect
	}

	p, err := deps.detectPlatform()
	if err != nil {
		p = platform.Platform{OS: "unknown", Arch: "unknown"}
	}

	r := output.NewRenderer(w, gf.Quiet)

	if !deps.configExists() {
		r.DashboardNoConfig(version, fmt.Sprintf("%s/%s", p.OS, p.Arch))
		return nil
	}

	cfg, err := deps.loadConfig()
	if err != nil {
		return fmt.Errorf("cannot load config: %w", err)
	}

	// Count enabled tools vs available platform catalog tools + custom tools
	platformCatalog := platform.CatalogFor(p.OS)
	totalAvailable := len(platformCatalog) + len(cfg.Custom)

	enabledCount := 0
	for _, tool := range platformCatalog {
		if tCfg, ok := cfg.Tools[tool.ID]; !ok || tCfg.Enabled {
			enabledCount++
		}
	}
	for range cfg.Custom {
		enabledCount++
	}

	r.Dashboard(output.DashboardData{
		Version:        version,
		Platform:       fmt.Sprintf("%s/%s", p.OS, p.Arch),
		EnabledTools:   enabledCount,
		AvailableTools: totalAvailable,
	})

	return nil
}
