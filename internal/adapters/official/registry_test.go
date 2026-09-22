package official

import (
	"testing"

	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/platform"
	"github.com/JhnFrankz/upp/internal/security"
)

func TestAllAdaptersCount(t *testing.T) {
	all := AllAdapters()
	if len(all) != 14 {
		t.Errorf("AllAdapters() returned %d adapters, want 14", len(all))
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
		if info.Trust != security.TrustOfficial {
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

	// Linux should include: apt, brew, pacman, nvm, npm, pnpm, bun, gh, docker, go, opencode, uv
	expectedPresent := []string{"apt", "brew", "pacman", "nvm", "npm", "pnpm", "bun", "gh", "docker", "go", "opencode", "uv"}
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

	// macOS should include: brew, nvm, npm, pnpm, bun, gh, docker, go, opencode, uv
	if !ids["brew"] {
		t.Error("AdaptersForPlatform(macos) missing brew")
	}
	if !ids["nvm"] {
		t.Error("AdaptersForPlatform(macos) missing nvm")
	}
	if !ids["gh"] {
		t.Error("AdaptersForPlatform(macos) missing gh")
	}
	if !ids["uv"] {
		t.Error("AdaptersForPlatform(macos) missing uv")
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

	// Windows should include: winget, scoop, nvm, npm, pnpm, bun, gh, docker, go, opencode, uv
	if !ids["winget"] {
		t.Error("AdaptersForPlatform(windows) missing winget")
	}
	if !ids["scoop"] {
		t.Error("AdaptersForPlatform(windows) missing scoop")
	}
	if !ids["nvm"] {
		t.Error("AdaptersForPlatform(windows) missing nvm")
	}
	if !ids["uv"] {
		t.Error("AdaptersForPlatform(windows) missing uv")
	}

	// Windows should NOT include: apt, brew
	if ids["apt"] {
		t.Error("AdaptersForPlatform(windows) should not contain apt")
	}
	if ids["brew"] {
		t.Error("AdaptersForPlatform(windows) should not contain brew")
	}
}

// TestResolveOwner is the golden per-platform ownership table (spec Tool
// Ownership Declaration): gh/docker resolve to apt on Linux, brew on macOS,
// winget on Windows; go resolves to brew on macOS and winget on Windows but
// has NO owner on Linux (standalone — manual binary replace); a manager
// adapter with no owner on a platform resolves to nil.
func TestResolveOwner(t *testing.T) {
	setExecFakes(t, execFakes{lookPath: map[string]bool{"apt": true}})
	tests := []struct {
		name    string
		tool    string
		os      string
		wantID  string
		wantNil bool
	}{
		{"gh-linux", "gh", "linux", "apt", false},
		{"gh-macos", "gh", "macos", "brew", false},
		{"gh-windows", "gh", "windows", "winget", false},
		{"docker-linux", "docker", "linux", "apt", false},
		{"docker-macos", "docker", "macos", "brew", false},
		{"docker-windows", "docker", "windows", "winget", false},
		{"go-macos", "go", "macos", "brew", false},
		{"go-windows", "go", "windows", "winget", false},
		{"go-linux-standalone", "go", "linux", "", true},
		{"npm-standalone", "npm", "linux", "", true},
		{"uv-linux-standalone", "uv", "linux", "", true},
		{"uv-macos-standalone", "uv", "macos", "", true},
		{"uv-windows-standalone", "uv", "windows", "", true},
		{"brew-manager-no-owner", "brew", "macos", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner := ResolveOwner(tt.tool, tt.os)
			if tt.wantNil {
				if owner != nil {
					t.Errorf("ResolveOwner(%q, %q) = %v, want nil", tt.tool, tt.os, owner)
				}
				return
			}
			if owner == nil {
				t.Fatalf("ResolveOwner(%q, %q) = nil, want %q", tt.tool, tt.os, tt.wantID)
			}
			if owner.Name() != tt.wantID {
				t.Errorf("ResolveOwner(%q, %q).Name() = %q, want %q", tt.tool, tt.os, owner.Name(), tt.wantID)
			}
		})
	}
}

// TestRuntimeGOOSToPlatform pins the WU1-documented gotcha translation:
// ResolveOwner is keyed by PLATFORM constants (linux/macos/windows), not
// runtime.GOOS (which returns "darwin" on macOS). A delegated Update() that
// calls ResolveOwner(id, plat) must translate darwin->macos, so macOS
// owner resolution succeeds on every host without needing to run on a Mac.
func TestRuntimeGOOSToPlatform(t *testing.T) {
	tests := []struct {
		goos string
		want string
	}{
		{"linux", platform.OSLinux},
		{"darwin", platform.OSMacOS},
		{"windows", platform.OSWindows},
	}
	for _, tt := range tests {
		t.Run(tt.goos, func(t *testing.T) {
			got, err := platform.NormalizeOS(tt.goos)
			if err != nil {
				t.Fatalf("platform.NormalizeOS(%q) unexpected error: %v", tt.goos, err)
			}
			if got != tt.want {
				t.Errorf("platform.NormalizeOS(%q) = %q, want %q", tt.goos, got, tt.want)
			}
		})
	}
}

// TestResolveOwnerViaRuntimeGOOSToPlatform proves the exact delegation
// resolution chain the gh/docker/go Update() methods use: translate
// runtime.GOOS to the platform key, then ResolveOwner. This closes the loop on
// the GOTCHA on any host — the darwin branch cannot run natively here, but the
// translation+resolution chain is fully exercised by supplying the translated
// key directly (the same value a real darwin runtime.GOOS would produce after
// platform.NormalizeOS).
func TestResolveOwnerViaRuntimeGOOSToPlatform(t *testing.T) {
	tests := []struct {
		name   string
		tool   string
		goos   string
		wantID string
	}{
		{"gh-darwin", "gh", "darwin", "brew"},
		{"docker-darwin", "docker", "darwin", "brew"},
		{"go-darwin", "go", "darwin", "brew"},
		{"go-linux", "go", "linux", ""}, // standalone, no owner
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, _ := platform.NormalizeOS(tt.goos)
			owner := ResolveOwner(tt.tool, key)
			if tt.wantID == "" {
				if owner != nil {
					t.Errorf("ResolveOwner(%q, %q) = %v, want nil (standalone)", tt.tool, key, owner)
				}
				return
			}
			if owner == nil {
				t.Fatalf("ResolveOwner(%q, %q) = nil, want %q", tt.tool, key, tt.wantID)
			}
			if owner.Name() != tt.wantID {
				t.Errorf("ResolveOwner(%q, %q).Name() = %q, want %q", tt.tool, key, owner.Name(), tt.wantID)
			}
		})
	}
}

