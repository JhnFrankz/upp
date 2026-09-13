```yaml
schema: gentle-ai.verify-result/v1
evidence_revision: sha256:aefa38e5904585886665f982ce12e257ca25ed95aab3138f07d828e7f2cf03ee
verdict: pass_with_warnings
blockers: 0
critical_findings: 0
requirements: 4/4
scenarios: 0/0
test_command: go test ./... -count=1 -race
test_exit_code: 0
test_output_hash: sha256:7693e359096fa0f5090774fcc81a7c3855bfba9547909b291b884c42faa7af0c
build_command: go build ./...
build_exit_code: 0
build_output_hash: sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
```

## Verification Report

**Change**: `unify-interactive-update-plan`
**Date**: 2026-09-13
**Mode**: Strict TDD (`openspec/config.yaml` `testing.strict_tdd: true`, runner `go test`)
**Artifact store**: openspec (native), hybrid with Engram
**Evidence Revision**: `sha256:aefa38e5904585886665f982ce12e257ca25ed95aab3138f07d828e7f2cf03ee`
**Verification scope**: proposal, delta specs (`command-interface`, `ux-patterns`), design, tasks, apply-progress (PR1 + PR2 + remediation), re-run on the current working tree.

> Evidence-revision definition (unchanged from the prior report, formula reproduced byte-exactly): SHA-256 of the concatenation, in fixed order, of the raw hex digests of test, build, smoke, and vet outputs:
> test `7693e359…` + build `e3b0c442…` (empty) + smoke `497b2d80…` + vet `e3b0c442…` (empty).

> **Re-verification delta**: the prior verify returned `fail` solely for two required ADDED scenarios with no covering test — C1 `Empty ID fallback` and C2 `Unresolvable selection`. A remediation apply added `TestRunUpdate_EmptyIDFallbackSelection` and `TestRunUpdate_UnresolvableSelection` (plus the `emptyIDAdapter` wrapper) in `internal/cli/update_test.go`, marked Phase 4 tasks 4.1–4.3 `[x]`, and merged the remediation batch into `apply-progress.md`. This report re-runs every gate afresh; the prior verdict was not trusted. Both C1 and C2 are now independently confirmed covered and green.

### Completeness

| Metric | Value |
|--------|-------|
| Tasks total | 19 |
| Tasks complete (checked `[x]`) | 19 (Phases 1–4) |
| Tasks incomplete (unchecked `[ ]`) | 0 |
| Native status (`gentle-ai sdd-status`) | `taskProgress` 19/19, `verify: ready`, `nextRecommended: verify` |

Phase 4 (4.1–4.3) is the verification checklist; all three are now checked and independently re-executed green in this run (4.1 full `-race` suite exit 0; 4.2 smoke 35/35; 4.3 CI exit stays `hasFailure`-only at `update.go:423-425`, interactive gated `!gf.CI` at `update.go:137`, `--dry-run` renders no selector per `TestRunUpdate_SelectorGateMatrix/--dry-run_skips_selector`).

### Build & Tests Execution

**Build**: ✅ Passed (exit 0, empty output → `sha256:e3b0c442…`)
```text
$ go build ./...
(no output)
```

**Tests**: ✅ Passed (exit 0; `go test ./... -count=1 -race`, 10 packages)
```text
?   github.com/JhnFrankz/upp/cmd/upp	[no test files]
ok  github.com/JhnFrankz/upp/internal/adapters          1.377s
ok  github.com/JhnFrankz/upp/internal/adapters/official 1.588s
ok  github.com/JhnFrankz/upp/internal/cli               1.150s
ok  github.com/JhnFrankz/upp/internal/config            1.025s
ok  github.com/JhnFrankz/upp/internal/engine            1.408s
ok  github.com/JhnFrankz/upp/internal/output            8.078s
ok  github.com/JhnFrankz/upp/internal/platform          1.017s
ok  github.com/JhnFrankz/upp/internal/security          1.029s
ok  github.com/JhnFrankz/upp/internal/selfupdate        1.448s
ok  github.com/JhnFrankz/upp/internal/uninstall         1.016s
```
Output hash: `sha256:7693e359096fa0f5090774fcc81a7c3855bfba9547909b291b884c42faa7af0c`

