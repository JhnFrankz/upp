# Uninstall Specification

## Purpose
Defines the target discovery, deletion lifecycle, and security model for `upp uninstall`, ensuring a complete, clean, cross-platform uninstallation under a Zero-Sudo policy.

## Requirements

### Requirement: Binary and Backup Discovery
The system MUST locate the running `upp` executable path (resolving any symlinks via `filepath.EvalSymlinks`) and search for any historical backup binaries matching `{binary}.backup.*` in the same binary directory.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Running binary only | Executable at `/home/user/.local/bin/upp`, no backups | Target discovery runs | Identifies `/home/user/.local/bin/upp` as the sole binary target |
| Binary with backups | `/home/user/.local/bin/upp` and `/home/user/.local/bin/upp.backup.20260812` | Target discovery runs | Identifies both binary and backup files |
| Symlink resolution | `/usr/bin/upp` -> `/opt/upp/bin/upp` | Target discovery runs | Resolves canonical target to `/opt/upp/bin/upp` and checks `/opt/upp/bin/` for backups |

### Requirement: Configuration and Cache Discovery
The system MUST discover the platform-appropriate user configuration directory (`~/.config/upp` on Linux/macOS, `%APPDATA%\upp` on Windows) and cache directory (`~/.cache/upp` on Linux, `~/Library/Caches/upp` on macOS, `%LOCALAPPDATA%\upp` on Windows).

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Standard Linux directories | Default environment | Discovery runs | Discovers `~/.config/upp` and `~/.cache/upp` |
| macOS directories | macOS environment | Discovery runs | Discovers `~/.config/upp` and `~/Library/Caches/upp` |
| Windows directories | Windows environment | Discovery runs | Discovers `%APPDATA%\upp` and `%LOCALAPPDATA%\upp` |

### Requirement: Zero-Sudo Best-Effort Deletion
`upp uninstall` MUST execute best-effort deletion across all discovered targets without ever escalating privileges or invoking `sudo`.
- For regular files (binary and backups), `os.Remove` MUST be used.
- For directories (configuration and cache), `os.RemoveAll` MUST be used.
- If all targets are removed successfully, the command MUST exit with status 0.
- If any target cannot be unlinked due to permission errors (e.g. system binary in `/usr/local/bin`), the command MUST emit a warning with the manual command (e.g. `sudo rm -rf <path>`) and return exit code 1.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Full user-space install | All targets owned by user | `upp uninstall` | Deletes binary, backups, config, and cache; exits 0 |
| Root-owned binary | Binary in `/usr/local/bin` (root), config in user home | `upp uninstall` | Deletes user config and cache, warns about `/usr/local/bin/upp` with `sudo rm -rf` command, exits 1 |
| Missing files | Config directory already deleted | `upp uninstall` | Deletes existing targets, skips non-existent ones gracefully, exits 0 |

### Requirement: Simulation Mode (`--dry-run`)
When `--dry-run` is provided, `upp uninstall` MUST list all candidate targets that would be removed without deleting or modifying any file on disk, and exit with status 0.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Dry-run preview | Binary, backups, config, and cache exist | `upp uninstall --dry-run` | Prints dry-run header and indented target list; no files deleted; exits 0 |
