package cli

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/JhnFrankz/upp/internal/uninstall"
)

func TestUninstall_DryRun(t *testing.T) {
	var buf bytes.Buffer
	gf := &GlobalFlags{}
	flags := UninstallFlags{DryRun: true}

	removeCalled := false
	removeAllCalled := false

	deps := uninstallDeps{
		execPath: func() (string, error) {
			return "/fake/bin/upp", nil
		},
		configDir: func() (string, error) {
			return "/fake/home/.config/upp", nil
		},
		cacheDir: func() (string, error) {
			return "/fake/home/.cache/upp", nil
		},
		discover: func(execPath, configDir, cacheDir string) ([]uninstall.Target, error) {
			return []uninstall.Target{
				{Type: uninstall.TargetBinary, Path: execPath, Exists: true},
				{Type: uninstall.TargetConfig, Path: configDir, Exists: true},
				{Type: uninstall.TargetCache, Path: cacheDir, Exists: true},
			}, nil
		},
		remove: func(path string) error {
			removeCalled = true
			return nil
		},
		removeAll: func(path string) error {
			removeAllCalled = true
			return nil
		},
	}

	err := runUninstall(gf, flags, &buf, deps)
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}

	if removeCalled || removeAllCalled {
		t.Errorf("remove should not be called under --dry-run")
	}

	out := buf.String()
	if !strings.Contains(out, "Dry run — no files will be removed") {
		t.Errorf("expected dry run header in output, got: %s", out)
	}
	if !strings.Contains(out, "Would remove binary: /fake/bin/upp") {
		t.Errorf("expected binary target in dry run output, got: %s", out)
	}
	if !strings.Contains(out, "Would remove config: /fake/home/.config/upp") {
		t.Errorf("expected config target in dry run output, got: %s", out)
	}
	if !strings.Contains(out, "Would remove cache: /fake/home/.cache/upp") {
		t.Errorf("expected cache target in dry run output, got: %s", out)
	}
}

func TestUninstall_FullSuccess(t *testing.T) {
	var buf bytes.Buffer
	gf := &GlobalFlags{}
	flags := UninstallFlags{DryRun: false}

	removed := make(map[string]bool)

	deps := uninstallDeps{
		execPath: func() (string, error) {
			return "/fake/bin/upp", nil
		},
		configDir: func() (string, error) {
			return "/fake/home/.config/upp", nil
		},
		cacheDir: func() (string, error) {
			return "/fake/home/.cache/upp", nil
		},
		discover: func(execPath, configDir, cacheDir string) ([]uninstall.Target, error) {
			return []uninstall.Target{
				{Type: uninstall.TargetBinary, Path: execPath, Exists: true},
				{Type: uninstall.TargetBackup, Path: "/fake/bin/upp.backup.1", Exists: true},
				{Type: uninstall.TargetConfig, Path: configDir, Exists: true},
			}, nil
		},
		remove: func(path string) error {
			removed[path] = true
			return nil
		},
		removeAll: func(path string) error {
			removed[path] = true
			return nil
		},
	}

	err := runUninstall(gf, flags, &buf, deps)
	if err != nil {
		t.Fatalf("expected nil error on success, got: %v", err)
	}

	if !removed["/fake/bin/upp"] || !removed["/fake/bin/upp.backup.1"] || !removed["/fake/home/.config/upp"] {
		t.Errorf("expected all targets to be removed, got removed map: %+v", removed)
	}

	out := buf.String()
	if !strings.Contains(out, "upp has been successfully uninstalled.") {
		t.Errorf("expected completion message, got: %s", out)
	}
}

func TestUninstall_PartialFailureWithWarnings(t *testing.T) {
	var buf bytes.Buffer
	gf := &GlobalFlags{}
	flags := UninstallFlags{DryRun: false}

	deps := uninstallDeps{
		execPath: func() (string, error) {
			return "/usr/local/bin/upp", nil
		},
		configDir: func() (string, error) {
			return "/fake/home/.config/upp", nil
		},
		cacheDir: func() (string, error) {
			return "/fake/home/.cache/upp", nil
		},
		discover: func(execPath, configDir, cacheDir string) ([]uninstall.Target, error) {
			return []uninstall.Target{
				{Type: uninstall.TargetBinary, Path: execPath, Exists: true},
				{Type: uninstall.TargetConfig, Path: configDir, Exists: true},
			}, nil
		},
		remove: func(path string) error {
			if path == "/usr/local/bin/upp" {
				return errors.New("permission denied")
			}
			return nil
		},
		removeAll: func(path string) error {
			return nil
		},
	}

	err := runUninstall(gf, flags, &buf, deps)
	if err == nil {
		t.Fatalf("expected non-zero error on partial failure, got nil")
	}

	out := buf.String()
	if !strings.Contains(out, "could not remove binary (/usr/local/bin/upp)") {
		t.Errorf("expected warning about binary, got: %s", out)
	}
	if !strings.Contains(out, "sudo rm -rf /usr/local/bin/upp") {
		t.Errorf("expected manual remediation suggestion, got: %s", out)
	}
}

func TestUninstall_QuietMode(t *testing.T) {
	var buf bytes.Buffer
	gf := &GlobalFlags{Quiet: true}
	flags := UninstallFlags{DryRun: false}

	deps := uninstallDeps{
		execPath: func() (string, error) {
			return "/fake/bin/upp", nil
		},
		configDir: func() (string, error) {
			return "/fake/home/.config/upp", nil
		},
		cacheDir: func() (string, error) {
			return "/fake/home/.cache/upp", nil
		},
		discover: func(execPath, configDir, cacheDir string) ([]uninstall.Target, error) {
			return []uninstall.Target{
				{Type: uninstall.TargetBinary, Path: execPath, Exists: true},
			}, nil
		},
		remove: func(path string) error {
			return nil
		},
		removeAll: func(path string) error {
			return nil
		},
	}

	err := runUninstall(gf, flags, &buf, deps)
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}

	out := buf.String()
	if strings.Contains(out, "Removed binary") {
		t.Errorf("expected quiet mode to suppress individual item removals, got: %s", out)
	}
	if !strings.Contains(out, "upp has been successfully uninstalled.") {
		t.Errorf("expected completion message, got: %s", out)
	}
}

func TestUninstall_CommandHelp(t *testing.T) {
	root, gf := BuildRoot()
	AddCommands(root, gf)

	cmd, _, err := root.Find([]string{"uninstall"})
	if err != nil {
		t.Fatalf("command 'uninstall' not found in root command: %v", err)
	}
	if cmd.GroupID != "maintenance" {
		t.Errorf("expected GroupID to be 'maintenance', got: %s", cmd.GroupID)
	}
	if cmd.Flags().Lookup("dry-run") == nil {
		t.Errorf("expected --dry-run flag to be registered")
	}
}
