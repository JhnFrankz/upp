package cli

import (
	"io"
	"testing"
)

var benchmarkTools = []string{
	"git", "go", "node", "python", "rust",
	"docker", "kubectl", "helm", "terraform", "vault",
	"gh", "ripgrep", "fd", "bat", "jq",
	"fzf", "zoxide", "starship", "neovim", "tmux",
}

func BenchmarkFilterTools_NoFilter(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = FilterTools(benchmarkTools, nil, io.Discard)
	}
}

func BenchmarkFilterTools_WithFilter(b *testing.B) {
	onlyList := []string{"git", "go"}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = FilterTools(benchmarkTools, onlyList, io.Discard)
	}
}

func BenchmarkParseFilter_Empty(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ParseFilter("")
	}
}

func BenchmarkParseFilter_Single(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ParseFilter("git")
	}
}

func BenchmarkParseFilter_Multiple(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ParseFilter("git,go,node")
	}
}
