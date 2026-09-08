# Command Interface Specification

## Purpose

Define CLI commands, flags, and behavior. All commands share: `--quiet` (`-q`), `--verbose` (`-v`), `--ci`, `--only` flags.

## Requirements

### Requirement: Command Structure

The system MUST support the following core subcommands: `list`, `update`, `init`, `self-update`, and `uninstall`. The system MUST NOT support `check`, `export`, or `import` subcommands; attempting to invoke any of them MUST result in an unknown command error and exit with status 1.

Running `upp` with no arguments (bare invocation) MUST display an informative, non-destructive dashboard and welcome screen showing version and platform information, configured tools status overview, and primary command guidance. Bare invocation MUST NOT run update checks against package managers, MUST NOT apply updates, and MUST NOT execute any destructive actions.

`--dry-run` (with shorthand `-n`) is a flag for `update`, not a separate command. With `check` removed, `upp update --dry-run` is the only read-only/query surface for pending updates: it MUST report what would be updated without executing any change. `self-update` MUST NOT collide with or alias `update` (which updates tools).

| Command | Description | Interactive | Modifies System |
|---------|-------------|-------------|-----------------|
| `upp` (no args) | Show dashboard / welcome screen and guidance | No | No |
| `init` | First-run wizard | Yes | Yes (creates config) |
| `update` | Apply updates to selected tools | Yes | Yes |
| `update -n` / `--dry-run` | Read-only query: preview pending updates without executing | No | No |
| `self-update` | Update the upp binary itself | Yes (confirm) | Yes (replaces binary) |
| `uninstall` | Uninstall upp and remove all binaries, configuration, and caches | No | Yes (deletes binary/config/cache) |
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
| `uninstall` | User runs `upp uninstall` | Execution | Removes binary, backups, config, and cache under Zero-Sudo policy |
| `uninstall --dry-run` | User runs `upp uninstall --dry-run` | Execution | Lists planned deletions without modifying disk |
| Pruned `check` command | User runs `upp check` | Execution | Error: unknown command "check", exit 1 |
| Pruned `export` command | User runs `upp export` | Execution | Error: unknown command "export", exit 1 |
| Pruned `import` command | User runs `upp import config.toml` | Execution | Error: unknown command "import", exit 1 |
| Unknown command | User runs `upp foo` | Execution | Error with usage hint, exit 1 |
| `--help` | User runs `upp --help` | Execution | Usage text displayed, exit 0 |

(Previously: the command set included a standalone read-only `check` subcommand; `update --dry-run` is now the only query surface.)

### Requirement: Global Flags

The system MUST support exactly the following global persistent flags available across all commands:
- `--quiet` (shorthand `-q`): MUST reduce output to essential status only (fewer details, keep summary).
- `--verbose` (shorthand `-v`): MUST enable diagnostic logging, emitting detailed adapter subprocess stderr output when tool execution or update fails.
- `--ci`: MUST disable prompts (non-interactive execution, exit non-zero on failure).
- `--only`: accepts comma-separated tool names for filtering active tools.

The system MUST NOT register any other global flag. `--skip`, `--manager`, and `--update-group` MUST NOT exist; attempting to use any of them MUST produce the default unknown-flag rejection (error + usage, non-zero exit).

Filtering rules for `--only`:
- `--only` processes ONLY the listed tools
- Non-existent tool names in `--only` produce a warning and are ignored
- Tool names are case-insensitive
- `--only` does NOT override the config — it filters the active tool set

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| `--quiet` | Update running | Flag passed | Fewer status lines, summary shown |
| `-q` shorthand | Check or update running | `-q` flag passed | Output is identical to `--quiet` |
| `--verbose` on failure | Tool adapter fails | `--verbose` flag passed | Adapter subprocess stderr diagnostics are rendered inline |
| `-v` shorthand | Tool adapter fails | `-v` flag passed | Output is identical to `--verbose` |
| `--ci` | Update running | Flag passed | No prompts, exit non-zero on failure |
| `--only` | Update running | `--only brew,npm` | Only brew and npm processed |
| Unknown tool in `--only` | `--only brew,nonexistent` | Flag passed | Warning: "nonexistent not found", brew processed |
| Case insensitive | `--only Brew,NPM` | Flag passed | Matches brew, npm (case-insensitive) |
| `--skip` rejected | Update running | `upp update --skip apt` | Error: unknown flag, usage hint, exit non-zero |
| `--manager` rejected | Update running | `upp update --manager apt` | Error: unknown flag, usage hint, exit non-zero |

