package lock

import (
	"os"
	"path/filepath"
	"testing"
)

func TestShouldRetryUnlink_ENOENTAndMismatch(t *testing.T) {
	dir := t.TempDir()
	p1 := filepath.Join(dir, "a.lock")
	p2 := filepath.Join(dir, "b.lock")
	if err := os.WriteFile(p1, []byte("1"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p2, []byte("2"), 0o600); err != nil {
		t.Fatal(err)
	}
	s1, err1 := os.Stat(p1)
	s2, err2 := os.Stat(p2)

	if !shouldRetryUnlink(s1, err1, nil, os.ErrNotExist) {
		t.Error("expected retry when path was unlinked (os.ErrNotExist)")
	}
	if !shouldRetryUnlink(s1, err1, s2, err2) {
		t.Error("expected retry when statF != statP")
	}
	if shouldRetryUnlink(s1, err1, s1, err1) {
		t.Error("expected no retry when statF == statP")
	}
}
