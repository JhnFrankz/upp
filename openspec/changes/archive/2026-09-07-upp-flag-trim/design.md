# Design: upp-flag-trim — Trim CLI Flag Surface; Delete Dead Group Path

## Technical Approach

Pure deletion of the never-invoked opt-in group path (`runUpdateGroup` — verified zero production callers; its only caller is test helper `runUpdateGroupWith` at update_test.go:1406) and the redundant global `--skip` filter, plus a docs/smoke sweep. The default delegated path (`runUpdateSequential` :137-330, `runUpdateInteractive` :357-475) already implements the target behavior — zero production routing changes. STRICT TDD: flag-rejection RED tests land before flag deletion.

## Architecture Decisions

### D1 — Manager removal chain (`internal/cli/update.go`)
**Choice**: Delete `UpdateFlags.Manager` (parser.go:26-32), both registrations (:39-40), routing branch (:77-98), `runUpdateGroup` (:601-827 incl. doc), `groupTool` (:833-838), `filterGroupSkip` (:850-870) — the sole user of `strings` in update.go, so drop import (:8).
**Alternatives**: keep as deprecated — rejected: two execution paths guarantee divergence (proposal intent); keep `runUpdateGroup` unexported for future reuse — rejected: dead code is the bug.
**Rationale**: mechanical deletion; default path unchanged.
**KEEP (verified callers, not group-exclusive)**: `adapterByName` (called by `resolvingOwner`:918 — default-path critical), `resolvingOwner` (:159 runUpdateSequential, :490 runUpdateInteractive; its runUpdate:91 caller dies with the D1 routing branch), `resolveEffectiveUpdatePolicy` (:217, :541), `ownedPackage` (default-path callers :166, :231, :273, :497, :556), `updateCmdName` (:232, :498). `info.Manager` map fields and `KindManager`/`KindTool` are ownership DATA driving default delegation — explicitly KEEP.

### D2 — `--skip` + `ParseFilter` simplification (`internal/cli/parser.go`)
**Choice**: new signatures `ParseFilter(only string) []string` (thin `parseCommaList` wrapper) and `FilterTools(tools, onlyList []string, stderr io.Writer) []string`; delete `GlobalFlags.Skip` (:20), registration (:61), `filterSkip` (:148-168). Case-insensitivity + unknown-tool warning stay in `filterOnly` (pinned by command-interface spec).
**Alternatives**: drop `ParseFilter`, call `parseCommaList` inline at 3 call sites — rejected: keeps parse/filter pairing testable; single `Filter(only string, tools, stderr)` — rejected: conflates parsing with filtering, breaks existing table tests' shape.
**Verified callers to migrate**: update.go:101-102, list.go:50-53, tests (see T2).

### D3 — Renderer deletions (`internal/output/render.go`)
**Choice**: delete `GroupBulkSummary` type+method (:376-467), `GroupBatchTool` (:471-485), `GroupBatchPreview` type+method (:489-~540).
**Rationale**: verified sole caller of all three is `runUpdateGroup`; default path renders only flat `UpdateSummary` (:322, :406, :467) — untouched.

### D4 — Summary-coverage interpretation (ux-patterns delta)
**Choice**: no production change (proposal: default-path behavior change out of scope). The delta's default-run group scenarios are satisfied at OUTCOME level — flat `UpdateSummary` already reports each owned tool updated/current/skipped/failed. Literal group-header rendering is NOT introduced.
**Rationale**: deleting renderers while adding a group-header renderer would contradict the deletion intent; see Open Question.

## Data Flow

```
runUpdate ─→ ParseFilter(gf.Only) ─→ FilterTools ─→ [TTY&& !ci !quiet !dry-run]
        │                                               ├─→ runUpdateInteractive
        └───────────────────────────────────────────────┴─→ runUpdateSequential
   owned tool → resolvingOwner → manager.CheckPackage / UpdatePackage
   all results → r.UpdateSummary (flat, per-tool outcomes)
   DELETED: runUpdateGroup, GroupBatchPreview, GroupBulkSummary
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/cli/update.go` | Modify | D1 chain (~-270 lines); drop `strings` import |
| `internal/cli/parser.go` | Modify | D2: drop `Skip`+`Manager` fields/flags; new `ParseFilter`/`FilterTools` |
| `internal/cli/list.go` | Modify | :50-53 migrate to new signatures |
| `internal/cli/selfupdate.go` | Modify | :33 comment + :41 Long → "`--only` is ignored: it filters tools for update/check, not releases." |
| `internal/output/render.go` | Modify | D3 (~-165 lines) |
| `internal/cli/parser_test.go` | Modify | T2a adapt + T-RED rejection tests |
| `internal/cli/update_test.go` | Modify | T2b delete/adapt lists |
| `internal/cli/integration_test.go` | Modify | :642-663 skip rows, :875-886 field/flag assertions, :1048 |
| `internal/selfupdate/selfupdate_test.go` | Modify | :400 drop `Skip` (keep `Only`) |
| `internal/output/render_test.go` | Modify | T1 delete 8 tests + section comments |
| `README.md` | Modify | :16 `--only` only; :106 `--only` ignored; :120 drop "takes precedence" phrasing; :121 delete row; add equivalence note (`--manager apt --skip docker` → `--only gh`) |
| `scripts/smoke-test.sh` | Modify | :224 → `update -n --only npm`; :225 → `update -n --only brew`; add `run_test_exit_code` rejection checks (`update --manager apt` ≠0, `update --skip apt` ≠0) |

