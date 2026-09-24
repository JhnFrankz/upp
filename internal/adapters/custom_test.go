package adapters

import (
	"context"
	"errors"
	"testing"

	"github.com/JhnFrankz/upp/internal/security"
)

func TestNewCustomAdapter_RequiresCommand(t *testing.T) {
	_, err := NewCustomAdapter("mytool", "", "", false)
	if err == nil {
		t.Error("NewCustomAdapter with empty command should return error")
	}
}

// TestCustomAdapter_IsTrusted covers IsTrusted directly (security
// classification): trusted must be exactly what config declared.
func TestCustomAdapter_IsTrusted(t *testing.T) {
	tests := []struct {
		name    string
		trusted bool
	}{
		{"untrusted default", false},
		{"trusted via config", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ca, err := NewCustomAdapter("mytool", "mytool --update", "", tt.trusted)
			if err != nil {
				t.Fatal(err)
			}
			if got := ca.IsTrusted(); got != tt.trusted {
				t.Errorf("IsTrusted() = %v, want %v", got, tt.trusted)
			}
		})
	}
}

func TestNewCustomAdapter_Success(t *testing.T) {
	ca, err := NewCustomAdapter("mytool", "mytool --update", "mytool --version", false)
	if err != nil {
		t.Fatalf("NewCustomAdapter() unexpected error: %v", err)
	}
	if ca.Name() != "mytool" {
		t.Errorf("Name() = %q, want %q", ca.Name(), "mytool")
	}
}

func TestCustomAdapter_Detect_Found(t *testing.T) {
	setExecFakes(t, execFakes{
		lookPath: map[string]bool{"echo": true},
	})

	ca, err := NewCustomAdapter("echo", "echo hello", "", false)
	if err != nil {
		t.Fatal(err)
	}
	if !ca.Detect() {
		t.Error("Detect() = false for 'echo', want true")
	}
}

func TestCustomAdapter_Detect_NotFound(t *testing.T) {
	setExecFakes(t, execFakes{
		lookPath: map[string]bool{"nonexistent-tool-xyz": false},
	})

	ca, err := NewCustomAdapter("nonexistent", "nonexistent-tool-xyz --update", "", false)
	if err != nil {
		t.Fatal(err)
	}
	if ca.Detect() {
		t.Error("Detect() = true for nonexistent tool, want false")
	}
}

func TestCustomAdapter_Info_Untrusted(t *testing.T) {
	ca, err := NewCustomAdapter("mytool", "mytool --update", "", false)
	if err != nil {
		t.Fatal(err)
	}
	info := ca.Info()
	if info.Trust != security.TrustCustomUntrusted {
		t.Errorf("Info().Trust = %v, want TrustCustomUntrusted", info.Trust)
	}
	if info.ID != "mytool" {
		t.Errorf("Info().ID = %q, want %q", info.ID, "mytool")
	}
	if info.Command != "mytool --update" {
		t.Errorf("Info().Command = %q, want %q", info.Command, "mytool --update")
	}
	if len(info.Privileges) != 0 {
		t.Errorf("Info().Privileges = %v, want empty for non-privileged command", info.Privileges)
	}
}

func TestCustomAdapter_Info_Trusted(t *testing.T) {
	ca, err := NewCustomAdapter("mytool", "sudo mytool --update", "", true)
	if err != nil {
		t.Fatal(err)
	}
	info := ca.Info()
	if info.Trust != security.TrustCustomTrusted {
		t.Errorf("Info().Trust = %v, want TrustCustomTrusted (trusted=true must never map to Official)", info.Trust)
	}
	if info.Trust == security.TrustOfficial {
		t.Error("trusted=true must never classify as TrustOfficial")
	}
	if info.Command != "sudo mytool --update" {
		t.Errorf("Info().Command = %q, want %q", info.Command, "sudo mytool --update")
	}
	if len(info.Privileges) != 1 || info.Privileges[0] != "sudo" {
		t.Errorf("Info().Privileges = %v, want [sudo]", info.Privileges)
	}
}

