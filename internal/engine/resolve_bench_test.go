package engine

import (
	"testing"

	"github.com/JhnFrankz/upp/internal/config"
	"github.com/JhnFrankz/upp/internal/platform"
)

func BenchmarkEngine_Resolve_Standard(b *testing.B) {
	eng := New(&config.Config{}, platform.OSLinux)
	filter := Filter{}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := eng.Resolve(filter)
		if err != nil {
			b.Fatalf("Resolve failed: %v", err)
		}
	}
}

func BenchmarkEngine_Resolve_Filtered(b *testing.B) {
	eng := New(&config.Config{}, platform.OSLinux)
	filter := Filter{Only: []string{"git", "go", "node"}}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := eng.Resolve(filter)
		if err != nil {
			b.Fatalf("Resolve failed: %v", err)
		}
	}
}
