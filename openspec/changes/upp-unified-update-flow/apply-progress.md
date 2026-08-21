# Apply Progress: upp-unified-update-flow

**Delivery**: Feature Branch Chain — child PRs target the immediate previous PR branch; only tracker `feature/unified-update-flow` merges to main.
**Mode**: Strict TDD (test runner: `go test ./... -count=1`)
**Attempt authority**: WU1 acq-wu1-20260821-000711 (proceed); WU2 acq-wu2-20260821-003247 (proceed, token sha256:f80520cf…8751), max 3 attempts / 550 raw-line budget.

---

## Work Unit 1 — Check engine relocation + onResult seam (COMPLETE)

**Branch**: `wu1/checkrun-seam` (child of tracker, based off `main` @ cfadbb4) · PR #97 → base `feature/unified-update-flow`

### Completed Tasks

- [x] **1.1** Created `internal/cli/checkrun.go`: moved `runChecks`, `safeCheck`, `checkOutcome`, `checkJob`, `calculateWorkerCount`, `defaultConcurrency` verbatim from `check.go`; new signature `runChecks(adapters []adapters.Adapter, onResult func(index int, oc checkOutcome)) []checkOutcome` (nil = silent). Stripped originals + the inline `ProgressInPlace` counter call from `check.go` (the `Renderer.ProgressInPlace` method itself remains in `internal/output/render.go`; its removal is not in any task). Callers updated: `check.go runCheck` → nil, `update.go runUpdateInteractive` → nil (interim until Unit 3 wires the board).
- [x] **1.2** Moved mechanism tests to `internal/cli/checkrun_test.go`: `TestCalculateWorkerCount_Clamping`, `TestSafeCheck_PanicRecovery`, `TestSafeCheck_TimeoutIsolation` verbatim; `TestRunChecks_CarriesUpdateInfo` rewritten to capture via callback; NEW `TestRunChecks_ReportsViaCallback` (per-index exactly-once reporting, callback outcome == returned slot outcome) and `TestRunChecks_NilCallbackSilent` (nil contract). Fakes `fakePanickingAdapter`/`fakeDelayedAdapter` moved with them. Command-level tests (`TestRunCheck_*`) remain in `check_test.go`.

### TDD Cycle Evidence (WU1)

| Task | Test File | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|------|-----------|-------|------------|-----|-------|-------------|----------|
| 1.1 | `internal/cli/checkrun_test.go` | Unit | ✅ cli pkg ok pre-change | ✅ Build fail: `not enough arguments in call to runChecks` (seam API absent) | ✅ `-run Checks -count=5` ok | ✅ 3 seam cases (report/nil/carries) | ➖ Verbatim move, none needed |
| 1.2 | `internal/cli/checkrun_test.go` | Unit | ✅ cli pkg ok pre-change | ✅ Same build fail (tests reference new signature) | ✅ Passed | ✅ Clamp table (11 cases) + panic×2 + timeout carried over | ➖ None needed |

### Work Unit Evidence (WU1)

| Evidence | Value |
|---|---|
| Focused command + result | `go test ./internal/cli/ -run Checks -count=1` → `ok github.com/JhnFrankz/upp/internal/cli` (also run `-count=5` → ok) |
| Runtime harness + result | `go test ./internal/cli/ ./internal/output/ -race -count=1` → both `ok` |
| Rollback boundary | Revert commit 8d1cb2a: restores old `runChecks(adapters, renderer, quiet, showProgress)` signature, counter UX, and original test placement atomically |

### Changed Lines (WU1)

Raw git churn: 451 insertions + 358 deletions = **809 lines** (≈660 verbatim relocations, ≈150 authored).

---

## Work Unit 2 — CheckBoard + Renderer.Color() (COMPLETE)

**Branch**: `wu2/checkboard` (based off `wu1/checkrun-seam` @ 8f11fe9; PR targets base `wu1/checkrun-seam`, NOT tracker/main)
**Scope guard**: board NOT wired into update.go here — that is task 2.1 (WU3).

### Completed Tasks

