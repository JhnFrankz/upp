package official

import (
	"github.com/JhnFrankz/upp/internal/adapters"
)

// UvAdapter manages Astral's uv package and tool manager on all platforms.
type UvAdapter struct{}

var _ adapters.Adapter = (*UvAdapter)(nil)

func (a *UvAdapter) Name() string { return "uv" }

func (a *UvAdapter) Detect() bool {
	return lookPath("uv")
}

func (a *UvAdapter) Info() adapters.ToolInfo {
	return adapters.ToolInfo{
		ID:           "uv",
		Name:         "uv",
		Platforms:    []string{"linux", "macos", "windows"},
		Trust:        adapters.TrustOfficial,
		UpdatePolicy: adapters.PolicyGated,
		Kind:         adapters.KindTool,
	}
}

func (a *UvAdapter) Check() (adapters.UpdateInfo, error) {
	return adapters.UpdateInfo{}, nil
}

func (a *UvAdapter) Update(dryRun bool) (adapters.Result, error) {
	return adapters.Result{}, nil
}
