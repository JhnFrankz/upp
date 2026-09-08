package cli

import (
	"context"
	"fmt"
	"runtime"
	"sync"

	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/config"
	"github.com/JhnFrankz/upp/internal/engine"
	"github.com/JhnFrankz/upp/internal/output"
)

// outcomeToToolResult converts an engine.CheckOutcome to an output.ToolResult.
func outcomeToToolResult(oc engine.CheckOutcome) output.ToolResult {
	name := oc.ToolName
	if name == "" {
		name = oc.ToolID
	}
	res := output.ToolResult{
		Name: name,
	}
	switch oc.Status {
	case engine.StatusAvailable:
		res.Status = output.StatusAvailable
		res.Version = fmt.Sprintf("%s → %s", oc.CurrentVersion, oc.LatestVersion)
	case engine.StatusCurrent:
		res.Status = output.StatusCurrent
		res.Version = oc.CurrentVersion
	case engine.StatusSkipped:
		res.Status = output.StatusSkipped
	case engine.StatusFailed:
		res.Status = output.StatusFailed
		res.Error = oc.Err
		res.Stderr = oc.Stderr
	}
	return res
}

// calculateWorkerCount clamps concurrency to [4, 8] based on CPU cores.
func calculateWorkerCount(numCPU int) int {
	return engine.CalculateWorkerCount(numCPU)
}

// defaultConcurrency returns the clamped worker count for the current machine.
func defaultConcurrency() int {
	return engine.CalculateWorkerCount(runtime.NumCPU())
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
	eng := engine.New(nil, "")
	outcomes, _ := eng.Check(context.Background(), []adapters.Adapter{a}, nil)
	if len(outcomes) == 0 {
		return checkOutcome{}
	}
	engineOc := outcomes[0]
	return checkOutcome{
		result:     outcomeToToolResult(engineOc),
		updateInfo: engineOc.RawUpdateInfo,
	}
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
func runChecks(adapterList []adapters.Adapter, onResult func(index int, oc checkOutcome)) []checkOutcome {
	eng := engine.New(nil, "")
	total := len(adapterList)
	outcomes := make([]checkOutcome, total)

	var mu sync.Mutex
	_, _ = eng.Check(context.Background(), adapterList, func(prog engine.CheckProgress) {
		co := checkOutcome{
			result:     outcomeToToolResult(prog.Outcome),
			updateInfo: prog.Outcome.RawUpdateInfo,
		}
		mu.Lock()
		outcomes[prog.Index] = co
		mu.Unlock()
		if onResult != nil {
			onResult(prog.Index, co)
		}
	})

	return outcomes
}

// buildAdapterList creates adapters for enabled tools from the config.
func buildAdapterList(cfg *config.Config, osName string) []adapters.Adapter {
	eng := engine.New(cfg, osName)
	res, _ := eng.Resolve(engine.Filter{})
	return res
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
