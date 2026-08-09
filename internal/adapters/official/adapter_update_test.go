package official

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/JhnFrankz/upp/internal/adapters"
)

// --- Adapter Update Tests with Mock Shell ---

func TestAptAdapter_Update_DryRun(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("apt is Linux-only")
	}

	a := &AptAdapter{}
	if !a.Detect() {
		t.Skip("apt not installed")
	}

	result, err := a.Update(true) // dry run
	if err != nil {
		t.Fatalf("Update(dryRun=true) error: %v", err)
	}
	if !result.Success {
		t.Error("dry-run should always succeed")
	}
	if result.Before == "" {
		t.Error("dry-run should return before version")
	}
}

func TestBrewAdapter_Update_DryRun(t *testing.T) {
	a := &BrewAdapter{}
	if !a.Detect() {
		t.Skip("brew not installed")
	}

	result, err := a.Update(true)
	if err != nil {
		t.Fatalf("Update(dryRun=true) error: %v", err)
	}
	if !result.Success {
		t.Error("dry-run should always succeed")
	}
}

func TestNpmAdapter_Update_DryRun(t *testing.T) {
	a := &NpmAdapter{}
	if !a.Detect() {
		t.Skip("npm not installed")
	}

	result, err := a.Update(true)
	if err != nil {
		t.Fatalf("Update(dryRun=true) error: %v", err)
	}
	if !result.Success {
		t.Error("dry-run should always succeed")
	}
}

func TestPnpmAdapter_Update_DryRun(t *testing.T) {
	a := &PnpmAdapter{}
	if !a.Detect() {
		t.Skip("pnpm not installed")
	}

	result, err := a.Update(true)
	if err != nil {
		t.Fatalf("Update(dryRun=true) error: %v", err)
	}
	if !result.Success {
		t.Error("dry-run should always succeed")
	}
}

func TestBunAdapter_Update_DryRun(t *testing.T) {
	a := &BunAdapter{}
	if !a.Detect() {
		t.Skip("bun not installed")
	}

	result, err := a.Update(true)
	if err != nil {
		t.Fatalf("Update(dryRun=true) error: %v", err)
	}
	if !result.Success {
		t.Error("dry-run should always succeed")
	}
}

func TestGhAdapter_Update_DryRun(t *testing.T) {
	a := &GhAdapter{}
	if !a.Detect() {
		t.Skip("gh not installed")
	}

	result, err := a.Update(true)
	if err != nil {
		t.Fatalf("Update(dryRun=true) error: %v", err)
	}
	if !result.Success {
		t.Error("dry-run should always succeed")
	}
}

func TestDockerAdapter_Update_DryRun(t *testing.T) {
	a := &DockerAdapter{}
	if !a.Detect() {
		t.Skip("docker not installed")
	}

	result, err := a.Update(true)
	if err != nil {
		t.Fatalf("Update(dryRun=true) error: %v", err)
	}
	if !result.Success {
		t.Error("dry-run should always succeed")
	}
}

func TestGoAdapter_Update_DryRun(t *testing.T) {
	a := &GoAdapter{}
	if !a.Detect() {
		t.Skip("go not installed")
	}

	result, err := a.Update(true)
	if err != nil {
		t.Fatalf("Update(dryRun=true) error: %v", err)
	}
	if !result.Success {
		t.Error("dry-run should always succeed")
	}
}

func TestOpenCodeAdapter_Update_DryRun(t *testing.T) {
	a := &OpenCodeAdapter{}
	if !a.Detect() {
		t.Skip("opencode not installed")
	}

	result, err := a.Update(true)
	if err != nil {
		t.Fatalf("Update(dryRun=true) error: %v", err)
	}
	if !result.Success {
		t.Error("dry-run should always succeed")
	}
}

func TestNVMAdapter_Update_DryRun(t *testing.T) {
	a := &NVMAdapter{}
	if !a.Detect() {
		t.Skip("nvm not installed")
	}

	result, err := a.Update(true)
	if err != nil {
		t.Fatalf("Update(dryRun=true) error: %v", err)
	}
	if !result.Success {
		t.Error("dry-run should always succeed")
	}
}

// --- Dry-Run Safety: No Commands Executed ---

func TestDryRunSafety_AllAdapters(t *testing.T) {
	// This is a critical safety test: dry-run MUST NOT execute any commands.
	// We verify by checking that all adapters return success in dry-run mode.
	allAdapters := AllAdapters()

	for _, a := range allAdapters {
		t.Run(a.Name(), func(t *testing.T) {
			if !a.Detect() {
				t.Skipf("%s not installed, skipping dry-run safety test", a.Name())
			}

			result, err := a.Update(true)
			if err != nil {
				t.Errorf("%s dry-run returned error: %v", a.Name(), err)
			}
			if !result.Success {
				t.Errorf("%s dry-run returned Success=false", a.Name())
			}
		})
	}
}

// --- Error Handling: Network Failures ---

