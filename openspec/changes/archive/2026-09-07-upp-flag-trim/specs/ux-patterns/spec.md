# Delta for ux-patterns

<!-- Archive note: no Purpose prose change needed. Two `--skip` mentions remain intentionally in "List Table Output" display-only column-filter examples; they are removed by this delta's MODIFIED block. -->

## REMOVED Requirements

### Requirement: Opt-In Flag UX

(Reason: the opt-in group path (`--manager`/`--update-group`) is deleted; production routing only ever executes the default delegated per-package path, so opt-in flag documentation and batch-preview UX have no behavior to describe. Flag-rejection scenarios for the removed flags live in `command-interface` and `bulk-update` deltas.)
(Migration: none — `upp update --help` simply no longer lists these flags; the group summary UX itself is preserved under the Summary Report requirement via the default path.)

## MODIFIED Requirements

### Requirement: Summary Report

Every `update` run (including `--dry-run` and default manager-group runs) MUST end with a summary showing:

- Count of tools updated / checked / skipped / failed
- List of tools in each category
- Overall status message

Summaries MUST count up-to-date and skipped tools explicitly ("N up to date, M skipped"). For default manager-group updates, the summary MUST render group package updates alongside standalone tools, reporting each owned tool that was updated, skipped (via deselection), current, or failed within its manager group. The summary MUST NOT print "All tools up to date." when any enabled tool was skipped or unchecked. A `--dry-run` summary MUST NOT print "All clean!" when any update is pending; pending updates MUST be reported explicitly. The tool list and status lines in the summary report MUST follow a 100% deterministic order matching canonical tool discovery order, unaffected by out-of-order concurrent completion during execution.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| All succeed in default run | 5/5 updated across manager groups and standalone tools | `upp update` | Summary: "✅ 5 updated, 0 failed. All clean!" with manager groups and tools listed in canonical order |
| Partial fail with group isolation | apt group: gh fails, docker succeeds; standalone npm succeeds | `upp update` | Summary: "✅ 2 updated, ❌ 1 failed. Review errors above.", showing gh failed under apt |
| No tools installed | All enabled tools not installed | `upp update` | Summary: "⏭️ All tools not installed. Nothing to do." |
| Up-to-date with skips | 8 current, 2 enabled tools skipped | `upp update --dry-run` | Summary counts skipped explicitly ("8 up to date, 2 skipped"); never "All tools up to date." |
| Dry-run pending | 3 updates pending (2 in brew group, 1 standalone), 7 current | `upp update --dry-run` | Summary reports "3 would update"; never pairs "All clean!" with pending updates |
| Concurrent deterministic order | Tools complete out-of-order across concurrent workers | `upp update --dry-run` finishes | Summary report lists tools strictly in canonical tool discovery order |
| Default group bulk summary | Linux, bare update with apt owning gh (updated) and docker (skipped) | `upp update` | Group summary lists apt group with gh updated, docker skipped |
| Filtered group partial fail | brew group: gh updated, docker failed, `--only gh,docker` | `upp update --only gh,docker` | Group summary lists gh updated, docker failed under brew |
| Group dry-run preview | apt group, gh pending, docker current | `upp update -n` | Group summary reports gh would update, docker current under apt group preview |

(Previously: manager-group summaries were only generated when explicitly triggered via `--manager`/`--update-group` opt-in flags; default runs did not render manager-group package updates or per-tool group outcomes.)

### Requirement: List Table Output

`upp list` MUST render a table whose columns are labeled to match their data and MUST include the tool ID in its own column, and MUST group rows under their owning manager. Manager adapters render as group headers; owned tools (gh, docker, go) render as child rows beneath their resolved manager for the current platform. Owned tools MUST NOT render as standalone top-level rows. Grouping is DISPLAY-ONLY: `--only` filter names remain the per-tool IDs and MUST NOT change semantics.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Correct columns | 10 tools detected | `upp list` | Header `ID \| Name \| Status \| Version`; each row's ID usable with `--only` |
| Filter round-trip | Row shows ID `gh` | `upp list --only gh` | `gh` listed (row ID matches filter name) |
| Grouped by manager | Platform Linux, docker owned by apt | `upp list` | apt renders as header; docker renders as child row beneath it |
| Owned tool not independent | Platform macOS, gh owned by brew | `upp list` | gh appears under brew group, not as its own top-level row |
| Filters ignore grouping | `--only gh` on Linux, gh grouped under apt | `upp list --only gh` | gh still selected by ID regardless of being grouped under apt |

(Previously: `upp list` rendered a flat per-tool table with no manager grouping; owned tools appeared as independent top-level rows.)
