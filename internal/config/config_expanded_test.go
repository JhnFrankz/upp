package config

import (
	"os"
	"path/filepath"
	"testing"
)

// --- Config Migration Tests ---

func TestConfigVersion_Migration(t *testing.T) {
	tests := []struct {
		name    string
		version int
		wantErr bool
	}{
		{"version 0 invalid", 0, true},
		{"version 1 valid", 1, false},
		{"version 2 future", 2, false},
		{"version -1 invalid", -1, true},
		{"version 100 valid", 100, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Version:  tt.version,
				Settings: Settings{},
				Tools:    make(map[string]ToolConfig),
				Custom:   make(map[string]CustomTool),
			}
			err := Validate(cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate(version=%d) error = %v, wantErr %v", tt.version, err, tt.wantErr)
			}
		})
	}
}

func TestConfigVersion_DefaultValue(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Version != ConfigVersion {
		t.Errorf("DefaultConfig().Version = %d, want %d", cfg.Version, ConfigVersion)
	}
}

func TestConfigVersion_PreservedOnLoad(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cfgDir := filepath.Join(tmpDir, ".config", "upp")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Write config with version 1
	tomlContent := `version = 1

[settings]
`
	if err := os.WriteFile(filepath.Join(cfgDir, "config.toml"), []byte(tomlContent), 0o644); err != nil {
		t.Fatal(err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if loaded.Version != 1 {
		t.Errorf("loaded Version = %d, want 1", loaded.Version)
	}
}

// --- Config Validation Edge Cases ---

func TestValidate_EnabledToolNotOfficialNoCustom(t *testing.T) {
	cfg := &Config{
		Version:  1,
		Settings: Settings{},
		Tools: map[string]ToolConfig{
			"nonexistent-tool": {Enabled: true},
		},
		Custom: make(map[string]CustomTool),
	}

	err := Validate(cfg)
	if err == nil {
		t.Error("Validate() should error for enabled non-official tool without custom command")
	}
}

func TestValidate_EnabledToolWithCustomCommand(t *testing.T) {
	cfg := &Config{
		Version:  1,
		Settings: Settings{},
		Tools: map[string]ToolConfig{
			"mytool": {Enabled: true},
		},
		Custom: map[string]CustomTool{
			"mytool": {Command: "mytool --update"},
		},
	}

	err := Validate(cfg)
	if err != nil {
		t.Errorf("Validate() should not error for enabled tool with custom command: %v", err)
	}
}

func TestValidate_CustomToolMissingCommand(t *testing.T) {
	cfg := &Config{
		Version:  1,
		Settings: Settings{},
		Tools:    make(map[string]ToolConfig),
		Custom: map[string]CustomTool{
			"mytool": {Command: ""},
		},
	}

	err := Validate(cfg)
	if err == nil {
		t.Error("Validate() should error for custom tool with empty command")
	}
}

func TestValidate_MultipleTools(t *testing.T) {
	cfg := &Config{
		Version:  1,
		Settings: Settings{},
		Tools: map[string]ToolConfig{
			"apt":   {Enabled: true},
			"npm":   {Enabled: false},
			"brew":  {Enabled: true},
			"myapp": {Enabled: true},
		},
		Custom: map[string]CustomTool{
			"myapp": {Command: "myapp --update"},
		},
	}

	err := Validate(cfg)
	if err != nil {
		t.Errorf("Validate() should not error: %v", err)
	}
}

// --- Config Save/Load Edge Cases ---

func TestSaveAndLoad_RoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cfg := DefaultConfig()
	cfg.Tools["apt"] = ToolConfig{Enabled: true}
	cfg.Custom["test"] = CustomTool{Command: "test --update", Trusted: true}

	if err := Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if !loaded.Tools["apt"].Enabled {
		t.Error("apt should be enabled")
	}
	if !loaded.Custom["test"].Trusted {
		t.Error("test should be trusted")
	}
}

func TestSave_CreatesDirectories(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cfg := DefaultConfig()
	if err := Save(cfg); err != nil {
		t.Fatalf("Save should create directories: %v", err)
	}

	cfgPath, _ := ConfigPath()
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		t.Error("config file should exist after Save")
	}
}

func TestLoad_InvalidTOML(t *testing.T) {
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

func TestLoad_MissingFile(t *testing.T) {
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

// --- DefaultConfigWithDefaults ---

func TestDefaultConfigWithDefaults_AllPlatforms(t *testing.T) {
	cfg := DefaultConfigWithDefaults()

	if cfg.Version != 1 {
		t.Errorf("version = %d, want 1", cfg.Version)
	}
	if len(cfg.Tools) == 0 {
		t.Error("tools should not be empty")
	}
}

// --- Config Path Tests ---

func TestConfigPath_Linux(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	path, err := ConfigPath()
	if err != nil {
		t.Fatalf("ConfigPath() error: %v", err)
	}

	expected := filepath.Join(tmpDir, ".config", "upp", "config.toml")
	if path != expected {
		t.Errorf("ConfigPath() = %q, want %q", path, expected)
	}
}

func TestConfigDir_Linux(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	dir, err := ConfigDir()
	if err != nil {
		t.Fatalf("ConfigDir() error: %v", err)
	}

	expected := filepath.Join(tmpDir, ".config", "upp")
	if dir != expected {
		t.Errorf("ConfigDir() = %q, want %q", dir, expected)
	}
}

// --- Validate Disabled Tool Not Checked ---

func TestValidate_DisabledToolNotChecked(t *testing.T) {
	cfg := &Config{
		Version:  1,
		Settings: Settings{},
		Tools: map[string]ToolConfig{
			"nonexistent": {Enabled: false},
		},
		Custom: make(map[string]CustomTool),
	}

	err := Validate(cfg)
	if err != nil {
		t.Errorf("Validate() should not error for disabled non-official tool: %v", err)
	}
}
