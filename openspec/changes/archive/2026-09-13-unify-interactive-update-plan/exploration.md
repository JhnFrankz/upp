# Exploration: Unify the interactive `upp update` flow with `engine.Plan`

## 1. Executive Summary

The default TTY path (`upp update`, terminal, no `--ci/--quiet/--dry-run`) has two defects:

1. **Silent drop (C-01).** The selector is built with `oc.ToolName` (display name, e.g. `"Pacman Package Manager"`) but the adapter map is keyed by `a.Name()` (short ID, e.g. `"pacman"`). On mismatch the tool is skipped with a silent `continue`. Because almost every official adapter has `Info().Name != a.Name()`, the interactive path silently drops nearly every selected update today.
2. **Divergent policy (C-03).** `runUpdateSequential` derives risk/policy/privileges from `eng.Plan`; `runUpdateInteractive` does not and re-implements the same decisions in `processSelectedOutcome`. The two paths can (and do) disagree.

The change is a **bug fix + internal refactor**: make the interactive path consume `engine.Plan` as the single source of truth for identity, risk, policy, and pending-set membership, and eliminate the silent `continue`. No user-facing flag changes.

---

## 2. Current State

### 2.1 Identity model

- `adapters.Adapter.Name()` is documented as the "tool identifier" but is only the **short ID** in practice (`adapters/interface.go:51-64`).
- `adapters.ToolInfo` carries two fields (`adapters/interface.go:124-136`):
  - `ID` — stable machine key, used by config (`engine/resolve.go:39`), ownership, and `--only` (`engine/resolve.go:89-98`).
  - `Name` — human display string.
- Verified divergence for real adapters (Name() vs Info().Name):
  - `apt` → `"APT Package Manager"` (`official/apt.go:13` vs `:184`)
  - `brew` → `"Homebrew"` (`official/brew.go:13` vs `:152`)
  - `pacman` → `"Pacman Package Manager"` (`official/pacman.go:20` vs `:29`)
  - `nvm` → `"Node Version Manager"` (`official/nvm.go:16` vs `:152`)
  - `gh` → `"GitHub CLI"` (`official/gh.go:14` vs `:82`)
  - `docker` → `"Docker"`, `bun` → `"Bun"`, `go` → `"Go"`, `opencode` → `"OpenCode"`
  - Equal only for `npm`, `pnpm`, `uv`, and custom tools (`CustomAdapter.Info()` sets `ID: c.id, Name: c.id`, `adapters/custom.go:134-143`).
- The canonical values are already pinned in `official/info_test.go:27-40`.

### 2.2 Where identity flows

- `engine.safeCheck` builds `CheckOutcome.ToolID = info.ID` (fallback `a.Name()`), `ToolName = info.Name` (fallback `a.Name()`) — `engine/check.go:53-61`.
- `engine.Plan` sets `PlannedUpdate.ToolID = info.ID`, `PlannedUpdate.ToolName = info.Name` — `engine/plan.go:104-121, 153-164`.
- `output.SelectOption.ID` is documented as "tool ID (--only key)"; `Label` is the display name — `output/selector.go:13-18`.
- `outcomeToToolResult` uses `ToolName` for the summary label — `cli/checkrun.go:16-39` (display is correct there).

### 2.3 The silent drop (C-01)

`runUpdateInteractive` (`cli/update.go:346-448`):

- Builds pending options with `ID: name` where `name = oc.ToolName` (display) — `update.go:368-382`, with `Label: name`.
- Builds `selectedSet` from those display names — `update.go:409-412`.
- Builds `adapterMap := adapterByID(filteredAdapters)` — `update.go:416`.
- `adapterByID` keys on `a.Name()` (short ID) — `cli/checkrun.go:122-127`.
- Lookup `adapterMap[name]` with the display name misses, then `continue` drops the tool **with no result and no warning** — `update.go:429-432`.

Net effect: only tools where `Info().Name == Name()` (`npm`, `pnpm`, `uv`, custom) survive; `apt`, `brew`, `pacman`, `nvm`, `gh`, `docker`, `go`, `bun`, `opencode` are dropped after the user selects them. The CheckBoard still shows them as checked, so the loss is invisible.

