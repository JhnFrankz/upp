// Package cli defines the Cobra command tree, global flags, and
// the filter logic for --only/--skip.
package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// GlobalFlags holds the flags shared by all commands.
type GlobalFlags struct {
	Quiet   bool
	Verbose bool
	CI      bool
	Only    string
	Skip    string
}

// UpdateFlags holds flags specific to the update command.
type UpdateFlags struct {
	DryRun bool
}

// BuildRoot creates the root cobra.Command with global flags.
// Running `upp` with no args shows the dashboard welcome screen, read-only.
func BuildRoot() (*cobra.Command, *GlobalFlags) {
	gf := &GlobalFlags{}

	root := &cobra.Command{
		Use:           "upp",
		Short:         "Cross-platform dev environment updater",
		Long:          "upp updates your development tools across Linux, macOS, and Windows.",
		SilenceUsage:  true,
		SilenceErrors: true,
		// The cobra completion built-in is hidden from help (D6, spec
		// command-interface).
		CompletionOptions: cobra.CompletionOptions{
			HiddenDefaultCmd: true,
		},
		// No args → show dashboard welcome screen (read-only)
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDashboard(gf, cmd.Root().Version, os.Stdout, cliDeps.dashboard)
		},
	}

	root.PersistentFlags().BoolVarP(&gf.Quiet, "quiet", "q", false, "reduce output to essential status only")
	root.PersistentFlags().BoolVarP(&gf.Verbose, "verbose", "v", false, "enable verbose diagnostic output on failure")
	root.PersistentFlags().BoolVar(&gf.CI, "ci", false, "non-interactive mode (exit non-zero on failure)")
	root.PersistentFlags().StringVar(&gf.Only, "only", "", "process only these tools (comma-separated)")
	root.PersistentFlags().StringVar(&gf.Skip, "skip", "", "skip these tools (comma-separated)")

	return root, gf
}

// AddCommands registers all subcommands on the root command.
// Commands are grouped for help output (spec command-interface):
// Commands (list/update), Maintenance (init/self-update).
func AddCommands(root *cobra.Command, gf *GlobalFlags) {
	root.AddGroup(
		&cobra.Group{ID: "commands", Title: "Commands"},
		&cobra.Group{ID: "maintenance", Title: "Maintenance"},
	)

	update := NewUpdateCommand(gf)
	update.GroupID = "commands"
	list := NewListCommand(gf)
	list.GroupID = "commands"
	init := NewInitCommand(gf)
	init.GroupID = "maintenance"
	selfUpdate := NewSelfUpdateCommand(gf)
	selfUpdate.GroupID = "maintenance"

	root.AddCommand(
		init,
		update,
		selfUpdate,
		list,
	)
}

// ParseFilter extracts the --only and --skip lists from flags.
// --only wins over --skip when both are provided.
// Tool names are lowercased for case-insensitive matching.
func ParseFilter(only, skip string) (onlyList, skipList []string) {
	if only != "" {
		onlyList = parseCommaList(only)
		return onlyList, nil // --only wins, --skip ignored
	}
	if skip != "" {
		skipList = parseCommaList(skip)
	}
	return nil, skipList
}

// FilterTools applies the --only/--skip filter to a list of tool IDs.
// It warns about unknown tool names and returns the filtered list.
func FilterTools(tools []string, onlyList, skipList []string, stderr io.Writer) []string {
	toolSet := make(map[string]bool, len(tools))
	for _, t := range tools {
		toolSet[strings.ToLower(t)] = true
	}

	if len(onlyList) > 0 {
		return filterOnly(tools, onlyList, toolSet, stderr)
	}
	if len(skipList) > 0 {
		return filterSkip(tools, skipList, toolSet, stderr)
	}
	return tools
}

func filterOnly(tools, onlyList []string, toolSet map[string]bool, stderr io.Writer) []string {
	onlySet := make(map[string]bool, len(onlyList))
	for _, name := range onlyList {
		onlySet[strings.ToLower(name)] = true
	}

	// Warn about unknown tools
	for _, name := range onlyList {
		if !toolSet[strings.ToLower(name)] {
			_, _ = fmt.Fprintf(stderr, "Warning: tool %q not found, ignored\n", name)
		}
	}

	var result []string
	for _, t := range tools {
		if onlySet[strings.ToLower(t)] {
			result = append(result, t)
		}
	}
	return result
}

func filterSkip(tools, skipList []string, toolSet map[string]bool, stderr io.Writer) []string {
	skipSet := make(map[string]bool, len(skipList))
	for _, name := range skipList {
		skipSet[strings.ToLower(name)] = true
	}

	// Warn about unknown tools
	for _, name := range skipList {
		if !toolSet[strings.ToLower(name)] {
			_, _ = fmt.Fprintf(stderr, "Warning: tool %q not found, ignored\n", name)
		}
	}

	var result []string
	for _, t := range tools {
		if !skipSet[strings.ToLower(t)] {
			result = append(result, t)
		}
	}
	return result
}

func parseCommaList(s string) []string {
	var result []string
	for _, item := range strings.Split(s, ",") {
		trimmed := strings.TrimSpace(item)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// SilenceStdout redirects os.Stdout to /dev/null for the duration of fn.
// Restores it after fn completes. Returns any error from fn.
func SilenceStdout(fn func() error) error {
	origStdout := os.Stdout
	_, w, err := os.Pipe()
	if err != nil {
		return err
	}
	os.Stdout = w
	defer func() { os.Stdout = origStdout }()
	_ = w.Close()
	return fn()
}
