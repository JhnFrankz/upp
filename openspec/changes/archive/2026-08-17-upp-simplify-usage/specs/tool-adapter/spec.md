# Delta for tool-adapter

## MODIFIED Requirements

### Requirement: Version Comparison

Adapters MUST return semver-compatible version strings when available. Adapters SHOULD normalize version formats across platforms. When both current and latest versions parse as semver (leading `v` prefix tolerated), update availability MUST be determined by semantic version comparison, not string inequality: current > latest MUST report `update_available=false` (no downgrade). When either version cannot be parsed as semver, the adapter MUST NOT report an update based on string inequality alone; it reports unknown (`update_available=false`) without error. The nvm adapter MUST use this semver comparison for its update detection.
(Previously: update detection compared raw version strings for inequality (nvm: `current != latest`), so a newer current version was reported as an "update" to an older latest.)

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Semver version | Node v20.11.0 | `check()` | Returns `current="20.11.0"` |
| Non-semver | Docker "24.0.7" | `check()` | Returns raw version string |
| Newer current | nvm current `v26.7.0`, latest `v24.19.0` | `check()` | `update_available=false`; no downgrade |
| Older current | nvm current `v18.0.0`, latest `v20.11.0` | `check()` | `update_available=true` |
| Equal versions | nvm current `v20.11.0`, latest `20.11.0` | `check()` | `update_available=false` |
| Unparseable | nvm current `v26.7.0`, latest `stable` | `check()` | `update_available=false`, no error (unknown) |
