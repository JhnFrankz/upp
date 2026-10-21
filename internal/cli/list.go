package cli

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/config"
	"github.com/JhnFrankz/upp/internal/engine"
	"github.com/JhnFrankz/upp/internal/output"
	"github.com/JhnFrankz/upp/internal/platform"
	"github.com/JhnFrankz/upp/internal/security"
)

// NewListCommand creates the `upp list` command.
func NewListCommand(gf *GlobalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List detected tools and their status",
		Long:  "Show all tools available on the current platform with installation status.",
		RunE: func(cmd *cobra.Command, args []string) error {
			d := cliDeps.list
			if d.out == nil {
				d.out = cmd.OutOrStdout()
			}
			if d.errOut == nil {
				d.errOut = cmd.ErrOrStderr()
			}
			return runList(cmd.Context(), gf, d)
		},
	}
}

// listDeps carries the injectable seam for runList, mirroring
// updateDeps. The zero value uses the production adapter list
// builder.
type listDeps struct {
	buildAdapterList func(cfg *config.Config, osName string) []adapters.Adapter
	out              io.Writer
	errOut           io.Writer
}

func runList(ctx context.Context, gf *GlobalFlags, deps listDeps) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	out := deps.out
	if out == nil {
		out = os.Stdout
	}
	errOut := deps.errOut
	if errOut == nil {
		errOut = os.Stderr
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("cannot load config: %w", err)
	}

	p, err := platform.Detect()
	if err != nil {
		return fmt.Errorf("cannot detect platform: %w", err)
	}

	var allAdapters []adapters.Adapter
	if deps.buildAdapterList != nil {
		allAdapters = deps.buildAdapterList(cfg, p.OS)
	} else {
		var err error
		allAdapters, err = adapters.ResolveAdapters(cfg, p.OS)
		if err != nil {
			return fmt.Errorf("cannot resolve tools: %w", err)
		}
	}
	eng := engine.New(cfg, p.OS, engine.WithAdapters(allAdapters))

	only := ParseFilter(gf.Only)
	if len(only) > 0 {
		FilterTools(adapterIDs(allAdapters), only, errOut)
	}

	var adapterList []adapters.Adapter
	if len(only) == 0 {
		adapterList = allAdapters
	} else {
		adapterList = engine.FilterAdapters(allAdapters, only)
	}

	r := output.NewRenderer(out, gf.Quiet)

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

	toolGroups := engine.GroupByOwner(adapterList, p.OS, allAdapters)
	if len(toolGroups) == 0 {
		r.NoToolsConfigured()
		return nil
	}

	checkAdapters := engine.PrepareCheckAdapters(adapterList, p.OS, allAdapters)
	for i, a := range checkAdapters {
		if security.CheckNeedsConsent(a.Info().CheckCommand) {
			checkAdapters[i] = unconsentedCheckAdapter{Adapter: a}
		}
	}

	outcomes, err := eng.Check(ctx, checkAdapters, nil)
	if err != nil {
		return fmt.Errorf("cannot check tools: %w", err)
	}

	groups := presentGroups(toolGroups, outcomes)
	if len(groups) == 0 {
		r.NoToolsConfigured()
		return nil
	}

	r.ListTools(groups)
	return nil
}
