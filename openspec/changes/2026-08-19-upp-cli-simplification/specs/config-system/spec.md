# Delta for config-system

## REMOVED Requirements

### Requirement: Export/Import

**Reason**: Serialized config export and import CLI subcommands and programmatic helpers (`Export`, `ExportToFile`, `ImportFromFile`) are removed. Configuration file backup, sharing, and version control are handled through standard filesystem operations (`cp`, `cat`, git) directly on `~/.config/upp/config.toml` (or `%APPDATA%/upp/config.toml`).

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
```

The `[settings]` section MUST NOT contain an output-language key: upp output is English-only. The `settings.interactive` field MUST NOT exist in the config schema: prompt behavior is driven by the security-model risk matrix and TTY detection, not by config, and no CLI flag is added or removed. A `language` or `interactive` key in an existing config file is ignored (unknown settings are tolerated for forward compatibility) and MUST NOT be written by `upp init`.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Valid TOML | Config has valid TOML syntax | Config loaded | All fields parsed correctly |
| Invalid TOML | Malformed config file | Config loaded | Error message with line number, exit non-zero |
| Missing fields | Config has partial content | Config loaded | Missing fields use defaults |
| Stray interactive | Existing config has `interactive = false` | Config loaded | Key ignored; prompt behavior unchanged (risk matrix + TTY) |
| Init hygiene | Config loaded | `upp init` | `interactive` key never written to output |

(Previously: the requirement referenced `upp export` and `upp import` writing output configs.)
