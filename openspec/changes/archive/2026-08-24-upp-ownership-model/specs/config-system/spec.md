# Delta for config-system

## MODIFIED Requirements

### Requirement: Config Format

The config file MUST be TOML (human-readable, Git-compatible).

Minimum config structure:

```toml
[tools]
  [tools.apt]
  enabled = true
  platforms = ["linux"]

  [tools.brew]
  enabled = true
  platforms = ["linux", "macos"]

[custom]
  [custom.mytool]
  command = "mytool --update"
  trusted = false
  manager = "brew"   # optional: declare an owning manager
```

A custom tool MAY declare an owning `manager` key naming a known official manager (apt, brew, winget, scoop) for the tool's platform. When declared, the custom tool MUST group and update under that manager; an unknown or non-manager `manager` value MUST be ignored with a warning (forward-compatible). The `[settings]` section MUST NOT contain an output-language key: upp output is English-only. A `language` or `interactive` key in an existing config file is ignored (unknown settings are tolerated for forward compatibility) and MUST NOT be written by `upp init`. The `manager` key MUST NOT be written by `upp init` (an optional, user-declared field).

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Custom with manager | `[custom.mytool]` has `manager="brew"` on macOS | Config loaded | mytool grouped/updated under brew |
| Valid TOML | Config has valid TOML syntax | Config loaded | All fields parsed correctly |
| Invalid TOML | Malformed config file | Config loaded | Error message with line number, exit non-zero |
| Missing fields | Config has partial content | Config loaded | Missing fields use defaults |
| Unknown manager | Custom tool declares `manager="bogus"` | Config loaded | Key ignored; tool standalone; warning emitted |
| Init hygiene | Config loaded | `upp init` | `manager` key never written to output |

(Previously: the custom tool schema had only `command`, `check_cmd`, and `trusted`; no way to declare an owning manager.)
