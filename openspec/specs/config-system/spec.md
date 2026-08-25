# Configuration System Specification

## Purpose

Manage persistent configuration: file format, location, and first-run wizard state.

## Requirements

### Requirement: Config File Location

The system MUST store config at `~/.config/upp/config.toml` on Linux/macOS and `%APPDATA%/upp/config.toml` on Windows.

The system MUST create the config directory if it does not exist on first run.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| First run on Linux | No `~/.config/upp/` | `upp init` | Directory created, config written |
| Existing config | Config file exists | `upp init` | Existing config preserved, prompt before overwrite |

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

### Requirement: Config Defaults

The system MUST provide sensible defaults for all settings. Defaults MUST be applied when fields are absent from an existing config file. First-run state MUST be determined by explicit config file existence, not inferred from applied defaults. Partial configs MUST default tool sections to the platform catalog.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Missing file | No config file exists | `upp init` | First-run wizard runs and creates config |
| Empty file | Config file exists but is empty | Config loaded | All defaults applied; NOT first-run |
| Partial config | Only `[settings]` section present | Config loaded | Tool sections default to platform catalog; NOT first-run |
| Full config | All fields present | Config loaded | Loaded as-is; defaults not applied |

(Previously: missing-file state was inferred from defaults, so first run appeared to have an existing config and the init wizard never ran; partial configs did not default tools to the catalog.)

### Requirement: Config Validation

The system MUST validate config on load. Validation MUST check: TOML syntax, required fields, platform compatibility of tools, and custom tool format.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Valid config | Correct syntax and fields | Load | Success |
| Unknown tool | Config references non-existent tool | Load | Warning, tool ignored |
| Platform mismatch | Linux-only tool enabled on macOS | Load | Warning, tool disabled |
