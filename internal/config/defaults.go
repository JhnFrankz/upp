package config

import "github.com/JhnFrankz/upp/internal/platform"

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
	catalog := platform.CatalogFor(p.OS)

	for _, tool := range catalog {
		if _, exists := cfg.Tools[tool.ID]; !exists {
			cfg.Tools[tool.ID] = ToolConfig{Enabled: true}
		}
	}

}

// DefaultConfigWithDefaults returns a config pre-populated with platform catalog defaults.
func DefaultConfigWithDefaults() *Config {
	cfg := DefaultConfig()
	ApplyDefaults(cfg)
	return cfg
}
