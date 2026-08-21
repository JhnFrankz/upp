# Delta for command-interface

## REMOVED Requirements

### Requirement: `upp check`

**Reason**: The `upp check` subcommand is removed from the CLI. Its role as the zero-mutation query surface moves to `upp update --dry-run` (`-n`), which becomes the sole way to discover pending updates without executing them.

**Migration**: Scripts and CI that invoked `upp check` MUST switch to `upp update --dry-run`. Quiet/verbose/filter-flag coverage previously exercised through `check` MUST route through `list` and `upp update --dry-run` instead. External callers invoking `upp check` after removal get an unknown-command error with exit status 1.

## MODIFIED Requirements

### Requirement: Command Structure

The system MUST support the following core subcommands: `list`, `update`, `init`, and `self-update`. The system MUST NOT support `check`, `export`, or `import` subcommands; attempting to invoke any of them MUST result in an unknown command error and exit with status 1.

Running `upp` with no arguments (bare invocation) MUST display an informative, non-destructive dashboard and welcome screen showing version and platform information, configured tools status overview, and primary command guidance. Bare invocation MUST NOT run update checks against package managers, MUST NOT apply updates, and MUST NOT execute any destructive actions.

`--dry-run` (with shorthand `-n`) is a flag for `update`, not a separate command. With `check` removed, `upp update --dry-run` is the only read-only/query surface for pending updates: it MUST report what would be updated without executing any change. `self-update` MUST NOT collide with or alias `update` (which updates tools).

| Command | Description | Interactive | Modifies System |
|---------|-------------|-------------|-----------------|
| `upp` (no args) | Show dashboard / welcome screen and guidance | No | No |
| `init` | First-run wizard | Yes | Yes (creates config) |
| `update` | Apply updates to selected tools | Yes | Yes |
| `update -n` / `--dry-run` | Read-only query: preview pending updates without executing | No | No |
| `self-update` | Update the upp binary itself | Yes (confirm) | Yes (replaces binary) |
| `list` | List installed/detected tools | No | No |

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| No args | User runs `upp` | Execution | Shows dashboard/welcome screen with version, tools overview, and command guidance |
| No args + `--ci` | User runs `upp --ci` | Execution | Same dashboard output formatted for non-interactive / CI environment |
| `update` | User runs `upp update` | Execution | Interactive updates with confirmations |
| `update --dry-run` | User runs `upp update --dry-run` | Execution | Shows what would be updated, no changes |
| `update -n` | User runs `upp update -n` | Execution | Behaves identically to `upp update --dry-run`, no changes |
| `update --ci` | User runs `upp update --ci` | Execution | Non-interactive updates, exit non-zero on failure |
| `self-update` | User runs `upp self-update` | Execution | Checks release, verifies, prompts, replaces binary |
| Pruned `check` command | User runs `upp check` | Execution | Error: unknown command "check", exit 1 |
| Pruned `export` command | User runs `upp export` | Execution | Error: unknown command "export", exit 1 |
| Pruned `import` command | User runs `upp import config.toml` | Execution | Error: unknown command "import", exit 1 |
| Unknown command | User runs `upp foo` | Execution | Error with usage hint, exit 1 |
| `--help` | User runs `upp --help` | Execution | Usage text displayed, exit 0 |

(Previously: the command set included a standalone read-only `check` subcommand; `update --dry-run` is now the only query surface.)

### Requirement: Help Output Grouping

`upp --help` and `upp help` MUST group commands into exactly two labeled sections:
1. `Commands`: `list`, `update`
2. `Maintenance`: `init`, `self-update`

The former `Config Commands` group MUST NOT exist. The cobra `completion` built-in MUST be hidden from help output.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Simplified help groups | Root command invoked | `upp --help` | Commands grouped under `Commands` (`list`, `update`) and `Maintenance` (`init`, `self-update`); `Config Commands` is absent |
| Help subcommand | Root command invoked | `upp help` | Same 2-group structure as `upp --help` |
| Completion hidden | Root command built | `upp --help` | `completion` not listed in help output |
| Pruned commands absent | Root command invoked | `upp --help` | Neither `check`, `export`, nor `import` appears in any group |

(Previously: the `Commands` group listed `check`, `list`, `update`.)

### Requirement: Self-Update Flag Semantics

`upp self-update` MUST accept no flags in v1. Any unknown flag MUST produce the default cobra rejection (error + usage, non-zero exit). Persistent flags: `--ci` MUST deny the update (see Confirmation Gate); `--only`/`--skip` MUST be ignored (tool filters — documented in `self-update --help`); `--quiet` MUST NOT suppress the confirm prompt or deny message. Release detection and any self-update network activity happen only within `self-update` itself; no hint or detection output is appended to any other command. Help MUST show Short text "Update the upp binary itself".

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Unknown flag | `upp self-update --yes` | Execution | Error + usage, exit non-zero |
| `--only` ignored | `upp self-update --only brew` | Execution | Flag ignored, normal self-update flow |
| `--ci` | `upp self-update --ci` | Execution | Deny message, exit non-zero |
| `--quiet` prompt | `upp self-update --quiet` | TTY prompt | Confirm prompt still shown |

(Previously: the requirement stated the detection hint was excluded from `self-update` because it lived in `check`/bare `upp`; the hint surface no longer exists and detection is exclusive to `self-update`.)
