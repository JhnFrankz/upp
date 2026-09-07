# Delta for bulk-update

<!-- Archive note: after merging, update the Purpose paragraph to drop "and filterable via `--manager <mgr>` / `--update-group <mgr>` flags". The default delegated path is the only path. Purpose prose is outside requirement blocks and cannot be delta-edited. -->

## MODIFIED Requirements

### Requirement: Default Group Bulk Trigger

The system MUST execute manager-group bulk package updates for all owned tools by default during standard `upp update` execution. A bare `upp update` (no flag) MUST trigger manager-group bulk updates for all resolved manager groups containing enabled owned tools, updating those owned tools via their manager's package update mechanism while executing standalone tools via their standard adapters. There are no opt-in trigger flags: `--manager` and `--update-group` MUST NOT exist as flags, and the group path MUST NOT be separately invokable.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Default runs group bulk updates | `upp update` invoked with no flags on Linux with apt owning gh and docker | Execution | apt's owned group (gh, docker) bulk package updates execute by default alongside standalone tools |
| `--manager` rejected | Linux, env has apt owning gh/docker | `upp update --manager apt` | Error: unknown flag "manager", usage hint, exit non-zero; no update executed |
| `--update-group` rejected | macOS, brew owned tools present | `upp update --update-group brew` | Error: unknown flag "update-group", usage hint, exit non-zero; no update executed |
| Standalone tools preserved | Linux, apt owns gh and standalone bun/nvm enabled | `upp update` | apt group updates gh package and bun/nvm execute via standalone adapters |

(Previously: the requirement stated `--manager <mgr>` and `--update-group <mgr>` MUST remain supported as explicit filters restricting execution to the specified manager group; the opt-in trigger path is deleted and the default delegated path is the only update path.)

### Requirement: Owned-Tool Enumeration & Exclusion

When a group bulk update is triggered by default execution, the system MUST enumerate the manager's resolving owned tools on the current platform. Enumeration MUST be derived solely from owner declarations and the `--only` filter; the system MUST NOT exclude owned tools by any other flag.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Enumerates resolving group | Platform Linux, apt owns gh+docker | `upp update` | Batch contains gh and docker |
| Only filter narrows batch | Platform Linux, apt owns gh+docker | `upp update --only gh` | Batch contains gh only; docker excluded |

(Previously: the requirement excluded owned tools named by `--skip <owned-tool>`; `--skip` is removed and batch narrowing is expressed through the existing `--only` filter.)

### Requirement: Group Gate Inheritance (Gated)

A group bulk update MUST inherit the manager's `UpdatePolicy` for the group: a `PolicyGated` manager (apt) gates the whole group on group availability; a `PolicyAlwaysUpdate` manager (brew, winget, scoop) MUST always run its group update when requested.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Gated group blocks | apt group, no owned package has an update | `upp update` (default run) | Group skipped; owned tools reported current |
| Gated group runs | apt group, gh has a package update | `upp update` (default run) | Gh updated (and any other available owned tool) |
| AlwaysUpdate group runs | brew group ignored by check | `upp update` (default run) | Group update runs regardless of check result |

(Previously: the three gate-inheritance scenarios were invoked via `--manager apt` / `--manager brew`; gate-inheritance semantics are unchanged — a `PolicyGated` manager still gates the group on availability, a `PolicyAlwaysUpdate` manager still runs unconditionally — and the scenarios now invoke the default bulk path.)
