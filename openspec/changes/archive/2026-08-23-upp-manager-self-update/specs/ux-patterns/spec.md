# Delta for ux-patterns

## ADDED Requirements

### Requirement: Manager Self-Update Row Rendering

The brew manager row MUST render as current in interactive TTY `upp update` board runs, in `upp list`, and in `upp update --dry-run` (`-n`) output, because the brew adapter reports no self-update availability signal by design (`check()` sets `Latest=Current`, `UpdateAvailable=false`). The brew row MUST NOT appear in the pending CheckboxSelector in TTY runs; self-update runs only through the sequential/`--ci` PolicyAlwaysUpdate path. apt, winget, and scoop rows MUST render planned self-update actions in `-n` when their `check()` reports availability.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| brew current on board | TTY, brew check completes | `upp update` pre-check | brew line flips to ✓ up-to-date (current); no `current → new` shown |
| brew never pending | TTY, board settled, brew current | CheckboxSelector renders | brew absent from selector (no available signal); other pending tools listed |
| brew dry-run current | brew enabled, `--dry-run` | `upp update -n` | brew row shows current; no planned action (no `-n` signal exists) |
| apt dry-run planned | apt candidate > installed | `upp update -n` | apt row shows planned `sudo apt install --only-upgrade apt` action |
| winget dry-run planned | winget upgrade lists winget itself | `upp update -n` | winget row shows planned `winget upgrade winget` action |
| brew list version | brew installed, Homebrew 4.x | `upp list` | brew row shows its own version (e.g. 4.x.y) |

(Previously: manager rows were unspecified — brew conflated self + managed-package state, so no explicit current-only rendering contract existed for manager rows.)
