package official

import (
	"testing"
	"unicode/utf8"
)

func FuzzParseAptPolicyOutput(f *testing.F) {
	seeds := []string{
		"Installed: 2.34.1\nCandidate: 2.35.0\n",
		"Installed: (none)\nCandidate: 1.0\n",
		"Installed: 1.0\nCandidate: (none)\n",
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
		`{"formulae":[{"name":"gh","installed_versions":["2.40.0"],"current_version":"2.45.0"}],"casks":[]}`,
		`{"formulae":[],"casks":[{"name":"docker","installed_versions":["4.28.0"],"current_version":"4.29.0"}]}`,
		`{"formulae":[]}`,
		`null`,
		`{}`,
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
		{"Notice: Checking for updates\nGitHub.cli 2.40.0 2.45.0 winget", "GitHub.cli"},
		{"Notice: Checking for updates to GitHub.cli\nGitHub CLI GitHub.cli 2.40.0 2.45.0 winget", "github.cli"},
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

func FuzzParseScoopStatusOutput(f *testing.F) {
	seeds := []string{
		"WARN Scoop is out of date.\nName       Installed  Latest\n----       ---------  ------\nscoop      1.0.0      1.2.0\n",
		"Everything is ok\n",
		"WARN Scoop is out of date.\n",
		"",
		"scoop v0.22.0 v0.23.0\n",
		"scoop notversion notversion\n",
		"Random table without scoop row\ngit 2.43.0 2.44.0\n",
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		cur, lat, found := parseScoopStatusOutput(input)
		if found && (cur == "" || lat == "") {
			t.Fatalf("invariant violated: found is true but cur=%q or lat=%q is empty", cur, lat)
		}
	})
}

func FuzzExtractUvLatestVersion(f *testing.F) {
	seeds := []struct {
		output  string
		current string
	}{
		{"upgrading uv from 0.3.0 to 0.4.0\n", "0.3.0"},
		{"uv 0.4.0 (installed: 0.3.0)", "0.3.0"},
		{"everything up to date", "0.3.0"},
		{"upgrading to (0.4.0): done", "0.3.0"},
		{"", ""},
		{"corrupted text", "1.0.0"},
	}
	for _, seed := range seeds {
		f.Add(seed.output, seed.current)
	}

	f.Fuzz(func(t *testing.T, output, current string) {
		got := extractUvLatestVersion(output, current)
		if current != "" && got == "" {
			t.Fatalf("expected non-empty fallback, got empty string")
		}
	})
}

func FuzzParseUvSelfUpdateOutput(f *testing.F) {
	seeds := []string{
		"Everything up to date\n",
		"would update to 0.4.0\n",
		"new version available\n",
		"updating uv...\n",
		"",
		"random text\n",
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		_ = parseUvSelfUpdateOutput(input)
	})
}

func FuzzParseUvToolListOutdatedOutput(f *testing.F) {
	seeds := []string{
		"no tools installed\n",
		"no outdated tools\n",
		"tool-a 1.0.0 (latest 1.1.0)\n",
		"",
		"random text\n",
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		_ = parseUvToolListOutdatedOutput(input)
	})
}

func FuzzExtractGoVersion(f *testing.F) {
	seeds := []string{
		"go version go1.22.0 linux/amd64",
		"go version go1.21.5 darwin/arm64",
		"go version devel go1.22-abc1234 linux/amd64",
		"go version (go1.22.0)",
		"go version",
		"golang version 1.22.0",
		"good morning",
		"",
		"go run main.go",
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		_ = extractGoVersion(input)
	})
}

func FuzzTruncate(f *testing.F) {
	seeds := []struct {
		s      string
		maxLen int
	}{
		{"hello world", 5},
		{"🚀🔥✨ tools", 2},
		{"¡Hola Mundo!", 1},
		{"", 10},
		{"hello", 0},
		{"hello", -5},
		{"abc", 1000},
	}
	for _, seed := range seeds {
		f.Add(seed.s, seed.maxLen)
	}

	f.Fuzz(func(t *testing.T, s string, maxLen int) {
		got := truncate(s, maxLen)
		if !utf8.ValidString(got) {
			t.Fatalf("truncate(%q, %d) produced invalid UTF-8: %q", s, maxLen, got)
		}
	})
}
