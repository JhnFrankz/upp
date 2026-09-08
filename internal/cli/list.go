package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/config"
	"github.com/JhnFrankz/upp/internal/engine"
	"github.com/JhnFrankz/upp/internal/output"
	"github.com/JhnFrankz/upp/internal/platform"
)

// NewListCommand creates the `upp list` command.
func NewListCommand(gf *GlobalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List detected tools and their status",
		Long:  "Show all tools available on the current platform with installation status.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(gf, cliDeps.list)
		},
	}
}

// listDeps carries the injectable seam for runList, mirroring
// updateDeps. The zero value uses the production adapter list
// builder.
type listDeps struct {
	buildAdapterList func(cfg *config.Config, osName string) []adapters.Adapter
}

func runList(gf *GlobalFlags, deps listDeps) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("cannot load config: %w", err)
	}

	p, err := platform.Detect()
	if err != nil {
		return fmt.Errorf("cannot detect platform: %w", err)
	}

	var opts []engine.Option
	if deps.buildAdapterList != nil {
		opts = append(opts, engine.WithAdapters(deps.buildAdapterList(cfg, p.OS)))
	}
	eng := engine.New(cfg, p.OS, opts...)

	only := ParseFilter(gf.Only)
	adapterList, err := eng.Resolve(engine.Filter{Only: only})
	if err != nil {
		return fmt.Errorf("cannot resolve tools: %w", err)
	}

	r := output.NewRenderer(os.Stdout, gf.Quiet)

	// Build the grouped rows (manager headers first, then their owned tools,
	// then standalone tools) from the filtered adapter set. Grouping is
	// display-only: --only already filtered per-tool ID above, so the
	// rendered rows round-trip with the filter names (design: display-only).
	if len(adapterList) == 0 {
		if len(only) > 0 {
			// --only named tools but none matched: say it's the filter,
			// not an empty config (the two exits are not the same state).
			r.NoToolsMatchFilter(gf.Only)
			return nil
		}
		r.NoToolsConfigured()
		return nil
	}

	groups := output.GroupByOwner(adapterList, p.OS)
	if len(groups) == 0 {
		r.NoToolsConfigured()
		return nil
	}

	r.ListTools(groups)
	return nil
}
