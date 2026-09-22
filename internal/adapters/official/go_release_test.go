package official

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFetchGoRelease(t *testing.T) {
	origURL := goReleaseURL
	t.Cleanup(func() { goReleaseURL = origURL })

	sampleJSON := `[
		{
			"version": "go1.22.2",
			"stable": false,
			"files": [
				{"filename": "go1.22.2.linux-amd64.tar.gz", "os": "linux", "arch": "amd64", "version": "go1.22.2", "sha256": "wrong", "kind": "archive"}
			]
		},
		{
			"version": "go1.22.1",
			"stable": true,
			"files": [
				{"filename": "go1.22.1.src.tar.gz", "os": "", "arch": "", "version": "go1.22.1", "sha256": "aaa", "kind": "source"},
				{"filename": "go1.22.1.linux-amd64.tar.gz", "os": "linux", "arch": "amd64", "version": "go1.22.1", "sha256": "beef1234", "kind": "archive"},
				{"filename": "go1.22.1.darwin-arm64.tar.gz", "os": "darwin", "arch": "arm64", "version": "go1.22.1", "sha256": "feed5678", "kind": "archive"}
			]
		}
	]`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") != "upp" {
			t.Errorf("User-Agent = %q, want %q", r.Header.Get("User-Agent"), "upp")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, sampleJSON)
	}))
	defer ts.Close()

	goReleaseURL = ts.URL

	t.Run("finds stable matching linux-amd64 archive", func(t *testing.T) {
		rel, err := fetchGoRelease(context.Background(), "linux", "amd64")
		if err != nil {
			t.Fatalf("fetchGoRelease() unexpected error: %v", err)
		}
		if rel.Version != "go1.22.1" {
			t.Errorf("Version = %q, want %q", rel.Version, "go1.22.1")
		}
		if rel.Filename != "go1.22.1.linux-amd64.tar.gz" {
			t.Errorf("Filename = %q, want %q", rel.Filename, "go1.22.1.linux-amd64.tar.gz")
		}
		if rel.SHA256 != "beef1234" {
			t.Errorf("SHA256 = %q, want %q", rel.SHA256, "beef1234")
		}
		wantURL := "https://go.dev/dl/go1.22.1.linux-amd64.tar.gz"
		if rel.URL != wantURL {
			t.Errorf("URL = %q, want %q", rel.URL, wantURL)
		}
	})

	t.Run("returns error when no matching archive found", func(t *testing.T) {
		_, err := fetchGoRelease(context.Background(), "windows", "arm64")
		if err == nil {
			t.Fatal("fetchGoRelease() expected error for missing release, got nil")
		}
	})

	t.Run("handles HTTP error status", func(t *testing.T) {
		errServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "server error", http.StatusInternalServerError)
		}))
		defer errServer.Close()
		goReleaseURL = errServer.URL

		_, err := fetchGoRelease(context.Background(), "linux", "amd64")
		if err == nil {
			t.Fatal("fetchGoRelease() expected error on HTTP 500, got nil")
		}
	})
}

