# Delta for config-system

## ADDED Requirements

### Requirement: Self-Update Detection Setting

The system MUST support `settings.check_self_update` (boolean, default `false`) that opts into the update-detection hint at the end of `check`/bare `upp` output. With default config, `check`/bare `upp` MUST perform ZERO self-update network calls (test-enforced). The detection cache MUST be stored in the config directory as `self-update-cache.json` (`~/.config/upp/` on Linux/macOS, `%APPDATA%/upp/` on Windows), created on first hint check.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Default off | No setting present | Config loaded | `check_self_update=false` |
| Explicit on | `check_self_update = true` | Config loaded | Hint enabled |
| Zero network default | Default config | `upp check` | No self-update HTTP calls (test-enforced) |
| Cache location | Hint check runs | Cache written | `{config-dir}/self-update-cache.json` created |