(Previously: `--only` and `--skip` were both supported as inverse filters with `--only` winning on conflict; `--skip` is now removed as a duplicate of `--only` and all unknown flags are rejected by cobra. Earlier still: `--quiet` lacked the `-q` shorthand, and `--verbose` / `-v` did not exist.)

### Requirement: `upp init`

`upp init` MUST detect installed tools, present them for selection, and generate initial config. MUST be idempotent — running again with existing config prompts before overwriting.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Fresh install | No config | `upp init` | Detects tools, generates config |
| Existing config | Config exists | `upp init` | Prompts: overwrite, merge, or cancel |
| `--ci` mode | No config | `upp init --ci` | Generates config with all detected tools, no prompts |

### Requirement: Self-Update Flag Semantics

`upp self-update` MUST accept no flags in v1. Any unknown flag MUST produce the default cobra rejection (error + usage, non-zero exit). Persistent flags: `--ci` MUST deny the update (see Confirmation Gate); `--only` MUST be ignored (tool filter — documented in `self-update --help`); `--quiet` MUST NOT suppress the confirm prompt or deny message. Release detection and any self-update network activity happen only within `self-update` itself; no hint or detection output is appended to any other command. Help MUST show Short text "Update the upp binary itself".

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Unknown flag | `upp self-update --yes` | Execution | Error + usage, exit non-zero |
| `--only` ignored | `upp self-update --only brew` | Execution | Flag ignored, normal self-update flow |
| `--ci` | `upp self-update --ci` | Execution | Deny message, exit non-zero |
| `--quiet` prompt | `upp self-update --quiet` | TTY prompt | Confirm prompt still shown |

(Previously: the requirement stated the detection hint was excluded from `self-update` because it lived in `check`/bare `upp`; the hint surface no longer exists and detection is exclusive to `self-update`.)

### Requirement: `upp update`

`upp update` MUST process each enabled tool, execute updates, and report results. The command MUST delegate tool adapter resolution, concurrent version checking, and update plan formulation to `engine.Engine`. By default (bare `upp update`), the command MUST execute manager-group bulk package updates for all owned tools grouped under their resolving package managers, alongside standalone tool updates. `--dry-run` (with shorthand `-n`) MUST show planned update actions—including planned manager group package updates and standalone tool updates—without executing any changes. `--only` MUST filter which tools to process.

In TTY runs (where stdin is a TTY, and `--ci`, `--quiet`, and `--dry-run` are not set), `upp update` MUST render the interactive tool selection over the `--only`-filtered pending set before executing; users MUST be able to toggle individual owned tools within manager groups as well as standalone tools. The user's selection MUST narrow the update set further. Flag semantics MUST NOT change: `--only` filters the candidate tools prior to presentation, and `--dry-run` MUST remain strictly non-interactive (no selector rendered).

