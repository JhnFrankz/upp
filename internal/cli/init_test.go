package cli

import (
	"context"
	"strings"
	"testing"

	"github.com/JhnFrankz/upp/internal/lock"
)

func TestInit_LockAlreadyRunning(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	deps := initDeps{
		acquireLock: func() (*lock.Lock, error) {
			return nil, &lock.ErrAlreadyRunning{PID: 1234}
		},
	}

	err := runInit(context.Background(), &GlobalFlags{}, deps)
	if err == nil {
		t.Fatal("expected error when lock is already held, got nil")
	}
	if !strings.Contains(err.Error(), "another instance of upp is currently running") {
		t.Errorf("expected error containing 'another instance of upp is currently running', got: %v", err)
	}
}

func TestInit_LockAcquiredAndReleased(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	lockPath, err := lock.DefaultLockPath()
	if err != nil {
		t.Fatalf("DefaultLockPath: %v", err)
	}

	acquireCalled := false
	deps := initDeps{
		acquireLock: func() (*lock.Lock, error) {
			acquireCalled = true
			return lock.Acquire(lockPath)
		},
	}

	err = runInit(context.Background(), &GlobalFlags{CI: true}, deps)
	if err != nil {
		t.Fatalf("runInit unexpected error: %v", err)
	}
	if !acquireCalled {
		t.Fatal("acquireLock was not called")
	}

	// Verify the lock was released by defer — acquiring again should succeed.
	l2, err := lock.Acquire(lockPath)
	if err != nil {
		t.Fatalf("lock was not released after runInit returned: %v", err)
	}
	_ = l2.Release()
}
