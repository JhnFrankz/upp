package official

import (
	"context"
	"fmt"

	"github.com/JhnFrankz/upp/internal/adapters"
)

// delegatedToolAdapter implements the shared behavior for official tools
// (such as docker and gh) that detect via PATH, report version via
// `<id> --version`, and delegate Check/Update to their resolving OS manager.
type delegatedToolAdapter struct {
	id         string
	aptInfo    adapters.ToolInfo
	pacmanInfo adapters.ToolInfo
}

func (d *delegatedToolAdapter) detect() bool {
	return lookPath(d.id)
}

func (d *delegatedToolAdapter) currentVersion(ctx context.Context) (string, error) {
	return extractVersion(commandOutput(ctx, d.id, "--version")), nil
}

func (d *delegatedToolAdapter) check(ctx context.Context, self adapters.Adapter) (adapters.UpdateInfo, error) {
	if !d.detect() {
		return adapters.UpdateInfo{}, fmt.Errorf("%w: %s is not installed", adapters.ErrNotInstalled, d.id)
	}

	current, _ := d.currentVersion(ctx)
	return DelegatedCheck(ctx, self, current)
}

func (d *delegatedToolAdapter) update(ctx context.Context, self adapters.Adapter, dryRun bool) (adapters.Result, error) {
	if !d.detect() {
		return adapters.Result{Success: false}, fmt.Errorf("%w: %s is not installed", adapters.ErrNotInstalled, d.id)
	}

	return DelegatedUpdate(ctx, self, dryRun)
}

func (d *delegatedToolAdapter) info() adapters.ToolInfo {
	if defaultLinuxManager() == "pacman" {
		return d.pacmanInfo
	}
	return d.aptInfo
}
