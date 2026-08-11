package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/JhnFrankz/upp/internal/config"
)

// probeHome isolates HOME to a fresh temp dir so every probe (audit + init)
// exercises the real CLI against an empty, hermetic config location.
func probeHome(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	return tmpDir
}

// probeSetup creates a HOME with fake tools in PATH and saves a config with a
// single custom tool "evil-tool". Every fake tool appends to the returned
// marker file when invoked — the marker's absence proves the tool command never
// executed. The fake sudo ignores its arguments, so even an executed
// "sudo rm -rf ..." can never damage the host.
func probeSetup(t *testing.T, tool config.CustomTool) (marker string) {
	t.Helper()
	tmpDir := probeHome(t)

	binDir := filepath.Join(tmpDir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	marker = filepath.Join(tmpDir, "marker.txt")
	fakeTools := map[string]string{
		"sudo":          "echo FAKE_SUDO_RAN >> \"" + marker + "\"\nexit 0\n",
		"evil-tool":     "echo EVIL_RAN >> \"" + marker + "\"\nexit 0\n",
		"harmless-tool": "echo HARMLESS_RAN >> \"" + marker + "\"\nexit 0\n",
	}
	for name, body := range fakeTools {
		script := "#!/bin/sh\n" + body
		if err := os.WriteFile(filepath.Join(binDir, name), []byte(script), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", binDir+":"+os.Getenv("PATH"))

	cfg := config.DefaultConfig()
	cfg.Custom["evil-tool"] = tool
	if err := config.Save(cfg); err != nil {
		t.Fatal(err)
	}
	return marker
}
