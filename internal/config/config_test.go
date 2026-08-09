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
	if cfg.Settings.Language != "en" {
		t.Errorf("expected language 'en', got %q", cfg.Settings.Language)
	}
	if !cfg.Settings.Interactive {
		t.Error("expected interactive to be true")
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
				Settings: Settings{Language: "en"},
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
	// Temporarily override home directory to avoid loading real config.
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
	if cfg.Settings.Language != "en" {
		t.Errorf("expected default language 'en', got %q", cfg.Settings.Language)
	}
}

func TestLoadValidTOML(t *testing.T) {
	origHome := os.Getenv("HOME")
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	// Create config directory and file.
	cfgDir := filepath.Join(tmpDir, ".config", "upp")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}

	tomlContent := `
version = 1

[settings]
language = "es"
interactive = false

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

	if cfg.Settings.Language != "es" {
		t.Errorf("expected language 'es', got %q", cfg.Settings.Language)
	}
	if cfg.Settings.Interactive {
		t.Error("expected interactive to be false")
	}
	if !cfg.Tools["apt"].Enabled {
		t.Error("expected apt to be enabled")
	}
}

func TestLoadInvalidTOML(t *testing.T) {
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
