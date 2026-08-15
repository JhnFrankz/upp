package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JhnFrankz/upp/internal/config"
)

// runInitCmd executes `upp init` with the given args through the real CLI
// (root.Execute). When stdin is non-empty, it is served from a temp file
// replacing os.Stdin so fmt.Scanln prompts read deterministic input.
// Returns the captured stdout plus the error from root.Execute().
func runInitCmd(t *testing.T, stdin string, args ...string) (string, error) {
	t.Helper()
	if stdin != "" {
		f, err := os.CreateTemp("", "upp-init-stdin-*")
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = f.Close() }()
		defer func() { _ = os.Remove(f.Name()) }()
		if _, err := f.WriteString(stdin); err != nil {
			t.Fatal(err)
		}
		if _, err := f.Seek(0, 0); err != nil {
			t.Fatal(err)
		}
		origStdin := os.Stdin
		os.Stdin = f
		defer func() { os.Stdin = origStdin }()
	}

	var runErr error
	out := withCapturedStdout(func() {
		root, gf := BuildRoot()
		AddCommands(root, gf)
		root.SetArgs(append([]string{"init"}, args...))
		runErr = root.Execute()
	})
	return out, runErr
}

// Probe: first run — no config file exists — must run the wizard and CREATE
// the config. First-run state comes from explicit file existence
// (config-system: missing file → wizard runs and creates config).
func TestInitProbe_MissingConfig_WizardCreates(t *testing.T) {
	tmpDir := probeHome(t)

	out, err := runInitCmd(t, "")
	if err != nil {
		t.Fatalf("first-run init should not error: %v", err)
	}
	cfgPath := filepath.Join(tmpDir, ".config", "upp", "config.toml")
	if _, statErr := os.Stat(cfgPath); statErr != nil {
		t.Errorf("first-run init must create config at %s (wizard never ran)", cfgPath)
	}
	if !strings.Contains(out, "Config written to") {
		t.Errorf("init should report config creation, got: %q", out)
	}
}

// Probe: existing config — interactive init must PROMPT for overwrite and,
// unless confirmed, leave the file byte-for-byte unchanged.
func TestInitProbe_ExistingConfig_PromptsAndPreserves(t *testing.T) {
	tmpDir := probeHome(t)

	cfg := config.DefaultConfig()
	cfg.Custom["keepme"] = config.CustomTool{Command: "keepme --update", Trusted: true}
	if err := config.Save(cfg); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(tmpDir, ".config", "upp", "config.toml")
	before, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}

	out, err := runInitCmd(t, "n\n")
	if err != nil {
		t.Fatalf("init with existing config should not error on deny: %v", err)
	}
	if !strings.Contains(out, "Overwrite with new detection?") {
		t.Errorf("existing-config init must prompt for overwrite; output: %q", out)
	}
	after, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Error("init without confirmation must not change the existing config file")
	}
}

// Probe (triangulation): existing config confirmed with "y" — the wizard
// overwrites with a fresh detection-based config.
func TestInitProbe_ExistingConfig_ConfirmedOverwrites(t *testing.T) {
	probeHome(t)

	cfg := config.DefaultConfig()
	if err := config.Save(cfg); err != nil {
		t.Fatal(err)
	}

	out, err := runInitCmd(t, "y\n")
	if err != nil {
		t.Fatalf("init with confirmed overwrite should not error: %v", err)
	}
	if !strings.Contains(out, "Config written to") {
		t.Errorf("confirmed overwrite should regenerate config, got: %q", out)
	}
}
