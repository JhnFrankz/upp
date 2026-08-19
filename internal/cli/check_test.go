package cli

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/config"
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
	resDetect := safeCheck(panicDetect)
	if resDetect.Status != output.StatusFailed {
		t.Errorf("safeCheck(panicDetect) status = %v, want StatusFailed", resDetect.Status)
	}
	if resDetect.Error == nil || !strings.Contains(resDetect.Error.Error(), "panic during check") {
		t.Errorf("safeCheck(panicDetect) error = %v, want panic description", resDetect.Error)
	}

	// Panic in Check
	panicCheck := &fakePanickingAdapter{name: "panic-check", panicCheck: true}
	resCheck := safeCheck(panicCheck)
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
	res := safeCheck(timeoutAdapt)
	if res.Status != output.StatusFailed {
		t.Errorf("safeCheck(timeout) status = %v, want StatusFailed", res.Status)
	}
	if res.Error == nil || !strings.Contains(res.Error.Error(), "timed out") {
		t.Errorf("safeCheck(timeout) error = %v, want timeout description", res.Error)
	}
}

func TestRunCheck_Concurrent_OrderingAndIsolation(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Adapter 0: fast, up to date
	// Adapter 1: panics in Check
	// Adapter 2: slow with delay, available update
	// Adapter 3: timeout error
	// Adapter 4: up to date
	a0 := &fakeUpdateAdapter{name: "tool-0", policy: adapters.PolicyGated, trust: adapters.TrustOfficial, info: adapters.UpdateInfo{CurrentVersion: "1.0.0"}}
	a1 := &fakePanickingAdapter{name: "tool-1", panicCheck: true}
	a2 := &fakeDelayedAdapter{name: "tool-2", delay: 20 * time.Millisecond, info: adapters.UpdateInfo{UpdateAvailable: true, CurrentVersion: "1.0.0", LatestVersion: "1.2.0"}}
	a3 := &fakeDelayedAdapter{name: "tool-3", checkErr: context.DeadlineExceeded}
	a4 := &fakeUpdateAdapter{name: "tool-4", policy: adapters.PolicyGated, trust: adapters.TrustOfficial, info: adapters.UpdateInfo{CurrentVersion: "1.0.0"}}

	adapterListFunc := func(*config.Config, string) []adapters.Adapter {
		return []adapters.Adapter{a0, a1, a2, a3, a4}
	}

	setCLIDeps(t, checkDeps{buildAdapterList: adapterListFunc}, updateDeps{}, listDeps{}, selfUpdateDeps{})

	out := withCapturedStdout(func() {
		gf := &GlobalFlags{}
		err := runCheck(gf, "v0.1.0", cliDeps.check)
		if err != nil {
			t.Errorf("runCheck returned error: %v", err)
		}
	})

	// Verify all tools are represented in the output summary / details
	if !strings.Contains(out, "1 available") {
		t.Errorf("expected '1 available' in summary, got:\n%s", out)
	}
	if !strings.Contains(out, "2 up to date") {
		t.Errorf("expected '2 up to date' in summary, got:\n%s", out)
	}
	if !strings.Contains(out, "2 failed") {
		t.Errorf("expected '2 failed' in summary, got:\n%s", out)
	}
}
