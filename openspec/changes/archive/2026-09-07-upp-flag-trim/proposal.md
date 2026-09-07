# Proposal: Trim CLI Flag Surface; Delete Dead Group Path

## Intent

`upp update`'s `--manager`/`--update-group` flags select an opt-in group path (`runUpdateGroup`) that production never calls — default update already delegates owned tools to their manager (PackageChecker/PackageUpdater). Two execution paths guarantee divergence. Global `--skip` duplicates `--only` inversely. This change collapses to one update path and a 5-flag surface.

## Scope

### In Scope
- Remove `--manager` + alias `--update-group` from `upp update`; delete `runUpdateGroup`, `filterGroupSkip`, and their exclusive tests
- Remove global `--skip`; simplify `ParseFilter`/`FilterTools` and callers (update, list)
- Delete group-only renderers `GroupBulkSummary`, `GroupBatchTool`, `GroupBatchPreview` + their tests
- Fix `self-update --help` (drop `--skip` mention), README flag table, smoke-test.sh `--skip` invocations
- Delta specs for 6 domains (see Capabilities)

### Out of Scope
- Dead `--ci` branch in `internal/cli/init.go:79-94`
- README missing `uninstall` docs; `list --only <unknown>` misleading message; config.yaml prose drift
- Any command additions/removals; any behavior change to the default delegated update path
- `openspec/changes/upp-check/` cleanup (left as-is)

## Capabilities

> Contract with sdd-spec. Verified against `openspec/specs/` on HEAD 38ccbec.

### New Capabilities
None.

### Modified Capabilities
- `command-interface`: surface reduced to `--quiet/-q`, `--verbose/-v`, `--ci`, `--only` (global) + `--dry-run/-n` (update, uninstall). "Global Flags" drops `--skip` rules/scenarios; "Self-Update Flag Semantics" drops `--skip` (help text); "`upp update`" drops explicit-manager-filter paragraph/scenarios.
- `bulk-update`: remove opt-in trigger + `--skip` exclusion from "Default Group Bulk Trigger" / "Owned-Tool Enumeration & Exclusion"; reword "Group Gate Inheritance (Gated)" scenarios to default-path invocation (gate-inheritance semantics unchanged); KEEP default delegated per-package requirements.
- `ux-patterns`: REMOVE "Opt-In Flag UX"; reword `--skip`/`--manager` scenario invocations (Summary Report, List Table Output).
- `security-model`: reword 3 "Confirmation for Destructive Operations" scenarios invoking `--manager` to default-path equivalents.
- `tool-ownership-model`: "Resolved-Owner Group Bulk Update" drops `--skip` exclusion clause/scenario.
- `tool-adapter`: reword "Update Gating" group-level scenarios (`Gated group gates on group availability`, `AlwaysUpdate group runs`) from `--manager` invocations to default-path invocation; zero semantic change — gating semantics unchanged.

(`self-update` spec pins no flags; its help-text delta lives in `command-interface`.)

## Approach

Mechanical deletion + spec deltas. Production routing (`internal/cli/update.go:77-98`) already implements the final behavior; delete the `Manager` field, its routing branch, and the dead path. Accepted equivalence: `upp update --manager apt --skip docker` → `upp update --only gh`.

## Affected Areas

| Area | Impact |
|------|--------|
| `internal/cli/update.go` | Remove Manager binding, routing branch, `runUpdateGroup` (:614-827), `filterGroupSkip` |
| `internal/cli/parser.go` | Remove `--skip`; simplify `ParseFilter`/`FilterTools` |
| `internal/cli/selfupdate.go`, `README.md`, `scripts/smoke-test.sh` | Drop `--skip` mentions/uses |
| `internal/output/render.go` | Delete group-only renderers/types |
| `internal/cli/update_test.go`, `internal/output/render_test.go` | Delete dead-path tests; keep default-path group summary tests |
| `openspec/specs/{command-interface,bulk-update,ux-patterns,security-model,tool-ownership-model}/spec.md` | Delta specs |

## Risks

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| `ParseFilter`/`FilterTools` signature ripple | Medium | `go build ./...` + `go test ./... -count=1` |
| Render-test deletion kills default-path group summary coverage | Medium | Delete only opt-in-path tests; default summary scenarios stay pinned |
| User scripts using removed flags break | Low | Accepted by user; cobra rejects explicitly; equivalence documented in README |

## Rollback Plan

Work lands as conventional commits on a change branch; `git revert <range>` fully restores flags, path, renderers, and specs. No migrations; config schema untouched.

## Dependencies

None external; cobra already rejects unregistered flags.

## Success Criteria

- [ ] Flag surface == 5: `--quiet/-q`, `--verbose/-v`, `--ci`, `--only` global; `--dry-run/-n` on update + uninstall
- [ ] `--manager`, `--update-group`, `--skip` rejected as unknown flags (non-zero exit)
- [ ] `runUpdateGroup`, `filterGroupSkip`, group renderers, dead-path tests deleted
- [ ] `self-update --help` no longer mentions `--skip`
- [ ] `go test ./... -count=1` green; smoke-test passes
