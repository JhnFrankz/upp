package uninstall_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/JhnFrankz/upp/internal/uninstall"
)

func TestDiscoverTargets_SymlinkAndBackups(t *testing.T) {
	tmpDir := t.TempDir()

	binDir := filepath.Join(tmpDir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir binDir: %v", err)
	}

	realBin := filepath.Join(binDir, "upp-real")
	if err := os.WriteFile(realBin, []byte("binary-content"), 0o755); err != nil {
		t.Fatalf("write realBin: %v", err)
	}

	symlinkBin := filepath.Join(binDir, "upp")
	if err := os.Symlink(realBin, symlinkBin); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	// Create backup files matching realBin name: upp-real.backup.20260812
	backup1 := filepath.Join(binDir, "upp-real.backup.20260812")
	if err := os.WriteFile(backup1, []byte("backup-1"), 0o755); err != nil {
		t.Fatalf("write backup1: %v", err)
	}
	backup2 := filepath.Join(binDir, "upp-real.backup.20260813")
	if err := os.WriteFile(backup2, []byte("backup-2"), 0o755); err != nil {
		t.Fatalf("write backup2: %v", err)
	}
	// Other unrelated file in same dir
	unrelated := filepath.Join(binDir, "other.txt")
	if err := os.WriteFile(unrelated, []byte("unrelated"), 0o644); err != nil {
		t.Fatalf("write unrelated: %v", err)
	}

	configDir := filepath.Join(tmpDir, "config", "upp")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir configDir: %v", err)
	}

	cacheDir := filepath.Join(tmpDir, "cache", "upp")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatalf("mkdir cacheDir: %v", err)
	}

	targets, err := uninstall.DiscoverTargets(symlinkBin, configDir, cacheDir)
	if err != nil {
		t.Fatalf("DiscoverTargets failed: %v", err)
	}

	// Should have:
	// 1 binary (resolved to realBin)
	// 2 backups
	// 1 config
	// 1 cache
	if len(targets) != 5 {
		t.Fatalf("expected 5 targets, got %d: %+v", len(targets), targets)
	}

	// Verify main binary resolved to realBin
	if targets[0].Type != uninstall.TargetBinary || targets[0].Path != realBin || !targets[0].Exists {
		t.Errorf("unexpected binary target: %+v", targets[0])
	}

	// Verify config and cache
	var foundConfig, foundCache bool
	backupCount := 0
	for _, target := range targets[1:] {
		switch target.Type {
		case uninstall.TargetBackup:
			backupCount++
			if target.Path != backup1 && target.Path != backup2 {
				t.Errorf("unexpected backup path: %s", target.Path)
			}
		case uninstall.TargetConfig:
			foundConfig = true
			if target.Path != configDir || !target.Exists {
				t.Errorf("unexpected config target: %+v", target)
			}
		case uninstall.TargetCache:
			foundCache = true
			if target.Path != cacheDir || !target.Exists {
				t.Errorf("unexpected cache target: %+v", target)
			}
		}
	}

	if backupCount != 2 {
		t.Errorf("expected 2 backups, got %d", backupCount)
	}
	if !foundConfig {
		t.Errorf("config target missing")
	}
	if !foundCache {
		t.Errorf("cache target missing")
	}
}

func TestDiscoverTargets_EmptyExecPath(t *testing.T) {
	_, err := uninstall.DiscoverTargets("", "", "")
	if err == nil {
		t.Errorf("expected error for empty execPath, got nil")
	}
}

func TestExecute_Success(t *testing.T) {
	tmpDir := t.TempDir()

	binPath := filepath.Join(tmpDir, "upp")
	if err := os.WriteFile(binPath, []byte("bin"), 0o755); err != nil {
		t.Fatalf("write bin: %v", err)
	}
	configDir := filepath.Join(tmpDir, "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir config: %v", err)
	}

	targets := []uninstall.Target{
		{Type: uninstall.TargetBinary, Path: binPath, Exists: true},
		{Type: uninstall.TargetConfig, Path: configDir, Exists: true},
		{Type: uninstall.TargetCache, Path: filepath.Join(tmpDir, "non-existent"), Exists: false},
	}

	errs := uninstall.Execute(targets, nil, nil)
	if len(errs) != 0 {
		t.Fatalf("expected 0 errors, got: %v", errs)
	}

	if _, err := os.Stat(binPath); !os.IsNotExist(err) {
		t.Errorf("expected binary to be removed")
	}
	if _, err := os.Stat(configDir); !os.IsNotExist(err) {
		t.Errorf("expected config to be removed")
	}
}

func TestExecute_BestEffortPartialFailure(t *testing.T) {
	targets := []uninstall.Target{
		{Type: uninstall.TargetBinary, Path: "/usr/local/bin/upp", Exists: true},
		{Type: uninstall.TargetConfig, Path: "/home/user/.config/upp", Exists: true},
	}

	mockRemove := func(path string) error {
		if path == "/usr/local/bin/upp" {
			return errors.New("permission denied")
		}
		return nil
	}
	mockRemoveAll := func(path string) error {
		return nil
	}

	errs := uninstall.Execute(targets, mockRemove, mockRemoveAll)
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d: %v", len(errs), errs)
	}

	if errs[0].Target.Path != "/usr/local/bin/upp" {
		t.Errorf("expected failure on binary path, got: %s", errs[0].Target.Path)
	}
}