// TestKindManagerConsistency pins that every KindManager adapter is the
// documented owner set (apt, brew, pacman, winget, scoop) and every other adapter is
// KindTool, so the manager/tool dichotomy cannot silently drift.
func TestKindManagerConsistency(t *testing.T) {
	managers := map[string]bool{"apt": true, "brew": true, "pacman": true, "winget": true, "scoop": true}
	for _, a := range AllAdapters() {
		id := a.Name()
		info := a.Info()
		if managers[id] {
			if info.Kind != adapters.KindManager {
				t.Errorf("adapter %q should be KindManager, got %d", id, info.Kind)
			}
		} else {
			if info.Kind != adapters.KindTool {
				t.Errorf("adapter %q should be KindTool, got %d", id, info.Kind)
			}
		}
	}
}

// TestManagerOwnedToolCardinality verifies a manager reports the tools it owns
// on each platform by scanning owner declarations (spec Manager Owned-Tool
// Cardinality), not a hardcoded per-platform set.
func TestManagerOwnedToolCardinality(t *testing.T) {
	setExecFakes(t, execFakes{lookPath: map[string]bool{"apt": true}})
	tests := []struct {
		name      string
		manager   string
		os        string
		wantOwned []string
	}{
		{"apt-linux", "apt", "linux", []string{"gh", "docker"}},
		{"brew-macos", "brew", "macos", []string{"gh", "docker", "go"}},
		{"winget-windows", "winget", "windows", []string{"gh", "docker", "go"}},
		{"pacman-linux", "pacman", "linux", []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var owned []string
			for _, a := range AllAdapters() {
				info := a.Info()
				if info.Kind != adapters.KindTool {
					continue
				}
				if info.Manager[tt.os] == tt.manager {
					owned = append(owned, a.Name())
				}
			}
			if len(owned) != len(tt.wantOwned) {
				t.Errorf("manager %q owns %v on %q, want %v", tt.manager, owned, tt.os, tt.wantOwned)
			}
			for _, want := range tt.wantOwned {
				found := false
				for _, got := range owned {
					if got == want {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("manager %q should own %q on %q (owns %v)", tt.manager, want, tt.os, owned)
				}
			}
		})
	}

	t.Run("pacman-linux-when-apt-absent", func(t *testing.T) {
		setExecFakes(t, execFakes{lookPath: map[string]bool{"apt": false, "pacman": true}})
		var owned []string
		for _, a := range AllAdapters() {
			info := a.Info()
			if info.Kind != adapters.KindTool {
				continue
			}
			if info.Manager["linux"] == "pacman" {
				owned = append(owned, a.Name())
			}
		}
		wantOwned := []string{"gh", "docker"}
		if len(owned) != len(wantOwned) {
			t.Errorf("pacman owns %v on linux when apt absent, want %v", owned, wantOwned)
		}
	})
}