**Smoke**: ✅ Passed (exit 0) — fresh `./upp` built first, then `bash scripts/smoke-test.sh --skip-build` → `Results: 35 passed, 0 failed, 35 total`
```text
=== Results: 35 passed, 0 failed, 35 total — All tests passed!
```
Output hash: `sha256:497b2d8065dfc3eabd2a369ca3f1f6ecb003d2129cfaab04eeee5e13a47bbdae`

**Vet**: ✅ Clean (`go vet ./...` exit 0, empty output → `sha256:e3b0c442…`).
**gofmt**: ✅ Clean (`gofmt -l .` empty; `gofmt -s -l .` also empty).

**Coverage** (`go test ./internal/cli/ ./internal/output/ -count=1 -coverprofile`): cli 89.4%, output 88.9%, combined total 89.2%. Coverage threshold: 0 → ✅ Above.
The two previously-uncovered branches are now covered:
- `toolSelectionID` (`checkrun.go:127`) **66.7% → 100.0%** — the `return a.Name()` fallback branch is now executed by `TestRunUpdate_EmptyIDFallbackSelection`.
- `runUpdateInteractive` (`update.go:274`) **79.1% → 81.3%** — the `update.go:409-418` "selected ID absent from plan" block is now covered by `TestRunUpdate_UnresolvableSelection` (absent from the `count=0` profile blocks).
Other changed functions: `executePlannedUpdate` 90.6%, `UpdateSummary` 100.0%, `statusIcon` 90.0%, `statusIconPlain` 87.5%, `outcomeSelectionID` 66.7% (its `return oc.ToolName` fallback is defensive and not a spec branch). **golangci-lint: ➖ not installed locally** (only `go vet` ran).

### Spec Compliance Matrix

Delta accounts for **4 requirements** and **29 scenario rows** (requirement blocks: command-interface 3, ux-patterns 1; the YAML envelope uses native `#### Scenario:` heading counts, which are 0, so `scenarios: 0/0`).

#### 1. `command-interface` — ADDED: Canonical Tool Selection Identity (4)

| Scenario | Test / Evidence | Result |
|----------|-----------------|--------|
| Distinct ID and label | `TestRunUpdate_CanonicalIdentitySelection` (`update_test.go:1468`) asserts option `{ID:"gh", Label:"GitHub CLI"}` | ✅ COMPLIANT |
| Selection maps to adapter | `TestRunUpdate_CanonicalIdentitySelection` asserts `gh.updated` | ✅ COMPLIANT |
| Empty ID fallback | `TestRunUpdate_EmptyIDFallbackSelection` (`update_test.go:1536`) via `emptyIDAdapter` (`:145`): `Info().ID=""`, `Name()=="npm"` → option `{ID:"npm", Label:"npm", Version:"10.0.0 → 10.1.0"}`, `fake.updated==true`, summary `Updated: npm`. Coverage `toolSelectionID` 100%. **Non-vacuous**: without the fallback, `adapterByID` keys `""` so the literal selection `"npm"` misses the adapter map and the tool fails/drops, breaking the `fake.updated` + `Updated: npm` assertions. | ✅ COMPLIANT |
| Unresolvable selection | `TestRunUpdate_UnresolvableSelection` (`update_test.go:1594`): selects `"ghost-tool"` (absent from the plan-derived pending set) → asserts `Failed: ghost-tool` and `1 failed`, pending `npm` reported `Deselected: npm`, `npm.updated==false`. Covers `update.go:409-418`. **Non-vacuous**: the explicit-failure output only exists from that block. | ✅ COMPLIANT |

#### 2. `command-interface` — ADDED: Deselected Pending Tools Reporting (4)

| Scenario | Test / Evidence | Result |
|----------|-----------------|--------|
| Deselected tool reported | `TestRunUpdate_InteractiveSelection` (`update_test.go:748`) asserts `2 deselected`; `TestUpdateSummary_Deselected` (`render_test.go:337`) | ✅ COMPLIANT |
| Distinct from skipped | `TestUpdateSummary_DeselectedDistinctFromSkipped` (`render_test.go:391`) asserts `1 skipped, 1 deselected` | ✅ COMPLIANT |
| Not silently dropped | `TestRunUpdate_InteractiveSelection` asserts `Deselected: deselected-tool, current-tool`; `TestRunUpdate_InteractiveSelection_OwnedToolDelegation` (`update_test.go:918`) asserts `Deselected: docker` | ✅ COMPLIANT |
| All pending deselected | `TestUpdateSummary_OnlyDeselected` (`render_test.go:367`) asserts `2 deselected` and no "All tools not installed" | ✅ COMPLIANT |

