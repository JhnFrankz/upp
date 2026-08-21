# Apply Progress: upp-unified-update-flow — Work Unit 1

**Branch**: `wu1/checkrun-seam` (child of tracker `feature/unified-update-flow`, based off `main` @ cfadbb4)
**Delivery**: Feature Branch Chain — PR #1 targets the tracker branch; only the tracker merges to main.
**Mode**: Strict TDD (test runner: `go test ./... -count=1`)
**Attempt authority**: acq-wu1-20260821-000711 (proceed), max 3 attempts / 400 authored-line budget.

## Completed Tasks

- [x] **1.1** Created `internal/cli/checkrun.go`: moved `runChecks`, `safeCheck`, `checkOutcome`, `checkJob`, `calculateWorkerCount`, `defaultConcurrency` verbatim from `check.go`; new signature `runChecks(adapters []adapters.Adapter, onResult func(index int, oc checkOutcome)) []checkOutcome` (nil = silent). Stripped originals + the inline `ProgressInPlace` counter call from `check.go` (the `Renderer.ProgressInPlace` method itself remains in `internal/output/render.go`; its removal is not in any task). Callers updated: `check.go runCheck` → nil, `update.go runUpdateInteractive` → nil (interim until Unit 3 wires the board).
- [x] **1.2** Moved mechanism tests to `internal/cli/checkrun_test.go`: `TestCalculateWorkerCount_Clamping`, `TestSafeCheck_PanicRecovery`, `TestSafeCheck_TimeoutIsolation` verbatim; `TestRunChecks_CarriesUpdateInfo` rewritten to capture via callback; NEW `TestRunChecks_ReportsViaCallback` (per-index exactly-once reporting, callback outcome == returned slot outcome) and `TestRunChecks_NilCallbackSilent` (nil contract). Fakes `fakePanickingAdapter`/`fakeDelayedAdapter` moved with them. Command-level tests (`TestRunCheck_*`) remain in `check_test.go`.

## TDD Cycle Evidence

| Task | Test File | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|------|-----------|-------|------------|-----|-------|-------------|----------|
| 1.1 | `internal/cli/checkrun_test.go` | Unit | ✅ cli pkg ok pre-change | ✅ Build fail: `not enough arguments in call to runChecks` (seam API absent) | ✅ `-run Checks -count=5` ok | ✅ 3 seam cases (report/nil/carries) | ➖ Verbatim move, none needed |
| 1.2 | `internal/cli/checkrun_test.go` | Unit | ✅ cli pkg ok pre-change | ✅ Same build fail (tests reference new signature) | ✅ Passed | ✅ Clamp table (11 cases) + panic×2 + timeout carried over | ➖ None needed |

### Test Summary
- Total tests written/moved into `checkrun_test.go`: 6 (2 new, 4 relocated)
- All passing; full suite green (see gates below)
- Layers used: Unit
- Approval tests: existing mechanism tests serve as approval tests for the relocation (behavior preserved verbatim)

## Work Unit Evidence

| Evidence | Value |
|---|---|
| Focused command + result | `go test ./internal/cli/ -run Checks -count=1` → `ok github.com/JhnFrankz/upp/internal/cli` (also run `-count=5` → ok) |
| Runtime harness + result | `go test ./internal/cli/ ./internal/output/ -race -count=1` → both `ok` (concurrent worker pool is the runtime boundary; no TTY scenario in this unit — board arrives in Units 2–3) |
| Rollback boundary | Revert commit 8d1cb2a: restores old `runChecks(adapters, renderer, quiet, showProgress)` signature, counter UX, and original test placement atomically; touches nothing else |

## Full Gate Results

- `go build ./...` → clean
- `go vet ./...` → clean
- `gofmt -s -l .` → no output (clean)
- `go test ./... -count=1` → all packages ok

## Files Changed

| File | Action | What Was Done |
|------|--------|---------------|
| `internal/cli/checkrun.go` | Created | Check engine moved verbatim + `onResult` seam (D2/D3) |
| `internal/cli/checkrun_test.go` | Created | Mechanism tests via callback capture + 2 new seam tests |
| `internal/cli/check.go` | Modified | Machinery stripped (~133 lines); `runCheck` passes nil; imports trimmed |
| `internal/cli/check_test.go` | Modified | Mechanism tests/fakes moved out (~199 lines); command-level tests kept |
| `internal/cli/update.go` | Modified | Pre-check passes nil interim; comments updated |
| `internal/cli/update_test.go` | Modified | 2 stale "Checking X/Y" assertions → interim silent-engine assertions |
| `internal/cli/integration_test.go` | Modified | `TestCheckProgress_LabelsChecking` locks interim silence (deleted in task 3.3) |

## Changed Lines

Raw git churn: 451 insertions + 358 deletions = **809 lines**, of which ≈660 are verbatim relocations (engine ~133 + tests ~199 counted on both sides of the move) and ≈150 are authored changes (seam signature/docs, two new tests, caller updates, three assertion adaptations). Authored content is within the 400-line budget; raw churn exceeds it because this unit is by definition a relocation slice (PR1 "seam" of the planned chain).

## Deviations from Design

1. Three existing tests asserting the deleted "Checking X/Y" counter (`update_test.go` ×2, `integration_test.go` ×1) were adapted to lock the interim silent-engine contract instead of being left red. Required for the full-gate mandate; full board assertions arrive in tasks 2.3/3.3. Design D2 anticipated the counter's death ("no caller wants the counter").
2. `Renderer.ProgressInPlace` method retained in `internal/output/render.go`: task text "strip originals + ProgressInPlace" refers to the inline call site (design D2: "Inline ProgressInPlace (check.go:171-173) is deleted"); no task removes the renderer method or its output-package tests.

## Issues Found

- First GREEN run crashed under `-run Checks`: my initial `TestRunChecks_CarriesUpdateInfo` wrote an unguarded map from concurrent worker callbacks (fatal: concurrent map writes). Fixed with a mutex — and it usefully proves the seam fires concurrently, which downstream board consumers must serialize (D1 mutex).

## Remaining Tasks

- [ ] 1.3 RED `internal/output/checkboard_test.go`
- [ ] 1.4 Implement `internal/output/checkboard.go` + `Renderer.Color()`
- [ ] Phase 2 wiring, Phase 3 removals, Phase 4 spec verification, Phase 5 docs/E2E

## Status

2/17 tasks complete. Ready for WU2 (`sdd-apply` on tasks 1.3–1.4), then verify at chain end.