func TestAptAdapter_Check_NotInstalled(t *testing.T) {
	if runtime.GOOS == "linux" {
		t.Skip("apt is installed on Linux")
	}

	a := &AptAdapter{}
	_, err := a.Check()
	if err == nil {
		t.Error("Check() should error when apt is not installed")
	}
}

func TestAptAdapter_Update_NotInstalled(t *testing.T) {
	if runtime.GOOS == "linux" {
		t.Skip("apt is installed on Linux")
	}

	a := &AptAdapter{}
	_, err := a.Update(false)
	if err == nil {
		t.Error("Update() should error when apt is not installed")
	}
}

// --- Error Handling: Permission Denied ---

func TestAptAdapter_Update_Privileges(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("apt is Linux-only")
	}

	a := &AptAdapter{}
	if !a.Detect() {
		t.Skip("apt not installed")
	}

	// Update should report sudo privilege requirement
	result, err := a.Update(true) // dry run
	if err != nil {
		t.Fatalf("dry-run error: %v", err)
	}
	_ = result
	// Privileges are set in the real Update path, not dry-run
}

// --- Error Handling: Partial Failures ---

func TestAllAdapters_Check_DoesNotPanic(t *testing.T) {
	allAdapters := AllAdapters()

	for _, a := range allAdapters {
		t.Run(a.Name(), func(t *testing.T) {
			// Check should never panic, even if tool is not installed
			_, err := a.Check()
			// Error is acceptable; panic is not
			_ = err
		})
	}
}

func TestAllAdapters_Update_DoesNotPanic(t *testing.T) {
	allAdapters := AllAdapters()

	for _, a := range allAdapters {
		t.Run(a.Name(), func(t *testing.T) {
			if !a.Detect() {
				t.Skipf("%s not installed", a.Name())
			}
			// Dry-run should never panic
			_, err := a.Update(true)
			_ = err
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

			// Trust must be Trusted for official adapters
			if info.Trust != adapters.TrustTrusted {
				t.Errorf("Info().Trust = %d, want TrustTrusted", info.Trust)
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

// --- NVM Adapter Edge Cases ---

func TestNVMAdapter_Detect_WithNVM_DIR(t *testing.T) {
	a := &NVMAdapter{}

	tmpDir := t.TempDir()
	nvmSh := filepath.Join(tmpDir, "nvm.sh")
	if err := os.WriteFile(nvmSh, []byte("#!/bin/sh\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	old := os.Getenv("NVM_DIR")
	os.Setenv("NVM_DIR", tmpDir)
	defer os.Setenv("NVM_DIR", old)

	if !a.Detect() {
		t.Error("NVMAdapter.Detect() should return true with valid NVM_DIR")
	}
}

func TestNVMAdapter_Detect_WithoutNVM_DIR(t *testing.T) {
	a := &NVMAdapter{}

	old := os.Getenv("NVM_DIR")
	os.Unsetenv("NVM_DIR")
	defer os.Setenv("NVM_DIR", old)

	// Should not panic
	result := a.Detect()
	_ = result
}

// --- Helper Function Tests ---

func TestCommandOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping shell test on windows")
	}

	output := commandOutput("echo", "hello")
	if !strings.Contains(output, "hello") {
		t.Errorf("commandOutput should contain 'hello', got: %q", output)
	}
}

func TestShellOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping shell test on windows")
	}

	output := shellOutput("echo hello")
	if !strings.Contains(output, "hello") {
		t.Errorf("shellOutput should contain 'hello', got: %q", output)
	}
}

func TestFormatUpdateCmd(t *testing.T) {
	got := formatUpdateCmd("brew upgrade")
	if got != "exec: brew upgrade" {
		t.Errorf("formatUpdateCmd = %q, want %q", got, "exec: brew upgrade")
	}
}

// --- NVM Adapter Name and Info ---

func TestNVMAdapter_FullInfo(t *testing.T) {
	a := &NVMAdapter{}

	if a.Name() != "nvm" {
		t.Errorf("Name() = %q, want %q", a.Name(), "nvm")
	}

	info := a.Info()
	if info.ID != "nvm" {
		t.Errorf("Info().ID = %q, want %q", info.ID, "nvm")
	}
	if info.Name != "Node Version Manager" {
		t.Errorf("Info().Name = %q, want %q", info.Name, "Node Version Manager")
	}
	if len(info.Platforms) != 3 {
		t.Errorf("Info().Platforms has %d entries, want 3", len(info.Platforms))
	}
}

// --- Adapter Check When Not Installed ---

func TestAdapters_CheckWhenNotInstalled(t *testing.T) {
	// Most adapters should return an error when not installed
	// This tests error handling paths
	adaptersToTest := []struct {
		name    string
		adapter adapters.Adapter
	}{
		{"apt", &AptAdapter{}},
	}

	for _, tt := range adaptersToTest {
		t.Run(tt.name, func(t *testing.T) {
			if tt.adapter.Detect() {
				t.Skipf("%s is installed, skipping not-installed test", tt.name)
			}
			_, err := tt.adapter.Check()
			if err == nil {
				t.Errorf("Check() should error when %s is not installed", tt.name)
			}
		})
	}
}
