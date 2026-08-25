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

### Requirement: Live Check Board

In TTY interactive `upp update` runs, the pre-check board MUST render one stable line per filtered tool, laid out grouped under per-manager headers in canonical discovery order before any result arrives. Manager headers render first, then their owned tools, then standalone tools. An owned tool MUST NOT appear as a top-level line separate from its manager group. Per-tool completion flip, up-to-date visibility, failed-check ✗ behavior, atomic concurrent rendering, the settled-board gating of the selector, and non-color fallback MUST remain unchanged. Grouping MUST NOT reorder stable board lines or alter completion ordering.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Board renders grouped | TTY, Linux, apt+gh+docker | `upp update` pre-check starts | apt header, then gh+docker child lines, then standalone tools; one stable line per tool |
| Owned tool in group | Platform Linux, docker owned by apt | Pre-check renders | docker line appears beneath apt header, not top-level |
| Per-tool completion flip | brew finishes first, v1.2 → v1.3 | brew check completes | Only brew's line flips to ✓ showing `1.2 → 1.3`; other lines unchanged |
| Settled board gates selector | Board settled, 2 of 5 tools pending | Pre-check ends | CheckboxSelector lists only the 2 pending tools; current and failed excluded |
| Atomic concurrent rendering | Worker pool completes checks concurrently | Multiple lines update | Mutex serializes updates; no interleaved or corrupted output |
| Non-color fallback | stdout lacks color support | Pre-check runs | One plain line per completion; no ANSI cursor control |

(Previously: the board rendered a flat per-tool list with no manager grouping or headers.)

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
   - `upp update -n`: Preview pending updates without making changes (`--dry-run`).
   - `upp update`: Apply updates to all enabled tools.
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

(Previously: the quick-reference listed the removed `upp check` as the preview entry point; `upp update -n` is now the documented query surface.)

### Requirement: Summary Report

Every `update` run (including `--dry-run`) MUST end with a summary showing:

- Count of tools updated / checked / skipped / failed
- List of tools in each category
- Overall status message

Summaries MUST count up-to-date and skipped tools explicitly ("N up to date, M skipped"). The summary MUST NOT print "All tools up to date." when any enabled tool was skipped or unchecked. A `--dry-run` summary MUST NOT print "All clean!" when any update is pending; pending updates MUST be reported explicitly. The tool list and status lines in the summary report MUST follow a 100% deterministic order matching canonical tool discovery order, unaffected by out-of-order concurrent completion during execution.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| All succeed | 5/5 updated | Summary | "✅ 5 updated, 0 failed. All clean!" |
| Partial fail | 3/5 updated, 2 failed | Summary | "✅ 3 updated, ❌ 2 failed. Review errors above." |
| No tools | All skipped | Summary | "⏭️ All tools not installed. Nothing to do." |
| Up-to-date with skips | 8 current, 2 enabled tools skipped (not installed) | `upp update --dry-run` | Summary counts skipped explicitly ("8 up to date, 2 skipped"); never "All tools up to date." |
| Dry-run pending | 3 updates pending, 7 current | `upp update --dry-run` | Summary reports "3 would update"; never pairs "All clean!" with pending updates |
| Concurrent deterministic order | Tools complete out-of-order across concurrent workers | `upp update --dry-run` finishes | Summary report lists tools strictly in canonical tool discovery order |

(Previously: the summary contract was anchored on `check` runs; the read-only query surface is now `update --dry-run`.)

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

When the global `--verbose` (or `-v`) flag is enabled, adapter execution failures during `update` (including `--dry-run`) MUST render detailed diagnostic information inline beneath the failed tool entry. Diagnostic information MUST include the captured subprocess stderr / error output from the failing tool adapter.

When `--verbose` / `-v` is omitted (default), adapter failure output MUST remain concise, showing only the tool name and short failure cause, suppressing raw subprocess stderr to keep terminal output clean.

In `--quiet` mode, verbose error diagnostics MUST be suppressed.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Verbose failure diagnostics | Adapter fails with subprocess stderr "lock frontend held by another process" | `upp update -v` or `upp update --verbose` | Failing tool line is followed by indented subprocess stderr output |
| Short flag `-v` diagnostics | Adapter fails during update | `upp update -v` | Behaves identically to `--verbose`, displaying detailed stderr diagnostics |
| Default non-verbose failure | Adapter fails during dry-run | `upp update --dry-run` (no `-v`) | Only concise inline error line displayed; raw stderr suppressed |
| Success with verbose | All tools succeed | `upp update -v` | Standard success output rendered without debug noise |
| Quiet takes precedence | Adapter fails in quiet mode with `-v` | `upp update -q -v` | Detailed subprocess stderr is suppressed in favor of quiet output |

(Previously: diagnostics were specified over `check`/`update`; `check` is removed and `update --dry-run` carries the read-only path.)

### Requirement: Progress Indication

