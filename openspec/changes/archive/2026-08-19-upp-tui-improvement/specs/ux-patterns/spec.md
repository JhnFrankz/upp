# Delta for ux-patterns

## ADDED Requirements

### Requirement: Interactive Update Tool Selection

In TTY `upp update` runs, the system MUST show a checkbox selector of pending tool updates before any update executes, with all pending tools pre-checked. The system MUST skip the selector when `--ci` is set, when stdin is not a TTY, when `--quiet` is set, or when `--dry-run` is set. When no pending updates exist, the system MUST skip the selector and show the normal summary.

The selector is a user-choice UI, NOT a security confirmation: per-tool `security.ConfirmAction` gating MUST still run unchanged for every selected custom tool.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Default TTY show | TTY stdin, 3 pending updates | `upp update` | Selector rendered with all 3 tools pre-checked; no update executed yet |
| Enter updates all | Selector shown, 3 pre-checked | Enter pressed | All 3 updated; summary matches full-set update |
| Esc cancels run | Selector shown, pending updates | Esc pressed | Nothing updated, clear cancel message, exit 0 |
| `q` cancels run | Selector shown, pending updates | `q` pressed | Nothing updated, clear cancel message, exit 0 |
| No pending updates | TTY, all tools current | `upp update` | No selector; normal summary shown |
| `--ci` bypass | `--ci` set, pending updates | `upp update --ci` | No selector; current non-interactive behavior unchanged |
| Non-TTY bypass | stdin not a TTY, pending updates | `upp update` | No selector rendered; updates proceed non-interactively |
| `--quiet` bypass | `--quiet` set, pending updates | `upp update --quiet` | No selector; quiet per-tool output + summary |
| `--dry-run` bypass | `--dry-run` set, pending updates | `upp update --dry-run` | No selector; planned actions listed, no changes |
| Not a security confirmation | Custom high-risk tool selected in selector | Selector submitted | `security.ConfirmAction` prompt still shown before execution |
