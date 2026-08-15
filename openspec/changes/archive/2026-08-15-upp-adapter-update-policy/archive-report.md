# Archive Report: upp-adapter-update-policy

**Archived**: 2026-08-15
**Archive path**: `openspec/changes/archive/2026-08-15-upp-adapter-update-policy/`
**Artifact store mode**: hybrid (filesystem merge + archive move + Engram archive report)
**Status**: SUCCESS — SDD cycle complete

## Final State

- **Tasks**: 19/19 complete (`tasks.md` — 19 `[x]`, 0 `[ ]`; all four phases 1.1–4.6 checked). Task Completion Gate passed; no stale checkboxes, no reconciliation needed.
- **Verify verdict**: PASS — 0 blockers, 0 critical findings, 0 warnings (per `verify-report.md`, observation #232). Evidence revision: main @ 7f8ad023. All gates green on fresh execution: `go test ./... -count=1` exit 0 (9 packages, 8 with test files), `-race` exit 0, `go vet ./...` clean, `gofmt -l internal/` empty, smoke 23/23 against the freshly built binary, focused `TestCheck|TestCommandOutputErr|TestShellOutputErr` **57/57 subtests** (49 TestCheck rows incl. real `sh -c exit N` children + 4 + 4 helper subtests). Compliance matrix: **2/2 requirements, 12/12 scenarios** (Update Gating 7, Check Failure Signal 5).
- **Delivery (complete at close)**: BOTH PRs MERGED on main @ 7f8ad02 — PR #45 (d8d9963, `UpdatePolicy` enum + gate rewrite + explicitness) and PR #46 (7f8ad02, check failure signal + portability correction). Issue #44 closed. The review for PR #46 went through RDD recovery (`scope_changed`, maintainer-authorized) + reopen-results (4 lenses quarantined, re-reviewed on the corrected candidate) and was APPROVED — per the orchestrator's final-state handoff (final-state authority rank 3); the review lifecycle is orchestrator/review-system tracked, no review artifact files exist inside the change folder (no `state.yaml`, no `review/`).
- **Documented deviation (1, non-breaking, spec-compliant)**: D4 mechanism — design.md and apply-progress.md:38 claim shell `timeout 15` is kept for npm/pnpm; the implementation REMOVED the shell wrapper in favor of direct exec bounded by `runCmdArgsFn`'s `CheckTimeout` (npm.go:30-36, pnpm.go:32-38) — portable (no GNU `timeout` on macOS). Literal GNU-timeout exit-124 is no longer observable; timeouts surface as `DeadlineExceeded`-structured errors. Spec scenarios still hold. Flagged as docs drift (see follow-up 3).
- **Implementation**: `internal/adapters/interface.go` (`UpdatePolicy` enum, `PolicyGated=0`/`PolicyAlwaysUpdate=1`, ToolInfo field), 13 `Info()` sites (apt/npm/npm/pnpm → Gated; 9 others → AlwaysUpdate), `internal/cli/update.go` gate rewrite (:171 predicate, `gatedOfficialAdapters` deleted), `internal/adapters/official/helper.go` (`commandOutputErr`/`shellOutputErr`/`commandFailureErr`/`isExitCode`, `%w`-chained structured errors), `apt.go`/`nvm.go`/`npm.go`/`pnpm.go` error-aware detection reads + maskless exit interpretation; tests: `info_test.go`, `adapter_update_test.go`, `check_test.go`, `cli/update_test.go` (matrix re-keyed ID→policy), `integration_test.go`, `audit_probe_test.go` (21 explicit fake literals).
- **Follow-ups recorded (verify 6 SUGGESTIONs + review 2, all out of scope)**: (1) D4 docs drift — design.md/apply-progress claim `timeout 15` kept; code uses Go `CheckTimeout`; (2) nvm `Update()` still sources hardcoded `~/.nvm/nvm.sh` (nvm.go:90) ignoring `NVM_DIR` while `Check()` honours it (nvm.go:46,57) — pre-existing; (3) `runCmdArgsFn` (helper.go:68-85) lacks the process-group kill (`Setpgid`/negative-pid SIGKILL) + `WaitDelay` treatment `runCmdFn` has — a grandchild-spawning `outdated` process is not group-killed on timeout; pre-existing gap; (4) check_test.go exit-code harness wires the real child error after `setExecFakes` (check_test.go:591-608) — order-dependent, brittle to fake redesign; (5) apt `Check()` depends on bash `-o pipefail` (apt.go:27,37) — dash-only environments fail detection with a structured error; pre-existing; (6) npm exit-1 availability decided by stdout shape (npm.go:39,43) — exit 1 + empty stdout treated as operational failure; (7) fake wiring aliasing (review); (8) duplicated command literals (review).

## Pending Delivery (orchestrator-owned, NOT part of archive)

The change folder artifacts are **untracked** in git (docs delivery pending). Both PRs' code is merged on main @ 7f8ad02; the archive folder + main-spec sync are the remaining docs. After archive, the orchestrator creates the docs PR (archive folder + `openspec/specs/tool-adapter/spec.md` merge). Rollback boundary for the code remains `git revert` of the two merge commits (PR #45 revert restores the CLI-side ID-list gate; PR #46 revert restores silent `check()` failure swallowing).

## Gates

| Gate | Result |
|------|--------|
| Task Completion Gate | PASSED — 19 `[x]`, 0 `[ ]` in persisted `tasks.md` |
| Native Review Receipt Gate | Review conducted and APPROVED (PR #46: recovery `scope_changed` → maintainer-authorized → reopen-results, 4 lenses quarantined and re-reviewed on the corrected candidate) — per orchestrator final-state handoff; receipt tracked by the review system, archive proceeds on the documented approval |
| CRITICAL findings gate | No CRITICAL issues in verify-report (0/0) |
| Action Context Guard | Repo-local archive operations inside the repo root; no workspace-planning mode; no `allowedEditRoots` restriction violated |

## Spec Sync (delta → main)

| Domain | Action | Details |
|--------|--------|---------|
| tool-adapter | Updated | **MODIFIED Update Gating REPLACED** the old ID-list-wording requirement block (old 6-scenario block removed) with the delta's byte-faithful requirement (policy wording + `(Previously: ...)` note + 7 scenarios incl. 'Gated check fails'); **ADDED Check Failure Signal APPENDED** (5 scenarios) after Go Adapter Architecture. All 6 other requirements preserved byte-identical (Adapter Interface, Official Adapter Catalog, Adapter Error Handling, Version Comparison, Subprocess Timeouts, Go Adapter Architecture). |

**Sync decision recorded**: unlike the previous archived change (pure append), this delta carries a MODIFIED requirement, so the merge is replace-then-append: the old Update Gating block was fully replaced by the delta's MODIFIED block — byte-faithful, including the `(Previously: ...)` parenthetical kept verbatim (documents the superseded ID-list mechanism; consistent with the OpenSpec MODIFIED-replaces-full-block convention). The ADDED block was appended byte-faithfully. Resulting baseline reads coherently: Update Gating (policy wording, 7 scenarios) followed by Subprocess Timeouts and Go Adapter Architecture, then Check Failure Signal. Non-destructive to other requirements; no `rules.archive` destructive-warning trigger.

Merge verification (8/8 byte-level checks PASS): prefix preserved, MODIFIED block byte-identical to delta, middle (Subprocess Timeouts + Go Adapter Architecture) preserved byte-identical, ADDED block byte-identical to delta, 8 requirement headings total, old ID-list wording gone, policy wording present, other 6 headings intact. Unified diff of pre-merge snapshot vs merged: `@@ -72,16 +72,18 @@` (replacement) and `@@ -102,3 +104,16 @@` (append). Installed file `cmp`-verified byte-identical to the verified merged temp.

Main spec updated: `openspec/specs/tool-adapter/spec.md` (104 → 119 lines).

## Mechanical Copy Evidence

Per the Mechanical Copy Contract: pre-move recursive snapshot → `git mv` attempted (refused: "fatal: source directory is empty" — files untracked, so git refused the move) → `mv` fallback succeeded → source-gone check passed → `diff -r` readback:

```
(diff -r output: empty — no differences)
```

Empty `diff -r` is the only passing evidence; `archive-report.md` is additive-only and excluded from the comparison.

## Archive Contents

- `proposal.md` ✅
- `specs/tool-adapter/spec.md` ✅ (delta — MODIFIED Update Gating 7 scenarios + ADDED Check Failure Signal 5 scenarios, 12 total)
- `design.md` ✅
- `tasks.md` ✅ (19/19 tasks complete, no unchecked)
- `apply-progress.md` ✅ (slices 1+2 + corrections)
- `verify-report.md` ✅ (PASS)
- `archive-report.md` ✅ (this file, additive)

Active changes directory (`openspec/changes/`) no longer contains this change (only `archive/` remains).

## Engram Lineage (observation IDs read)

| Artifact | Engram observation |
|----------|--------------------|
| proposal | #227 |
| spec (delta) | #228 |
| design | #229 |
| tasks | #230 |
| apply-progress | #231 |
| verify-report | #232 |
| archive-report | saved under topic `sdd/upp-adapter-update-policy/archive-report` |

## SDD Cycle Complete

The change has been fully planned, implemented (19/19 tasks, PRs #45 + #46 merged on main @ 7f8ad02 with the PR #46 review recovered and approved), verified (PASS, 0 blockers, 0 critical), and archived. Delta spec synced into the tool-adapter main spec (MODIFIED replaced + ADDED appended). Docs delivery (archive folder + spec sync PR) is orchestrator-owned and pending. Ready for the next change.
