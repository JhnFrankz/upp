package cli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/config"
)

// --- Mock Adapter ---

// mockAdapter is a test double that implements adapters.Adapter.
type mockAdapter struct {
	name       string
	detect     bool
	updateInfo adapters.UpdateInfo
	checkErr   error
	result     adapters.Result
	updateErr  error
}

func (m *mockAdapter) Name() string { return m.name }

func (m *mockAdapter) Detect() bool { return m.detect }

func (m *mockAdapter) Check() (adapters.UpdateInfo, error) {
	return m.updateInfo, m.checkErr
}

func (m *mockAdapter) Update(dryRun bool) (adapters.Result, error) {
	if dryRun {
		return adapters.Result{Success: true, Before: "1.0.0", After: "1.0.0"}, nil
	}
	return m.result, m.updateErr
}

func (m *mockAdapter) Info() adapters.ToolInfo {
	return adapters.ToolInfo{
		ID:        m.name,
		Name:      m.name,
		Platforms: []string{"linux", "darwin", "windows"},
		Trust:     adapters.TrustTrusted,
	}
}

// --- Helper: capture stdout ---

// withCapturedStdout runs fn while capturing os.Stdout.
// Returns the captured output.
func withCapturedStdout(fn func()) string {
	origStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = origStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	return buf.String()
}

// --- Filter Integration Tests ---

func TestFilterTools_Integration(t *testing.T) {
	tests := []struct {
		name      string
		tools     []string
		only      string
		skip      string
		wantCount int
		wantNames []string
	}{
		{
			name:      "no filter returns all",
			tools:     []string{"apt", "brew", "npm", "docker"},
			wantCount: 4,
		},
		{
			name:      "only filter",
			tools:     []string{"apt", "brew", "npm", "docker"},
			only:      "brew,npm",
			wantCount: 2,
			wantNames: []string{"brew", "npm"},
		},
		{
			name:      "skip filter",
			tools:     []string{"apt", "brew", "npm", "docker"},
			skip:      "apt,docker",
			wantCount: 2,
			wantNames: []string{"brew", "npm"},
		},
		{
			name:      "only wins over skip",
			tools:     []string{"apt", "brew", "npm", "docker"},
			only:      "brew",
			skip:      "apt",
			wantCount: 1,
			wantNames: []string{"brew"},
		},
		{
			name:      "empty only returns all",
			tools:     []string{"apt", "brew"},
			only:      "",
			wantCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stderr bytes.Buffer
			onlyList, skipList := ParseFilter(tt.only, tt.skip)
			result := FilterTools(tt.tools, onlyList, skipList, &stderr)

			if len(result) != tt.wantCount {
				t.Errorf("expected %d tools, got %d: %v", tt.wantCount, len(result), result)
			}

			if tt.wantNames != nil {
				for i, name := range tt.wantNames {
					if i < len(result) && result[i] != name {
						t.Errorf("expected tool[%d] = %q, got %q", i, name, result[i])
					}
				}
			}
		})
	}
}

// --- Root Command Integration Tests ---

func TestRootCommand_NoArgs(t *testing.T) {
	root, gf := BuildRoot()
	AddCommands(root, gf)

	if root.Use != "upp" {
		t.Errorf("expected Use='upp', got %q", root.Use)
	}

	commands := root.Commands()
	if len(commands) != 6 {
		t.Errorf("expected 6 subcommands, got %d", len(commands))
	}
}

