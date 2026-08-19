# Tasks: Concurrent Tool Checking Engine (`upp-concurrent-check`)

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~150–200 |
| 400-line budget risk | Low |
| Chained PRs recommended | No |
| Suggested split | Single PR |
| Delivery strategy | single-pr |
| Chain strategy | stacked-to-main |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: stacked-to-main
400-line budget risk: Low

Recommendation: Single PR (~150–200 lines). Changes are self-contained within `internal/output/render.go` and `internal/cli/check.go`.

### Work Units

| Unit | Goal | Likely PR | Test command | Rollback boundary |
|------|------|-----------|--------------|-------------------|
| 1 | Output Synchronization & Progress | Single PR | `go test ./internal/output/ -run TestRenderer_ConcurrentProgress -count=1` | `render.go`, `render_test.go` |
| 2 | Concurrent Worker Pool Engine | Single PR | `go test ./internal/cli/ -run TestRunCheck_Concurrent -count=1` | `check.go`, `check_test.go` |
| 3 | Order Determinism & Integration Guards | Single PR | `go test ./internal/cli/ -run TestCheck_DeterministicOrder -count=1` | `integration_test.go` |
| 4 | Verification & Quality | Single PR | `go test ./... -count=1 -race` | None (read-only verification) |

## Phase 1: Output Synchronization & In-Place Progress

- [x] 1.1 RED: `internal/output/render_test.go` — Add `TestRenderer_ConcurrentProgress_ThreadSafe` verifying concurrent progress updates do not panic, corrupt output, or cause data races under `-race`.
- [x] 1.2 GREEN: `internal/output/render.go` — Add `sync.Mutex` (`mu`) to `Renderer` struct to synchronize `Progress()` and implement `ProgressInPlace(op string, current, total int, name string)` using `\r` for interactive TTYs with clean fallback for non-TTY/CI.

## Phase 2: Concurrent Worker Pool Engine

- [x] 2.1 RED: `internal/cli/check_test.go` — Add `TestRunCheck_Concurrent_OrderingAndIsolation` testing worker pool bounds clamping (`[4, 8]`), mock adapter panic recovery returning `output.StatusFailed`, and timeout isolation.
- [x] 2.2 GREEN: `internal/cli/check.go` — Implement `calculateWorkerCount()` clamping to `[4, 8]`, worker pool goroutines with `checkJob` channels, deferred `recover()` panic containment in `safeCheck()`, atomic progress tracking, and direct index slotting into pre-allocated `results` slice.

## Phase 3: Order Determinism & Integration Guards

- [x] 3.1 RED/GREEN: `internal/cli/integration_test.go` — Add `TestCheck_DeterministicOrderUnderConcurrency` with adapters completing with varied/reverse delays, verifying summary output always preserves canonical tool discovery sequence.
- [x] 3.2 VERIFY: Run `go test ./... -count=1 -race` ensuring zero race conditions across CLI and output packages.

## Phase 4: Full Suite Verification

- [x] 4.1 VERIFY: Run `go test ./... -count=1 -race` for full workspace unit and race test suite.
- [x] 4.2 VERIFY: Run `go vet ./...` and `gofmt -s -w internal/cli/ internal/output/`.
- [x] 4.3 VERIFY: Run `bash scripts/smoke-test.sh --skip-build` to validate CLI end-to-end functionality.
