# Delta for ux-patterns

## MODIFIED Requirements

### Requirement: Progress Indication

For long-running operations, the system SHOULD show per-operation progress labeled with the operation: "Checking X/Y" for `check` and bare `upp`; "Updating X/Y" only for `update`. A read-only operation MUST NOT print "Updating". During concurrent tool checks, progress rendering MUST be synchronized across worker threads so that "Checking X/Y" output lines are emitted atomically and never interleave or corrupt terminal output.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Multi-tool check | 10 tools enabled | `upp check` | Progress shown: "Checking 3/10: brew" |
| Multi-tool update | 10 tools enabled | `upp update` | Progress shown: "Updating 3/10: brew" |
| Single tool | 1 tool enabled | Update running | No progress indicator needed |
| Concurrent check progress | Multiple tools checked concurrently | `upp check` runs worker pool | Progress updates are rendered atomically without line interleaving or corrupted output |

(Previously: progress always read "Updating X/Y", even for the read-only `check` command; concurrent worker output was not formally synchronized.)

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
