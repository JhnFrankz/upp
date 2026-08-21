# Delta for ux-patterns

## ADDED Requirements

### Requirement: Live Check Board

In TTY interactive `upp update` runs (stdin is a TTY; `--ci`, `--quiet`, and `--dry-run` are not set), the pre-check phase MUST render a live board with exactly one stable line per filtered tool, laid out in canonical tool discovery order before any result arrives. When an individual tool's check completes, its own line MUST flip in place to a ✓ marker with `current → new`; up-to-date tools MUST remain visible on the board marked ✓ up-to-date; a failed check MUST flip that line to ✗ with its inline error. Completion order MUST NOT reorder board lines. When the board settles, the CheckboxSelector MUST render over the pending-only tool set; up-to-date and failed tools MUST NOT appear in the selector. Board rendering MUST be atomic across concurrent workers: a mutex MUST serialize line updates so output never interleaves or corrupts. When color is unavailable (non-color stdout or non-TTY output), the board MUST fall back to one plain line per completion without ANSI cursor control.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Board renders up-front | TTY, 5 enabled tools | `upp update` pre-check starts | One stable line per tool shown immediately, in canonical order |
| Per-tool completion flip | brew finishes first, v1.2 → v1.3 available | brew's check completes | Only brew's line flips to ✓ showing `1.2 → 1.3`; other lines unchanged |
| Up-to-date stays visible | npm is current | npm's check completes | npm's line shows ✓ up-to-date and remains on the board |
| Failed check flips to ✗ | apt's check errors | apt's check completes | apt's line flips to ✗ with its inline error |
| Settled board gates selector | Board settled, 2 of 5 tools pending | Pre-check phase ends | CheckboxSelector lists only the 2 pending tools; current and failed tools excluded |
| Atomic concurrent rendering | Worker pool completes checks concurrently | Multiple lines update at once | Mutex serializes updates; no interleaved or corrupted output |
| Non-color fallback | stdout lacks color support | Pre-check runs | One plain line per completion; no ANSI cursor control |
| Bypass modes unchanged | `--ci`, `--quiet`, `--dry-run`, or non-TTY stdin | `upp update` | No board; sequential output per existing flag contracts |

## MODIFIED Requirements

### Requirement: Progress Indication

For long-running operations, the system SHOULD show per-operation progress labeled with the operation: "Updating X/Y" during the update execution phase only. A read-only operation MUST NOT print "Updating". In TTY interactive `upp update` runs, the pre-check phase MUST be surfaced through the Live Check Board instead of a mutating counter; the system MUST NOT print a single-line "Checking X/Y" counter during that phase. During concurrent tool operations, progress rendering MUST be synchronized across worker threads so that output lines are emitted atomically and never interleave or corrupt terminal output.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Multi-tool update | 10 tools enabled | `upp update` executes selection | Progress shown: "Updating 3/10: brew" |
| Single tool | 1 tool enabled | Update running | No progress indicator needed |
| No checking counter | TTY, multiple tools | Interactive pre-check runs | Live Check Board renders; no "Checking X/Y" counter line |
| Concurrent progress atomicity | Multiple tools processed concurrently | Worker pool runs | Progress updates rendered atomically without line interleaving |

(Previously: progress used a single mutating "Checking X/Y" counter line for `check` and the interactive pre-check.)

### Requirement: List Table Output

`upp list` MUST render a table whose columns are labeled to match their data and MUST include the tool ID in its own column. The ID column MUST show the IDs used by `--only`/`--skip` and config (`apt`, `brew`, ...), so table rows map to filter names.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Correct columns | 10 tools detected | `upp list` | Header `ID \| Name \| Status \| Version`; each row's ID usable with `--only`/`--skip` |
| Filter round-trip | Row shows ID `apt` | `upp list --only apt` | `apt` listed (row ID matches filter name) |

(Previously: the filter round-trip scenario was demonstrated through the now-removed `upp check --only apt`.)

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

## REMOVED Requirements

### Requirement: Self-Update Detection Hint

**Reason**: The hint surface is removed entirely along with `upp check`: no code path renders it, `settings.check_self_update` is dropped, and self-update stays manual via `upp self-update`.

**Migration**: None — users run `upp self-update` manually; no replacement hint exists.
