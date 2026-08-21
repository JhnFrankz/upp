# Proposal: Unified Update Flow — Live Check Board, Remove `check`

## Intent

`upp update` hides its search behind one mutating counter (`Checking X/Y`); the self-update hint lives only in `check`. Unify: check live, show per-tool results, then ask what to update — removing `check` and the hint.

## Scope

### In Scope
- **Live check board** (TTY interactive update): one stable line per tool from the start; each flips to ✓ with `current → new` when its own search completes; up-to-date tools stay visible ✓ up-to-date; then CheckboxSelector renders over pending-only tools.
- **Remove `upp check`**: registration (`parser.go:68`), `cliDeps.check` slot, tests, rewrite ~9 `smoke-test.sh` assertions (Tests 2/5/7/8/9), spec deltas, README lines 65/91/109/144.
- **Remove self-update hint entirely**: `maybeShowSelfUpdateHint`, `settings.check_self_update`, `SelfUpdateHint`, 8 hint tests. No surface inherits it; self-update stays manual via `upp self-update`; drop README's false bare-`upp` claim.
- **Relocate shared machinery** (`runChecks`/`safeCheck`/`checkOutcome`/`checkJob`/`defaultConcurrency`) out of `check.go`; the update pre-check needs them. Relocate, never delete.
- Accept `update --dry-run` as sole read-only/query surface for scripts and CI.

### Out of Scope / Non-Goals
- Manager grouping (docker under apt/winget) — future change.
- CheckboxSelector redesign (stays as delivered by upp-tui-improvement).
- Sequential/non-TTY redesign; security-gating changes.

## Capabilities

### New Capabilities
None.

### Modified Capabilities
- `command-interface`: drop `Requirement: upp check`; Command Structure and Help Output Grouping lose `check`.
- `ux-patterns`: Progress Indication gains the live board (atomic, non-interleaved); Self-Update Detection Hint removed; List Table Output re-anchored (was `upp check --only apt`).
- `self-update`: hint path removed (hint-silence mandate and `upp check` API-failure scenario go).

## Approach

Exploration Approach 1: new output-layer board (`internal/output`) owning multi-line ANSI cursor control, mutex-synchronized across workers; completion-callback seam on `runChecks` replaces inline `ProgressInPlace` (check.go:171-173). Non-color/non-TTY falls back to line-per-completion (precedent: `r.color`). Carried-outcome design D4/D6/D7 and bypass contracts (`--ci`/`--quiet`/`--dry-run`/non-TTY) untouched.

## Affected Areas

- `internal/output/`: render.go modified; new board component
- `internal/cli/check.go`, `update.go`, `parser.go`, `deps.go`: relocate engine; delete command/hint; wire board; drop registration
- `scripts/smoke-test.sh`; CLI tests (check, check_hint, integration, help, parser, update): rewritten/updated
- Specs (command-interface, ux-patterns, self-update); README.md: deltas

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| CI smoke-test step fails until rewritten | High | Rewrite in this change via `list`/`update --dry-run` |
| External `upp check` callers break | Medium | Accepted (decision 4); document `--dry-run` |
| `update_test.go` asserts "Checking X/Y" | High | Update expectations alongside board |
| Multi-line ANSI edge cases | Medium | Mutex + non-color fallback |

## Rollback Plan

Single branch: `git revert` restores `check`, hint, counter UX atomically. Spec deltas apply at archive.

## Success Criteria

- [ ] TTY `upp update`: persistent per-tool board; per-line completion flips; up-to-date visible ✓; pending-only selector after
- [ ] `upp check` unknown-command; smoke-test.sh green in CI
- [ ] No code path renders the hint; manual `upp self-update` intact
- [ ] Machinery survives relocation; carried-outcome tests pass; flag contracts preserved