// TestCustomAdapter_InfoDeclaresExplicitUpdatePolicy guards the explicit
// UpdatePolicy: PolicyAlwaysUpdate declared in CustomAdapter.Info(). The zero
// value of UpdatePolicy is PolicyGated (policy-driven gate, PR #45), so a
// refactor dropping the field would silently flip custom tools to gated
// updates; this test pins the declaration to prevent that silent fallback.
func TestCustomAdapter_InfoDeclaresExplicitUpdatePolicy(t *testing.T) {
	tests := []struct {
		name    string
		trusted bool
	}{
		{"untrusted", false},
		{"trusted", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ca, err := NewCustomAdapter("mytool", "mytool --update", "", tt.trusted)
			if err != nil {
				t.Fatal(err)
			}
			if got := ca.Info().UpdatePolicy; got != PolicyAlwaysUpdate {
				t.Errorf("Info().UpdatePolicy = %v, want PolicyAlwaysUpdate", got)
			}
		})
	}
}

func TestCustomAdapter_Check_NoCheckCmd(t *testing.T) {
	ca, err := NewCustomAdapter("mytool", "mytool --update", "", false)
	if err != nil {
		t.Fatal(err)
	}
	info, err := ca.Check(context.Background())
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if info.CurrentVersion != "" || info.LatestVersion != "" {
		t.Errorf("Check() should return empty versions when no check_cmd, got current=%q latest=%q",
			info.CurrentVersion, info.LatestVersion)
	}
}

func TestCustomAdapter_Check_WithCheckCmd(t *testing.T) {
	setExecFakes(t, execFakes{
		lookPath: map[string]bool{"echo": true},
		shell: map[string]fakeResult{
			"echo 1.2.3": {stdout: "1.2.3", err: nil},
		},
	})

	ca, err := NewCustomAdapter("echo", "echo hello", "echo 1.2.3", false)
	if err != nil {
		t.Fatal(err)
	}
	info, err := ca.Check(context.Background())
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if info.CurrentVersion != "1.2.3" {
		t.Errorf("Check() current version = %q, want %q", info.CurrentVersion, "1.2.3")
	}
}

func TestCustomAdapter_Check_MissingBinary(t *testing.T) {
	setExecFakes(t, execFakes{
		lookPath: map[string]bool{"missing-tool": false},
	})

	ca, err := NewCustomAdapter("missing-tool", "missing-tool --update", "missing-tool --version", false)
	if err != nil {
		t.Fatal(err)
	}
	_, err = ca.Check(context.Background())
	if err == nil {
		t.Fatal("Check() expected error when binary missing, got nil")
	}
}

func TestCustomAdapter_Check_CheckTimeoutKills(t *testing.T) {
	setExecFakes(t, execFakes{
		lookPath: map[string]bool{"mytool": true},
		shell: map[string]fakeResult{
			"sleep 2": {err: context.DeadlineExceeded},
		},
	})

	ca, err := NewCustomAdapter("mytool", "mytool --update", "sleep 2", false)
	if err != nil {
		t.Fatal(err)
	}
	_, err = ca.Check(context.Background())
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Check() error = %v, want errors.Is(err, context.DeadlineExceeded)", err)
	}
}

func TestCustomAdapter_Update_DryRun(t *testing.T) {
	setExecFakes(t, execFakes{
		lookPath: map[string]bool{"mytool": true},
	})

	ca, err := NewCustomAdapter("mytool", "mytool --update", "", false)
	if err != nil {
		t.Fatal(err)
	}
	result, err := ca.Update(context.Background(), true) // dry run
	if err != nil {
		t.Fatalf("Update(dryRun=true) error = %v", err)
	}
	if !result.Success {
		t.Error("Update(dryRun=true) Success = false, want true")
	}
}

