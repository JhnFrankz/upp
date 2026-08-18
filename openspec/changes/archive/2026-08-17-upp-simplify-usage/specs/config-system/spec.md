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
```

The `[settings]` section MUST NOT contain an output-language key: upp output is
English-only. The `settings.interactive` field MUST NOT exist in the config schema:
prompt behavior is driven by the security-model risk matrix and TTY detection, not
by config, and no CLI flag is added or removed. A `language` or `interactive` key in
an existing config file is ignored (unknown settings are tolerated for forward
compatibility) and MUST NOT be written by `upp init`, `upp export`, or `upp import`.
(Previously: the minimum structure listed `[settings] interactive = true`; the field was written but never read by production code — dead config removed as simplification, no behavior change.)

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Valid TOML | Config has valid TOML syntax | Config loaded | All fields parsed correctly |
| Invalid TOML | Malformed config file | Config loaded | Error message with line number, exit non-zero |
| Missing fields | Config has partial content | Config loaded | Missing fields use defaults |
| Stray interactive | Existing config has `interactive = false` | Config loaded | Key ignored; prompt behavior unchanged (risk matrix + TTY) |
| Init/export hygiene | Config loaded | `upp init`, `upp export`, `upp import` | `interactive` key never written to output |