**PC2 confirmation**: deselected pending tools are emitted as `output.StatusDeselected` (`update.go:380-383`), counted via `countByStatusType` (`render.go:292`), and rendered as a distinct summary part + `Deselected:` detail line (`render.go:316-317,381-384`). Never counted as skipped/updated and never dropped. ✅

#### 3. `command-interface` — MODIFIED: `upp update` (14)

| Scenario | Test / Evidence | Result |
|----------|-----------------|--------|
| Normal default update | `TestRunUpdate_DefaultBulkGroupExecution` (`update_test.go:1568`) | ✅ COMPLIANT |
| Per-tool isolated failure | `TestRunUpdate_PerToolErrorIsolation` (`update_test.go:1622`) | ✅ COMPLIANT |
| `--ci` failure exit | `TestVerifyPins_StrictTTDScenarios` subtest "update --ci non-zero on failure" (`update_test.go:2083`) | ✅ COMPLIANT |
| `--ci` elevated risk fail-closed | `TestRunUpdate_CISudoFailsClosedWithEnforceRisk` (`update_test.go:1673`); `TestExecutePlannedUpdate_Coverage` privilege-parity case (`:1357`) | ✅ COMPLIANT |
| Dry run full flag | `TestRunUpdate_DryRunPlannedFlags` (`update_test.go:1833`) | ✅ COMPLIANT |
| Dry run short flag | `TestUpdateCommand_DryRunShorthand` (`update_test.go:542`) | ✅ COMPLIANT |
| Selector over filtered set | Only non-interactive narrowing is tested: `TestRunUpdate_OnlyNarrowsGroupBatch` (`update_test.go:1881`, `runUpdateDefault`, stdin not TTY). No interactive selector test runs with `--only` | ⚠️ PARTIAL |
| Plan-derived pending set | `TestRunUpdate_ManagerSelfUpdateBrewInSelector` (`update_test.go:1245`); `TestRunUpdate_AlwaysUpdateCurrentSelectedExecutes` (`:1643`) | ✅ COMPLIANT |
| Canonical identity selection | `TestRunUpdate_CanonicalIdentitySelection` (`update_test.go:1468`) | ✅ COMPLIANT |
| Granular selection in manager group | `TestRunUpdate_InteractiveSelection_OwnedToolDelegation` (`update_test.go:918`) | ✅ COMPLIANT |
| Dry-run non-interactive | `TestRunUpdate_SelectorGateMatrix/--dry-run_skips_selector` (`update_test.go:664`) | ✅ COMPLIANT |
| `--manager` rejected | `TestUpdateCommand_ManagerFlagsRejected` (`parser_test.go:316`); smoke "upp update --manager apt (removed, exit 1)" | ✅ COMPLIANT |
| `--update-group` rejected | Same test + smoke "upp update --update-group brew (removed, exit 1)" | ✅ COMPLIANT |
| Engine delegation seam | `update.go:290` (`eng.Check`), `update.go:301` (`eng.Plan`), `list.go:53` (`eng.Resolve`) — exercised by all update tests | ✅ COMPLIANT |

#### 4. `ux-patterns` — MODIFIED: Manager Self-Update Row Rendering (7)

| Scenario | Test / Evidence | Result |
|----------|-----------------|--------|
| brew current on board | `TestRunUpdate_ManagerSelfUpdateBrewInSelector` asserts `brew up-to-date` | ✅ COMPLIANT |
| brew pending in selector | Same test asserts brew in `wantOpts` (`update_test.go:1291-1295`) and `Deselected: brew` | ✅ COMPLIANT |
| bun pending in selector | `TestRunUpdate_AlwaysUpdateCurrentSelectedExecutes` asserts bun in the selector and, when selected, executed | ✅ COMPLIANT |
| brew dry-run current | `TestRunUpdate_ManagerSelfUpdateDryRun` (`update_test.go:1158`) | ✅ COMPLIANT |
| apt dry-run planned | Same test asserts `2 would update` for apt+winget | ✅ COMPLIANT |
| winget dry-run planned | Same test asserts `2 would update` | ✅ COMPLIANT |
| brew list version | Composed evidence only: brew `Check` parses `Homebrew 4.1.0` (`official/check_test.go:243-256`), `listEntryFor` uses `CurrentVersion` (`group.go:165-180`), `TestListTools` renders Version (`render_test.go:412`). No `upp list` test wires the brew adapter end to end | ⚠️ PARTIAL |