func TestRootCommand_Help(t *testing.T) {
	root, gf := BuildRoot()
	AddCommands(root, gf)

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetArgs([]string{"--help"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("help command failed: %v", err)
	}

	output := buf.String()
	// Cobra shows Long description when both Long and Short are set
	if !strings.Contains(output, "upp updates your development tools") {
		t.Errorf("help output should contain description, got: %q", output)
	}
	if !strings.Contains(output, "init") {
		t.Error("help output should list init command")
	}
	if !strings.Contains(output, "update") {
		t.Error("help output should list update command")
	}
}

func TestRootCommand_Version(t *testing.T) {
	root, gf := BuildRoot()
	AddCommands(root, gf)
	root.Version = "1.2.3"

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetArgs([]string{"--version"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("version command failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "1.2.3") {
		t.Errorf("version output should contain '1.2.3', got: %q", output)
	}
}

// --- Init Command Integration Tests ---

func TestInitCommand_CI_Mode(t *testing.T) {
	origHome := os.Getenv("HOME")
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	output := withCapturedStdout(func() {
		root, gf := BuildRoot()
		AddCommands(root, gf)
		root.SetArgs([]string{"init", "--ci"})
		root.Execute()
	})

	// Config should be written
	cfgPath := filepath.Join(tmpDir, ".config", "upp", "config.toml")
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		t.Error("init --ci should create config file")
	}

	if !strings.Contains(output, "Config written to") {
		t.Errorf("init --ci should output config path, got: %q", output)
	}
}

func TestInitCommand_DetectsTools(t *testing.T) {
	origHome := os.Getenv("HOME")
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	output := withCapturedStdout(func() {
		root, gf := BuildRoot()
		AddCommands(root, gf)
		root.SetArgs([]string{"init", "--ci"})
		root.Execute()
	})

	if !strings.Contains(output, "detecting installed tools") {
		t.Error("init should show detecting message")
	}
}

// --- Export/Import Round-Trip Integration Test ---

func TestExportImport_RoundTrip(t *testing.T) {
	origHome := os.Getenv("HOME")
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	// Create initial config
	cfg := config.DefaultConfigWithDefaults()
	cfg.Settings.Language = "es"
	cfg.Settings.Interactive = false
	cfg.Custom["test-tool"] = config.CustomTool{
		Command:  "test-tool --update",
		CheckCmd: "test-tool --version",
		Trusted:  true,
	}
	if err := config.Save(cfg); err != nil {
		t.Fatalf("Save initial config: %v", err)
	}

	// Export to file
	exportPath := filepath.Join(tmpDir, "exported.toml")
	withCapturedStdout(func() {
		root, gf := BuildRoot()
		AddCommands(root, gf)
		root.SetArgs([]string{"export", "-o", exportPath})
		root.Execute()
	})

	// Verify export file exists and has content
	data, err := os.ReadFile(exportPath)
	if err != nil {
		t.Fatalf("cannot read export file: %v", err)
	}
	if len(data) == 0 {
		t.Error("exported file is empty")
	}

	// Reset HOME and load from import
	os.Setenv("HOME", t.TempDir())

	withCapturedStdout(func() {
		root, gf := BuildRoot()
		AddCommands(root, gf)
		root.SetArgs([]string{"import", exportPath, "--ci"})
		root.Execute()
	})

	// Verify imported config
	loaded, err := config.Load()
	if err != nil {
		t.Fatalf("Load imported config: %v", err)
	}

	if loaded.Settings.Language != "es" {
		t.Errorf("imported language = %q, want %q", loaded.Settings.Language, "es")
	}
	if loaded.Settings.Interactive {
		t.Error("imported interactive should be false")
	}
	if loaded.Custom["test-tool"].Command != "test-tool --update" {
		t.Errorf("imported custom tool command = %q", loaded.Custom["test-tool"].Command)
	}
	if !loaded.Custom["test-tool"].Trusted {
		t.Error("imported custom tool should be trusted")
	}
}

// --- List Command Integration Test ---

func TestListCommand_NoConfig(t *testing.T) {
	origHome := os.Getenv("HOME")
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	output := withCapturedStdout(func() {
		root, gf := BuildRoot()
		AddCommands(root, gf)
		root.SetArgs([]string{"list"})
		root.Execute()
	})

	// list with no config should still work (uses defaults)
	if len(output) == 0 {
		t.Error("list should produce some output")
	}
}

// --- Check Command Integration Test ---

func TestCheckCommand_NoConfig(t *testing.T) {
	origHome := os.Getenv("HOME")
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	output := withCapturedStdout(func() {
		root, gf := BuildRoot()
		AddCommands(root, gf)
		root.SetArgs([]string{"check"})
		root.Execute()
	})

	// check with no config should still work
	if len(output) == 0 {
		t.Error("check should produce some output")
	}
}

// --- BuildAdapterList Integration Test ---

func TestBuildAdapterList_RespectsEnabledTools(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Tools["apt"] = config.ToolConfig{Enabled: false}
	cfg.Tools["npm"] = config.ToolConfig{Enabled: true}
	cfg.Tools["brew"] = config.ToolConfig{Enabled: false}

	adapterList := buildAdapterList(cfg, "linux")

	for _, a := range adapterList {
		info := a.Info()
		if info.ID == "apt" {
			t.Error("apt should not be in adapter list when disabled")
		}
		if info.ID == "brew" {
			t.Error("brew should not be in adapter list when disabled")
		}
	}
}

func TestBuildAdapterList_IncludesCustomAdapters(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Custom["mytool"] = config.CustomTool{
		Command:  "mytool --update",
		CheckCmd: "mytool --version",
		Trusted:  false,
	}

	adapterList := buildAdapterList(cfg, "linux")

	found := false
	for _, a := range adapterList {
		if a.Name() == "mytool" {
			found = true
			break
		}
	}
	if !found {
		t.Error("custom adapter 'mytool' should be in adapter list")
	}
}

// --- CI Mode Integration Test ---

func TestCIMode_RejectsUntrustedCustomTools(t *testing.T) {
	origHome := os.Getenv("HOME")
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	// Create a mock script so the tool is detectable
	mockDir := filepath.Join(tmpDir, "bin")
	os.MkdirAll(mockDir, 0o755)
	mockScript := filepath.Join(mockDir, "untrusted-tool")
	os.WriteFile(mockScript, []byte("#!/bin/sh\necho 1.0.0"), 0o755)

	// Prepend mock dir to PATH
	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", mockDir+":"+oldPath)
	defer os.Setenv("PATH", oldPath)

	cfg := config.DefaultConfigWithDefaults()
	cfg.Custom["untrusted-tool"] = config.CustomTool{
		Command: "untrusted-tool --update",
		Trusted: false,
	}
	if err := config.Save(cfg); err != nil {
		t.Fatalf("Save config: %v", err)
	}

	// In CI mode, update should fail because untrusted tool can't be confirmed
	root, gf := BuildRoot()
	AddCommands(root, gf)
	root.SetArgs([]string{"update", "--ci", "--only", "untrusted-tool"})

	err := root.Execute()
	// Should error because untrusted tool in CI
	if err == nil {
		t.Error("update --ci with untrusted tool should fail")
	}
}

// --- Dry-Run Integration Test ---

func TestDryRun_NoCommandsExecuted(t *testing.T) {
	origHome := os.Getenv("HOME")
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	cfg := config.DefaultConfigWithDefaults()
	if err := config.Save(cfg); err != nil {
		t.Fatalf("Save config: %v", err)
	}

	output := withCapturedStdout(func() {
		root, gf := BuildRoot()
		AddCommands(root, gf)
		root.SetArgs([]string{"update", "--dry-run"})
		root.Execute()
	})

	if !strings.Contains(output, "Dry run") {
		t.Errorf("dry-run output should contain 'Dry run', got: %q", output)
	}
}

// --- AdapterIDs Integration Test ---

func TestAdapterIDs(t *testing.T) {
	adapterList := []adapters.Adapter{
		&mockAdapter{name: "apt"},
		&mockAdapter{name: "brew"},
		&mockAdapter{name: "npm"},
	}

	ids := adapterIDs(adapterList)
	if len(ids) != 3 {
		t.Fatalf("expected 3 IDs, got %d", len(ids))
	}
	if ids[0] != "apt" || ids[1] != "brew" || ids[2] != "npm" {
		t.Errorf("unexpected IDs: %v", ids)
	}
}

func TestAdapterByID(t *testing.T) {
	adapterList := []adapters.Adapter{
		&mockAdapter{name: "apt"},
		&mockAdapter{name: "brew"},
	}

	m := adapterByID(adapterList)
	if len(m) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(m))
	}
	if m["apt"] == nil {
		t.Error("missing 'apt' in map")
	}
	if m["brew"] == nil {
		t.Error("missing 'brew' in map")
	}
}

