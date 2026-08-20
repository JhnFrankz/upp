# Design: Interactive Update Tool Selection

## Technical Approach

Add a hand-rolled ANSI `CheckboxSelector` in `internal/output` (new `selector.go`). `upp update` in a TTY (gate: `stdinIsTTY() && !ci && !quiet && !dry-run`) runs a concurrent pre-check over the `--only`/`--skip`-filtered adapters via `runChecks` extracted from `check.go`, renders pending (`StatusAvailable`) tools pre-checked, then runs the existing per-tool loop (security matrix + policy gating intact) over the user's selection. Esc/q → nothing updated, fixed cancel message, exit 0. Non-TTY/`--ci`/`--quiet`/`--dry-run` paths stay byte-identical. Raw mode via `golang.org/x/term.MakeRaw` with `defer` restore.

## Architecture Decisions

| # | Option | Tradeoff | Decision |
|---|--------|----------|----------|
| 1 | Raw mode: `x/term` vs tcell vs cooked input | tcell = full TUI lib (v2 diff-pane territory); cooked can't read arrows | `x/term.MakeRaw`; package vars `makeRawFn`/`restoreTerm` as test seams (mirrors `shellExecWithTimeoutFn`, adapters/exec.go); `defer` restore |
| 2 | Injectable reader | Mirror `ConfirmConfig.Reader` (security/confirm.go) and `selfUpdateDeps.stdin` | `updateDeps` gains `stdinIsTTY func() bool` and `selector func(pending []SelectOption) ([]string, bool)`; zero value = production |
| 3 | `runChecks` home | Lives in package `cli`; both commands share it | Extract to check.go: `runChecks(adapters, r, quiet, showProgress) []checkOutcome`; `safeCheck` returns `checkOutcome{result, updateInfo}` (check_test.go call sites read `.result`) |
| 4 | Double-check avoidance | Sequential loop re-calls `a.Check()` | Interactive loop switches on carried `checkOutcome`: Skipped/Failed → append as-is; Available → confirm → gate → update → overwrite. No second `Check()` (RED test: call count = 1 per tool) |
| 5 | Pre-check progress in captured output | "Checking X/Y" lines appear pre-selector | runChecks renders them (TTY anyway); interactive-path tests include them deliberately; non-interactive paths never reach runChecks → byte-identical |
| 6 | Selection → summary | Deselected pending tools need honest counts | Results = all non-available outcomes (current/skipped/failed) + updated selected; deselected pending tools dropped (never processed); counts reflect executed selection |
| 7 | Always-update tools without reported update (brew et al.) | Today's TTY updates them; selector lists only pending | Interactive TTY now updates the pending set only — spec-sanctioned ("selector over pending set"); non-interactive keeps today's contract byte-identical |
| 8 | Cancel message copy (spec left open) | Any wording; must be clear, English, exit 0 | `Update cancelled — no changes made.` via new `Renderer.UpdateCancelled()` |
| 9 | Per-row version display | Full diff pane = v2 | Reuse safeCheck's `"Current → Latest"` string as `SelectOption.Version` |

## Data Flow

```
upp update (TTY && !ci && !quiet && !dry-run)
  config.Load → platform.Detect → buildAdapterList
  → ParseFilter/FilterTools (--only/--skip pre-filter)
  → runChecks (worker pool [4,8], safeCheck) ──► []checkOutcome
  → pending := outcomes[StatusAvailable]
  → none? ──► skip selector, UpdateSummary(all outcomes)
  → CheckboxSelector.Run (raw mode, ANSI render)
     ├─ Esc/q ──► "Update cancelled — no changes made." → exit 0
     └─ Enter ──► selected []string
  → per selected tool: ConfirmAction → policy gate → Update
  → results = non-available outcomes + updated selected
  → UpdateSummary
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/output/selector.go` | Create | CheckboxSelector: ANSI render, key handling, raw mode + restore seams |
| `internal/output/render.go` | Modify | Add `UpdateCancelled()`; existing methods byte-identical |
| `internal/output/selector_test.go` | Create | Key/toggle/cancel/raw-mode RED tests |
| `internal/cli/check.go` | Modify | Extract `runChecks` + `checkOutcome`; `runCheck` delegates |
| `internal/cli/update.go` | Modify | Gate, pre-check branch, selector, carried-outcome loop, cancel path |
| `internal/cli/update_test.go` | Modify | Gate matrix, subset loop, cancel, no-double-check tests |
| `internal/cli/check_test.go` | Modify | `safeCheck` call sites → `.result` |
| `go.mod`, `go.sum` | Modify | `golang.org/x/term` only |

