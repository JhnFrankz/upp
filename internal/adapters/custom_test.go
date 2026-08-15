package adapters

import (
	"context"
	"errors"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestNewCustomAdapter_RequiresCommand(t *testing.T) {
	_, err := NewCustomAdapter("mytool", "", "", false)
	if err == nil {
		t.Error("NewCustomAdapter with empty command should return error")
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
	// echo is always on PATH.
	ca, err := NewCustomAdapter("echo", "echo hello", "", false)
	if err != nil {
		t.Fatal(err)
	}
	if !ca.Detect() {
		t.Error("Detect() = false for 'echo', want true")
	}
}

func TestCustomAdapter_Detect_NotFound(t *testing.T) {
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
	if runtime.GOOS == "windows" {
		t.Skip("skipping shell test on windows")
	}

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

func TestCustomAdapter_Update_DryRun(t *testing.T) {
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

func TestCustomAdapter_Update_Execute(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping shell test on windows")
	}

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
	ca, err := NewCustomAdapter("fail", "exit 1", "", false)
	if err != nil {
		t.Fatal(err)
	}
	result, err := ca.Update(false)
	if err != nil {
		t.Fatalf("Update() returned unexpected error: %v", err)
	}
	if result.Success {
		t.Error("Update() with 'exit 1' should return Success=false")
	}
}

func TestCustomAdapter_Privileges(t *testing.T) {
	ca, err := NewCustomAdapter("mytool", "sudo mytool --update", "", false)
	if err != nil {
		t.Fatal(err)
	}
	result, err := ca.Update(false)
	if err != nil {
		t.Fatal(err)
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
	if runtime.GOOS == "windows" {
		t.Skip("skipping shell test on windows")
	}

	stdout, err := shellExec("echo hello")
	if err != nil {
		t.Fatalf("shellExec() error = %v", err)
	}
	if stdout != "hello" {
		t.Errorf("shellExec() stdout = %q, want %q", stdout, "hello")
	}
}

func TestCustomAdapter_Detect_WithRealCommand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}

	// Create a temporary script in PATH.
	dir := t.TempDir()
	script := filepath.Join(dir, "test-tool")
	cmd := exec.Command("sh", "-c", "echo '#!/bin/sh\necho 1.0.0' > "+script+" && chmod +x "+script)
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}

	// Prepend temp dir to PATH so the test tool is discoverable.
	// exec.Command inherits env; we set PATH explicitly.
	t.Setenv("PATH", dir)

	ca, err := NewCustomAdapter("test-tool", "test-tool --update", "test-tool --version", false)
	if err != nil {
		t.Fatal(err)
	}
	if !ca.Detect() {
		t.Error("Detect() should find test-tool in temp PATH")
	}
}

func TestShellExec_UpdateTimeoutKills(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping shell test on windows")
	}

	orig := UpdateTimeout
	UpdateTimeout = 100 * time.Millisecond
	t.Cleanup(func() { UpdateTimeout = orig })

	_, err := shellExec("sleep 2")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("shellExec() error = %v, want errors.Is(err, context.DeadlineExceeded)", err)
	}
}

func TestCustomAdapter_Check_CheckTimeoutKills(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping shell test on windows")
	}

	orig := CheckTimeout
	CheckTimeout = 100 * time.Millisecond
	t.Cleanup(func() { CheckTimeout = orig })

	ca, err := NewCustomAdapter("mytool", "mytool --update", "sleep 2", false)
	if err != nil {
		t.Fatal(err)
	}
	_, err = ca.Check()
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Check() error = %v, want errors.Is(err, context.DeadlineExceeded)", err)
	}
}
