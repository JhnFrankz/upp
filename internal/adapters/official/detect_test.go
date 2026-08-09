package official

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/JhnFrankz/upp/internal/adapters"
)

// TestDetect verifies Detect() for every lookPath-based official adapter,
// hermetically: the lookPathFn seam decides whether the tool exists, so no
// real PATH lookup ever happens.
func TestDetect(t *testing.T) {
	lookPathAdapters := []struct {
		name    string
		adapter adapters.Adapter
		tool    string
	}{
		{"apt", &AptAdapter{}, "apt"},
		{"brew", &BrewAdapter{}, "brew"},
		{"npm", &NpmAdapter{}, "npm"},
		{"pnpm", &PnpmAdapter{}, "pnpm"},
		{"bun", &BunAdapter{}, "bun"},
		{"gh", &GhAdapter{}, "gh"},
		{"docker", &DockerAdapter{}, "docker"},
		{"go", &GoAdapter{}, "go"},
		{"opencode", &OpenCodeAdapter{}, "opencode"},
		{"winget", &WingetAdapter{}, "winget"},
		{"scoop", &ScoopAdapter{}, "scoop"},
	}

	for _, tt := range lookPathAdapters {
		t.Run(tt.name+"/on-path", func(t *testing.T) {
			setExecFakes(t, execFakes{lookPath: map[string]bool{tt.tool: true}})
			if got := tt.adapter.Detect(); !got {
				t.Errorf("%s.Detect() = false, want true when %s is on PATH", tt.name, tt.tool)
			}
		})
		t.Run(tt.name+"/missing", func(t *testing.T) {
			setExecFakes(t, execFakes{lookPath: map[string]bool{tt.tool: false}})
			if got := tt.adapter.Detect(); got {
				t.Errorf("%s.Detect() = true, want false when %s is not on PATH", tt.name, tt.tool)
			}
		})
	}
}

// TestNVMDetect covers the nvm adapter, which detects via NVM_DIR/nvm.sh or
// ~/.nvm/nvm.sh instead of lookPath. HOME is isolated to a temp dir so the
// real user home can never leak into the result.
func TestNVMDetect(t *testing.T) {
	t.Run("NVM_DIR with nvm.sh", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "nvm.sh"), []byte("#!/bin/sh\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Setenv("NVM_DIR", dir)
		t.Setenv("HOME", t.TempDir())

		if got := (&NVMAdapter{}).Detect(); !got {
			t.Error("NVMAdapter.Detect() = false, want true with NVM_DIR/nvm.sh present")
		}
	})

	t.Run("no NVM_DIR and no ~/.nvm", func(t *testing.T) {
		t.Setenv("NVM_DIR", "")
		t.Setenv("HOME", t.TempDir())

		if got := (&NVMAdapter{}).Detect(); got {
			t.Error("NVMAdapter.Detect() = true, want false without NVM_DIR and without ~/.nvm")
		}
	})
}
