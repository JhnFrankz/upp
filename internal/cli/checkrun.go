package cli

import (
	"fmt"
	"runtime"
	"sync"

	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/adapters/official"
	"github.com/JhnFrankz/upp/internal/config"
	"github.com/JhnFrankz/upp/internal/output"
)

type checkJob struct {
	index   int
	adapter adapters.Adapter
}

// calculateWorkerCount clamps concurrency to [4, 8] based on CPU cores.
func calculateWorkerCount(numCPU int) int {
	if numCPU < 4 {
		return 4
	}
	if numCPU > 8 {
		return 8
	}
	return numCPU
}

// defaultConcurrency returns the clamped worker count for the current machine.
func defaultConcurrency() int {
	return calculateWorkerCount(runtime.NumCPU())
}

// checkOutcome pairs the rendered ToolResult with the raw adapters.UpdateInfo
// returned by Check(). Callers (interactive update pre-check) need the
// versions to render the selector without a second Check() call (design D3).
// updateInfo is the zero value whenever Detect or Check failed — never act
// on stale version data.
type checkOutcome struct {
	result     output.ToolResult
	updateInfo adapters.UpdateInfo
}

// safeCheck runs Detect and Check on an adapter with panic containment.
func safeCheck(a adapters.Adapter) (oc checkOutcome) {
	var name string
	defer func() {
		if rec := recover(); rec != nil {
			if name == "" {
				name = a.Name()
			}
			oc.result = output.ToolResult{
				Name:   name,
				Status: output.StatusFailed,
				Error:  fmt.Errorf("panic during check: %v", rec),
			}
			// updateInfo stays the zero value: a panicking check must never
			// carry version data forward.
		}
	}()

	info := a.Info()
	name = info.Name

	if !a.Detect() {
		oc.result = output.ToolResult{
			Name:   info.Name,
			Status: output.StatusSkipped,
		}
		return oc
	}

	updateInfo, err := a.Check()
	if err != nil {
		oc.result = output.ToolResult{
			Name:   info.Name,
			Status: output.StatusFailed,
			Error:  timeoutErr(info.Name, "check", err),
			Stderr: err.Error(),
		}
		// updateInfo stays the zero value on check failure.
		return oc
	}

	oc.updateInfo = updateInfo
	if updateInfo.UpdateAvailable {
		oc.result = output.ToolResult{
			Name:    info.Name,
			Status:  output.StatusAvailable,
			Version: fmt.Sprintf("%s → %s", updateInfo.CurrentVersion, updateInfo.LatestVersion),
		}
		return oc
	}

	oc.result = output.ToolResult{
		Name:    info.Name,
		Status:  output.StatusCurrent,
		Version: updateInfo.CurrentVersion,
	}
	return oc
}

// runChecks runs Detect + Check concurrently over the given adapters with a
// worker pool clamped to [4, 8] workers and deterministic index slotting, so
// the returned []checkOutcome is always in input order (design D3). It is
// shared by the interactive update pre-check and its tests.
//
// onResult is the completion seam (design D2): it fires once per adapter
// with that adapter's slot index and outcome, from the worker goroutine
// that produced it. Callers serialize their own state (the CheckBoard holds
// a mutex). A nil onResult runs silently. safeCheck guarantees reported
// outcomes never panic.
func runChecks(adapters []adapters.Adapter, onResult func(index int, oc checkOutcome)) []checkOutcome {
	total := len(adapters)
	outcomes := make([]checkOutcome, total)

	workerCount := defaultConcurrency()
	if workerCount > total {
		workerCount = total
	}

	jobs := make(chan checkJob, total)
	for i, a := range adapters {
		jobs <- checkJob{index: i, adapter: a}
	}
	close(jobs)

	var wg sync.WaitGroup

	for w := 0; w < workerCount; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				oc := safeCheck(job.adapter)
				outcomes[job.index] = oc
				if onResult != nil {
					onResult(job.index, oc)
				}
			}
		}()
	}

	wg.Wait()
	return outcomes
}

// buildAdapterList creates adapters for enabled tools from the config.
func buildAdapterList(cfg *config.Config, osName string) []adapters.Adapter {
	platformAdapters := official.AdaptersForPlatform(osName)
	var result []adapters.Adapter

	for _, a := range platformAdapters {
		info := a.Info()
		toolCfg, exists := cfg.Tools[info.ID]
		if exists && !toolCfg.Enabled {
			continue
		}
		result = append(result, a)
	}

	// Add custom adapters
	for id, custom := range cfg.Custom {
		// A custom tool MAY declare an owning manager (spec Config Format).
		// The config `manager` string is resolved HERE (in the CLI layer) to
		// an adapters.Adapter, because the adapters package must not import
		// the official registry (no import cycle). Only a known manager-kind
		// official tool (apt/brew/winget/scoop) is acceptable as an owner;
		// an unknown/non-manager value leaves the tool standalone — but config
		// Validate already cleared such a value, so this is a defensive guard.
		var managerArgs []adapters.Adapter
		if custom.Manager != "" {
			if mgr := official.AdapterByName(custom.Manager); mgr != nil && mgr.Info().Kind == adapters.KindManager {
				managerArgs = append(managerArgs, mgr)
			}
		}
		a, err := adapters.NewCustomAdapter(id, custom.Command, custom.CheckCmd, custom.Trusted, managerArgs...)
		if err != nil {
			continue
		}
		result = append(result, a)
	}

	return result
}

func adapterIDs(adapterList []adapters.Adapter) []string {
	var ids []string
	for _, a := range adapterList {
		ids = append(ids, a.Name())
	}
	return ids
}

func adapterByID(adapterList []adapters.Adapter) map[string]adapters.Adapter {
	m := make(map[string]adapters.Adapter, len(adapterList))
	for _, a := range adapterList {
		m[a.Name()] = a
	}
	return m
}
