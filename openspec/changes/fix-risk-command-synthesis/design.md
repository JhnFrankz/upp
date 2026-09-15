# Design: RiskCommand From the Real Executed Command

## Technical Approach

Adapters declare their real update commands as **data on `ToolInfo`**; `engine.Plan` derives `RiskCommand`/`Privileges` from those declarations instead of `UpdateCmdName` synthesis; each manager's `Update()`/`UpdatePackage()` is refactored to execute its own declaration (shared constants), making plan/execution divergence structurally impossible and test-enforced via the `runCmdFn` seam. The gate (`update.go:444-453`) keeps consuming `plan.RiskCommand` — only its input becomes truthful.

## Architecture Decisions

### D1: Declaration mechanism — `ToolInfo` fields, not interface methods

| Option | Tradeoff | Verdict |
|---|---|---|
| A: method on `PackageUpdater` (`UpdateCommand(pkg)`) | Per-package natural, but scoop is `KindManager` with **no** `PackageUpdater` (needs a second mechanism); interface churn across managers; plan needs type assertions | rejected |
| B: `ToolInfo` declaration fields | Zero interface churn; plan already reads `owner.Info()`; uniform for all 5 managers incl. scoop; matches spec text ("adapter declaration") | **chosen** |
| C: `*UpdateCommands` sub-struct on `ToolInfo` | Nil-guard noise at every read site and test fixture | rejected |

New flat fields: `SelfUpdateCommand` (exact command `Update()` runs), `PackageUpdateCommand` (per-package template with `<pkg>` placeholder).

### D2: Single source of truth = constants consumed by both declaration and execution

Each manager defines package-level command constants; `Info()` returns them; `Update()`/`UpdatePackage()` run them (`runCmd(pacmanSelfUpdate)`, `runCmd(renderPackageCommand(pacmanPackageCmd, pkg))`). Editing a command edits one identifier; the byte-equality test (T3) catches re-inlined drift.

### D3: Plan derivation by row class (plan.go)

