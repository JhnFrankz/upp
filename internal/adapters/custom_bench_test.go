package adapters

import "testing"

func BenchmarkExtractBaseCommand(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = extractBaseCommand("  /usr/local/bin/mytool  --check --json ")
		_ = extractBaseCommand("git")
	}
}
