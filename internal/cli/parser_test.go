package cli

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func TestParseFilter_Only(t *testing.T) {
	onlyList, skipList := ParseFilter("brew,npm", "")

	if len(onlyList) != 2 {
		t.Fatalf("expected 2 only items, got %d", len(onlyList))
	}
	if onlyList[0] != "brew" || onlyList[1] != "npm" {
		t.Errorf("unexpected only list: %v", onlyList)
	}
	if len(skipList) != 0 {
		t.Errorf("expected empty skip list when --only is set, got %v", skipList)
	}
}

func TestParseFilter_Skip(t *testing.T) {
	onlyList, skipList := ParseFilter("", "apt,docker")

	if len(onlyList) != 0 {
		t.Errorf("expected empty only list, got %v", onlyList)
	}
	if len(skipList) != 2 {
		t.Fatalf("expected 2 skip items, got %d", len(skipList))
	}
	if skipList[0] != "apt" || skipList[1] != "docker" {
		t.Errorf("unexpected skip list: %v", skipList)
	}
}

func TestParseFilter_OnlyWinsOverSkip(t *testing.T) {
	onlyList, skipList := ParseFilter("brew", "apt")

	if len(onlyList) != 1 {
		t.Fatalf("expected 1 only item, got %d", len(onlyList))
	}
	if onlyList[0] != "brew" {
		t.Errorf("expected 'brew', got %q", onlyList[0])
	}
	// --only wins: --skip is ignored
	if len(skipList) != 0 {
		t.Errorf("expected empty skip list when --only wins, got %v", skipList)
	}
}

func TestParseFilter_CaseInsensitive(t *testing.T) {
	onlyList, _ := ParseFilter("Brew,NPM", "")

	if onlyList[0] != "Brew" {
		t.Errorf("ParseFilter should preserve case, got %q", onlyList[0])
	}
	// Case insensitivity is handled in FilterTools, not ParseFilter
}

func TestParseFilter_Empty(t *testing.T) {
	onlyList, skipList := ParseFilter("", "")

	if len(onlyList) != 0 {
		t.Errorf("expected empty only list, got %v", onlyList)
	}
	if len(skipList) != 0 {
		t.Errorf("expected empty skip list, got %v", skipList)
	}
}

func TestParseFilter_TrimsSpaces(t *testing.T) {
	onlyList, _ := ParseFilter(" brew , npm ", "")

	if len(onlyList) != 2 {
		t.Fatalf("expected 2 items, got %d", len(onlyList))
	}
	if onlyList[0] != "brew" {
		t.Errorf("expected 'brew', got %q", onlyList[0])
	}
	if onlyList[1] != "npm" {
		t.Errorf("expected 'npm', got %q", onlyList[1])
	}
}

func TestFilterTools_Only(t *testing.T) {
	tools := []string{"apt", "brew", "npm", "docker"}
	onlyList := []string{"brew", "npm"}
	var stderr bytes.Buffer

	result := FilterTools(tools, onlyList, nil, &stderr)

	if len(result) != 2 {
		t.Fatalf("expected 2 filtered tools, got %d", len(result))
	}
	if result[0] != "brew" || result[1] != "npm" {
		t.Errorf("unexpected result: %v", result)
	}
}

func TestFilterTools_Skip(t *testing.T) {
	tools := []string{"apt", "brew", "npm", "docker"}
	skipList := []string{"apt", "docker"}
	var stderr bytes.Buffer

	result := FilterTools(tools, nil, skipList, &stderr)

	if len(result) != 2 {
		t.Fatalf("expected 2 filtered tools, got %d", len(result))
	}
	if result[0] != "brew" || result[1] != "npm" {
		t.Errorf("unexpected result: %v", result)
	}
}

