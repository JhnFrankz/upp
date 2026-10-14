package catalog

import (
	"slices"
	"testing"

	"github.com/JhnFrankz/upp/internal/platform"
)

func TestAllToolNames(t *testing.T) {
	names := AllToolNames()
	if len(names) != 14 {
		t.Fatalf("AllToolNames() length = %d, want 14", len(names))
	}

	// Verify defensive copy
	names[0] = "mutated"
	fresh := AllToolNames()
	if fresh[0] == "mutated" {
		t.Errorf("AllToolNames() did not return a defensive copy")
	}
}

func TestIsOfficial(t *testing.T) {
	tests := []struct {
		id   string
		want bool
	}{
		{"apt", true},
		{"brew", true},
		{"bun", true},
		{"docker", true},
		{"gh", true},
		{"go", true},
		{"npm", true},
		{"nvm", true},
		{"opencode", true},
		{"pacman", true},
		{"pnpm", true},
		{"scoop", true},
		{"uv", true},
		{"winget", true},
		{"unknown", false},
		{"", false},
		{"APT", false},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			got := IsOfficial(tt.id)
			if got != tt.want {
				t.Errorf("IsOfficial(%q) = %v, want %v", tt.id, got, tt.want)
			}
		})
	}
}

func TestIsManager(t *testing.T) {
	tests := []struct {
		id   string
		want bool
	}{
		{"apt", true},
		{"brew", true},
		{"pacman", true},
		{"winget", true},
		{"scoop", true},
		{"npm", false},
		{"nvm", false},
		{"go", false},
		{"docker", false},
		{"", false},
		{"brew-cask", false},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			got := IsManager(tt.id)
			if got != tt.want {
				t.Errorf("IsManager(%q) = %v, want %v", tt.id, got, tt.want)
			}
		})
	}
}

func TestToolNamesForPlatform(t *testing.T) {
	tests := []struct {
		os       string
		wantLen  int
		contains string
	}{
		{platform.OSLinux, 12, "apt"},
		{platform.OSMacOS, 10, "brew"},
		{platform.OSWindows, 10, "winget"},
		{"unsupported", 0, ""},
	}

	for _, tt := range tests {
		t.Run(tt.os, func(t *testing.T) {
			names := ToolNamesForPlatform(tt.os)
			if len(names) != tt.wantLen {
				t.Errorf("ToolNamesForPlatform(%q) length = %d, want %d", tt.os, len(names), tt.wantLen)
			}
			if tt.contains != "" && !slices.Contains(names, tt.contains) {
				t.Errorf("ToolNamesForPlatform(%q) missing expected %q", tt.os, tt.contains)
			}
		})
	}
}

func TestPlatformsFor(t *testing.T) {
	tests := []struct {
		id   string
		want []string
	}{
		{"apt", []string{platform.OSLinux}},
		{"pacman", []string{platform.OSLinux}},
		{"brew", []string{platform.OSLinux, platform.OSMacOS}},
		{"nvm", []string{platform.OSLinux, platform.OSMacOS}},
		{"winget", []string{platform.OSWindows}},
		{"scoop", []string{platform.OSWindows}},
		{"bun", []string{platform.OSLinux, platform.OSMacOS, platform.OSWindows}},
		{"unknown", nil},
		{"", nil},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			got := PlatformsFor(tt.id)
			if !slices.Equal(got, tt.want) {
				t.Errorf("PlatformsFor(%q) = %v, want %v", tt.id, got, tt.want)
			}
		})
	}
}
