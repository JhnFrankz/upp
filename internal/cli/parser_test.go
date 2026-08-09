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

func TestAddCommands(t *testing.T) {
	root, gf := BuildRoot()
	AddCommands(root, gf)

	expectedCommands := []string{"init", "update", "check", "list", "export", "import"}
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
