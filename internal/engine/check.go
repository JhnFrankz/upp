package engine

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"

	"github.com/JhnFrankz/upp/internal/adapters"
)

// TimeoutErr maps a context deadline exceeded onto a structured error naming
// the tool, operation, and timeout limit. Non-timeout errors pass through
// unchanged; the %w chain preserves errors.Is detection.
func TimeoutErr(name, op string, err error) error {
	if !errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	limit := adapters.UpdateTimeout
	if op == "check" {
		limit = adapters.CheckTimeout
	}
	return fmt.Errorf("%s %s timed out after %s: %w", name, op, limit, err)
}

type checkJob struct {
	index   int
	adapter adapters.Adapter
}

// safeCheck runs Detect and Check on an adapter with panic containment.
func safeCheck(ctx context.Context, a adapters.Adapter) (oc CheckOutcome) {
	var toolID, toolName string
	defer func() {
		if rec := recover(); rec != nil {
			if toolName == "" {
				toolName = a.Name()
			}
			if toolID == "" {
				toolID = toolName
			}
			oc = CheckOutcome{
				ToolID:        toolID,
				ToolName:      toolName,
				Status:        StatusFailed,
				Err:           fmt.Errorf("panic during check: %v", rec),
				RawUpdateInfo: adapters.UpdateInfo{},
			}
		}
	}()

	info := a.Info()
	toolID = info.ID
	if toolID == "" {
		toolID = a.Name()
	}
	toolName = info.Name
	if toolName == "" {
		toolName = a.Name()
	}

	if ctx.Err() != nil {
		return CheckOutcome{
			ToolID:   toolID,
			ToolName: toolName,
			Status:   StatusFailed,
			Err:      ctx.Err(),
		}
	}

	if !a.Detect() {
		return CheckOutcome{
			ToolID:   toolID,
			ToolName: toolName,
			Status:   StatusSkipped,
		}
	}

	updateInfo, err := a.Check()
	if err != nil {
		wrappedErr := TimeoutErr(toolName, "check", err)
		return CheckOutcome{
			ToolID:   toolID,
			ToolName: toolName,
			Status:   StatusFailed,
			Err:      wrappedErr,
			Stderr:   err.Error(),
		}
	}

	if updateInfo.UpdateAvailable {
		return CheckOutcome{
			ToolID:          toolID,
			ToolName:        toolName,
			Status:          StatusAvailable,
			CurrentVersion:  updateInfo.CurrentVersion,
			LatestVersion:   updateInfo.LatestVersion,
			UpdateAvailable: true,
			RawUpdateInfo:   updateInfo,
		}
	}

	return CheckOutcome{
		ToolID:          toolID,
		ToolName:        toolName,
		Status:          StatusCurrent,
		CurrentVersion:  updateInfo.CurrentVersion,
		LatestVersion:   updateInfo.LatestVersion,
		UpdateAvailable: false,
		RawUpdateInfo:   updateInfo,
	}
}

// Check executes concurrent version checks across the provided adapters using a
// bounded worker pool, deferred panic recovery, deterministic index slotting,
// and cooperative context cancellation.
func (e *Engine) Check(ctx context.Context, adapterList []adapters.Adapter, onProgress func(CheckProgress)) ([]CheckOutcome, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	total := len(adapterList)
	outcomes := make([]CheckOutcome, total)
	if total == 0 {
		return outcomes, nil
	}

	workers := e.numWorkers
	if workers < 1 {
		workers = CalculateWorkerCount(runtime.NumCPU())
	}
	workerCount := min(workers, total)

	jobs := make(chan checkJob, total)
	for i, a := range adapterList {
		jobs <- checkJob{index: i, adapter: a}
	}
	close(jobs)

	var wg sync.WaitGroup

	for w := 0; w < workerCount; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}

				job, ok := <-jobs
				if !ok {
					return
				}

				oc := safeCheck(ctx, job.adapter)
				outcomes[job.index] = oc
				if onProgress != nil {
					onProgress(CheckProgress{
						Index:   job.index,
						Total:   total,
						Outcome: oc,
					})
				}
			}
		}()
	}

	wg.Wait()

	if ctx.Err() != nil {
		return outcomes, ctx.Err()
	}

	return outcomes, nil
}
