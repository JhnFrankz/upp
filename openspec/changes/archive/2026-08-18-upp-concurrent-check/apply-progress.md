# Apply Progress: upp-concurrent-check

**Change**: upp-concurrent-check · **Phase**: apply · **Date**: 2026-08-18
**Mode**: Strict TDD (red/green/refactor/verify gates executed) · **Delivery**: single-pr, stacked-to-main · **Review budget**: ~150–200 changed lines.

## Decisions Applied

- **D1 (Automatic Bounded Worker Pool)**: Concurrency bounded dynamically to $\min(8, \max(4, \text{runtime.NumCPU()}))$, with zero configuration flags required.
- **D2 (Deterministic Result Ordering via Index Slotting)**: Pre-allocated `results := make([]output.ToolResult, total)` indexed by canonical discovery order `job.index`, eliminating race conditions and sorting overhead.
- **D3 (Single-Line In-Place Progress Rendering)**: Mutex-synchronized `Renderer.mu` and atomic counter (`completed.Add(1)`). In interactive TTYs, renders single-line in-place updates with `\r`. In non-TTY/CI environments, falls back to clean line-buffered output.
- **D4 (Per-Tool Panic & Timeout Isolation)**: `safeCheck` wraps each adapter execution in deferred `recover()` and structured timeout handling, mapping errors to `output.StatusFailed` so single adapter faults never break the pool.

## Tasks Progress (all complete)

### Phase 1: Output Synchronization & In-Place Progress
- [x] 1.1 RED: `internal/output/render_test.go` — Add `TestRenderer_ConcurrentProgress_ThreadSafe` verifying concurrent progress updates do not panic, corrupt output, or cause data races under `-race`.
- [x] 1.2 GREEN: `internal/output/render.go` — Add `sync.Mutex` (`mu`) to `Renderer` struct to synchronize `Progress()` and implement `ProgressInPlace(op string, current, total int, name string)` using `\r` for interactive TTYs with clean fallback for non-TTY/CI.

### Phase 2: Concurrent Worker Pool Engine
- [x] 2.1 RED: `internal/cli/check_test.go` — Add `TestRunCheck_Concurrent_OrderingAndIsolation` testing worker pool bounds clamping (`[4, 8]`), mock adapter panic recovery returning `output.StatusFailed`, and timeout isolation.
- [x] 2.2 GREEN: `internal/cli/check.go` — Implement `calculateWorkerCount()` clamping to `[4, 8]`, worker pool goroutines with `checkJob` channels, deferred `recover()` panic containment in `safeCheck()`, atomic progress tracking, and direct index slotting into pre-allocated `results` slice.

### Phase 3: Order Determinism & Integration Guards
- [x] 3.1 RED/GREEN: `internal/cli/integration_test.go` — Add `TestCheck_DeterministicOrderUnderConcurrency` with adapters completing with varied/reverse delays, verifying summary output always preserves canonical tool discovery sequence.
- [x] 3.2 VERIFY: Run `go test ./... -count=1 -race` ensuring zero race conditions across CLI and output packages.

### Phase 4: Full Suite Verification
- [x] 4.1 VERIFY: Run `go test ./... -count=1 -race` for full workspace unit and race test suite.
- [x] 4.2 VERIFY: Run `go vet ./...` and `gofmt -s -w internal/cli/ internal/output/`.
- [x] 4.3 VERIFY: Run `bash scripts/smoke-test.sh --skip-build` to validate CLI end-to-end functionality.

## TDD Cycle Evidence

| Work Unit | Task | Tests | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|---|---|---|---|---|---|---|---|---|
| **WU-1 Output Sync & Progress** | 1.1, 1.2 | `render_test.go` (`TestRenderer_ConcurrentProgress_ThreadSafe`, `TestProgressInPlace_TTY`, `TestProgressInPlace_NonTTY`, `TestProgressInPlace_SingleTool`, `TestProgressInPlace_Quiet`) | Unit | ✅ baseline unit tests pass | ✅ compile error (`r.ProgressInPlace undefined`) | ✅ pass (`render.go` sync.Mutex + ProgressInPlace) | ✅ 5 test cases covering TTY, Non-TTY, Quiet, Single tool, and 20 concurrent goroutines | ✅ clean lock release with defer |
| **WU-2 Worker Pool & Isolation** | 2.1, 2.2 | `check_test.go` (`TestCalculateWorkerCount_Clamping`, `TestSafeCheck_PanicRecovery`, `TestSafeCheck_TimeoutIsolation`, `TestRunCheck_Concurrent_OrderingAndIsolation`) | Unit | ✅ output unit tests pass | ✅ compile error (`undefined: calculateWorkerCount, defaultConcurrency, safeCheck`) | ✅ pass (`check.go` worker pool + safeCheck recover) | ✅ clamping matrix ([-1..64] -> [4..8]), panic in Detect and Check, timeout errors | ✅ atomic.Int32 counter and waitgroup drain |
| **WU-3 Order Determinism** | 3.1, 3.2 | `integration_test.go` (`TestCheck_DeterministicOrderUnderConcurrency`) | Integration | ✅ cli unit tests pass | ✅ behavior failure (progress vs summary section order assertion) | ✅ pass (canonical index slotting preserves discovery order) | ✅ 5 adapters with inverse delays (5ms to 50ms) verifying summary order `alpha < gamma < epsilon` | ✅ clean test setup and captured stdout |
| **WU-4 Verification** | 4.1, 4.2, 4.3 | `go test ./... -race`, `go vet ./...`, `scripts/smoke-test.sh` | Full Suite | ✅ unit tests pass | n/a | ✅ pass | ✅ full race detector across all 9 packages, 23/23 smoke tests pass | ✅ gofmt formatting |

