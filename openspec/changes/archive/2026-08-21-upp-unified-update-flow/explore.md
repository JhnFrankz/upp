# Exploration: upp-unified-update-flow

## Current State

### `upp update` end-to-end today

Entry: `NewUpdateCommand` (internal/cli/update.go:19) → `runUpdate` (update.go:52).

1. Load config, `platform.Detect()`, build adapter list (`buildAdapterList`: official + custom adapters for enabled tools).
2. Filter: `ParseFilter(gf.Only, gf.Skip)` → `FilterTools` (filter warnings go to stderr).
3. Renderer: `output.NewRendererVerbose(os.Stdout, gf.Quiet, gf.Verbose)`; `--dry-run` prints `DryRunHeader`.
4. **Interactive gate** (update.go:91): `stdinIsTTY() && !gf.CI && !gf.Quiet && !uf.DryRun` → `runUpdateInteractive`; every other combination → `runUpdateSequential`.

Sequential path (`runUpdateSequential`, update.go:103): per tool in order — Detect → `Progress("Updating X/Y")` → `Check()` → dry-run planned action, or `security.ConfirmAction` (trust/risk matrix; `--ci` yields `ConfirmError` for untrusted custom tools) → policy gate (`PolicyGated` adapters update only when `UpdateAvailable`; `PolicyAlwaysUpdate` always run) → `Update(false)` → collect `ToolResult`. Ends with `r.UpdateSummary`; `--ci` exits non-zero on any failure.

Interactive path (`runUpdateInteractive`, update.go:274):

1. **Concurrent pre-check**: `runChecks(filteredAdapters, r, gf.Quiet, true)` (internal/cli/check.go:145) — worker pool clamped to [4,8] CPUs, `safeCheck` wraps Detect+Check with panic containment, results slotted by index (deterministic order), and each outcome **carries** `adapters.UpdateInfo` so `Check()` is never re-invoked (design D4).
2. Progress during pre-check: `r.ProgressInPlace("Checking", cur, total, name)` — a SINGLE mutating counter line (`\r` rewrite when stdout is color-capable, newline lines otherwise). Not one line per tool.
3. Pending = outcomes with `StatusAvailable`, mapped to `output.SelectOption{ID, Label, Version: "cur → latest"}`.
4. No pending → render `UpdateSummary` from outcomes and skip the selector entirely (spec ux-patterns "No pending updates").
5. Selector seam → production `CheckboxSelector(os.Stdout, os.Stdin, opts).Run()` (internal/output/selector.go): all options pre-checked, space toggles, `a`/`n` all/none, arrows move, Enter confirms, Esc/q cancels; raw mode on `*os.File` stdin restored on every exit path.
6. Cancel → `r.UpdateCancelled()`, exit 0.
7. **Carried-outcome loop** over ALL outcomes: Skipped/Failed/Current append as-is; Available tools are updated only if selected, via `processSelectedOutcome` (progress "Updating i/N" over the executed selection, `ConfirmAction` still runs — the selector is a choice UI, NOT a security confirmation — policy gate re-evaluated with the carried `updateInfo`). Deselected pending tools are dropped. Always-update tools without a reported update are NOT force-updated in interactive runs (design D7).
8. `UpdateSummary`; `--ci` failure aggregation identical to sequential.

### `upp check` today

`NewCheckCommand` (internal/cli/check.go:22) → `runCheck` (check.go:185): same config/platform/filter/renderer preamble as update, then `runChecks(..., showProgress=true)`, map outcomes to results, `r.CheckSummary(results)`, then `maybeShowSelfUpdateHint` (opt-in via `settings.check_self_update`, suppressed by `--quiet`, cached 24h, silent offline, never changes exit code). Always read-only; exits 0 regardless of pending updates.

**Hidden coupling discovered**: `maybeShowSelfUpdateHint` is called ONLY from `runCheck` (check.go:217,232). Despite README line 109 claiming bare `upp` also shows the hint, root.go contains no hint wiring — `upp check` is the only code path that renders the self-update hint today.

### Flag contracts in force (must be preserved)

| Mode | Behavior |
|------|----------|
| Interactive gate | Selector only when stdin is TTY AND `--ci`, `--quiet`, `--dry-run` all unset (update.go:91; spec command-interface `upp update`, ux-patterns "Interactive Update Tool Selection") |
| `--dry-run` / `-n` | Header + planned actions `name (cur → latest)`, `StatusAvailable` rows, summary reports "N would update" and never "All clean!" (render.go D3), zero mutations, no selector |
| `--quiet` | `Progress`/`ProgressInPlace` early-return, detail summary suppressed, hint suppressed, no selector |
| `--ci` | No selector; `ConfirmAction` returns `ConfirmError` for untrusted custom tools (fail closed); exit non-zero "update completed with failures" |
| Non-TTY | Sequential path, byte-identical legacy behavior; `ProgressInPlace` falls back to newline lines when stdout lacks color |
| `upp check` | Read-only always, exit 0 regardless of findings, hint opt-in |

