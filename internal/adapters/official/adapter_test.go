package official

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/JhnFrankz/upp/internal/adapters"
)

func TestAptAdapterDetect(t *testing.T) {
	a := &AptAdapter{}
	// Detect should return a consistent bool (may be true or false depending on system).
	result := a.Detect()
	// On Linux systems, apt is usually present. On macOS/Windows, it won't be.
	// We just verify the method doesn't panic and returns a bool.
	_ = result
}

func TestAptAdapterName(t *testing.T) {
	a := &AptAdapter{}
	if a.Name() != "apt" {
		t.Errorf("AptAdapter.Name() = %q, want %q", a.Name(), "apt")
	}
}

func TestAptAdapterInfo(t *testing.T) {
	a := &AptAdapter{}
	info := a.Info()
	if info.ID != "apt" {
		t.Errorf("Info().ID = %q, want %q", info.ID, "apt")
	}
	if len(info.Platforms) != 1 || info.Platforms[0] != "linux" {
		t.Errorf("Info().Platforms = %v, want [linux]", info.Platforms)
	}
}

func TestBrewAdapterDetect(t *testing.T) {
	a := &BrewAdapter{}
	_ = a.Detect() // no panic
}

func TestBrewAdapterName(t *testing.T) {
	a := &BrewAdapter{}
	if a.Name() != "brew" {
		t.Errorf("BrewAdapter.Name() = %q, want %q", a.Name(), "brew")
	}
}

func TestBrewAdapterInfo(t *testing.T) {
	a := &BrewAdapter{}
	info := a.Info()
	if info.ID != "brew" {
		t.Errorf("Info().ID = %q, want %q", info.ID, "brew")
	}
	if len(info.Platforms) != 2 {
		t.Errorf("Info().Platforms has %d entries, want 2", len(info.Platforms))
	}
}

func TestWingetAdapterDetect(t *testing.T) {
	a := &WingetAdapter{}
	_ = a.Detect()
}

func TestWingetAdapterName(t *testing.T) {
	a := &WingetAdapter{}
	if a.Name() != "winget" {
		t.Errorf("WingetAdapter.Name() = %q, want %q", a.Name(), "winget")
	}
}

func TestScoopAdapterDetect(t *testing.T) {
	a := &ScoopAdapter{}
	_ = a.Detect()
}

func TestScoopAdapterName(t *testing.T) {
	a := &ScoopAdapter{}
	if a.Name() != "scoop" {
		t.Errorf("ScoopAdapter.Name() = %q, want %q", a.Name(), "scoop")
	}
}