**Compliance summary**: **27/29** scenarios fully COMPLIANT, 2 PARTIAL, **0 UNTESTED** (27/29 have passing runtime coverage; previously 25/29 with 2 UNTESTED).

### D7 Reversal — Intentional Behavior Change (NOT a defect) ✅

PC1 supersedes the existing main-spec rule at `openspec/specs/ux-patterns/spec.md:227-236` (`Manager Self-Update Row Rendering`, including the `brew never pending` scenario). The delta (`specs/ux-patterns/spec.md:5-23`) rewrites the full requirement block, replacing `brew never pending` with `brew pending in selector` and adding `bun pending in selector`. Implementation derives the pending set from `engine.Plan` (`update.go:301,311`), and `plan.go:92-97` plans `PolicyAlwaysUpdate` tools unconditionally.

The old D7-pinned tests were flipped, not deleted; all are green in this run:
- `TestRunUpdate_ManagerSelfUpdateBrewInSelector` (`update_test.go:1245`) asserts brew in `wantOpts` and `Deselected: brew`.
- `TestRunUpdate_InteractiveSelection` (`update_test.go:748`) asserts `2 deselected` + `Deselected: deselected-tool, current-tool`.
- `TestRunUpdate_SelectorCancel` (`update_test.go:988`) proves brew is presented.
- `TestRunUpdate_AlwaysUpdateCurrentSelectedExecutes` (`update_test.go:1643`) proves a selected always-update current tool executes end to end (`Name()=="bun"` ≠ `Info().Name=="Bun"`).

This is an intended behavior change consistent with the MODIFIED delta spec, explicitly **not** a regression.

### Correctness (Static Evidence)

| Requirement | Status | Notes |
|-------------|--------|-------|
| Canonical Tool Selection Identity | ✅ Implemented | `toolSelectionID` + `adapterByID` (`checkrun.go:127-143`); cursor IDs = `PlanUpdate.ToolID`; both previously-untested branches now have passing runtime tests |
| Deselected Pending Tools Reporting | ✅ Implemented | `StatusDeselected` const/icon/label/line, summary count/part/guard/detail (`render.go:25-29,122,141,160,229-230,292,316-317,327,338,348-351,366,381-384`) |
| `upp update` (Plan-driven) | ✅ Implemented | Pending = `plan.Updates` (`update.go:301-327`); selected executes `planByID[oc.ToolID]` (`update.go:363-399`); unresolvable selection → `StatusFailed` (`:386-397`, `:409-418`); `processSelectedOutcome` deleted (grep: ABSENT) |
| Manager Self-Update Row Rendering | ✅ Implemented | brew/bun planned unconditionally via `PolicyAlwaysUpdate`; dry-run preserved |

### Coherence (Design)

| Decision | Followed? | Notes |
|----------|-----------|-------|
| D1 Canonical identity `Info().ID` fallback `Name()` | ✅ Yes | `checkrun.go:127-132`; option `ID=ToolID`, `Label=ToolName` (`update.go:321-326`); fallback covered by `TestRunUpdate_EmptyIDFallbackSelection` |
| D2 Single shared executor | ✅ Yes | `executePlannedUpdate` (`update.go:437`), called by both paths (`:399` and sequential); `processSelectedOutcome` deleted |
| D3 Risk/privilege from `PlannedUpdate` | ✅ Yes | `p.RiskCommand/Privileges/Trust`, `EnforceRisk: p.ManagerID != ""` (`update.go:444-453`); parity case tested |
| D4 `StatusDeselected` after `StatusCurrent` | ✅ Yes | const at `render.go:29`; `countByStatus` 3-value signature kept (`render.go:599`) |
| D5 Always-update current included | ✅ Yes | `plan.Updates` unconditional (`plan.go:92-97`); pending = `plan.Updates`; end-to-end test green |

Deviations are documented in `apply-progress.md:136-140` and are non-breaking (variadic `allAdapters`, no policy gate in the executor by design).

### TDD Compliance (Strict)