func TestFilterTools_CaseInsensitive(t *testing.T) {
	tools := []string{"apt", "brew", "npm", "docker"}
	onlyList := []string{"Brew", "NPM"}
	var stderr bytes.Buffer

	result := FilterTools(tools, onlyList, nil, &stderr)

	if len(result) != 2 {
		t.Fatalf("expected 2 filtered tools, got %d", len(result))
	}
	if result[0] != "brew" || result[1] != "npm" {
		t.Errorf("unexpected result: %v", result)
	}
}

func TestFilterTools_UnknownToolWarning(t *testing.T) {
	tools := []string{"apt", "brew", "npm"}
	onlyList := []string{"brew", "nonexistent"}
	var stderr bytes.Buffer

	result := FilterTools(tools, onlyList, nil, &stderr)

	// Warning should be written to stderr
	warning := stderr.String()
	if !strings.Contains(warning, "nonexistent") {
		t.Errorf("expected warning about 'nonexistent', got: %q", warning)
	}

	// Only brew should be in result
	if len(result) != 1 || result[0] != "brew" {
		t.Errorf("expected [brew], got %v", result)
	}
}

func TestFilterTools_SkipUnknownWarning(t *testing.T) {
	tools := []string{"apt", "brew", "npm"}
	skipList := []string{"brew", "ghost"}
	var stderr bytes.Buffer

	result := FilterTools(tools, nil, skipList, &stderr)

	warning := stderr.String()
	if !strings.Contains(warning, "ghost") {
		t.Errorf("expected warning about 'ghost', got: %q", warning)
	}

	// apt and npm should remain
	if len(result) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(result))
	}
}

func TestFilterTools_NoFilter(t *testing.T) {
	tools := []string{"apt", "brew", "npm"}
	var stderr bytes.Buffer

	result := FilterTools(tools, nil, nil, &stderr)

	if len(result) != 3 {
		t.Fatalf("expected 3 tools (no filter), got %d", len(result))
	}
}

func TestBuildRoot_Flags(t *testing.T) {
	root, gf := BuildRoot()

	if root.Use != "upp" {
		t.Errorf("expected Use='upp', got %q", root.Use)
	}

	// Check flags exist
	quietFlag := root.PersistentFlags().Lookup("quiet")
	if quietFlag == nil {
		t.Error("missing --quiet flag")
	}
	if gf.Quiet {
		t.Error("--quiet should default to false")
	}

	ciFlag := root.PersistentFlags().Lookup("ci")
	if ciFlag == nil {
		t.Error("missing --ci flag")
	}
	if gf.CI {
		t.Error("--ci should default to false")
	}

	onlyFlag := root.PersistentFlags().Lookup("only")
	if onlyFlag == nil {
		t.Error("missing --only flag")
	}

	skipFlag := root.PersistentFlags().Lookup("skip")
	if skipFlag == nil {
		t.Error("missing --skip flag")
	}
}

func TestBuildRoot_FlagShorthands(t *testing.T) {
	root, gf := BuildRoot()

	quietFlag := root.PersistentFlags().Lookup("quiet")
	if quietFlag == nil || quietFlag.Shorthand != "q" {
		t.Errorf("expected --quiet to have shorthand -q, got %v", quietFlag)
	}

	verboseFlag := root.PersistentFlags().Lookup("verbose")
	if verboseFlag == nil || verboseFlag.Shorthand != "v" {
		t.Errorf("expected --verbose to have shorthand -v, got %v", verboseFlag)
	}

	err := root.ParseFlags([]string{"-q", "-v"})
	if err != nil {
		t.Fatalf("ParseFlags error: %v", err)
	}
	if !gf.Quiet {
		t.Error("expected -q to set gf.Quiet = true")
	}
	if !gf.Verbose {
		t.Error("expected -v to set gf.Verbose = true")
	}
}