## Interfaces / Contracts

```go
// internal/output/selector.go
type SelectOption struct {
    ID      string // tool ID (--only/--skip key)
    Label   string // display name
    Version string // inline "Current → Latest"
}
type SelectResult struct {
    Selected  []string // selected IDs in display order
    Cancelled bool     // Esc/q
}
func NewCheckboxSelector(w io.Writer, r io.Reader, opts []SelectOption) *CheckboxSelector
func (s *CheckboxSelector) Run() (SelectResult, error) // MakeRaw + defer Restore

// internal/cli/check.go
type checkOutcome struct {
    result     output.ToolResult
    updateInfo adapters.UpdateInfo // zero on detect/check failure
}
func runChecks(adapters []adapters.Adapter, r *output.Renderer, quiet, showProgress bool) []checkOutcome

// internal/cli/update.go — updateDeps additions (zero value = production)
stdinIsTTY func() bool
selector   func(pending []output.SelectOption) ([]string, bool) // selected, cancelled
```

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit (output) | ↑/↓/Space/`a`/`n`/Enter/Esc/`q` (incl. `\r` vs `\n`), toggle, all/none, unknown keys, deterministic order | Table-driven via injectable reader |
| Unit (output) | Raw mode: MakeRaw once on `*os.File` reader; Restore on cancel, confirm, AND panic (RED); non-`*os.File` reader skips raw mode | Seamed `makeRawFn`/`restoreTerm` |
| Unit (cli) | Gate matrix (TTY×ci×quiet×dry-run), no-pending skip, selector over filtered set, deselected tool's `Update` not called, cancel → exit 0 + message, ConfirmAction still runs per selected custom tool, `Check()` count = 1 | Seamed `stdinIsTTY`/`selector` + fake adapters |
| Integration | Existing `--ci`/`--quiet`/`--dry-run`/non-TTY tests stay green (stdin = /dev/null under `go test` → gate false) | Unchanged |
| E2E | Non-TTY smoke (no selector, byte-identical); manual TTY verify; `make build-all` for linux/darwin/windows amd64+arm64 raw-mode compile | scripts/smoke-test.sh |

## Threat Matrix

| Boundary | Applicability | Design response | Planned RED tests |
|----------|---------------|-----------------|-------------------|
| Documentation-like paths | N/A — no file classification added | — | — |
| Git repository selection | N/A — no git invocation | — | — |
| Commit state | N/A — no VCS automation | — | — |
| Push state | N/A — no VCS automation | — | — |
| PR commands | N/A — no PR automation | — | — |
| Terminal raw mode (x/term) | **Applicable** — process-integration boundary: terminal state entered/restored | Safe: Restore via `defer` on every exit path (confirm, cancel, error, panic). Failure: terminal left in raw state | Restore called on cancel, on confirm, and on injected panic during `Run`; MakeRaw called exactly once |
| Subprocess execution | N/A — no new exec boundary; adapters/exec.go unchanged, covered by existing tests | — | — |

## Migration / Rollout

No migration. Rollback: revert the selector commit or flip the gate — non-interactive paths are byte-identical today. `runChecks` extraction is additive; keep on revert.

## Open Questions

None.
