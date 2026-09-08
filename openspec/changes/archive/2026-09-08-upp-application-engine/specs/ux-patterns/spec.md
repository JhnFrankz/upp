# Delta for ux-patterns

## MODIFIED Requirements

### Requirement: Live Check Board

In TTY interactive `upp update` runs, the pre-check board MUST render one stable line per filtered tool, laid out grouped under per-manager headers in canonical discovery order before any result arrives. Manager headers render first, then their owned tools, then standalone tools. An owned tool MUST NOT appear as a top-level line separate from its manager group. As a standalone tool across Linux, macOS, and Windows, `uv` MUST render under the standalone tools section with its own stable line in canonical tool discovery order.

The pre-check board MUST receive per-tool completion events bridged from `engine.CheckProgress` notifications emitted during `engine.Check`, mapping pure domain outcomes (`engine.CheckOutcome`) to presentation models (`output.ToolResult`). Per-tool completion flip, up-to-date visibility, failed-check ✗ behavior, atomic concurrent rendering, the settled-board gating of the selector, and non-color fallback MUST remain unchanged. Grouping MUST NOT reorder stable board lines or alter completion ordering.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Board renders grouped | TTY, Linux, apt+gh+docker | `upp update` pre-check starts | apt header, then gh+docker child lines, then standalone tools; one stable line per tool |
| Owned tool in group | Platform Linux, docker owned by apt | Pre-check renders | docker line appears beneath apt header, not top-level |
| Per-tool completion flip | brew finishes first, v1.2 → v1.3 | brew check completes | Only brew's line flips to ✓ showing `1.2 → 1.3`; other lines unchanged |
| Settled board gates selector | Board settled, 2 of 5 tools pending | Pre-check ends | CheckboxSelector lists only the 2 pending tools; current and failed excluded |
| Atomic concurrent rendering | Worker pool completes checks concurrently | Multiple lines update | Mutex serializes updates; no interleaved or corrupted output |
| Non-color fallback | stdout lacks color support | Pre-check runs | One plain line per completion; no ANSI cursor control |
| uv standalone on board | TTY, `uv` enabled on any platform | `upp update` pre-check starts | `uv` renders as a standalone tool line below manager groups |
| uv completion flip | `uv` check completes with pending update | `uv` check completes | `uv` line flips to ✓ showing pending update version details |
| uv current flip | `uv` check completes with no updates pending | `uv` check completes | `uv` line flips to ✓ up-to-date; excluded from settled selector |
| Bridged progress update | Worker finishes check via engine | Callback receives `engine.CheckProgress` | Mapped to `output.ToolResult` and flips corresponding line on `CheckBoard` |

(Previously: the pre-check board was updated directly by CLI-internal `runChecks` worker callbacks using `checkOutcome` structs tightly coupled to `output.ToolResult`.)

### Requirement: Summary Report

Every `update` run (including `--dry-run` and default manager-group runs) MUST end with a summary showing:

- Count of tools updated / checked / skipped / failed
- List of tools in each category
- Overall status message

