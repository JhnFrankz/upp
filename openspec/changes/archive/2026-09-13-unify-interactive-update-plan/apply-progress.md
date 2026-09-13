# Apply Progress: Unify interactive `upp update` with `engine.Plan`

## Batches

- Change: `unify-interactive-update-plan`
- Mode: **Strict TDD** (`openspec/config.yaml` `testing.strict_tdd: true`, runner `go test`)
- Delivery: `auto-chain`, chain strategy `stacked-to-main`
- **PR 1** = tasks 1.1–1.3, 2.1–2.5 (canonical identity + shared Plan executor). **DONE.**
- **PR 2** = tasks 1.4–1.5, 3.1–3.5 (plan-derived pending + `StatusDeselected`). **DONE — final slice.**
- **Remediation** = C1 + C2 + W1 + W4 (tests for the two UNTESTED ADDED scenarios + bookkeeping). **DONE.**
- No commit/push/branch performed; all changes left in the working tree for the orchestrator.
- Phase 4 (4.1–4.3) was the pre-merge verification phase; its commands were re-run green in the remediation batch, so the Phase 4 tasks are now marked `[x]` (W1).

## Completed Tasks (cumulative)

### PR 1 (previous batch)

- [x] 1.1 Rename `TestProcessSelectedOutcome_Coverage` → `TestExecutePlannedUpdate_Coverage`; each case builds `engine.PlannedUpdate`.
- [x] 1.2 Identity test `TestRunUpdate_CanonicalIdentitySelection`: `Name()="gh"`, `Info().ID="gh"`, `Info().Name="GitHub CLI"` → option `{ID:"gh", Label:"GitHub CLI"}`, `gh` updated (no silent drop).
- [x] 1.3 Privilege-parity case: `RiskCommand="sudo apt install --only-upgrade gh"`, `ManagerID="apt"`, CI → `ConfirmError` → `StatusFailed` (threat matrix).
- [x] 2.1 `toolSelectionID(a)` = `Info().ID` else `Name()`; `adapterByID` re-keyed (left `adapterIDs` on `Name()`).
- [x] 2.2 `executePlannedUpdate(...)` reads `p.RiskCommand/Privileges/Trust`, `EnforceRisk: p.ManagerID != ""`.
- [x] 2.3 Deleted `processSelectedOutcome`; sequential update block delegates to `executePlannedUpdate`.
- [x] 2.4 Unresolvable selected ID → explicit `output.StatusFailed`; no silent `continue`.
- [x] 2.5 `go test ./internal/cli/ -count=1` green.

### PR 2 (previous batch)

- [x] 1.4 Flipped the D7 tests: always-update current tools are now in `wantOpts` (`TestRunUpdate_InteractiveSelection`, `TestRunUpdate_SelectorCancel`, renamed `TestRunUpdate_ManagerSelfUpdateBrewInSelector`), and added `TestRunUpdate_AlwaysUpdateCurrentSelectedExecutes` (`Name()="bun"` ≠ `Info().Name="Bun"`) proving a selected always-update current tool executes across the plan→selector→execution boundary.
- [x] 1.5 `internal/output/render_test.go`: `TestUpdateSummary_Deselected`, `TestUpdateSummary_OnlyDeselected`, `TestUpdateSummary_DeselectedDistinctFromSkipped`; deselected never counted skipped/updated, and "All tools not installed" is absent when only deselected. Icon tables extended with `StatusDeselected`.
- [x] 1.6 RED confirmed before implementation (`go test ./internal/output/ ./internal/cli/ -count=1` → output build failure `undefined: StatusDeselected`; cli selector-option count mismatches + `TestRunUpdate_AlwaysUpdateCurrentSelectedExecutes` selector options = 0).
- [x] 3.1 `runUpdateInteractive` builds the selector from `plan.Updates` (`ID=u.ToolID`, `Label=u.ToolName`, `Version=Current → Latest`) in canonical order; executes `planByID[oc.ToolID]`.
- [x] 3.2 Emits `output.StatusDeselected` for pending-not-selected; no silent drop. An unplanned/unknown selected ID also surfaces as an explicit `StatusFailed` (spec Unresolvable selection).
- [x] 3.3 `internal/output/render.go`: added `StatusDeselected` const (after `StatusCurrent`), icon `"☐ "`, plain `"[deselected]"`, label `"deselected"`, verbose line `(deselected)`.
- [x] 3.4 `UpdateSummary`: `deselected := countByStatusType(..., StatusDeselected)` (`countByStatus` 3-value signature kept); `"%d deselected"` part; `deselected == 0` added to the not-installed guard and `allClean`; `Deselected:` detail line; deselected-only branch renders the deselected icon.
- [x] 3.5 `go test ./internal/output/ ./internal/cli/ -count=1` green.

