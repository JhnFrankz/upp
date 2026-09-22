//go:build windows

package doctor

import (
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
