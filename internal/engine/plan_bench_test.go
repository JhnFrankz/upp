package engine

import (
	"fmt"
	"testing"

	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/config"
	"github.com/JhnFrankz/upp/internal/platform"
)

func BenchmarkEngine_Plan(b *testing.B) {
	sizes := []int{10, 50, 100}
	statuses := []CheckStatus{StatusAvailable, StatusCurrent, StatusFailed, StatusSkipped}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("%d_tools", size), func(b *testing.B) {
			adapterList := make([]adapters.Adapter, size)
			outcomes := make([]CheckOutcome, size)

			for i := 0; i < size; i++ {
				id := fmt.Sprintf("tool-%d", i)
				name := fmt.Sprintf("Tool %d", i)
				st := statuses[i%len(statuses)]

				adapterList[i] = &mockPlanAdapter{
					info: adapters.ToolInfo{
						ID:           id,
						Name:         name,
						UpdatePolicy: adapters.PolicyGated,
						Command:      id + " update",
					},
				}

				outcomes[i] = CheckOutcome{
					ToolID:          id,
					ToolName:        name,
					Status:          st,
					CurrentVersion:  "1.0.0",
					LatestVersion:   "1.1.0",
					UpdateAvailable: (st == StatusAvailable),
				}
			}

			eng := New(&config.Config{}, platform.OSLinux, WithAdapters(adapterList))
			filter := Filter{}

			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, err := eng.Plan(outcomes, filter)
				if err != nil {
					b.Fatalf("Plan failed: %v", err)
				}
			}
		})
	}
}
