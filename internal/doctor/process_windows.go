//go:build windows

package doctor

import (
	"os"

	"golang.org/x/sys/windows"
)

func defaultProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	const da = windows.PROCESS_QUERY_LIMITED_INFORMATION
	h, err := windows.OpenProcess(da, false, uint32(pid))
	if err != nil {
		return false
	}
	defer windows.CloseHandle(h)

	var exitCode uint32
	err = windows.GetExitCodeProcess(h, &exitCode)
	if err != nil {
		return false
	}
	const stillActive = 259
	return exitCode == stillActive
}

func probeDirWrite(dir string) error {
	fi, err := os.Stat(dir)
	if err != nil {
		return err
	}
	if fi.Mode().Perm()&0o200 == 0 {
		return os.ErrPermission
	}
	pathp, err := windows.UTF16PtrFromString(dir)
	if err != nil {
		return err
	}
	h, err := windows.CreateFile(
		pathp,
		windows.FILE_WRITE_DATA|windows.FILE_APPEND_DATA,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_FLAG_BACKUP_SEMANTICS,
		0,
	)
	if err != nil {
		return err
	}
	_ = windows.CloseHandle(h)
	return nil
}
