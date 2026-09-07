# Tasks: upp-flag-trim — Trim Flag Surface; Delete Dead Group Path

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ≈1015 (additions ≈115, deletions ≈900, net ≈−785) |
| 400-line budget risk | Medium |
| Chained PRs recommended | Yes |
| Suggested split | PR 1 manager chain → PR 2 skip chain → PR 3 renderers+verify |
| Delivery strategy | ask-on-risk |
| Chain strategy | stacked-to-main (user-selected 2026-09-06) |

Decision needed before apply: Yes
Chained PRs recommended: Yes
Chain strategy: stacked-to-main (user-selected 2026-09-06)
400-line budget risk: Medium

Slice order (dependency-verified): the manager chain MUST precede the skip chain — `runUpdateGroup` (update.go:625,:637) reads `GlobalFlags.Skip` via `ParseFilter`, so skip-first cannot compile. Manager-first keeps D1 intact and matches D2's migrate list exactly.

Unit 1 ≈500 lines alone (>90% deletion): `runUpdateGroup`, its harness (`groupScenario`, `runUpdateGroupWith`) and 5 tests form one compile unit. Optional further split: surface-rejection PR, then path-deletion PR. User decision required (ask-on-risk).

### Suggested Work Units

| Unit | Goal | PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|----|----------------------|-----------------|-------------------|
| 1 | Reject + delete `--manager`/`--update-group`, dead group path | 1 | `go test ./internal/cli -run 'Manager\|Group' -count=1` | smoke-test.sh `--skip-build` (+`--manager` rejection) | parser Manager block; update.go group path; update_test sweep |
| 2 | Delete global `--skip`; simplify `ParseFilter`/`FilterTools` | 2 | `go test ./internal/cli/... ./internal/selfupdate/... -count=1` | smoke-test.sh (`--only` rows, `--skip` rejection) | parser skip block; call-site migrations; docs |
| 3 | Delete group renderers; T3 pin; final verification | 3 | `go test ./internal/output/... ./internal/cli -count=1` | smoke-test.sh full green | render.go + render_test.go deletions |

## Phase 1 — Manager chain (PR 1)

- [x] 1.1 [RED][S] `internal/cli/parser_test.go`: add `TestUpdateCommand_ManagerFlagsRejected` (pattern: TestUnknownCommand_Check :350) — `Lookup("manager")` and `Lookup("update-group")` nil; `ParseFlags([]string{"--manager","apt"})` errors. Run → RED. Pins command-interface + bulk-update rejection scenarios.
- [x] 1.2 [S] `internal/cli/update.go`: delete manager routing branch (:77-98) [D1]. `resolvingOwner` caller :91 dies; function stays (:159/:490).
- [x] 1.3 [M] `internal/cli/update_test.go` DELETE [T2b]: `groupScenario` :1375; `runUpdateGroupWith` :1400; `TestRunUpdate_ManagerFilterRestrictsToGroup` :1473; `TestRunUpdate_GroupCISudoFails` :1797 (dup of :1575); `TestRunUpdate_GroupDryRunPlansWithoutExecuting` :1848 (dup of :1614). 1.1 → GREEN.
- [x] 1.4 [S] `internal/cli/parser.go`: delete `UpdateFlags.Manager` (:26-32) + both registrations (:39-40) [D1]. (Registrations :39-40 actually live in update.go NewUpdateCommand — deleted there.)
- [x] 1.5 [M] `internal/cli/update.go`: delete `runUpdateGroup` (:601-827), `groupTool` (:833-838), `filterGroupSkip` (:850-870), `strings` import (:8, sole user) [D1]. KEEP: `adapterByName`, `resolvingOwner`, `resolveEffectiveUpdatePolicy`, `ownedPackage`, `updateCmdName`, ownership data (`info.Manager`, `KindManager`/`KindTool`).
- [x] 1.6 [M] `internal/cli/update_test.go` ADAPT via `updateDeps` seam (new `runUpdateDefault` helper; osKey "linux"; manager fakes `noDetect:true`) [T2b]: `TestRunUpdate_GroupGatedBlocksAndRuns` :1696; `TestRunUpdate_GroupCheckFailed` :1769 ("Failed: gh" in flat summary); `TestRunUpdate_GroupNonSudoProceeds` :1821. RENAME (never delete) :1661 `TestRunUpdate_GroupSkipExcludesOwnedTool` → `TestRunUpdate_OnlyNarrowsGroupBatch` (`gf.Only="gh"`, docker untouched).
- [x] 1.7 [S] `scripts/smoke-test.sh`: add `run_test_exit_code` check — `update --manager apt` exits ≠0. Verify: `go test ./... -count=1` + smoke green. (Both `--manager` and `--update-group` rejection rows added; smoke 33/33 green.)

