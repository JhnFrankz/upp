# Delta for tool-ownership-model

## MODIFIED Requirements

### Requirement: Resolved-Owner Group Bulk Update

Given a manager and platform, the system MUST be able to update that manager's resolving owned set as one group. The group update MUST enumerate the manager's owned tools (from owner declarations), MUST check each owned tool's package availability, and MUST run each owned tool's per-manager package command. The manager's own self-only row MUST remain distinct from the group.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| brew group on macOS | Platform macOS, brew owns gh, docker, go | Group update for brew | Runs brew's package commands for gh, docker, go |
| apt group on Linux | Platform Linux, apt owns gh, docker | Default `upp update` group update for apt | Group updates gh and docker |
| Manager self distinct | Platform Linux, apt group update | Group update for apt | Owned tools updated via package commands; apt self handled separately |

(Previously: the group update excluded any owned tool named by `--skip`; the `--skip` flag is removed and the exclusion clause is dropped with it.)
