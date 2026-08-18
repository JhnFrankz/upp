# Delta for ux-patterns

## ADDED Requirements

### Requirement: List Table Output

`upp list` MUST render a table whose columns are labeled to match their data and MUST include the tool ID in its own column. The ID column MUST show the IDs used by `--only`/`--skip` and config (`apt`, `brew`, ...), so table rows map to filter names.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Correct columns | 10 tools detected | `upp list` | Header `ID | Name | Status | Version`; each row's ID usable with `--only`/`--skip` |
| Filter round-trip | Row shows ID `apt` | `upp check --only apt` | `apt` processed (row ID matches filter name) |

## MODIFIED Requirements

### Requirement: Summary Report

Every `update` (including `--dry-run`) and `check` run MUST end with a summary showing:

- Count of tools updated / checked / skipped / failed
- List of tools in each category
- Overall status message

`check` summaries MUST count skipped tools explicitly ("N up to date, M skipped"). The summary MUST NOT print "All tools up to date." when any enabled tool was skipped or unchecked. A `--dry-run` summary MUST NOT print "All clean!" when any update is pending; pending updates MUST be reported explicitly.
(Previously: check counted only available/current/failed, silently dropping skipped tools, and could print "All tools up to date." while enabled tools were unchecked; dry-run paired "All clean!" with pending updates.)

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| All succeed | 5/5 updated | Summary | "✅ 5 updated, 0 failed. All clean!" |
| Partial fail | 3/5 updated, 2 failed | Summary | "✅ 3 updated, ❌ 2 failed. Review errors above." |
| No tools | All skipped | Summary | "⏭️ All tools not installed. Nothing to do." |
| Check with skips | 8 current, 2 enabled tools skipped (not installed) | `upp check` | Summary counts skipped explicitly ("8 up to date, 2 skipped"); never "All tools up to date." |
| Dry-run pending | 3 updates pending, 7 current | `upp update --dry-run` | Summary reports "3 would update"; never pairs "All clean!" with pending updates |

### Requirement: Progress Indication

For long-running operations, the system SHOULD show per-operation progress labeled with the operation: "Checking X/Y" for `check` and bare `upp`; "Updating X/Y" only for `update`. A read-only operation MUST NOT print "Updating".
(Previously: progress always read "Updating X/Y", even for the read-only `check` command.)

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Multi-tool check | 10 tools enabled | `upp check` | Progress shown: "Checking 3/10: brew" |
| Multi-tool update | 10 tools enabled | `upp update` | Progress shown: "Updating 3/10: brew" |
| Single tool | 1 tool enabled | Update running | No progress indicator needed |

### Requirement: Self-Update Confirmation Prompt

`upp self-update` MUST show an English confirmation before replacing the binary, including current and target versions and the resolved binary path. Non-TTY stdin or `--ci` MUST print a clear English deny message and exit non-zero — never hang, auto-proceed, or silently skip.
(Previously: the requirement said "localized (en/es)"; output is English-only since the language drop — stale spec text corrected, no behavior change.)

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| TTY prompt | TTY, update available | `upp self-update` | English prompt with versions + path, waits for y/N |
| User declines | TTY, user answers n | Prompt | No changes, exit 0 |
| Non-TTY | stdin is not a TTY | `upp self-update` | Clear English deny message, exit non-zero |
| `--ci` | `upp self-update --ci` | Execution | Clear English deny message, exit non-zero |
