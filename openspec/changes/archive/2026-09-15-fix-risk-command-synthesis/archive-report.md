# Archive Report: fix-risk-command-synthesis

**Change**: fix-risk-command-synthesis — RiskCommand From the Real Executed Command
**Archived**: 2026-09-15
**Archived to**: `openspec/changes/archive/2026-09-15-fix-risk-command-synthesis/`
**Artifact store**: openspec (native, authoritative)
**Final HEAD**: `4bebe86` (merge of PR #147) — this phase performed no commit, push, or branch operation.

## Purpose

Close the SDD cycle: merge the two delta specs into the canonical specs (source of truth), move the
change folder into the archive, and record the terminal state of the change. The archived record
describes the state AT CLOSE. `apply-progress.md` (absent for this change) and `verify-report.md` are
intermediate snapshots; the orchestrator's final-state handoff outranks them where they differ.

## Task Completion Gate

The persisted `tasks.md` is the completion-visibility source of truth and the gate passed before any
spec sync or archive move:

- Completed tasks: **14/14** (`[x]`) — Phases 1, 2, and 3.
- Unchecked implementation tasks: **0** (`grep -c '^- \[ \]'` = 0).
- Native status (`gentle-ai sdd-status fix-risk-command-synthesis`): `tasks 14/14 complete`,
  `apply: all_done`, `verify: all_done`, `archive: ready`, `next: archive`.

Task **3.4** ("Carry the archive note below into verify/archive") was the only box still unchecked at
verify time (`verify-report.md`, at verification time, recorded 8/8 in-scope tasks and 3.4 as an
intentional carry-forward). This phase is exactly where that note is consumed, so 3.4 was ticked to
`[x]` here — not an exceptional stale-checkbox reconciliation of code work. No code task was
reconciled; the note it carries is recorded verbatim below.

## Archive Readiness Gate

Archive proceeded only after native SDD status reported `archive: ready`, `next: archive`,
`planning_home.mode: repo-local` at `/home/jhan/projects/upp/openspec`. The workspace root is inside
the allowed edit roots, so no `workspace-planning` guard applied and no linked repo was touched.

## Per-Domain Merge Results

Both deltas were composed into their existing canonical specs with the mandatory native command
`gentle-ai sdd-archive-compose` (never a model-driven Read/Edit merge). Command invocations and
results:

```bash
gentle-ai sdd-archive-compose \
  --canonical "openspec/specs/security-model/spec.md" \
  --delta "openspec/changes/fix-risk-command-synthesis/specs/security-model/spec.md" \
  --output "openspec/specs/security-model/spec.md.compose-tmp"      # exit 0
gentle-ai sdd-archive-compose \
  --canonical "openspec/specs/tool-adapter/spec.md" \
  --delta "openspec/changes/fix-risk-command-synthesis/specs/tool-adapter/spec.md" \
  --output "openspec/specs/tool-adapter/spec.md.compose-tmp"        # exit 0
```

Only a zero exit is composition evidence; both composed readbacks (`diff` canonical vs
`.compose-tmp`) showed exclusively the intended delta changes — every unrelated requirement was
preserved byte-for-byte. The `.compose-tmp` files were atomically moved onto the canonical paths
with `mv`; no `.compose-tmp` file remains.

| Domain | Delta action | Final canonical state |
|--------|--------------|------------------------|
| security-model | 2 MODIFIED (`Confirmation for Destructive Operations`, `Output Transparency`) | **6 requirements** (unchanged count). See exact change table below. All 4 unrelated requirements intact. |
| tool-adapter | 1 MODIFIED (`Adapter Interface`) | **13 requirements** (unchanged count). See exact change table below. All 12 unrelated requirements intact. |

### Exact canonical requirement changes applied

**security-model / `Confirmation for Destructive Operations`** (MODIFIED):

- Body paragraph replaced. Inserted the gate-input binding: the classified command and privileges MUST
  come from the adapter's declared real update command (see tool-adapter) and MUST byte-equal the
  command the update path actually runs; the system MUST NOT classify a synthesized command
  (`<manager> upgrade <pkg>`); `plan.RiskCommand` and plan privileges MUST byte-equal the executed
  command for every managed adapter (pacman included); custom manager-delegated tools MUST be
  classified by the delegated manager's real command and declared privileges. The pre-existing
  owned-group reclassification and `--ci` clauses were retained.
- Added 3 scenario rows: `Gate input is the executed command`, `Pacman package gate sees sudo`,
  `Custom manager-delegated confirm`.
- The requirement's `(Previously: ...)` note was replaced with the delta's gate-input-source note.

**security-model / `Output Transparency`** (MODIFIED):

- Added the binding paragraph: the displayed command and privileges MUST be the adapter-declared
  command that will actually execute — for managed adapters and custom manager-delegated tools alike —
  never a synthesized string.
- Added the `(Previously: the requirement listed what to display but did not bind ...)` note.
- Added 2 scenario rows: `Pacman row transparency`, `Custom delegated transparency`.

**tool-adapter / `Adapter Interface`** (MODIFIED):

- Added paragraph: every manager adapter MUST declare its real self-update and per-package update
  commands (including privilege elevation) and the privileges each requires; pacman MUST declare
  `Privileges: ["sudo"]`; plan metadata (`RiskCommand`, plan privileges) MUST be derived from those
  declarations and MUST byte-equal the executed command; no synthesized templates; custom
  manager-delegated tools MUST inherit the manager's real declared command and privileges.
- Added 4 scenario rows: `pacman declares sudo`, `Declared equals executed`,
  `apt/brew/winget byte-stable`, `Custom delegated inherits declaration`.
- The requirement's `(Previously: ...)` note was replaced with the delta's note (the prior
  `list() → ToolInfo` note is superseded by this MODIFIED block).

**Merge verification**: post-move requirement-heading counts are security-model 6→6 and tool-adapter
13→13; `Output Transparency` and `Adapter Interface` each appear exactly once. No unrelated
requirement was dropped or duplicated. The native composer consumes the blank line that separated the
replaced requirement block from the following heading; that spacing artifact already exists in this
repo's canonical specs from prior native compositions (e.g. `openspec/specs/command-interface/spec.md:126`,
`:159`, `openspec/specs/ux-patterns/spec.md:246`), so it matches established repository state and was
not hand-patched.

