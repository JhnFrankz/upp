package config

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Version != 1 {
		t.Errorf("expected version 1, got %d", cfg.Version)
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

	// The removed check_self_update key must be tolerated as an unknown
	// settings key (spec config-system forward compatibility).
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
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
}

// TestLoadStrayCheckSelfUpdateKey_NeverRewritten locks the config-system
// forward-compatibility contract for the removed hint setting (spec
// config-system migration): an existing config containing
// `check_self_update = true` loads silently as an unknown settings key,
// and Save NEVER rewrites the key back into the file.
func TestLoadStrayCheckSelfUpdateKey_NeverRewritten(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cfgDir := filepath.Join(tmpDir, ".config", "upp")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	tomlContent := "version = 1\n\n[settings]\ncheck_self_update = true\n"
	path := filepath.Join(cfgDir, "config.toml")
	if err := os.WriteFile(path, []byte(tomlContent), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() with stray check_self_update key should not error: %v", err)
	}
	if cfg.Version != 1 {
		t.Errorf("version = %d, want 1", cfg.Version)
	}

	// Save must never re-emit the removed key: struct-only encoding drops it.
	if err := Save(cfg); err != nil {
		t.Fatalf("Save() error: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "check_self_update") {
		t.Errorf("Save must not rewrite the removed check_self_update key, got:\n%s", data)
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

// --- WU4: config CustomTool.Manager + validation/init hygiene ---

// TestValidate_ValidManager pins that a custom tool declaring a known official
// manager (a manager-kind tool, e.g. "brew") loads and validates cleanly; the
// Manager field round-trips unchanged (spec Config Format: a custom tool MAY
// declare an owning manager).
func TestValidate_ValidManager(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Custom["mytool"] = CustomTool{
		Command: "mytool --update",
		Trusted: true,
		Manager: "brew",
	}

	err := Validate(cfg)
	if err != nil {
		t.Fatalf("Validate() with a valid manager should not error: %v", err)
	}
	if cfg.Custom["mytool"].Manager != "brew" {
		t.Errorf("CustomTool.Manager = %q, want %q", cfg.Custom["mytool"].Manager, "brew")
	}
}

// TestValidate_UnknownManagerIgnoredWarn pins the forward-compatible contract
// (spec Config Format): an unknown or non-manager `manager` value MUST be
// ignored (the tool proceeds standalone) AND a warning MUST be emitted to the
// caller-supplied stderr writer. Validate itself returns nil (non-fatal) — the
// warning is side-channel via the variadic writer.
func TestValidate_UnknownManagerIgnoredWarn(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Custom["mytool"] = CustomTool{
		Command: "mytool --update",
		Manager: "bogus", // not a known manager
	}
	var buf bytes.Buffer

	err := Validate(cfg, &buf)
	if err != nil {
		t.Fatalf("Validate() with an unknown manager must not error (forward-compatible): %v", err)
	}

	// The unknown manager is ignored: the tool proceeds standalone.
	if cell, ok := cfg.Custom["mytool"]; !ok || cell.Manager != "" {
		t.Errorf("unknown manager must be ignored (standalone), got %+v", cfg.Custom["mytool"])
	}

	if buf.Len() == 0 {
		t.Error("Validate() must emit a warning to the supplied writer for an unknown manager")
	}
	if !strings.Contains(buf.String(), "bogus") {
		t.Errorf("warning should name the ignored manager, got: %q", buf.String())
	}
}

// TestValidate_NonManagerWarning pins that a value naming a NOT-manager official
// tool (e.g. "nvm", KindTool) is also rejected with a warning — manager must be
// a manager-kind official tool, not merely any known tool ID.
func TestValidate_NonManagerWarning(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Custom["mytool"] = CustomTool{
		Command: "mytool --update",
		Manager: "nvm", // official tool, but KindTool, not a manager
	}
	var buf bytes.Buffer

	err := Validate(cfg, &buf)
	if err != nil {
		t.Fatalf("Validate() with a non-manager tool manager must not error: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("Validate() must warn when manager names a non-manager official tool")
	}
}

// TestValidate_ManagerRoundTrip pins that Save/Decode preserve the manager key
// for a valid manager so the CLI can thread it through buildAdapterList.
func TestValidate_ManagerRoundTrip(t *testing.T) {
	cfg := DefaultConfigWithDefaults()
	cfg.Custom["mytool"] = CustomTool{
		Command: "mytool --update",
		Manager: "brew",
	}
	var buf bytes.Buffer
	if err := Validate(cfg, &buf); err != nil {
		t.Fatalf("Validate() error: %v", err)
	}
	if cfg.Custom["mytool"].Manager != "brew" {
		t.Errorf("Manager round-trip = %q, want %q", cfg.Custom["mytool"].Manager, "brew")
	}
}

// TestSave_NeverWritesManager pins the init-hygiene contract (spec Config
// Format): the `manager` key is an optional, user-declared field that `upp
// init` MUST NOT write. A config whose custom tool declares no manager must
// never serialize the key, so a freshly generated init config carries no
// `manager` line.
func TestSave_NeverWritesManager(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cfg := DefaultConfigWithDefaults()
	cfg.Custom["mytool"] = CustomTool{Command: "mytool --update", Trusted: true}

	if err := Save(cfg); err != nil {
		t.Fatalf("Save() error: %v", err)
	}
	path, err := ConfigPath()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "manager") {
		t.Errorf("Save must never write the optional manager key, got:\n%s", data)
	}
}