### Remediation (this batch)

Addresses the verify-report FAIL (C1, C2) plus bookkeeping (W1, W4). No production
behavior changed; only new tests + SDD bookkeeping.

- [x] C1 `TestRunUpdate_EmptyIDFallbackSelection` (`internal/cli/update_test.go`): fakes an adapter whose `Info().ID == ""` but `Name() == "npm"` (via the new `emptyIDAdapter` wrapper, which never sets `Info().ID`), drives the interactive selector, and asserts `ID="npm"` (the `Name()` fallback), that the tool resolves and executes, and that the summary reports `Updated: npm` — never a silent drop. Covers `toolSelectionID` fallback (66.7% → **100%**).
- [x] C2 `TestRunUpdate_UnresolvableSelection` (`internal/cli/update_test.go`): selects `"ghost-tool"`, an ID absent from the plan-derived pending set, and asserts the interactive loop surfaces it as an explicit `Failed: ghost-tool` / `1 failed` result (never a silent no-op), while the unselected pending `npm` is reported `Deselected: npm`. Covers `update.go:409-418` (`runUpdateInteractive` 79.1% → **81.3%**).
- [x] W1 `tasks.md` Phase 4 (4.1–4.3) marked `[x]` after re-running every command green in this batch.
- [x] W4 TDD table TRIANGULATE row for 3.2 corrected to name the covering test (`TestRunUpdate_UnresolvableSelection`) instead of claiming an untested branch.

## Files Changed (cumulative)

| File | Action | What Was Done |
|------|--------|---------------|
| `internal/cli/checkrun.go` | Modified (PR 1) | Added `toolSelectionID`; re-keyed `adapterByID` by canonical ID; `adapterIDs` unchanged. |
| `internal/cli/update.go` | Modified (PR 1 + PR 2) | PR 1: shared `executePlannedUpdate` + `outcomeSelectionID`; deleted `processSelectedOutcome`; sequential delegates. PR 2: selector/pending built from `plan.Updates` (PC1); deselected pending tools emitted as `StatusDeselected` (PC2); unresolvable selected IDs fail explicitly. |
| `internal/cli/update_test.go` | Modified (PR 1 + PR 2 + Remediation) | PR 1: `infoName` override, migrated coverage test, canonical identity test, `engine` import. PR 2: flipped the three D7 spots, added the always-update-selected execution test, added deselected assertions to the owned-tool delegation test. Remediation: added the `emptyIDAdapter` wrapper + `TestRunUpdate_EmptyIDFallbackSelection` (C1) + `TestRunUpdate_UnresolvableSelection` (C2). |
| `internal/output/render.go` | Modified (PR 2) | `StatusDeselected` const/icon/plain/label/line; `UpdateSummary` deselected count/part/guard/`allClean`/branch; `Deselected:` detail line. |
| `internal/output/render_test.go` | Modified (PR 2) | Deselected summary/detail/guard coverage + icon table cases. |
| `openspec/changes/unify-interactive-update-plan/tasks.md` | Modified | Marked 1.1–1.3/2.1–2.5 (PR 1), 1.4–1.6/3.1–3.5 (PR 2), and 4.1–4.3 (Remediation, W1) `[x]`. |

## TDD Cycle Evidence

