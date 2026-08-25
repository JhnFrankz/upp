# Delta for ux-patterns

## MODIFIED Requirements

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

### Requirement: Interactive Update Tool Selection

In TTY `upp update` runs, the pending-update checkbox selector MUST group pending updates under per-manager headers. Manager adapters with pending self-update render as their group header; owned tools with a pending delegated update render as child rows within their owning manager's group. Grouping is DISPLAY-ONLY and applies to the pending-only set, which is unchanged. The selector remains a user-choice UI, NOT a security confirmation: per-tool `security.ConfirmAction` gating MUST still run unchanged for every selected custom tool.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Selector groups pending | TTY, Linux, apt+gh pending | CheckboxSelector renders | apt group header with gh child row pre-checked |
| Owned tool in group | Platform Windows, winget+gh pending, scoop pending | CheckboxSelector renders | winget group with gh child; scoop as standalone group |
| Bypass unchanged | `--ci`, non-TTY, `--quiet`, or `--dry-run` | `upp update` | No selector; existing non-interactive behavior unchanged |
| Not a security confirmation | Custom high-risk tool selected in selector | Selector submitted | `security.ConfirmAction` prompt still shown before execution |

(Previously: the selector rendered a flat pending-only list with no manager grouping.)
