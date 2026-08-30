# Delta for bulk-update

## MODIFIED Requirements

### Requirement: Default Group Bulk Trigger

The system MUST execute manager-group bulk package updates for all owned tools by default during standard `upp update` execution. A bare `upp update` (no flag) MUST trigger manager-group bulk updates for all resolved manager groups containing enabled owned tools, updating those owned tools via their manager's package update mechanism while executing standalone tools via their standard adapters. The `--manager <mgr>` and `--update-group <mgr>` flags MUST remain supported as explicit filters that restrict execution to the specified manager group.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Default runs group bulk updates | `upp update` invoked with no `--manager`/`--update-group` flags on Linux with apt owning gh and docker | Execution | apt's owned group (gh, docker) bulk package updates execute by default alongside standalone tools |
| Explicit manager filter | Linux, env has apt owning gh/docker and standalone tools | `upp update --manager apt` | apt's owned group (gh, docker) bulk-updated; standalone tools excluded |
| Explicit update-group filter | macOS, brew owned tools (gh, docker, go) and standalone tools present | `upp update --update-group brew` | brew's owned group bulk-updated; standalone tools excluded |
| Standalone tools preserved | Linux, apt owns gh and standalone bun/nvm enabled | `upp update` | apt group updates gh package and bun/nvm execute via standalone adapters |

(Previously: `Opt-In Group Bulk Trigger` specified that bare `upp update` must not trigger group bulk updates, and manager-group bulk updates were strictly opt-in via `--manager` or `--update-group`.)

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