Summaries MUST count up-to-date and skipped tools explicitly ("N up to date, M skipped"). For default manager-group updates, the summary MUST render group package updates alongside standalone tools, reporting each owned tool that was updated, skipped (via deselection), current, or failed within its manager group. Standalone tool updates across all platforms, including `uv`, MUST be reported in the summary report reflecting their composite execution outcome (updated, current, skipped, or failed) in canonical discovery order. The summary MUST NOT print "All tools up to date." when any enabled tool was skipped or unchecked. A `--dry-run` summary MUST NOT print "All clean!" when any update is pending; pending updates MUST be reported explicitly. The tool list and status lines in the summary report MUST follow a 100% deterministic order matching canonical tool discovery order, unaffected by out-of-order concurrent completion during execution. The deterministic order MUST be guaranteed by the application engine's index-slotted outcome gathering, mapped 1:1 to `output.ToolResult` presentation models.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| All succeed in default run | 5/5 updated across manager groups and standalone tools | `upp update` | Summary: "✅ 5 updated, 0 failed. All clean!" with manager groups and tools listed in canonical order |
| Partial fail with group isolation | apt group: gh fails, docker succeeds; standalone npm succeeds | `upp update` | Summary: "✅ 2 updated, ❌ 1 failed. Review errors above.", showing gh failed under apt |
| No tools installed | All enabled tools not installed | `upp update` | Summary: "⏭️ All tools not installed. Nothing to do." |
| Up-to-date with skips | 8 current, 2 enabled tools skipped | `upp update --dry-run` | Summary counts skipped explicitly ("8 up to date, 2 skipped"); never "All tools up to date." |
| Dry-run pending | 3 updates pending (2 in brew group, 1 standalone), 7 current | `upp update --dry-run` | Summary reports "3 would update"; never pairs "All clean!" with pending updates |
| Concurrent deterministic order | Tools complete out-of-order across concurrent workers | `upp update --dry-run` finishes | Summary report lists tools strictly in canonical tool discovery order |
| Default group bulk summary | Linux, bare update with apt owning gh (updated) and docker (skipped) | `upp update` | Flat summary reports gh updated and docker skipped alongside standalone tools; each owned tool is reported within the flat summary |
| Filtered group partial fail | brew group: gh updated, docker failed, `--only gh,docker` | `upp update --only gh,docker` | Flat summary reports gh updated and docker failed |
| Group dry-run preview | apt group, gh pending, docker current | `upp update -n` | Flat summary reports gh would update and docker current |
| uv standalone in summary | `uv` updated successfully alongside manager groups and other standalone tools | `upp update` | Summary lists `uv` under updated tools in canonical discovery order |
| uv dry-run summary | `uv` has pending tool updates, `--dry-run` | `upp update -n` | Summary reports `uv` would update; does not report "All clean!" |

(Previously: deterministic summary ordering was handled by CLI-internal `runChecks` slotting; deterministic outcome ordering is now guaranteed by `engine.Check` index slotting and mapped to presentation models.)

## NEW Requirements

### Requirement: Engine Progress and Outcome Mapping

The CLI presentation layer MUST bridge `engine.CheckProgress` events to `output.CheckBoard` and map `engine.CheckOutcome` domain structs 1:1 to `output.ToolResult` presentation models.

Mapping rules:
1. `engine.StatusAvailable` MUST map to `output.StatusAvailable` with `Version` formatted as `"<CurrentVersion> → <LatestVersion>"`.
2. `engine.StatusCurrent` MUST map to `output.StatusCurrent` with `Version` set to `CurrentVersion`.
3. `engine.StatusSkipped` MUST map to `output.StatusSkipped`.
4. `engine.StatusFailed` MUST map to `output.StatusFailed` with `Error` populated from `outcome.Err` and `Stderr` populated from `outcome.Stderr`.

The mapping MUST preserve exact visual layout, ANSI cursor positioning, emoji and styling indicators, and summary ordering without modification.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| StatusAvailable mapping | `CheckOutcome` with `StatusAvailable`, Current "1.0", Latest "1.1" | Bridge maps to `output.ToolResult` | Result has `Status: StatusAvailable`, `Version: "1.0 → 1.1"` |
| StatusCurrent mapping | `CheckOutcome` with `StatusCurrent`, Current "2.0" | Bridge maps to `output.ToolResult` | Result has `Status: StatusCurrent`, `Version: "2.0"` |
| StatusSkipped mapping | `CheckOutcome` with `StatusSkipped` | Bridge maps to `output.ToolResult` | Result has `Status: StatusSkipped` |
| StatusFailed mapping | `CheckOutcome` with `StatusFailed`, error, and stderr | Bridge maps to `output.ToolResult` | Result has `Status: StatusFailed`, `Error: err`, `Stderr: stderr` |
| CheckBoard line flipping | Progress callback receives `CheckProgress` for index $i$ | CLI invokes `board.Complete(i, res)` | Row $i$ on `CheckBoard` flips to mapped outcome atomically |
