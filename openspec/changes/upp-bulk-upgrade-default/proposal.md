# Proposal: Default Manager-Group Bulk Updates

## Intent
Previously, owned tools (`gh`, `docker`, `go`) delegated `Update()` to their manager's self-only update, leaving packages un-upgraded unless users opted into `--manager <mgr>`. Making manager-group bulk package updates the default behavior in `upp update` ensures owned packages are actually upgraded during standard update runs while maintaining error isolation and risk controls.

## Scope
### In Scope
- Make manager-group package updates the default execution mode in `upp update`.
- Wire owned tool adapters (`gh`, `docker`, `go`) to delegate `Update()` to `owner.(PackageUpdater).UpdatePackage(pkg)` with resolved package names.
- Preserve standalone tool execution alongside manager groups.
- Enforce risk prompts for high-risk/sudo package commands in TTY and fail-closed in `--ci`.
- Granular per-tool selection within manager groups in TTY CheckboxSelector.
- Error isolation across sibling packages and standalone tools.

### Out of Scope
- Adding new package managers (e.g., dnf, pacman).
- Modifying manager self-update logic (remains self-only).
- Changing configuration schema.

## Capabilities
### New Capabilities
None

### Modified Capabilities
- `bulk-update`: Manager-group bulk update becomes default path instead of opt-in flag.
- `tool-ownership-model`: Owned tools delegate `Update()` to `PackageUpdater.UpdatePackage(pkg)`.
- `command-interface`: `upp update` runs manager groups by default without `--manager` flag.
- `ux-patterns`: CheckboxSelector and summaries reflect granular per-tool selections within manager groups.

## Approach
1. **Delegation Seam**: Update `gh`, `docker`, and `go` `Update()` methods to invoke `owner.(PackageUpdater).UpdatePackage(pkg)` using platform-resolved package names.
2. **Default Group Execution**: Refactor `cli/update.go` so `runUpdate` groups owned tools by manager, checks package availability via `PackageChecker`, and updates them via `PackageUpdater`.
3. **Risk & Safety**: Pass `EnforceRisk: true` on elevated commands, prompting in interactive TTY and aborting safely in `--ci`.
4. **Error Isolation**: Execute package updates in isolated per-tool blocks so single-package failures do not abort remaining tools.

## Affected Areas
| Area | Impact | Description |
|------|--------|-------------|
| `internal/adapters/official/` | Modified | Delegate owned tool updates (`gh`, `docker`, `go`) to `UpdatePackage` |
| `internal/cli/update.go` | Modified | Make manager group updates default in `runUpdate` |
| `internal/output/` | Modified | Support granular selection & summary rendering |
| `internal/cli/*_test.go` | Modified | Update CLI & integration test suites for default bulk |

## Risks
| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Sudo elevation prompts block headless CI | Med | Fail-closed in `--ci` mode when elevated risk detected |
| One package failure halts entire manager group | Med | Per-tool error boundary prevents group abort |
| Wrong package name executed with sudo | Low | Package names verified against parity test matrix |

## Rollback Plan
Revert changes in `internal/cli/update.go` and `internal/adapters/official/{gh,docker,go}.go` to restore opt-in `--manager` flag routing and self-only delegation.

## Success Criteria
- [ ] `upp update` upgrades owned packages (`gh`, `docker`, `go`) via manager by default
- [ ] Standalone tools continue to update via dedicated adapters
- [ ] TTY CheckboxSelector allows granular tool toggling within manager groups
- [ ] Single package update failure isolates without aborting siblings
- [ ] Sudo-heavy package updates fail-closed in `--ci` and prompt in TTY
- [ ] All unit and integration tests pass (`go test ./...`)
