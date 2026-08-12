# Delta for command-interface

## MODIFIED Requirements

### Requirement: Command Structure

The system MUST support: `init`, `update`, `self-update`, `check`, `list`, `export`, `import`. Running `upp` with no arguments shows status and available updates (read-only, like `check`). Running `upp update` applies updates. `--dry-run` is a flag for `update`, not a separate command. `self-update` MUST NOT collide with or alias `update` (which updates tools).

| Command | Description | Interactive | Modifies System |
|---------|-------------|-------------|-----------------|
| `upp` (no args) | Show status + available updates | No | No |
| `init` | First-run wizard | Yes | Yes (creates config) |
| `update` | Apply updates to selected tools | Yes | Yes |
| `update --dry-run` | Preview updates without executing | No | No |
| `self-update` | Update the upp binary itself | Yes (confirm) | Yes (replaces binary) |
| `check` | Check for available updates | No | No |
| `list` | List installed/detected tools | No | No |
| `export` | Export config to stdout/file | No | No |
| `import <file>` | Import config | Yes (confirm replace) | Yes (replaces config) |

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| No args | User runs `upp` | Execution | Shows status and available updates (read-only) |
| No args + `--ci` | User runs `upp --ci` | Execution | Same as `upp`, output formatted for CI |
| `update` | User runs `upp update` | Execution | Interactive updates with confirmations |
| `update --dry-run` | User runs `upp update --dry-run` | Execution | Shows what would be updated, no changes |
| `update --ci` | User runs `upp update --ci` | Execution | Non-interactive updates, exit non-zero on failure |
| `self-update` | User runs `upp self-update` | Execution | Checks release, verifies, prompts, replaces binary |
| Unknown command | User runs `upp foo` | Execution | Error with usage hint, exit 1 |
| `--help` | User runs `upp --help` | Execution | Usage text displayed, exit 0 |

(Previously: the command set was `init`, `update`, `check`, `list`, `export`, `import` — no self-update command existed.)

### Requirement: `upp check`

`upp check` MUST query each enabled tool for updates and display summary. No changes made. When `settings.check_self_update` is enabled, `upp check` (and bare `upp`) MUST append the self-update hint after the summary; otherwise it MUST NOT make any self-update network call.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Updates available | 3 tools have updates | `upp check` | Lists 3 with versions, summary count |
| All current | No updates | `upp check` | Shows "all up to date" |
| Self-update hint | `check_self_update=true`, newer release cached | `upp check` | Hint line after summary, exit code unchanged |
| Hint disabled | `check_self_update=false` (default) | `upp check` | No hint, zero self-update network calls |

(Previously: `upp check` had no self-update hint behavior.)

## ADDED Requirements

### Requirement: Self-Update Flag Semantics

`upp self-update` MUST accept no flags in v1. Any unknown flag MUST produce the default cobra rejection (error + usage, non-zero exit). Persistent flags: `--ci` MUST deny the update (see Confirmation Gate); `--only`/`--skip` MUST be ignored (tool filters — documented in `self-update --help`); `--quiet` MUST NOT suppress the confirm prompt or deny message. The detection hint is never part of `self-update` output (it lives in `check`/bare `upp` only). Help MUST show Short text "Update the upp binary itself".

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Unknown flag | `upp self-update --yes` | Execution | Error + usage, exit non-zero |
| `--only` ignored | `upp self-update --only brew` | Execution | Flag ignored, normal self-update flow |
| `--ci` | `upp self-update --ci` | Execution | Deny message, exit non-zero |
| `--quiet` prompt | `upp self-update --quiet` | TTY prompt | Confirm prompt still shown |
