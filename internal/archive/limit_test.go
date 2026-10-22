package archive_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/JhnFrankz/upp/internal/archive"
)

func TestExtractToFileWithLimit(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("succeeds when content is smaller than limit", func(t *testing.T) {
		targetPath := filepath.Join(tempDir, "small.bin")
		content := []byte("small content")
		err := archive.ExtractToFileWithLimit(targetPath, bytes.NewReader(content), 0644, int64(len(content)+10))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got, err := os.ReadFile(targetPath)
		if err != nil {
			t.Fatalf("ReadFile() error: %v", err)
		}
		if !bytes.Equal(got, content) {
			t.Fatalf("got %q, want %q", got, content)
		}
	})

	t.Run("stops and cleans up when content exceeds limit", func(t *testing.T) {
		targetPath := filepath.Join(tempDir, "oversized.bin")
		content := bytes.Repeat([]byte("A"), 1024)
		limit := int64(512)
		err := archive.ExtractToFileWithLimit(targetPath, bytes.NewReader(content), 0644, limit)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, archive.ErrDecompressionLimit) {
			t.Fatalf("expected ErrDecompressionLimit, got %v", err)
		}
		if _, statErr := os.Stat(targetPath); !os.IsNotExist(statErr) {
			t.Errorf("expected partial file to be removed, got stat error: %v", statErr)
		}
	})

	t.Run("succeeds when content exactly equals limit", func(t *testing.T) {
		targetPath := filepath.Join(tempDir, "exact.bin")
		content := []byte("exact content")
		err := archive.ExtractToFileWithLimit(targetPath, bytes.NewReader(content), 0644, int64(len(content)))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, statErr := os.Stat(targetPath); statErr != nil {
			t.Fatalf("expected file to exist, got: %v", statErr)
		}
	})

	t.Run("succeeds when content is empty and limit is zero", func(t *testing.T) {
		targetPath := filepath.Join(tempDir, "zero_empty.bin")
		err := archive.ExtractToFileWithLimit(targetPath, bytes.NewReader(nil), 0644, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		fi, statErr := os.Stat(targetPath)
		if statErr != nil {
			t.Fatalf("expected file to exist, got: %v", statErr)
		}
		if fi.Size() != 0 {
			t.Fatalf("expected 0 bytes, got %d", fi.Size())
		}
	})

	t.Run("fails when content is non-empty and limit is zero", func(t *testing.T) {
		targetPath := filepath.Join(tempDir, "zero_nonempty.bin")
		err := archive.ExtractToFileWithLimit(targetPath, bytes.NewReader([]byte("X")), 0644, 0)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, archive.ErrDecompressionLimit) {
			t.Fatalf("expected ErrDecompressionLimit, got: %v", err)
		}
		if _, statErr := os.Stat(targetPath); !os.IsNotExist(statErr) {
			t.Errorf("expected partial file to be removed, got stat error: %v", statErr)
		}
	})

	t.Run("defaults to MaxExtractFileSize when limit is negative", func(t *testing.T) {
		targetPath := filepath.Join(tempDir, "negative_limit.bin")
		content := []byte("default limit content")
		err := archive.ExtractToFileWithLimit(targetPath, bytes.NewReader(content), 0644, -1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got, err := os.ReadFile(targetPath)
		if err != nil {
			t.Fatalf("ReadFile() error: %v", err)
		}
		if !bytes.Equal(got, content) {
			t.Fatalf("got %q, want %q", got, content)
		}
	})

	t.Run("returns error when reader is nil", func(t *testing.T) {
		targetPath := filepath.Join(tempDir, "nil_reader.bin")
		err := archive.ExtractToFileWithLimit(targetPath, nil, 0644, 100)
		if err == nil {
			t.Fatal("expected error for nil reader, got nil")
		}
		if _, statErr := os.Stat(targetPath); !os.IsNotExist(statErr) {
			t.Errorf("expected file not to be created, got stat: %v", statErr)
		}
	})
}
