package official

import (
	"testing"
)

func FuzzParseAptPolicyOutput(f *testing.F) {
	seeds := []string{
		"Installed: 2.34.1\nCandidate: 2.35.0\n",
		"Installed: (none)\nCandidate: 1.0\n",
		"Installed: 1.0\nCandidate: \n",
		"",
		"Installed:\nCandidate:\n",
		"Random non-apt output without keywords",
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		current, latest := parseAptPolicyOutput(input)
		if current == "" {
			t.Fatalf("expected non-empty fallback for current version, got %q", current)
		}
		if latest == "" {
			t.Fatalf("expected non-empty fallback for latest version, got %q", latest)
		}
	})
}

func FuzzParsePacmanQOutput(f *testing.F) {
	seeds := []string{
		"pacman 6.0.2-1\n",
		"github-cli 2.45.0-1",
		"onlyonefield",
		"",
		"   spaces   before and after   ",
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		got := parsePacmanQOutput(input)
		if got == "" {
			t.Fatalf("expected non-empty version or fallback, got %q", got)
		}
	})
}

func FuzzParsePacmanSiOutput(f *testing.F) {
	seeds := []string{
		"Repository      : extra\nName            : pacman\nVersion         : 6.0.2-1\nDescription     : A library-based package manager",
		"Version:1.0",
		"Version : ",
		"",
		"No version line at all\nJust text",
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		got := parsePacmanSiOutput(input)
		if got == "" {
			t.Fatalf("expected non-empty version or fallback, got %q", got)
		}
	})
}

func FuzzParseBrewOutdatedJSON(f *testing.F) {
	seeds := []string{
		`[{"name":"gh","installed_versions":["2.40.0"],"current_version":"2.45.0"}]`,
		"[]",
		"{invalid json",
		"",
		`[{"name":"gh"}]`,
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		curr, lat, found := parseBrewOutdatedJSON(input)
		_ = curr
		if found && lat == "" {
			t.Fatalf("invariant violated: found is true but latest is empty")
		}
	})
}

func FuzzParseWingetUpgradeOutput(f *testing.F) {
	seeds := []string{
		"Name Id Version Available Source\n------------------------------------\nwinget Microsoft.DesktopAppInstaller 1.2.3 1.2.4 winget",
		"No upgrades available",
		"",
		"Header only\n----------------",
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		curr, lat, ok := parseWingetUpgradeOutput(input)
		_ = curr
		_ = lat
		_ = ok
	})
}

func FuzzParseWingetPackageUpgradeOutput(f *testing.F) {
	seeds := []struct {
		out string
		pkg string
	}{
		{"Name Id Version Available Source\n------------------------------------\ngh GitHub.cli 2.40.0 2.45.0 winget", "GitHub.cli"},
		{"Name Id Version Available Source\n------------------------------------\ngh GitHub.cli 2.40.0 2.45.0 winget", "Docker.Docker"},
		{"", "gh"},
		{"corrupted table", ""},
	}
	for _, seed := range seeds {
		f.Add(seed.out, seed.pkg)
	}

	f.Fuzz(func(t *testing.T, out, pkg string) {
		curr, lat, ok := parseWingetPackageUpgradeOutput(out, pkg)
		_ = curr
		_ = lat
		_ = ok
	})
}

func FuzzExtractVersionFromString(f *testing.F) {
	seeds := []string{
		"v1.2.3",
		"1.2.3",
		"git version 2.43.0",
		"upp 0.1.0-dev",
		"no numbers at all",
		"",
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		got := extractVersionFromString(input)
		_ = got
	})
}

func FuzzParseVersionFromHeader(f *testing.F) {
	seeds := []string{
		"ToolName v3.2.1 (c) 2026\nSecond line",
		"v1.0",
		"",
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		got := parseVersionFromHeader(input)
		_ = got
	})
}
