# Proposal: Hermetic CustomAdapter Execution & Privileges Consistency

## Intent

Make `CustomAdapter` execution completely hermetic in unit tests, eliminate the 10-minute hang on interactive `sudo` in `TestCustomAdapter_Privileges`, and guarantee consistent `Privileges` population across both dry-run and live updates.

## Scope

### In Scope
- Add mockable test seams (`shellExecWithTimeoutFn`, `lookPathFn`, `setExecFakes`) in `internal/adapters`.
- Refactor `custom_test.go` to use injected fakes, eliminating real subprocess execution and OS-specific skips (`runtime.GOOS == "windows"`).
- Populate `Result.Privileges` in `CustomAdapter.Update(dryRun=true)` via `detectPrivileges(c.command)` before dry-run return.

### Out of Scope
- Modifying official adapters (already hermetic via `internal/adapters/official/`).
- Changes to CLI commands or configuration parsing.

## Capabilities

### New Capabilities
None.

### Modified Capabilities
- `tool-adapter`: Clarify that `CustomAdapter` populates `Privileges` for both dry-run (`dryRun=true`) and live (`dryRun=false`) update executions.

## Approach

1. **Package Seam**: Introduce `shellExecWithTimeoutFn` and `lookPathFn` package variables in `internal/adapters`, defaulting to production implementations (`defaultShellExecWithTimeout`, `exec.LookPath`), mirroring `internal/adapters/official/helper.go`.
2. **Privileges Consistency**: In `CustomAdapter.Update`, evaluate `privileges := detectPrivileges(c.command)` upfront and populate `Result.Privileges` on dry-run returns.
3. **Hermetic Test Suite**: Add `setExecFakes` harness with `t.Cleanup` in `exec_mock_test.go` and refactor `custom_test.go` to use table-driven fakes without spawning host processes or skipping Windows.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/adapters/custom.go` | Modified | Delegate to `shellExecWithTimeoutFn`/`lookPathFn`; populate dry-run `Privileges` |
| `internal/adapters/exec.go` | New / Modified | Seam variables and default runner |
| `internal/adapters/exec_mock_test.go` | New | `setExecFakes` test helper with cleanup |
| `internal/adapters/custom_test.go` | Modified | Convert tests to hermetic fakes; remove OS skips |
| `openspec/specs/tool-adapter/spec.md` | Modified | Update spec for custom adapter dry-run privileges |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Seam leaks across tests | Low | Enforce restoration via `t.Cleanup` in `setExecFakes` |
| Production regression | Low | Production functions default to identical system exec calls |

## Rollback Plan

`git revert` the PR or discard changes in `internal/adapters/` and `openspec/`.

## Dependencies

None.

## Success Criteria

- [ ] `go test -race ./internal/adapters/...` runs in < 1s with zero hangs.
- [ ] `CustomAdapter.Update(true)` returns `Privileges` matching `detectPrivileges(cmd)`.
- [ ] All unit tests in `internal/adapters` pass on Linux, macOS, and Windows without `t.Skip`.
