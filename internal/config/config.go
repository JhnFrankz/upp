// Package config manages the TOML configuration file for upp.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/JhnFrankz/upp/internal/platform"
)

// ConfigVersion is the current config schema version.
const ConfigVersion = 1

// Config is the root configuration structure.
type Config struct {
	Version  int                   `toml:"version"`
	Settings Settings              `toml:"settings"`
	Tools    map[string]ToolConfig `toml:"tools"`
	Custom   map[string]CustomTool `toml:"custom"`
}

// Settings holds global preferences.
type Settings struct {
	Language    string `toml:"language"`
	Interactive bool   `toml:"interactive"`
}

// ToolConfig holds per-tool configuration.
type ToolConfig struct {
	Enabled   bool     `toml:"enabled"`
	Platforms []string `toml:"platforms,omitempty"`
}

// CustomTool holds a user-defined tool configuration.
type CustomTool struct {
	Command  string `toml:"command"`
	CheckCmd string `toml:"check_cmd,omitempty"`
	Trusted  bool   `toml:"trusted"`
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Version: ConfigVersion,
		Settings: Settings{
			Language:    "en",
			Interactive: true,
		},
		Tools:  make(map[string]ToolConfig),
		Custom: make(map[string]CustomTool),
	}
}

// ConfigDir returns the platform-appropriate config directory.
func ConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}

	p, err := platform.Detect()
	if err != nil {
		// Fallback to Linux-style path for unsupported platforms
		return filepath.Join(home, ".config", "upp"), nil
	}
	switch p.OS {
	case platform.OSWindows:
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		return filepath.Join(appData, "upp"), nil
	default:
		return filepath.Join(home, ".config", "upp"), nil
	}
}

// ConfigPath returns the full path to config.toml.
func ConfigPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.toml"), nil
}

// Load reads and parses the config file. Returns defaults if file doesn't exist.
func Load() (*Config, error) {
	cfg := DefaultConfig()

	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return cfg, nil // first run: use defaults
	}
	if err != nil {
		return nil, fmt.Errorf("cannot read config: %w", err)
	}

	if err := toml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("invalid TOML in %s: %w", path, err)
	}

	if err := Validate(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Save writes the config to disk, creating directories as needed.
func Save(cfg *Config) error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("cannot create config directory: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("cannot create config file: %w", err)
	}
	defer f.Close()

	enc := toml.NewEncoder(f)
	if err := enc.Encode(cfg); err != nil {
		return fmt.Errorf("cannot encode config: %w", err)
	}

	return nil
}

// Validate checks the config for structural issues.
// Returns warnings as nil (non-fatal) — callers should check warnings separately.
func Validate(cfg *Config) error {
	if cfg.Version < 1 {
		return fmt.Errorf("config version must be >= 1, got %d", cfg.Version)
	}

	if cfg.Settings.Language == "" {
		cfg.Settings.Language = "en"
	}

	// Detect current platform for compatibility checks
	currentOS, _ := platform.Detect()

	// Validate tools reference official catalog where possible
	for id, tool := range cfg.Tools {
		if tool.Enabled && !platform.IsOfficialTool(id) && cfg.Custom[id].Command == "" {
			return fmt.Errorf("tool %q is enabled but not official and has no custom command", id)
		}

		// Warn if tool is enabled but not available on current platform
		if tool.Enabled && currentOS.OS != "" {
			toolPlatforms := tool.Platforms
			if len(toolPlatforms) == 0 {
				// No platform restriction — check official catalog
				for _, official := range platform.OfficialTools {
					if official.ID == id {
						toolPlatforms = official.Platforms
						break
					}
				}
			}
			if len(toolPlatforms) > 0 {
				supported := false
				for _, p := range toolPlatforms {
					if p == currentOS.OS {
						supported = true
						break
					}
				}
				if !supported {
					// Non-fatal: disable the tool and continue
					enabled := false
					tool.Enabled = enabled
					cfg.Tools[id] = tool
				}
			}
		}
	}

	// Validate custom tools have required fields
	for id, custom := range cfg.Custom {
		if custom.Command == "" {
			return fmt.Errorf("custom tool %q missing required 'command' field", id)
		}
	}

	return nil
}
