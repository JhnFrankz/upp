# Command Interface Specification (Delta)

## Purpose
Extends the command interface to include the `uninstall` subcommand under the `Maintenance` command group.

## Requirements

### Requirement: Command Structure
The system MUST support `uninstall` alongside `list`, `update`, `init`, and `self-update`.

| Command | Description | Interactive | Modifies System |
|---------|-------------|-------------|-----------------|
| `uninstall` | Remove the upp binary and optional configuration | Yes (confirm) | Yes (deletes binary/config) |

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| `uninstall` help | User runs `upp uninstall --help` | Execution | Displays usage, short description, and flags (`--purge`, `-y`/`--yes`, `-n`/`--dry-run`) |
| `uninstall` bare | User runs `upp uninstall` | Execution | Prompts for confirmation before removing binary and backups |
| `uninstall --purge` | User runs `upp uninstall --purge` | Execution | Prompts for confirmation before removing binary, backups, and config directory |

### Requirement: Help Output Grouping
`upp --help` and `upp help` MUST group commands into exactly two labeled sections:
1. `Commands`: `list`, `update`
2. `Maintenance`: `init`, `self-update`, `uninstall`

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Maintenance group | User runs `upp --help` | Execution | `init`, `self-update`, and `uninstall` appear under `Maintenance` |