Spec anchors: openspec/specs/command-interface/spec.md (`upp update`, Help Output Grouping) and openspec/specs/ux-patterns/spec.md (Interactive Update Tool Selection, Progress Indication — requires atomic, non-interleaved concurrent progress).

## Affected Areas

- `internal/cli/update.go` — interactive gate, pre-check invocation, carried-outcome loop (redesign target for flow ordering/live display).
- `internal/cli/check.go` — `runChecks`/`safeCheck`/`checkOutcome` shared engine; also home of the `check` command and the self-update hint (removal-impact epicenter).
- `internal/output/render.go` — `ProgressInPlace` (single-line counter), `UpdateSummary`/`CheckSummary`, `SelfUpdateHint`.
- `internal/output/selector.go` — `CheckboxSelector`, `SelectOption{ID, Label, Version}` (already supports inline "cur → latest").
- `internal/cli/parser.go` — command registration (line 68 registers check; help grouping lists check under Commands).
- `internal/cli/deps.go` — `cliDeps.check` injection slot.
- Tests: `internal/cli/check_test.go`, `check_hint_test.go` (8 hint tests drive `runCheck`), `integration_test.go` (many `root.SetArgs([]string{"check"})`), `help_test.go:50` and `parser_test.go:265` (assert check exists), `update_test.go` (interactive-path tests deliberately assert "Checking X/Y" progress lines, design D5).
- `scripts/smoke-test.sh` — ~9 assertions exercise `check` (help text, plain, `--quiet`/`-q`, `--verbose`/`-v`, `--only`/`--skip`).
- `.github/workflows/ci.yml` — "Smoke test" step runs smoke-test.sh (transitive breakage only; no direct `upp check` invocation).
- Specs: `openspec/specs/command-interface/spec.md` (Command Structure table, `Requirement: upp check`, Help Output Grouping), `openspec/specs/ux-patterns/spec.md` (Progress Indication "Checking X/Y", Self-Update Detection Hint, List Table Output scenario uses `upp check --only apt`).
- `README.md` — lines 65 (usage block), 91 (command table), 109 (hint description), 144 (config comment).

## Gap Analysis vs Desired Flow

Desired: during `upp update`, show a LIVE per-tool search display (one line per tool, check mark when its search finishes, showing current → new version), THEN a selection prompt listing only tools WITH pending updates.

Already satisfied:
- The selection prompt already lists ONLY pending tools (pending filter at update.go:282-291).
- Version data ("cur → latest") is available at each check's completion (`checkOutcome.updateInfo`; `result.Version` already formatted).

Missing:
- The pre-check renders ONE mutating counter line (`ProgressInPlace`), not a persistent per-tool status board. The desired UX needs a multi-line live board: N stable lines rendered up-front, each flipping pending/spinner → ✓ (or ✗) with `cur → latest` as its worker completes.
- `runChecks` currently calls `r.ProgressInPlace` inline per completion (check.go:171-173); there is no per-tool completion hook a board could subscribe to.

Implementation surface implied by the gap:
- A new output-layer component (e.g., a live check board in `internal/output`) owning multi-line ANSI cursor control, mutex-synchronized across workers (ux-patterns mandates atomic, non-interleaved progress).
- A completion-callback seam on `runChecks` (or board injection) replacing/augmenting the inline `ProgressInPlace` call.
- Non-color/non-TTY fallback must stay line-based (precedent: `ProgressInPlace` branches on `r.color`).
- Final summary ordering is unaffected (index slotting already guarantees deterministic order regardless of completion order).

## Check Subcommand Removal Impact

Verdict: removal is mechanically feasible; nothing outside `internal/cli` calls `check` programmatically. But it breaks CI transitively and carries two product-level consequences.

Breaks (must change):
- `internal/cli/parser.go:68` — registration; `help_test.go:50` and `parser_test.go:265` assert its existence; help grouping spec lists it.
- `scripts/smoke-test.sh` — Tests 2, 5, 7, 8, 9 (~9 assertions) use `check`; the CI "Smoke test" step fails until rewritten (e.g., route quiet/verbose/filter coverage through `list`/`update --dry-run`).
- Tests: `check_test.go` (`runCheck`-level cases; `runChecks`/`safeCheck` mechanism tests survive if the functions stay), `check_hint_test.go` (all 8 tests drive `runCheck`), `integration_test.go` (10+ invocations of `root.SetArgs([]string{"check"})`).