## Work Unit Evidence (Hard Gate)

| Work Unit | Focused Test Command | Result | Verification Details | Rollback Boundary |
|---|---|---|---|---|
| **WU-1** | `go test ./internal/output/ -run TestRenderer_ConcurrentProgress -count=1` | PASS (0.004s) | 20 goroutines x 50 iterations concurrent progress updates with zero race conditions | `internal/output/render.go`, `internal/output/render_test.go` |
| **WU-2** | `go test ./internal/cli/ -run "TestCalculateWorkerCount\|TestSafeCheck\|TestRunCheck_Concurrent" -count=1` | PASS (0.025s) | Bounded worker pool [4, 8], panic recovery in Detect/Check, timeout isolation | `internal/cli/check.go`, `internal/cli/check_test.go` |
| **WU-3** | `go test ./internal/cli/ -run TestCheck_DeterministicOrderUnderConcurrency -count=1` | PASS (0.053s) | Deterministic result slotting under out-of-order execution delays | `internal/cli/integration_test.go` |
| **WU-4** | `go test ./... -count=1 -race && bash scripts/smoke-test.sh --skip-build` | PASS (1.613s, 23/23 smoke) | Zero data races across all packages, end-to-end smoke verification passed | Read-only verification |

## Verification Summary

- `gofmt -s -w internal/cli/ internal/output/` → clean, no diffs.
- `go vet ./...` → clean, 0 warnings/errors.
- `go test ./... -count=1 -race` → **ALL 9 PACKAGES PASSED** under race detector:
  - `github.com/JhnFrankz/upp/cmd/upp` (no test files)
  - `github.com/JhnFrankz/upp/internal/adapters` (1.021s)
  - `github.com/JhnFrankz/upp/internal/adapters/official` (1.613s)
  - `github.com/JhnFrankz/upp/internal/cli` (1.207s)
  - `github.com/JhnFrankz/upp/internal/config` (1.041s)
  - `github.com/JhnFrankz/upp/internal/output` (1.055s)
  - `github.com/JhnFrankz/upp/internal/platform` (1.016s)
  - `github.com/JhnFrankz/upp/internal/security` (1.042s)
  - `github.com/JhnFrankz/upp/internal/selfupdate` (1.517s)
- `bash scripts/smoke-test.sh --skip-build` → **23 passed, 0 failed, 23 total**.

## Files Touched

| File | Change | Description |
|---|---|---|
| `internal/output/render.go` | Modified | Added `mu sync.Mutex` to `Renderer`, synchronized `Progress`, implemented `ProgressInPlace` with `\r` and non-TTY fallback. |
| `internal/output/render_test.go` | Modified | Added unit tests for thread-safe concurrent progress, TTY in-place rendering, non-TTY fallback, single-tool and quiet suppression. |
| `internal/cli/check.go` | Modified | Implemented `calculateWorkerCount` [4, 8], `defaultConcurrency`, `safeCheck` with deferred `recover()`, worker pool goroutines with direct index slotting. |
| `internal/cli/check_test.go` | Created | Unit tests for worker count clamping, panic containment in Detect/Check, timeout isolation, and concurrent `runCheck`. |
| `internal/cli/integration_test.go` | Modified | Added integration test `TestCheck_DeterministicOrderUnderConcurrency` validating canonical tool order preservation under varied worker delays. |
| `openspec/changes/upp-concurrent-check/tasks.md` | Modified | Marked all tasks complete `[x]`. |
| `openspec/changes/upp-concurrent-check/apply-progress.md` | Created | Full TDD cycle and work unit evidence documentation. |
