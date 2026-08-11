package official

import (
	"testing"

	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/platform"
)

func TestAllAdaptersCount(t *testing.T) {
	all := AllAdapters()
	if len(all) != 12 {
		t.Errorf("AllAdapters() returned %d adapters, want 12", len(all))
	}
}

func TestAllAdaptersImplementInterface(t *testing.T) {
	all := AllAdapters()
	for _, a := range all {
		// AllAdapters() already returns []adapters.Adapter, so interface
		// conformance is enforced by the compiler at the return sites.

		// Name must be non-empty.
		if a.Name() == "" {
			t.Errorf("adapter %T has empty Name()", a)
		}

		// Info must return valid metadata.
		info := a.Info()
		if info.ID == "" {
			t.Errorf("adapter %T has empty Info().ID", a)
		}
		if info.Name == "" {
			t.Errorf("adapter %T has empty Info().Name", a)
		}
		if len(info.Platforms) == 0 {
			t.Errorf("adapter %T has empty Info().Platforms", a)
		}
		if info.Trust != adapters.TrustOfficial {
			t.Errorf("adapter %T has trust level %d, want TrustOfficial", a, info.Trust)
		}
	}
}

func TestAllAdaptersUniqueNames(t *testing.T) {
	all := AllAdapters()
	seen := make(map[string]bool)
	for _, a := range all {
		name := a.Name()
		if seen[name] {
			t.Errorf("duplicate adapter name: %s", name)
		}
		seen[name] = true
	}
}

func TestAdaptersForPlatformLinux(t *testing.T) {
	result := AdaptersForPlatform(platform.OSLinux)

	ids := make(map[string]bool)
	for _, a := range result {
		ids[a.Name()] = true
	}

	// Linux should include: apt, brew, nvm, npm, pnpm, bun, gh, docker, go, opencode
	expectedPresent := []string{"apt", "brew", "nvm", "npm", "pnpm", "bun", "gh", "docker", "go", "opencode"}
	for _, id := range expectedPresent {
		if !ids[id] {
			t.Errorf("AdaptersForPlatform(linux) missing adapter: %s", id)
		}
	}

	// Linux should NOT include: winget, scoop
	expectedAbsent := []string{"winget", "scoop"}
	for _, id := range expectedAbsent {
		if ids[id] {
			t.Errorf("AdaptersForPlatform(linux) should not contain: %s", id)
		}
	}
}

func TestAdaptersForPlatformMacOS(t *testing.T) {
	result := AdaptersForPlatform(platform.OSMacOS)

	ids := make(map[string]bool)
	for _, a := range result {
		ids[a.Name()] = true
	}

	// macOS should include: brew, nvm, npm, pnpm, bun, gh, docker, go, opencode
	if !ids["brew"] {
		t.Error("AdaptersForPlatform(macos) missing brew")
	}
	if !ids["nvm"] {
		t.Error("AdaptersForPlatform(macos) missing nvm")
	}
	if !ids["gh"] {
		t.Error("AdaptersForPlatform(macos) missing gh")
	}

	// macOS should NOT include: apt, winget, scoop
	if ids["apt"] {
		t.Error("AdaptersForPlatform(macos) should not contain apt")
	}
	if ids["winget"] {
		t.Error("AdaptersForPlatform(macos) should not contain winget")
	}
	if ids["scoop"] {
		t.Error("AdaptersForPlatform(macos) should not contain scoop")
	}
}

func TestAdaptersForPlatformWindows(t *testing.T) {
	result := AdaptersForPlatform(platform.OSWindows)

	ids := make(map[string]bool)
	for _, a := range result {
		ids[a.Name()] = true
	}

	// Windows should include: winget, scoop, nvm, npm, pnpm, bun, gh, docker, go, opencode
	if !ids["winget"] {
		t.Error("AdaptersForPlatform(windows) missing winget")
	}
	if !ids["scoop"] {
		t.Error("AdaptersForPlatform(windows) missing scoop")
	}
	if !ids["nvm"] {
		t.Error("AdaptersForPlatform(windows) missing nvm")
	}

	// Windows should NOT include: apt, brew
	if ids["apt"] {
		t.Error("AdaptersForPlatform(windows) should not contain apt")
	}
	if ids["brew"] {
		t.Error("AdaptersForPlatform(windows) should not contain brew")
	}
}

func TestAdapterByName(t *testing.T) {
	tests := []struct {
		name    string
		wantNil bool
	}{
		{"apt", false},
		{"brew", false},
		{"winget", false},
		{"scoop", false},
		{"nvm", false},
		{"npm", false},
		{"pnpm", false},
		{"bun", false},
		{"gh", false},
		{"docker", false},
		{"go", false},
		{"opencode", false},
		{"nonexistent", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := AdapterByName(tt.name)
			if tt.wantNil {
				if a != nil {
					t.Errorf("AdapterByName(%q) = %v, want nil", tt.name, a)
				}
			} else {
				if a == nil {
					t.Errorf("AdapterByName(%q) = nil, want non-nil", tt.name)
				} else if a.Name() != tt.name {
					t.Errorf("AdapterByName(%q).Name() = %q", tt.name, a.Name())
				}
			}
		})
	}
}

func TestAdaptersForCurrentPlatform(t *testing.T) {
	p, err := platform.Detect()
	if err != nil {
		t.Skipf("unsupported platform: %v", err)
	}
	result := AdaptersForCurrentPlatform()

	if len(result) == 0 {
		t.Errorf("AdaptersForCurrentPlatform() returned empty for OS %q", p.OS)
	}

	// Verify all returned adapters claim to support the current platform.
	for _, a := range result {
		info := a.Info()
		found := false
		for _, plat := range info.Platforms {
			if plat == p.OS {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("adapter %s does not claim platform %q but was returned", a.Name(), p.OS)
		}
	}
}
