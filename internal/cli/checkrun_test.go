package cli

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/output"
)

// fakePanickingAdapter panics during Check or Detect.
type fakePanickingAdapter struct {
	name        string
	panicDetect bool
	panicCheck  bool
}

func (f *fakePanickingAdapter) Name() string { return f.name }
func (f *fakePanickingAdapter) Detect() bool {
	if f.panicDetect {
		panic(fmt.Sprintf("%s panicked in Detect", f.name))
	}
	return true
}
func (f *fakePanickingAdapter) Check() (adapters.UpdateInfo, error) {
	if f.panicCheck {
		panic(fmt.Sprintf("%s panicked in Check", f.name))
	}
	return adapters.UpdateInfo{CurrentVersion: "1.0.0"}, nil
}
func (f *fakePanickingAdapter) Update(dryRun bool) (adapters.Result, error) {
	return adapters.Result{Success: true}, nil
}
func (f *fakePanickingAdapter) Info() adapters.ToolInfo {
	return adapters.ToolInfo{
		ID:           f.name,
		Name:         f.name,
		Trust:        adapters.TrustOfficial,
		UpdatePolicy: adapters.PolicyGated,
	}
}

// fakeDelayedAdapter introduces a controllable delay to test concurrency & ordering.
type fakeDelayedAdapter struct {
	name     string
	delay    time.Duration
	info     adapters.UpdateInfo
	checkErr error
}

func (f *fakeDelayedAdapter) Name() string { return f.name }
func (f *fakeDelayedAdapter) Detect() bool { return true }
func (f *fakeDelayedAdapter) Check() (adapters.UpdateInfo, error) {
	if f.delay > 0 {
		time.Sleep(f.delay)
	}
	return f.info, f.checkErr
}
func (f *fakeDelayedAdapter) Update(dryRun bool) (adapters.Result, error) {
	return adapters.Result{Success: true}, nil
}
func (f *fakeDelayedAdapter) Info() adapters.ToolInfo {
	return adapters.ToolInfo{
		ID:           f.name,
		Name:         f.name,
		Trust:        adapters.TrustOfficial,
		UpdatePolicy: adapters.PolicyGated,
	}
}

func TestCalculateWorkerCount_Clamping(t *testing.T) {
	tests := []struct {
		numCPU   int
		expected int
	}{
		{-1, 4},
		{0, 4},
		{1, 4},
		{2, 4},
		{3, 4},
		{4, 4},
		{6, 6},
		{8, 8},
		{9, 8},
		{16, 8},
		{64, 8},
	}

	for _, tt := range tests {
		got := calculateWorkerCount(tt.numCPU)
		if got != tt.expected {
			t.Errorf("calculateWorkerCount(%d) = %d, want %d", tt.numCPU, got, tt.expected)
		}
	}

	def := defaultConcurrency()
	if def < 4 || def > 8 {
		t.Errorf("defaultConcurrency() = %d, must be within [4, 8]", def)
	}
}

func TestSafeCheck_PanicRecovery(t *testing.T) {
	// Panic in Detect
	panicDetect := &fakePanickingAdapter{name: "panic-detect", panicDetect: true}
	resDetect := safeCheck(panicDetect).result
	if resDetect.Status != output.StatusFailed {
		t.Errorf("safeCheck(panicDetect) status = %v, want StatusFailed", resDetect.Status)
	}
	if resDetect.Error == nil || !strings.Contains(resDetect.Error.Error(), "panic during check") {
		t.Errorf("safeCheck(panicDetect) error = %v, want panic description", resDetect.Error)
	}

	// Panic in Check
	panicCheck := &fakePanickingAdapter{name: "panic-check", panicCheck: true}
	resCheck := safeCheck(panicCheck).result
	if resCheck.Status != output.StatusFailed {
		t.Errorf("safeCheck(panicCheck) status = %v, want StatusFailed", resCheck.Status)
	}
	if resCheck.Error == nil || !strings.Contains(resCheck.Error.Error(), "panic during check") {
		t.Errorf("safeCheck(panicCheck) error = %v, want panic description", resCheck.Error)
	}
}

func TestSafeCheck_TimeoutIsolation(t *testing.T) {
	timeoutAdapt := &fakeDelayedAdapter{
		name:     "slow-tool",
		checkErr: context.DeadlineExceeded,
	}
	res := safeCheck(timeoutAdapt).result
	if res.Status != output.StatusFailed {
		t.Errorf("safeCheck(timeout) status = %v, want StatusFailed", res.Status)
	}
	if res.Error == nil || !strings.Contains(res.Error.Error(), "timed out") {
		t.Errorf("safeCheck(timeout) error = %v, want timeout description", res.Error)
	}
}

