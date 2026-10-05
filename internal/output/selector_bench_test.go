package output

import (
	"fmt"
	"io"
	"strings"
	"testing"
)

func BenchmarkCheckboxSelector_Run(b *testing.B) {
	opts := make([]SelectOption, 10)
	for i := 0; i < 10; i++ {
		opts[i] = SelectOption{
			ID:      fmt.Sprintf("tool-%d", i),
			Label:   fmt.Sprintf("Tool %d", i),
			Version: "1.0.0 → 2.0.0",
		}
	}
	input := strings.Repeat("j k \r", 10)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		sel := NewCheckboxSelector(io.Discard, strings.NewReader(input), opts).WithColor(true)
		_, _ = sel.Run()
	}
}
