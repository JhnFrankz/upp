package config

import (
	"bytes"
	"testing"

	"github.com/BurntSushi/toml"
)

func FuzzParseConfig(f *testing.F) {
	seeds := [][]byte{
		[]byte("version = 1\n"),
		[]byte(`version = 1

[settings]

[tools.git]
enabled = true

[tools.go]
enabled = true
`),
		[]byte(`version = 1

[custom.mytool]
command = "echo update"
check_cmd = "echo 1.0.0"
trusted = true
manager = "brew"
package = "mytool"
`),
		[]byte("version = 0\n"),
		[]byte("[tools.git]\nenabled = true\n"),
		[]byte("invalid toml [[]]\n"),
		[]byte(""),
		[]byte("version = -1\n"),
		[]byte(`version = 1
[tools.git]
enabled = false
platforms = ["linux", "darwin", "windows"]
`),
		[]byte(`version = 1
[custom.foo]
command = ""
manager = "unknown_mgr"
`),
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		cfg := DefaultConfig()
		if err := toml.Unmarshal(data, cfg); err != nil {
			return
		}
		var warn bytes.Buffer
		_ = Validate(cfg, &warn)
		ApplyDefaults(cfg)
	})
}