func TestDownloadAndVerifyGo(t *testing.T) {
	content := "tarball payload data"
	hasher := sha256.New()
	hasher.Write([]byte(content))
	contentSHA := fmt.Sprintf("%x", hasher.Sum(nil))

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") != "upp" {
			t.Errorf("User-Agent = %q, want %q", r.Header.Get("User-Agent"), "upp")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(content))
	}))
	defer ts.Close()

	t.Run("successful download and verification", func(t *testing.T) {
		destDir := t.TempDir()
		destFile := filepath.Join(destDir, "go.tar.gz")
		rel := GoRelease{
			Filename: "go.tar.gz",
			SHA256:   contentSHA,
			URL:      ts.URL,
		}

		err := downloadAndVerifyGo(context.Background(), rel, destFile)
		if err != nil {
			t.Fatalf("downloadAndVerifyGo() unexpected error: %v", err)
		}
		got, err := os.ReadFile(destFile)
		if err != nil {
			t.Fatalf("failed reading downloaded file: %v", err)
		}
		if string(got) != content {
			t.Errorf("downloaded content = %q, want %q", string(got), content)
		}
	})

	t.Run("checksum mismatch removes destination file and fails", func(t *testing.T) {
		destDir := t.TempDir()
		destFile := filepath.Join(destDir, "go.tar.gz")
		rel := GoRelease{
			Filename: "go.tar.gz",
			SHA256:   "0000000000000000000000000000000000000000000000000000000000000000",
			URL:      ts.URL,
		}

		err := downloadAndVerifyGo(context.Background(), rel, destFile)
		if err == nil {
			t.Fatal("downloadAndVerifyGo() expected error on checksum mismatch, got nil")
		}
		if !strings.Contains(err.Error(), "checksum mismatch") {
			t.Errorf("error = %q, want checksum mismatch message", err.Error())
		}
		if _, statErr := os.Stat(destFile); !os.IsNotExist(statErr) {
			t.Errorf("destination file %q was not removed after checksum failure", destFile)
		}
	})

	t.Run("HTTP error aborts and cleans up", func(t *testing.T) {
		errServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "not found", http.StatusNotFound)
		}))
		defer errServer.Close()

		destDir := t.TempDir()
		destFile := filepath.Join(destDir, "go.tar.gz")
		rel := GoRelease{
			Filename: "go.tar.gz",
			SHA256:   contentSHA,
			URL:      errServer.URL,
		}

		err := downloadAndVerifyGo(context.Background(), rel, destFile)
		if err == nil {
			t.Fatal("downloadAndVerifyGo() expected error on HTTP 404, got nil")
		}
		if _, statErr := os.Stat(destFile); !os.IsNotExist(statErr) {
			t.Errorf("destination file %q was created/not cleaned up on HTTP error", destFile)
		}
	})
}

func createTestTarball(t *testing.T, entries map[string]string) string {
	t.Helper()
	tmpDir := t.TempDir()
	archivePath := filepath.Join(tmpDir, "test.tar.gz")

	f, err := os.Create(archivePath)
	if err != nil {
		t.Fatalf("create archive: %v", err)
	}
	defer func() { _ = f.Close() }()

	gw := gzip.NewWriter(f)
	defer func() { _ = gw.Close() }()

	tw := tar.NewWriter(gw)
	defer func() { _ = tw.Close() }()

	for name, content := range entries {
		hdr := &tar.Header{
			Name:     name,
			Mode:     0755,
			Size:     int64(len(content)),
			Typeflag: tar.TypeReg,
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("write tar header: %v", err)
		}
		if _, err := io.WriteString(tw, content); err != nil {
			t.Fatalf("write tar body: %v", err)
		}
	}
	return archivePath
}

func TestExtractGoTarball(t *testing.T) {
	t.Run("safely extracts directory and files", func(t *testing.T) {
		archive := createTestTarball(t, map[string]string{
			"go/bin/go":    "binary go",
			"go/bin/gofmt": "binary gofmt",
			"go/src/hello": "hello world",
		})

		destDir := t.TempDir()
		err := extractGoTarball(archive, destDir)
		if err != nil {
			t.Fatalf("extractGoTarball() unexpected error: %v", err)
		}

		for _, relPath := range []string{"go/bin/go", "go/bin/gofmt", "go/src/hello"} {
			fullPath := filepath.Join(destDir, relPath)
			if _, statErr := os.Stat(fullPath); statErr != nil {
				t.Errorf("expected extracted file at %q, stat err: %v", fullPath, statErr)
			}
		}
	})

	t.Run("rejects zip-slip path traversal attempt", func(t *testing.T) {
		archive := createTestTarball(t, map[string]string{
			"../evil.sh": "rm -rf /",
		})

		destDir := t.TempDir()
		err := extractGoTarball(archive, destDir)
		if err == nil {
			t.Fatal("extractGoTarball() expected error on path traversal, got nil")
		}
	})
}
