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

func BenchmarkCustomAdapter_Info(b *testing.B) {
	ca, err := NewCustomAdapter("ripgrep", "cargo install ripgrep", "rg --version", true)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ca.Info()
	}
}
