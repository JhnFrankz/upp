package config

import (
	"github.com/JhnFrankz/upp/internal/adapters/official"
	"github.com/JhnFrankz/upp/internal/platform"
)

// ApplyDefaults merges the official platform catalog into the config.
// For each official tool on the current platform that is not already
// configured, it adds a ToolConfig with Enabled=true.
func ApplyDefaults(cfg *Config) {
	if cfg.Tools == nil {
		cfg.Tools = make(map[string]ToolConfig)
	}

	p, err := platform.Detect()
	if err != nil {
		return // unsupported platform, skip catalog merge
	}
	adapters := official.AdaptersForPlatform(p.OS)

	for _, a := range adapters {
		if _, exists := cfg.Tools[a.Name()]; !exists {
			cfg.Tools[a.Name()] = ToolConfig{Enabled: true}
		}
	}
}

// DefaultConfigWithDefaults returns a config pre-populated with platform catalog defaults.
func DefaultConfigWithDefaults() *Config {
	cfg := DefaultConfig()
	ApplyDefaults(cfg)
	return cfg
}
