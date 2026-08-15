# Archive Report: upp-adapter-update-correctness

**Archived**: 2026-08-14
**Archive path**: `openspec/changes/archive/2026-08-14-upp-adapter-update-correctness/`
**Artifact store mode**: hybrid (filesystem merge + archive move + Engram archive report)
**Status**: SUCCESS — SDD cycle complete

## Final State

- **Tasks**: 19/19 complete (`tasks.md` — 19 `[x]`, 0 `[ ]`; structured status `taskProgress.allComplete: true`). Task Completion Gate passed; no stale checkboxes, no reconciliation needed.
- **Verify verdict**: PASS — 0 blockers, 0 critical findings, 0 warnings (per `verify-report.md`, observation #212). Evidence revision: main @ e758d409. All gates green on fresh execution: `go test ./... -count=1` exit 0 (9 Go packages, 8 with test files), `-race` exit 0, `go vet ./...` clean, `gofmt -l internal/` empty, smoke 23/23 against a freshly built binary, GroupKill survivor probe PASS, focused gating/timeout 10/10 subtests.
- **Delivery (complete at close)**: BOTH chained PRs ALREADY MERGED on main @ e758d40 — PR #38 (timeouts + linux/arm64, merge commit 401bb4f) and PR #39 (gating + seam + timeoutErr, merge commit e758d40). Two review corrections delivered through the PR reviews, both approved via targeted validation: (1) process-group kill (`Setpgid` + negative-pid SIGKILL + `WaitDelay=5s`, proven live by `TestRunCmd_GroupKillProvesGrandchildrenDie`) in PR #38's review; (2) stub exemption in PR #39's review — `gatedOfficialAdapters` restricted to apt/npm/nvm/pnpm so brew/bun/docker/gh/go/opencode (stubs, `update_available=false` by design) still update; winget/scoop always update by design. No CRITICAL, no blockers, no WARNINGs; 3 SUGGESTION-level follow-ups recorded (see below).
- **Corrected counts (final state)**: the authoritative delta spec at close has **3 requirements / 12 scenarios** (Update Gating 6 incl. the stub-exemption correction rows 'Stub official exempt' and 'winget/scoop exempt' as distinct rows, Subprocess Timeouts 4, Go Adapter Architecture 2). The delta spec was UPDATED during PR #39's correction; the filesystem delta is the authoritative spec and was the basis for both verify (12/12) and this archive's spec sync. NOTE: Engram observation #207 (`sdd/upp-adapter-update-correctness/spec`, persisted 2026-08-14 18:25) is a PRE-correction snapshot ("3 ADDED requirements, 11 scenarios") and is superseded by the filesystem delta — do not treat it as current.
- **Native review**: no native RDD review was started for this candidate — structured status reports `reviewGate` structurally absent (all `reviewPolicy/Ledger/Receipt/Bundle/Context/State` paths missing, no `state.yaml` in the change folder). Archive proceeded under ordinary repository policy. The PR reviews referenced above are GitHub reviews, not native receipts; absence of native review is not a defect.
- **Implementation**: `internal/adapters/timeouts.go` (CheckTimeout=30s, UpdateTimeout=10m), `internal/adapters/custom.go` (`shellExecWithTimeout`), `internal/adapters/official/helper.go` (seam bodies → CommandContext+WithTimeout), `internal/adapters/official/go.go` (`goTarballURL(goarch)` + `runtime.GOARCH`), `internal/cli/update.go` (`updateDeps` seam + gating block + `timeoutErr`), `internal/cli/check.go` (`timeoutErr` at check site); tests: `timeout_test.go`, `go_arch_test.go`, `custom_test.go` additions, `update_test.go`. Two documented non-structural design deviations (matrix asserts summary labels + fake flag instead of per-tool icon lines; check.go:91 `timeoutErr` wiring without direct unit test).
- **Follow-up suggestions (verify-report, open, out of scope)**: (1) `upp check` never exits non-zero on check failures (pre-existing, check.go:114-116); (2) `timeoutErr` reads the timeout vars at error-mapping time, not at context creation (limit drift potential; current tests consistent); (3) `gatedOfficialAdapters` is a hardcoded ID list (update.go:48-53) — an adapter gaining real detection later must be added manually.

## Pending Delivery (orchestrator-owned, NOT part of archive)

The change folder artifacts are **untracked** in git (docs delivery pending). Both PRs' code is merged on main @ e758d40; the archive folder + main-spec sync are the remaining docs. After archive, the orchestrator creates the docs PR (archive folder + `openspec/specs/tool-adapter/spec.md` merge). Rollback boundary for the code remains `git revert` of the two merge commits (PR #38 revert restores no-timeouts/amd64-only URL; PR #39 revert restores unconditional updates).

## Gates

| Gate | Result |
|------|--------|
| Task Completion Gate | PASSED — 19 `[x]`, 0 `[ ]` in persisted `tasks.md`; structured status `allComplete: true` |
| Native Review Receipt Gate | reviewGate structurally absent → proceed under ordinary repository policy |
| CRITICAL findings gate | No CRITICAL issues in verify-report (0/0) |
| Action Context Guard | `actionContext.mode: repo-local` (not workspace-planning); `allowedEditRoots: [<repo-root>]` — all operations inside the repo root |

## Spec Sync (delta → main)

| Domain | Action | Details |
|--------|--------|---------|
| tool-adapter | Updated | 3 ADDED requirements appended byte-faithfully to `openspec/specs/tool-adapter/spec.md` after the existing 4: "Update Gating" (6 scenarios incl. stub exemption correction), "Subprocess Timeouts" (4 scenarios), "Go Adapter Architecture" (2 scenarios). All 4 pre-existing requirements preserved unchanged (Adapter Interface, Official Adapter Catalog, Adapter Error Handling, Version Comparison). Non-destructive — no MODIFIED/REMOVED/RENAMED, no `rules.archive` warning required. Merge verified: `diff` of pre-merge snapshot vs merged shows pure append (`71a72,104`); appended block byte-identical to the delta spec (`diff` empty). |

Main spec updated: `openspec/specs/tool-adapter/spec.md` (71 → 104 lines).

## Mechanical Copy Evidence

Per the Mechanical Copy Contract: pre-move recursive snapshot → `git mv` attempted (refused: "fatal: source directory is empty" — files untracked, so git refused the move) → `mv` fallback succeeded → source-gone check passed → `diff -r` readback:

```
(diff -r output: empty — no differences)
```

Empty `diff -r` is the only passing evidence; `archive-report.md` is additive-only and excluded from the comparison.

## Archive Contents

- `proposal.md` ✅
- `specs/tool-adapter/spec.md` ✅ (delta — 3 ADDED requirements, 12 scenarios)
- `design.md` ✅
- `tasks.md` ✅ (19/19 tasks complete, no unchecked)
- `apply-progress.md` ✅ (slices 1+2 + corrections)
- `verify-report.md` ✅ (PASS)
- `archive-report.md` ✅ (this file, additive)

Active changes directory (`openspec/changes/`) no longer contains this change (only `archive/` remains).

## Engram Lineage (observation IDs read)

| Artifact | Engram observation |
|----------|--------------------|
| proposal | #206 |
| spec (delta) | #207 — ⚠️ pre-correction snapshot (11 scenarios); superseded by the filesystem delta (12 scenarios) |
| design | #208 |
| tasks | #209 |
| apply-progress | #210 |
| verify-report | #212 |
| archive-report | saved under topic `sdd/upp-adapter-update-correctness/archive-report` |

## SDD Cycle Complete

The change has been fully planned, implemented (19/19 tasks, both PRs #38 + #39 merged on main @ e758d40 with review corrections delivered), verified (PASS, 0 blockers, 0 critical), and archived. Delta spec synced into the tool-adapter main spec. Docs delivery (archive folder + spec sync PR) is orchestrator-owned and pending. Ready for the next change.
