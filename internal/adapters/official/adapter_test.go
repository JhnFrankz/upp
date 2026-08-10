package official

import (
	"testing"

	"github.com/JhnFrankz/upp/internal/adapters"
)

// TestAdapterNames is the golden Name() table for all 12 official adapters.
// The per-adapter name/detect/info smoke tests previously scattered across
// this file are folded into the shared cross-adapter tables: Name() here,
// Detect() in detect_test.go, Info() in info_test.go, Check() in
// check_test.go and Update() in update_test.go. The version-helper tables
// (extractVersion, isVersionLike, truncate, extractGoVersion) fold into the
// *_EdgeCases supersets in adapter_update_test.go.
func TestAdapterNames(t *testing.T) {
	tests := []struct {
		name    string
		newAdpt func() adapters.Adapter
		want    string
	}{
		{"apt", func() adapters.Adapter { return &AptAdapter{} }, "apt"},
		{"brew", func() adapters.Adapter { return &BrewAdapter{} }, "brew"},
		{"npm", func() adapters.Adapter { return &NpmAdapter{} }, "npm"},
		{"pnpm", func() adapters.Adapter { return &PnpmAdapter{} }, "pnpm"},
		{"bun", func() adapters.Adapter { return &BunAdapter{} }, "bun"},
		{"gh", func() adapters.Adapter { return &GhAdapter{} }, "gh"},
		{"docker", func() adapters.Adapter { return &DockerAdapter{} }, "docker"},
		{"go", func() adapters.Adapter { return &GoAdapter{} }, "go"},
		{"opencode", func() adapters.Adapter { return &OpenCodeAdapter{} }, "opencode"},
		{"winget", func() adapters.Adapter { return &WingetAdapter{} }, "winget"},
		{"scoop", func() adapters.Adapter { return &ScoopAdapter{} }, "scoop"},
		{"nvm", func() adapters.Adapter { return &NVMAdapter{} }, "nvm"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.newAdpt().Name(); got != tt.want {
				t.Errorf("Name() = %q, want %q", got, tt.want)
			}
		})
	}
}
