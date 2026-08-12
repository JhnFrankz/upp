# Delta for ux-patterns

## ADDED Requirements

### Requirement: Self-Update Detection Hint

When `settings.check_self_update` is enabled and a newer release is known, `check`/bare `upp` MUST append exactly one hint line after the summary: `⬆️ upp v{latest} available (current {current}) — run "upp self-update"`. The hint MUST NOT change the exit code. The hint MUST be omitted when `--quiet` is set, when the check failed or was offline (silent, no error), or when no newer release exists.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Newer release | `check_self_update=true`, v0.1.1 vs v0.1.0 | `upp check` | One hint line after summary, exit unchanged |
| Offline | `check_self_update=true`, API unreachable | `upp check` | No hint, no error, exit unchanged |
| Quiet | `check_self_update=true`, `--quiet` | `upp check` | No hint line |
| Up to date | `check_self_update=true`, latest = current | `upp check` | No hint line |

### Requirement: Self-Update Confirmation Prompt

`upp self-update` MUST show a localized (en/es) confirmation before replacing the binary, including current and target versions and the resolved binary path. Non-TTY stdin or `--ci` MUST print a clear localized deny message and exit non-zero — never hang, auto-proceed, or silently skip.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| TTY prompt | TTY, update available | `upp self-update` | Localized prompt with versions + path, waits for y/N |
| User declines | TTY, user answers n | Prompt | No changes, exit 0 |
| Non-TTY | stdin is not a TTY | `upp self-update` | Clear deny message, exit non-zero |
| `--ci` | `upp self-update --ci` | Execution | Clear deny message, exit non-zero |
