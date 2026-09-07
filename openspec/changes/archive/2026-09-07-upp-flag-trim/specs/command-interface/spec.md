# Delta for command-interface

<!-- Archive note: after merging, update the Purpose paragraph line "All commands share: ..." to drop `--skip` (list `--quiet` (`-q`), `--verbose` (`-v`), `--ci`, `--only`). Purpose prose is outside requirement blocks and cannot be delta-edited. -->

## MODIFIED Requirements

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

(Previously: `--only` and `--skip` were both supported as inverse filters with `--only` winning on conflict; `--skip` is now removed as a duplicate of `--only` and all unknown flags are rejected by cobra.)

### Requirement: Self-Update Flag Semantics

`upp self-update` MUST accept no flags in v1. Any unknown flag MUST produce the default cobra rejection (error + usage, non-zero exit). Persistent flags: `--ci` MUST deny the update (see Confirmation Gate); `--only` MUST be ignored (tool filter — documented in `self-update --help`); `--quiet` MUST NOT suppress the confirm prompt or deny message. Release detection and any self-update network activity happen only within `self-update` itself; no hint or detection output is appended to any other command. Help MUST show Short text "Update the upp binary itself".

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Unknown flag | `upp self-update --yes` | Execution | Error + usage, exit non-zero |
| `--only` ignored | `upp self-update --only brew` | Execution | Flag ignored, normal self-update flow |
| `--ci` | `upp self-update --ci` | Execution | Deny message, exit non-zero |
| `--quiet` prompt | `upp self-update --quiet` | TTY prompt | Confirm prompt still shown |

(Previously: `--only`/`--skip` were both documented as ignored tool filters in `self-update --help`; `--skip` no longer exists, so only `--only` is documented as ignored.)

### Requirement: `upp update`

`upp update` MUST process each enabled tool, execute updates, and report results. By default (bare `upp update`), the command MUST execute manager-group bulk package updates for all owned tools grouped under their resolving package managers, alongside standalone tool updates. `--dry-run` (with shorthand `-n`) MUST show planned update actions—including planned manager group package updates and standalone tool updates—without executing any changes. `--only` MUST filter which tools to process.

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

(Previously: bare `upp update` executed standard per-tool adapter updates without manager-group bulk package updates; group bulk updates were strictly opt-in via `--manager` or `--update-group`, and `--skip` filtered tools inversely. The default delegated path is now the only update path and all three flags are removed.)
