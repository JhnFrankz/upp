package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Version != 1 {
		t.Errorf("expected version 1, got %d", cfg.Version)
	}
	if cfg.Settings.CheckSelfUpdate {
		t.Error("expected check_self_update to default to false")
	}
	if cfg.Tools == nil {
		t.Error("tools map should not be nil")
	}
	if cfg.Custom == nil {
		t.Error("custom map should not be nil")
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name:    "valid config",
			cfg:     DefaultConfig(),
			wantErr: false,
		},
		{
			name: "invalid version",
			cfg: &Config{
				Version:  0,
				Settings: Settings{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoadMissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() on missing file should not error: %v", err)
	}

	if cfg.Version != 1 {
		t.Errorf("expected default version 1, got %d", cfg.Version)
	}
}

func TestLoadValidTOML(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cfgDir := filepath.Join(tmpDir, ".config", "upp")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}

	tomlContent := `
version = 1

[settings]
check_self_update = false

[tools.apt]
enabled = true
`
	if err := os.WriteFile(filepath.Join(cfgDir, "config.toml"), []byte(tomlContent), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Settings.CheckSelfUpdate {
		t.Error("expected check_self_update to be false")
	}
	if !cfg.Tools["apt"].Enabled {
		t.Error("expected apt to be enabled")
	}
}

func TestLoadInvalidTOML(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cfgDir := filepath.Join(tmpDir, ".config", "upp")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}

	badToml := `this is not valid toml {{{`
	if err := os.WriteFile(filepath.Join(cfgDir, "config.toml"), []byte(badToml), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Load()
	if err == nil {
		t.Error("Load() should error on invalid TOML")
	}
}

// TestLoadCheckSelfUpdate covers the opt-in hint setting (spec
// config-system): absent → false (TOML zero value), explicit false →
// false, explicit true → enabled.
func TestLoadCheckSelfUpdate(t *testing.T) {
	tests := []struct {
		name     string
		settings string // [settings] table body, "" for absent
		want     bool
	}{
		{name: "absent defaults to false", settings: "", want: false},
		{name: "explicit false", settings: "check_self_update = false", want: false},
		{name: "explicit true enables", settings: "check_self_update = true", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Setenv("HOME", tmpDir)

			cfgDir := filepath.Join(tmpDir, ".config", "upp")
			if err := os.MkdirAll(cfgDir, 0o755); err != nil {
				t.Fatal(err)
			}

			tomlContent := "version = 1\n\n[settings]\n"
			if tt.settings != "" {
				tomlContent += tt.settings + "\n"
			}
			if err := os.WriteFile(filepath.Join(cfgDir, "config.toml"), []byte(tomlContent), 0o644); err != nil {
				t.Fatal(err)
			}

			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load() error: %v", err)
			}
			if cfg.Settings.CheckSelfUpdate != tt.want {
				t.Errorf("CheckSelfUpdate = %v, want %v", cfg.Settings.CheckSelfUpdate, tt.want)
			}
		})
	}
}

// --- Phase 2: Defaults Tests ---

func TestApplyDefaultsEmptyConfig(t *testing.T) {
	cfg := &Config{
		Version:  ConfigVersion,
		Settings: Settings{},
		Tools:    nil,
		Custom:   nil,
	}

	ApplyDefaults(cfg)

	if cfg.Tools == nil {
		t.Fatal("ApplyDefaults should initialize Tools map")
	}
	// Should have tools from the platform catalog
	if len(cfg.Tools) == 0 {
		t.Error("expected at least one tool from platform catalog")
	}
}

func TestApplyDefaultsPartialConfig(t *testing.T) {
	cfg := &Config{
		Version:  ConfigVersion,
		Settings: Settings{},
		Tools: map[string]ToolConfig{
			"apt": {Enabled: false},
		},
		Custom: make(map[string]CustomTool),
	}

	ApplyDefaults(cfg)

	// User's explicit setting should be preserved

	// User's explicit tool config should be preserved
	if cfg.Tools["apt"].Enabled {
		t.Error("expected apt to remain disabled (user override)")
	}

	// Other platform tools should be added with defaults
	for id, tool := range cfg.Tools {
		if id == "apt" {
			continue
		}
		if !tool.Enabled {
			t.Errorf("expected tool %q to be enabled by default", id)
		}
	}
}

