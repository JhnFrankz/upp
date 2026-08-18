# Delta for command-interface

## ADDED Requirements

### Requirement: Help Output Grouping

`upp --help` and `upp help` MUST group commands into labeled sections (e.g., "Tool Commands": `check`/`update`/`list`; "Config Commands": `init`/`export`/`import`; "Maintenance": `self-update`). The cobra `completion` built-in MUST be hidden from help output.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Grouped help | All commands registered | `upp --help` | Commands listed under labeled groups; `completion` absent |
| Help subcommand | All commands registered | `upp help` | Same grouped output as `--help` |
| Completion hidden | Root command built | `upp --help` | `completion` not listed among commands |