## Mechanical Copy Contract Evidence

Archival is a mechanical filesystem operation; no artifact content passed through the model's
Read/Write path.

**Spec composition**: native `gentle-ai sdd-archive-compose`, both exit 0 (above).

**Archive move**: `git mv` was NOT used because this phase is forbidden from any git mutation
(`git add`/`commit`/`stash`/`checkout`/`mv`/...). The guarded plain `mv` ran after a pre-move
recursive snapshot; the destination did not pre-exist. The mandatory readback ran before any report
was written:

```text
# MANDATORY readback: diff -r "$snapshot_root/source" "$destination"
(no output — empty diff, exit 0)
ARCHIVE_MOVE_AND_DIFF_READBACK_EXIT=0
```

The empty `diff -r` output is the only passing evidence and confirms the archived tree is
byte-identical to the pre-move source snapshot. This `archive-report.md` is the only additive file
(it did not exist in the source snapshot). Post-move checks: source directory absent; the active
`openspec/changes/` directory contains only `archive/`; archived `tasks.md` has 14 checked and 0
unchecked tasks.

## Final-State Handoff (postdates the persisted artifacts)

`verify-report.md` describes an uncommitted, pre-landing candidate and is an intermediate snapshot.
The following facts are current at archive time and outrank it:

- **The change is fully LANDED on `main`.** All six PRs merged, in order, as merge commits:
  #142 `a39e9ee` (adapter declarations), #143 `46038c7` (winget read-only fix), #144 `59eaabb` (WU2 S1),
  #145 `ec8ae1c` (WU2 S2), #146 `e78f833` (WU2 S3), #147 `4bebe86` (WU2 S4, the change record).
  `main` tip: `4bebe86`. No open PRs. Issue #141 auto-closed.
