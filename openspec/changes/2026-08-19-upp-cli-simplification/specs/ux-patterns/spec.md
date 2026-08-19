# Delta for ux-patterns

## ADDED Requirements

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

## MODIFIED Requirements

### Requirement: Progress Indication

For long-running operations, the system SHOULD show per-operation progress labeled with the operation: "Checking X/Y" for `check`; "Updating X/Y" only for `update`. A read-only operation MUST NOT print "Updating". During concurrent tool checks, progress rendering MUST be synchronized across worker threads so that "Checking X/Y" output lines are emitted atomically and never interleave or corrupt terminal output.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Multi-tool check | 10 tools enabled | `upp check` | Progress shown: "Checking 3/10: brew" |
| Multi-tool update | 10 tools enabled | `upp update` | Progress shown: "Updating 3/10: brew" |
| Single tool | 1 tool enabled | Update running | No progress indicator needed |
| Concurrent check progress | Multiple tools checked concurrently | `upp check` runs worker pool | Progress updates are rendered atomically without line interleaving or corrupted output |

(Previously: progress indicated bare `upp` printed "Checking X/Y"; bare `upp` is now a non-checking dashboard.)