| Task | Test File | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|------|-----------|-------|------------|-----|-------|-------------|----------|
| 1.4 | `internal/cli/update_test.go` | Integration | ✅ cli package ok | ✅ Written (option-count mismatches, selector options = 0) | ✅ Passed | ✅ 4 tests (InteractiveSelection, SelectorCancel, BrewInSelector, AlwaysUpdateCurrentSelected) | ✅ Clean |
| 1.5 | `internal/output/render_test.go` | Unit | ✅ output package ok | ✅ Written (`undefined: StatusDeselected`) | ✅ Passed | ✅ 3 summary cases + 2 icon tables | ✅ Clean |
| 3.1 | `internal/cli/update.go` | Integration | ✅ cli package ok | ✅ 1.4 covers | ✅ Passed | ✅ plan-order + group label + canonical ID | ✅ Clean |
| 3.2 | `internal/cli/update.go` | Unit/Integration | ✅ cli package ok | ✅ 1.4/1.5 cover | ✅ Passed | ✅ `TestRunUpdate_InteractiveSelection` + `_OwnedToolDelegation` (deselected) + `TestRunUpdate_UnresolvableSelection` (unresolvable branch, C2) | ✅ Clean |
| 3.3 | `internal/output/render.go` | Unit | ✅ output package ok | ✅ 1.5 covers | ✅ Passed | ✅ emoji + plain icon cases | ✅ Clean |
| 3.4 | `internal/output/render.go` | Unit | ✅ output package ok | ✅ 1.5 covers | ✅ Passed | ✅ updated+desel, only-desel, desel+skipped | ✅ Clean |
| 3.5 | — | — | — | — | ✅ `go test ./internal/output/ ./internal/cli/ -count=1` ok | — | — |
| 1.6 | — | — | — | ✅ `go test ./... -count=1` RED confirmed | — | — | — |
| C1 | `internal/cli/update_test.go` | Integration | ✅ cli package ok (baseline 0.099s) | ✅ Mutation: `toolSelectionID` fallback removed → `TestRunUpdate_EmptyIDFallbackSelection` FAILS (`Failed: npm`, `fake.updated=false`); mutation reverted | ✅ Passed | ✅ `TestRunUpdate_EmptyIDFallbackSelection`; `toolSelectionID` 66.7% → 100% | ✅ Clean |
| C2 | `internal/cli/update_test.go` | Integration | ✅ cli package ok | ✅ Mutation: `update.go:409-418` condition inverted → `TestRunUpdate_UnresolvableSelection` FAILS (ghost-tool silently dropped: only `1 deselected`); mutation reverted | ✅ Passed | ✅ `TestRunUpdate_UnresolvableSelection`; branch 409-418 covered, `runUpdateInteractive` 79.1% → 81.3% | ✅ Clean |

### Test Summary

- Tests written/rewritten this batch: 7 (4 cli + 3 output) plus 3 extended icon/keyed assertions.
- Layers used this batch: Unit (3), Integration (4).
- Approval tests (refactoring): none — the batch is additive/behavioral, not a refactor of existing code.
- Pure functions created: none new (`StatusDeselected` is a const; the plan-derived pending build is inline and deterministic).
- PR 1 (previous) cumulative: 5 tests, Unit (4) + Integration (1).
- Remediation batch: 2 integration tests + 1 test wrapper (`emptyIDAdapter`) added; 0 production lines changed. Both are characterization tests for already-shipped-but-untested branches, so non-vacuity was proven by temporary production mutations (recorded in the RED column) that were reverted byte-for-byte (`git diff --stat` for `checkrun.go`/`update.go` unchanged at 18/325 before and after).

## Work Unit Evidence

### PR 2 work unit — plan-derived pending + `StatusDeselected`

| Evidence | Value |
|---|---|
| Focused test command and exact result | `go test ./internal/output/ ./internal/cli/ -count=1` → `ok github.com/JhnFrankz/upp/internal/output 6.882s`; `ok github.com/JhnFrankz/upp/internal/cli 0.075s` |
| Runtime harness command/scenario and exact result | `go build -o upp ./cmd/upp && bash scripts/smoke-test.sh --skip-build` (read-only) → `Results: 35 passed, 0 failed, 35 total — All tests passed!` |
| Rollback boundary | Revert `internal/output/render.go`, `internal/output/render_test.go`, the PR 2 hunks in `internal/cli/update.go` + `internal/cli/update_test.go`, and the tasks.md 1.4–1.6/3.1–3.5 checkboxes; PR 1's identity/executor work is untouched. No persisted state, flags, or specs changed. |

### Remediation work unit — ADDED scenario coverage (C1/C2)

| Evidence | Value |
|---|---|
| Focused test command and exact result | `go test ./internal/cli/ -count=1 -run 'TestRunUpdate_EmptyIDFallbackSelection\|TestRunUpdate_UnresolvableSelection' -v` → both PASS, `ok github.com/JhnFrankz/upp/internal/cli 0.003s` |
| Runtime harness command/scenario and exact result | `go build -o upp ./cmd/upp && bash scripts/smoke-test.sh --skip-build` (read-only) → `Results: 35 passed, 0 failed, 35 total — All tests passed!` |
| Rollback boundary | Revert only the remediation additions in `internal/cli/update_test.go` (`emptyIDAdapter` wrapper + the two tests) and the `tasks.md` 4.1–4.3 checkboxes / this apply-progress batch; no production file was modified by this batch (`checkrun.go`/`update.go` diff unchanged). |

### PR 1 work unit (previous batch, retained)

| Evidence | Value |
|---|---|
| Focused test command and exact result | `go test ./internal/cli/ -count=1 -run 'TestExecutePlannedUpdate_Coverage\|TestRunUpdate_CanonicalIdentitySelection' -v` → `ok ... 0.004s` |
| Runtime harness command/scenario and exact result | N/A — unit/integration harness over injected adapters; no real package mutation. |
| Rollback boundary | Revert `internal/cli/checkrun.go`, `internal/cli/update.go`, `internal/cli/update_test.go` and tasks 1.1–1.3/2.1–2.5 checkboxes. |

