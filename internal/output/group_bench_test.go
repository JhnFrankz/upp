package output_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/engine"
	"github.com/JhnFrankz/upp/internal/output"
)

type benchAdapter struct {
	name string
}

func (b *benchAdapter) Name() string { return b.name }
func (b *benchAdapter) Detect() bool { return true }
func (b *benchAdapter) Check(ctx context.Context) (adapters.UpdateInfo, error) {
	return adapters.UpdateInfo{}, nil
}
func (b *benchAdapter) Update(ctx context.Context, dryRun bool) (adapters.Result, error) {
	return adapters.Result{}, nil
}
func (b *benchAdapter) Info() adapters.ToolInfo {
	return adapters.ToolInfo{ID: b.name, Name: b.name}
}

func BenchmarkPresentGroups(b *testing.B) {
	// Construct realistic 15 tools across 3 groups
	toolNames := []string{
		"brew", "gh", "node", "python", "go",
		"rust", "git", "docker", "kubectl", "terraform",
		"npm", "pnpm", "yarn", "bun", "deno",
	}

	var allAdapters []adapters.Adapter
	for _, name := range toolNames {
		allAdapters = append(allAdapters, &benchAdapter{name: name})
	}

	toolGroups := []engine.ToolGroup{
		{
			Header:   "Homebrew",
			Manager:  allAdapters[0],
			Adapters: allAdapters[0:5],
		},
		{
			Header:   "Managers",
			Manager:  allAdapters[5],
			Adapters: allAdapters[5:10],
		},
		{
			Header:   "",
			Manager:  nil,
			Adapters: allAdapters[10:15],
		},
	}

	outcomes := make([]engine.CheckOutcome, len(toolNames))
	for i, name := range toolNames {
		outcomes[i] = engine.CheckOutcome{
			ToolID:          name,
			ToolName:        name,
			Status:          engine.StatusAvailable,
			CurrentVersion:  fmt.Sprintf("1.%d.0", i),
			LatestVersion:   fmt.Sprintf("1.%d.1", i),
			UpdateAvailable: true,
		}
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = output.PresentGroups(toolGroups, outcomes)
	}
}
