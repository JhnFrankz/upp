package official

import (
	"testing"
)

func BenchmarkAdapterByName_Hit(b *testing.B) {
	tools := []string{"git", "go", "bun", "apt"}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, tool := range tools {
			_ = AdapterByName(tool)
		}
	}
}

func BenchmarkAdapterByName_Miss(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = AdapterByName("nonexistent")
	}
}

func BenchmarkIsOfficial(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = IsOfficial("go")
		_ = IsOfficial("unknown")
	}
}

func BenchmarkIsManager(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = IsManager("apt")
		_ = IsManager("bun")
	}
}

func BenchmarkPlatformsFor(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = PlatformsFor("go")
		_ = PlatformsFor("apt")
	}
}

func BenchmarkAdaptersForPlatform(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = AdaptersForPlatform("linux")
	}
}
