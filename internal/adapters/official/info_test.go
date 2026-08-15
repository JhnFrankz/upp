package official

import (
	"reflect"
	"testing"

	"github.com/JhnFrankz/upp/internal/adapters"
)

// infoCase is one table row for Info(): golden static metadata per adapter.
type infoCase struct {
	name    string
	newAdpt func() adapters.Adapter
	want    adapters.ToolInfo
}

// TestInfo verifies Info() for all 12 official adapters: golden
// ID/Name/Platforms/Trust/UpdatePolicy, plus Name() consistency with the
// adapter ID. UpdatePolicy is declared explicitly at every Info() site
// (design D6, spec Update Gating) — the golden value pins the declared
// policy per adapter. No exec seam involved — pure static metadata.
func TestInfo(t *testing.T) {
	tests := []infoCase{
		{"apt", func() adapters.Adapter { return &AptAdapter{} }, adapters.ToolInfo{ID: "apt", Name: "APT Package Manager", Platforms: []string{"linux"}, Trust: adapters.TrustOfficial, UpdatePolicy: adapters.PolicyGated}},
		{"brew", func() adapters.Adapter { return &BrewAdapter{} }, adapters.ToolInfo{ID: "brew", Name: "Homebrew", Platforms: []string{"linux", "macos"}, Trust: adapters.TrustOfficial, UpdatePolicy: adapters.PolicyAlwaysUpdate}},
		{"npm", func() adapters.Adapter { return &NpmAdapter{} }, adapters.ToolInfo{ID: "npm", Name: "npm", Platforms: []string{"linux", "macos", "windows"}, Trust: adapters.TrustOfficial, UpdatePolicy: adapters.PolicyGated}},
		{"pnpm", func() adapters.Adapter { return &PnpmAdapter{} }, adapters.ToolInfo{ID: "pnpm", Name: "pnpm", Platforms: []string{"linux", "macos", "windows"}, Trust: adapters.TrustOfficial, UpdatePolicy: adapters.PolicyGated}},
		{"bun", func() adapters.Adapter { return &BunAdapter{} }, adapters.ToolInfo{ID: "bun", Name: "Bun", Platforms: []string{"linux", "macos", "windows"}, Trust: adapters.TrustOfficial, UpdatePolicy: adapters.PolicyAlwaysUpdate}},
		{"gh", func() adapters.Adapter { return &GhAdapter{} }, adapters.ToolInfo{ID: "gh", Name: "GitHub CLI", Platforms: []string{"linux", "macos", "windows"}, Trust: adapters.TrustOfficial, UpdatePolicy: adapters.PolicyAlwaysUpdate}},
		{"docker", func() adapters.Adapter { return &DockerAdapter{} }, adapters.ToolInfo{ID: "docker", Name: "Docker", Platforms: []string{"linux", "macos", "windows"}, Trust: adapters.TrustOfficial, UpdatePolicy: adapters.PolicyAlwaysUpdate}},
		{"go", func() adapters.Adapter { return &GoAdapter{} }, adapters.ToolInfo{ID: "go", Name: "Go", Platforms: []string{"linux", "macos", "windows"}, Trust: adapters.TrustOfficial, UpdatePolicy: adapters.PolicyAlwaysUpdate}},
		{"opencode", func() adapters.Adapter { return &OpenCodeAdapter{} }, adapters.ToolInfo{ID: "opencode", Name: "OpenCode", Platforms: []string{"linux", "macos", "windows"}, Trust: adapters.TrustOfficial, UpdatePolicy: adapters.PolicyAlwaysUpdate}},
		{"winget", func() adapters.Adapter { return &WingetAdapter{} }, adapters.ToolInfo{ID: "winget", Name: "Windows Package Manager", Platforms: []string{"windows"}, Trust: adapters.TrustOfficial, UpdatePolicy: adapters.PolicyAlwaysUpdate}},
		{"scoop", func() adapters.Adapter { return &ScoopAdapter{} }, adapters.ToolInfo{ID: "scoop", Name: "Scoop", Platforms: []string{"windows"}, Trust: adapters.TrustOfficial, UpdatePolicy: adapters.PolicyAlwaysUpdate}},
		{"nvm", func() adapters.Adapter { return &NVMAdapter{} }, adapters.ToolInfo{ID: "nvm", Name: "Node Version Manager", Platforms: []string{"linux", "macos", "windows"}, Trust: adapters.TrustOfficial, UpdatePolicy: adapters.PolicyGated}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := tt.newAdpt()
			info := a.Info()
			if !reflect.DeepEqual(info, tt.want) {
				t.Errorf("Info() = %+v, want %+v", info, tt.want)
			}
			if name := a.Name(); name != tt.want.ID {
				t.Errorf("Name() = %q, want %q", name, tt.want.ID)
			}
		})
	}
}