// TestRunChecks_ReportsViaCallback proves the design D2 seam: runChecks
// invokes onResult exactly once per adapter with its deterministic slot
// index, and the reported outcome is identical to the one stored in the
// returned slice — callers (the future CheckBoard) can render completions
// live without re-deriving positions.
func TestRunChecks_ReportsViaCallback(t *testing.T) {
	tools := []*fakeDelayedAdapter{
		{name: "tool-current", info: adapters.UpdateInfo{CurrentVersion: "1.0.0"}},
		{name: "tool-available", info: adapters.UpdateInfo{UpdateAvailable: true, CurrentVersion: "1.0.0", LatestVersion: "2.0.0"}},
		{name: "tool-failed", checkErr: context.DeadlineExceeded},
	}
	adaptersIn := []adapters.Adapter{tools[0], tools[1], tools[2]}

	var mu sync.Mutex
	reported := map[int]checkOutcome{}
	outcomes := runChecks(adaptersIn, func(index int, oc checkOutcome) {
		mu.Lock()
		defer mu.Unlock()
		if _, dup := reported[index]; dup {
			t.Errorf("onResult called twice for index %d", index)
		}
		reported[index] = oc
	})

	if len(reported) != len(adaptersIn) {
		t.Fatalf("onResult fired for %d indexes, want %d", len(reported), len(adaptersIn))
	}

	indexes := make([]int, 0, len(reported))
	for idx := range reported {
		indexes = append(indexes, idx)
	}
	sort.Ints(indexes)
	for i, idx := range indexes {
		if idx != i {
			t.Fatalf("reported indexes = %v, want each index 0..%d exactly once", indexes, len(adaptersIn)-1)
		}
		if reported[idx] != outcomes[idx] {
			t.Errorf("onResult index %d reported %+v, want the slot outcome %+v", idx, reported[idx], outcomes[idx])
		}
	}

	if reported[0].result.Status != output.StatusCurrent {
		t.Errorf("index 0 status = %v, want StatusCurrent", reported[0].result.Status)
	}
	if reported[1].result.Status != output.StatusAvailable || reported[1].result.Version != "1.0.0 → 2.0.0" {
		t.Errorf("index 1 = %+v, want StatusAvailable with '1.0.0 → 2.0.0'", reported[1].result)
	}
	if reported[2].result.Status != output.StatusFailed {
		t.Errorf("index 2 status = %v, want StatusFailed", reported[2].result.Status)
	}
}

// TestRunChecks_NilCallbackSilent proves the nil-callback contract (design
// D2): passing no onResult is silent — no panic — while the returned
// outcomes stay complete and slot-aligned.
func TestRunChecks_NilCallbackSilent(t *testing.T) {
	solo := &fakeDelayedAdapter{name: "solo-tool", info: adapters.UpdateInfo{CurrentVersion: "9.9.9"}}

	outcomes := runChecks([]adapters.Adapter{solo}, nil)

	if len(outcomes) != 1 {
		t.Fatalf("runChecks returned %d outcomes, want 1", len(outcomes))
	}
	if outcomes[0].result.Status != output.StatusCurrent || outcomes[0].result.Name != "solo-tool" {
		t.Errorf("outcomes[0] = %+v, want current solo-tool result", outcomes[0].result)
	}
}

// TestRunChecks_CarriesUpdateInfo proves the design D3 contract: runChecks
// reports checkOutcome values that carry the raw adapters.UpdateInfo from
// Check() alongside the rendered ToolResult — the interactive update loop
// (Phase 3) needs the versions to render the selector without a second
// Check() call. The info MUST be zero when Detect or Check failed, so callers
// never act on stale version data.
func TestRunChecks_CarriesUpdateInfo(t *testing.T) {
	available := &fakeDelayedAdapter{
		name: "tool-available",
		info: adapters.UpdateInfo{
			UpdateAvailable: true,
			CurrentVersion:  "1.0.0",
			LatestVersion:   "2.0.0",
		},
	}
	current := &fakeDelayedAdapter{
		name: "tool-current",
		info: adapters.UpdateInfo{
			CurrentVersion: "3.1.4",
		},
	}
	failed := &fakeDelayedAdapter{
		name:     "tool-failed",
		checkErr: context.DeadlineExceeded,
	}

	captured := map[string]checkOutcome{}
	var mu sync.Mutex
	runChecks([]adapters.Adapter{available, current, failed}, func(_ int, oc checkOutcome) {
		mu.Lock()
		defer mu.Unlock()
		captured[oc.result.Name] = oc
	})

	if len(captured) != 3 {
		t.Fatalf("onResult captured %d outcomes, want 3", len(captured))
	}

	// Available: full UpdateInfo carried → "current → latest" inline string.
	avail := captured["tool-available"]
	if avail.result.Status != output.StatusAvailable {
		t.Errorf("tool-available status = %v, want StatusAvailable", avail.result.Status)
	}
	if !avail.updateInfo.UpdateAvailable {
		t.Error("tool-available updateInfo.UpdateAvailable = false, want true")
	}
	if avail.updateInfo.CurrentVersion != "1.0.0" || avail.updateInfo.LatestVersion != "2.0.0" {
		t.Errorf("tool-available updateInfo = %+v, want Current 1.0.0 / Latest 2.0.0", avail.updateInfo)
	}
	if want := "1.0.0 → 2.0.0"; avail.result.Version != want {
		t.Errorf("tool-available result.Version = %q, want %q", avail.result.Version, want)
	}

	// Current: UpdateInfo carried with the current version.
	cur := captured["tool-current"]
	if cur.result.Status != output.StatusCurrent {
		t.Errorf("tool-current status = %v, want StatusCurrent", cur.result.Status)
	}
	if cur.updateInfo.CurrentVersion != "3.1.4" {
		t.Errorf("tool-current updateInfo = %+v, want CurrentVersion 3.1.4", cur.updateInfo)
	}

	// Failed: updateInfo MUST be zero — never act on stale version data.
	fail := captured["tool-failed"]
	if fail.result.Status != output.StatusFailed {
		t.Errorf("tool-failed status = %v, want StatusFailed", fail.result.Status)
	}
	if fail.updateInfo != (adapters.UpdateInfo{}) {
		t.Errorf("tool-failed updateInfo = %+v, want zero value", fail.updateInfo)
	}
}