## Verification Commands (cumulative, run after PR 2)

| Command | Result |
|---|---|
| `go test ./... -count=1` | ✅ all 10 packages ok (adapters, official, cli, config, engine, output, platform, security, selfupdate, uninstall) |
| `go test ./internal/cli/... ./internal/output/... -count=1 -race` | ✅ `ok internal/cli 1.133s`; `ok internal/output 7.450s` |
| `go vet ./...` | ✅ clean (no output, exit 0) |
| `gofmt -l .` | ✅ clean (no output, exit 0) |
| `bash scripts/smoke-test.sh --skip-build` | ✅ 35 passed, 0 failed |

### Remediation batch re-run (all required gates)

| Command | Result |
|---|---|
| `go test ./... -count=1 -race` | ✅ exit 0 — all 10 packages ok (`cli 1.144s`, `output 9.368s`, …) |
| `go build ./...` | ✅ exit 0 (empty output) |
| `go vet ./...` | ✅ exit 0 (empty output) |
| `gofmt -l .` | ✅ empty (exit 0) |
| `go build -o upp ./cmd/upp && bash scripts/smoke-test.sh --skip-build` | ✅ exit 0 — `Results: 35 passed, 0 failed, 35 total` |
| Coverage | `toolSelectionID` 100% (was 66.7%); `runUpdateInteractive` 81.3% (was 79.1%); branch `update.go:409-418` now covered. Remaining uncovered `update.go:386-397` is the defensive pending-with-no-adapter branch, unreachable when adapters are injected (plan keys ≡ adapter-map keys). |

## Changed Lines

- **Remediation (this batch): ~121 authored changed lines** — all additions to `internal/cli/update_test.go` (`emptyIDAdapter` wrapper + the two C1/C2 tests); 0 deletions and 0 production lines. Measured two ways: `git diff --stat` for `update_test.go` grew 307 → 428 (+121) and `wc -l` grew 1995 → 2116 (+121). **Well under the 400-line review budget — no `size:exception` needed.**
- PR 2 (previous batch): ~317 authored changed lines (Additions+Deletions) — `internal/output/render.go` 33+7, `internal/output/render_test.go` 81+0, and ~196 on `internal/cli/{update,update_test}.go` (derived as the current cumulative diff 650 minus PR 1's reported 454).
- PR 1 (previous batch): 454 authored changed lines (documented as a `size:exception` recommendation).

## Deviations from Design

- PR 1: `executePlannedUpdate` takes `allAdapters ...[]adapters.Adapter` (variadic) rather than the design's `all []adapters.Adapter`, matching surrounding conventions; `uf *UpdateFlags` is omitted because dry-run is handled before the executor. The executor intentionally has no policy-gate branch — eligibility lives entirely in `engine.Plan` (design D2/D5).
- PR 2: the interactive loop iterates `outcomes` in canonical order and consults `planByID` to decide pending vs not (design line 49), then adds an explicit `StatusFailed` for any selected ID that matched no plan/outcome — preserving PR 1's "Unresolvable selection" contract. Deselected results carry no `Version` (the status is what matters); the summary reports a distinct count and detail line.
- PR 2: added a deselected-only summary branch (`StatusDeselected` icon) so "all pending deselected" does not fall through to the current-tool line — a small render extension implied by the "All pending deselected" scenario.

## Issues Found

- Verification FAIL (C1/C2) resolved by this batch: both required ADDED scenarios now have runtime-covering tests (`TestRunUpdate_EmptyIDFallbackSelection`, `TestRunUpdate_UnresolvableSelection`). W1 (Phase 4 checkboxes) and W4 (TDD TRIANGULATE row) resolved.
- No production behavior changed; all required verification commands pass; the remediation batch lands well within budget.

## Remaining Tasks

- [x] 4.1 `go test ./... -count=1` green. (Re-run as `-race` in the remediation batch → exit 0.)
- [x] 4.2 `bash scripts/smoke-test.sh --skip-build` (read-only). (Re-run in the remediation batch → 35 passed, 0 failed.)
- [x] 4.3 Confirm CI exit stays `hasFailure`-only and `--dry-run` renders no selector. (Unchanged gate `update.go:423-425`; `TestRunUpdate_SelectorGateMatrix` "--dry-run skips selector" passed in the `-race` run; smoke "upp update --dry-run (query header)".)
