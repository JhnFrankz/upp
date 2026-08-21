package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/config"
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
	if deps.buildAdapterList == nil {
		deps.buildAdapterList = buildAdapterList
	}
	adapterList := deps.buildAdapterList(cfg, p.OS)

	// Apply --only/--skip so table rows round-trip with the filter names.
	only, skip := ParseFilter(gf.Only, gf.Skip)
	adapterMap := adapterByID(adapterList)
	filtered := make([]adapters.Adapter, 0, len(adapterList))
	for _, id := range FilterTools(adapterIDs(adapterList), only, skip, os.Stderr) {
		filtered = append(filtered, adapterMap[id])
	}
	adapterList = filtered

	r := output.NewRenderer(os.Stdout, gf.Quiet)

	var entries []output.ListEntry
	for _, a := range adapterList {
		info := a.Info()
		installed := a.Detect()

		status := output.StatusSkipped
		version := ""

		if installed {
			status = output.StatusCurrent
			// Try to get version
			updateInfo, err := a.Check()
			if err == nil {
				version = updateInfo.CurrentVersion
			}
		}

		entries = append(entries, output.ListEntry{
			ID:      info.ID,
			Name:    info.Name,
			Status:  status,
			Version: version,
		})
	}

	if len(entries) == 0 {
		fmt.Println("No tools configured.")
		return nil
	}

	r.ListTools(entries)
	return nil
}
