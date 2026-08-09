package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/adapters/official"
	"github.com/JhnFrankz/upp/internal/config"
	"github.com/JhnFrankz/upp/internal/output"
	"github.com/JhnFrankz/upp/internal/platform"
)

// NewCheckCommand creates the `upp check` command.
func NewCheckCommand(gf *GlobalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "check",
		Short: "Check for available updates (read-only)",
		Long:  "Query each enabled tool for updates without making any changes.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCheck(gf)
		},
	}
}

func runCheck(gf *GlobalFlags) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("cannot load config: %w", err)
	}

	p := platform.Detect()
	adapterList := buildAdapterList(cfg, p.OS)

	toolIDs := adapterIDs(adapterList)
	onlyList, skipList := ParseFilter(gf.Only, gf.Skip)
	filteredIDs := FilterTools(toolIDs, onlyList, skipList, os.Stderr)

	adapterMap := adapterByID(adapterList)
	var filteredAdapters []adapters.Adapter
	for _, id := range filteredIDs {
		if a, ok := adapterMap[id]; ok {
			filteredAdapters = append(filteredAdapters, a)
		}
	}

	r := output.NewRenderer(os.Stdout, gf.Quiet)

	var results []output.ToolResult
	total := len(filteredAdapters)

	for i, a := range filteredAdapters {
		info := a.Info()
		if !a.Detect() {
			results = append(results, output.ToolResult{
				Name:   info.Name,
				Status: output.StatusSkipped,
			})
			continue
		}

		if !gf.Quiet && total > 1 {
			r.Progress(i+1, total, info.Name)
		}

		updateInfo, err := a.Check()
		if err != nil {
			results = append(results, output.ToolResult{
				Name:   info.Name,
				Status: output.StatusFailed,
				Error:  err,
			})
			continue
		}

		if updateInfo.UpdateAvailable {
			results = append(results, output.ToolResult{
				Name:    info.Name,
				Status:  output.StatusAvailable,
				Version: fmt.Sprintf("%s → %s", updateInfo.CurrentVersion, updateInfo.LatestVersion),
			})
		} else {
			results = append(results, output.ToolResult{
				Name:    info.Name,
				Status:  output.StatusCurrent,
				Version: updateInfo.CurrentVersion,
			})
		}
	}

	r.CheckSummary(results)
	return nil
}

// buildAdapterList creates adapters for enabled tools from the config.
func buildAdapterList(cfg *config.Config, osName string) []adapters.Adapter {
	platformAdapters := official.AdaptersForPlatform(osName)
	var result []adapters.Adapter

	for _, a := range platformAdapters {
		info := a.Info()
		toolCfg, exists := cfg.Tools[info.ID]
		if exists && !toolCfg.Enabled {
			continue
		}
		result = append(result, a)
	}

	// Add custom adapters
	for id, custom := range cfg.Custom {
		a, err := adapters.NewCustomAdapter(id, custom.Command, custom.CheckCmd, custom.Trusted)
		if err != nil {
			continue
		}
		result = append(result, a)
	}

	return result
}

func adapterIDs(adapterList []adapters.Adapter) []string {
	var ids []string
	for _, a := range adapterList {
		ids = append(ids, a.Name())
	}
	return ids
}

func adapterByID(adapterList []adapters.Adapter) map[string]adapters.Adapter {
	m := make(map[string]adapters.Adapter, len(adapterList))
	for _, a := range adapterList {
		m[a.Name()] = a
	}
	return m
}