- **WU2 was sliced into four commits so that EVERY slice is independently green**: S1 CLI test harness
  (74 lines), S2 engine derivation + synthesis retirement (392), S3 gate tests + comments (177),
  S4 the change record (346).
- **The final committed tree is byte-equal to the reviewed candidate**
  (`3062040592399ed097d83d7778a03e8477c2d681`), so the review receipt covers every committed byte.
- **Native 4-lens review passed with authority burned** (lineage `review-01dec08d10d3e230`):
  18 findings, all non-blocking, 0 implementation-critical.
- **`sdd-verify` returned PASS WITH WARNINGS**: 28 of 29 delta scenarios satisfied,
  0 implementation-critical findings; the single unmet one is the npm scenario below.
- **Post-merge battery on `main` (`4bebe86`)**: `gofmt -s -l .` clean, `go vet ./...` exit 0,
  `golangci-lint` no issues, `go test ./... -count=1 -race` green across 11 packages,
  `bash scripts/smoke-test.sh --skip-build` 35/35.

**Delivery-plan blocker resolution.** `verify-report.md`, at verification time, recorded one BLOCKER
(D1): the originally proposed two-slice split would have landed a red intermediate commit
(`TestRunUpdate_CISudoFailsClosedWithEnforceRisk`, `TestRunUpdate_DefaultGroupSummarySkipsDeselected`).
That finding applied to the split definition, not to the shipped bytes (the full tree was green). It
is resolved by the landed WU2 four-slice stack (S1–S4), each slice independently green as recorded
above. Envelope `critical_findings` remained 0 throughout.

## Pre-existing Unmet Scenario — npm (REQUIRED PROMINENT NOTE)

**Status: PRE-EXISTING, NOT A REGRESSION, AND OUT OF THIS CHANGE'S SCOPE.**

The delta scenario `security-model / Output Transparency / Standard update` requires the display to
show `npm update -g`, while the plan yields `npm update`:

- `internal/adapters/official/npm.go:68` executes `npm update -g`.
- `npm.Info()` (`internal/adapters/official/npm.go:97-106`) declares no `Command`, so
  `internal/engine/plan.go:202-207` (standalone official fallback) yields `npm update`.
- This behavior is identical at base `HEAD`: it was already unmet before this change. `proposal.md`
  and `design.md` (`Open Questions`, residual fiction) explicitly scope standalone official tools
  (go/Linux, npm, pnpm, nvm, …) as outside this change.
- The candidate newly **pins** this divergent behavior with a test at
  `internal/engine/plan_test.go:1086-1095`; that is deliberate documentation of the pre-existing
  state, not a regression.

This is the single unmet scenario in the 28/29 `sdd-verify` result. No corrective action is required
for this change. A future change can fill `ToolInfo.Command` for standalone official adapters; the
mechanism already supports it.

## Task 2.4 Substitution Note (preserved verbatim)

A verifier asked that the task 2.4 substitution note survive into the archive so the substitution
stays auditable. It is preserved verbatim from the archived `tasks.md`:

> **2.4** RED→GREEN `internal/engine/plan_test.go`: `plan.RiskCommand` byte-equals the executed command declaration (`SelfUpdateCommand` / `RenderPackageCommand(PackageUpdateCommand, pkg)`); privileges equal declared. The execution half (declaration == the command Update/UpdatePackage runs, captured through the `runCmdFn` seam) lives in `internal/adapters/official/update_test.go:TestUpdateRunsDeclaredCommand`; the seam is package-private (engine imports official, so official cannot import engine) — the two equalities are composed rather than stubbed cross-package.

