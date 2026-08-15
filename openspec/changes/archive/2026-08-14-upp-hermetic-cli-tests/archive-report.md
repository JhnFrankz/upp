# Archive Report: upp-hermetic-cli-tests

**Archived**: 2026-08-14
**Archive path**: `openspec/changes/archive/2026-08-14-upp-hermetic-cli-tests/`
**Artifact store mode**: hybrid (filesystem archive move + Engram archive report)
**Status**: SUCCESS — SDD cycle complete

## Final State

- **Tasks**: 29/29 checked, 0 unchecked in the persisted `tasks.md` (Phase 1: 6 — 1.1–1.6; Phase 2: 4 — 2.1–2.4; Phase 3: 14 — 3.1–3.14; Phase 4: 5 — 4.1–4.5). Task Completion Gate passed; no stale checkboxes, no reconciliation needed.
  - ⚠️ Count note (recorded per Final-State Authority): `verify-report.md` and `apply-progress.md` state "25/25" and verify-report's completeness table sums "6+4+14+5 = 25" — that arithmetic is inconsistent with the persisted tasks.md, which contains 29 distinct checkbox lines. The persisted tasks artifact outranks the intermediate snapshots, so the final-state count is **29/29**; the "25/25" figure in the snapshots is an internal counting error and is not echoed as current fact.