Needs doc/spec delta (not a code break):
- command-interface spec: subcommand list/table, `Requirement: upp check`, Help Output Grouping (`Commands`: check, list, update).
- ux-patterns spec: Progress Indication, Self-Update Detection Hint, List Table Output scenario.
- README lines 65, 91, 109, 144.

Must survive removal (shared machinery living in check.go):
- `runChecks`, `safeCheck`, `checkOutcome`, `checkJob`, `defaultConcurrency` — the interactive update pre-check depends on all of them. Relocate or retain; do not delete with the command.

Product consequences:
- Self-update hint becomes unreachable: `maybeShowSelfUpdateHint` is wired only into `runCheck`. Removing check without rewiring kills the hint (and contradicts ux-patterns "Self-Update Detection Hint"). Note the README's claim that bare `upp` hints is NOT true in code today — either fix the README or wire the hint into the dashboard as part of this change.
- Loss of the only zero-mutation query surface: `upp update` without `-n` MUTATES in non-TTY/`--ci`/`--quiet` runs. After removal, `update --dry-run` is the sole read-only path. External scripts calling `upp check` (not visible from this repo) would fail with an unknown-command error.

## Approaches

1. **Live status board via a `runChecks` completion hook** — add an optional per-tool completion callback (or board interface) to `runChecks`; a new output component renders N stable lines, flipping each to ✓/✗ with `cur → latest` on completion; selector flow unchanged afterwards.
   - Pros: minimal blast radius (output layer + one seam in check.go); preserves carried-outcome design D4/D6/D7 and all bypass contracts; hermetically testable via existing seams; clean non-TTY fallback (line per completion).
   - Cons: new multi-line ANSI management code; short-terminal/resize edge cases; update_test.go progress-line assertions need updating.
   - Effort: Medium

2. **Post-hoc listing** — keep the counter during the pool, then print one ✓/✗ line per tool (with versions) after all checks finish, before the selector.
   - Pros: tiny effort; zero concurrency-rendering risk.
   - Cons: not live — the user stares at a counter during the slow phase; does not meet the stated intent.
   - Effort: Low

3. **Full event-driven TUI** — spinner frames with a ticker-driven redraw loop over the board.
   - Pros: richest UX.
   - Cons: timer goroutines interacting with raw mode, larger race surface, overkill for subprocess-bounded checks, hardest to test hermetically.
   - Effort: High

## Recommendation

Approach 1. The data (carried `updateInfo` with both versions) and the concurrency engine (`runChecks`) already exist; the work is concentrated in a new output-layer board plus a completion-hook seam on `runChecks`. All security gating, bypass contracts, and the carried-outcome loop stay untouched. For the `check` removal question: recommend the proposal treat removal as a separate decision with explicit deltas (smoke-test rewrite, hint rewiring to the dashboard or drop, read-only story via `update --dry-run`) rather than bundling it silently into the unified-flow change.

## Risks

- WARNING — Self-update hint regression: hint is reachable only via `upp check` today; removing check without rewiring breaks ux-patterns "Self-Update Detection Hint" (and exposes the stale README claim about bare `upp`).
- WARNING — Read-only surface loss: scripted/CI consumers of `upp check` get an unknown-command failure; `update --dry-run` becomes the only non-mutating path.
- WARNING — Smoke-test/CI breakage is guaranteed until smoke-test.sh is rewritten (9 assertions); shellcheck/actionlint gates also cover that script.
- SUGGESTION — Live board must honor the atomic-progress requirement (mutex) and quiet/non-TTY fallbacks; keep `ProgressInPlace` semantics for any remaining counter-style paths.
- SUGGESTION — update_test.go interactive tests deliberately assert "Checking X/Y" lines (design D5); the board changes those expectations — plan test updates in the proposal.
- SUGGESTION — Shared check machinery lives in check.go; if the command is removed, relocate `runChecks`/`safeCheck`/`checkOutcome` rather than deleting the file wholesale.

## Ready for Proposal

Yes. The orchestrator can proceed to `propose`: intent is check-first-then-select for `upp update` with a live per-tool pre-check display; scope should include the output-layer board, the `runChecks` hook, and an explicit decision (with its own deltas) on whether `check` is removed in this change or a follow-up.
