package engine

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/config"
)

// testDelayedAdapter introduces a controllable delay to test concurrency, ordering, and detection.
type testDelayedAdapter struct {
	id          string
	name        string
	delay       time.Duration
	info        adapters.UpdateInfo
	checkErr    error
	notDetected bool
	checkCalled atomic.Bool
}

func (a *testDelayedAdapter) Name() string {
	if a.name != "" {
		return a.name
	}
	return a.id
}

func (a *testDelayedAdapter) Detect() bool {
	return !a.notDetected
}

func (a *testDelayedAdapter) Check() (adapters.UpdateInfo, error) {
	a.checkCalled.Store(true)
	if a.delay > 0 {
		time.Sleep(a.delay)
	}
	return a.info, a.checkErr
}

func (a *testDelayedAdapter) Update(dryRun bool) (adapters.Result, error) {
	return adapters.Result{Success: true}, nil
}

func (a *testDelayedAdapter) Info() adapters.ToolInfo {
	return adapters.ToolInfo{
		ID:           a.id,
		Name:         a.Name(),
		Trust:        adapters.TrustOfficial,
		UpdatePolicy: adapters.PolicyGated,
	}
}

// testPanickingAdapter panics during Check or Detect to verify panic containment.
type testPanickingAdapter struct {
	id          string
	name        string
	panicDetect bool
	panicCheck  bool
}

func (a *testPanickingAdapter) Name() string {
	if a.name != "" {
		return a.name
	}
	return a.id
}

func (a *testPanickingAdapter) Detect() bool {
	if a.panicDetect {
		panic(fmt.Sprintf("%s panicked in Detect", a.Name()))
	}
	return true
}

func (a *testPanickingAdapter) Check() (adapters.UpdateInfo, error) {
	if a.panicCheck {
		panic(fmt.Sprintf("%s panicked in Check", a.Name()))
	}
	return adapters.UpdateInfo{CurrentVersion: "1.0.0"}, nil
}

func (a *testPanickingAdapter) Update(dryRun bool) (adapters.Result, error) {
	return adapters.Result{Success: true}, nil
}

func (a *testPanickingAdapter) Info() adapters.ToolInfo {
	return adapters.ToolInfo{
		ID:           a.id,
		Name:         a.Name(),
		Trust:        adapters.TrustOfficial,
		UpdatePolicy: adapters.PolicyGated,
	}
}

// testConcurrencyTrackingAdapter tracks peak active concurrent checks via atomic counter.
type testConcurrencyTrackingAdapter struct {
	id     string
	name   string
	delay  time.Duration
	active *int32
	peak   *int32
}

func (a *testConcurrencyTrackingAdapter) Name() string {
	if a.name != "" {
		return a.name
	}
	return a.id
}

func (a *testConcurrencyTrackingAdapter) Detect() bool {
	return true
}

func (a *testConcurrencyTrackingAdapter) Check() (adapters.UpdateInfo, error) {
	cur := atomic.AddInt32(a.active, 1)
	for {
		p := atomic.LoadInt32(a.peak)
		if cur <= p || atomic.CompareAndSwapInt32(a.peak, p, cur) {
			break
		}
	}
	if a.delay > 0 {
		time.Sleep(a.delay)
	}
	atomic.AddInt32(a.active, -1)
	return adapters.UpdateInfo{CurrentVersion: "1.0.0"}, nil
}

func (a *testConcurrencyTrackingAdapter) Update(dryRun bool) (adapters.Result, error) {
	return adapters.Result{Success: true}, nil
}

func (a *testConcurrencyTrackingAdapter) Info() adapters.ToolInfo {
	return adapters.ToolInfo{
		ID:           a.id,
		Name:         a.Name(),
		Trust:        adapters.TrustOfficial,
		UpdatePolicy: adapters.PolicyGated,
	}
}