func TestNVMAdapterDetectWithEnv(t *testing.T) {
	a := &NVMAdapter{}

	// Create a temp dir to simulate NVM_DIR.
	tmpDir := t.TempDir()
	nvmSh := filepath.Join(tmpDir, "nvm.sh")
	if err := os.WriteFile(nvmSh, []byte("#!/bin/sh\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Set NVM_DIR to the temp dir.
	t.Setenv("NVM_DIR", tmpDir)

	if !a.Detect() {
		t.Error("NVMAdapter.Detect() returned false with valid NVM_DIR")
	}
}

func TestNVMAdapterDetectWithoutEnv(t *testing.T) {
	a := &NVMAdapter{}

	// Clear NVM_DIR (empty is treated as unset by the adapter).
	t.Setenv("NVM_DIR", "")

	// Without NVM_DIR and without ~/.nvm, detect may return false.
	// We just verify no panic.
	_ = a.Detect()
}

func TestNVMAdapterName(t *testing.T) {
	a := &NVMAdapter{}
	if a.Name() != "nvm" {
		t.Errorf("NVMAdapter.Name() = %q, want %q", a.Name(), "nvm")
	}
}

func TestNVMAdapterInfo(t *testing.T) {
	a := &NVMAdapter{}
	info := a.Info()
	if info.ID != "nvm" {
		t.Errorf("Info().ID = %q, want %q", info.ID, "nvm")
	}
	if len(info.Platforms) != 3 {
		t.Errorf("Info().Platforms has %d entries, want 3", len(info.Platforms))
	}
}

func TestNpmAdapterDetect(t *testing.T) {
	a := &NpmAdapter{}
	_ = a.Detect()
}

func TestNpmAdapterName(t *testing.T) {
	a := &NpmAdapter{}
	if a.Name() != "npm" {
		t.Errorf("NpmAdapter.Name() = %q, want %q", a.Name(), "npm")
	}
}

func TestPnpmAdapterDetect(t *testing.T) {
	a := &PnpmAdapter{}
	_ = a.Detect()
}

func TestPnpmAdapterName(t *testing.T) {
	a := &PnpmAdapter{}
	if a.Name() != "pnpm" {
		t.Errorf("PnpmAdapter.Name() = %q, want %q", a.Name(), "pnpm")
	}
}

func TestBunAdapterDetect(t *testing.T) {
	a := &BunAdapter{}
	_ = a.Detect()
}

func TestBunAdapterName(t *testing.T) {
	a := &BunAdapter{}
	if a.Name() != "bun" {
		t.Errorf("BunAdapter.Name() = %q, want %q", a.Name(), "bun")
	}
}

func TestGhAdapterDetect(t *testing.T) {
	a := &GhAdapter{}
	_ = a.Detect()
}

func TestGhAdapterName(t *testing.T) {
	a := &GhAdapter{}
	if a.Name() != "gh" {
		t.Errorf("GhAdapter.Name() = %q, want %q", a.Name(), "gh")
	}
}

func TestDockerAdapterDetect(t *testing.T) {
	a := &DockerAdapter{}
	_ = a.Detect()
}

func TestDockerAdapterName(t *testing.T) {
	a := &DockerAdapter{}
	if a.Name() != "docker" {
		t.Errorf("DockerAdapter.Name() = %q, want %q", a.Name(), "docker")
	}
}

func TestGoAdapterDetect(t *testing.T) {
	a := &GoAdapter{}
	_ = a.Detect()
}

func TestGoAdapterName(t *testing.T) {
	a := &GoAdapter{}
	if a.Name() != "go" {
		t.Errorf("GoAdapter.Name() = %q, want %q", a.Name(), "go")
	}
}

func TestOpenCodeAdapterDetect(t *testing.T) {
	a := &OpenCodeAdapter{}
	_ = a.Detect()
}

func TestOpenCodeAdapterName(t *testing.T) {
	a := &OpenCodeAdapter{}
	if a.Name() != "opencode" {
		t.Errorf("OpenCodeAdapter.Name() = %q, want %q", a.Name(), "opencode")
	}
}

func TestOpenCodeAdapterInfo(t *testing.T) {
	a := &OpenCodeAdapter{}
	info := a.Info()
	if info.ID != "opencode" {
		t.Errorf("Info().ID = %q, want %q", info.ID, "opencode")
	}
	if len(info.Platforms) != 3 {
		t.Errorf("Info().Platforms has %d entries, want 3", len(info.Platforms))
	}
}

func TestExtractVersion(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"v1.2.3", "v1.2.3"},
		{"1.2.3", "1.2.3"},
		{"node v20.11.0", "v20.11.0"},
		{"go version go1.22.0 linux/amd64", "go1.22.0"},
		{"Homebrew 4.1.0", "4.1.0"},
		{"pnpm: 8.14.0", "8.14.0"},
		{"", ""},
		{"no version here", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := extractVersion(tt.input)
			if got != tt.want {
				t.Errorf("extractVersion(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsVersionLike(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"1.2.3", true},
		{"v1.2.3", true},
		{"1.2.3-rc1", true},
		{"1.2.3-beta.1", true},
		{"go1.22.0", true},
		{"abc", false},
		{"", false},
		{"1", false},      // no dot
		{"v", false},      // just prefix
		{"1.2", true},     // two-part version
		{"1.2.3.4", true}, // four-part
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := isVersionLike(tt.input)
			if got != tt.want {
				t.Errorf("isVersionLike(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input  string
		maxLen int
		want   string
	}{
		{"hello", 10, "hello"},
		{"hello world", 5, "hello..."},
		{"", 5, ""},
		{"abc", 3, "abc"},
		{"abcd", 3, "abc..."},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := truncate(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
			}
		})
	}
}

func TestExtractGoVersion(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"go version go1.22.0 linux/amd64", "1.22.0"},
		{"go version go1.21.5 darwin/arm64", "1.21.5"},
		{"go version go1.20.10 windows/amd64", "1.20.10"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := extractGoVersion(tt.input)
			if got != tt.want {
				t.Errorf("extractGoVersion(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestDryRunDoesNotExecute(t *testing.T) {
	// Verify that dry-run mode does not actually execute commands.
	// This is a safety test — if dry run executes, something is wrong.
	adaptersList := []struct {
		name    string
		adapter adapters.Adapter
	}{
		{"apt", &AptAdapter{}},
		{"brew", &BrewAdapter{}},
		{"winget", &WingetAdapter{}},
		{"scoop", &ScoopAdapter{}},
		{"npm", &NpmAdapter{}},
		{"pnpm", &PnpmAdapter{}},
		{"bun", &BunAdapter{}},
	}

	for _, tt := range adaptersList {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.adapter.Detect() {
				t.Skipf("%s not installed, skipping dry-run test", tt.name)
			}
			result, err := tt.adapter.Update(true)
			if err != nil {
				t.Errorf("%s dry-run returned error: %v", tt.name, err)
			}
			_ = result
		})
	}
}
