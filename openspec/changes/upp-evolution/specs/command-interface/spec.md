# Command Interface Specification

## Purpose

Define CLI commands, flags, and behavior. All commands share: `--quiet`, `--ci`, `--only`, `--skip` flags.

## Requirements

### Requirement: Command Structure

The system MUST support: `init`, `update`, `check`, `list`, `export`, `import`. Running `upp` with no arguments shows status and available updates (read-only, like `check`). Running `upp update` applies updates. `--dry-run` is a flag for `update`, not a separate command.

| Command | Description | Interactive | Modifies System |
|---------|-------------|-------------|-----------------|
| `upp` (no args) | Show status + available updates | No | No |
| `init` | First-run wizard | Yes | Yes (creates config) |
| `update` | Apply updates to selected tools | Yes | Yes |
| `update --dry-run` | Preview updates without executing | No | No |
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
| Unknown command | User runs `upp foo` | Execution | Error with usage hint, exit 1 |
| `--help` | User runs `upp --help` | Execution | Usage text displayed, exit 0 |

### Requirement: Global Flags

`--quiet` MUST reduce output (fewer details, keep summary). `--ci` MUST disable prompts (non-interactive, exit non-zero on failure).

`--only` and `--skip` accept comma-separated tool names for filtering:
- `--only` processes ONLY the listed tools (takes precedence over `--skip`)
- `--skip` processes ALL enabled tools EXCEPT the listed ones
- If both `--only` and `--skip` are provided, `--only` wins — `--skip` is ignored
- Non-existent tool names in `--only`/`--skip` produce a warning and are ignored
- Tool names are case-insensitive
- `--only` and `--skip` do NOT override the config — they filter the active tool set

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| `--quiet` | Update running | Flag passed | Fewer status lines, summary shown |
| `--ci` | Update running | Flag passed | No prompts, exit non-zero on failure |
| `--only` | Update running | `--only brew,npm` | Only brew and npm processed |
| `--skip` | Update running | `--skip apt,docker` | All except apt and docker processed |
| `--only` + `--skip` | Both provided | `--only brew --skip apt` | Only brew processed (--only wins) |
| Unknown tool in `--only` | `--only brew,nonexistent` | Flag passed | Warning: "nonexistent not found", brew processed |
| Unknown tool in `--skip` | `--skip brew,nonexistent` | Flag passed | Warning: "nonexistent not found", all except brew processed |
| Case insensitive | `--only Brew,NPM` | Flag passed | Matches brew, npm (case-insensitive) |

### Requirement: `upp init`

`upp init` MUST detect installed tools, present them for selection, and generate initial config. MUST be idempotent — running again with existing config prompts before overwriting.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Fresh install | No config | `upp init` | Detects tools, generates config |
| Existing config | Config exists | `upp init` | Prompts: overwrite, merge, or cancel |
| `--ci` mode | No config | `upp init --ci` | Generates config with all detected tools, no prompts |

### Requirement: `upp check`

`upp check` MUST query each enabled tool for updates and display summary. No changes made.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Updates available | 3 tools have updates | `upp check` | Lists 3 with versions, summary count |
| All current | No updates | `upp check` | Shows "all up to date" |

### Requirement: `upp update`

`upp update` MUST process each enabled tool, execute updates, and report results. `--dry-run` shows planned actions without executing. `--only`/`--skip` filter which tools to process.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Normal update | 5 tools enabled | `upp update` | Each tool updated, summary shown |
| Partial failure | Tool 3 of 5 fails | `upp update` | 1-2 updated, 3 failed, 4-5 attempted |
| `--ci` failure | Tool fails in CI | `upp update --ci` | Exit non-zero, summary shows failures |
| Dry run | 3 tools have updates | `upp update --dry-run` | Lists planned actions, no changes |

### Requirement: `upp export` / `upp import`

`upp export` MUST output config to stdout or file (`-o`). `upp import <file>` MUST replace config after TOML validation.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Export | Config exists | `upp export` | TOML to stdout |
| Export to file | Config exists | `upp export -o out.toml` | File written |
| Import | Valid file | `upp import in.toml` | Config replaced |
| Import invalid | Malformed file | `upp import bad.toml` | Error, no changes |