For long-running operations, the system SHOULD show per-operation progress labeled with the operation: "Updating X/Y" during the update execution phase only. A read-only operation MUST NOT print "Updating". In TTY interactive `upp update` runs, the pre-check phase MUST be surfaced through the Live Check Board instead of a mutating counter; the system MUST NOT print a single-line "Checking X/Y" counter during that phase. During concurrent tool operations, progress rendering MUST be synchronized across worker threads so that output lines are emitted atomically and never interleave or corrupt terminal output.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Multi-tool update | 10 tools enabled | `upp update` executes selection | Progress shown: "Updating 3/10: brew" |
| Single tool | 1 tool enabled | Update running | No progress indicator needed |
| No checking counter | TTY, multiple tools | Interactive pre-check runs | Live Check Board renders; no "Checking X/Y" counter line |
| Concurrent progress atomicity | Multiple tools processed concurrently | Worker pool runs | Progress updates rendered atomically without line interleaving |

(Previously: progress used a single mutating "Checking X/Y" counter line for `check` and the interactive pre-check.)

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

`upp list` MUST render a table whose columns are labeled to match their data and MUST include the tool ID in its own column, and MUST group rows under their owning manager. Manager adapters render as group headers; owned tools (gh, docker, go) render as child rows beneath their resolved manager for the current platform. Owned tools MUST NOT render as standalone top-level rows. Grouping is DISPLAY-ONLY: `--only`/`--skip` filter names remain the per-tool IDs and MUST NOT change semantics.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Correct columns | 10 tools detected | `upp list` | Header `ID \| Name \| Status \| Version`; each row's ID usable with `--only`/`--skip` |
| Filter round-trip | Row shows ID `gh` | `upp list --only gh` | `gh` listed (row ID matches filter name) |
| Grouped by manager | Platform Linux, docker owned by apt | `upp list` | apt renders as header; docker renders as child row beneath it |
| Owned tool not independent | Platform macOS, gh owned by brew | `upp list` | gh appears under brew group, not as its own top-level row |
| Filters ignore grouping | `--only gh` and `--skip apt` on Linux | `upp list --only gh --skip apt` | gh still selected by ID regardless of being grouped under apt |

(Previously: `upp list` rendered a flat per-tool table with no manager grouping; owned tools appeared as independent top-level rows.)

### Requirement: Interactive Update Tool Selection

In TTY `upp update` runs, the pending-update checkbox selector MUST group pending updates under per-manager headers. Manager adapters with pending self-update render as their group header; owned tools with a pending delegated update render as child rows within their owning manager's group. Grouping is DISPLAY-ONLY and applies to the pending-only set, which is unchanged. The selector remains a user-choice UI, NOT a security confirmation: per-tool `security.ConfirmAction` gating MUST still run unchanged for every selected custom tool.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Selector groups pending | TTY, Linux, apt+gh pending | CheckboxSelector renders | apt group header with gh child row pre-checked |
| Owned tool in group | Platform Windows, winget+gh pending, scoop pending | CheckboxSelector renders | winget group with gh child; scoop as standalone group |
| Bypass unchanged | `--ci`, non-TTY, `--quiet`, or `--dry-run` | `upp update` | No selector; existing non-interactive behavior unchanged |
| Not a security confirmation | Custom high-risk tool selected in selector | Selector submitted | `security.ConfirmAction` prompt still shown before execution |

(Previously: the selector rendered a flat pending-only list with no manager grouping.)

### Requirement: Manager Self-Update Row Rendering

The brew manager row MUST render as current in interactive TTY `upp update` board runs, in `upp list`, and in `upp update --dry-run` (`-n`) output, because the brew adapter reports no self-update availability signal by design (`check()` sets `Latest=Current`, `UpdateAvailable=false`). The brew row MUST NOT appear in the pending CheckboxSelector in TTY runs; self-update runs only through the sequential/`--ci` PolicyAlwaysUpdate path. apt, winget, and scoop rows MUST render planned self-update actions in `-n` when their `check()` reports availability.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| brew current on board | TTY, brew check completes | `upp update` pre-check | brew line flips to ✓ up-to-date (current); no `current → new` shown |
| brew never pending | TTY, board settled, brew current | CheckboxSelector renders | brew absent from selector (no available signal); other pending tools listed |
| brew dry-run current | brew enabled, `--dry-run` | `upp update -n` | brew row shows current; no planned action (no `-n` signal exists) |
| apt dry-run planned | apt candidate > installed | `upp update -n` | apt row shows planned `sudo apt install --only-upgrade apt` action |
| winget dry-run planned | winget upgrade lists winget itself | `upp update -n` | winget row shows planned `winget upgrade winget` action |
| brew list version | brew installed, Homebrew 4.x | `upp list` | brew row shows its own version (e.g. 4.x.y) |

(Previously: manager rows were unspecified — brew conflated self + managed-package state, so no explicit current-only rendering contract existed for manager rows.)
