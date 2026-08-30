# Delta for ux-patterns

## MODIFIED Requirements

### Requirement: Summary Report

Every `update` run (including `--dry-run` and default manager-group runs) MUST end with a summary showing:

- Count of tools updated / checked / skipped / failed
- List of tools in each category
- Overall status message

Summaries MUST count up-to-date and skipped tools explicitly ("N up to date, M skipped"). For default and filtered manager-group updates, the summary MUST render group package updates alongside standalone tools, reporting each owned tool that was updated, skipped (via `--skip` or deselection), current, or failed within its manager group. The summary MUST NOT print "All tools up to date." when any enabled tool was skipped or unchecked. A `--dry-run` summary MUST NOT print "All clean!" when any update is pending; pending updates MUST be reported explicitly. The tool list and status lines in the summary report MUST follow a 100% deterministic order matching canonical tool discovery order, unaffected by out-of-order concurrent completion during execution.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| All succeed in default run | 5/5 updated across manager groups and standalone tools | `upp update` | Summary: "✅ 5 updated, 0 failed. All clean!" with manager groups and tools listed in canonical order |
| Partial fail with group isolation | apt group: gh fails, docker succeeds; standalone npm succeeds | `upp update` | Summary: "✅ 2 updated, ❌ 1 failed. Review errors above.", showing gh failed under apt |
| No tools installed | All enabled tools not installed | `upp update` | Summary: "⏭️ All tools not installed. Nothing to do." |
| Up-to-date with skips | 8 current, 2 enabled tools skipped | `upp update --dry-run` | Summary counts skipped explicitly ("8 up to date, 2 skipped"); never "All tools up to date." |
| Dry-run pending | 3 updates pending (2 in brew group, 1 standalone), 7 current | `upp update --dry-run` | Summary reports "3 would update"; never pairs "All clean!" with pending updates |
| Concurrent deterministic order | Tools complete out-of-order across concurrent workers | `upp update --dry-run` finishes | Summary report lists tools strictly in canonical tool discovery order |
| Default group bulk summary | Linux, bare update with apt owning gh (updated) and docker (skipped) | `upp update --skip docker` | Group summary lists apt group with gh updated, docker skipped |
| Filtered group partial fail | brew group: gh updated, docker failed | `upp update --manager brew` | Group summary lists gh updated, docker failed under brew |
| Group dry-run preview | apt group, gh pending, docker current | `upp update -n` | Group summary reports gh would update, docker current under apt group preview |

(Previously: manager-group summaries were only generated when explicitly triggered via `--manager`/`--update-group` opt-in flags; default runs did not render manager-group package updates or per-tool group outcomes.)
