# Proposal: Built-in `upp uninstall` Command

## Intent
`upp` is distributed as a single self-contained binary, but currently lacks a native uninstallation command. Adding `upp uninstall` under the `Maintenance` command group allows users to safely remove the `upp` binary, timestamped self-update backups (`upp.backup.*`), and optionally purge user configuration (`~/.config/upp` or `%APPDATA%\upp`) with explicit confirmation gates and zero-sudo security guarantees.

## Scope
### In Scope
- Add `upp uninstall` subcommand under the `Maintenance` command group.
- Detect current running executable path and locate any adjacent self-update backups (`{binary}.backup.*`).
- Interactive confirmation prompt in TTY by default before executing destructive removal.
- `--purge` flag to delete the user configuration directory (`~/.config/upp` or `%APPDATA%\upp`).
- `-y` / `--yes` flag to bypass the confirmation prompt in scripted environments.
- `-n` / `--dry-run` flag to preview planned deletions without modifying the filesystem.
- Zero-sudo security policy: fail closed with clear diagnostics if target directories/files are not writable.
- Safe cross-platform binary removal on Linux, macOS, and Windows.
- Non-TTY / `--ci` protection: reject unconfirmed execution in non-interactive sessions unless `-y`/`--yes` is explicitly supplied.

### Out of Scope
- Uninstalling third-party development tools or package managers managed by `upp` (e.g. `brew`, `apt`, `npm packages`).
- Adding system daemon/service uninstallation (none exist).

## Capabilities
### New Capabilities
- `uninstall`: Defines the end-to-end binary and configuration removal lifecycle, including backup file discovery, permissions verification, dry-run simulation, and atomic deletion order.

### Modified Capabilities
- `command-interface`: Register `uninstall` subcommand under the `Maintenance` help group; define flag semantics (`--purge`, `-y`/`--yes`, `-n`/`--dry-run`).
- `ux-patterns`: Standardize confirmation prompts, preview outputs, and completion summaries for uninstallation.
- `security-model`: Classify `uninstall` and `--purge` within the trust matrix and enforce fail-closed zero-sudo permission checks.

## Approach
1. **Lifecycle Package (`internal/uninstall`)**: Implement core removal functions:
   - Resolve executable location (`os.Executable()`, resolving symlinks).
   - Discover adjacent self-update backups (`{binary}.backup.*`).
   - Check directory writability before attempting removal.
   - Remove backups, remove binary, and if `--purge` requested, remove configuration directory (`config.Dir()`).
   - Support dry-run returning the list of candidate paths without deletion.
2. **CLI Subcommand (`internal/cli/uninstall.go`)**: Implement `newUninstallCmd` with `--purge`, `-y`/`--yes`, and `-n`/`--dry-run`.
3. **Interactive Confirmation**: Prompt in TTY displaying paths to be removed; abort cleanly on decline (exit 0).
4. **Help Grouping & Registration**: Wire into root command under the `Maintenance` section alongside `init` and `self-update`.
5. **Hermetic Testing**: Comprehensive unit & integration tests covering dry-run, happy path removal, `--purge` config deletion, unwritable directory errors, and non-TTY gates.

## Affected Areas
| Area | Impact | Description |
|------|--------|-------------|
| `internal/uninstall/` | New | Binary & config removal logic, backup discovery, writability preflights |
| `internal/cli/uninstall.go` | New | Cobra subcommand wiring, flag parsing, confirmation prompt |
| `internal/cli/root.go` | Modified | Register `uninstall` in `Maintenance` command group |
| `internal/output/` | Modified | Add render methods for uninstall preview, confirmation, and summary |
| `cmd/upp/main.go` | Modified | Register uninstall subcommand if necessary |
| `openspec/specs/` | Modified | Add `uninstall` domain and update delta specs |

## Risks
| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Executable locked while running on Windows | Med | Use platform-aware unlink or rename-to-temp before delete on Windows |
| Unintended config deletion without `--purge` | Low | Config deletion requires explicit `--purge` flag or separate confirmation |
| Permissions error mid-removal leaves partial state | Low | Preflight directory writability check before unlinking files |
| Accidental removal in headless scripts | Low | Require explicit `-y`/`--yes` when stdin is non-TTY |

## Rollback Plan
Revert commit adding `internal/uninstall/` and `internal/cli/uninstall.go`.

## Success Criteria
- [ ] `upp uninstall` removes running binary and adjacent `upp.backup.*` files
- [ ] `upp uninstall --purge` removes binary, backups, and `~/.config/upp` directory
- [ ] `upp uninstall -n` / `--dry-run` reports planned deletions without deleting files
- [ ] `upp uninstall -y` executes non-interactively without prompt
- [ ] Non-TTY execution without `-y` fails closed with a clear error
- [ ] Unwritable target directory fails closed with an actionable message (never calls sudo)
- [ ] `upp --help` shows `uninstall` under `Maintenance` group
- [ ] Full test suite passes with race detector (`go test -race ./...`) and linting (`make lint`)
