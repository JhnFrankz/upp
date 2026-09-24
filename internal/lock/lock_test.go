package lock_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/JhnFrankz/upp/internal/lock"
)

func TestAcquireAndRelease(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "sub", "test.lock")

	l, err := lock.Acquire(lockPath)
	if err != nil {
		t.Fatalf("Acquire failed: %v", err)
	}
	if l == nil {
		t.Fatal("expected non-nil lock")
	}

	// Verify file exists
	if _, err := os.Stat(lockPath); err != nil {
		t.Fatalf("lock file does not exist: %v", err)
	}

	if err := l.Release(); err != nil {
		t.Fatalf("Release failed: %v", err)
	}

	// Calling release again should be a no-op / nil error
	if err := l.Release(); err != nil {
		t.Fatalf("second Release failed: %v", err)
	}
}

func TestAcquireWhileLocked(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "test.lock")

	l1, err := lock.Acquire(lockPath)
	if err != nil {
		t.Fatalf("first Acquire failed: %v", err)
	}
	defer func() { _ = l1.Release() }()

	// Try acquiring again with a different call (same process or simulate concurrent instance)
	// In Unix, flock locks per open file description, so opening the same file again and flocking will fail.
	l2, err := lock.Acquire(lockPath)
	if err == nil {
		_ = l2.Release()
		t.Fatal("expected error acquiring already-locked file, got nil")
	}

	var errRunning *lock.ErrAlreadyRunning
	if !errors.As(err, &errRunning) {
		t.Fatalf("expected *lock.ErrAlreadyRunning, got %T: %v", err, err)
	}

	if errRunning.PID != os.Getpid() {
		t.Fatalf("expected PID %d, got %d", os.Getpid(), errRunning.PID)
	}

	if errRunning.Error() == "" {
		t.Fatal("expected non-empty Error() string")
	}
}

func TestReacquireAfterRelease(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "test.lock")

	l1, err := lock.Acquire(lockPath)
	if err != nil {
		t.Fatalf("first Acquire failed: %v", err)
	}

	if err := l1.Release(); err != nil {
		t.Fatalf("Release failed: %v", err)
	}

	l2, err := lock.Acquire(lockPath)
	if err != nil {
		t.Fatalf("second Acquire failed: %v", err)
	}
	defer func() { _ = l2.Release() }()
}

func TestDefaultLockPath(t *testing.T) {
	p, err := lock.DefaultLockPath()
	if err != nil {
		t.Fatalf("DefaultLockPath failed: %v", err)
	}
	if p == "" {
		t.Fatal("expected non-empty default lock path")
	}
	if filepath.Base(p) != "upp.lock" {
		t.Fatalf("expected lock file named upp.lock, got %s", filepath.Base(p))
	}
}

func TestErrAlreadyRunningMessage(t *testing.T) {
	tests := []struct {
		name     string
		err      *lock.ErrAlreadyRunning
		expected string
	}{
		{
			name:     "with PID",
			err:      &lock.ErrAlreadyRunning{PID: 1234},
			expected: "another instance of upp is currently running (PID: 1234)",
		},
		{
			name:     "without PID",
			err:      &lock.ErrAlreadyRunning{PID: 0},
			expected: "another instance of upp is currently running",
		},
		{
			name:     "negative PID",
			err:      &lock.ErrAlreadyRunning{PID: -1},
			expected: "another instance of upp is currently running",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.err.Error(); got != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, got)
			}
		})
	}
}

func TestReleaseDeletesLockFile(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "test.lock")

	l, err := lock.Acquire(lockPath)
	if err != nil {
		t.Fatalf("Acquire failed: %v", err)
	}

	if _, err := os.Stat(lockPath); err != nil {
		t.Fatalf("expected lock file to exist, got: %v", err)
	}

	if err := l.Release(); err != nil {
		t.Fatalf("Release failed: %v", err)
	}

	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Fatalf("expected lock file to be removed after Release, but stat err was: %v", err)
	}
}
