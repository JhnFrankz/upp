# Design: Unify interactive `upp update` with `engine.Plan`

## Technical Approach

Make `engine.Plan` the single source of truth for identity, pending set, risk, and policy in both update paths. One identity helper keys selector options and adapter lookup identically; one executor applies a `PlannedUpdate`; `processSelectedOutcome` is deleted. Deselections become first-class `output.StatusDeselected`.

## Architecture Decisions

| # | Option | Tradeoff | Decision |
|---|--------|----------|----------|
| D1 Identity | Key on `Info().Name` / key on `Info().ID` fallback `Name()` | Display-name keying is the C-01 silent-drop bug (`update.go:370-381` IDs vs `checkrun.go:125` map) | `toolSelectionID(a)` = `Info().ID`, fallback `Name()`. Equals `PlannedUpdate.ToolID` (`plan.go:104-114`). Selector `ID = u.ToolID`, `Label = u.ToolName`; `adapterByID` keyed by `toolSelectionID`. |
| D2 Executor | Keep `processSelectedOutcome` + a second Plan path / one shared executor | Duplication re-diverges policy (C-03) | New `executePlannedUpdate` runs confirm + owner `UpdatePackage`/`Update`; delete `processSelectedOutcome` (`update.go:452-556`); sequential delegates its update block (`update.go:241-319`) too. |
| D3 Risk/privilege source | `info.Privileges` (`update.go:479`) / `PlannedUpdate` | Plan derives owner privileges + `detectPrivileges` (`plan.go:133-151`); old path missed them | Executor uses `p.RiskCommand`, `p.Privileges`, `p.Trust`, `EnforceRisk: p.ManagerID != ""`; `security.ConfirmAction` call shape unchanged. |
| D4 Deselected status | Reuse `StatusSkipped` / add `StatusDeselected` | Spec requires distinct count, not skipped/current | Append `StatusDeselected` after `StatusCurrent`; count via `countByStatusType` (keeps `countByStatus`'s 3-value signature used at `render_test.go:641`). |
| D5 Always-update current (PC1) | Exclude / include from pending | Reverses `ux-patterns:229` | `plan.Updates` already includes `PolicyAlwaysUpdate` even when current (`plan.go:92-97`); pending = `plan.Updates`. Not-installed (`StatusSkipped`) tools stay out of `plan.Updates`. |

## Data Flow

    eng.Check(grouped) ──→ []CheckOutcome
          │                      ├─→ CheckBoard (progress)
          │                      └─→ eng.Plan(outcomes, Filter{})
          │                                 │
          │                    plan.Updates: ToolID, ToolName, RiskCommand,
          │                    Privileges, Trust, ManagerID, Current→Latest
          │                                 │
          │                  selector options (ID=ToolID, Label=ToolName)
          │                                 │
          │                    selected set (canonical IDs)
          └─────────→ executePlannedUpdate(p, oc, adapter) ×N
                                 │
                 output.ToolResult{Updated|Failed|Skipped|Deselected}

## Interfaces / Contracts

```go
// internal/cli/checkrun.go
func toolSelectionID(a adapters.Adapter) string // Info().ID, else Name()
func adapterByID(list []adapters.Adapter) map[string]adapters.Adapter // keyed by toolSelectionID

// internal/cli/update.go
func executePlannedUpdate(gf *GlobalFlags, uf *UpdateFlags, p engine.PlannedUpdate,
    oc engine.CheckOutcome, a adapters.Adapter, index, total int,
    r *output.Renderer, osName string, all []adapters.Adapter) output.ToolResult

// internal/output/render.go
StatusDeselected Status // icon "☐ ", plain "[deselected]", label "deselected"
```

Interactive builds `plan, _ := eng.Plan(outcomes, engine.Filter{})`, `planByID`, and `adapterByID`; it iterates outcomes in canonical order, executes `planByID[oc.ToolID]` when selected, emits `StatusDeselected` when not, else `outcomeToToolResult`. A selected ID with no adapter emits `StatusFailed` (spec "Unresolvable selection"), never a silent `continue`.

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/cli/checkrun.go` | Modify | Add `toolSelectionID`; re-key `adapterByID`. Leave `adapterIDs` (FilterTools warning contract) unchanged. |
| `internal/cli/update.go` | Modify | Plan-driven pending + executor; delete `processSelectedOutcome`; sequential delegates. |
| `internal/output/render.go` | Modify | `StatusDeselected` const/icon/label/line/summary/detail. |
| `internal/output/render_test.go` | Modify | Deselected summary + detail coverage. |
| `internal/cli/update_test.go` | Modify | D7 migration, canonical-identity test, coverage migration. |

`UpdateSummary`: add `deselected := countByStatusType(...)`, a `"%d deselected"` part, include `deselected == 0` in both the "All tools not installed" guard and `allClean`, and a `Deselected:` detail line. CI exit stays `hasFailure`-only (interactive is gated `!gf.CI`, `update.go:137`).

## Testing Strategy

| Layer | What to Test | Approach |
|-------|--------------|----------|
| Unit | `executePlannedUpdate` decision branches | Migrate `TestProcessSelectedOutcome_Coverage` → `TestExecutePlannedUpdate_Coverage`, building `engine.PlannedUpdate` directly. |
| Unit | Plan privilege parity (`sudo` owned → CI `ConfirmError`) | New table case feeding `RiskCommand="sudo apt install --only-upgrade gh"`, `ManagerID="apt"`. |
| Integration | `Name() != Info().Name` crosses selector→execution | New: fake `Name()="gh"`, `Info().ID="gh"`, `Info().Name="GitHub CLI"`; assert option `{ID:"gh",Label:"GitHub CLI"}`, `gh` updated. |
| Integration | D7 flip | Update `update_test.go:835-839,1016-1023,1249-1252`: current always-update tool now in `wantOpts`, reported `StatusDeselected`. New test selects it → asserts it executes (today impossible). |
| Unit | `StatusDeselected` summary/counts | `render_test.go`: never counted as skipped/updated; "All tools not installed" not printed when only deselected. |

## Threat Matrix

| Boundary | Applicability |
|---|---|
| Documentation-like paths | N/A — no path/extension classification. |
| Git repository selection | N/A — no git invocation. |
| Commit state | N/A — no VCS automation. |
| Push state | N/A — no refspec handling. |
| PR commands | N/A — no PR composition. |

Shell/process boundary IS touched: TTY now executes previously dropped updates and risk inputs become Plan-derived. Safe behavior: `security.ConfirmAction` gates every `PlannedUpdate`; elevated risk fails closed in CI. RED test: Plan-derived `sudo` owned tool → CI `ConfirmError`.

## Migration / Rollout

No migration. Release note: TTY `upp update` now shows and executes `PolicyAlwaysUpdate`-current tools and reports deselections.

## Rollback Plan

`git revert` this change's commits; diff confined to `internal/{cli,output}`; no persisted state or flag changes.

**Budget**: authored additions+deletions ≈ 380–470 (executor/interactive rewrite ~180, render ~40, tests ~180). Likely at/over the 400-line review budget → `sdd-tasks` should forecast a chained/stacked PR by work unit (executor, status, tests).

## Open Questions

None.
