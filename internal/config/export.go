package config

import (
	"bytes"
	"fmt"
	"os"

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
	dir := path[:len(path)-len("/config.toml")]
	if dir == path {
		// Fallback: use parent directory
		for i := len(path) - 1; i >= 0; i-- {
			if path[i] == '/' {
				dir = path[:i]
				break
			}
		}
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("cannot create directory: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("cannot create file: %w", err)
	}
	defer f.Close()

	enc := toml.NewEncoder(f)
	if err := enc.Encode(cfg); err != nil {
		return fmt.Errorf("cannot encode config: %w", err)
	}

	return nil
}

// ImportFromFile reads a TOML config file and returns a validated Config.
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
