# Archive Report: upp-unified-update-flow

**Change**: upp-unified-update-flow
**Archived**: 2026-08-21 (ISO prefix `2026-08-21-upp-unified-update-flow`)
**Mode**: hybrid (OpenSpec filesystem + Engram)
**Project**: upp — Go binary, cobra CLI, TOML config
**Branch at archive**: `wu5/spec-docs-e2e` (HEAD `285fae2` pre-archive work)
**Verdict at close**: PASS WITH WARNINGS — archived as intentional-with-warnings (two carried SUGGESTION-1/2 stale-text items were fixed AT ARCHIVE in this cycle, see below)

## Final State (at close)

- **Delivery**: feature-branch-chain on tracker `feature/unified-update-flow`. Child PRs #97–#101 (work units wu1..wu5) are **OPEN on GitHub** — merging is user-managed there, matching how prior archives in this repo closed (e.g. `2026-08-19-upp-tui-improvement`). Canonical spec deltas were applied regardless, per established repo convention.
- **Tasks**: 18/18 complete (`tasks.md` has zero unchecked implementation tasks; 18 `[x]`). Runtime ledger state: `complete`.
- **Verification** (final `verify-report.md`, validator-valid): `pass_with_warnings`, 0 CRITICAL, 53/53 scenarios COMPLIANT, 10/10 requirements (7 MODIFIED/ADDED tables + 3 REMOVED mandates). Evidence `evidence_revision: sha256:8f6ecd06d45a57f6ca0171cd70886c03bb2cd9b05955acbf989a3e346bedd2e3`. At verification time (`285fae2`): full suite 329 top-level / 935 incl. subtests, 0 failed; `-race` clean on changed packages; CI-pinned golangci-lint v2.12.2 0 issues; smoke 31/31; `go build ./...` clean.
- **Verify WARNING-1** (lint toolchain skew) was resolved at the re-verification and carried as a resolved record only.

## Archive-time refreshes (carried SUGGESTION-1/2, fixed in this cycle)

Both non-blocking SUGGESTIONs from the verify report were refreshed at archive as directed by the orchestrator; post-edit `go build ./...` and `go test ./internal/cli/ -count=1` stayed green (comment/config-only, no behavior change):

1. **SUGGESTION-1** — `internal/cli/parser.go:29` comment reworded to `Running \`upp\` with no args shows the dashboard welcome screen, read-only.`
2. **SUGGESTION-2** — `openspec/config.yaml`: `check` removed from the cobra subcommand list (`list/update/init/self-update`) and from the apply guideline (`Keep upp list/update/init/self-update and ...`); all other bytes untouched.

Both refreshes landed in commit `4363525` together with the canonical spec sync.

## Spec Sync (delta → main specs)

| Domain | Action | Result |
|--------|--------|--------|
| `command-interface` | MODIFIED "Command Structure", "Help Output Grouping", "Self-Update Flag Semantics" (3); REMOVED "`upp check`" (1) | `openspec/specs/command-interface/spec.md` (7 → 6 requirements); all other requirements preserved byte-identical |
| `config-system` | REMOVED "Self-Update Detection Setting" (1) | `openspec/specs/config-system/spec.md` (5 → 4 requirements); all other requirements preserved byte-identical |
| `self-update` | MODIFIED "GitHub Release Detection" (1) | `openspec/specs/self-update/spec.md` (8 → 8 requirements); all other requirements preserved byte-identical |
| `ux-patterns` | ADDED "Live Check Board" (1, 8 scenarios); MODIFIED "Progress Indication", "List Table Output", "Bare Dashboard Welcome Screen", "Summary Report", "Verbose Error Diagnostics Rendering" (5); REMOVED "Self-Update Detection Hint" (1) | `openspec/specs/ux-patterns/spec.md` (13 → 13 requirements); all other requirements preserved byte-identical |

All 4 deltas carried explicit `Reason`/`Migration` notes for each REMOVED requirement (satisfying the removal precondition). Both merges were mechanical: requirement blocks were spliced into the canonical files by exact byte regions (shell/Python byte assembly, no model reproduction of artifact content), and every replaced/inserted block was re-extracted from the merged file and byte-compared against the delta source block — **10/10 blocks byte-identical**. A post-merge sweep confirmed zero `check_self_update` references and zero `upp check` references outside `(Previously: ...)` history notes in `openspec/specs/`.

## Archive Move

`openspec/changes/upp-unified-update-flow/` → `openspec/changes/archive/2026-08-21-upp-unified-update-flow/` via `git mv` (tracked `apply-progress.md` + `tasks.md` recorded as renames; the untracked planning files — design.md, explore.md, proposal.md, specs/, verify-report.md — moved with the directory). Pre-move recursive snapshot (`cp -R` to a temp root) compared with `diff -r` after the move: **EMPTY (exit 0, byte-identical)**. Source directory confirmed absent post-move. Archive contains all artifacts: proposal.md, explore.md, design.md, specs/ (4 domain deltas), tasks.md, apply-progress.md, verify-report.md. Active changes directory no longer contains this change.

## Native Review Receipt Gate

`reviewGate` is structurally ABSENT: no review was started for this candidate; no review code ran and nothing to read or block on. Archive proceeded under ordinary repository policy. No review topics exist for this candidate.

## Task Completion Gate

- Persisted `tasks.md` (source of truth for hybrid): 0 unchecked, 18/18 `[x]`. Gate passed without repair; no stale-checkbox reconciliation was needed.
- No contradictions between sources: orchestrator final-state facts (18/18, ledger `complete`), persisted `tasks.md`, and the validator-valid `verify-report.md` agree.

## Archive Persistence

- Filesystem: `openspec/changes/archive/2026-08-21-upp-unified-update-flow/archive-report.md` (additive, excluded from the `diff -r` readback)
- Engram: `sdd/upp-unified-update-flow/archive-report` (type architecture, capture_prompt false) and final-state update to `sdd/upp-unified-update-flow/verify-status` (type decision, capture_prompt false)

## Residual Follow-ups (non-blocking)

1. **Merge the PR chain**: PRs #97–#101 remain open on tracker `feature/unified-update-flow`; merging into main is user-managed on GitHub.
2. **Lint pin discipline** (verify WARNING-1, resolved record): keep linting via the CI-pinned golangci-lint v2.12.2 (or a version gate) so a future environment without it cannot reintroduce the vacuous-pass skew.

## SDD Cycle Status

**Complete.** The change was explored, proposed, specified (4 delta specs, 10 requirements), designed, implemented (18 tasks, 5 work units as PR chain #97–#101), verified (pass_with_warnings, 0 CRITICAL, 53/53 scenarios), and archived. Source of truth (`openspec/specs/`) reflects the new behavior: `check` and the self-update detection hint are gone; `upp update --dry-run` is the sole read-only query surface; self-update detection is exclusive to `upp self-update`. Ready for the next change.
