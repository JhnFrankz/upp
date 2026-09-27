package engine

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/config"
	"github.com/JhnFrankz/upp/internal/security"
)

type testResilienceAdapter struct {
	id      string
	name    string
	checkFn func(ctx context.Context) (adapters.UpdateInfo, error)
}

func (a *testResilienceAdapter) Name() string {
	if a.name != "" {
		return a.name
	}
	return a.id
}

func (a *testResilienceAdapter) Detect() bool {
	return true
}

func (a *testResilienceAdapter) Check(ctx context.Context) (adapters.UpdateInfo, error) {
	if a.checkFn != nil {
		return a.checkFn(ctx)
	}
	return adapters.UpdateInfo{CurrentVersion: "1.0.0"}, nil
}

func (a *testResilienceAdapter) Update(ctx context.Context, dryRun bool) (adapters.Result, error) {
	return adapters.Result{Success: true}, nil
}

func (a *testResilienceAdapter) Info() adapters.ToolInfo {
	return adapters.ToolInfo{
		ID:           a.id,
		Name:         a.Name(),
		Trust:        security.TrustOfficial,
		UpdatePolicy: adapters.PolicyGated,
	}
}

func TestEngine_Check_GoroutineLeakOnCancellation(t *testing.T) {
	cfg := &config.Config{}
	eng := New(cfg, "linux", WithWorkerCount(16))

	const totalAdapters = 50
	var startedCount atomic.Int32
	mockAdapters := make([]adapters.Adapter, totalAdapters)
	for i := 0; i < totalAdapters; i++ {
		mockAdapters[i] = &testResilienceAdapter{
			id:   fmt.Sprintf("tool-%02d", i),
			name: fmt.Sprintf("Tool %02d", i),
			checkFn: func(ctx context.Context) (adapters.UpdateInfo, error) {
				startedCount.Add(1)
				select {
				case <-ctx.Done():
					return adapters.UpdateInfo{}, ctx.Err()
				case <-time.After(500 * time.Millisecond):
					return adapters.UpdateInfo{CurrentVersion: "1.0.0"}, nil
				}
			},
		}
	}

	time.Sleep(10 * time.Millisecond)
	initialGoroutines := runtime.NumGoroutine()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var progressCount atomic.Int32
	onProgress := func(p CheckProgress) {
		progressCount.Add(1)
	}

	type checkResult struct {
		outcomes []CheckOutcome
		err      error
	}
	resCh := make(chan checkResult, 1)

	go func() {
		outcomes, err := eng.Check(ctx, mockAdapters, onProgress)
		resCh <- checkResult{outcomes: outcomes, err: err}
	}()

	// Wait until at least several checks have started or 10ms have elapsed, then call cancel()
	start := time.Now()
	for startedCount.Load() < 4 && time.Since(start) < 10*time.Millisecond {
		time.Sleep(1 * time.Millisecond)
	}
	cancel()

	var res checkResult
	select {
	case res = <-resCh:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for eng.Check to return after cancel")
	}

	if !errors.Is(res.err, context.Canceled) {
		t.Fatalf("expected error matching context.Canceled, got %v", res.err)
	}

	// Poll/wait up to 500ms for goroutines to drain: verify runtime.NumGoroutine() <= initialGoroutines + 1
	deadline := time.Now().Add(500 * time.Millisecond)
	var finalGoroutines int
	drained := false
	for time.Now().Before(deadline) {
		runtime.Gosched()
		finalGoroutines = runtime.NumGoroutine()
		if finalGoroutines <= initialGoroutines+1 {
			drained = true
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !drained {
		t.Fatalf("goroutine leak detected: initial=%d, final=%d (expected <= %d)",
			initialGoroutines, finalGoroutines, initialGoroutines+1)
	}
}

func TestEngine_Check_ConcurrentPanicRecovery(t *testing.T) {
	cfg := &config.Config{}
	eng := New(cfg, "linux", WithWorkerCount(8))

	const totalAdapters = 20
	type adapterBehavior int
	const (
		behaviorPanic adapterBehavior = iota
		behaviorSuccess
		behaviorError
	)

	behaviors := make([]adapterBehavior, totalAdapters)
	for i := 0; i < 5; i++ {
		behaviors[i] = behaviorPanic
	}
	for i := 5; i < 15; i++ {
		behaviors[i] = behaviorSuccess
	}
	for i := 15; i < 20; i++ {
		behaviors[i] = behaviorError
	}

	r := rand.New(rand.NewSource(42))
	r.Shuffle(len(behaviors), func(i, j int) {
		behaviors[i], behaviors[j] = behaviors[j], behaviors[i]
	})

	mockAdapters := make([]adapters.Adapter, totalAdapters)
	for i := 0; i < totalAdapters; i++ {
		idx := i
		b := behaviors[i]
		mockAdapters[i] = &testResilienceAdapter{
			id:   fmt.Sprintf("tool-%02d", idx),
			name: fmt.Sprintf("Tool %02d", idx),
			checkFn: func(ctx context.Context) (adapters.UpdateInfo, error) {
				// Introduce minor microsecond jitter so workers execute concurrently
				time.Sleep(time.Duration(10+(idx%5)*10) * time.Microsecond)
				switch b {
				case behaviorPanic:
					panic("simulated adapter failure")
				case behaviorError:
					return adapters.UpdateInfo{}, errors.New("simulated normal error")
				case behaviorSuccess:
					return adapters.UpdateInfo{
						CurrentVersion:  "1.0.0",
						LatestVersion:   "1.1.0",
						UpdateAvailable: true,
					}, nil
				default:
					return adapters.UpdateInfo{CurrentVersion: "1.0.0"}, nil
				}
			},
		}
	}

	time.Sleep(10 * time.Millisecond)
	initialGoroutines := runtime.NumGoroutine()

	outcomes, err := eng.Check(context.Background(), mockAdapters, nil)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if len(outcomes) != totalAdapters {
		t.Fatalf("expected %d outcomes, got %d", totalAdapters, len(outcomes))
	}

	for i := 0; i < totalAdapters; i++ {
		oc := outcomes[i]
		expectedID := fmt.Sprintf("tool-%02d", i)
		if oc.ToolID != expectedID {
			t.Errorf("outcome[%d].ToolID = %q, want %q", i, oc.ToolID, expectedID)
		}

		switch behaviors[i] {
		case behaviorPanic:
			if oc.Status != StatusFailed {
				t.Errorf("outcome[%d] panicked adapter status = %v, want %v", i, oc.Status, StatusFailed)
			}
			if oc.Err == nil || !strings.Contains(oc.Err.Error(), "panic during check") {
				t.Errorf("outcome[%d] panicked adapter err = %v, want 'panic during check'", i, oc.Err)
			}
			if oc.Err == nil || !strings.Contains(oc.Err.Error(), "simulated adapter failure") {
				t.Errorf("outcome[%d] panicked adapter err = %v, want to contain 'simulated adapter failure'", i, oc.Err)
			}
		case behaviorSuccess:
			if oc.Status != StatusCurrent && oc.Status != StatusAvailable {
				t.Errorf("outcome[%d] successful adapter status = %v, want StatusCurrent or StatusAvailable", i, oc.Status)
			}
			if oc.Err != nil {
				t.Errorf("outcome[%d] successful adapter err = %v, want nil", i, oc.Err)
			}
		case behaviorError:
			if oc.Status != StatusFailed {
				t.Errorf("outcome[%d] error adapter status = %v, want %v", i, oc.Status, StatusFailed)
			}
			if oc.Err == nil || strings.Contains(oc.Err.Error(), "panic during check") {
				t.Errorf("outcome[%d] error adapter err = %v, should not be a panic error", i, oc.Err)
			}
			if oc.Err == nil || !strings.Contains(oc.Err.Error(), "simulated normal error") {
				t.Errorf("outcome[%d] error adapter err = %v, want 'simulated normal error'", i, oc.Err)
			}
		}
	}

	// Poll/wait up to 500ms for goroutines to drain: verify no goroutines leak
	deadline := time.Now().Add(500 * time.Millisecond)
	var finalGoroutines int
	drained := false
	for time.Now().Before(deadline) {
		runtime.Gosched()
		finalGoroutines = runtime.NumGoroutine()
		if finalGoroutines <= initialGoroutines+1 {
			drained = true
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !drained {
		t.Fatalf("goroutine leak detected after concurrent panics: initial=%d, final=%d (expected <= %d)",
			initialGoroutines, finalGoroutines, initialGoroutines+1)
	}
}
