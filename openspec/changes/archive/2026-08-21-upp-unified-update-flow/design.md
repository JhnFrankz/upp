# Design: Unified Update Flow — Live Check Board, Remove `check`

## Technical Approach

Swap the interactive pre-check's single mutating "Checking X/Y" counter for a live per-tool board owned by a new `internal/output` component, driven through a completion-callback seam on `runChecks`. Delete `upp check`, the self-update hint, and `settings.check_self_update`; relocate the shared check engine to `internal/cli/checkrun.go`. `upp update --dry-run` becomes the sole read-only query surface. Carried-outcome designs D4/D6/D7 and all bypass contracts (`--ci`/`--quiet`/`--dry-run`/non-TTY) are untouched.

## Architecture Decisions

| # | Decision (Choice) | Alternatives considered | Rationale |
|---|-------------------|-------------------------|-----------|
| D1 | New `CheckBoard` in `internal/output/checkboard.go`; owns multi-line ANSI cursor control (cursor-up + clear-to-end redraw) behind its own `sync.Mutex` | Reuse `Renderer.mu`; post-hoc listing; ticker-driven TUI | Spec mandates atomic, non-interleaved updates; renderer mutex guards single writes only. Explore Approaches 2/3 rejected: not live / timer-goroutine race surface |
| D2 | `runChecks` signature: replace `(r *Renderer, quiet, showProgress bool)` with `onComplete func(index int, oc checkOutcome)`; nil = silent. Inline `ProgressInPlace` (check.go:171-173) is deleted, not replaced | Keep counter plus optional hook | After `check` dies, no caller wants the counter; one seam serves the board and tests |
| D3 | Move `runChecks`, `safeCheck`, `checkOutcome`, `checkJob`, `calculateWorkerCount`, `defaultConcurrency` verbatim to `internal/cli/checkrun.go`; then delete `check.go` (command, `runCheck`, `checkDeps`, `maybeShowSelfUpdateHint`) | Retain check.go; new package | Move-never-delete; same `cli` package → zero import churn |
| D4 | Board lifecycle: `Start()` paints N pending lines in filtered-adapter (canonical) order before the first result; `Complete(i, res)` flips exactly one line; `Finish()` settles the final frame before the selector renders | Full-board redraw per event | Per-line flip avoids flicker; index slotting already guarantees completion order never reorders lines |
| D5 | Fallback: board branches on stdout color support via a new exported `Renderer.Color()` getter passed to `NewCheckBoard(w, color, …)`; non-color → one plain line per completion, no ANSI. `--quiet` cannot reach the board (interactive gate, update.go:91) | Board detects TTY itself | Mirrors `ProgressInPlace`'s `r.color` precedent; keeps one TTY-detection source (`isTerminal`) |
| D6 | Dry-run summary wiring: `UpdateSummary` gains an explicit "N up to date" count/part and `detailSummary` lists Current tools. Verbose stderr diagnostics already render on the update path (render.go:343-351) — no new wiring needed there | Leave summary as-is | ux-patterns Summary Report scenario "8 up to date, 2 skipped" fails today: current tools are counted nowhere on the update path |

## Data Flow

