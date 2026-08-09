# UX Patterns Specification

## Purpose

Output formatting, interactive prompts, summary display, and cross-platform terminal behavior.

## Requirements

### Requirement: Default Interactive Mode

The system MUST be interactive by default. Before each tool update, the system MUST prompt the user to proceed unless `--ci` is set.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Default run | No flags | Update tool | Prompt: "Update brew? [Y/n]" |
| `--ci` run | `--ci` flag | Update tool | No prompt, auto-proceed |
| `--quiet` run | `--quiet` flag | Update tool | Prompt still shown (quiet affects detail, not prompts) |

### Requirement: Output Language

The system MUST output in English by default. Output language MUST be configurable via `settings.language` in config.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Default language | No language setting | Any command | English output |
| Spanish config | `language = "es"` | Any command | Spanish output |

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

### Requirement: Summary Report

Every `update` (including `--dry-run`) and `check` run MUST end with a summary showing:

- Count of tools updated / checked / skipped / failed
- List of tools in each category
- Overall status message

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| All succeed | 5/5 updated | Summary | "✅ 5 updated, 0 failed. All clean!" |
| Partial fail | 3/5 updated, 2 failed | Summary | "✅ 3 updated, ❌ 2 failed. Review errors above." |
| No tools | All skipped | Summary | "⏭️ All tools not installed. Nothing to do." |

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

### Requirement: Progress Indication

For long-running operations (multi-tool update), the system SHOULD show progress (e.g., "Updating 3/10...").

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Multi-tool | 10 tools enabled | Update running | Progress shown: "Updating 3/10: brew" |
| Single tool | 1 tool enabled | Update running | No progress indicator needed |
