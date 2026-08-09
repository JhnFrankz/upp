// Package main is the entry point for the upp CLI binary.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// version is set at build time via -ldflags.
var version = "dev"

func main() {
	rootCmd := &cobra.Command{
		Use:   "upp",
		Short: "Cross-platform dev environment updater",
		Long:  "upp updates your development tools across Linux, macOS, and Windows.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	rootCmd.Version = version

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