```
runUpdateInteractive (update.go:274)
  └─ runChecks(filteredAdapters, onResult)      checkrun.go — worker pool [4..8]
       ├─ safeCheck(a) → checkOutcome{result, updateInfo}   (panic-safe, unchanged)
       └─ onResult(i, oc) ──→ board.Complete(i, oc.result)  [mutex-serialized]
                                 │ settled board persists on stdout
  └─ pending = StatusAvailable outcomes ────→ CheckboxSelector (pending-only)
  └─ carried-outcome loop (D4/D6/D7 intact) → UpdateSummary
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/output/checkboard.go` | Create | `CheckBoard`: per-line state machine (pending → ✓ current→new \| ✓ up-to-date \| ✗ + inline error), mutex, color fallback |
| `internal/output/render.go` | Modify | Delete `SelfUpdateHint` (:571); add `Color()`; `UpdateSummary`/`detailSummary` count and list up-to-date (D6) |
| `internal/cli/checkrun.go` | Create | Machinery moved verbatim + `onComplete` seam (D2/D3) |
| `internal/cli/check.go` | Delete | Emptied by relocation + removals |
| `internal/cli/update.go` | Modify | Interactive pre-check constructs board, passes `board.Complete` as `onResult` |
| `internal/cli/parser.go` | Modify | Drop check registration (:68-69, :83) and stale grouping comment |
| `internal/cli/deps.go` | Modify | Drop `check checkDeps` slot |
| `internal/config/config.go` | Modify | Delete `Settings.CheckSelfUpdate` (:29) and its default |
| `scripts/smoke-test.sh` | Modify | Tests 2/5/7/8/9 (~9 assertions) → `list` / `update --dry-run` variants; add `upp check` → exit 1 |
| `README.md` | Modify | Lines 65/91/109/144: drop check + hint; document `-n` query surface |
| CLI test files | Modify/Delete | See Testing Strategy |

## Interfaces / Contracts

```go
// internal/cli/checkrun.go
func runChecks(adapters []adapters.Adapter,
    onResult func(index int, oc checkOutcome)) []checkOutcome

// internal/output/checkboard.go
func NewCheckBoard(w io.Writer, color bool, tools []string) *CheckBoard
func (b *CheckBoard) Start()                             // N pending lines, canonical order
func (b *CheckBoard) Complete(index int, res ToolResult) // flip one line; mutex-serialized
func (b *CheckBoard) Finish()                            // settle final frame; idempotent
```

`ToolResult.Status` drives each flip: Available → ✓ `Version` ("cur → new"), Current → ✓ up-to-date, Skipped → ✓ not installed, Failed → ✗ with `Error`. Selector input stays pending-only (`StatusAvailable`), excluding current and failed tools — spec-mandated.

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit | Board states, canonical ordering, atomicity, fallback | `bytes.Buffer` + forced color on/off; concurrent `Complete` under `-race`; one line per tool; no "Checking X/Y" anywhere |
| Unit | Hook seam | Mechanism tests renamed into `checkrun_test.go` (worker clamp, `safeCheck` panic/timeout isolation, carries-updateInfo) capture completions via callback |
| Integration | TTY interactive flow | Existing `updateDeps.stdinIsTTY`/`selector` fakes; update_test.go:577/:680 "Checking 1/N" assertions become board assertions |
| E2E | smoke-test.sh | `list`, `update --dry-run` with `-q`/`-v`/`--only`/`--skip`; `upp check` unknown-command exit 1 |

**Dies:** `check_hint_test.go` (8 tests), `TestCheckProgress_LabelsChecking` (integration_test.go:491), `SelfUpdateHint` render tests, `runCheck`-command-level tests, "check" entries in help_test.go:50 and parser_test.go:265, the 8 `SetArgs([]string{"check"})` integration invocations (rewritten onto `update --dry-run`). **Config backward-compat:** extend the unknown-key tables (config_test.go:266, :347) with `check_self_update = true` → loads silently ignored and `Save` never rewrites it: guaranteed by BurntSushi's non-strict `toml.Unmarshal` (config.go:119) and struct-only encoding in `Save` (:153). Leftover `self-update-cache.json` has no remaining reader.

## Threat Matrix

N/A — no routing, shell, subprocess-launch, VCS/PR automation, executable-file classification, or process-integration boundary changes; `safeCheck` subprocess behavior relocates verbatim.

## Migration / Rollout

No data migration; feature flags unnecessary. Rollback: single-branch `git revert` restores command, hint, and counter UX atomically; spec deltas apply at archive.

## Open Questions

- [ ] None blocking. Boards taller than the viewport rely on normal terminal scroll; no resize handling in v1 (consistent with the selector's fixed-frame model).
