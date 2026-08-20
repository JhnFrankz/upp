# Proposal: Interactive tool selection for `upp update`

## Intent

`upp update` is all-or-nothing per run: users cannot choose which pending tools to update. Add a TTY checkbox selector so interactive runs let the user confirm/narrow the tool set before any update executes — while `--ci`, non-TTY, `--quiet`, and `--dry-run` keep today's exact contracts.

## Scope

### In Scope
- Checkbox selector (hand-rolled ANSI, `internal/output`) with ↑/↓, Space, `a`, `n`, Enter, Esc/q
- Concurrent pre-check phase feeding the selector (`runChecks` extracted from `check.go`)
- Update loop over user-selected subset only; per-row `Current → Latest` inline
- Delta specs for `ux-patterns` + `command-interface`

### Out of Scope
- Full diff/side-by-side pane (v2; pushes toward a TUI library)
- Paging/scrolling, mouse support, configurable keys, `interactive` config key (forbidden by config-system)
- New flags; changes to `--ci`/`--quiet`/`--dry-run`/`--only`/`--skip` semantics

## Capabilities

### New Capabilities
None

### Modified Capabilities
- `ux-patterns`: new Requirement "Interactive Update Tool Selection" — selector shown by default in TTY `upp update`; skipped when `--ci`, non-TTY, `--quiet`, or `--dry-run`; all pending tools pre-checked; Enter = update all; Esc/q cancels run (nothing updated, clear message); no pending updates → selector skipped, normal summary. Selector is a user-choice UI, NOT a security confirmation — `security.ConfirmAction` still runs per tool.
- `command-interface`: MODIFIED `upp update` requirement — TTY runs render the selector over the `--only`/`--skip`-filtered set; selection narrows further; no flag semantics change. `--dry-run` stays non-interactive.

## Approach

Hand-rolled ANSI selector per exploration recommendation: `CheckboxSelector` type in `internal/output/selector.go` with injectable `(w io.Writer, r io.Reader, opts []SelectOption)` (mirrors `ConfirmConfig.Reader`). Gate: `TTY && !ci && !quiet && !dry-run`. `update` extracts `runChecks` from `check.go`, pre-checks concurrently, renders selector from `StatusAvailable` results, then runs its existing per-tool loop (security matrix + policy gating intact) over selected IDs. Esc/q → nothing updates, exit 0. Raw mode via `golang.org/x/term.MakeRaw` with `defer` restore; non-TTY never renders (callers gate on `stdinIsTTY`).

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/output/selector.go` | New | CheckboxSelector: key handling, ANSI render, raw mode |
| `internal/output/render.go` | Modified | Selector rendering helpers; existing methods byte-identical |
| `internal/cli/update.go` | Modified | Pre-check phase + selector gate + selected-subset loop |
| `internal/cli/check.go` | Modified | Extract reusable `runChecks` worker pool |
| `internal/cli/update_test.go`, `check_test.go`, `internal/output/selector_test.go` | Modified/New | Table-driven: keys, toggle, cancel, non-TTY bypass, determinism |
| `openspec/specs/ux-patterns/spec.md`, `command-interface/spec.md` | Modified | Delta specs in change folder |
| `go.mod` | Modified | `golang.org/x/term` only |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Raw-mode state not restored on panic/signal | Med | `defer` restore + RED test via injectable seam |
| Windows console arrow-key handling | Med | `x/term` handles raw mode; verify in build-all matrix |
| Double `Check()` for gated tools | Med | Carry pre-check results into loop; gating semantics unchanged |
| Pre-check "Checking X/Y" appears in captured output (existing tests) | Med | Gate pre-check progress on selector rendering (TTY); update tests deliberately |
| Spec collisions (selector vs Default Interactive Mode) | Low | Framed as TTY user-choice UI; security prompts unchanged |

## Rollback Plan

Revert the selector commit (or flip the gate to skip rendering) — non-TTY/`--ci`/`--quiet`/`--dry-run` paths are byte-identical today, so reverting restores current behavior with zero migration. `runChecks` extraction is additive; keep it on revert.

## Dependencies

- `golang.org/x/term` (for `MakeRaw`/restore) — decision: YES, adopt. Pure-Go, stdlib-adjacent, needed for reliable key reading; verify build-all targets. No other new deps.

## Success Criteria

- [ ] `go test ./... -count=1 -race` + smoke test pass
- [ ] TTY run: selector shows all pending checked; Enter updates all (matches today's behavior)
- [ ] Esc/q cancels with nothing updated, exit 0
- [ ] `--ci`/non-TTY/`--quiet`/`--dry-run` output byte-identical to today
- [ ] No pending updates → no selector, normal summary
- [ ] Deselecting a tool skips its update; summary counts reflect selection
