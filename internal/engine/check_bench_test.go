package engine

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/config"
	"github.com/JhnFrankz/upp/internal/security"
)

type benchMockAdapter struct {
	id       string
	name     string
	workload time.Duration
}

func (a *benchMockAdapter) Name() string {
	return a.name
}

func (a *benchMockAdapter) Detect() bool {
	return true
}

func (a *benchMockAdapter) Check(ctx context.Context) (adapters.UpdateInfo, error) {
	if a.workload > 0 {
		time.Sleep(a.workload)
	}
	return adapters.UpdateInfo{
		CurrentVersion:  "1.0.0",
		LatestVersion:   "1.1.0",
		UpdateAvailable: true,
	}, nil
}

func (a *benchMockAdapter) Update(ctx context.Context, dryRun bool) (adapters.Result, error) {
	return adapters.Result{Success: true}, nil
}

func (a *benchMockAdapter) Info() adapters.ToolInfo {
	return adapters.ToolInfo{
		ID:           a.id,
		Name:         a.name,
		Trust:        security.TrustOfficial,
		UpdatePolicy: adapters.PolicyGated,
	}
}

func createBenchAdapters(n int, workload time.Duration) []adapters.Adapter {
	adapterList := make([]adapters.Adapter, n)
	for i := 0; i < n; i++ {
		adapterList[i] = &benchMockAdapter{
			id:       fmt.Sprintf("tool-%03d", i),
			name:     fmt.Sprintf("Tool %03d", i),
			workload: workload,
		}
	}
	return adapterList
}

// BenchmarkEngine_Check_Scaling benchmarks eng.Check with varying tool counts
// and worker pool sizes simulating realistic lightweight check workloads.
func BenchmarkEngine_Check_Scaling(b *testing.B) {
	cfg := &config.Config{}
	ctx := context.Background()

	toolCounts := []int{10, 25, 50, 100}
	workerCounts := []int{4, 8, 16}

	for _, tools := range toolCounts {
		mockAdapters := createBenchAdapters(tools, 50*time.Microsecond)
		for _, workers := range workerCounts {
			name := fmt.Sprintf("Tools-%d/Workers-%d", tools, workers)
			b.Run(name, func(b *testing.B) {
				eng := New(cfg, "linux", WithWorkerCount(workers))
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_, err := eng.Check(ctx, mockAdapters, nil)
					if err != nil {
						b.Fatalf("unexpected error: %v", err)
					}
				}
			})
		}
	}
}

// BenchmarkEngine_Check_ProgressContention measures the overhead of progressMu serialization
// when 50 tools complete nearly simultaneously, comparing nil onProgress vs synchronized onProgress.
func BenchmarkEngine_Check_ProgressContention(b *testing.B) {
	cfg := &config.Config{}
	ctx := context.Background()
	const totalTools = 50

	mockAdapters := createBenchAdapters(totalTools, 10*time.Microsecond)
	eng := New(cfg, "linux", WithWorkerCount(16))

	b.Run("NoProgress", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := eng.Check(ctx, mockAdapters, nil)
			if err != nil {
				b.Fatalf("unexpected error: %v", err)
			}
		}
	})

	b.Run("WithProgress", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			completedCount := 0
			completedIndices := make([]int, 0, totalTools)
			_, err := eng.Check(ctx, mockAdapters, func(p CheckProgress) {
				completedCount++
				completedIndices = append(completedIndices, p.Index)
			})
			if err != nil {
				b.Fatalf("unexpected error: %v", err)
			}
			if completedCount != totalTools {
				b.Fatalf("completedCount = %d, want %d", completedCount, totalTools)
			}
		}
	})
}

// BenchmarkEngine_Check_Allocations measures and asserts heap allocations per check cycle for 20 tools.
func BenchmarkEngine_Check_Allocations(b *testing.B) {
	cfg := &config.Config{}
	ctx := context.Background()
	const totalTools = 20

	mockAdapters := createBenchAdapters(totalTools, 0)
	eng := New(cfg, "linux", WithWorkerCount(8))

	// Assert allocations per cycle are bounded and measurable
	allocs := testing.AllocsPerRun(10, func() {
		outcomes, err := eng.Check(ctx, mockAdapters, nil)
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
		if len(outcomes) != totalTools {
			b.Fatalf("outcomes len = %d, want %d", len(outcomes), totalTools)
		}
	})
	if allocs == 0 {
		b.Fatalf("expected non-zero allocations per cycle, got 0")
	}
	if allocs > 200 {
		b.Fatalf("excessive heap allocations per check cycle: got %.1f, want <= 200", allocs)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		outcomes, err := eng.Check(ctx, mockAdapters, nil)
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
		if len(outcomes) != totalTools {
			b.Fatalf("outcomes len = %d, want %d", len(outcomes), totalTools)
		}
	}
}