func TestCheck_BoundedConcurrency(t *testing.T) {
	cfg := &config.Config{}
	var active, peak int32
	const totalAdapters = 12
	adapterList := make([]adapters.Adapter, totalAdapters)
	for i := 0; i < totalAdapters; i++ {
		adapterList[i] = &testConcurrencyTrackingAdapter{
			id:     fmt.Sprintf("tool-%02d", i),
			name:   fmt.Sprintf("Tool %02d", i),
			delay:  25 * time.Millisecond,
			active: &active,
			peak:   &peak,
		}
	}

	eng := New(cfg, "linux", WithConcurrency(4))
	outcomes, err := eng.Check(context.Background(), adapterList, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(outcomes) != totalAdapters {
		t.Fatalf("Check returned %d outcomes, want %d", len(outcomes), totalAdapters)
	}

	maxConcurrent := atomic.LoadInt32(&peak)
	if maxConcurrent > 4 {
		t.Errorf("peak concurrent checks = %d, want <= 4", maxConcurrent)
	}
	if maxConcurrent < 2 {
		t.Errorf("peak concurrent checks = %d, expected > 1 for concurrent execution", maxConcurrent)
	}
}

func TestCheck_PanicContainment_Detect(t *testing.T) {
	cfg := &config.Config{}
	panicDetect := &testPanickingAdapter{
		id:          "panic-detect",
		name:        "Panic Detect Tool",
		panicDetect: true,
	}
	normal := &testDelayedAdapter{
		id:    "normal-tool",
		name:  "Normal Tool",
		delay: 5 * time.Millisecond,
		info:  adapters.UpdateInfo{CurrentVersion: "1.0.0"},
	}

	eng := New(cfg, "linux")
	outcomes, err := eng.Check(context.Background(), []adapters.Adapter{panicDetect, normal}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(outcomes) != 2 {
		t.Fatalf("Check returned %d outcomes, want 2", len(outcomes))
	}

	// Panic in Detect
	panickedOutcome := outcomes[0]
	if panickedOutcome.Status != StatusFailed {
		t.Errorf("status = %v, want StatusFailed", panickedOutcome.Status)
	}
	if panickedOutcome.Err == nil || !strings.Contains(panickedOutcome.Err.Error(), "panic during check") {
		t.Errorf("err = %v, want description containing 'panic during check'", panickedOutcome.Err)
	}
	if panickedOutcome.RawUpdateInfo != (adapters.UpdateInfo{}) {
		t.Errorf("RawUpdateInfo = %+v, want zero value", panickedOutcome.RawUpdateInfo)
	}

	// Normal tool succeeds
	normalOutcome := outcomes[1]
	if normalOutcome.Status != StatusCurrent {
		t.Errorf("normal tool status = %v, want StatusCurrent", normalOutcome.Status)
	}
	if normalOutcome.Err != nil {
		t.Errorf("normal tool unexpected error = %v", normalOutcome.Err)
	}
}

func TestCheck_PanicContainment_Check(t *testing.T) {
	cfg := &config.Config{}
	panicCheck := &testPanickingAdapter{
		id:         "panic-check",
		name:       "Panic Check Tool",
		panicCheck: true,
	}

	eng := New(cfg, "linux")
	outcomes, err := eng.Check(context.Background(), []adapters.Adapter{panicCheck}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(outcomes) != 1 {
		t.Fatalf("Check returned %d outcomes, want 1", len(outcomes))
	}

	oc := outcomes[0]
	if oc.Status != StatusFailed {
		t.Errorf("status = %v, want StatusFailed", oc.Status)
	}
	if oc.Err == nil || !strings.Contains(oc.Err.Error(), "panic during check") {
		t.Errorf("err = %v, want description containing 'panic during check'", oc.Err)
	}
	if oc.RawUpdateInfo != (adapters.UpdateInfo{}) {
		t.Errorf("RawUpdateInfo = %+v, want zero value", oc.RawUpdateInfo)
	}
}

func TestCheck_DetectSkipped(t *testing.T) {
	cfg := &config.Config{}
	skipped := &testDelayedAdapter{
		id:          "skipped-tool",
		name:        "Skipped Tool",
		notDetected: true,
	}

	eng := New(cfg, "linux")
	outcomes, err := eng.Check(context.Background(), []adapters.Adapter{skipped}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(outcomes) != 1 {
		t.Fatalf("Check returned %d outcomes, want 1", len(outcomes))
	}

	oc := outcomes[0]
	if oc.Status != StatusSkipped {
		t.Errorf("status = %v, want StatusSkipped", oc.Status)
	}
	if skipped.checkCalled.Load() {
		t.Error("Check() was called on undetected adapter; want skipped")
	}
}

func TestCheck_TimeoutWrapping(t *testing.T) {
	cfg := &config.Config{}
	timeoutTool := &testDelayedAdapter{
		id:       "timeout-tool",
		name:     "Timeout Tool",
		checkErr: context.DeadlineExceeded,
	}

	eng := New(cfg, "linux")
	outcomes, err := eng.Check(context.Background(), []adapters.Adapter{timeoutTool}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(outcomes) != 1 {
		t.Fatalf("Check returned %d outcomes, want 1", len(outcomes))
	}

	oc := outcomes[0]
	if oc.Status != StatusFailed {
		t.Errorf("status = %v, want StatusFailed", oc.Status)
	}
	if !errors.Is(oc.Err, context.DeadlineExceeded) {
		t.Errorf("expected errors.Is context.DeadlineExceeded, got %v", oc.Err)
	}
	if !strings.Contains(oc.Err.Error(), "timed out after 30s") {
		t.Errorf("expected timeout message with 30s limit, got %q", oc.Err.Error())
	}
	if !strings.Contains(oc.Stderr, context.DeadlineExceeded.Error()) {
		t.Errorf("expected Stderr to contain %q, got %q", context.DeadlineExceeded.Error(), oc.Stderr)
	}
}

func TestCheck_DeterministicSlotting(t *testing.T) {
	cfg := &config.Config{}
	// Out-of-order completion: a2 finishes first (5ms), a1 second (25ms), a0 last (50ms)
	a0 := &testDelayedAdapter{id: "tool-0", name: "Tool 0", delay: 50 * time.Millisecond}
	a1 := &testDelayedAdapter{id: "tool-1", name: "Tool 1", delay: 25 * time.Millisecond}
	a2 := &testDelayedAdapter{id: "tool-2", name: "Tool 2", delay: 5 * time.Millisecond}

	eng := New(cfg, "linux", WithConcurrency(4))
	outcomes, err := eng.Check(context.Background(), []adapters.Adapter{a0, a1, a2}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(outcomes) != 3 {
		t.Fatalf("Check returned %d outcomes, want 3", len(outcomes))
	}

	if outcomes[0].ToolID != "tool-0" {
		t.Errorf("outcomes[0].ToolID = %q, want 'tool-0'", outcomes[0].ToolID)
	}
	if outcomes[1].ToolID != "tool-1" {
		t.Errorf("outcomes[1].ToolID = %q, want 'tool-1'", outcomes[1].ToolID)
	}
	if outcomes[2].ToolID != "tool-2" {
		t.Errorf("outcomes[2].ToolID = %q, want 'tool-2'", outcomes[2].ToolID)
	}
}

func TestCheck_ProgressCallback(t *testing.T) {
	cfg := &config.Config{}
	tools := []adapters.Adapter{
		&testDelayedAdapter{id: "t0", name: "T0", info: adapters.UpdateInfo{CurrentVersion: "1.0.0"}},
		&testDelayedAdapter{id: "t1", name: "T1", info: adapters.UpdateInfo{UpdateAvailable: true, CurrentVersion: "1.0.0", LatestVersion: "2.0.0"}},
		&testDelayedAdapter{id: "t2", name: "T2", checkErr: context.DeadlineExceeded},
	}

	var mu sync.Mutex
	reported := make(map[int]CheckProgress)

	eng := New(cfg, "linux")
	outcomes, err := eng.Check(context.Background(), tools, func(p CheckProgress) {
		mu.Lock()
		defer mu.Unlock()
		if _, exists := reported[p.Index]; exists {
			t.Errorf("progress callback invoked twice for index %d", p.Index)
		}
		reported[p.Index] = p
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(reported) != len(tools) {
		t.Fatalf("progress fired for %d tools, want %d", len(reported), len(tools))
	}

	for i := 0; i < len(tools); i++ {
		p, ok := reported[i]
		if !ok {
			t.Fatalf("missing progress for index %d", i)
		}
		if p.Total != len(tools) {
			t.Errorf("progress[%d].Total = %d, want %d", i, p.Total, len(tools))
		}
		if p.Index != i {
			t.Errorf("progress[%d].Index = %d, want %d", i, p.Index, i)
		}
		if p.Outcome.ToolID != outcomes[i].ToolID {
			t.Errorf("progress[%d].Outcome.ToolID = %q, want %q", i, p.Outcome.ToolID, outcomes[i].ToolID)
		}
		if p.Outcome.Status != outcomes[i].Status {
			t.Errorf("progress[%d].Outcome.Status = %v, want %v", i, p.Outcome.Status, outcomes[i].Status)
		}
	}
}

func TestCheck_NilCallbackSilent(t *testing.T) {
	cfg := &config.Config{}
	solo := &testDelayedAdapter{id: "solo", name: "Solo", info: adapters.UpdateInfo{CurrentVersion: "2.5.0"}}

	eng := New(cfg, "linux")
	outcomes, err := eng.Check(context.Background(), []adapters.Adapter{solo}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(outcomes) != 1 {
		t.Fatalf("Check returned %d outcomes, want 1", len(outcomes))
	}
	if outcomes[0].Status != StatusCurrent {
		t.Errorf("outcomes[0].Status = %v, want StatusCurrent", outcomes[0].Status)
	}
}

func TestCheck_ContextCancellation_Immediate(t *testing.T) {
	cfg := &config.Config{}
	tool := &testDelayedAdapter{id: "tool", name: "Tool", delay: 50 * time.Millisecond}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Pre-cancel

	eng := New(cfg, "linux")
	outcomes, err := eng.Check(ctx, []adapters.Adapter{tool}, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Check(canceledCtx) err = %v, want context.Canceled", err)
	}
	if outcomes != nil {
		t.Errorf("Check(canceledCtx) outcomes = %v, want nil", outcomes)
	}
	if tool.checkCalled.Load() {
		t.Error("tool.Check() was called on pre-canceled context; want immediate return without check")
	}
}

func TestCheck_ContextCancellation_MidFlight(t *testing.T) {
	cfg := &config.Config{}
	const totalAdapters = 10
	slowAdapters := make([]*testDelayedAdapter, totalAdapters)
	adapterList := make([]adapters.Adapter, totalAdapters)
	for i := 0; i < totalAdapters; i++ {
		slowAdapters[i] = &testDelayedAdapter{
			id:    fmt.Sprintf("slow-%02d", i),
			name:  fmt.Sprintf("Slow %02d", i),
			delay: 50 * time.Millisecond,
		}
		adapterList[i] = slowAdapters[i]
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var completedCount atomic.Int32
	eng := New(cfg, "linux", WithConcurrency(2))

	baseGoroutines := runtime.NumGoroutine()

	_, err := eng.Check(ctx, adapterList, func(p CheckProgress) {
		if completedCount.Add(1) == 2 {
			cancel() // Cancel mid-flight after 2 completions
		}
	})

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}

	// Verify workers discarded remaining queued tasks (not all 10 ran)
	var checksRan int
	for i := 0; i < totalAdapters; i++ {
		if slowAdapters[i].checkCalled.Load() {
			checksRan++
		}
	}
	if checksRan >= totalAdapters {
		t.Errorf("all %d checks ran; expected remaining queued jobs to be dropped on cancel", checksRan)
	}

	// Poll briefly to ensure all goroutines have cleaned up and no leak occurs
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if runtime.NumGoroutine() <= baseGoroutines+1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if current := runtime.NumGoroutine(); current > baseGoroutines+2 {
		t.Errorf("potential goroutine leak: base = %d, current = %d", baseGoroutines, current)
	}
}

func TestCheck_ContextDeadlineExceeded(t *testing.T) {
	cfg := &config.Config{}
	slow := &testDelayedAdapter{
		id:    "very-slow",
		name:  "Very Slow",
		delay: 150 * time.Millisecond,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	eng := New(cfg, "linux", WithConcurrency(2))
	_, err := eng.Check(ctx, []adapters.Adapter{slow}, nil)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Check(deadlineCtx) err = %v, want context.DeadlineExceeded", err)
	}
}

func TestTimeoutErr(t *testing.T) {
	t.Run("check deadline exceeded wraps with check timeout", func(t *testing.T) {
		err := TimeoutErr("npm", "check", context.DeadlineExceeded)
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("errors.Is(err, context.DeadlineExceeded) = false")
		}
		want := fmt.Sprintf("npm check timed out after %s: %v", adapters.CheckTimeout, context.DeadlineExceeded)
		if err.Error() != want {
			t.Errorf("TimeoutErr = %q, want %q", err.Error(), want)
		}
	})

	t.Run("update deadline exceeded wraps with update timeout", func(t *testing.T) {
		err := TimeoutErr("brew", "update", context.DeadlineExceeded)
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("errors.Is(err, context.DeadlineExceeded) = false")
		}
		want := fmt.Sprintf("brew update timed out after %s: %v", adapters.UpdateTimeout, context.DeadlineExceeded)
		if err.Error() != want {
			t.Errorf("TimeoutErr = %q, want %q", err.Error(), want)
		}
	})

	t.Run("non-timeout error passes through unchanged", func(t *testing.T) {
		original := errors.New("command not found")
		got := TimeoutErr("git", "check", original)
		if got != original {
			t.Errorf("TimeoutErr non-timeout = %v, want original %v", got, original)
		}
	})
}
