package security

import "testing"

func BenchmarkDetectPrivileges_Benign(b *testing.B) {
	cmd := "brew upgrade && echo done"
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = DetectPrivileges(cmd)
	}
}

func BenchmarkDetectPrivileges_Elevated(b *testing.B) {
	cmd := "sudo apt install -y pkg"
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = DetectPrivileges(cmd)
	}
}