// --- Trust Level String Integration Test ---

func TestTrustLevelString(t *testing.T) {
	tests := []struct {
		level adapters.TrustLevel
		want  string
	}{
		{adapters.TrustTrusted, "official"},
		{adapters.TrustUntrusted, "custom"},
	}

	for _, tt := range tests {
		got := trustLevelString(tt.level)
		if got != tt.want {
			t.Errorf("trustLevelString(%d) = %q, want %q", tt.level, got, tt.want)
		}
	}
}

// --- Quiet Mode Integration Test ---

func TestQuietMode_SuppressesProgress(t *testing.T) {
	origHome := os.Getenv("HOME")
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	cfg := config.DefaultConfigWithDefaults()
	if err := config.Save(cfg); err != nil {
		t.Fatalf("Save config: %v", err)
	}

	output := withCapturedStdout(func() {
		root, gf := BuildRoot()
		AddCommands(root, gf)
		root.SetArgs([]string{"check", "--quiet"})
		root.Execute()
	})

	// Quiet mode should not contain progress indicators
	if strings.Contains(output, "Updating") {
		t.Error("quiet mode should not contain progress indicators")
	}
}

// --- Error Handling Integration Tests ---

func TestImport_MissingFile(t *testing.T) {
	root, gf := BuildRoot()
	AddCommands(root, gf)
	root.SetArgs([]string{"import", "/nonexistent/path.toml"})

	err := root.Execute()
	if err == nil {
		t.Error("import with missing file should fail")
	}
}