func TestApplyDefaultsDoesNotOverride(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Tools["npm"] = ToolConfig{Enabled: false}

	ApplyDefaults(cfg)

	if cfg.Tools["npm"].Enabled {
		t.Error("ApplyDefaults should not override existing tool config")
	}
}

func TestDefaultConfigWithDefaults(t *testing.T) {
	cfg := DefaultConfigWithDefaults()

	if cfg.Version != 1 {
		t.Errorf("expected version 1, got %d", cfg.Version)
	}
	if len(cfg.Tools) == 0 {
		t.Error("expected platform tools to be populated")
	}
}

// --- Phase 2: Load-state Tests (D6: defaults only for existing files) ---

func TestExists_MissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	if Exists() {
		t.Error("Exists() should be false when no config file exists")
	}
}

func TestExists_ExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	if err := Save(DefaultConfig()); err != nil {
		t.Fatal(err)
	}
	if !Exists() {
		t.Error("Exists() should be true after Save")
	}
}

// TestLoadStrayInteractiveKey locks the D8 tolerance contract: an existing
// config with a leftover `interactive` key loads silently (unknown settings
// ignored), and export/import NEVER re-emit the key.
func TestLoadStrayInteractiveKey(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cfgDir := filepath.Join(tmpDir, ".config", "upp")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	tomlContent := "version = 1\n\n[settings]\ninteractive = false\n"
	if err := os.WriteFile(filepath.Join(cfgDir, "config.toml"), []byte(tomlContent), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() with stray interactive key should not error: %v", err)
	}
	if cfg.Version != 1 {
		t.Errorf("version = %d, want 1", cfg.Version)
	}
	if cfg.Settings.CheckSelfUpdate {
		t.Error("stray interactive key must not affect check_self_update")
	}
}

func TestLoadMissingFile_NoDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() on missing file should not error: %v", err)
	}
	// D6: ApplyDefaults runs only when the file exists — a missing file must
	// NOT be merged with the platform catalog.
	if len(cfg.Tools) != 0 {
		t.Errorf("missing file: expected no catalog defaults, got %d tools", len(cfg.Tools))
	}
}

func TestLoadEmptyFile_AppliesDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cfgDir := filepath.Join(tmpDir, ".config", "upp")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "config.toml"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() on empty file should not error: %v", err)
	}
	// Empty existing file → all defaults applied; NOT first-run.
	if len(cfg.Tools) == 0 {
		t.Error("empty existing file should get platform catalog defaults")
	}
}

func TestLoadPartialConfig_CatalogDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cfgDir := filepath.Join(tmpDir, ".config", "upp")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	tomlContent := "version = 1\n\n[settings]\nlanguage = \"es\"\n"
	if err := os.WriteFile(filepath.Join(cfgDir, "config.toml"), []byte(tomlContent), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() on partial config should not error: %v", err)
	}
	// Unknown settings (e.g. a leftover language key) are tolerated and
	// ignored; tool sections default to the catalog.
	if len(cfg.Tools) == 0 {
		t.Error("partial config: tool sections should default to the platform catalog")
	}
}

func TestLoadFullConfig_AsIs(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	orig := DefaultConfigWithDefaults()
	orig.Custom["mytool"] = CustomTool{Command: "mytool --update", Trusted: true}
	if err := Save(orig); err != nil {
		t.Fatal(err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() on full config should not error: %v", err)
	}
	if len(loaded.Tools) != len(orig.Tools) {
		t.Errorf("full config: defaults must not add tools (got %d, want %d)", len(loaded.Tools), len(orig.Tools))
	}
	if loaded.Custom["mytool"].Command != "mytool --update" {
		t.Errorf("full config: custom tool lost, got %q", loaded.Custom["mytool"].Command)
	}
}