The `verify-report.md` direct verdict on this substitution (at verification time, corroborated by the
orchestrator's final-state handoff): the spec requirement is genuinely satisfied by composition; the
structural claim is TRUE (`runCmdFn`/`runCmdArgsFn`/`lookPathFn` are unexported package-level vars in
`package official`; `engine` imports `official` and not the reverse; no exported test seam exists), so
a literal cross-package capture is not required.

## Deferred Follow-ups (recorded, not silently dropped)

These are non-blocking. No action was taken in this phase.

1. **Latent fail-open fallback** at `internal/engine/plan.go:153-156`: when an owner declares no
   per-package template, the plan falls back to the standalone `"<name> update"` string. Unreachable
   with the shipped registry (only apt/brew/winget are reachable as owners, and all three declare the
   template; `scoop` declares none but owns nothing; custom `manager="scoop"` takes the delegated
   branch), but unguarded. Recommended: fail closed (owner `SelfUpdateCommand` or set `ManagerID`)
   and/or an assertion/comment so a future ownership declaration cannot silently reintroduce the
   divergence. (Corresponds to `verify-report.md` W1, at verification time.)
2. **Pre-existing privileged-manager auto-proceed gap** deliberately preserved for byte-stability: a
   manager whose real self command is privileged but whose `Info()` declares no `Privileges` (apt
   today) still auto-proceeds, because `ManagerID` is set from the declared privileges while
   `detectPrivileges` runs later. Byte-stability for apt/brew/winget was a hard constraint of this
   change; closing this gap would alter apt's self-row decision and belongs to a separate change.

Additional lower-priority notes recorded from `verify-report.md` (intermediate snapshot, at
verification time) for a future change, no action taken here:

3. **W3 — execution-half equality strength**: `internal/adapters/official/update_test.go:1446-1456`
   asserts the declaration is present among captured commands, not that the executed set is exactly
   the declaration, and omits `res.Privileges == info.Privileges`. Composition remains sufficient for
   the delta's equality requirement.
4. **W4 — pacman-package CLI coverage gap**: the `Pacman package gate sees sudo` scenario is proven
   compositionally (plan-level synthetic fixture + gate-level fake self-row); the registry has no
   pacman-owned tool, so the production path is unreachable today.
5. **W5 — no `apply-progress.md` artifact** existed for this change; RED→GREEN was instead
   independently re-derived by hunk reversion in `verify-report.md`. Persist the artifact for future
   changes.

## RDD Review Status (separate from SDD)

Review authority is independent of the SDD cycle and produced no blocking outcome. Recorded for the
audit trail only; this phase performed no review lifecycle operation:

- Native review lineage `review-01dec08d10d3e230` completed with authority burned — 4 lenses, 18
  findings, all non-blocking, 0 implementation-critical.
- The final committed tree is byte-equal to the acknowledged candidate
  `3062040592399ed097d83d7778a03e8477c2d681`.

## Intentional Archive Actions

- Task 3.4 was ticked during this phase because this phase is where its carried note is consumed. No
  code task was reconciled and no intentional-partial archive was used.
- No REMOVED or RENAMED requirements, no large-section removals, and no destructive merge occurred.
- `openspec/config.yaml` `rules.archive` ("Warn before merging destructive deltas"): no destructive
  delta was present, so no warning was required.

## Artifacts

- Merged canonical specs:
  - `openspec/specs/security-model/spec.md` (2 MODIFIED: `Confirmation for Destructive Operations`, `Output Transparency`)
  - `openspec/specs/tool-adapter/spec.md` (1 MODIFIED: `Adapter Interface`)
- Archive folder: `openspec/changes/archive/2026-09-15-fix-risk-command-synthesis/`
  (`proposal.md`, `specs/security-model/spec.md`, `specs/tool-adapter/spec.md`, `design.md`,
  `tasks.md`, `verify-report.md`, this `archive-report.md`)
- `openspec/changes/fix-risk-command-synthesis/tasks.md` — task 3.4 ticked, then moved with the folder.

**No source code was modified** in this archive phase: changes are confined to `openspec/`. No commit,
push, or branch was performed; the orchestrator stages and commits.

## SDD Cycle Complete

The change has been fully planned, implemented, verified, and archived. Ready for the next change.
