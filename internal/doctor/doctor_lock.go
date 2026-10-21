package doctor

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/JhnFrankz/upp/internal/lock"
)

func checkProcessLock(results []CheckResult, deps DoctorDeps) []CheckResult {
	lockPath, err := deps.LockPath()
	if err != nil {
		return append(results, CheckResult{
			Category: "Process Lock",
			Name:     "Process Lock",
			Status:   SeverityWarn,
			Message:  fmt.Sprintf("Failed to resolve lock path: %v", err),
		})
	}

	data, err := os.ReadFile(lockPath)
	if err != nil {
		if os.IsNotExist(err) {
			return append(results, CheckResult{
				Category: "Process Lock",
				Name:     "Process Lock",
				Status:   SeverityOK,
				Message:  "No active process lock",
				Detail:   lockPath,
			})
		}
		return append(results, CheckResult{
			Category: "Process Lock",
			Name:     "Process Lock",
			Status:   SeverityWarn,
			Message:  fmt.Sprintf("Cannot read lock file: %v", err),
			Detail:   lockPath,
			FixHint:  fmt.Sprintf("Remove inaccessible lock file: rm %s", lockPath),
		})
	}

	s := strings.TrimSpace(string(data))
	pid, perr := strconv.Atoi(s)
	if perr != nil || pid <= 0 {
		return append(results, CheckResult{
			Category: "Process Lock",
			Name:     "Process Lock",
			Status:   SeverityWarn,
			Message:  "Corrupted lock file found",
			Detail:   lockPath,
			FixHint:  fmt.Sprintf("Remove corrupted lock file: rm %s", lockPath),
		})
	}

	f, oerr := os.OpenFile(lockPath, os.O_RDWR, 0o600)
	if oerr != nil {
		return append(results, CheckResult{
			Category: "Process Lock",
			Name:     "Process Lock",
			Status:   SeverityWarn,
			Message:  fmt.Sprintf("Cannot open lock file: %v", oerr),
			Detail:   lockPath,
			FixHint:  fmt.Sprintf("Remove inaccessible lock file: rm %s", lockPath),
		})
	}

	tryLock := deps.TryLock
	if tryLock == nil {
		tryLock = lock.TryLock
	}

	lockErr := tryLock(f)
	if errors.Is(lockErr, lock.ErrLocked) {
		_ = f.Close()
		return append(results, CheckResult{
			Category: "Process Lock",
			Name:     "Process Lock",
			Status:   SeverityWarn,
			Message:  fmt.Sprintf("Lock held by active process (PID: %d)", pid),
			Detail:   lockPath,
			FixHint:  fmt.Sprintf("Wait for running upp process (PID %d) to finish or terminate it if stuck.", pid),
		})
	}
	if lockErr != nil {
		_ = f.Close()
		return append(results, CheckResult{
			Category: "Process Lock",
			Name:     "Process Lock",
			Status:   SeverityWarn,
			Message:  fmt.Sprintf("Cannot test lock on file: %v", lockErr),
			Detail:   lockPath,
			FixHint:  fmt.Sprintf("Check permissions or remove lock file: rm %s", lockPath),
		})
	}

	_ = lock.Unlock(f)
	_ = f.Close()

	if !deps.ProcessAlive(pid) {
		return append(results, CheckResult{
			Category: "Process Lock",
			Name:     "Process Lock",
			Status:   SeverityWarn,
			Message:  fmt.Sprintf("Stale lock file detected (PID: %d is dead)", pid),
			Detail:   lockPath,
			FixHint:  fmt.Sprintf("Remove stale lock file: rm %s", lockPath),
		})
	}

	return append(results, CheckResult{
		Category: "Process Lock",
		Name:     "Process Lock",
		Status:   SeverityWarn,
		Message:  fmt.Sprintf("Stale lock file detected (PID: %d does not hold lock)", pid),
		Detail:   lockPath,
		FixHint:  fmt.Sprintf("Remove stale lock file: rm %s", lockPath),
	})
}