- [x] **1.3** RED `internal/output/checkboard_test.go`: 10 tests — canonical-order pending lines on `Start`; per-status flips Available (`✓ name cur → new`)/Current (`✓ name up-to-date`)/Skipped (`✓ name not installed`)/Failed (`✗ name: err`) on `Complete`; only-target-line isolation; out-of-order completion never reorders rows; idempotent `Finish`; non-color fallback (Start emits nothing, one plain line per completion, zero ANSI bytes); 8-goroutine concurrent `Complete` across all four statuses under `-race`; out-of-range index defense. Assertions run through a minimal VT100-subset terminal simulator (`\r`, `\n`, `\x1b[<n>A/B`, `\x1b[K`) asserting the settled visible frame, not raw escape bytes.
- [x] **1.4** Implemented `internal/output/checkboard.go`: `NewCheckBoard(w io.Writer, color bool, tools []string) *CheckBoard` + `Start/Complete/Finish`. Per-line state machine stores rendered row text; color mode rewrites one row via cursor-up + clear-to-end + cursor-down return (never full-board redraw, D4); private `sync.Mutex` serializes every write (D1); fallback prints one plain line per completion (D5). Added exported `Renderer.Color()` getter in `render.go` so the board reuses the renderer's single TTY-detection source (D5).

### TDD Cycle Evidence (WU2)

| Task | Test File | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|------|-----------|-------|------------|-----|-------|-------------|----------|
| 1.3 | `internal/output/checkboard_test.go` | Unit | ✅ output+cli pkgs ok pre-change (baseline captured before edits) | ✅ Build fail: `undefined: NewCheckBoard` ×10 | ✅ 10/10 pass after fix of test-side stray `buf.Reset()` | ✅ 10 cases incl. concurrency + fallback + defense | ✅ gofmt/vet clean after signature fix |
| 1.4 | `internal/output/checkboard_test.go` | Unit | same baseline | ✅ same build fail (production code absent) | ✅ `go test ./internal/output/ -run CheckBoard -count=1` → ok | covered by 1.3 matrix | ➖ helpers extracted (`boardMarkerLine`), tests still green |

### Work Unit Evidence (WU2)

| Evidence | Value |
|---|---|
| Focused command + result | `go test ./internal/output/ -run CheckBoard -count=1` → `ok github.com/JhnFrankz/upp/internal/output` (10 tests) |
| Runtime harness + result | `go test ./internal/output/ ./internal/cli/ -race -count=1` → both `ok` (concurrent `Complete` serialization is the runtime boundary; no TTY scenario in this unit — wiring arrives in WU3) |
| Rollback boundary | Delete `internal/output/checkboard.go` + `internal/output/checkboard_test.go` and revert the 5-line `Renderer.Color()` addition in `render.go` (commit 13560a7); touches nothing else — no caller exists yet by design |

### Full Gate Results (WU2)

- `go build ./...` → clean
- `go vet ./...` → clean
- `gofmt -s -l .` → no output (clean)
- `go test ./... -count=1` → all 9 packages ok

### Files Changed (WU2)

| File | Action | What Was Done |
|------|--------|---------------|
| `internal/output/checkboard.go` | Created | `CheckBoard` component: per-line state machine, mutex, cursor-up+clear redraw, color fallback |
| `internal/output/checkboard_test.go` | Created | 10 behavioral tests + VT100-subset terminal simulator harness |
| `internal/output/render.go` | Modified | Added exported `Color()` getter (+5 lines) |

### Changed Lines (WU2)

Raw git churn: **519 insertions, 0 deletions** (commit 13560a7) — within the 550 raw-line attempt budget. ≈100 lines of that are the terminal-simulator test harness.

---

## Remaining Tasks

- [ ] 2.1 Wire board into `update.go` `runUpdateInteractive` (WU3)
- [ ] 2.2 `render.go` up-to-date summary count/listing (D6)
- [ ] 2.3 `update_test.go` board assertions replace "Checking X/Y" interim locks
- [ ] Phase 3 removals, Phase 4 spec verification, Phase 5 docs/E2E

## Status

4/17 tasks complete. Ready for WU3 (`sdd-apply` on tasks 2.1–2.3), then verify at chain end.
