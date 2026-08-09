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
				Settings: Settings{Language: "en"},
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
	origHome := os.Getenv("HOME")
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	cfgDir := filepath.Join(tmpDir, ".config", "upp")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Write config with version 1
	tomlContent := `version = 1

[settings]
language = "en"
interactive = true
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

// --- Export/Import Round-Trip with Complex Configs ---

func TestRoundTrip_ComplexConfig(t *testing.T) {
	tmpDir := t.TempDir()

	original := DefaultConfig()
	original.Settings.Language = "es"
	original.Settings.Interactive = false
	original.Tools["apt"] = ToolConfig{Enabled: true, Platforms: []string{"linux"}}
	original.Tools["npm"] = ToolConfig{Enabled: false}
	original.Tools["brew"] = ToolConfig{Enabled: true, Platforms: []string{"linux", "macos"}}
	original.Custom["deploy"] = CustomTool{
		Command:  "deploy.sh --prod --env=staging",
		CheckCmd: "deploy.sh --version",
		Trusted:  true,
	}
	original.Custom["lint"] = CustomTool{
		Command: "eslint --fix .",
		Trusted: false,
	}

	// Export
	exportPath := filepath.Join(tmpDir, "export.toml")
	if err := ExportToFile(original, exportPath); err != nil {
		t.Fatalf("ExportToFile: %v", err)
	}

	// Import
	imported, err := ImportFromFile(exportPath)
	if err != nil {
		t.Fatalf("ImportFromFile: %v", err)
	}

	// Verify all fields preserved
	if imported.Settings.Language != "es" {
		t.Errorf("language = %q, want %q", imported.Settings.Language, "es")
	}
	if imported.Settings.Interactive {
		t.Error("interactive should be false")
	}
	if !imported.Tools["apt"].Enabled {
		t.Error("apt should be enabled")
	}
	if imported.Tools["npm"].Enabled {
		t.Error("npm should be disabled")
	}
	if !imported.Tools["brew"].Enabled {
		t.Error("brew should be enabled")
	}
	if imported.Tools["apt"].Platforms[0] != "linux" {
		t.Errorf("apt platforms = %v", imported.Tools["apt"].Platforms)
	}
	if imported.Custom["deploy"].Command != "deploy.sh --prod --env=staging" {
		t.Errorf("deploy command = %q", imported.Custom["deploy"].Command)
	}
	if !imported.Custom["deploy"].Trusted {
		t.Error("deploy should be trusted")
	}
	if imported.Custom["lint"].Trusted {
		t.Error("lint should be untrusted")
	}

	// Re-export and verify byte-identical
	reExportPath := filepath.Join(tmpDir, "reexport.toml")
	if err := ExportToFile(imported, reExportPath); err != nil {
		t.Fatalf("ExportToFile (re-export): %v", err)
	}

	origData, _ := os.ReadFile(exportPath)
	reExportData, _ := os.ReadFile(reExportPath)

	if string(origData) != string(reExportData) {
		t.Errorf("round-trip failed:\noriginal:\n%s\nre-exported:\n%s", origData, reExportData)
	}
}

func TestRoundTrip_EmptyCustomTools(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := DefaultConfig()
	cfg.Tools["apt"] = ToolConfig{Enabled: true}

	exportPath := filepath.Join(tmpDir, "export.toml")
	if err := ExportToFile(cfg, exportPath); err != nil {
		t.Fatal(err)
	}

	imported, err := ImportFromFile(exportPath)
	if err != nil {
		t.Fatal(err)
	}

	if !imported.Tools["apt"].Enabled {
		t.Error("apt should be enabled after round-trip")
	}
}

func TestRoundTrip_MultipleCustomTools(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := DefaultConfig()
	cfg.Custom["tool1"] = CustomTool{Command: "tool1 --update", Trusted: true}
	cfg.Custom["tool2"] = CustomTool{Command: "tool2 --update", Trusted: false}
	cfg.Custom["tool3"] = CustomTool{Command: "tool3 --update", Trusted: true, CheckCmd: "tool3 --version"}

	exportPath := filepath.Join(tmpDir, "export.toml")
	if err := ExportToFile(cfg, exportPath); err != nil {
		t.Fatal(err)
	}

	imported, err := ImportFromFile(exportPath)
	if err != nil {
		t.Fatal(err)
	}

	if len(imported.Custom) != 3 {
		t.Errorf("expected 3 custom tools, got %d", len(imported.Custom))
	}
	if !imported.Custom["tool1"].Trusted {
		t.Error("tool1 should be trusted")
	}
	if imported.Custom["tool2"].Trusted {
		t.Error("tool2 should be untrusted")
	}
	if imported.Custom["tool3"].CheckCmd != "tool3 --version" {
		t.Errorf("tool3 check_cmd = %q", imported.Custom["tool3"].CheckCmd)
	}
}

// --- Config Validation Edge Cases ---

func TestValidate_EmptyLanguage(t *testing.T) {
	cfg := &Config{
		Version:  1,
		Settings: Settings{Language: ""},
		Tools:    make(map[string]ToolConfig),
		Custom:   make(map[string]CustomTool),
	}

	err := Validate(cfg)
	if err != nil {
		t.Errorf("Validate() with empty language should not error: %v", err)
	}

	// Should be set to default
	if cfg.Settings.Language != "en" {
		t.Errorf("language should be set to 'en', got %q", cfg.Settings.Language)
	}
}

func TestValidate_EnabledToolNotOfficialNoCustom(t *testing.T) {
	cfg := &Config{
		Version:  1,
		Settings: Settings{Language: "en"},
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
		Settings: Settings{Language: "en"},
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
		Settings: Settings{Language: "en"},
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
		Settings: Settings{Language: "en"},
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
	origHome := os.Getenv("HOME")
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	cfg := DefaultConfig()
	cfg.Settings.Language = "fr"
	cfg.Settings.Interactive = false
	cfg.Tools["apt"] = ToolConfig{Enabled: true}
	cfg.Custom["test"] = CustomTool{Command: "test --update", Trusted: true}

	if err := Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.Settings.Language != "fr" {
		t.Errorf("language = %q, want %q", loaded.Settings.Language, "fr")
	}
	if loaded.Settings.Interactive {
		t.Error("interactive should be false")
	}
	if !loaded.Tools["apt"].Enabled {
		t.Error("apt should be enabled")
	}
	if !loaded.Custom["test"].Trusted {
		t.Error("test should be trusted")
	}
}

func TestSave_CreatesDirectories(t *testing.T) {
	origHome := os.Getenv("HOME")
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

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
	origHome := os.Getenv("HOME")
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

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
	origHome := os.Getenv("HOME")
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() on missing file should not error: %v", err)
	}

	if cfg.Version != 1 {
		t.Errorf("expected default version 1, got %d", cfg.Version)
	}
}

// --- ExportToFile Edge Cases ---

func TestExportToFile_CreatesDirectories(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "deep", "nested", "path", "config.toml")

	cfg := DefaultConfig()
	if err := ExportToFile(cfg, path); err != nil {
		t.Fatalf("ExportToFile should create directories: %v", err)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("exported file should exist")
	}
}

func TestExportToFile_EmptyConfig(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "config.toml")

	cfg := DefaultConfig()
	if err := ExportToFile(cfg, path); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if len(data) == 0 {
		t.Error("exported file is empty")
	}
}

// --- ImportFromFile Edge Cases ---

func TestImportFromFile_MissingFile(t *testing.T) {
	_, err := ImportFromFile("/nonexistent/path/config.toml")
	if err == nil {
		t.Error("ImportFromFile should error on missing file")
	}
}

func TestImportFromFile_InvalidTOML(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "bad.toml")

	badToml := `this is not valid toml {{{`
	if err := os.WriteFile(path, []byte(badToml), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := ImportFromFile(path)
	if err == nil {
		t.Error("ImportFromFile should error on invalid TOML")
	}
}

func TestImportFromFile_InvalidVersion(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "bad-version.toml")

	tomlContent := `version = 0

[settings]
language = "en"
`
	if err := os.WriteFile(path, []byte(tomlContent), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := ImportFromFile(path)
	if err == nil {
		t.Error("ImportFromFile should error on version 0")
	}
}

// --- Export to Stdout ---

func TestExport_Stdout(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Tools["apt"] = ToolConfig{Enabled: true}

	// Capture stdout
	origStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Export(cfg)

	w.Close()
	os.Stdout = origStdout

	if err != nil {
		t.Fatalf("Export() error: %v", err)
	}

	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	if len(output) == 0 {
		t.Error("Export() produced no output")
	}
}

// --- DefaultConfigWithDefaults ---

func TestDefaultConfigWithDefaults_AllPlatforms(t *testing.T) {
	cfg := DefaultConfigWithDefaults()

	if cfg.Version != 1 {
		t.Errorf("version = %d, want 1", cfg.Version)
	}
	if cfg.Settings.Language != "en" {
		t.Errorf("language = %q, want %q", cfg.Settings.Language, "en")
	}
	if !cfg.Settings.Interactive {
		t.Error("interactive should be true")
	}
	if len(cfg.Tools) == 0 {
		t.Error("tools should not be empty")
	}
}

// --- Config Path Tests ---

func TestConfigPath_Linux(t *testing.T) {
	origHome := os.Getenv("HOME")
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

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
	origHome := os.Getenv("HOME")
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	dir, err := ConfigDir()
	if err != nil {
		t.Fatalf("ConfigDir() error: %v", err)
	}

	expected := filepath.Join(tmpDir, ".config", "upp")
	if dir != expected {
		t.Errorf("ConfigDir() = %q, want %q", dir, expected)
	}
}

// --- Validate Language Default ---

func TestValidate_SetsDefaultLanguage(t *testing.T) {
	cfg := &Config{
		Version:  1,
		Settings: Settings{Language: ""},
		Tools:    make(map[string]ToolConfig),
		Custom:   make(map[string]CustomTool),
	}

	_ = Validate(cfg)

	if cfg.Settings.Language != "en" {
		t.Errorf("Validate should set default language to 'en', got %q", cfg.Settings.Language)
	}
}

// --- Validate Disabled Tool Not Checked ---

func TestValidate_DisabledToolNotChecked(t *testing.T) {
	cfg := &Config{
		Version:  1,
		Settings: Settings{Language: "en"},
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