func TestImport_InvalidTOML(t *testing.T) {
	tmpDir := t.TempDir()
	badFile := filepath.Join(tmpDir, "bad.toml")
	if err := os.WriteFile(badFile, []byte("this is not toml {{{"), 0o644); err != nil {
		t.Fatal(err)
	}

	root, gf := BuildRoot()
	AddCommands(root, gf)
	root.SetArgs([]string{"import", badFile})

	err := root.Execute()
	if err == nil {
		t.Error("import with invalid TOML should fail")
	}
}

// --- Complex Config Integration Test ---

func TestComplexConfigRoundTrip(t *testing.T) {
	origHome := os.Getenv("HOME")
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	cfg := config.DefaultConfig()
	cfg.Settings.Language = "es"
	cfg.Settings.Interactive = false
	cfg.Tools["apt"] = config.ToolConfig{Enabled: true, Platforms: []string{"linux"}}
	cfg.Tools["npm"] = config.ToolConfig{Enabled: false}
	cfg.Tools["brew"] = config.ToolConfig{Enabled: true, Platforms: []string{"linux", "macos"}}
	cfg.Custom["deploy"] = config.CustomTool{
		Command:  "deploy.sh --prod --env=staging",
		CheckCmd: "deploy.sh --version",
		Trusted:  true,
	}
	cfg.Custom["lint"] = config.CustomTool{
		Command: "eslint --fix .",
		Trusted: false,
	}

	if err := config.Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.Settings.Language != "es" {
		t.Errorf("language = %q, want %q", loaded.Settings.Language, "es")
	}
	if loaded.Settings.Interactive {
		t.Error("interactive should be false")
	}
	if !loaded.Tools["apt"].Enabled {
		t.Error("apt should be enabled")
	}
	if loaded.Tools["npm"].Enabled {
		t.Error("npm should be disabled")
	}
	if !loaded.Tools["brew"].Enabled {
		t.Error("brew should be enabled")
	}
	if loaded.Tools["apt"].Platforms[0] != "linux" {
		t.Errorf("apt platforms = %v", loaded.Tools["apt"].Platforms)
	}
	if loaded.Custom["deploy"].Command != "deploy.sh --prod --env=staging" {
		t.Errorf("deploy command = %q", loaded.Custom["deploy"].Command)
	}
	if !loaded.Custom["deploy"].Trusted {
		t.Error("deploy should be trusted")
	}
	if loaded.Custom["lint"].Trusted {
		t.Error("lint should be untrusted")
	}
}

// --- Update Flow End-to-End Test ---

func TestUpdateFlow_ConfigToSummary(t *testing.T) {
	origHome := os.Getenv("HOME")
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	cfg := config.DefaultConfigWithDefaults()
	if err := config.Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	output := withCapturedStdout(func() {
		root, gf := BuildRoot()
		AddCommands(root, gf)
		root.SetArgs([]string{"update", "--dry-run"})
		root.Execute()
	})

	if !strings.Contains(output, "Dry run") && !strings.Contains(output, "All tools") {
		t.Errorf("update output should contain summary, got: %q", output)
	}
}

// --- Edge Case: Empty Config ---

