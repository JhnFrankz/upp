package selfupdate

import (
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
