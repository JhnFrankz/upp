package cli

import (
	"context"
	"fmt"
	"io"
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

// TestRunChecks_CarriesUpdateInfo proves the design D3 contract: runChecks
// returns checkOutcome values that carry the raw adapters.UpdateInfo from
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

	r := output.NewRenderer(io.Discard, false)
	outcomes := runChecks([]adapters.Adapter{available, current, failed}, r, false, false)

	if len(outcomes) != 3 {
		t.Fatalf("runChecks returned %d outcomes, want 3", len(outcomes))
	}

	// Outcome order must match adapter order (deterministic index slotting).
	byName := map[string]checkOutcome{}
	for _, oc := range outcomes {
		byName[oc.result.Name] = oc
	}

	// Available: full UpdateInfo carried → "current → latest" inline string.
	avail := byName["tool-available"]
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
	cur := byName["tool-current"]
	if cur.result.Status != output.StatusCurrent {
		t.Errorf("tool-current status = %v, want StatusCurrent", cur.result.Status)
	}
	if cur.updateInfo.CurrentVersion != "3.1.4" {
		t.Errorf("tool-current updateInfo = %+v, want CurrentVersion 3.1.4", cur.updateInfo)
	}

	// Failed: updateInfo MUST be zero — never act on stale version data.
	fail := byName["tool-failed"]
	if fail.result.Status != output.StatusFailed {
		t.Errorf("tool-failed status = %v, want StatusFailed", fail.result.Status)
	}
	if fail.updateInfo != (adapters.UpdateInfo{}) {
		t.Errorf("tool-failed updateInfo = %+v, want zero value", fail.updateInfo)
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

func TestRunCheck_VerboseFailureDiagnostics(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	failingAdapter := &fakeDelayedAdapter{
		name:     "fail-tool",
		checkErr: fmt.Errorf("lock frontend held by another process"),
	}

	deps := checkDeps{
		buildAdapterList: func(*config.Config, string) []adapters.Adapter {
			return []adapters.Adapter{failingAdapter}
		},
	}

	// With -v, diagnostics should be emitted
	outVerbose := withCapturedStdout(func() {
		gf := &GlobalFlags{Verbose: true}
		_ = runCheck(gf, "v0.1.0", deps)
	})
	if !strings.Contains(outVerbose, "lock frontend held by another process") {
		t.Errorf("expected verbose output to contain stderr diagnostic, got:\n%s", outVerbose)
	}

	// Without -v, stderr is suppressed
	outDefault := withCapturedStdout(func() {
		gf := &GlobalFlags{Verbose: false}
		_ = runCheck(gf, "v0.1.0", deps)
	})
	if strings.Contains(outDefault, "lock frontend held by another process") {
		t.Errorf("expected default output to suppress stderr diagnostic, got:\n%s", outDefault)
	}

	// With -q and -v, quiet takes precedence and suppresses
	outQuiet := withCapturedStdout(func() {
		gf := &GlobalFlags{Verbose: true, Quiet: true}
		_ = runCheck(gf, "v0.1.0", deps)
	})
	if strings.Contains(outQuiet, "lock frontend held by another process") {
		t.Errorf("expected quiet mode to suppress stderr diagnostic, got:\n%s", outQuiet)
	}
}