| Row class (reachability) | RiskCommand after | Decision vs today |
|---|---|---|
| Manager self (`owner==nil`, `Kind==KindManager`) | `info.SelfUpdateCommand` | unchanged except pacman (D4) |
| Owned package (`owner!=nil`, `pkg!=""`) | `render(owner.PackageUpdateCommand, pkg)` | apt/brew/winget byte-stable (existing pins pass); pacman now High |
| Custom delegated (`owner!=nil`, `info.Manager==nil`, `pkg==""`) | `owner.SelfUpdateCommand` — matches execution `custom.go:91` → `manager.Update()` (discovery #10) | unchanged except pacman-delegated now High (spec) |
| Custom standalone | `info.Command` (unchanged path) | unchanged |
| Official standalone | fallback `<Name> update` (unchanged) | unchanged — residual fiction documented |

Unreachable corner (official owned, `pkg==""`): standalone fallback rule. Note: apt/brew/winget **self-rows** change displayed command from the `"<Name> update"` fiction to the real command (mandated by tool-adapter/security-model transparency deltas) with **zero decision churn** — the byte-stable requirement pins the owned-row commands, which stay identical.

### D4: Pacman self-row gate enablement — `ManagerID` on privileged self-rows

Spec requires pacman self-update to prompt and fail `--ci`; the gate auto-proceeds `TrustOfficial` unless `EnforceRisk` (update.go:452, derived `p.ManagerID != ""`). Plan sets `ManagerID` on a manager self-row **iff its `Info()` declares privileges**. Only pacman declares today → exactly pacman rows tighten; apt/brew/winget keep byte-identical decisions. Rule is fail-safe: declarations can only tighten the gate. Rejected: `EnforceRisk bool` on `PlannedUpdate` (touches gate logic; proposal scopes update.go to verification). update.go:435-436 gets a comment-only clarification; no logic change.

### D5: Privileges

pacman `Info()` declares `Privileges: ["sudo"]`. Plan precedence untouched (row-declared → owner-inherited → `detectPrivileges` fallback, plan.go:118-151). `detectPrivileges` dedup stays out of scope (audit #5).

### D6: Synthesis retirement

`UpdateCmdName` deleted (resolve.go) with its tests. `renderPackageCommand` (adapters package: replace first `<pkg>`; if placeholder absent, append `" "+pkg`) replaces synthesis.

## Data Flow

```
manager Info(): SelfUpdateCommand / PackageUpdateCommand / Privileges  (consts)
      │                                     ▲ same consts
      ▼                                     │
engine.Plan ──render──► PlannedUpdate.RiskCommand / Privileges ──► gate (ClassifyCommand → ConfirmAction)
      owner.Info()                                          │ approved
                                                            ▼
                                executePlannedUpdate → Update()/UpdatePackage() → runCmd(consts)
```

## File Changes

| File | Action | Description |
|---|---|---|
| `internal/adapters/interface.go` | Modify | ToolInfo fields, `PackagePlaceholder` const, `renderPackageCommand` |
| `internal/adapters/official/{apt,brew,winget,scoop,pacman}.go` | Modify | command constants; `Info()` declares; `Update`/`UpdatePackage` consume; pacman `Privileges:["sudo"]` |
| `internal/engine/plan.go` | Modify | row-class derivation (D3/D4); drop `UpdateCmdName` |
| `internal/engine/resolve.go` | Modify | delete `UpdateCmdName` |
| `internal/cli/update.go` | Modify | comment-only (435-436) |
| `internal/engine/{plan,resolve}_test.go`, `internal/adapters/official/{info,update}_test.go`, `internal/cli/update_test.go` | Modify/New | T1-T4 below |

## Interfaces / Contracts

```go
// ToolInfo additions (managers; empty for tools)
SelfUpdateCommand    string // exact command Update() executes
PackageUpdateCommand string // per-package command; "<pkg>" placeholder

const PackagePlaceholder = "<pkg>"
func RenderPackageCommand(template, pkg string) string
```

Declarations: apt `sudo apt install --only-upgrade` / self `…apt`; brew `brew upgrade <pkg>` / `brew update`; winget `winget upgrade <pkg>` / `winget upgrade winget`; scoop `scoop update scoop` (self only); pacman `sudo pacman -S --noconfirm <pkg>` / `sudo pacman -S --noconfirm pacman`, `Privileges:["sudo"]`.

## Testing Strategy

| Layer | What | Approach |
|---|---|---|
| T1 Unit | Declaration sweep: every registry `KindManager` declares required commands; pacman privileges; `renderPackageCommand` table (placeholder present/absent) | table-driven over `official.AllAdapters()` |
| T2 Unit | Row-class pins: pacman self (`sudo pacman -S --noconfirm pacman`, sudo, `ManagerID` set), pacman owned render (synthetic fixture), custom `manager="pacman"` row inherits manager self command; existing apt/brew/winget/custom pins pass unchanged | `plan_test.go` |
| T3 Integration | Byte-equality: stub `runCmdFn` (helper.go:23), capture the command `Update(false)`/`UpdatePackage("gh")` executes per manager; assert `plan.RiskCommand` byte-equals the capture; privileges equal declared | engine+adapters via seam |
| T4 Gate | Privileged-manager fake: interactive prompts, `--ci` non-zero; custom `manager="pacman"` classified by real self command; apt/brew/winget decisions unchanged | `cli/update_test.go` fakes |
| E2E | Smoke | `bash scripts/smoke-test.sh --skip-build` |

## Threat Matrix

N/A — no routing, shell-command construction, subprocess, VCS/PR automation, executable-file classification, or process-integration boundary changes. Executed commands are byte-unchanged; only plan metadata and the gate's classification input become truthful. Recorded as not applicable per `references/threat-matrix.md`.

## Migration / Rollout

No migration required: plan metadata is in-memory. Reverting the commits restores prior gate inputs and prompt behavior byte-for-byte.

## Open Questions

None blocking. Residual (documented, out of scope): standalone official tools (go/Linux, npm, pnpm, nvm, …) keep the `"<Name> update"` fiction; a future change can fill `ToolInfo.Command` for them — the mechanism already supports it.