### 2.4 The divergent policy (C-03)

`runUpdateSequential` (`update.go:150-335`) calls `eng.Plan` (`update.go:171`) and consults `PlannedUpdate.RiskCommand` / `ManagerID` (`update.go:242-251`).

`runUpdateInteractive` never calls `Plan`. `processSelectedOutcome` (`update.go:452-556`) re-implements:

- Risk command construction via `UpdateCmdName` + `OwnedPackage` — `update.go:459-472` (duplicates `engine/plan.go:127-147`).
- Policy gate via `ResolveEffectiveUpdatePolicy` — `update.go:500` (duplicates `engine/plan.go:89-102`).

The pending set is also derived from `oc.Status == StatusAvailable` only (`update.go:369-382`), which excludes `PolicyAlwaysUpdate` tools that `Plan` treats as eligible even when `UpdateAvailable=false` (`engine/plan.go:92-97`). `Plan` already models this; the interactive path does not.

### 2.5 Test blind spot

`fakeUpdateAdapter.Info()` forces `ID: f.name, Name: f.name` (`cli/update_test.go:78-90`), so the CLI suite can never observe `Info().Name != Name()`. `engine/plan_test.go` *does* exercise `ID != Name` (`:136-142`), proving the engine layer is already identity-correct. Selector assertions (`update_test.go:856-868`, `:1237-1248`) pin `ID`/`Label` to short names and pass only because the fakes collapse the two.

---

## 3. Affected Areas

- `internal/cli/update.go` — pending construction, `selectedSet`, adapter lookup, `processSelectedOutcome`; the core of both defects.
- `internal/cli/checkrun.go` — `adapterByID` keying (`:122-127`); `adapterIDs` (`:114-120`) feeds `--only` validation.
- `internal/engine/types.go` — `PlannedUpdate` (`:64-75`) lacks an `UpdateAvailable` bit, which the unified pending derivation needs.
- `internal/engine/plan.go` — likely the home of a "pending" helper or a `Filter` option; already computes `RiskCommand`/`Privileges`/`ManagerID`.
- `internal/output/selector.go` — no behavior change expected; may only need doc/field clarification.
- `internal/output/render.go` — only if "deselected" becomes a first-class report status (`UpdateSummary` at `:276-339`, `detailSummary` at `:341+`).
- `internal/cli/update_test.go` — fake identity + new regression coverage.
- `internal/engine/plan_test.go` — if `PlannedUpdate` gains a field.
- `internal/output/render_test.go` — if a `StatusDeselected` is introduced.

---

## 4. Open Design Questions — Options & Tradeoffs

### Q1. Correct identity model: `ToolID` (stable) vs `Name` (display)

**Recommendation: canonical identity is `ToolInfo.ID`, with `Adapter.Name()` as fallback only.**

| Option | Pros | Cons | Effort |
|---|---|---|---|
| A. `ToolInfo.ID` is identity; add `adapterID(a)` helper (`Info().ID`, fallback `Name()`) used by selector IDs and the adapter map | Matches config/ownership/`--only`; `PlannedUpdate.ToolID` already exists; fixes C-01 at the root | `Adapter.Name()` remains a second, undocumented-as-identity shortcut that can drift | Low |
| B. `Adapter.Name()` is identity everywhere | Smallest diff (map already uses it) | `Name()` is documented ambiguously and is not the config key; custom/official could diverge; contradicts `SelectOption.ID` doc | Low |
| C. Keep both but centralize in one `ToolKey` helper | Same as A, explicit | Still two notions unless `Name()` is deprecated as identity | Low |

Who consumes what:
- **Selector**: `SelectOption.ID` must be `ToolID`; `Label` stays `ToolName` (display).
- **Adapter map**: key by `Info().ID` (fallback `Name()`); optionally index both to be tolerant.
- **Summary**: keep `ToolName` (display) — already correct.
- **`--only`**: already matches both `Name()` and `ID` (`engine/resolve.go:89-98`), so no user-visible change.