func TestCustomAdapter_Update_DryRun_Privileges(t *testing.T) {
	setExecFakes(t, execFakes{
		lookPath: map[string]bool{"sudo": true, "mytool": true},
	})

	ca, err := NewCustomAdapter("mytool", "sudo mytool --update", "", false)
	if err != nil {
		t.Fatal(err)
	}
	result, err := ca.Update(context.Background(), true)
	if err != nil {
		t.Fatalf("Update(dryRun=true) unexpected error = %v", err)
	}
	if !result.Success {
		t.Error("Update(dryRun=true) Success = false, want true")
	}
	if len(result.Privileges) != 1 || result.Privileges[0] != "sudo" {
		t.Errorf("Update(dryRun=true) Privileges = %v, want [sudo]", result.Privileges)
	}
	if result.Before != "sudo mytool --update" || result.After != "sudo mytool --update" {
		t.Errorf("Update(dryRun=true) Before/After mismatch: got (%q, %q)", result.Before, result.After)
	}
}

func TestCustomAdapter_Update_MissingBinary(t *testing.T) {
	setExecFakes(t, execFakes{
		lookPath: map[string]bool{"missing-tool": false},
	})

	ca, err := NewCustomAdapter("missing-tool", "missing-tool --update", "", false)
	if err != nil {
		t.Fatal(err)
	}
	result, err := ca.Update(context.Background(), false)
	if err != nil {
		t.Fatalf("Update() returned error %v, expected error inside Result", err)
	}
	if result.Success {
		t.Error("Update(dryRun=false) Success = true, want false when binary missing")
	}
	if result.Error == nil {
		t.Error("Update(dryRun=false) Result.Error = nil, want structured error when binary missing")
	}

	dryResult, err := ca.Update(context.Background(), true)
	if err != nil {
		t.Fatalf("Update(dryRun=true) returned error %v", err)
	}
	if dryResult.Success {
		t.Error("Update(dryRun=true) Success = true, want false when binary missing")
	}
	if dryResult.Error == nil {
		t.Error("Update(dryRun=true) Result.Error = nil, want structured error when binary missing")
	}
}

func TestCustomAdapter_Update_Execute(t *testing.T) {
	setExecFakes(t, execFakes{
		lookPath: map[string]bool{"echo": true},
		shell: map[string]fakeResult{
			"echo updated": {stdout: "updated", err: nil},
		},
	})

	ca, err := NewCustomAdapter("echo", "echo updated", "", false)
	if err != nil {
		t.Fatal(err)
	}
	result, err := ca.Update(context.Background(), false)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if !result.Success {
		t.Error("Update() Success = false, want true")
	}
}

func TestCustomAdapter_Update_Failure(t *testing.T) {
	setExecFakes(t, execFakes{
		lookPath: map[string]bool{"fail-cmd": true},
		shell: map[string]fakeResult{
			"fail-cmd": {stdout: "", stderr: "command failed with trace", err: errors.New("exit status 1")},
		},
	})

	ca, err := NewCustomAdapter("fail", "fail-cmd", "", false)
	if err != nil {
		t.Fatal(err)
	}
	result, err := ca.Update(context.Background(), false)
	if err != nil {
		t.Fatalf("Update() returned unexpected error: %v", err)
	}
	if result.Success {
		t.Error("Update() with failing command should return Success=false")
	}
	if result.Stderr != "command failed with trace" {
		t.Errorf("result.Stderr = %q, want %q", result.Stderr, "command failed with trace")
	}
}