func TestEmptyConfig_AllToolsSkipped(t *testing.T) {
	origHome := os.Getenv("HOME")
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	// Create config with all tools explicitly disabled
	cfg := config.DefaultConfigWithDefaults()
	for id := range cfg.Tools {
		cfg.Tools[id] = config.ToolConfig{Enabled: false}
	}
	if err := config.Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	output := withCapturedStdout(func() {
		root, gf := BuildRoot()
		AddCommands(root, gf)
		root.SetArgs([]string{"check"})
		root.Execute()
	})

	// With all tools disabled, check should show "All tools up to date" or similar
	if !strings.Contains(output, "All tools up to date") && !strings.Contains(output, "No tools") {
		t.Errorf("all-disabled config check should show appropriate message, got: %q", output)
	}
}

// --- Filter with Unknown Tools ---

func TestFilter_UnknownToolWarning(t *testing.T) {
	tools := []string{"apt", "brew", "npm"}
	onlyList, _ := ParseFilter("brew,nonexistent", "")
	var stderr bytes.Buffer

	result := FilterTools(tools, onlyList, nil, &stderr)

	warning := stderr.String()
	if !strings.Contains(warning, "nonexistent") {
		t.Errorf("should warn about unknown tool, got: %q", warning)
	}
	if len(result) != 1 || result[0] != "brew" {
		t.Errorf("expected [brew], got %v", result)
	}
}

// --- Skip All Tools ---

func TestSkipAllTools(t *testing.T) {
	tools := []string{"apt", "brew", "npm"}
	_, skipList := ParseFilter("", "apt,brew,npm")
	var stderr bytes.Buffer

	result := FilterTools(tools, nil, skipList, &stderr)

	if len(result) != 0 {
		t.Errorf("expected 0 tools after skipping all, got %d", len(result))
	}
}

// --- Config Version Validation ---

func TestConfigVersion_InvalidVersions(t *testing.T) {
	tests := []struct {
		name    string
		version int
		wantErr bool
	}{
		{"version 0 invalid", 0, true},
		{"version 1 valid", 1, false},
		{"version -1 invalid", -1, true},
		{"version 100 valid", 100, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Version:  tt.version,
				Settings: config.Settings{Language: "en"},
				Tools:    make(map[string]config.ToolConfig),
				Custom:   make(map[string]config.CustomTool),
			}
			err := config.Validate(cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate(version=%d) error = %v, wantErr %v", tt.version, err, tt.wantErr)
			}
		})
	}
}

// --- Custom Tool Validation ---

func TestCustomTool_MissingCommand(t *testing.T) {
	cfg := &config.Config{
		Version:  1,
		Settings: config.Settings{Language: "en"},
		Tools:    make(map[string]config.ToolConfig),
		Custom: map[string]config.CustomTool{
			"mytool": {Command: ""},
		},
	}

	err := config.Validate(cfg)
	if err == nil {
		t.Error("Validate should fail with empty custom tool command")
	}
}

// --- Multiple Custom Tools ---

func TestMultipleCustomTools(t *testing.T) {
	origHome := os.Getenv("HOME")
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	cfg := config.DefaultConfig()
	cfg.Custom["tool1"] = config.CustomTool{Command: "tool1 --update", Trusted: true}
	cfg.Custom["tool2"] = config.CustomTool{Command: "tool2 --update", Trusted: false}
	cfg.Custom["tool3"] = config.CustomTool{Command: "tool3 --update", Trusted: true}

	adapterList := buildAdapterList(cfg, "linux")

	customCount := 0
	for _, a := range adapterList {
		info := a.Info()
		if info.ID == "tool1" || info.ID == "tool2" || info.ID == "tool3" {
			customCount++
		}
	}

	if customCount != 3 {
		t.Errorf("expected 3 custom adapters, got %d", customCount)
	}
}

// --- Init → Check → Update Lifecycle Test ---

func TestInitCheckUpdateLifecycle(t *testing.T) {
	origHome := os.Getenv("HOME")
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	// Step 1: init --ci
	withCapturedStdout(func() {
		root, gf := BuildRoot()
		AddCommands(root, gf)
		root.SetArgs([]string{"init", "--ci"})
		root.Execute()
	})

	// Verify config exists
	cfgPath := filepath.Join(tmpDir, ".config", "upp", "config.toml")
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		t.Fatal("config should exist after init")
	}

	// Step 2: check
	withCapturedStdout(func() {
		root, gf := BuildRoot()
		AddCommands(root, gf)
		root.SetArgs([]string{"check"})
		root.Execute()
	})

	// Step 3: update --dry-run
	output := withCapturedStdout(func() {
		root, gf := BuildRoot()
		AddCommands(root, gf)
		root.SetArgs([]string{"update", "--dry-run"})
		root.Execute()
	})

	if !strings.Contains(output, "Dry run") && !strings.Contains(output, "All tools") {
		t.Errorf("lifecycle output missing summary, got: %q", output)
	}
}