// TestBuildRoot_BareInvocationRunsDashboard locks the UX contract that a bare
// `upp` (no subcommand) routes to the read-only dashboard welcome screen.
func TestBuildRoot_BareInvocationRunsDashboard(t *testing.T) {
	writeCheckConfig(t, "")

	root, gf := BuildRoot()
	root.Version = "v0.1.0"
	AddCommands(root, gf)
	root.SetArgs([]string{})

	output := withCapturedStdout(func() {
		if err := root.Execute(); err != nil {
			t.Errorf("bare upp must run dashboard and exit 0, got error: %v", err)
		}
	})

	if !strings.Contains(output, "upp v0.1.0") || !strings.Contains(output, "Commands:") {
		t.Errorf("bare upp must print the dashboard banner and commands, got:\n%s", output)
	}
}

func TestAddCommands(t *testing.T) {
	root, gf := BuildRoot()
	AddCommands(root, gf)

	expectedCommands := []string{"init", "update", "self-update", "check", "list"}
	commands := root.Commands()

	if len(commands) != len(expectedCommands) {
		t.Fatalf("expected %d commands, got %d", len(expectedCommands), len(commands))
	}

	commandNames := make(map[string]bool)
	for _, cmd := range commands {
		// Use field may include args like "import <file>", so match first word
		name := strings.Fields(cmd.Use)[0]
		commandNames[name] = true
	}

	for _, name := range expectedCommands {
		if !commandNames[name] {
			t.Errorf("missing command: %s", name)
		}
	}
}

func TestUnknownCommand_Export(t *testing.T) {
	root, gf := BuildRoot()
	AddCommands(root, gf)
	root.SetArgs([]string{"export"})

	err := root.Execute()
	if err == nil {
		t.Fatal("upp export must be rejected as unknown command")
	}
	if !strings.Contains(err.Error(), "unknown command") {
		t.Errorf("expected 'unknown command' error, got: %v", err)
	}
}

func TestUnknownCommand_Import(t *testing.T) {
	root, gf := BuildRoot()
	AddCommands(root, gf)
	root.SetArgs([]string{"import", "config.toml"})

	err := root.Execute()
	if err == nil {
		t.Fatal("upp import must be rejected as unknown command")
	}
	if !strings.Contains(err.Error(), "unknown command") {
		t.Errorf("expected 'unknown command' error, got: %v", err)
	}
}

func TestSilenceStdout(t *testing.T) {
	err := SilenceStdout(func() error {
		return nil
	})
	if err != nil {
		t.Errorf("SilenceStdout should not error, got %v", err)
	}
}

func TestSilenceStdout_Error(t *testing.T) {
	err := SilenceStdout(func() error {
		return fmt.Errorf("test error")
	})
	if err == nil {
		t.Error("SilenceStdout should propagate errors")
	}
	if err.Error() != "test error" {
		t.Errorf("expected 'test error', got %q", err.Error())
	}
}

// --- Self-update command registration and flag semantics ---

func TestSelfUpdateCommand_Short(t *testing.T) {
	cmd := NewSelfUpdateCommand(&GlobalFlags{})
	if cmd.Short != "Update the upp binary itself" {
		t.Errorf("Short = %q, want %q", cmd.Short, "Update the upp binary itself")
	}
	if cmd.Use != "self-update" {
		t.Errorf("Use = %q, want %q", cmd.Use, "self-update")
	}
}

func TestSelfUpdateCommand_NoLocalFlags(t *testing.T) {
	cmd := NewSelfUpdateCommand(&GlobalFlags{})
	if cmd.HasAvailableLocalFlags() {
		t.Error("self-update must accept no local flags in v1")
	}
}

func TestSelfUpdateCommand_UnknownFlagRejected(t *testing.T) {
	root, gf := BuildRoot()
	AddCommands(root, gf)
	root.SetArgs([]string{"self-update", "--yes"})

	err := root.Execute()
	if err == nil {
		t.Fatal("unknown flag --yes must be rejected")
	}
	if !strings.Contains(err.Error(), "unknown flag") {
		t.Errorf("rejection should mention the unknown flag, got: %v", err)
	}
}
