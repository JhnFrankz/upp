package config

import (
	"github.com/JhnFrankz/upp/internal/adapters/official"
)

// ApplyDefaults merges the official platform catalog into the config.
// For each official tool on the current platform that is not already
// configured, it adds a ToolConfig with Enabled=true.
func ApplyDefaults(cfg *Config) {
	if cfg.Tools == nil {
		cfg.Tools = make(map[string]ToolConfig)
	}

	p, err := detectPlatformFn()
	if err != nil {
		return // unsupported platform, skip catalog merge
	}
	names := official.ToolNamesForPlatform(p.OS)

	for _, name := range names {
		if _, exists := cfg.Tools[name]; !exists {
			cfg.Tools[name] = ToolConfig{Enabled: true}
		}
	}
}

// DefaultConfigWithDefaults returns a config pre-populated with platform catalog defaults.
func DefaultConfigWithDefaults() *Config {
	cfg := DefaultConfig()
	ApplyDefaults(cfg)
	return cfg
}
