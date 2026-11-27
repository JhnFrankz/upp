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

	sess, err := resolveSessionAdapters(gf, deps.buildAdapterList, errOut)
	if err != nil {
		return err
	}
	allAdapters := sess.allAdapters
	adapterList := sess.filteredAdapters
	only := sess.onlyList
	eng := sess.eng

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

	toolGroups := engine.GroupByOwner(adapterList, sess.osName, allAdapters)
	if len(toolGroups) == 0 {
		r.NoToolsConfigured()
		return nil
	}

	checkAdapters := engine.PrepareCheckAdapters(adapterList, sess.osName, allAdapters)
	for i, a := range checkAdapters {
		inner := a
		if u, ok := a.(interface{ Unwrap() adapters.Adapter }); ok {
			inner = u.Unwrap()
		}
		if ca, ok := inner.(*adapters.CustomAdapter); ok && ca.ManagerChecker() != nil {
			continue
		}
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
