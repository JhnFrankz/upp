# Bulk Update Specification

## Purpose

Define the system's ability to update a manager's owned tools as one group (bulk update), executed by default during standard `upp update` and filterable via `--manager <mgr>` / `--update-group <mgr>` flags. This restores bulk-upgrade behavior anchored PER-MANAGER (not per-arbitrary-command), using the ownership model from vision point 6, so owned tools (gh, docker, go) are updated via their manager's package command (`UpdatePackage`) by default alongside standalone tools.

## Requirements

### Requirement: Default Group Bulk Trigger

The system MUST execute manager-group bulk package updates for all owned tools by default during standard `upp update` execution. A bare `upp update` (no flag) MUST trigger manager-group bulk updates for all resolved manager groups containing enabled owned tools, updating those owned tools via their manager's package update mechanism while executing standalone tools via their standard adapters. The `--manager <mgr>` and `--update-group <mgr>` flags MUST remain supported as explicit filters that restrict execution to the specified manager group.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Default runs group bulk updates | `upp update` invoked with no `--manager`/`--update-group` flags on Linux with apt owning gh and docker | Execution | apt's owned group (gh, docker) bulk package updates execute by default alongside standalone tools |
| Explicit manager filter | Linux, env has apt owning gh/docker and standalone tools | `upp update --manager apt` | apt's owned group (gh, docker) bulk-updated; standalone tools excluded |
| Explicit update-group filter | macOS, brew owned tools (gh, docker, go) and standalone tools present | `upp update --update-group brew` | brew's owned group bulk-updated; standalone tools excluded |
| Standalone tools preserved | Linux, apt owns gh and standalone bun/nvm enabled | `upp update` | apt group updates gh package and bun/nvm execute via standalone adapters |

(Previously: `Opt-In Group Bulk Trigger` specified that bare `upp update` must not trigger group bulk updates, and manager-group bulk updates were strictly opt-in via `--manager` or `--update-group`.)

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

For each batch tool in a manager group, the system MUST run that tool's per-manager package command (e.g. `brew upgrade gh`, `sudo apt install --only-upgrade gh`, `winget upgrade --id Docker.DockerCLI`) by invoking `UpdatePackage(pkg)` on the manager adapter in canonical tool discovery order, bounded by timeout, and collect per-tool execution results. The system MUST isolate errors across tools so that a failure in one owned tool's package update does not halt or abort the execution of sibling tools in the group or standalone tools. For package commands requiring elevated privileges (such as `sudo`), the system MUST prompt for confirmation in interactive TTY sessions and MUST fail closed (`EnforceRisk: true`) when running in `--ci` mode without prompting. The manager adapter's own self-only update row MUST NOT be conflated with the owned-tool group package updates.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Executes package command | macOS, gh in brew batch | `upp update` | Runs brew package update via `UpdatePackage("gh")` and collects gh result |
| Canonical order execution | macOS, brew group owning docker, gh, and go | `upp update` | Package updates execute sequentially in deterministic canonical discovery order (docker, gh, go) |
| Per-tool error isolation | Linux, apt group batch where gh package update fails | `upp update` | gh reports failure, docker package update still executes, remaining tools proceed |
| Elevated sudo fails closed in CI | Linux, apt package update requires sudo, `--ci` flag | `upp update --ci` with `EnforceRisk: true` | Command fails closed non-zero without prompting for password |
| Elevated sudo prompts in TTY | Linux, apt package update requires sudo, interactive TTY | `upp update` | Security confirmation prompt displayed before executing privileged package command |
| Manager self separate | Linux, apt group batch | `upp update` | Owned tools updated via `UpdatePackage`; apt self handled by apt's own self-only path |

(Previously: package commands were executed via privileged executor only when group bulk was explicitly invoked, lacked explicit `UpdatePackage(pkg)` delegation seam specification, canonical ordering enforcement, and formal `EnforceRisk: true` fail-closed CI risk controls.)
