package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/JhnFrankz/upp/internal/adapters/official"
	"github.com/JhnFrankz/upp/internal/config"
	"github.com/JhnFrankz/upp/internal/output"
	"github.com/JhnFrankz/upp/internal/platform"
)

// NewInitCommand creates the `upp init` command.
func NewInitCommand(gf *GlobalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize upp configuration",
		Long:  "Detect installed tools and generate the initial config file.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit(gf)
		},
	}
}

func runInit(gf *GlobalFlags) error {
	cfg, _ := config.Load()
	lang := "en"
	if cfg != nil {
		lang = cfg.Settings.Language
	}
	r := output.NewRendererWithLang(os.Stdout, gf.Quiet, lang)
	r.InitHeader()

	// Detect platform
	p, err := platform.Detect()
	if err != nil {
		return fmt.Errorf("cannot detect platform: %w", err)
	}

	// Get platform-specific adapters
	platformAdapters := official.AdaptersForPlatform(p.OS)

	// Detect which tools are installed
	var detected []string
	for _, a := range platformAdapters {
		if a.Detect() {
			info := a.Info()
			detected = append(detected, info.ID)
			r.InitDetected(info.Name)
		}
	}

	if len(detected) == 0 {
		r.Warning("No supported tools detected.")
	}

	// Check if config already exists
	existingCfg, _ := config.Load()
	if existingCfg != nil && existingCfg.Version > 0 {
		if !gf.CI {
			r.Warning("Config already exists. Use --ci to overwrite.")
			return nil
		}
	}

	// Build config with detected tools
	cfg = config.DefaultConfigWithDefaults()

	// Enable only detected tools
	for id := range cfg.Tools {
		cfg.Tools[id] = config.ToolConfig{Enabled: false}
	}
	for _, id := range detected {
		cfg.Tools[id] = config.ToolConfig{Enabled: true}
	}

	// In CI mode, skip confirmation
	if gf.CI {
		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("cannot save config: %w", err)
		}
		path, _ := config.ConfigPath()
		r.InitConfigGenerated(path)
		return nil
	}

	// Interactive: if existing config, warn
	if existingCfg != nil && existingCfg.Version > 0 {
		fmt.Println()
		fmt.Println("  Config already exists. Overwrite with new detection?")
		fmt.Print("  [y/N] ")
		var response string
		if _, err := fmt.Scanln(&response); err != nil || (response != "y" && response != "yes") {
			fmt.Println("  Cancelled.")
			return nil
		}
	}

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("cannot save config: %w", err)
	}

	path, _ := config.ConfigPath()
	r.InitConfigGenerated(path)
	return nil
}
