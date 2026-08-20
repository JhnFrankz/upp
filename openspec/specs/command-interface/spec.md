# Command Interface Specification

## Purpose

Define CLI commands, flags, and behavior. All commands share: `--quiet` (`-q`), `--verbose` (`-v`), `--ci`, `--only`, `--skip` flags.

## Requirements

### Requirement: Command Structure

The system MUST support the following core subcommands: `check`, `list`, `update`, `init`, and `self-update`. The system MUST NOT support `export` or `import` subcommands; attempting to invoke `upp export` or `upp import` MUST result in an unknown command error and exit with status 1.

Running `upp` with no arguments (bare invocation) MUST display an informative, non-destructive dashboard and welcome screen showing version and platform information, configured tools status overview, and primary command guidance. Bare invocation MUST NOT run update checks against package managers, MUST NOT apply updates, and MUST NOT execute any destructive actions.

`--dry-run` (with shorthand `-n`) is a flag for `update`, not a separate command. `self-update` MUST NOT collide with or alias `update` (which updates tools).

| Command | Description | Interactive | Modifies System |
|---------|-------------|-------------|-----------------|
| `upp` (no args) | Show dashboard / welcome screen and guidance | No | No |
| `init` | First-run wizard | Yes | Yes (creates config) |
| `update` | Apply updates to selected tools | Yes | Yes |
| `update -n` / `--dry-run` | Preview updates without executing | No | No |
| `self-update` | Update the upp binary itself | Yes (confirm) | Yes (replaces binary) |
| `check` | Check for available updates | No | No |
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
| Pruned `export` command | User runs `upp export` | Execution | Error: unknown command "export", exit 1 |
| Pruned `import` command | User runs `upp import config.toml` | Execution | Error: unknown command "import", exit 1 |
| Unknown command | User runs `upp foo` | Execution | Error with usage hint, exit 1 |
| `--help` | User runs `upp --help` | Execution | Usage text displayed, exit 0 |

(Previously: the command set included `export` and `import`, and running `upp` with no args was a duplicate of `upp check`.)

### Requirement: Global Flags

The system MUST support the following global persistent flags available across all commands:
- `--quiet` (shorthand `-q`): MUST reduce output to essential status only (fewer details, keep summary).
- `--verbose` (shorthand `-v`): MUST enable diagnostic logging, emitting detailed adapter subprocess stderr output when tool execution or update fails.
- `--ci`: MUST disable prompts (non-interactive execution, exit non-zero on failure).
- `--only` and `--skip`: accept comma-separated tool names for filtering active tools.

Filtering rules for `--only` and `--skip`:
- `--only` processes ONLY the listed tools (takes precedence over `--skip`)
- `--skip` processes ALL enabled tools EXCEPT the listed ones
- If both `--only` and `--skip` are provided, `--only` wins — `--skip` is ignored
- Non-existent tool names in `--only`/`--skip` produce a warning and are ignored
- Tool names are case-insensitive
- `--only` and `--skip` do NOT override the config — they filter the active tool set

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| `--quiet` | Update running | Flag passed | Fewer status lines, summary shown |
| `-q` shorthand | Check or update running | `-q` flag passed | Output is identical to `--quiet` |
| `--verbose` on failure | Tool adapter fails | `--verbose` flag passed | Adapter subprocess stderr diagnostics are rendered inline |
| `-v` shorthand | Tool adapter fails | `-v` flag passed | Output is identical to `--verbose` |
| `--ci` | Update running | Flag passed | No prompts, exit non-zero on failure |
| `--only` | Update running | `--only brew,npm` | Only brew and npm processed |
| `--skip` | Update running | `--skip apt,docker` | All except apt and docker processed |
| `--only` + `--skip` | Both provided | `--only brew --skip apt` | Only brew processed (--only wins) |
| Unknown tool in `--only` | `--only brew,nonexistent` | Flag passed | Warning: "nonexistent not found", brew processed |
| Unknown tool in `--skip` | `--skip brew,nonexistent` | Flag passed | Warning: "nonexistent not found", all except brew processed |
| Case insensitive | `--only Brew,NPM` | Flag passed | Matches brew, npm (case-insensitive) |

(Previously: `--quiet` lacked the `-q` shorthand, and `--verbose` / `-v` did not exist.)

### Requirement: `upp init`