- **Verify verdict**: PASS — 0 blockers, 0 critical findings, 0 warnings (per `verify-report.md`, evidence revision main @ fe79f8f). All gates green on fresh execution at close: CLI timing gate `ok 0.039s` test time / **0.458s real** (<2s criterion; baseline 33.371s — **~73× improvement**), full suite `go test ./... -count=1` exit 0 (8 packages with test files), `-race` exit 0, `go vet ./...` clean, `gofmt -l internal/cli/` empty, smoke 23/23 against a freshly built binary.
- **Delivery (complete at close)**: PR #42 (test/hermetic-cli-tests) ALREADY MERGED on main @ fe79f8f (merge commit fe79f8f; change commit 920ce74; issue #41 closed). Single PR per delivery strategy decision; rollback boundary is one `git revert` of the merge (tests + nil-default seams only, no production behavior change).
- **Spec-neutral confirmed**: 0 requirements / 0 scenarios. No `specs/` delta directory was ever created under the change folder; the top-level `spec.md` records only the spec-neutral verification verdict (mirrors the `upp-security-zero-value-hardening` precedent). **No spec sync performed — recorded decision, see Spec Sync below.**
- **Native review**: no native RDD review was started for this candidate — structured status reports `reviewGate` structurally absent (no `state.yaml` in the change folder, no review artifacts). Archive proceeded under ordinary repository policy. Absence of native review is not a defect.
- **Implementation**: `internal/cli/deps.go` (new — package-level `cliDeps` var with race-safety doc), `internal/cli/check.go` (`checkDeps.buildAdapterList` nil-default seam; RunE → `cliDeps.check`), `internal/cli/list.go` (`listDeps` + `runList(gf, deps)` nil-default seam; RunE → `cliDeps.list`), `internal/cli/update.go` + `selfupdate.go` (RunE → `cliDeps.update`/`cliDeps.selfUpdate` only); tests: `update_test.go` (`fakeUpdateAdapter` +command/+privileges, `fakeAdapterList`, `runUpdateWithFlags`), `integration_test.go` (`mockAdapter` retired, `setCLIDeps` helper + `t.Cleanup` restore, 8 cobra-entry conversions), `audit_probe_test.go` (5 probes → fake with `updated`-flag assertions), `probe_test.go` (`probeSetup` retired, `probeHome` kept), `check_hint_test.go` (DefaultOff → `writeCheckConfig(t, "")`). Zero production behavior change; `init.go`, `internal/adapters`, `internal/security` untouched.
- **Follow-up suggestions (recorded in PR #42 body, open, out of scope)**: (1) bare-upp seam gap — `parser.go` constructs commands with no deps and no test executes bare `root.Execute()` through the seam'd RunE bodies; (2) catalog parity — `platform.CatalogFor` vs `official.AdaptersForPlatform` consistency is not asserted; (3) probe wiring coverage — the 5 probes drive `runUpdate` directly with `updateDeps`, not through the cobra RunE → `cliDeps.update` path, so probe+`--ci`/`--quiet` flag interactions are not jointly tested through the root command; (4) nil-default triplication — the nil-default fallback shape is repeated in `checkDeps`/`listDeps`/`updateDeps`; (5) `setCLIDeps` positional noise — positional args ordering of the helper is a readability nit.

## Pending Delivery (orchestrator-owned, NOT part of archive)

The change folder artifacts are **untracked** in git (docs delivery pending). All change code is merged on main @ fe79f8f; the archive folder is the remaining docs. After archive, the orchestrator creates the docs PR containing the archive folder (and any other pending docs). Rollback boundary for the code remains a single `git revert` of the PR #42 merge commit.

## Gates

| Gate | Result |
|------|--------|
| Task Completion Gate | PASSED — 29 `[x]`, 0 `[ ]` in persisted `tasks.md` (see count note in Final State) |
| Native Review Receipt Gate | reviewGate structurally absent → proceed under ordinary repository policy |
| CRITICAL findings gate | No CRITICAL issues in verify-report (0/0); verdict PASS |
| Action Context Guard | repo-local operations only; all archive operations inside the repo root |

## Spec Sync (delta → main)

| Domain | Action | Details |
|--------|--------|---------|
| (none) | **None — spec-neutral confirmed** | The change folder contains NO `specs/` delta directory; the top-level `spec.md` records the spec-neutral verdict (0 requirements / 0 scenarios, verified against all 9 specs under `openspec/specs/`). Per the spec's "Why No Delta Specs" section and the accepted proposal (New/Modified Capabilities: None), the archive phase performs **no spec sync**. Main specs under `openspec/specs/` are untouched. Mirrors the archived `2026-08-14-upp-security-zero-value-hardening` spec-neutral precedent. |

This is the explicit recorded decision: **spec sync intentionally skipped because no delta specs exist** — nothing to merge, nothing to preserve, `rules.archive` destructive-merge warning not applicable.

## Mechanical Copy Evidence

Per the Mechanical Copy Contract: pre-move recursive snapshot → `git mv` attempted (refused: "fatal: source directory is empty, source=openspec/changes/upp-hermetic-cli-tests..." — files untracked, so git refused the move) → `mv` fallback succeeded → source-gone check passed → `diff -r` readback:

```
(diff -r output: empty — no differences)
```

Empty `diff -r` is the only passing evidence; `archive-report.md` is additive-only and excluded from the comparison.

## Archive Contents

- `proposal.md` ✅
- `spec.md` ✅ (spec-neutral verdict — no delta specs)
- `design.md` ✅
- `tasks.md` ✅ (29/29 tasks complete, no unchecked)
- `apply-progress.md` ✅ (all batches + timing evidence)
- `verify-report.md` ✅ (PASS)
- `archive-report.md` ✅ (this file, additive)

Active changes directory (`openspec/changes/`) no longer contains this change (only `archive/` remains).

## Engram Lineage

| Artifact | Location |
|----------|----------|
| proposal / spec / design / tasks / apply-progress / verify-report | Filesystem (hybrid mode; filesystem is the source of truth) — read directly from `openspec/changes/upp-hermetic-cli-tests/` before the move |
| archive-report | saved under topic `sdd/upp-hermetic-cli-tests/archive-report` (Engram, observation ID in phase result) |

## SDD Cycle Complete

The change has been fully planned, implemented (29/29 tasks, PR #42 merged on main @ fe79f8f), verified (PASS, 0 blockers, 0 critical; 33.371s → 0.458s real, ~73×), and archived. Spec-neutral — no spec sync performed. Docs delivery (archive folder PR) is orchestrator-owned and pending. Ready for the next change.
