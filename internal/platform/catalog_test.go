package platform

import (
	"testing"
)

func TestCatalogForLinux(t *testing.T) {
	catalog := CatalogFor(OSLinux)

	// Linux should have apt, brew, nvm, npm, pnpm, bun, gh, docker, go, opencode.
	// winget and scoop should NOT be present.
	ids := make(map[string]bool)
	for _, tool := range catalog {
		ids[tool.ID] = true
	}

	expectedPresent := []string{"apt", "brew", "nvm", "npm", "pnpm", "bun", "gh", "docker", "go", "opencode"}
	for _, id := range expectedPresent {
		if !ids[id] {
			t.Errorf("Linux catalog missing expected tool: %s", id)
		}
	}

	expectedAbsent := []string{"winget", "scoop"}
	for _, id := range expectedAbsent {
		if ids[id] {
			t.Errorf("Linux catalog should not contain: %s", id)
		}
	}
}

func TestCatalogForMacOS(t *testing.T) {
	catalog := CatalogFor(OSMacOS)

	ids := make(map[string]bool)
	for _, tool := range catalog {
		ids[tool.ID] = true
	}

	// apt should NOT be on macOS.
	if ids["apt"] {
		t.Error("macOS catalog should not contain apt")
	}

	// brew should be on macOS.
	if !ids["brew"] {
		t.Error("macOS catalog missing brew")
	}

	// winget/scoop should NOT be on macOS.
	if ids["winget"] {
		t.Error("macOS catalog should not contain winget")
	}
	if ids["scoop"] {
		t.Error("macOS catalog should not contain scoop")
	}
}

func TestCatalogForWindows(t *testing.T) {
	catalog := CatalogFor(OSWindows)

	ids := make(map[string]bool)
	for _, tool := range catalog {
		ids[tool.ID] = true
	}

	// Windows should have winget, scoop, nvm, npm, pnpm, bun, gh, docker, go, opencode.
	// apt and brew should NOT be present.
	if !ids["winget"] {
		t.Error("Windows catalog missing winget")
	}
	if !ids["scoop"] {
		t.Error("Windows catalog missing scoop")
	}
	if ids["apt"] {
		t.Error("Windows catalog should not contain apt")
	}
	if ids["brew"] {
		t.Error("Windows catalog should not contain brew")
	}
}

func TestIsOfficialTool(t *testing.T) {
	if !IsOfficialTool("apt") {
		t.Error("apt should be official")
	}
	if !IsOfficialTool("opencode") {
		t.Error("opencode should be official")
	}
	if IsOfficialTool("mycustomtool") {
		t.Error("mycustomtool should not be official")
	}
}

func TestFilterByPlatform(t *testing.T) {
	linuxTools := FilterByPlatform(OSLinux)
	for _, id := range linuxTools {
		if id == "winget" || id == "scoop" {
			t.Errorf("FilterByPlatform(linux) should not include %s", id)
		}
	}
}
