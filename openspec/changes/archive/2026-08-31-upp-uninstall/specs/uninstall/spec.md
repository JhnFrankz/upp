# Uninstall Specification

## Purpose
Defines the binary, backup, and configuration removal lifecycle for `upp`, ensuring safe, atomic, zero-sudo uninstallation across all supported platforms.

## Requirements

### Requirement: Binary and Backup Discovery
The system MUST locate the running `upp` executable path (resolving any symlinks) and search for any adjacent self-update backup files matching the pattern `{binary}.backup.*` in the same directory.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Running binary only | Executable at `/usr/local/bin/upp`, no backups | Discovery runs | Identifies `/usr/local/bin/upp` as the sole deletion candidate |
| Binary with backups | `/usr/local/bin/upp` and `/usr/local/bin/upp.backup.20260831` | Discovery runs | Identifies both binary and backup files |
| Symlink resolution | `/usr/bin/upp` -> `/opt/upp/bin/upp` | Discovery runs | Resolves target to `/opt/upp/bin/upp` and checks `/opt/upp/bin/` for backups |

### Requirement: Removal Execution and Order
The uninstallation execution MUST proceed in the following order:
1. Preflight writability check on the binary directory and configuration directory (if `--purge` is set). If not writable, execution MUST abort with a non-zero exit code before deleting any file.
2. Remove all discovered backup files (`{binary}.backup.*`).
3. Remove the primary executable binary (`upp`).
4. If `--purge` is enabled, remove the configuration directory (`~/.config/upp` or `%APPDATA%\upp`).
5. On Windows, where a running executable cannot be deleted directly while running, the system MUST rename the executable to a temporary file before unlinking or stage deletion cleanly.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Standard uninstall | User confirms uninstall | Execution | Removes backups, removes binary; leaves config intact |
| Purge uninstall | User passes `--purge` and confirms | Execution | Removes backups, removes binary, removes config directory |
| Unwritable directory | Directory `/usr/local/bin` not writable | Execution | Aborts before unlinking any file with error: "binary directory is not writable" |

### Requirement: Dry-Run Mode
When `-n` or `--dry-run` is provided, the system MUST list all candidate files and directories that would be removed without deleting or modifying any file on disk.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Dry run standard | `upp uninstall -n` | Execution | Outputs list of binary and backup paths; no files deleted |
| Dry run purge | `upp uninstall --dry-run --purge` | Execution | Outputs list of binary, backups, and config dir; no files deleted |

### Requirement: Interactive Confirmation & Non-Interactive Safety
In interactive TTY mode without `-y`/`--yes`, `upp uninstall` MUST display the target removal paths and require explicit user confirmation (`y/N`) before proceeding. Declining the prompt MUST exit with status 0 without deleting anything.

In non-TTY environments (or under `--ci`), `upp uninstall` without `-y`/`--yes` MUST fail closed with a non-zero exit code, refusing to execute unconfirmed deletions.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| TTY confirmed | Interactive terminal | User inputs 'y' | Files removed, exit 0 |
| TTY declined | Interactive terminal | User inputs 'n' | Aborts with "Uninstallation cancelled", exit 0 |
| Non-TTY without `--yes` | Non-interactive stdin | `upp uninstall` | Fails closed with error "uninstallation requires interactive confirmation or -y/--yes", exit 1 |
| Non-TTY with `--yes` | Non-interactive stdin | `upp uninstall -y` | Removes files without prompt, exit 0 |
