# Proposal: Unify interactive `upp update` with `engine.Plan`

## Intent

TTY `upp update` (no `--ci/--quiet/--dry-run`) silently drops nearly every selected update (C-01: display-name vs short-ID key mismatch) and re-implements policy divergent from the sequential path (C-03). Make `engine.Plan` the single source of truth for identity, pending set, risk, and policy in both paths; make deselection visible.

## Scope

### In Scope
- Canonical identity = `ToolInfo.ID` (fallback `Adapter.Name()`) for selector IDs and adapter lookup.
- Pending set = `plan.Updates`, including `PolicyAlwaysUpdate` tools even when current (PC1).
- One Plan-driven executor shared by both entry points; delete `processSelectedOutcome`.
- Distinct `output.StatusDeselected` for deselected pending tools (PC2).

### Out of Scope
- No flag changes; `--only`/`--ci`/`--dry-run`/`--quiet` unchanged.
- No engine schema change: `plan.Updates` already equals the pending set (`plan.go:92-97`); no `UpdateAvailable` field needed.
- No move of execution into `engine` (rejected Q3-B).

## Capabilities

### New Capabilities
- None.

### Modified Capabilities
- `command-interface`: TTY pending-set derivation and deselection reporting.
- `ux-patterns`: selector pending set (AlwaysUpdate-current included) and deselected summary status.

## Approach

- Identity: `SelectOption.ID = PlannedUpdate.ToolID`, `Label = ToolName`; key `adapterByID`/`adapterIDs` on `Info().ID` (fallback `Name()`).
- Pending: build the selector from `plan.Updates`, preserving canonical order and owner grouping.
- Executor: one shared helper runs confirm + owner `UpdatePackage`/`Update` for a `PlannedUpdate`; delete `processSelectedOutcome`; align `prepareCheckAdapters`.
- Deselection: emit `StatusDeselected` with its own count and detail line; never silent-drop.

## Affected Areas

All paths below are Modified.

| Area | Description |
|------|-------------|
| `internal/cli/update.go` | Plan-driven pending; shared executor |
| `internal/cli/checkrun.go` | key maps on `Info().ID` |
| `internal/output/render.go` | `StatusDeselected` count/icon/detail |
| `internal/cli/update_test.go` | fake `displayName`; D7 tests; regression |
| `internal/output/render_test.go` | deselected coverage |

Estimate: 300–420 authored lines; may approach the 400-line budget (confirm in `sdd-tasks`).

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| PC1 reverses the `ux-patterns` rule "brew MUST NOT appear in pending selector" + D7 tests (`update_test.go:835-839,1016-1023,1249-1252`) | High | Intended; delta spec and tests together |
| TTY now executes previously-dropped updates | High (intended) | Release note; single executor parity |
| `StatusDeselected` ripples counts/tests | Med | Reuse summary renderer; CI exit stays `hasFailure`-only |
| Privileges: `Plan` derives (`plan.go:149-151`) vs old `info.Privileges` (`update.go:479`) | Med | Pass `u.Privileges`; parity test |

## Rollback Plan

Revert this change's commits (`git revert`); no migration, diff confined to `internal/{cli,output}`.

## Dependencies

- Confirmed PC1/PC2 (`preproposal.yaml` revision 2).

## Success Criteria

- [ ] TTY selection of apt/brew/pacman/nvm/gh/docker/go/bun/opencode executes (no silent `continue`).
- [ ] Both paths derive identity/risk/policy from `engine.Plan`; `processSelectedOutcome` deleted.
- [ ] Deselected pending tools appear under a distinct summary status.
- [ ] `go test ./... -count=1` green with updated D7/selector tests.
