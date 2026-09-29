package engine

import (
	"testing"

	"github.com/JhnFrankz/upp/internal/adapters/official"
	"github.com/JhnFrankz/upp/internal/platform"
)

func BenchmarkGroupByOwner(b *testing.B) {
	adaps := official.AdaptersForPlatform(platform.OSLinux)
	osName := platform.OSLinux

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = GroupByOwner(adaps, osName)
	}
}