### Q2. Derive the pending set from `plan.Updates` instead of `StatusAvailable` only

The brew/bun case: `Plan` marks `PolicyAlwaysUpdate` adapters eligible even when `UpdateAvailable=false` (`engine/plan.go:92-97`), so `plan.Updates` ⊇ actual updates. The current TTY product rule (design D7, pinned by `update_test.go:837-839, 1016-1023, 1249-1252`) is that current AlwaysUpdate tools do **not** appear in the selector. `PlannedUpdate` currently drops the `UpdateAvailable` bit (`engine/types.go:64-75`), so `plan.Updates` alone cannot reproduce that rule.

| Option | Pros | Cons | Effort |
|---|---|---|---|
| A. Pending = outcomes with `UpdateAvailable`, but each carries its matched `PlannedUpdate` for risk/policy metadata | Preserves D7 exactly; uses `Plan` for all metadata; no `PlannedUpdate` schema change | Pending membership still decided in CLI (partially) | Low |
| B. Add `UpdateAvailable` to `PlannedUpdate` + engine helper `PendingUpdates(plan)` | Single semantic owner in engine; CLI has zero policy logic; D7 encoded once | Schema/test change in `engine` | Low-Med |
| C. Pending = all `plan.Updates` (AlwaysUpdate-current included) | Simplest unification | Changes TTY behavior; contradicts D7 and existing tests/spec; needs explicit product decision | Low |

**Recommendation: B.** Add `UpdateAvailable` to `PlannedUpdate` (or a `Reason` enum) and an engine helper so "what is pending" lives with `Plan`. If the team wants to keep the change minimal, A is an acceptable interim that still fixes C-01/C-03. C requires an explicit product call and spec delta.

### Q3. Does the sequential path stay, get unified, or become a strategy?

| Option | Pros | Cons | Effort |
|---|---|---|---|
| A. Keep both entry points; both consume `eng.Plan`; extract one shared "execute one `PlannedUpdate`" helper; delete `processSelectedOutcome` duplication | Bounded diff; preserves TTY vs non-TTY/CI/dry-run gate; single policy source | Two call sites remain (but share execution) | Med |
| B. Move execution into engine as a strategy (`engine.Apply(plan, selection)`) | Strongest unification; engine owns plan+apply | Engine is currently pure planning; large refactor + spec churn; risks 400-line budget | High |
| C. Delete sequential; always run interactive with all preselected | One path | Breaks non-TTY/CI/dry-run (must not prompt); not viable | — |

**Recommendation: A.** The two entry points exist for a real reason (the TTY gate at `update.go:131-139`). Unify the *decisions* (Plan), not the *entry points*. Keep the diff inside `cli/update.go` + a small engine addition.

### Q4. Semantics of "deselected" vs "current" vs "skipped"

Current (`output/render.go:19-25`, `:276-339`):
- `StatusUpdated` — updated.
- `StatusCurrent` — installed and up to date.
- `StatusSkipped` — not installed / confirm-deny.
- `StatusAvailable` — dry-run "would update" only.

Today a **deselected** pending tool is silently dropped (`update.go:426-428`), so it appears in no count and no detail line. That is a second silent-loss channel.

| Option | Pros | Cons | Effort |
|---|---|---|---|
| A. Deselected → `StatusSkipped` | Tiny diff; shows in "N skipped" | Conflates "user opted out" with "not installed"/"denied" | Low |
| B. New `StatusDeselected` with its own count + detail line | Honest model; fully closes the silent-loss class | Ripples through `render.go` (+tests), possibly specs | Med |
| C. Keep dropped | No diff | Silent loss remains; contradicts this change's goal | — |

**Recommendation: B**, budget permitting; otherwise A. The change's stated goal ("eliminate silent loss") argues for making deselection visible. Do not reuse `StatusCurrent` for deselected.

### Q5. Migrate tests that assume `ID == Name` without losing coverage

