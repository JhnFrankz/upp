package archive_test

import (
	"path/filepath"
	"testing"

	"github.com/JhnFrankz/upp/internal/archive"
)

func BenchmarkSafeTargetPath_Valid(b *testing.B) {
	tempDir := filepath.Clean(b.TempDir())
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = archive.SafeTargetPath(tempDir, "bin/go")
	}
}

func BenchmarkSafeTargetPath_Escapes(b *testing.B) {
	tempDir := filepath.Clean(b.TempDir())
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = archive.SafeTargetPath(tempDir, "../evil")
	}
}

func BenchmarkValidatePath_Valid(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = archive.ValidatePath("pkg/tool/sub/binary")
	}
}
