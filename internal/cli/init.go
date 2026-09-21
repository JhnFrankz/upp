package cli

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/JhnFrankz/upp/internal/adapters/official"
	"github.com/JhnFrankz/upp/internal/config"
	"github.com/JhnFrankz/upp/internal/output"
	"github.com/JhnFrankz/upp/internal/platform"
)

// ErrInitDeniedCI is returned when `upp init --ci` would overwrite an existing
// config. The deny happens before any detection work: the exit is non-zero and
// the config file is never touched. This mirrors self-update's Confirmation
// Gate — never auto-proceed, never hang, never silently skip.
var ErrInitDeniedCI = errors.New("init denied in --ci mode")

// NewInitCommand creates the `upp init` command.
func NewInitCommand(gf *GlobalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize upp configuration",
		Long:  "Detect installed tools and generate the initial config file.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit(cmd.Context(), gf)
		},
	}
}

func runInit(ctx context.Context, gf *GlobalFlags) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	// An existing config gates the destructive overwrite this command
	// performs. --ci can never answer the overwrite prompt, and the repo's
	// doctrine is to deny rather than auto-proceed or silently skip
	// (self-update Confirmation Gate; security.ConfirmAction). Checked before
	// any detection work so the deny has no side effects.
	if config.Exists() && gf.CI {
		return fmt.Errorf("%w: rerun `upp init` interactively to confirm the overwrite", ErrInitDeniedCI)
	}

	r := output.NewRenderer(os.Stdout, gf.Quiet)
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
		if err := ctx.Err(); err != nil {
			return err
		}
		if a.Detect() {
			info := a.Info()
			detected = append(detected, info.ID)
			r.InitDetected(info.Name)
		}
	}

	if len(detected) == 0 {
		r.Warning("No supported tools detected.")
	}

	// First-run state comes from explicit file existence (D5) — never from
	// applied defaults. Existing config: confirm before overwriting. Reaching
	// here implies no --ci, which denied above.
	if config.Exists() {
		fmt.Println()
		fmt.Println("  Config already exists. Overwrite with new detection?")
		fmt.Print("  [y/N] ")
		var response string
		if _, err := fmt.Scanln(&response); err != nil || (response != "y" && response != "yes") {
			fmt.Println("  Canceled.")
			return nil
		}
	}

	// Build config with detected tools
	cfg := config.DefaultConfigWithDefaults()

	// Enable only detected tools
	for id := range cfg.Tools {
		cfg.Tools[id] = config.ToolConfig{Enabled: false}
	}
	for _, id := range detected {
		cfg.Tools[id] = config.ToolConfig{Enabled: true}
	}

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("cannot save config: %w", err)
	}

	path, _ := config.ConfigPath()
	r.InitConfigGenerated(path)
	return nil
}
