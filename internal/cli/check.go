package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/adapters/official"
	"github.com/JhnFrankz/upp/internal/config"
	"github.com/JhnFrankz/upp/internal/output"
	"github.com/JhnFrankz/upp/internal/platform"
	"github.com/JhnFrankz/upp/internal/selfupdate"
)

// NewCheckCommand creates the `upp check` command.
func NewCheckCommand(gf *GlobalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "check",
		Short: "Check for available updates (read-only)",
		Long:  "Query each enabled tool for updates without making any changes.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCheck(gf, cmd.Root().Version, cliDeps.check)
		},
	}
}

// selfUpdateCacheFile is the detection-cache file name inside the config
// directory (spec config-system: "{config-dir}/self-update-cache.json").
const selfUpdateCacheFile = "self-update-cache.json"

// checkDeps carries the injectable seam for the check hint (design D9),
// mirroring selfUpdateDeps. The zero value uses the production client
// factory: a selfupdate.Client on the API base with the detection cache
// at {config-dir}/self-update-cache.json.
type checkDeps struct {
	clientFactory func(cachePath string) *selfupdate.Client
	// buildAdapterList mirrors updateDeps (update.go). The zero value
	// uses the production builder.
	buildAdapterList func(cfg *config.Config, osName string) []adapters.Adapter
}

func runCheck(gf *GlobalFlags, version string, deps checkDeps) error {
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
			r.Progress("Checking", i+1, total, info.Name)
		}

		updateInfo, err := a.Check()
		if err != nil {
			results = append(results, output.ToolResult{
				Name:   info.Name,
				Status: output.StatusFailed,
				Error:  timeoutErr(info.Name, "check", err),
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
	maybeShowSelfUpdateHint(gf, r, cfg, version, deps)
	return nil
}

// maybeShowSelfUpdateHint appends the opt-in update hint after the check
// summary (design D9, spec ux-patterns): only when settings.
// check_self_update is enabled, output is not quiet, the current build
// is a release (not dev/dirty), and the cached or freshly fetched latest
// release is newer than the current version. ANY failure — config dir,
// network, parse — is silent and the exit code is unchanged. The client
// is never constructed when the setting is off or output is quiet:
// default config performs ZERO self-update network calls (spec
// config-system, test-enforced).
func maybeShowSelfUpdateHint(gf *GlobalFlags, r *output.Renderer, cfg *config.Config, version string, deps checkDeps) {
	if deps.clientFactory == nil {
		deps.clientFactory = func(cachePath string) *selfupdate.Client {
			return selfupdate.NewClient(selfUpdateAPIBase, cachePath)
		}
	}

	if !cfg.Settings.CheckSelfUpdate || gf.Quiet {
		return
	}

	current, err := selfupdate.Parse(version)
	if err != nil || current.Dev || current.Dirty {
		return // unparseable/dev/dirty: never claim an update, no network
	}

	configDir, err := config.ConfigDir()
	if err != nil {
		return
	}

	latest, ok := deps.clientFactory(filepath.Join(configDir, selfUpdateCacheFile)).LatestCached()
	if !ok {
		return // offline or any failure: silent, exit unchanged
	}

	latestV, err := selfupdate.Parse(latest)
	if err != nil {
		return // unparseable upstream tag: silent, no hint
	}
	if current.Compare(latestV) >= 0 {
		return // up to date (or newer locally): no hint
	}

	r.SelfUpdateHint(formatVersion(current), latest)
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
