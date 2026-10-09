package selfupdate

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func BenchmarkVerifyChecksum(b *testing.B) {
	tmpDir := b.TempDir()
	archivePath := filepath.Join(tmpDir, "upp-linux-amd64.tar.gz")
	dummyData := bytes.Repeat([]byte("test binary archive chunk data\n"), 32) // ~1 KB
	if err := os.WriteFile(archivePath, dummyData, 0o600); err != nil {
		b.Fatalf("failed to write dummy archive: %v", err)
	}

	sum := sha256.Sum256(dummyData)
	validHash := hex.EncodeToString(sum[:])

	var buf bytes.Buffer
	targetName := "upp-linux-amd64.tar.gz"
	// Generate 50 realistic entries
	for i := 0; i < 50; i++ {
		if i == 25 {
			fmt.Fprintf(&buf, "%s  %s\n", validHash, targetName)
		} else {
			fakeHash := hex.EncodeToString(bytes.Repeat([]byte{byte(i + 1)}, 32))
			fmt.Fprintf(&buf, "%s  upp-artifact-%d.tar.gz\n", fakeHash, i)
		}
	}
	checksumsBytes := buf.Bytes()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := verifyChecksum(archivePath, checksumsBytes, targetName); err != nil {
			b.Fatalf("verifyChecksum failed: %v", err)
		}
	}
}

func BenchmarkCheckEntry_Valid(b *testing.B) {
	hdr := &tar.Header{
		Name:     "upp-linux-amd64/upp",
		Typeflag: tar.TypeReg,
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := checkEntry(hdr); err != nil {
			b.Fatalf("checkEntry failed: %v", err)
		}
	}
}

func BenchmarkCheckZipEntry_Valid(b *testing.B) {
	zf := &zip.File{
		FileHeader: zip.FileHeader{
			Name: "upp-windows-amd64/upp.exe",
		},
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := checkZipEntry(zf); err != nil {
			b.Fatalf("checkZipEntry failed: %v", err)
		}
	}
}

func BenchmarkStageBinary(b *testing.B) {
	tmpDir := b.TempDir()
	srcPath := filepath.Join(tmpDir, "src-binary")
	dummyData := bytes.Repeat([]byte("binary chunk 1234\n"), 1024*64) // ~1.2 MB
	if err := os.WriteFile(srcPath, dummyData, 0o755); err != nil {
		b.Fatalf("failed to write dummy binary: %v", err)
	}

	destFile, err := os.Create(filepath.Join(tmpDir, "dest-binary"))
	if err != nil {
		b.Fatalf("failed to create dest binary: %v", err)
	}
	defer func() { _ = destFile.Close() }()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := destFile.Seek(0, 0); err != nil {
			b.Fatalf("seek failed: %v", err)
		}
		if err := stageBinary(destFile, srcPath); err != nil {
			b.Fatalf("stageBinary failed: %v", err)
		}
	}
}
