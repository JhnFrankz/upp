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