func TestCustomAdapter_Privileges(t *testing.T) {
	setExecFakes(t, execFakes{
		lookPath: map[string]bool{"sudo": true, "mytool": true},
		shell: map[string]fakeResult{
			"sudo mytool --update": {stdout: "updated", err: nil},
		},
	})

	ca, err := NewCustomAdapter("mytool", "sudo mytool --update", "", false)
	if err != nil {
		t.Fatal(err)
	}
	result, err := ca.Update(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Success {
		t.Error("Update() Success = false, want true")
	}
	if len(result.Privileges) == 0 || result.Privileges[0] != "sudo" {
		t.Errorf("Update() Privileges = %v, want [sudo]", result.Privileges)
	}
}

// --- WU2: custom-tool manager delegation (spec Resolved Owner Update
// Delegation / Update Gating) ---
//
// A custom tool that carries a resolving owner manager MUST delegate its
// Update() to that manager and MUST NOT invoke its own command. The
// ManagerAdapter() accessor lets the CLI gate derive the effective policy.

// fakeManagerAdapter is a minimal manager stand-in for custom-tool delegation
// tests: it records whether Update was invoked, so an owned custom tool that
// (wrongly) ran its own command instead of delegating fails the assertion.
type fakeManagerAdapter struct {
	name    string
	updated bool
}

func (f *fakeManagerAdapter) Name() string { return f.name }
func (f *fakeManagerAdapter) Detect() bool { return true }
func (f *fakeManagerAdapter) Check(ctx context.Context) (UpdateInfo, error) {
	return UpdateInfo{CurrentVersion: "1.0.0", LatestVersion: "1.0.0", UpdateAvailable: false}, nil
}
func (f *fakeManagerAdapter) Update(ctx context.Context, dryRun bool) (Result, error) {
	f.updated = true
	return Result{Success: true, Before: "1.0.0", After: "1.1.0"}, nil
}
func (f *fakeManagerAdapter) Info() ToolInfo {
	return ToolInfo{ID: f.name, Name: f.name, UpdatePolicy: PolicyGated}
}

// TestCustomAdapter_VariadicManagerNil keeps the 4-arg form working: no
// manager passed => standalone custom tool, ManagerAdapter() returns nil.
func TestCustomAdapter_VariadicManagerNil(t *testing.T) {
	ca, err := NewCustomAdapter("mytool", "mytool --update", "", false)
	if err != nil {
		t.Fatal(err)
	}
	if ca.ManagerAdapter() != nil {
		t.Error("ManagerAdapter() = non-nil, want nil for a custom tool with no manager")
	}
}

// TestCustomAdapter_Update_DelegatesToManager proves an owned custom tool
// delegates its Update() to the manager: the manager's Update is invoked, the
// custom tool's own command is never run, and the result carries the manager's
// before/after versions.
func TestCustomAdapter_Update_DelegatesToManager(t *testing.T) {
	setExecFakes(t, execFakes{
		lookPath: map[string]bool{"brew": true, "mytool": true},
	})
	mgr := &fakeManagerAdapter{name: "brew"}
	ca, err := NewCustomAdapter("mytool", "mytool --update", "", false, mgr)
	if err != nil {
		t.Fatal(err)
	}

	result, err := ca.Update(context.Background(), false)
	if err != nil {
		t.Fatalf("Update() unexpected error: %v", err)
	}
	if !mgr.updated {
		t.Error("manager.Update() was not invoked — custom tool did not delegate")
	}
	if !result.Success {
		t.Error("Update() Success = false, want true (manager succeeded)")
	}
	if result.Before != "1.0.0" || result.After != "1.1.0" {
		t.Errorf("Update() Before/After = (%q, %q), want manager's (1.0.0, 1.1.0)", result.Before, result.After)
	}
}

// TestCustomAdapter_Info_ManagerAdapter exposes the manager so the CLI gate can
// resolve the effective policy on the delegated path.
func TestCustomAdapter_Info_ManagerAdapter(t *testing.T) {
	mgr := &fakeManagerAdapter{name: "brew"}
	ca, err := NewCustomAdapter("mytool", "mytool --update", "", false, mgr)
	if err != nil {
		t.Fatal(err)
	}
	if got := ca.ManagerAdapter(); got != mgr {
		t.Errorf("ManagerAdapter() = %v, want the injected manager %v", got, mgr)
	}
	if ca.Info().Kind != KindTool {
		t.Errorf("owned custom tool Info().Kind = %v, want KindTool", ca.Info().Kind)
	}
}

func TestExtractBaseCommand(t *testing.T) {
	tests := []struct {
		cmd  string
		want string
	}{
		{"mytool --update", "mytool"},
		{"sudo apt upgrade", "sudo"},
		{"echo hello", "echo"},
		{"", ""},
		{"   ", ""},
	}

	for _, tt := range tests {
		t.Run(tt.cmd, func(t *testing.T) {
			if got := extractBaseCommand(tt.cmd); got != tt.want {
				t.Errorf("extractBaseCommand(%q) = %q, want %q", tt.cmd, got, tt.want)
			}
		})
	}
}

func TestIsVersionLike(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"1.2.3", true},
		{"v1.2.3", true}, // leading "v" stripped, then valid version
		{"1.2", true},
		{"abc", false},
		{"", false},
		{"1.2.3-rc1", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			// Test the full extraction path
			got := isVersionLike(tt.input)
			if got != tt.want {
				t.Errorf("isVersionLike(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestShellExec(t *testing.T) {
	setExecFakes(t, execFakes{
		shell: map[string]fakeResult{
			"echo hello": {stdout: "hello", err: nil},
		},
	})

	stdout, _, err := shellExec(context.Background(), "echo hello")
	if err != nil {
		t.Fatalf("shellExec() error = %v", err)
	}
	if stdout != "hello" {
		t.Errorf("shellExec() stdout = %q, want %q", stdout, "hello")
	}
}

func TestCustomAdapter_Detect_WithRealCommand(t *testing.T) {
	setExecFakes(t, execFakes{
		lookPath: map[string]bool{
			"test-tool": true,
		},
	})

	ca, err := NewCustomAdapter("test-tool", "test-tool --update", "test-tool --version", false)
	if err != nil {
		t.Fatal(err)
	}
	if !ca.Detect() {
		t.Error("Detect() should find test-tool via lookPath")
	}
}

func TestShellExec_UpdateTimeoutKills(t *testing.T) {
	setExecFakes(t, execFakes{
		shell: map[string]fakeResult{
			"sleep 2": {err: context.DeadlineExceeded},
		},
	})

	_, _, err := shellExec(context.Background(), "sleep 2")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("shellExec() error = %v, want errors.Is(err, context.DeadlineExceeded)", err)
	}
}

type fakePackageCheckerUpdater struct {
	name           string
	checkedPackage string
	updatedPackage string
	checkResult    UpdateInfo
	updateResult   Result
}

func (f *fakePackageCheckerUpdater) Name() string { return f.name }
func (f *fakePackageCheckerUpdater) Detect() bool { return true }
func (f *fakePackageCheckerUpdater) Check(ctx context.Context) (UpdateInfo, error) {
	return UpdateInfo{}, nil
}
func (f *fakePackageCheckerUpdater) Update(ctx context.Context, dryRun bool) (Result, error) {
	return Result{Success: true}, nil
}
func (f *fakePackageCheckerUpdater) Info() ToolInfo {
	return ToolInfo{ID: f.name, Name: f.name, Kind: KindManager}
}
func (f *fakePackageCheckerUpdater) CheckPackage(ctx context.Context, packageName string) (UpdateInfo, error) {
	f.checkedPackage = packageName
	return f.checkResult, nil
}
func (f *fakePackageCheckerUpdater) UpdatePackage(ctx context.Context, packageName string) (Result, error) {
	f.updatedPackage = packageName
	return f.updateResult, nil
}

func TestCustomAdapter_Package_GetterSetter(t *testing.T) {
	ca, err := NewCustomAdapter("ripgrep", "rg --update", "", false)
	if err != nil {
		t.Fatal(err)
	}
	if ca.Package() != "ripgrep" {
		t.Errorf("Package() = %q, want default %q", ca.Package(), "ripgrep")
	}

	ca.SetPackage("rg-pkg")
	if ca.Package() != "rg-pkg" {
		t.Errorf("Package() = %q, want %q", ca.Package(), "rg-pkg")
	}

	ca.SetPackage("")
	if ca.Package() != "ripgrep" {
		t.Errorf("Package() = %q, want fallback %q", ca.Package(), "ripgrep")
	}

	ca.SetPackage("   ")
	if ca.Package() != "ripgrep" {
		t.Errorf("Package() = %q, want fallback %q when package is whitespace", ca.Package(), "ripgrep")
	}

	ca.SetPackage("  ripgrep-trimmed  ")
	if ca.Package() != "ripgrep-trimmed" {
		t.Errorf("Package() = %q, want %q", ca.Package(), "ripgrep-trimmed")
	}

	ca.WithPackage("rg-chained")
	if ca.Package() != "rg-chained" {
		t.Errorf("WithPackage() = %q, want %q", ca.Package(), "rg-chained")
	}

	ca2, err := NewCustomAdapterWithPackage("bat", "bat --update", "", "bat-extras", false)
	if err != nil {
		t.Fatal(err)
	}
	if ca2.Package() != "bat-extras" {
		t.Errorf("Package() = %q, want %q", ca2.Package(), "bat-extras")
	}

	ca3, err := NewCustomAdapterWithPackage("fd", "fd --update", "", "", false)
	if err != nil {
		t.Fatal(err)
	}
	if ca3.Package() != "fd" {
		t.Errorf("Package() = %q, want %q", ca3.Package(), "fd")
	}
}

func TestCustomAdapter_Check_DelegatesToPackageChecker(t *testing.T) {
	mgr := &fakePackageCheckerUpdater{
		name: "pacman",
		checkResult: UpdateInfo{
			CurrentVersion:  "13.0.0",
			LatestVersion:   "14.0.0",
			UpdateAvailable: true,
		},
	}
	ca, err := NewCustomAdapterWithPackage("rg", "rg --version", "", "ripgrep", false, mgr)
	if err != nil {
		t.Fatal(err)
	}

	info, err := ca.Check(context.Background())
	if err != nil {
		t.Fatalf("Check() error: %v", err)
	}
	if mgr.checkedPackage != "ripgrep" {
		t.Errorf("CheckPackage arg = %q, want %q", mgr.checkedPackage, "ripgrep")
	}
	if !info.UpdateAvailable || info.CurrentVersion != "13.0.0" || info.LatestVersion != "14.0.0" {
		t.Errorf("Check() returned %+v, want manager's checkResult", info)
	}
}

func TestCustomAdapter_Update_DelegatesToPackageUpdater(t *testing.T) {
	mgr := &fakePackageCheckerUpdater{
		name: "pacman",
		updateResult: Result{
			Success: true,
			Before:  "13.0.0",
			After:   "14.0.0",
		},
	}
	ca, err := NewCustomAdapterWithPackage("rg", "rg --version", "", "ripgrep", false, mgr)
	if err != nil {
		t.Fatal(err)
	}

	// Test dryRun == true: should not invoke UpdatePackage and should return Result with Before/After = c.command
	dryResult, err := ca.Update(context.Background(), true)
	if err != nil {
		t.Fatalf("Update(dryRun=true) error: %v", err)
	}
	if !dryResult.Success || dryResult.Before != "rg --version" || dryResult.After != "rg --version" {
		t.Errorf("Update(dryRun=true) = %+v, want Success=true and Before/After='rg --version'", dryResult)
	}
	if mgr.updatedPackage != "" {
		t.Errorf("Update(dryRun=true) must not call UpdatePackage, called with %q", mgr.updatedPackage)
	}

	// Test dryRun == false: should invoke UpdatePackage("ripgrep")
	res, err := ca.Update(context.Background(), false)
	if err != nil {
		t.Fatalf("Update(dryRun=false) error: %v", err)
	}
	if mgr.updatedPackage != "ripgrep" {
		t.Errorf("UpdatePackage arg = %q, want %q", mgr.updatedPackage, "ripgrep")
	}
	if !res.Success || res.Before != "13.0.0" || res.After != "14.0.0" {
		t.Errorf("Update(dryRun=false) = %+v, want manager's updateResult", res)
	}
}

func TestCustomAdapter_Info_ManagerAndPackage(t *testing.T) {
	mgr := &fakePackageCheckerUpdater{name: "pacman"}
	ca, err := NewCustomAdapterWithPackage("rg", "rg --update", "", "ripgrep", false, mgr)
	if err != nil {
		t.Fatal(err)
	}

	info := ca.Info()
	expectedPlatforms := []string{"linux", "macos", "windows"}
	if len(info.Platforms) != len(expectedPlatforms) {
		t.Fatalf("Platforms = %v, want %v", info.Platforms, expectedPlatforms)
	}
	for i, p := range expectedPlatforms {
		if info.Platforms[i] != p {
			t.Errorf("Platforms[%d] = %q, want %q", i, info.Platforms[i], p)
		}
	}

	if info.Manager["linux"] != "pacman" || info.Manager["macos"] != "pacman" || info.Manager["windows"] != "pacman" {
		t.Errorf("Manager map = %v, want pacman for all platforms", info.Manager)
	}
	if info.ManagerPackage["linux"] != "ripgrep" || info.ManagerPackage["macos"] != "ripgrep" || info.ManagerPackage["windows"] != "ripgrep" {
		t.Errorf("ManagerPackage map = %v, want ripgrep for all platforms", info.ManagerPackage)
	}

	// Standalone tool platforms
	standalone, err := NewCustomAdapter("tool", "tool --update", "", false)
	if err != nil {
		t.Fatal(err)
	}
	sInfo := standalone.Info()
	for _, p := range sInfo.Platforms {
		if p == "darwin" {
			t.Errorf("Platforms contains 'darwin', should be 'macos': %v", sInfo.Platforms)
		}
	}
}

type fakeNonPackageManager struct {
	name      string
	updateRan bool
	dryRunArg bool
}

func (f *fakeNonPackageManager) Name() string { return f.name }
func (f *fakeNonPackageManager) Detect() bool { return true }
func (f *fakeNonPackageManager) Check(ctx context.Context) (UpdateInfo, error) {
	return UpdateInfo{CurrentVersion: "1.0", LatestVersion: "1.0"}, nil
}
func (f *fakeNonPackageManager) Update(ctx context.Context, dryRun bool) (Result, error) {
	f.updateRan = true
	f.dryRunArg = dryRun
	return Result{Success: true, Before: "1.0", After: "2.0"}, nil
}
func (f *fakeNonPackageManager) Info() ToolInfo {
	return ToolInfo{ID: f.name, Name: f.name, Kind: KindManager, SelfUpdateCommand: f.name + " update " + f.name}
}

func TestCustomAdapter_ManagerFallback_NonPackageCheckerUpdater(t *testing.T) {
	mgr := &fakeNonPackageManager{name: "scoop"}
	ca, err := NewCustomAdapterWithPackage("mytool", "mytool --update", "", "mytool", false, mgr)
	if err != nil {
		t.Fatal(err)
	}

	// Check fallback: mgr does not implement PackageChecker and checkCmd is empty, returns empty UpdateInfo
	info, err := ca.Check(context.Background())
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if info.UpdateAvailable {
		t.Errorf("Check() UpdateAvailable = true, want false")
	}

	// Update fallback: mgr does not implement PackageUpdater, delegates to mgr.Update(ctx, dryRun)
	res, err := ca.Update(context.Background(), true)
	if err != nil {
		t.Fatalf("Update(dryRun=true) error = %v", err)
	}
	if !mgr.updateRan || !mgr.dryRunArg {
		t.Errorf("mgr.Update must be called with dryRun=true, got ran=%v dryRun=%v", mgr.updateRan, mgr.dryRunArg)
	}
	if !res.Success || res.Before != "1.0" || res.After != "2.0" {
		t.Errorf("Update(dryRun=true) = %+v, want manager's result", res)
	}
}
