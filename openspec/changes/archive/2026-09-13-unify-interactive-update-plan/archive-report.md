# Archive Report: unify-interactive-update-plan

**Change**: unify-interactive-update-plan — Unify interactive `upp update` with `engine.Plan`
**Archived**: 2026-09-13
**Archived to**: `openspec/changes/archive/2026-09-13-unify-interactive-update-plan/`
**Artifact store**: hybrid (repo-local `openspec/` files + Engram mirror)
**Final HEAD**: `f31876c` — nothing committed or pushed by this phase.

## Purpose

Close the SDD cycle: merge the two delta specs into the canonical specs (source of truth), move the
change folder into the archive, and record the terminal state of the change.

## Task Completion Gate

The persisted `tasks.md` is the completion-visibility source of truth and the gate passed before any
spec sync or archive move:

- Completed tasks: **19/19** (`[x]`) — Phases 1, 2, 3, and the Phase 4 verification checklist.
- Unchecked implementation tasks: **0** (`grep -c '^- \[ \]'` = 0).
- Native status (`gentle-ai sdd-status`): `tasks 19/19 complete`, `apply: all_done`, `verify: all_done`,
  `archive: ready`, `nextRecommended: archive`.

No exceptional stale-checkbox reconciliation was needed or performed.

## Archive Readiness Gate

Archive proceeded only after native SDD status (`gentle-ai sdd-status unify-interactive-update-plan`)
reported `dependencies.archive: ready` and `nextRecommended: archive`. `actionContext.mode` was
`repo-local` and the workspace root is inside `allowedEditRoots`, so no workspace-planning guard applied.

## Per-Domain Merge Results

Both deltas were composed into their existing canonical specs with the mandatory native command
`gentle-ai sdd-archive-compose` (never a model-driven Read/Edit merge). Command invocations and results:

```bash
gentle-ai sdd-archive-compose \
  --canonical "openspec/specs/command-interface/spec.md" \
  --delta "openspec/changes/unify-interactive-update-plan/specs/command-interface/spec.md" \
  --output "openspec/specs/command-interface/spec.md.compose-tmp"   # exit 0
gentle-ai sdd-archive-compose \
  --canonical "openspec/specs/ux-patterns/spec.md" \
  --delta "openspec/changes/unify-interactive-update-plan/specs/ux-patterns/spec.md" \
  --output "openspec/specs/ux-patterns/spec.md.compose-tmp"          # exit 0
```

Only a zero exit is composition evidence; both composed readbacks (`diff` canonical vs `.compose-tmp`)
showed exclusively the intended delta changes — every unrelated requirement was preserved
byte-for-byte. The `.compose-tmp` files were atomically moved onto the canonical paths with `mv`.

| Domain | Delta action | Final canonical state |
|--------|--------------|------------------------|
| command-interface | 2 ADDED (Canonical Tool Selection Identity; Deselected Pending Tools Reporting) + 1 MODIFIED (`upp update`) | **9 requirements** (7 pre-existing + 2 new). The `upp update` MODIFIED paragraph was replaced with the Plan-derived-pending contract, its scenario table gained `Plan-derived pending set` and `Canonical identity selection` and updated `Granular selection in manager group`, and its `(Previously: ...)` note was replaced. The 2 ADDED requirements were appended after `Help Output Grouping`. All 7 pre-existing requirements intact. |
| ux-patterns | 1 MODIFIED (Manager Self-Update Row Rendering) | **16 requirements** (unchanged count). The requirement body was rewritten to the `engine.Plan`-derived pending-selector contract; the `brew never pending` scenario was replaced by `brew pending in selector` and `bun pending in selector`; the delta's `(Previously: ...)` and `(Migration: ...)` notes were carried through. All 16 requirement headings still present; the other 15 requirements unchanged. |

**Merge verification**: post-move requirement-heading counts are command-interface 7→9 and ux-patterns
16→16. The delta-addressed requirements (`Canonical Tool Selection Identity`, `Deselected Pending Tools
Reporting`, `Manager Self-Update Row Rendering`) appear exactly once each. No unrelated requirement was
dropped or duplicated.

## Mechanical Copy Contract Evidence

Archival is a mechanical filesystem operation; no artifact content passed through the model's
Read/Write path.

**Spec composition**: native `gentle-ai sdd-archive-compose`, both exit 0 (above).

**Archive move**: the source change folder was untracked, so `git mv` refused with
`fatal: source directory is empty`; the guarded fallback verified the source was unchanged against a
pre-move recursive snapshot and then performed a plain `mv`. The mandatory readback ran before any
report was written:

```text
# git mv (guarded fallback path):
fatal: source directory is empty, source=openspec/changes/unify-interactive-update-plan, destination=openspec/changes/archive/2026-09-13-unify-interactive-update-plan

# MANDATORY readback: diff -r "$snapshot_root/source" "$destination"
(no output — empty diff, exit 0)
ARCHIVE_MOVE_AND_DIFF_READBACK_EXIT=0
```

The empty `diff -r` output is the only passing evidence and confirms the archived tree is byte-identical
to the pre-move source snapshot. This `archive-report.md` is the only additive file (it did not exist in
the source snapshot). Post-move checks: source directory absent; archived `tasks.md` has 19 checked and
0 unchecked tasks; the active `openspec/changes/` directory no longer contains this change.

