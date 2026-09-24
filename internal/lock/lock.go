package lock

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ErrLocked indicates that the file is currently locked by another process.
var ErrLocked = errors.New("file is locked")

var errLocked = ErrLocked

// ErrAlreadyRunning indicates that another instance of upp currently holds the process lock.
type ErrAlreadyRunning struct {
	PID int
}

func (e *ErrAlreadyRunning) Error() string {
	if e.PID > 0 {
		return fmt.Sprintf("another instance of upp is currently running (PID: %d)", e.PID)
	}
	return "another instance of upp is currently running"
}

// Lock represents an acquired advisory process lock.
type Lock struct {
	path string
	file *os.File
}

// DefaultLockPath resolves cacheDir/upp/upp.lock using os.UserCacheDir(),
// falling back to ~/.cache/upp/upp.lock if os.UserCacheDir() fails.
func DefaultLockPath() (string, error) {
	cacheRoot, err := os.UserCacheDir()
	if err != nil {
		home, herr := os.UserHomeDir()
		if herr != nil {
			return "", fmt.Errorf("cannot determine cache directory: %w", err)
		}
		return filepath.Join(home, ".cache", "upp", "upp.lock"), nil
	}
	return filepath.Join(cacheRoot, "upp", "upp.lock"), nil
}

// Acquire attempts to acquire an exclusive advisory process lock on the given path.
// If the lock is already held by another process, it reads the PID from the lock file
// and returns &ErrAlreadyRunning{PID: pid}.
func Acquire(path string) (*Lock, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("cannot create lock directory: %w", err)
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("cannot open lock file: %w", err)
	}

	if err := tryLock(f); err != nil {
		_ = f.Close()
		if errors.Is(err, errLocked) {
			pid := readPID(path)
			return nil, &ErrAlreadyRunning{PID: pid}
		}
		return nil, fmt.Errorf("failed to acquire lock: %w", err)
	}

	if err := f.Truncate(0); err != nil {
		_ = unlock(f)
		_ = f.Close()
		return nil, fmt.Errorf("failed to truncate lock file: %w", err)
	}
	if _, err := f.Seek(0, 0); err != nil {
		_ = unlock(f)
		_ = f.Close()
		return nil, fmt.Errorf("failed to seek lock file: %w", err)
	}
	if _, err := fmt.Fprintf(f, "%d\n", os.Getpid()); err != nil {
		_ = unlock(f)
		_ = f.Close()
		return nil, fmt.Errorf("failed to write PID to lock file: %w", err)
	}
	if err := f.Sync(); err != nil {
		_ = unlock(f)
		_ = f.Close()
		return nil, fmt.Errorf("failed to sync lock file: %w", err)
	}

	return &Lock{path: path, file: f}, nil
}

func readPID(path string) int {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	s := strings.TrimSpace(string(data))
	pid, err := strconv.Atoi(s)
	if err != nil || pid <= 0 {
		return 0
	}
	return pid
}

// Release releases the lock and closes the underlying file.
func (l *Lock) Release() error {
	if l == nil || l.file == nil {
		return nil
	}
	_ = os.Remove(l.path)
	unlockErr := unlock(l.file)
	closeErr := l.file.Close()
	l.file = nil
	_ = os.Remove(l.path)
	if unlockErr != nil {
		return unlockErr
	}
	return closeErr
}

// TryLock attempts to acquire an advisory lock on the file without blocking.
// If the file is already locked, it returns ErrLocked.
func TryLock(f *os.File) error {
	return tryLock(f)
}

// Unlock releases an advisory lock on the file.
func Unlock(f *os.File) error {
	return unlock(f)
}
