package cli

import (
	"path/filepath"
	"testing"
)

// probeHome isolates HOME to a fresh temp dir so every probe (audit + init)
// exercises the real CLI against an empty, hermetic config location.
func probeHome(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)
	t.Setenv("APPDATA", filepath.Join(tmpDir, "AppData", "Roaming"))
	return tmpDir
}