- Change `fakeUpdateAdapter` to carry a distinct `displayName` field; `Info()` returns `ID: f.name, Name: f.displayName` defaulting to `f.name` when empty (`cli/update_test.go:78-90`). Defaulting keeps every existing assertion green.
- Add one regression test with `name != displayName` asserting: (a) the selector option `ID == name` and `Label == displayName`; (b) the update actually executes (no silent `continue`); (c) the summary reports the display name.
- Add a real-adapter identity guard asserting `Info().Name != Name()` for at least `brew`/`pacman` (data already in `official/info_test.go:27-40`), so the suite can never regress to "everything is npm-like".
- Keep `engine/plan_test.go:136-142` (`ID != Name`) as the engine-level guard.

### Q6. Blast radius and CLI output backward-compat

- The **fix changes output by design**: today selected tools are dropped; after the fix they are updated. The visual selector/list rendering is otherwise unchanged because `Label` was already the display name and the CheckBoard already used `Info().Name` (`update.go:358`).
- `SelectOption.ID` transitions from display name to stable ID. `ID` is not rendered, so no visual change; it is the selection key only.
- If Q4 option B is taken, the summary gains a "deselected" count/line — a deliberate, documented output delta.
- `--only` semantics are untouched (already matches both keys).
- Secondary divergence worth deciding: sequential wraps owned adapters with `prepareCheckAdapters` (`update.go:165`) while interactive does not (`update.go:362`). Official owned adapters self-delegate `Check()` (`official/gh.go:20-45`), so parity holds today, but a custom owned tool could diverge. Align both paths in the unified flow.

---

## 5. Recommendation

Adopt **Q1-A + Q2-B + Q3-A + Q4-B (or A) + Q5 + Q6**:

1. Make `ToolInfo.ID` the canonical selection identity; key `adapterByID` on `Info().ID` (fallback `Name()`); set `SelectOption.ID = PlannedUpdate.ToolID`, `Label = ToolName`.
2. Add `UpdateAvailable` to `PlannedUpdate` and an engine `PendingUpdates` helper so the pending set (and the brew/bun D7 rule) is defined once in `engine`.
3. Route the interactive path through `eng.Plan`, consume `RiskCommand`/`Privileges`/`ManagerID`, and delete the `processSelectedOutcome` re-implementation.
4. Make deselection visible in the report (new status preferred; `StatusSkipped` acceptable fallback).
5. Harden the fakes with `displayName` and add the regression cases.

This is a **Medium** effort, mostly confined to `internal/cli/update.go`, `internal/cli/checkrun.go`, and a small `internal/engine` addition, with test updates in `internal/cli/update_test.go` and possibly `internal/output/render.go`.

---

## 6. Risks

- **Behavior-change surprise**: fixing the drop means TTY runs now execute updates users previously (unknowingly) didn't get. Intended, but should be called out in the proposal.
- **`PlannedUpdate` schema change** ripples into `engine/plan_test.go` and any consumers; keep the addition additive.
- **New `StatusDeselected`** touches render counts/detail and tests; verify CI exit semantics stay unchanged (`hasFailure` only).
- **Privileges semantics shift**: `Plan` may derive privileges from the manager and `detectPrivileges(riskCmd)` (`engine/plan.go:133-151`) whereas `processSelectedOutcome` passes only `info.Privileges` (`update.go:479`). Confirm `ConfirmAction` behavior is preserved or intentionally improved.
- **D7 ambiguity**: if the team actually wants AlwaysUpdate tools in the TTY selector, Q2-C must be chosen deliberately and specs updated — otherwise the change silently alters product behavior.

## 7. Ready for Proposal

**Yes.** The two defects are evidenced at `file:line`, the identity divergence is confirmed for real adapters, and the design forks (Q2 pending semantics, Q4 deselected status) are bounded and can be decided in the proposal. The orchestrator should tell the user that (a) the interactive path is broken for nearly every real tool today, and (b) the proposal must explicitly choose whether AlwaysUpdate-current tools (brew/bun) stay out of the TTY selector (recommended, preserves D7) and whether deselection becomes a visible report status.
