package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Export writes the config as TOML to stdout.
func Export(cfg *Config) error {
	var buf bytes.Buffer
	enc := toml.NewEncoder(&buf)
	if err := enc.Encode(cfg); err != nil {
		return fmt.Errorf("cannot encode config: %w", err)
	}
	fmt.Print(buf.String())
	return nil
}

// ExportToFile writes the config as TOML to the specified path,
// creating directories as needed.
func ExportToFile(cfg *Config, path string) error {
	dir := filepath.Dir(path)

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("cannot create directory: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("cannot create file: %w", err)
	}
	defer func() { _ = f.Close() }()

	enc := toml.NewEncoder(f)
	if err := enc.Encode(cfg); err != nil {
		return fmt.Errorf("cannot encode config: %w", err)
	}

	return nil
}

// ImportFromFile reads a TOML config file and returns a validated Config.
// The config is imported exactly as-is; call ApplyDefaults() separately
// if you want to merge platform defaults for missing tools.
func ImportFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read file: %w", err)
	}

	cfg := DefaultConfig()
	if err := toml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("invalid TOML in %s: %w", path, err)
	}

	if err := Validate(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
