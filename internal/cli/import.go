package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/JhnFrankz/upp/internal/config"
)

// NewImportCommand creates the `upp import` command.
func NewImportCommand(gf *GlobalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "import <file>",
		Short: "Import config from a TOML file",
		Long:  "Replace the current configuration with the contents of a TOML file.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runImport(gf, args[0])
		},
	}
}

func runImport(gf *GlobalFlags, filePath string) error {
	// Validate the file first
	cfg, err := config.ImportFromFile(filePath)
	if err != nil {
		return fmt.Errorf("invalid config file: %w", err)
	}

	// Interactive confirmation
	if !gf.CI {
		fmt.Printf("Import config from %s? This will replace your current config.\n", filePath)
		fmt.Print("[y/N] ")
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "yes" {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	// Save the imported config
	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("cannot save imported config: %w", err)
	}

	fmt.Printf("Config imported from %s\n", filePath)
	return nil
}
