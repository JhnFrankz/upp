# Tasks: Unify interactive `upp update` with `engine.Plan`

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 380–470 authored |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | PR 1 (identity + executor) → PR 2 (pending + deselected) |
| Delivery strategy | auto-chain |
| Chain strategy | pending |

Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: pending
400-line budget risk: High

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|----------------------|-----------------|-------------------|
| 1 | Canonical identity + shared Plan executor (C-01, C-03) | PR 1 | `go test ./internal/cli/ -run 'ExecutePlannedUpdate\|OwnedToolDelegation\|CISudo'` | N/A — table/unit harness, no real package mutation | `internal/cli/checkrun.go`, `internal/cli/update.go` + WU1 tests |
| 2 | Plan-derived pending + `StatusDeselected` (PC1, PC2) | PR 2 | `go test ./internal/output/ ./internal/cli/ -run 'Deselect\|Selector\|ManagerSelfUpdate'` | `bash scripts/smoke-test.sh --skip-build` (read-only) | `internal/output/render.go` + pending block + D7 tests |

Task mapping (each PR must end green): PR 1 = 1.1–1.3, 2.1–2.5; PR 2 = 1.4–1.5, 3.1–3.5; Phase 4 runs before merge to main.

## Phase 1: RED — Failing tests (strict TDD)

- [x] 1.1 `internal/cli/update_test.go`: rename `TestProcessSelectedOutcome_Coverage` (1298) → `TestExecutePlannedUpdate_Coverage`; build `engine.PlannedUpdate` per case.
- [x] 1.2 Add identity test: fake `Name()="gh"`, `Info().ID="gh"`, `Info().Name="GitHub CLI"` → option `{ID:"gh",Label:"GitHub CLI"}` and `gh` updated.
- [x] 1.3 Add privilege-parity RED case: `RiskCommand="sudo apt install --only-upgrade gh"`, `ManagerID="apt"`, CI → `ConfirmError`/`StatusFailed` (threat matrix).
- [x] 1.4 Flip D7 at `internal/cli/update_test.go:835-839,1016-1023,1249-1252`: always-update current tools now in `wantOpts`; selecting one executes.
- [x] 1.5 `internal/output/render_test.go`: deselected never counted skipped/updated; "All tools not installed" absent when only deselected.
- [x] 1.6 Run `go test ./... -count=1` — confirm RED.

## Phase 2: GREEN — Identity + executor (PR 1)

- [x] 2.1 `internal/cli/checkrun.go`: add `toolSelectionID(a)` = `Info().ID` else `Name()`; re-key `adapterByID` (leave `adapterIDs`).
- [x] 2.2 `internal/cli/update.go`: add `executePlannedUpdate(...)` reading `p.RiskCommand/Privileges/Trust`, `EnforceRisk: p.ManagerID != ""`.
- [x] 2.3 Delete `processSelectedOutcome`; delegate sequential update block (`update.go:241-319`) to `executePlannedUpdate`.
- [x] 2.4 Unresolvable selected ID → `output.StatusFailed`, never a silent `continue`.
- [x] 2.5 `go test ./internal/cli/ -count=1`.

## Phase 3: GREEN — Pending + deselected (PR 2)

- [x] 3.1 `internal/cli/update.go`: build selector from `plan.Updates` (`ID=ToolID`, `Label=ToolName`) in canonical order; execute `planByID[oc.ToolID]`.
- [x] 3.2 Emit `output.StatusDeselected` for pending-not-selected (no silent drop).
- [x] 3.3 `internal/output/render.go`: add `StatusDeselected` const, icon `"☐ "`, plain `"[deselected]"`, label `"deselected"`.
- [x] 3.4 `UpdateSummary`: `deselected := countByStatusType(...)` (keep `countByStatus` 3-value signature); add `"%d deselected"` part; include `deselected == 0` in the not-installed guard and `allClean`; add `Deselected:` detail line.
- [x] 3.5 `go test ./internal/output/ ./internal/cli/ -count=1`.

## Phase 4: Verification

- [x] 4.1 `go test ./... -count=1` green.
- [x] 4.2 `bash scripts/smoke-test.sh --skip-build` (read-only).
- [x] 4.3 Confirm CI exit stays `hasFailure`-only and `--dry-run` renders no selector.
