package cli

import (
	"testing"
)

// probeHome isolates HOME to a fresh temp dir so every probe (audit + init)
// exercises the real CLI against an empty, hermetic config location.
func probeHome(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	return tmpDir
}
