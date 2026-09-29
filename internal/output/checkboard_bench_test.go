package output

import (
	"fmt"
	"io"
	"testing"
)

func BenchmarkCheckBoard_RewriteRow(b *testing.B) {
	tools := make([]string, 20)
	for i := 0; i < 20; i++ {
		tools[i] = fmt.Sprintf("tool-%02d", i)
	}
	cb := NewCheckBoard(io.Discard, true, tools)
	cb.Start()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		cb.rewriteRow(5)
	}
}

func BenchmarkSanitizeBoardError(b *testing.B) {
	err := fmt.Errorf("   \n   first error line with details\n   second error line that should be ignored\n")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = sanitizeBoardError(err)
	}
}