| Check | Result | Details |
|-------|--------|---------|
| TDD Evidence reported | ✅ | `apply-progress.md:59-72` (TDD Cycle Evidence table, incl. C1/C2 rows) |
| All tasks have tests | ✅ | Phases 1–3 (16 tasks) all have covering test rows; Phase 4 (4.1–4.3) is an execution-only checklist, not code |
| RED confirmed (tests exist) | ✅ | RED recorded before implementation (1.6); C1/C2 RED proven by temporary production mutations (recorded in apply-progress, reverted byte-for-byte) |
| GREEN confirmed (tests pass) | ✅ | Full `-race` suite green at runtime in this run; the two new tests pass under `-run` verbosely |
| Triangulation adequate | ✅ | Every required scenario now has an executing test; the two prior UNTESTED scenarios added dedicated tests; C2 also covers the unresolvable branch |
| Safety Net for modified files | ✅ | cli/output packages green pre-change |

**TDD Compliance**: 6/6 checks passed.

### Test Layer Distribution

| Layer | Tests | Files | Tools |
|-------|-------|-------|-------|
| Unit | 3 | `internal/output/render_test.go` (deselected summary/guard/icon) | stdlib `testing` |
| Integration | 9+ | `internal/cli/update_test.go` (interactive selector, plan-derived pending, identity, fallback, unresolvable, D7 flip, coverage migration) | stdlib `testing` |
| E2E | 35 smoke assertions | `scripts/smoke-test.sh` via built `./upp` | bash smoke harness |
| **Total** | 12 Go tests touched + smoke | | |

### Changed File Coverage

| File | Line % | Uncovered Lines | Rating |
|------|--------|-----------------|--------|
| `internal/cli/checkrun.go` | package 89.4% | unrelated checkrun fallbacks | ✅ Excellent (`toolSelectionID` 100%) |
| `internal/cli/update.go` | package 89.4% | defensive pending-with-no-adapter `386-397`, plan-error `303-304`, `eng==nil` `276-280`, CI `424-425` | ✅ Acceptable (`runUpdateInteractive` 81.3%, executor 90.6%) |
| `internal/output/render.go` | package 88.9% | — (`UpdateSummary` 100%) | ✅ Excellent |

**Average changed file coverage**: 89.2% (combined cli+output profile).

### Assertion Quality

**Assertion quality**: ✅ All assertions verify real behavior. The new tests assert concrete selector option values (`ID/Label/Version`), execution flags (`fake.updated`), and decoded summary output (`Failed: ghost-tool`, `1 failed`, `Updated: npm`, `Deselected: npm`). No tautologies, no ghost loops, no smoke-only assertions; the `emptyIDAdapter` wrapper is a production-code invocation, not a mock-count proxy.

### Quality Metrics
**Linter**: ➖ Not available (`golangci-lint` not installed locally; `go vet` clean).
**Type Checker**: ✅ No errors (`go vet ./...` exit 0).

### Issues Found

**CRITICAL**
- None. C1 (`Empty ID fallback`) and C2 (`Unresolvable selection`) from the prior report are resolved with green, non-vacuous covering tests.

**WARNING**
- W2 — "Selector over filtered set": no interactive selector test with `--only`; only non-interactive narrowing is covered (`internal/cli/update_test.go:1881`). Pre-existing, unrelated to the remediation.
- W3 — "brew list version": no `upp list` test wires brew's version end to end; evidence is composed across three layers (`official/check_test.go`, `group.go`, `render_test.go`). Pre-existing.
- W4 — Strict TDD: `outcomeSelectionID` fallback (`update.go:526`) remains uncovered at 66.7%; it is a defensive fallback (the engine already carries the canonical ID) and not a spec scenario branch.

**SUGGESTION**
- S1 — The `apt`/`winget` dry-run scenario THEN names the literal command (`sudo apt install --only-upgrade apt`); the renderer emits the planned version annotation instead. Pre-existing, not introduced by this change.
- S2 — Residual identity inconsistency adjacent to D1: `GroupByOwner` keys `entryByID` by `a.Name()` (`internal/output/group.go:35`) while `ListEntry.ID` uses `info.ID` (`group.go:176`). Not on this change's selector/executor paths.
- S3 — `golangci-lint` is not installed in this environment; only `go vet` ran. CI remains the authority for lint.

### Verdict

**PASS WITH WARNINGS**

Implementation, build, race tests, smoke, vet, gofmt, the D7 reversal, PC1 (always-update current tools presented and executable), and PC2 (distinct deselected reporting) are all correct and green. Both previously-UNTESTED required ADDED scenarios now have independently confirmed, non-vacuous runtime coverage, and the two previously-uncovered code branches are now exercised. The verdict carries warnings only for two pre-existing PARTIAL scenarios (interactive `--only` selector, end-to-end brew list version) unrelated to this change's remediation. No source code was modified by this verification.