Execution across tools and manager groups MUST maintain per-tool error isolation, ensuring that failures in individual package updates or standalone adapters do not halt execution of remaining tools. In `--ci` mode, any failure or unconfirmed elevated risk MUST cause the command to exit with a non-zero status after completing all non-dependent updates. `--manager` and `--update-group` MUST NOT exist as flags; supplying either MUST produce the default unknown-flag rejection (error + usage, non-zero exit).

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Normal default update | 5 tools enabled (apt owning gh/docker, plus npm, bun, nvm) | `upp update` | gh and docker updated via apt manager-group package updates, standalone tools updated, summary shown |
| Per-tool isolated failure | Tool 3 of 5 fails (e.g. gh in apt group) | `upp update` | Tools 1-2 updated, gh fails with isolated error, docker and standalone tools 4-5 attempted and updated |
| `--ci` failure exit | Tool fails during update in CI | `upp update --ci` | Non-dependent tools complete, exit non-zero, summary shows failures |
| `--ci` elevated risk fail-closed | Sudo package update required in CI | `upp update --ci` | Fails closed non-zero immediately without prompt (`EnforceRisk: true`) |
| Dry run full flag | 3 tools have updates (2 in brew group, 1 standalone) | `upp update --dry-run` | Lists planned actions for brew group packages and standalone tools, no changes made |
| Dry run short flag | 3 tools have updates | `upp update -n` | Behaves identically to `upp update --dry-run`, no changes made |
| Selector over filtered set | TTY, `--only brew,gh,npm` where brew owns gh | `upp update --only brew,gh,npm` | Selector lists brew group containing gh and standalone npm; other tools excluded |
| Granular selection in manager group | TTY, selector shows apt group with gh and docker pre-checked | User deselects docker | Only gh is updated via apt package update; docker is skipped; summary counts match selection |
| Dry-run non-interactive | TTY, `--dry-run`, pending updates | `upp update --dry-run` | No selector rendered; planned actions listed, no changes made |
| `--manager` rejected | Update running | `upp update --manager apt` | Error: unknown flag "manager", usage hint, exit non-zero |
| `--update-group` rejected | Update running | `upp update --update-group brew` | Error: unknown flag "update-group", usage hint, exit non-zero |
| Engine delegation seam | Update invoked | `upp update` | Resolves adapters, executes concurrent pre-checks, and builds update plan via `engine.Engine` |

(Previously: `upp update` directly coordinated discovery, concurrent check execution, and planning within `internal/cli` loops; it now delegates resolution, checking, and planning to `engine.Engine`.)

### Requirement: Orchestration Delegation

The `list` and `update` commands MUST delegate tool adapter discovery, resolution, check dispatching, and update plan formulation to `engine.Engine`.

1. `upp list` MUST invoke `engine.Resolve(filter)` to obtain active, enabled adapters for the current platform and render the resulting groups via `output.GroupByOwner`.
2. `upp update` MUST invoke `engine.Resolve(filter)` for tool discovery, `engine.Check(ctx, adapters, onProgress)` for concurrent version checking, and `engine.Plan(outcomes, filter)` for update action formulation.
3. The presentation layer (`internal/cli`) MUST consume pure domain models (`CheckProgress`, `CheckOutcome`, `UpdatePlan`) emitted by the engine and map them to presentation renderers (`output.CheckBoard`, `output.Renderer`, `output.CheckboxSelector`) without altering command flags (`--quiet`/`-q`, `--verbose`/`-v`, `--ci`, `--only`, `--dry-run`/`-n`), visual layouts, exit codes, or interactive selection workflows.
4. Tool execution and security confirmation (`security.ConfirmAction`) MUST remain in the CLI presentation layer.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| List command delegation | Host platform with enabled tools | `upp list` | Invokes `engine.Resolve(filter)` to obtain active adapters and renders table via `output.GroupByOwner` |
| Update command delegation | Host platform with enabled tools | `upp update` | Invokes `engine.Resolve()`, `engine.Check()`, and `engine.Plan()`, bridging progress to `output.CheckBoard` |
| Flag preservation | Command passed `-q`, `-v`, `--ci`, `--only` | Command execution | Flags parsed by CLI and passed as domain `Filter` to engine; visual flags applied to CLI renderer |
| Exit code preservation | Update encounters failed check or unconfirmed CI risk | Command execution | Command exits with non-zero status identical to previous behavior |
| Headless engine compatibility | CLI presentation disabled or piped stdout | Command execution | Engine executes headless resolution, checking, and planning without presentation dependencies |

### Requirement: Help Output Grouping

`upp --help` and `upp help` MUST group commands into exactly two labeled sections:
1. `Commands`: `list`, `update`
2. `Maintenance`: `init`, `self-update`, `uninstall`

The former `Config Commands` group MUST NOT exist. The cobra `completion` built-in MUST be hidden from help output.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Simplified help groups | Root command invoked | `upp --help` | Commands grouped under `Commands` (`list`, `update`) and `Maintenance` (`init`, `self-update`, `uninstall`); `Config Commands` is absent |
| Help subcommand | Root command invoked | `upp help` | Same 2-group structure as `upp --help` |
| Completion hidden | Root command built | `upp --help` | `completion` not listed in help output |
| Pruned commands absent | Root command invoked | `upp --help` | Neither `check`, `export`, nor `import` appears in any group |

(Previously: the `Commands` group listed `check`, `list`, `update`.)
