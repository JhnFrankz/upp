# UX Patterns Specification

## Purpose

Output formatting, interactive prompts, summary display, and cross-platform terminal behavior.

## Requirements

### Requirement: Default Interactive Mode

The system MUST be interactive by default. Official tool updates MUST NOT prompt (security-model takes precedence). Every custom tool update MUST pass the security-model risk matrix before execution, prompting only when the matrix requires confirmation. `--ci` MUST suppress prompts; `--quiet` MUST NOT suppress prompts.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Official default run | Tool is `brew` (official), no flags | Update requested | Proceeds without prompt |
| Custom high-risk run | `mytool` (custom) uses `sudo`, no flags | Update requested | Prompt: "Run `sudo ...` for mytool? [y/N]" |
| `--ci` low-risk run | `--ci` flag, custom low-risk | Update requested | No prompt, auto-proceed |
| `--ci` high-risk run | `--ci` flag, custom high-risk | Update requested | Fails non-zero (per security-model) |
| `--quiet` run | `--quiet` flag, custom medium-risk | Update requested | Prompt still shown (quiet affects detail, not prompts) |

(Previously: every tool update — including official — prompted "Update brew? [Y/n]" unless `--ci`; contradicted security-model's official no-prompt rule.)

### Requirement: Output Language

The system MUST output in English only. Output language is NOT configurable: the `[settings]` section MUST NOT carry an output-language key (see config-system), and a stray `language` key in an existing config file is ignored.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Any command | Any config state | Any command | English output |

### Requirement: Color and Emoji

The system MUST use ANSI colors and emoji for status indicators in terminal output. Output MUST degrade gracefully when color is not supported (pipe, dumb terminal).

Status indicators:

- ✅ Updated successfully
- ⏭️ Skipped (not installed)
- ❌ Failed
- ⬆️ Update available
- ✔️ Already current

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Color terminal | TTY detected | Output | Colors and emoji displayed |
| Pipe output | `upp list \| less` | Output | No ANSI codes, no emoji (or plain-text fallback) |

### Requirement: Bare Dashboard Welcome Screen

Running `upp` with no arguments (bare invocation) MUST render an educational, non-destructive dashboard and welcome screen. The dashboard MUST display:
1. Header banner with `upp` version and host platform (OS/architecture).
2. Configured tools overview showing the number of enabled tools and total configured/platform tools.
3. Quick-reference workflow guidance listing the primary commands:
   - `upp check`: Check for available updates without making changes.
   - `upp update`: Apply updates to all enabled tools (supports `-n` / `--dry-run`).
   - `upp list`: Inspect installed and detected tool status.
   - `upp --help`: View full command options and help.

Bare invocation MUST be strictly informative and read-only. It MUST NOT make network calls to package managers, MUST NOT apply updates, and MUST exit with status 0 upon rendering.

When `--quiet` (or `-q`) is supplied to bare `upp`, the decorative banner and guidance MUST be suppressed. If no configuration file exists, bare `upp` MUST render a guidance message directing the user to run `upp init`.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Interactive dashboard | Valid configuration with enabled tools | `upp` (no args) | Renders header banner, tools overview count, and quick-reference command guide; exit 0 |
| Non-TTY / Pipe | Stdout is redirected to a pipe | `upp \| cat` | Renders plain text dashboard without ANSI escape codes or colored styling |
| Quiet mode | Bare invocation with `-q` or `--quiet` | `upp -q` | Banner and guidance suppressed, minimal output emitted |
| Missing config | No `config.toml` exists | `upp` (no args) | Displays notice and directs user to run `upp init` |

### Requirement: Summary Report

Every `update` (including `--dry-run`) and `check` run MUST end with a summary showing:

- Count of tools updated / checked / skipped / failed
- List of tools in each category
- Overall status message

`check` summaries MUST count skipped tools explicitly ("N up to date, M skipped"). The summary MUST NOT print "All tools up to date." when any enabled tool was skipped or unchecked. A `--dry-run` summary MUST NOT print "All clean!" when any update is pending; pending updates MUST be reported explicitly. The tool list and status lines in the summary report MUST follow a 100% deterministic order matching canonical tool discovery order, unaffected by out-of-order concurrent completion during execution.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| All succeed | 5/5 updated | Summary | "✅ 5 updated, 0 failed. All clean!" |
| Partial fail | 3/5 updated, 2 failed | Summary | "✅ 3 updated, ❌ 2 failed. Review errors above." |
| No tools | All skipped | Summary | "⏭️ All tools not installed. Nothing to do." |
| Check with skips | 8 current, 2 enabled tools skipped (not installed) | `upp check` | Summary counts skipped explicitly ("8 up to date, 2 skipped"); never "All tools up to date." |
| Dry-run pending | 3 updates pending, 7 current | `upp update --dry-run` | Summary reports "3 would update"; never pairs "All clean!" with pending updates |
| Concurrent check deterministic order | Tools complete out-of-order across concurrent workers | `upp check` finishes | Summary report lists tools strictly in canonical tool discovery order |

(Previously: check counted only available/current/failed, silently dropping skipped tools, and could print "All tools up to date." while enabled tools were unchecked; dry-run paired "All clean!" with pending updates; summary ordering under concurrency was not specified.)

### Requirement: `--quiet` Verbosity

`--quiet` MUST reduce per-tool output to essential status only (one line per tool). Summary MUST still be displayed.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Quiet mode | `--quiet` | Updating 3 tools | 3 status lines + summary |
| Normal mode | No `--quiet` | Updating 3 tools | Detailed per-tool output + summary |

### Requirement: Error Display

Errors MUST be displayed inline with the failing tool, NOT as a separate error stream. Errors MUST include: tool name, operation, and brief cause.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Tool failure | `brew update` fails | Summary | "❌ brew: update failed (network error)" |
| Config error | Invalid TOML | Config load | "❌ Config error at line 5: expected `"`, found `}`" |

### Requirement: Verbose Error Diagnostics Rendering

When the global `--verbose` (or `-v`) flag is enabled, adapter execution failures during `check` or `update` MUST render detailed diagnostic information inline beneath the failed tool entry. Diagnostic information MUST include the captured subprocess stderr / error output from the failing tool adapter.

When `--verbose` / `-v` is omitted (default), adapter failure output MUST remain concise, showing only the tool name and short failure cause, suppressing raw subprocess stderr to keep terminal output clean.

In `--quiet` mode, verbose error diagnostics MUST be suppressed.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Verbose failure diagnostics | Adapter fails with subprocess stderr "lock frontend held by another process" | `upp check -v` or `upp update --verbose` | Failing tool line is followed by indented subprocess stderr output |
| Short flag `-v` diagnostics | Adapter fails during update | `upp update -v` | Behaves identically to `--verbose`, displaying detailed stderr diagnostics |
| Default non-verbose failure | Adapter fails during check | `upp check` (no `-v`) | Only concise inline error line displayed; raw stderr suppressed |
| Success with verbose | All tools succeed | `upp check -v` | Standard success output rendered without debug noise |
| Quiet takes precedence | Adapter fails in quiet mode with `-v` | `upp check -q -v` | Detailed subprocess stderr is suppressed in favor of quiet output |

### Requirement: Progress Indication

For long-running operations, the system SHOULD show per-operation progress labeled with the operation: "Checking X/Y" for `check`; "Updating X/Y" only for `update`. A read-only operation MUST NOT print "Updating". During concurrent tool checks, progress rendering MUST be synchronized across worker threads so that "Checking X/Y" output lines are emitted atomically and never interleave or corrupt terminal output.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Multi-tool check | 10 tools enabled | `upp check` | Progress shown: "Checking 3/10: brew" |
| Multi-tool update | 10 tools enabled | `upp update` | Progress shown: "Updating 3/10: brew" |
| Single tool | 1 tool enabled | Update running | No progress indicator needed |
| Concurrent check progress | Multiple tools checked concurrently | `upp check` runs worker pool | Progress updates are rendered atomically without line interleaving or corrupted output |

(Previously: progress indicated bare `upp` printed "Checking X/Y"; bare `upp` is now a non-checking dashboard.)

### Requirement: Self-Update Detection Hint

When `settings.check_self_update` is enabled and a newer release is known, `check`/bare `upp` MUST append exactly one hint line after the summary: `⬆️ upp v{latest} available (current {current}) — run "upp self-update"`. The hint MUST NOT change the exit code. The hint MUST be omitted when `--quiet` is set, when the check failed or was offline (silent, no error), or when no newer release exists.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Newer release | `check_self_update=true`, v0.1.1 vs v0.1.0 | `upp check` | One hint line after summary, exit unchanged |
| Offline | `check_self_update=true`, API unreachable | `upp check` | No hint, no error, exit unchanged |
| Quiet | `check_self_update=true`, `--quiet` | `upp check` | No hint line |
| Up to date | `check_self_update=true`, latest = current | `upp check` | No hint line |

### Requirement: Self-Update Confirmation Prompt

`upp self-update` MUST show an English confirmation before replacing the binary, including current and target versions and the resolved binary path. Non-TTY stdin or `--ci` MUST print a clear English deny message and exit non-zero — never hang, auto-proceed, or silently skip.
(Previously: the requirement said "localized (en/es)"; output is English-only since the language drop — stale spec text corrected, no behavior change.)

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| TTY prompt | TTY, update available | `upp self-update` | English prompt with versions + path, waits for y/N |
| User declines | TTY, user answers n | Prompt | No changes, exit 0 |
| Non-TTY | stdin is not a TTY | `upp self-update` | Clear deny message, exit non-zero |
| `--ci` | `upp self-update --ci` | Execution | Clear deny message, exit non-zero |

### Requirement: List Table Output

`upp list` MUST render a table whose columns are labeled to match their data and MUST include the tool ID in its own column. The ID column MUST show the IDs used by `--only`/`--skip` and config (`apt`, `brew`, ...), so table rows map to filter names.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Correct columns | 10 tools detected | `upp list` | Header `ID | Name | Status | Version`; each row's ID usable with `--only`/`--skip` |
| Filter round-trip | Row shows ID `apt` | `upp check --only apt` | `apt` processed (row ID matches filter name) |
