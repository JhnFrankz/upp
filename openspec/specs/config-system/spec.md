# Configuration System Specification

## Purpose

Manage persistent configuration: file format, location, import/export, and first-run wizard state.

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
[settings]
language = "en"          # output language
interactive = true       # prompt before updates

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
```

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Valid TOML | Config has valid TOML syntax | Config loaded | All fields parsed correctly |
| Invalid TOML | Malformed config file | Config loaded | Error message with line number, exit non-zero |
| Missing fields | Config has partial content | Config loaded | Missing fields use defaults |

### Requirement: Config Defaults

The system MUST provide sensible defaults for all settings. Defaults MUST be applied when fields are absent from an existing config file. First-run state MUST be determined by explicit config file existence, not inferred from applied defaults. Partial configs MUST default tool sections to the platform catalog.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Missing file | No config file exists | `upp init` | First-run wizard runs and creates config |
| Empty file | Config file exists but is empty | Config loaded | All defaults applied; NOT first-run |
| Partial config | Only `[settings]` section present | Config loaded | Tool sections default to platform catalog; NOT first-run |
| Full config | All fields present | Config loaded | Loaded as-is; defaults not applied |

(Previously: missing-file state was inferred from defaults, so first run appeared to have an existing config and the init wizard never ran; partial configs did not default tools to the catalog.)

### Requirement: Export/Import

`upp export` MUST serialize the current config to stdout or a file. `upp import` MUST load a config file and merge/replace the current config.

Export format MUST be identical to the config file format (round-trippable).

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Export to stdout | Config exists | `upp export` | TOML output to stdout |
| Export to file | Config exists | `upp export -o backup.toml` | File written |
| Import replaces | `backup.toml` has tools section | `upp import backup.toml` | Current config replaced |
| Import validates | `bad.toml` has syntax error | `upp import bad.toml` | Error reported, no changes |

### Requirement: Config Validation

The system MUST validate config on load. Validation MUST check: TOML syntax, required fields, platform compatibility of tools, and custom tool format.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Valid config | Correct syntax and fields | Load | Success |
| Unknown tool | Config references non-existent tool | Load | Warning, tool ignored |
| Platform mismatch | Linux-only tool enabled on macOS | Load | Warning, tool disabled |