// --- BuildRoot Flag Defaults ---

func TestBuildRoot_FlagDefaults(t *testing.T) {
	root, gf := BuildRoot()

	if gf.Quiet {
		t.Error("quiet should default to false")
	}
	if gf.CI {
		t.Error("ci should default to false")
	}
	if gf.Only != "" {
		t.Error("only should default to empty")
	}
	if gf.Skip != "" {
		t.Error("skip should default to empty")
	}

	flagTests := []struct {
		name   string
		exists bool
	}{
		{"quiet", root.PersistentFlags().Lookup("quiet") != nil},
		{"ci", root.PersistentFlags().Lookup("ci") != nil},
		{"only", root.PersistentFlags().Lookup("only") != nil},
		{"skip", root.PersistentFlags().Lookup("skip") != nil},
	}

	for _, ft := range flagTests {
		if !ft.exists {
			t.Errorf("flag %q should exist", ft.name)
		}
	}
}

// --- Subcommand Registration ---

func TestSubcommandRegistration(t *testing.T) {
	root, gf := BuildRoot()
	AddCommands(root, gf)

	expected := map[string]bool{
		"init":   false,
		"update": false,
		"check":  false,
		"list":   false,
		"export": false,
		"import": false,
	}

	for _, cmd := range root.Commands() {
		name := strings.Fields(cmd.Use)[0]
		if _, ok := expected[name]; ok {
			expected[name] = true
		}
	}

	for name, found := range expected {
		if !found {
			t.Errorf("missing subcommand: %s", name)
		}
	}
}

// --- Export Flags ---

func TestExportCommand_Flags(t *testing.T) {
	root, gf := BuildRoot()
	AddCommands(root, gf)

	var exportCmd *cobra.Command
	for _, cmd := range root.Commands() {
		if strings.Fields(cmd.Use)[0] == "export" {
			exportCmd = cmd
			break
		}
	}

	if exportCmd == nil {
		t.Fatal("export command not found")
	}

	outputFlag := exportCmd.Flags().Lookup("output")
	if outputFlag == nil {
		t.Error("export should have --output/-o flag")
	}
}

// --- Import Args Validation ---

func TestImportCommand_RequiresArgs(t *testing.T) {
	root, gf := BuildRoot()
	AddCommands(root, gf)
	root.SetArgs([]string{"import"})

	err := root.Execute()
	if err == nil {
		t.Error("import without args should fail")
	}
}

// --- Check Summary Output ---

func TestCheckCommand_SummaryOutput(t *testing.T) {
	origHome := os.Getenv("HOME")
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	cfg := config.DefaultConfigWithDefaults()
	if err := config.Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	output := withCapturedStdout(func() {
		root, gf := BuildRoot()
		AddCommands(root, gf)
		root.SetArgs([]string{"check"})
		root.Execute()
	})

	hasSummary := strings.Contains(output, "All tools up to date") ||
		strings.Contains(output, "available") ||
		strings.Contains(output, "current")
	if !hasSummary {
		t.Errorf("check should produce summary, got: %q", output)
	}
}

// --- Init with Existing Config ---

func TestInitCommand_AlreadyExists(t *testing.T) {
	origHome := os.Getenv("HOME")
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	cfg := config.DefaultConfigWithDefaults()
	if err := config.Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	output := withCapturedStdout(func() {
		root, gf := BuildRoot()
		AddCommands(root, gf)
		root.SetArgs([]string{"init", "--ci"})
		root.Execute()
	})

	if !strings.Contains(output, "Config written to") {
		t.Errorf("init should confirm config written, got: %q", output)
	}
}

// --- Benchmark-style: multiple filter operations ---

func TestFilterPerformance(t *testing.T) {
	tools := make([]string, 100)
	for i := range tools {
		tools[i] = fmt.Sprintf("tool%d", i)
	}

	onlyList, skipList := ParseFilter("tool10,tool50,tool90", "")
	var stderr bytes.Buffer

	result := FilterTools(tools, onlyList, skipList, &stderr)
	if len(result) != 3 {
		t.Errorf("expected 3 filtered tools, got %d", len(result))
	}
}
