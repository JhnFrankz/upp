package cli

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/config"
)

// --- Helper: inject package-level deps ---

// setCLIDeps swaps the package-level cliDeps var for the duration of the
// test, restoring the previous value on cleanup. Unset fields keep the
// production (zero) behavior. Sequential-only: no t.Parallel exists in this
// package — adding any requires synchronization (see deps.go).
func setCLIDeps(t *testing.T, update updateDeps, list listDeps, selfUpdate selfUpdateDeps) {
	t.Helper()
	prev := cliDeps
	cliDeps.update = update
	cliDeps.list = list
	cliDeps.selfUpdate = selfUpdate
	t.Cleanup(func() { cliDeps = prev })
}

// --- Helper: capture stdout ---

// withCapturedStdout runs fn while capturing os.Stdout.
// Returns the captured output.
func withCapturedStdout(fn func()) string {
	origStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn()

	_ = w.Close()
	os.Stdout = origStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
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
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	root, gf := BuildRoot()
	AddCommands(root, gf)
	root.Version = "0.2.0"

	if root.Use != "upp" {
		t.Errorf("expected Use='upp', got %q", root.Use)
	}

	commands := root.Commands()
	if len(commands) != 4 {
		t.Errorf("expected 4 subcommands, got %d", len(commands))
	}

	// Bare execution in empty dir outputs no-config dashboard
	out := withCapturedStdout(func() {
		root.SetArgs([]string{})
		if err := root.Execute(); err != nil {
			t.Fatalf("bare upp execution error: %v", err)
		}
	})

	if !strings.Contains(out, "upp 0.2.0") || !strings.Contains(out, "No configuration found") {
		t.Errorf("expected no-config dashboard banner, got:\n%s", out)
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
	if !strings.Contains(output, "Commands") || !strings.Contains(output, "Maintenance") {
		t.Errorf("help output should contain 2 groups Commands and Maintenance, got:\n%s", output)
	}
	if strings.Contains(output, "Tool Commands") || strings.Contains(output, "Config Commands") {
		t.Errorf("help output must not contain legacy groups, got:\n%s", output)
	}
	for _, cmd := range []string{"init", "update", "list", "self-update"} {
		if !strings.Contains(output, cmd) {
			t.Errorf("help output should list %s command", cmd)
		}
	}
	for _, pruned := range []string{"export", "import"} {
		if strings.Contains(output, pruned) {
			t.Errorf("help output must not list pruned command %s", pruned)
		}
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
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	output := withCapturedStdout(func() {
		root, gf := BuildRoot()
		AddCommands(root, gf)
		root.SetArgs([]string{"init", "--ci"})
		_ = root.Execute()
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
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	output := withCapturedStdout(func() {
		root, gf := BuildRoot()
		AddCommands(root, gf)
		root.SetArgs([]string{"init", "--ci"})
		_ = root.Execute()
	})

	if !strings.Contains(output, "detecting installed tools") {
		t.Error("init should show detecting message")
	}
}

// --- Pruned Commands Integration Test ---

func TestPrunedCommands_ExportImportRejected(t *testing.T) {
	root, gf := BuildRoot()
	AddCommands(root, gf)

	root.SetArgs([]string{"export"})
	if err := root.Execute(); err == nil {
		t.Fatal("upp export must fail as unknown command")
	}

	root.SetArgs([]string{"import", "config.toml"})
	if err := root.Execute(); err == nil {
		t.Fatal("upp import must fail as unknown command")
	}
}

// --- List Command Integration Test ---

func TestListCommand_NoConfig(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	fake := &fakeUpdateAdapter{
		name:   "apt",
		policy: adapters.PolicyGated,
		trust:  adapters.TrustOfficial,
		info:   adapters.UpdateInfo{CurrentVersion: "1.0.0"},
	}
	setCLIDeps(t, updateDeps{}, listDeps{buildAdapterList: fakeAdapterList(fake)}, selfUpdateDeps{})

	output := withCapturedStdout(func() {
		root, gf := BuildRoot()
		AddCommands(root, gf)
		root.SetArgs([]string{"list"})
		_ = root.Execute()
	})

	// list with no config should still work (uses defaults)
	if len(output) == 0 {
		t.Error("list should produce some output")
	}
}

// TestListCommand_FilterRoundTrip_GroupingDisplayOnly proves the --only/--skip
// filter round-trip survives the grouping change (task 3.5): runList filters
// by per-tool ID BEFORE GroupByOwner, so a filtered ID still appears as a row
// (usable with --only/--skip) even when its owning manager was filtered out —
// grouping is display-only and never drops or renames a row ID.
func TestListCommand_FilterRoundTrip_GroupingDisplayOnly(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// docker is owned by apt on linux. Request --only docker so apt (the
	// manager) is filtered out; docker must still render a row (round-trip
	// ID), never be dropped by a phantom manager group.
	docker := &fakeUpdateAdapter{
		name:   "docker",
		policy: adapters.PolicyAlwaysUpdate,
		trust:  adapters.TrustOfficial,
		info:   adapters.UpdateInfo{CurrentVersion: "26.0.0"},
	}
	setCLIDeps(t, updateDeps{}, listDeps{buildAdapterList: fakeAdapterList(docker)}, selfUpdateDeps{})

	output := withCapturedStdout(func() {
		root, gf := BuildRoot()
		AddCommands(root, gf)
		root.SetArgs([]string{"list", "--only", "docker"})
		_ = root.Execute()
	})

	if !strings.Contains(output, "docker") {
		t.Errorf("--only docker must round-trip the docker row despite grouping, got:\n%s", output)
	}
	// The filtered-out manager (apt) must not render a phantom header.
	if strings.Contains(output, "APT Package Manager") {
		t.Errorf("filtered-out manager must not create a phantom header, got:\n%s", output)
	}
}

// --- Update Dry-Run Command Integration Tests (ported from `upp check`) ---

// TestUpdateDryRunCommand_NoConfig is the `upp update --dry-run` port of the
// former TestCheckCommand_NoConfig: the read-only query surface works with
// no config present (uses defaults) and produces output.
func TestUpdateDryRunCommand_NoConfig(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	fake := &fakeUpdateAdapter{
		name:   "apt",
		policy: adapters.PolicyGated,
		trust:  adapters.TrustOfficial,
		info:   adapters.UpdateInfo{CurrentVersion: "1.0.0"},
	}
	setCLIDeps(t, updateDeps{buildAdapterList: fakeAdapterList(fake)}, listDeps{}, selfUpdateDeps{})

	output := withCapturedStdout(func() {
		root, gf := BuildRoot()
		AddCommands(root, gf)
		root.SetArgs([]string{"update", "--dry-run"})
		_ = root.Execute()
	})

	if len(output) == 0 {
		t.Error("update --dry-run should produce some output")
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
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Untrusted custom tool with command chaining → Medium risk, which CI
	// rejects under D4 (an untrusted CI low-risk command would auto-proceed).
	fake := &fakeUpdateAdapter{
		name:    "untrusted-tool",
		policy:  adapters.PolicyAlwaysUpdate,
		trust:   adapters.TrustCustomUntrusted,
		command: "untrusted-tool --update && echo done",
	}
	setCLIDeps(t, updateDeps{buildAdapterList: fakeAdapterList(fake)}, listDeps{}, selfUpdateDeps{})

	// In CI mode, update should fail because untrusted tool can't be confirmed
	root, gf := BuildRoot()
	AddCommands(root, gf)
	root.SetArgs([]string{"update", "--ci", "--only", "untrusted-tool"})

	err := root.Execute()
	// Should error because untrusted tool in CI
	if err == nil {
		t.Error("update --ci with untrusted tool should fail")
	}
	if fake.updated {
		t.Error("untrusted tool must not execute in CI mode")
	}
}

// --- Dry-Run Integration Test ---

func TestDryRun_NoCommandsExecuted(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	fake := &fakeUpdateAdapter{
		name:   "apt",
		policy: adapters.PolicyGated,
		trust:  adapters.TrustOfficial,
		info: adapters.UpdateInfo{
			CurrentVersion:  "1.0.0",
			LatestVersion:   "2.0.0",
			UpdateAvailable: true,
		},
	}
	setCLIDeps(t, updateDeps{buildAdapterList: fakeAdapterList(fake)}, listDeps{}, selfUpdateDeps{})

	output := withCapturedStdout(func() {
		root, gf := BuildRoot()
		AddCommands(root, gf)
		root.SetArgs([]string{"update", "--dry-run"})
		_ = root.Execute()
	})

	if !strings.Contains(output, "Dry run") {
		t.Errorf("dry-run output should contain 'Dry run', got: %q", output)
	}
	if fake.updated {
		t.Error("dry run must not execute updates (Update was called)")
	}
}

// --- AdapterIDs Integration Test ---

func TestAdapterIDs(t *testing.T) {
	adapterList := []adapters.Adapter{
		&fakeUpdateAdapter{name: "apt", policy: adapters.PolicyGated},
		&fakeUpdateAdapter{name: "brew", policy: adapters.PolicyAlwaysUpdate},
		&fakeUpdateAdapter{name: "npm", policy: adapters.PolicyGated},
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
		&fakeUpdateAdapter{name: "apt", policy: adapters.PolicyGated},
		&fakeUpdateAdapter{name: "brew", policy: adapters.PolicyAlwaysUpdate},
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

// --- Quiet Mode Integration Test ---

// TestQuietMode_SuppressesProgress is the `upp update --dry-run --quiet`
// port of the former check-path quiet test: quiet mode must not contain
// progress indicators.
func TestQuietMode_SuppressesProgress(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Two tools so progress WOULD print without --quiet (multi-tool loop).
	fakes := []*fakeUpdateAdapter{
		{name: "apt", policy: adapters.PolicyGated, trust: adapters.TrustOfficial, info: adapters.UpdateInfo{CurrentVersion: "1.0.0"}},
		{name: "npm", policy: adapters.PolicyGated, trust: adapters.TrustOfficial, info: adapters.UpdateInfo{CurrentVersion: "10.0.0"}},
	}
	setCLIDeps(t, updateDeps{buildAdapterList: fakeAdapterList(fakes[0], fakes[1])}, listDeps{}, selfUpdateDeps{})

	output := withCapturedStdout(func() {
		root, gf := BuildRoot()
		AddCommands(root, gf)
		root.SetArgs([]string{"update", "--dry-run", "--quiet"})
		_ = root.Execute()
	})

	// Quiet mode should not contain progress indicators
	if strings.Contains(output, "Updating") {
		t.Error("quiet mode should not contain progress indicators")
	}
}

// --- Complex Config Integration Test ---

func TestComplexConfigRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cfg := config.DefaultConfig()
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
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	fake := &fakeUpdateAdapter{
		name:   "apt",
		policy: adapters.PolicyGated,
		trust:  adapters.TrustOfficial,
		info: adapters.UpdateInfo{
			CurrentVersion:  "1.0.0",
			LatestVersion:   "2.0.0",
			UpdateAvailable: true,
		},
	}
	setCLIDeps(t, updateDeps{buildAdapterList: fakeAdapterList(fake)}, listDeps{}, selfUpdateDeps{})

	output := withCapturedStdout(func() {
		root, gf := BuildRoot()
		AddCommands(root, gf)
		root.SetArgs([]string{"update", "--dry-run"})
		_ = root.Execute()
	})

	if !strings.Contains(output, "Dry run") && !strings.Contains(output, "All tools") {
		t.Errorf("update output should contain summary, got: %q", output)
	}
}

// --- Edge Case: Empty Config ---

func TestEmptyConfig_AllToolsSkipped(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

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
		root.SetArgs([]string{"update", "--dry-run"})
		_ = root.Execute()
	})

	// With all tools disabled there is nothing to query: the dry-run
	// summary reports the honest "not installed" outcome.
	if !strings.Contains(output, "All tools not installed. Nothing to do.") {
		t.Errorf("all-disabled config dry run should show the nothing-to-do message, got: %q", output)
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
				Settings: config.Settings{},
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
		Settings: config.Settings{},
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
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

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

// --- WU4: buildAdapterList threads custom-tool manager ---

// TestBuildAdapterList_ThreadsCustomManager pins the WU4 contract (design
// Config; spec Config Format): a custom tool declaring a valid `manager` gets
// that manager resolved from the OFFICIAL registry and injected as the
// adapter's ManagerAdapter, so the delegated update + grouping paths can use
// it. The resolution happens in the CLI layer (buildAdapterList) because the
// adapters package must not import the official registry (no import cycle).
func TestBuildAdapterList_ThreadsCustomManager(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Custom["mytool"] = config.CustomTool{
		Command: "mytool --update",
		Trusted: false,
		Manager: "brew", // valid manager-kind official tool
	}

	adapterList := buildAdapterList(cfg, "linux")

	var custom *adapters.CustomAdapter
	for _, a := range adapterList {
		if ca, ok := a.(*adapters.CustomAdapter); ok && ca.Name() == "mytool" {
			custom = ca
			break
		}
	}
	if custom == nil {
		t.Fatal("custom adapter 'mytool' should be in adapter list")
	}
	mgr := custom.ManagerAdapter()
	if mgr == nil {
		t.Fatal("custom tool with manager='brew' should have an injected ManagerAdapter")
	}
	if mgr.Name() != "brew" {
		t.Errorf("ManagerAdapter().Name() = %q, want %q", mgr.Name(), "brew")
	}
	// The injected manager must be a manager-kind official adapter.
	if mgr.Info().Kind != adapters.KindManager {
		t.Errorf("injected manager Kind = %v, want KindManager", mgr.Info().Kind)
	}
}

// TestBuildAdapterList_UnknownManagerStaysStandalone pins that a custom tool
// with a manager value naming an UNKNOWN tool (already cleared by config
// Validate, but guarded here too) leaves the tool standalone: no manager is
// injected, so the delegated path is not taken.
func TestBuildAdapterList_UnknownManagerStaysStandalone(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Custom["mytool"] = config.CustomTool{
		Command: "mytool --update",
		Manager: "bogus", // unknown — not a manager-kind official tool
	}

	adapterList := buildAdapterList(cfg, "linux")

	var custom *adapters.CustomAdapter
	for _, a := range adapterList {
		if ca, ok := a.(*adapters.CustomAdapter); ok && ca.Name() == "mytool" {
			custom = ca
			break
		}
	}
	if custom == nil {
		t.Fatal("custom adapter 'mytool' should be in adapter list even with unknown manager")
	}
	if custom.ManagerAdapter() != nil {
		t.Error("custom tool with unknown manager must stay standalone (nil ManagerAdapter)")
	}
}

// --- Init → Update Dry-Run Lifecycle Test ---

func TestInitUpdateDryRunLifecycle(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	fake := &fakeUpdateAdapter{
		name:   "apt",
		policy: adapters.PolicyGated,
		trust:  adapters.TrustOfficial,
		info: adapters.UpdateInfo{
			CurrentVersion:  "1.0.0",
			LatestVersion:   "2.0.0",
			UpdateAvailable: true,
		},
	}
	setCLIDeps(t,
		updateDeps{buildAdapterList: fakeAdapterList(fake)},
		listDeps{},
		selfUpdateDeps{})

	// Step 1: init --ci (real runInit — LookPath-only, no subprocesses)
	withCapturedStdout(func() {
		root, gf := BuildRoot()
		AddCommands(root, gf)
		root.SetArgs([]string{"init", "--ci"})
		_ = root.Execute()
	})

	// Verify config exists
	cfgPath := filepath.Join(tmpDir, ".config", "upp", "config.toml")
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		t.Fatal("config should exist after init")
	}

	// Step 2: update --dry-run (read-only query surface, formerly `check`)
	output := withCapturedStdout(func() {
		root, gf := BuildRoot()
		AddCommands(root, gf)
		root.SetArgs([]string{"update", "--dry-run"})
		_ = root.Execute()
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
		"init":        false,
		"update":      false,
		"self-update": false,
		"list":        false,
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

// --- Update Dry-Run Summary Output (ported from check) ---

// fakeSkipAdapter is a hermetic adapter whose Detect() reports false, so
// the update dry-run records it as StatusSkipped (update.go Detect gate) —
// the S2 query-with-skips honesty path.
type fakeSkipAdapter struct {
	id string
}

func (f *fakeSkipAdapter) Name() string { return f.id }

func (f *fakeSkipAdapter) Detect() bool { return false }

func (f *fakeSkipAdapter) Check() (adapters.UpdateInfo, error) {
	return adapters.UpdateInfo{}, nil
}

func (f *fakeSkipAdapter) Update(dryRun bool) (adapters.Result, error) {
	return adapters.Result{Success: true}, nil
}

func (f *fakeSkipAdapter) Info() adapters.ToolInfo {
	return adapters.ToolInfo{ID: f.id, Name: f.id}
}

// TestUpdateDryRun_WithSkips is the `upp update --dry-run` port of the
// former TestCheckCommand_WithSkips. It locks the S2 D4 honesty contract
// end-to-end: when one enabled tool is current and another is skipped (not
// installed), the summary counts both ("1 up to date, 1 skipped") and never
// prints "All tools up to date.".
func TestUpdateDryRun_WithSkips(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	current := &fakeUpdateAdapter{
		name:   "apt",
		policy: adapters.PolicyGated,
		trust:  adapters.TrustOfficial,
		info:   adapters.UpdateInfo{CurrentVersion: "1.0.0"},
	}
	skipped := &fakeSkipAdapter{id: "nvm"}
	setCLIDeps(t, updateDeps{buildAdapterList: func(*config.Config, string) []adapters.Adapter {
		return []adapters.Adapter{current, skipped}
	}}, listDeps{}, selfUpdateDeps{})

	output := withCapturedStdout(func() {
		root, gf := BuildRoot()
		AddCommands(root, gf)
		root.SetArgs([]string{"update", "--dry-run"})
		_ = root.Execute()
	})

	if !strings.Contains(output, "1 up to date, 1 skipped") {
		t.Errorf("dry-run summary must count skipped tools explicitly, got:\n%s", output)
	}
	if strings.Contains(output, "All tools up to date.") {
		t.Errorf("dry run must never claim 'All tools up to date.' when a tool was skipped, got:\n%s", output)
	}
	// Non-quiet detail lists the skipped tool.
	if !strings.Contains(output, "nvm") {
		t.Errorf("non-quiet dry-run detail should list skipped tools, got:\n%s", output)
	}
}

// TestUpdateDryRun_SummaryOutput is the `upp update --dry-run` port of the
// former TestCheckCommand_SummaryOutput: the read-only query produces a
// summary.
func TestUpdateDryRun_SummaryOutput(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	fake := &fakeUpdateAdapter{
		name:   "apt",
		policy: adapters.PolicyGated,
		trust:  adapters.TrustOfficial,
		info:   adapters.UpdateInfo{CurrentVersion: "1.0.0"},
	}
	setCLIDeps(t, updateDeps{buildAdapterList: fakeAdapterList(fake)}, listDeps{}, selfUpdateDeps{})

	output := withCapturedStdout(func() {
		root, gf := BuildRoot()
		AddCommands(root, gf)
		root.SetArgs([]string{"update", "--dry-run"})
		_ = root.Execute()
	})

	hasSummary := strings.Contains(output, "All tools not installed") ||
		strings.Contains(output, "up to date") ||
		strings.Contains(output, "would update")
	if !hasSummary {
		t.Errorf("update --dry-run should produce summary, got: %q", output)
	}
}

// --- Init with Existing Config ---

func TestInitCommand_AlreadyExists(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cfg := config.DefaultConfigWithDefaults()
	if err := config.Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	output := withCapturedStdout(func() {
		root, gf := BuildRoot()
		AddCommands(root, gf)
		root.SetArgs([]string{"init", "--ci"})
		_ = root.Execute()
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

// TestUpdateDryRun_DeterministicOrderUnderConcurrency is the
// `upp update --dry-run` port of the former check-path ordering test: even
// when adapters complete in reverse/arbitrary order due to concurrency and
// varying execution times, the dry-run planned actions and summary strictly
// preserve canonical tool discovery order.
func TestUpdateDryRun_DeterministicOrderUnderConcurrency(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	fakes := []adapters.Adapter{
		&fakeDelayedAdapter{name: "alpha", delay: 50 * time.Millisecond, info: adapters.UpdateInfo{UpdateAvailable: true, CurrentVersion: "1.0.0", LatestVersion: "1.1.0"}},
		&fakeDelayedAdapter{name: "beta", delay: 10 * time.Millisecond, info: adapters.UpdateInfo{CurrentVersion: "2.0.0"}},
		&fakeDelayedAdapter{name: "gamma", delay: 40 * time.Millisecond, info: adapters.UpdateInfo{UpdateAvailable: true, CurrentVersion: "3.0.0", LatestVersion: "3.1.0"}},
		&fakeDelayedAdapter{name: "delta", delay: 20 * time.Millisecond, info: adapters.UpdateInfo{CurrentVersion: "4.0.0"}},
		&fakeDelayedAdapter{name: "epsilon", delay: 30 * time.Millisecond, info: adapters.UpdateInfo{UpdateAvailable: true, CurrentVersion: "5.0.0", LatestVersion: "5.1.0"}},
	}

	setCLIDeps(t, updateDeps{
		buildAdapterList: func(*config.Config, string) []adapters.Adapter {
			return fakes
		},
	}, listDeps{}, selfUpdateDeps{})

	output := withCapturedStdout(func() {
		root, gf := BuildRoot()
		AddCommands(root, gf)
		root.SetArgs([]string{"update", "--dry-run"})
		if err := root.Execute(); err != nil {
			t.Fatalf("update --dry-run execution failed: %v", err)
		}
	})

	// Summary counts
	if !strings.Contains(output, "3 would update") || !strings.Contains(output, "2 up to date") {
		t.Errorf("summary counts mismatch, got:\n%s", output)
	}

	// Verify available-tool order strictly preserves canonical discovery
	// sequence: alpha, gamma, epsilon (each pending tool appears exactly
	// once, in its planned-action line).
	idxAlpha := strings.Index(output, "alpha")
	idxGamma := strings.Index(output, "gamma")
	idxEpsilon := strings.Index(output, "epsilon")

	if idxAlpha == -1 || idxGamma == -1 || idxEpsilon == -1 {
		t.Fatalf("expected all available tools in dry-run output, got:\n%s", output)
	}

	if idxAlpha >= idxGamma || idxGamma >= idxEpsilon {
		t.Errorf("tool order violation: expected alpha < gamma < epsilon, got alpha=%d, gamma=%d, epsilon=%d in output:\n%s",
			idxAlpha, idxGamma, idxEpsilon, output)
	}
}

// TestUpdateDryRun_MixedStatusCounts ports the former runCheck command-level
// mixed-status coverage onto the read-only query surface: one pending,
// two current, two failing tools (one generic check error, one timeout)
// produce exact dry-run summary counts with no update executed.
func TestUpdateDryRun_MixedStatusCounts(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	a0 := &fakeUpdateAdapter{name: "tool-0", policy: adapters.PolicyGated, trust: adapters.TrustOfficial, info: adapters.UpdateInfo{CurrentVersion: "1.0.0"}}
	a1 := &fakeDelayedAdapter{name: "tool-1", checkErr: fmt.Errorf("lock frontend held by another process")}
	a2 := &fakeUpdateAdapter{name: "tool-2", policy: adapters.PolicyGated, trust: adapters.TrustOfficial, info: adapters.UpdateInfo{UpdateAvailable: true, CurrentVersion: "1.0.0", LatestVersion: "1.2.0"}}
	a3 := &fakeDelayedAdapter{name: "tool-3", checkErr: context.DeadlineExceeded}
	a4 := &fakeUpdateAdapter{name: "tool-4", policy: adapters.PolicyGated, trust: adapters.TrustOfficial, info: adapters.UpdateInfo{CurrentVersion: "1.0.0"}}

	setCLIDeps(t, updateDeps{buildAdapterList: func(*config.Config, string) []adapters.Adapter {
		return []adapters.Adapter{a0, a1, a2, a3, a4}
	}}, listDeps{}, selfUpdateDeps{})

	out := withCapturedStdout(func() {
		root, gf := BuildRoot()
		AddCommands(root, gf)
		root.SetArgs([]string{"update", "--dry-run"})
		if err := root.Execute(); err != nil {
			t.Errorf("update --dry-run returned error: %v", err)
		}
	})

	for _, a := range []*fakeUpdateAdapter{a0, a2, a4} {
		if a.updated {
			t.Errorf("dry run must not execute updates, but %s was updated", a.name)
		}
	}
	if !strings.Contains(out, "1 would update") {
		t.Errorf("expected '1 would update' in summary, got:\n%s", out)
	}
	if !strings.Contains(out, "2 up to date") {
		t.Errorf("expected '2 up to date' in summary, got:\n%s", out)
	}
	if !strings.Contains(out, "2 failed") {
		t.Errorf("expected '2 failed' in summary, got:\n%s", out)
	}
}