// TestOwnerMetadata verifies the per-Kind count the registry reports: 5
// managers (apt, brew, pacman, winget, scoop) and 9 tools (nvm, npm, pnpm, bun, gh,
// docker, go, opencode, uv) out of 14 adapters. This pins the manager/tool split
// so a new adapter with a wrong Kind cannot silently change the totals.
func TestOwnerMetadata(t *testing.T) {
	meta := OwnerMetadata()
	if meta.Total != 14 {
		t.Errorf("OwnerMetadata().Total = %d, want 14", meta.Total)
	}
	if meta.Managers != 5 {
		t.Errorf("OwnerMetadata().Managers = %d, want 5", meta.Managers)
	}
	if meta.Tools != 9 {
		t.Errorf("OwnerMetadata().Tools = %d, want 9", meta.Tools)
	}
	if meta.Managers+meta.Tools != meta.Total {
		t.Errorf("OwnerMetadata() inconsistent: managers(%d) + tools(%d) != total(%d)", meta.Managers, meta.Tools, meta.Total)
	}
}

func TestAdapterByName(t *testing.T) {
	tests := []struct {
		name    string
		wantNil bool
	}{
		{"apt", false},
		{"brew", false},
		{"pacman", false},
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
		{"uv", false},
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

func TestIsOfficial(t *testing.T) {
	officialIDs := []string{
		"apt", "brew", "pacman", "winget", "scoop",
		"nvm", "npm", "pnpm", "bun", "uv",
		"gh", "docker", "go", "opencode",
	}

	if len(officialIDs) != 14 {
		t.Fatalf("expected 14 official IDs, got %d", len(officialIDs))
	}

	for _, id := range officialIDs {
		t.Run("official_"+id, func(t *testing.T) {
			if !IsOfficial(id) {
				t.Errorf("IsOfficial(%q) = false, want true", id)
			}
		})
	}

	unknownIDs := []string{
		"unknown", "custom-tool", "pip", "cargo", "", "APT", "Brew", "node",
	}
	for _, id := range unknownIDs {
		t.Run("unknown_"+id, func(t *testing.T) {
			if IsOfficial(id) {
				t.Errorf("IsOfficial(%q) = true, want false", id)
			}
		})
	}
}

func TestIsManager(t *testing.T) {
	managers := []string{"apt", "brew", "pacman", "winget", "scoop"}
	for _, id := range managers {
		t.Run("manager_"+id, func(t *testing.T) {
			if !IsManager(id) {
				t.Errorf("IsManager(%q) = false, want true", id)
			}
		})
	}

	nonManagers := []string{
		"nvm", "npm", "pnpm", "bun", "uv",
		"gh", "docker", "go", "opencode",
		"unknown", "custom-tool", "",
	}
	for _, id := range nonManagers {
		t.Run("non_manager_"+id, func(t *testing.T) {
			if IsManager(id) {
				t.Errorf("IsManager(%q) = true, want false", id)
			}
		})
	}
}

func TestPlatformsFor(t *testing.T) {
	tests := []struct {
		id       string
		wantPlat []string
	}{
		{"apt", []string{platform.OSLinux}},
		{"brew", []string{platform.OSLinux, platform.OSMacOS}},
		{"pacman", []string{platform.OSLinux}},
		{"winget", []string{platform.OSWindows}},
		{"scoop", []string{platform.OSWindows}},
		{"nvm", []string{platform.OSLinux, platform.OSMacOS, platform.OSWindows}},
		{"npm", []string{platform.OSLinux, platform.OSMacOS, platform.OSWindows}},
		{"pnpm", []string{platform.OSLinux, platform.OSMacOS, platform.OSWindows}},
		{"bun", []string{platform.OSLinux, platform.OSMacOS, platform.OSWindows}},
		{"uv", []string{platform.OSLinux, platform.OSMacOS, platform.OSWindows}},
		{"gh", []string{platform.OSLinux, platform.OSMacOS, platform.OSWindows}},
		{"docker", []string{platform.OSLinux, platform.OSMacOS, platform.OSWindows}},
		{"go", []string{platform.OSLinux, platform.OSMacOS, platform.OSWindows}},
		{"opencode", []string{platform.OSLinux, platform.OSMacOS, platform.OSWindows}},
		{"unknown", nil},
		{"custom-tool", nil},
		{"", nil},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			got := PlatformsFor(tt.id)
			if tt.wantPlat == nil {
				if got != nil {
					t.Errorf("PlatformsFor(%q) = %v, want nil", tt.id, got)
				}
				return
			}
			if len(got) != len(tt.wantPlat) {
				t.Fatalf("PlatformsFor(%q) returned %d platforms %v, want %d %v", tt.id, len(got), got, len(tt.wantPlat), tt.wantPlat)
			}
			for i, p := range tt.wantPlat {
				if got[i] != p {
					t.Errorf("PlatformsFor(%q)[%d] = %q, want %q", tt.id, i, got[i], p)
				}
			}
		})
	}
}
