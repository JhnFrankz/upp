# Bulk Update Specification

## Purpose

Define the system's ability to update a manager's owned tools as one group (bulk update), exposed as an opt-in `--manager <mgr>` / `--update-group <mgr>` flag on `upp update`. This restores v0.3.0 bulk-upgrade behavior anchored PER-MANAGER (not per-arbitrary-command), using the ownership model from vision point 6, so owned tools (gh, docker, go) are updated via their manager's package command. The default `upp update` path is unchanged in this increment (bulk is opt-in).

## Requirements

### Requirement: Opt-In Group Bulk Trigger

The system MUST expose `--manager <mgr>` and `--update-group <mgr>` as opt-in flags on `upp update` that trigger a manager-group bulk update. A bare `upp update` (no flag) MUST NOT trigger group bulk update in this increment.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Default unchanged | `upp update` invoked with no `--manager`/`--update-group` | Execution | Standard per-tool update; no group batch triggered |
| Opt-in triggers group | Linux, env has apt owning gh/docker | `upp update --manager apt` | apt's owned group (gh, docker) bulk-updated |
| Update-group alias | macOS, brew owned tools present | `upp update --update-group brew` | brew's owned group bulk-updated |

### Requirement: Owned-Tool Enumeration & Exclusion

When a group bulk update is triggered, the system MUST enumerate the manager's resolving owned tools on the current platform and MUST exclude any owned tool named by `--skip <owned-tool>`.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Enumerates resolving group | Platform Linux, apt owns gh+docker | `upp update --manager apt` | Batch contains gh and docker |
| Skip excludes owned tool | Linux, apt owns gh+docker | `upp update --manager apt --skip docker` | Batch = {gh} only; docker excluded |

### Requirement: Per-Package Availability Check

For each owned tool in the batch, the system MUST check availability of the tool's package under its manager to detect a real update (e.g. `apt-cache policy gh`); a tool whose package has no update MUST NOT be updated.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Package update available | `apt-cache policy gh`: Installed 2.45.0, Candidate 2.46.0 | Batch check | gh included in batch |
| Package current | `apt-cache policy gh`: Installed == Candidate | Batch check | gh reported current, not updated |
| Check fails | `apt-cache policy gh` exits non-zero | Batch check | gh skipped with structured error; group continues |

### Requirement: Group Gate Inheritance (Gated)

A group bulk update MUST inherit the manager's `UpdatePolicy` for the group: a `PolicyGated` manager (apt) gates the whole group on group availability; a `PolicyAlwaysUpdate` manager (brew, winget, scoop) MUST always run its group update when requested.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Gated group blocks | apt group, no owned package has an update | `upp update --manager apt` | Group skipped; owned tools reported current |
| Gated group runs | apt group, gh has a package update | `upp update --manager apt` | Gh updated (and any other available owned tool) |
| AlwaysUpdate group runs | brew group ignored by check | `upp update --manager brew` | Group update runs regardless of check result |

### Requirement: Per-Owned-Tool Command Execution

For each batch tool, the system MUST run that tool's per-manager package command (e.g. `brew upgrade gh`, `sudo apt install --only-upgrade gh`) via the manager's privileged executor, bounded by timeout, and collect per-tool results. The manager's own self-only row MUST NOT be conflated with the owned-tool group update.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Executes package command | macOS, gh in brew batch | `upp update --manager brew` | Runs `brew upgrade gh`; gh result returned |
| Manager self separate | Linux, apt group batch | `upp update --manager apt` | Owned tools updated via package command; apt self handled by apt's own self-only path |