## Phase 2 — Skip chain (PR 2)

- [x] 2.1 [RED][S] `internal/cli/parser_test.go`: add `TestRootCommand_SkipFlagRejected` — root `Lookup("skip")` nil; `ParseFlags([]string{"--skip","apt"})` errors. Run → RED. Pins command-interface "--skip rejected". (Executed RED: both assertions failed pre-deletion.)
- [x] 2.2 [M] `internal/cli/parser.go` [D2]: delete `GlobalFlags.Skip` :20, registration :61, `filterSkip` :148-168; new `ParseFilter(only string) []string` (thin `parseCommaList` wrapper) + `FilterTools(tools, onlyList, stderr io.Writer)`. Case-insensitivity + unknown-tool warning stay in `filterOnly`.
- [x] 2.3 [S] Migrate call sites: `internal/cli/update.go` :101-102; `internal/cli/list.go` :50-53 [D2]. (Post-Phase-1 anchors: update.go :70-71; also fixed list.go display-only comment.)
- [x] 2.4 [M] `internal/cli/parser_test.go` [T2a]: delete `TestParseFilter_Skip` :56, `TestParseFilter_OnlyWinsOverSkip` :70, `TestFilterTools_Skip` :134, `TestFilterTools_SkipUnknownWarning` :183; fix skip lookup :241; adapt call sites :43/:86/:95/:106. 2.1 → GREEN.
- [x] 2.5 [S] `internal/cli/integration_test.go`: adapt :100; drop skip rows :642-663; drop `gf.Skip` assertions :875-886; fix :1048. (Also dropped `TestSkipAllTools` — skip-path only.)
- [x] 2.6 [S] `internal/cli/selfupdate.go` :33 comment + :41 Long → "--only is ignored: it filters tools for update/check, not releases."; `internal/selfupdate/selfupdate_test.go` :400 drop `Skip`, keep `Only`. (Anchor drift: the Skip test actually lives in `internal/cli/selfupdate_test.go` :393 `TestSelfUpdate_OnlySkipIgnored` → renamed `TestSelfUpdate_OnlyIgnored`.)
- [x] 2.7 [S] `README.md`: :16 --only only; :106 --only ignored; :120 drop "takes precedence"; :121 delete row; add equivalence note (`--manager apt --skip docker` → `--only gh`). `scripts/smoke-test.sh` :224/:225 → `update -n --only npm`/`--only brew`; add `update --skip apt` ≠0 check. Verify: full suite + smoke green. (Smoke 33/33; note: `--skip-build` reuses a stale binary — rebuild before smoke or the new rejection row fails against the old flag surface.)

## Phase 3 — Renderers + final verification (PR 3)

- [ ] 3.1 [RED-pin][S] T3: `internal/cli/update_test.go` NEW `TestRunUpdate_DefaultGroupSummarySkipsDeselected` (~35 lines): apt owns gh+docker; gh updates, docker prompt-denied → flat summary "Updated: gh" AND "Skipped: docker". Expected GREEN on HEAD (D4: no production change); RED ⇒ STOP and escalate (out-of-scope default-path bug). Pins ux-patterns "Default group bulk summary".
- [ ] 3.2 [M] `internal/output/render_test.go` [T1] DELETE 8 tests + section comments (:946-954, :1033-1041): `TestGroupBulkSummary_UpdatedAndSkipped` :959; `_PartialFail` :987; `_DryRun` :1011; `TestGroupBatchPreview_RendersPlannedBatch` :1046; `_CurrentAndUngated` :1077; `_CheckFailed` :1101; `TestGroupBulkSummary_CanonicalOrderNotStatusOrder` :1124; `_ExplicitCountsAndOrdering` :1177. SURVIVE: all `UpdateSummary` tests (incl. :1149, :1208).
- [ ] 3.3 [M] `internal/output/render.go` [D3]: delete `GroupBulkSummary` type+method (:376-467), `GroupBatchTool` (:471-485), `GroupBatchPreview` type+method (:489-~540). Flat `UpdateSummary` (:322/:406/:467) untouched.
- [ ] 3.4 [S] FINAL VERIFICATION: `go test ./... -count=1`; `go vet ./...`; `gofmt -s -l` empty; `bash scripts/smoke-test.sh --skip-build` green; flag surface exactly 5 (`--quiet/-q`, `--verbose/-v`, `--ci`, `--only` global; `--dry-run/-n` update+uninstall); `--manager`/`--update-group`/`--skip` all rejected non-zero.