## Interfaces / Contracts

```go
func ParseFilter(only string) []string
func FilterTools(tools []string, onlyList []string, stderr io.Writer) []string
```

## Testing Strategy (go-test stdlib, table-driven; `go test ./... -count=1`)

**RED-first (T-RED)**, parser_test.go, pattern of `TestUnknownCommand_Check` (:350):
- `TestUpdateCommand_ManagerFlagsRejected`: `Flags().Lookup("manager")` and `("update-group")` both nil; `ParseFlags([]string{"--manager","apt"})` returns error. Pins command-interface + bulk-update rejection scenarios.
- `TestRootCommand_SkipFlagRejected`: same for persistent `--skip`.

**T1 — render_test.go DELETE (8)**: `TestGroupBulkSummary_UpdatedAndSkipped` :959, `_PartialFail` :987, `_DryRun` :1011, `TestGroupBatchPreview_RendersPlannedBatch` :1046, `_CurrentAndUngated` :1077, `_CheckFailed` :1101, `TestGroupBulkSummary_CanonicalOrderNotStatusOrder` :1124, `_ExplicitCountsAndOrdering` :1177, plus comments :946-954/:1033-1041. **SURVIVE**: every `UpdateSummary` test (incl. :1149 verbose diagnostics, :1208 no-misleading-AllClean) — these are the live default-path summary pins. Note: the "group summary coverage" at :949+ cannot literally survive (types deleted); it is re-anchored at CLI level by T3.

**T2 — update_test.go**:
- DELETE: `groupScenario` :1375, `runUpdateGroupWith` :1400, `TestRunUpdate_ManagerFilterRestrictsToGroup` :1473, `TestRunUpdate_GroupCISudoFails` :1797 (duplicate of surviving :1575), `TestRunUpdate_GroupDryRunPlansWithoutExecuting` :1848 (duplicate of surviving :1614).
- ADAPT (drive `runUpdate` via `updateDeps` seam — as :1420 does — same assertions): `TestRunUpdate_GroupGatedBlocksAndRuns` :1696 (gated-block → gh current + no `UpdatePackage`; gated-run → updates; AlwaysUpdate runs regardless), `TestRunUpdate_GroupCheckFailed` :1769 (assert "Failed: gh" in flat summary), `TestRunUpdate_GroupNonSudoProceeds` :1821, and `TestRunUpdate_GroupSkipExcludesOwnedTool` :1661 — RENAME (not delete) to `TestRunUpdate_OnlyNarrowsGroupBatch` (gf.Only="gh", docker untouched; pins bulk-update "Only filter narrows batch"). Handle :1661 exactly once: rename+adapt, never delete.
- parser_test adapt: `TestParseFilter_Skip` :56, `TestParseFilter_OnlyWinsOverSkip` :70, `TestFilterTools_Skip` :134, `TestFilterTools_SkipUnknownWarning` :183, skip lookup :241; manager flag tests :388/:410/:425 replaced by T-RED.

**T3 — NEW (RED-first)**: `TestRunUpdate_DefaultGroupSummarySkipsDeselected` — apt owns gh+docker; gh updates, docker prompt-denied → flat summary reports "Updated: gh" AND "Skipped: docker" (~35 lines). Pins ux-patterns default-path group-outcome scenario.

**Error handling**: no new paths. Unknown flags → cobra default rejection (error + usage, exit ≠ 0). Dead-path errors (manager-not-found :81/:617, CI group failures) vanish with the code; surviving CI aggregation (:325, :470) unchanged.

## Threat Matrix

N/A — no routing, shell, subprocess, VCS/PR automation, executable-file classification, or process-integration boundary changes. Flag-parsing surface only; smoke-test.sh edits are test-data text.

## Migration / Rollout

No migration; config schema untouched. Removed flags fail loudly (cobra rejection); equivalence documented in README. Rollback: `git revert <range>`.

## Changed-Lines Forecast (Review Workload Guard)

Additions ≈ 115 (T-RED ~45, T3 ~35, adapted-test drivers ~10, docs/smoke ~20, plumbing ~5). Deletions ≈ 900 (update.go ~270, render.go ~165, render_test ~230, update_test ~190 + adapted shrink, parser.go ~30, docs/smoke ~15). **Net ≈ -785; total changed lines ≈ 1015.**

```
Decision needed before apply: Yes
Chained PRs recommended: Yes (optional 3-slice: (1) --skip chain, (2) --manager chain, (3) renderers+tests)
400-line budget risk: Medium (total >400 but >85% mechanical deletion; net strongly negative)
```

## Open Questions

- [ ] ux-patterns delta THEN cells say "Group summary lists apt group with gh updated" — confirm D4's outcome-level interpretation (flat summary, no group header) is acceptable, or whether archive-time prose should drop the word "Group" from those scenario names. Non-blocking for implementation.
