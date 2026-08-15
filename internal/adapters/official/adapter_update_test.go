package official

import (
	"errors"
	"runtime"
	"strings"
	"testing"

	"github.com/JhnFrankz/upp/internal/adapters"
)

// --- Hermetic wrapper tests ---
//
// commandOutput and shellOutput stay real (D1) but are driven through the
// seam leaf fakes, so no real subprocess runs. The previous versions of these
// tests executed real commands.

func TestCommandOutput(t *testing.T) {
	tests := []struct {
		name  string
		fakes execFakes
		call  func() string
		want  string
	}{
		{
			name:  "success-trimmed",
			fakes: execFakes{cmdArgs: map[string]fakeResult{"echo": {stdout: "hello\n"}}},
			call:  func() string { return commandOutput("echo", "hello") },
			want:  "hello",
		},
		{
			name:  "command-error-empty",
			fakes: execFakes{cmdArgs: map[string]fakeResult{"echo": {err: errors.New("boom")}}},
			call:  func() string { return commandOutput("echo", "hello") },
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setExecFakes(t, tt.fakes)
			if got := tt.call(); got != tt.want {
				t.Errorf("commandOutput() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestShellOutput(t *testing.T) {
	tests := []struct {
		name  string
		fakes execFakes
		call  func() string
		want  string
	}{
		{
			name:  "success-trimmed",
			fakes: execFakes{shell: map[string]fakeResult{"echo hello": {stdout: "hello\n"}}},
			call:  func() string { return shellOutput("echo hello") },
			want:  "hello",
		},
		{
			name:  "command-error-empty",
			fakes: execFakes{shell: map[string]fakeResult{"echo hello": {err: errors.New("boom")}}},
			call:  func() string { return shellOutput("echo hello") },
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setExecFakes(t, tt.fakes)
			if got := tt.call(); got != tt.want {
				t.Errorf("shellOutput() = %q, want %q", got, tt.want)
			}
		})
	}
}

// --- Platform-Specific Adapter Tests ---

func TestPlatformSpecificAdapters(t *testing.T) {
	tests := []struct {
		platform string
		expected []string
		excluded []string
	}{
		{
			platform: "linux",
			expected: []string{"apt", "brew", "nvm", "npm", "pnpm", "bun", "gh", "docker", "go", "opencode"},
			excluded: []string{"winget", "scoop"},
		},
		{
			platform: "macos",
			expected: []string{"brew", "nvm", "npm", "pnpm", "bun", "gh", "docker", "go", "opencode"},
			excluded: []string{"apt", "winget", "scoop"},
		},
		{
			platform: "windows",
			expected: []string{"winget", "scoop", "nvm", "npm", "pnpm", "bun", "gh", "docker", "go", "opencode"},
			excluded: []string{"apt", "brew"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.platform, func(t *testing.T) {
			result := AdaptersForPlatform(tt.platform)

			ids := make(map[string]bool)
			for _, a := range result {
				ids[a.Name()] = true
			}

			for _, id := range tt.expected {
				if !ids[id] {
					t.Errorf("AdaptersForPlatform(%s) missing: %s", tt.platform, id)
				}
			}
			for _, id := range tt.excluded {
				if ids[id] {
					t.Errorf("AdaptersForPlatform(%s) should not contain: %s", tt.platform, id)
				}
			}
		})
	}
}

// --- Info Consistency ---

func TestAllAdapters_InfoConsistency(t *testing.T) {
	allAdapters := AllAdapters()

	for _, a := range allAdapters {
		t.Run(a.Name(), func(t *testing.T) {
			info := a.Info()

			// ID must match Name()
			if info.ID != a.Name() {
				t.Errorf("Info().ID = %q, Name() = %q", info.ID, a.Name())
			}

			// Name must be non-empty
			if info.Name == "" {
				t.Error("Info().Name is empty")
			}

			// Platforms must be non-empty
			if len(info.Platforms) == 0 {
				t.Error("Info().Platforms is empty")
			}

			// Trust must be TrustOfficial for official adapters
			if info.Trust != adapters.TrustOfficial {
				t.Errorf("Info().Trust = %d, want TrustOfficial", info.Trust)
			}

			// UpdatePolicy must be one of the declared values (design D6,
			// spec Update Gating). The zero value is PolicyGated (fail
			// closed), so a missed AlwaysUpdate site is caught by the
			// TestInfo goldens; this assertion rejects any value outside
			// the declared set.
			if info.UpdatePolicy != adapters.PolicyGated && info.UpdatePolicy != adapters.PolicyAlwaysUpdate {
				t.Errorf("Info().UpdatePolicy = %d, want PolicyGated or PolicyAlwaysUpdate", info.UpdatePolicy)
			}
		})
	}
}

// --- Version Parsing Edge Cases ---

func TestExtractVersion_EdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"empty string", "", ""},
		{"no version", "command not found", ""},
		{"semver", "1.2.3", "1.2.3"},
		{"v-prefix", "v1.2.3", "v1.2.3"},
		{"go version", "go version go1.22.0 linux/amd64", "go1.22.0"},
		{"node version", "v20.11.0", "v20.11.0"},
		{"brew version", "Homebrew 4.1.0", "4.1.0"},
		{"pnpm version", "pnpm: 8.14.0", "8.14.0"},
		{"multiline", "first line\n1.2.3\nthird line", "1.2.3"},
		{"pre-release", "1.2.3-rc1", "1.2.3-rc1"},
		{"build metadata", "1.2.3+build.123", "1.2.3+build.123"},
		{"four-part", "1.2.3.4", "1.2.3.4"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractVersion(tt.input)
			if got != tt.want {
				t.Errorf("extractVersion(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsVersionLike_EdgeCases(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"", false},
		{"1", false},           // no dot
		{"1.2", true},          // two-part
		{"1.2.3", true},        // standard
		{"1.2.3.4", true},      // four-part
		{"v1.2.3", true},       // prefix
		{"go1.22.0", true},     // go prefix
		{"abc", false},         // no digits
		{"abc.def", false},     // no digits
		{"1.2.3-rc1", true},    // pre-release
		{"1.2.3-beta.1", true}, // pre-release
		{"1.2.3+build", true},  // build metadata
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

// --- Truncate Edge Cases ---

func TestTruncate_EdgeCases(t *testing.T) {
	tests := []struct {
		input  string
		maxLen int
		want   string
	}{
		{"", 5, ""},
		{"a", 1, "a"},
		{"ab", 1, "a..."},
		{"abc", 3, "abc"},
		{"abcd", 3, "abc..."},
		{"hello world", 5, "hello..."},
		{"", 0, ""},
	}

	for _, tt := range tests {
		t.Run(tt.input+string(rune(tt.maxLen)), func(t *testing.T) {
			got := truncate(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
			}
		})
	}
}

// --- LookPath Tests ---

func TestLookPath_Exists(t *testing.T) {
	if !lookPath("go") {
		t.Error("go should be on PATH")
	}
}

func TestLookPath_NotExists(t *testing.T) {
	if lookPath("nonexistent-tool-xyz-12345") {
		t.Error("nonexistent tool should not be on PATH")
	}
}

// --- RunCmd Tests ---
//
// These exercise the real leaf implementations through the seam delegates
// (no fakes installed), the only tests in the package that run real
// subprocesses — all deterministic (echo / exit 1), no env-dependent tools.

func TestRunCmd_Success(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping shell test on windows")
	}

	stdout, stderr, err := runCmd("echo hello")
	if err != nil {
		t.Fatalf("runCmd error: %v", err)
	}
	if !strings.Contains(stdout, "hello") {
		t.Errorf("stdout should contain 'hello', got: %q", stdout)
	}
	_ = stderr
}

func TestRunCmd_Failure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping shell test on windows")
	}

	_, _, err := runCmd("exit 1")
	if err == nil {
		t.Error("runCmd with 'exit 1' should return error")
	}
}

func TestRunCmdArgs_Success(t *testing.T) {
	stdout, _, err := runCmdArgs("echo", "hello")
	if err != nil {
		t.Fatalf("runCmdArgs error: %v", err)
	}
	if !strings.Contains(stdout, "hello") {
		t.Errorf("stdout should contain 'hello', got: %q", stdout)
	}
}

// --- ExtractGoVersion Tests ---

func TestExtractGoVersion_EdgeCases(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"go version go1.22.0 linux/amd64", "1.22.0"},
		{"go version go1.21.5 darwin/arm64", "1.21.5"},
		{"go version go1.20.10 windows/amd64", "1.20.10"},
		{"go version go1.19.0", "1.19.0"},
		{"", ""},
		{"not go version", ""},
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

// --- Helper Function Tests ---

func TestFormatUpdateCmd(t *testing.T) {
	got := formatUpdateCmd("brew upgrade")
	if got != "exec: brew upgrade" {
		t.Errorf("formatUpdateCmd = %q, want %q", got, "exec: brew upgrade")
	}
}
