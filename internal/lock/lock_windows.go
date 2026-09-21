//go:build windows

package lock

import (
	"errors"
	"os"

	"golang.org/x/sys/windows"
)

// lockOffset is positioned past the initial byte range where the PID is stored,
// so that other processes can read the PID without triggering a lock violation.
const lockOffset = 4096

func tryLock(f *os.File) error {
	var overlapped windows.Overlapped
	overlapped.Offset = lockOffset
	err := windows.LockFileEx(
		windows.Handle(f.Fd()),
		windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY,
		0,
		1,
		0,
		&overlapped,
	)
	if err != nil {
		if errors.Is(err, windows.ERROR_LOCK_VIOLATION) {
			return errLocked
		}
		return err
	}
	return nil
}

func unlock(f *os.File) error {
	var overlapped windows.Overlapped
	overlapped.Offset = lockOffset
	return windows.UnlockFileEx(
		windows.Handle(f.Fd()),
		0,
		1,
		0,
		&overlapped,
	)
}
