package lock

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/JhnFrankz/upp/internal/config"
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
	once sync.Once
	err  error
}

// DefaultLockPath resolves upp.lock inside config.CacheDir().
func DefaultLockPath() (string, error) {
	dir, err := config.CacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "upp.lock"), nil
}

// Acquire attempts to acquire an exclusive advisory process lock on the given path.
// If the lock is already held by another process, it reads the PID from the lock file
// and returns &ErrAlreadyRunning{PID: pid}.
func Acquire(path string) (*Lock, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("cannot create lock directory: %w", err)
	}

	for attempts := 0; attempts < 3; attempts++ {
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

		// Inode verification: confirm that the file we locked is still the one at path.
		// If another process unlinked it between open and lock, retry on a fresh open.
		statF, errF := f.Stat()
		statP, errP := os.Stat(path)
		if errF == nil && errP == nil && !os.SameFile(statF, statP) {
			_ = unlock(f)
			_ = f.Close()
			continue
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
		if _, err := f.WriteString(strconv.Itoa(os.Getpid()) + "\n"); err != nil {
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

	return nil, fmt.Errorf("failed to acquire lock: file unlinked concurrently")
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
	if l == nil {
		return nil
	}
	l.once.Do(func() {
		if l.file == nil {
			return
		}
		// Unlink the path while still holding the lock on platforms that permit it,
		// preventing a race where another process locks the descriptor between unlock and remove.
		removeErr := os.Remove(l.path)
		unlockErr := unlock(l.file)
		closeErr := l.file.Close()
		l.file = nil
		if removeErr != nil && !os.IsNotExist(removeErr) {
			// On Windows, open files cannot be deleted until closed; retry after close.
			removeErr = os.Remove(l.path)
		}
		if unlockErr != nil {
			l.err = unlockErr
			return
		}
		if closeErr != nil {
			l.err = closeErr
			return
		}
		if removeErr != nil && !os.IsNotExist(removeErr) {
			l.err = removeErr
			return
		}
	})
	return l.err
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