## Verification Summary (Final State)

The archive report describes the state at close. `apply-progress.md` and `verify-report.md` are
intermediate snapshots; the launch prompt's final-state facts outrank them where they differ, and the
orchestrator confirmed **no source code changed after `verify-report.md` was written**.

- **Verdict**: `pass_with_warnings` — 0 blockers, 0 critical findings (per `verify-report.md`, at
  verification time; corroborated as still current by the orchestrator).
- **Requirements**: 4/4; scenarios 27/29 fully COMPLIANT, 2 PARTIAL, 0 UNTESTED.
- **Runtime (current at archive time, per orchestrator final-state facts)**:
  - `go test ./... -count=1 -race` → exit 0 (10 packages).
  - `go build ./...` → exit 0.
  - `go vet ./...` → clean; `gofmt -l .` → clean.
  - Smoke `scripts/smoke-test.sh --skip-build` → 35/35 passed.
  - Coverage: cli 89.4%, output 88.9%, combined 89.2%.
- **D7 reversal**: intentional, PC1-confirmed behavior change, not a regression — the old
  `Manager Self-Update Row Rendering` "brew never pending" rule is superseded by this change; the
  D7-pinned tests were flipped (not deleted) and are green.

## RDD Review Status (separate from SDD — does not block archive)

Review authority is independent of the SDD cycle and produced no approval receipt. Recorded for the
audit trail only; this phase performed no review lifecycle operation:

- Review transaction `review-2b2d359f29cbcbbc` is terminally **stopped** (horizon terminal).
- 3/4 lenses admitted — `review-risk`, `review-resilience`, `review-readability` — with zero
  candidate-causal blocking findings.
- `review-reliability` was declared **unachievable** due to a provider output cap (~32k tokens).
- There is **no RDD approval receipt**.

## Delivery Status (planned, not executed)

Delivery is forecast high (>400 changed lines), so under the `ask-on-risk` strategy the coordinator will
ask whether to split into chained PRs (planned split: PR1 = canonical tool identity + shared Plan
executor; PR2 = plan-derived pending set + `StatusDeselected`) or proceed with a `size:exception`.
Working tree at archive time: 5 modified Go files (`internal/cli/checkrun.go`, `internal/cli/update.go`,
`internal/cli/update_test.go`, `internal/output/render.go`, `internal/output/render_test.go`) plus the
archived change folder. HEAD `f31876c`; nothing committed or pushed.

## Residual Notes / Follow-ups (recorded, no action taken)

From `verify-report.md` (intermediate snapshot, at verification time); no source code was modified by
this archive phase:

1. **W2 — Selector over filtered set** (pre-existing, PARTIAL): no interactive selector test with
   `--only`; only non-interactive narrowing is covered.
2. **W3 — brew list version** (pre-existing, PARTIAL): no `upp list` test wires brew's version end to
   end; evidence is composed across three layers.
3. **W4 — `outcomeSelectionID` fallback** (`update.go:526`) remains uncovered at 66.7%; it is a
   defensive fallback (the engine already carries the canonical ID), not a spec scenario branch.
4. **S1/S2/S3 suggestions** (informational): apt/winget dry-run renderer version annotation; residual
   `GroupByOwner` identity inconsistency not on this change's paths; `golangci-lint` not installed
   locally (CI remains the lint authority).

No CRITICAL or WARNING class issue from `verify-report.md` remains open as a candidate-caused defect;
the two WARNINGs are pre-existing PARTIAL scenarios unrelated to this change's remediation.

## Intentional Archive Actions

The `ux-patterns` delta MODIFIES the existing `Manager Self-Update Row Rendering` requirement, replacing
the `brew never pending` scenario and reversing the "brew MUST NOT appear in the pending selector" rule.
This is an **intentional PC1-confirmed reversal** (D7), explicitly authorized, and not a destructive
merge of unrelated content — all other requirements and scenarios were preserved. No REMOVED/RENAMED
requirements and no large-section removals occurred. No intentional-partial archive or stale-checkbox
reconciliation override was used.

## Artifacts

- Merged canonical specs:
  - `openspec/specs/command-interface/spec.md` (2 ADDED + 1 MODIFIED)
  - `openspec/specs/ux-patterns/spec.md` (1 MODIFIED)
- Archive folder: `openspec/changes/archive/2026-09-13-unify-interactive-update-plan/`
  (`proposal.md`, `specs/`, `design.md`, `tasks.md`, `verify-report.md`, `apply-progress.md`,
  `exploration.md`, `research.md`, `preproposal.yaml`, this `archive-report.md`)
- Engram mirror observations read/recorded for traceability:
  - `sdd/unify-interactive-update-plan/proposal` (#238)
  - `sdd/unify-interactive-update-plan/spec` (#240)
  - `sdd/unify-interactive-update-plan/design` (#242)
  - `sdd/unify-interactive-update-plan/tasks` (#243)
  - `sdd/unify-interactive-update-plan/apply-progress` (#244)
  - `sdd/unify-interactive-update-plan/verify-report` (#247)
  - supporting: explore (#235), research (#236), preproposal (#237)

**No source code was modified** in this archive phase. No commit, push, or branch was performed; git
state was left untouched.

## SDD Cycle Complete

The change has been fully planned, implemented, verified, and archived. Ready for the next change.
