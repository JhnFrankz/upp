package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/JhnFrankz/upp/internal/config"
)

// NewExportCommand creates the `upp export` command.
func NewExportCommand(gf *GlobalFlags) *cobra.Command {
	ef := &ExportFlags{}

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export config to stdout or file",
		Long:  "Output the current configuration as TOML.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runExport(gf, ef)
		},
	}

	cmd.Flags().StringVarP(&ef.Output, "output", "o", "", "write config to file instead of stdout")

	return cmd
}

func runExport(gf *GlobalFlags, ef *ExportFlags) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("cannot load config: %w", err)
	}

	if ef.Output != "" {
		if err := config.ExportToFile(cfg, ef.Output); err != nil {
			return fmt.Errorf("cannot export config: %w", err)
		}
		fmt.Printf("Config exported to %s\n", ef.Output)
		return nil
	}

	return config.Export(cfg)
}
