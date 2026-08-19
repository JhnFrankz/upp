package adapters

import (
	"context"
	"errors"
	"testing"
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
	if info.Trust != TrustCustomUntrusted {
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
	if info.Trust != TrustCustomTrusted {
		t.Errorf("Info().Trust = %v, want TrustCustomTrusted (trusted=true must never map to Official)", info.Trust)
	}
	if info.Trust == TrustOfficial {
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
	info, err := ca.Check()
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
	info, err := ca.Check()
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
	_, err = ca.Check()
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
	_, err = ca.Check()
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
	result, err := ca.Update(true) // dry run
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
	result, err := ca.Update(true)
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
	result, err := ca.Update(false)
	if err != nil {
		t.Fatalf("Update() returned error %v, expected error inside Result", err)
	}
	if result.Success {
		t.Error("Update(dryRun=false) Success = true, want false when binary missing")
	}
	if result.Error == nil {
		t.Error("Update(dryRun=false) Result.Error = nil, want structured error when binary missing")
	}

	dryResult, err := ca.Update(true)
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
	result, err := ca.Update(false)
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
			"fail-cmd": {stdout: "", err: errors.New("exit status 1")},
		},
	})

	ca, err := NewCustomAdapter("fail", "fail-cmd", "", false)
	if err != nil {
		t.Fatal(err)
	}
	result, err := ca.Update(false)
	if err != nil {
		t.Fatalf("Update() returned unexpected error: %v", err)
	}
	if result.Success {
		t.Error("Update() with failing command should return Success=false")
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
	result, err := ca.Update(false)
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

	stdout, err := shellExec("echo hello")
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

	_, err := shellExec("sleep 2")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("shellExec() error = %v, want errors.Is(err, context.DeadlineExceeded)", err)
	}
}