`upp init` MUST detect installed tools, present them for selection, and generate initial config. MUST be idempotent — running again with existing config prompts before overwriting.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Fresh install | No config | `upp init` | Detects tools, generates config |
| Existing config | Config exists | `upp init` | Prompts: overwrite, merge, or cancel |
| `--ci` mode | No config | `upp init --ci` | Generates config with all detected tools, no prompts |

### Requirement: `upp check`

`upp check` MUST query each enabled tool for updates and display summary. No changes made. When `settings.check_self_update` is enabled, `upp check` (and bare `upp`) MUST append the self-update hint after the summary; otherwise it MUST NOT make any self-update network call.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Updates available | 3 tools have updates | `upp check` | Lists 3 with versions, summary count |
| All current | No updates | `upp check` | Shows "all up to date" |
| Self-update hint | `check_self_update=true`, newer release cached | `upp check` | Hint line after summary, exit code unchanged |
| Hint disabled | `check_self_update=false` (default) | `upp check` | No hint, zero self-update network calls |

(Previously: `upp check` had no self-update hint behavior.)

### Requirement: Self-Update Flag Semantics

`upp self-update` MUST accept no flags in v1. Any unknown flag MUST produce the default cobra rejection (error + usage, non-zero exit). Persistent flags: `--ci` MUST deny the update (see Confirmation Gate); `--only`/`--skip` MUST be ignored (tool filters — documented in `self-update --help`); `--quiet` MUST NOT suppress the confirm prompt or deny message. The detection hint is never part of `self-update` output (it lives in `check`/bare `upp` only). Help MUST show Short text "Update the upp binary itself".

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Unknown flag | `upp self-update --yes` | Execution | Error + usage, exit non-zero |
| `--only` ignored | `upp self-update --only brew` | Execution | Flag ignored, normal self-update flow |
| `--ci` | `upp self-update --ci` | Execution | Deny message, exit non-zero |
| `--quiet` prompt | `upp self-update --quiet` | TTY prompt | Confirm prompt still shown |

### Requirement: `upp update`

`upp update` MUST process each enabled tool, execute updates, and report results. `--dry-run` (with single-letter shorthand `-n`) MUST show planned actions without executing. `--only`/`--skip` filter which tools to process.

In TTY runs (stdin is a TTY, and `--ci`, `--quiet`, and `--dry-run` are not set), `upp update` MUST render the interactive tool selection over the `--only`/`--skip`-filtered pending set before executing; the user's selection MUST narrow the update set further. Flag semantics MUST NOT change: `--only`/`--skip` filter exactly as before, and `--dry-run` MUST remain non-interactive (no selector rendered).

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Normal update | 5 tools enabled | `upp update` | Each tool updated, summary shown |
| Partial failure | Tool 3 of 5 fails | `upp update` | 1-2 updated, 3 failed, 4-5 attempted |
| `--ci` failure | Tool fails in CI | `upp update --ci` | Exit non-zero, summary shows failures |
| Dry run full flag | 3 tools have updates | `upp update --dry-run` | Lists planned actions, no changes |
| Dry run short flag | 3 tools have updates | `upp update -n` | Behaves identically to `--dry-run`, no changes |
| Selector over filtered set | TTY, `--only brew,npm`, 2 pending | `upp update --only brew,npm` | Selector lists only brew and npm; other tools not shown |
| Selection narrows further | TTY, selector shows 3 pre-checked | User deselects 1 tool | Only the 2 selected tools updated; summary counts match selection |
| Dry-run non-interactive | TTY, `--dry-run`, pending updates | `upp update --dry-run` | No selector; planned actions listed, no changes |

(Previously: `--dry-run` had no single-letter shorthand `-n`; TTY runs processed the filtered set directly with no interactive selection step.)

### Requirement: Help Output Grouping

`upp --help` and `upp help` MUST group commands into exactly two labeled sections:
1. `Commands`: `check`, `list`, `update`
2. `Maintenance`: `init`, `self-update`

The former `Config Commands` group MUST NOT exist. The cobra `completion` built-in MUST be hidden from help output.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Simplified help groups | Root command invoked | `upp --help` | Commands grouped under `Commands` (`check`, `list`, `update`) and `Maintenance` (`init`, `self-update`); `Config Commands` is absent |
| Help subcommand | Root command invoked | `upp help` | Same 2-group structure as `upp --help` |
| Completion hidden | Root command built | `upp --help` | `completion` not listed in help output |
| Pruned commands absent | Root command invoked | `upp --help` | Neither `export` nor `import` appears in any group |

(Previously: commands were grouped into 3 sections: "Tool Commands", "Config Commands", and "Maintenance".)
