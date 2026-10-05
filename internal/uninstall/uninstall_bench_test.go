package uninstall

import (
	"errors"
	"fmt"
	"testing"
)

func BenchmarkExecute(b *testing.B) {
	targets := make([]Target, 10)
	types := []TargetType{TargetBinary, TargetConfig, TargetCache}
	for i := 0; i < 10; i++ {
		targets[i] = Target{
			Type:   types[i%len(types)],
			Path:   fmt.Sprintf("/fake/path/target-%d", i),
			Exists: i%2 == 0,
		}
	}

	errMock := errors.New("permission denied")
	mockRemove := func(path string) error {
		if path == "/fake/path/target-0" || path == "/fake/path/target-6" {
			return errMock
		}
		return nil
	}
	mockRemoveAll := func(path string) error {
		return nil
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = Execute(targets, mockRemove, mockRemoveAll)
	}
}
