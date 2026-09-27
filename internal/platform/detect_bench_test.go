package platform

import "testing"

func BenchmarkNormalizeOS_Canonical(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = NormalizeOS("linux")
		_, _ = NormalizeOS("darwin")
	}
}

func BenchmarkNormalizeOS_Fallback(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = NormalizeOS("  Linux  ")
	}
}

func BenchmarkNormalizeArch_Canonical(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NormalizeArch("amd64")
		_ = NormalizeArch("arm64")
	}
}

func BenchmarkNormalizeArch_Fallback(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NormalizeArch("  AMD64  ")
	}
}
