# Delta for UX Patterns

## MODIFIED Requirements

### Requirement: Manager Self-Update Row Rendering

The brew manager row MUST render as current in interactive TTY `upp update` board runs, in `upp list`, and in `upp update --dry-run` (`-n`) output, because the brew adapter reports no self-update availability signal by design (`check()` sets `Latest=Current`, `UpdateAvailable=false`). The interactive pending CheckboxSelector MUST instead be derived from `engine.Plan` (`plan.Updates`), which plans `PolicyAlwaysUpdate` tools unconditionally (`internal/engine/plan.go:92-97,153-164`); therefore `PolicyAlwaysUpdate` managers and standalone tools (e.g. brew, bun) MUST appear in the TTY pending selector even when currently up to date. apt, winget, and scoop rows MUST render planned self-update actions in `-n` when their `check()` reports availability.

(Previously: the brew row MUST NOT appear in the TTY pending CheckboxSelector, and self-update ran only through the sequential/`--ci` PolicyAlwaysUpdate path — the reverse of this requirement. This delta supersedes that rule.)

(Migration: the D7 assertions that pin "always-update current tool excluded from the TTY pending set" at `internal/cli/update_test.go:835-839,1016-1023,1249-1252` MUST be rewritten to assert inclusion in this change.)

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| brew current on board | TTY, brew check completes | `upp update` pre-check | brew line flips to ✓ up-to-date (current); no `current → new` shown |
| brew pending in selector | TTY, board settled, brew current (`UpdateAvailable=false`) | CheckboxSelector renders | brew listed in the pending selector (planned unconditionally by `PolicyAlwaysUpdate`) |
| bun pending in selector | TTY, board settled, bun current (`UpdateAvailable=false`) | CheckboxSelector renders | bun listed in the pending selector |
| brew dry-run current | brew enabled, `--dry-run` | `upp update -n` | brew row shows current; no planned action (no `-n` signal exists) |
| apt dry-run planned | apt candidate > installed | `upp update -n` | apt row shows planned `sudo apt install --only-upgrade apt` action |
| winget dry-run planned | winget upgrade lists winget itself | `upp update -n` | winget row shows planned `winget upgrade winget` action |
| brew list version | brew installed, Homebrew 4.x | `upp list` | brew row shows its own version (e.g. 4.x.y) |

(Previously: manager rows were unspecified — brew conflated self + managed-package state, so no explicit current-only rendering contract existed for manager rows.)
