# Tasks: Interactive Update Tool Selection

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 650–850 |
| Budget risk (300-line) | High |
| Chained PRs | Yes — PR 1 → 2 → 3 |
| Delivery strategy | auto-chain |

Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: High

### Work Units

| # | Goal | PR | Test cmd | Harness | Rollback |
|---|------|----|----------|---------|----------|
| 1 | Selector + raw mode + cancel msg | PR 1 | `go test ./internal/output/` | N/A — injectable seams; TTY run after PR 3 | Revert selector.go, selector_test.go, render.go, go.mod/go.sum |
| 2 | runChecks/checkOutcome | PR 2 | `go test ./internal/cli/ -run Check` | N/A — pure refactor | Revert check.go, check_test.go |
| 3 | update.go gate + loop + cancel | PR 3 | `go test ./internal/cli/ -run Update` | Real TTY `upp update` | Revert update.go, update_test.go |

## Phase 1: Foundation — Selector

- [x] 1.1 Add `golang.org/x/term` to `go.mod`/`go.sum`
- [x] 1.2 RED `selector_test.go`: ↑/↓/Space/`a`/`n`/Enter/Esc/`q` (`\r`/`\n`), toggle, all/none, unknown keys, deterministic order
- [x] 1.3 RED `selector_test.go`: raw mode — MakeRaw once on `*os.File`; Restore on cancel, confirm, injected panic; non-File skips raw
- [x] 1.4 GREEN `selector.go`: `SelectOption{ID,Label,Version}`, `SelectResult{Selected,Cancelled}`, `NewCheckboxSelector(w,r,opts)`, `Run()` — `MakeRaw` + `defer` Restore; seams `makeRawFn`/`restoreTerm`
- [x] 1.5 `render.go`: `UpdateCancelled()` → `Update canceled — no changes made.`; existing methods byte-identical

## Phase 2: Core — check.go

- [x] 2.1 RED `check_test.go`: `runChecks` carries `updateInfo` (fake adapter; Available → "current → latest")
- [x] 2.2 GREEN `check.go`: `checkOutcome{result, updateInfo}`; `safeCheck` returns it
- [x] 2.3 Update `check_test.go` `safeCheck` call sites → `.result`
- [x] 2.4 GREEN `check.go`: extract `runChecks(adapters, r, quiet, showProgress) []checkOutcome` ([4,8] pool kept); `runCheck` delegates; byte-identical

## Phase 3: Core — update.go

- [x] 3.1 RED `update_test.go`: gate matrix — selector only `TTY&&!ci&&!quiet&&!dry-run`; no pending → skipped
- [x] 3.2 RED `update_test.go`: deselected `Update` NOT called; `Check()` count = 1; ConfirmAction still runs per selected custom tool; summary matches
- [x] 3.3 RED `update_test.go`: cancel → `UpdateCancelled`, nothing updated, exit 0; include "Checking X/Y" lines deliberately
- [x] 3.4 GREEN `update.go`: `updateDeps` += `stdinIsTTY func() bool`, `selector func([]output.SelectOption) ([]string, bool)` (zero = production)
- [x] 3.5 GREEN `update.go`: gate; pre-check via `runChecks`; pending = StatusAvailable; none → skip selector, summary
- [x] 3.6 GREEN `update.go`: carried-outcome loop — Skipped/Failed as-is; Available → ConfirmAction → gate → Update → overwrite (no 2nd `Check()`); deselected dropped
- [x] 3.7 GREEN `update.go`: cancel → `r.UpdateCancelled()` + exit 0; keep D7 — always-update tools NOT force-updated in TTY

## Phase 4: Integration / Docs

- [x] 4.1 `go test ./... -count=1` green
- [x] 4.2 `bash scripts/smoke-test.sh --skip-build` — non-TTY byte-identical
- [x] 4.3 `go test ./... -count=1 -race`; `go vet ./...`; `make build-all` (Windows raw-mode)
- [x] 4.4 Manual TTY: subset selection, Esc/q cancel, raw mode restored — VERIFIED WITH CAVEAT: zero pending updates on the dev machine today, so the selector could not be exercised live; covered by 27 unit tests (keys/toggle/cancel/raw-mode restore) + smoke non-TTY byte-identical + build-all Windows raw-mode compile. Manual live check deferred to the first real pending update.
- [x] 4.5 Update README if TTY described; delta specs ready for archive sync
