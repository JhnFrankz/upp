# Design: Hermetic CustomAdapter Execution & Privileges Consistency

## Technical Approach

`internal/adapters/custom.go` currently executes real subshells during unit tests, causing interactive `sudo` in `TestCustomAdapter_Privileges` to hang for 10 minutes and requiring Windows `t.Skip` skips across multiple test cases. Additionally, `CustomAdapter.Update(dryRun=true)` fails to populate `Result.Privileges`.

This design introduces:
1. Package-level function seams (`shellExecWithTimeoutFn`, `lookPathFn`) in `internal/adapters/exec.go`, matching `internal/adapters/official/helper.go`.
2. Upfront evaluation of `detectPrivileges(c.command)` in `CustomAdapter.Update` for both dry-run and live execution.
3. Fail-closed pre-execution detection in `Check()` and `Update()` when the base binary is absent from PATH.
4. An `exec_mock_test.go` test harness (`execFakes`, `setExecFakes` with `t.Cleanup`) and refactored `custom_test.go` removing all subprocess spawns and Windows skips.

## Architecture Decisions

### Decision D1: Package Seam vs Interface Injection

**Choice**: Package-level function pointers (`shellExecWithTimeoutFn`, `lookPathFn`) in `internal/adapters`, defaulting to `defaultShellExecWithTimeout` and `exec.LookPath`.
**Alternatives**: Injecting an executor interface into `NewCustomAdapter` or `CustomAdapter` struct.
**Rationale**: Preserves the public `NewCustomAdapter` signature and the `adapters.Adapter` interface contract. Mirrors the established pattern in `internal/adapters/official/helper.go` and guarantees 100% test hermeticity via `setExecFakes`.

### Decision D2: Privileges Evaluation in Dry-Run

**Choice**: Compute `privileges := detectPrivileges(c.command)` at the start of `CustomAdapter.Update` and attach to `Result.Privileges` across both `dryRun=true` and `dryRun=false` branches.
**Alternatives**: Compute privileges only on live execution or defer to caller.
**Rationale**: Ensures structural consistency in `Result` and enables CLI audit / dry-run previews to inspect elevated privilege requirements without executing commands.

### Decision D3: Fail-Closed on Missing Binary

**Choice**: `Check()` (when `checkCmd != ""`) and `Update()` verify `c.Detect()` via `lookPathFn` before invoking the platform shell, returning a structured error if missing.
**Alternatives**: Rely exclusively on shell non-zero exit codes.
**Rationale**: Prevents launching hung or broken subshells when a binary does not exist on PATH, providing immediate, predictable failure feedback.

### Decision D4: Hermetic Testing Harness & OS Portability

**Choice**: Add `exec_mock_test.go` with `execFakes` struct and `setExecFakes(t, fakes)` swapping seam variables with automatic `t.Cleanup` restoration. Remove all `runtime.GOOS == "windows"` skips in `custom_test.go`.
**Alternatives**: OS conditional compilation or external subprocess fixtures.
**Rationale**: Eliminates cross-platform divergence, prevents test hangs on privileged commands (`sudo`), and reduces `internal/adapters` unit test duration to < 1s.

## Data Flow

```
Update(dryRun):
  1. detectPrivileges(c.command) ──────────┐
  2. Detect() via lookPathFn               │
     ├─ [false] ──▶ return Result{Success: false, Error: missingBinaryErr, Privileges}
     └─ [true]                             ▼
         ├─ [dryRun=true]  ──▶ return Result{Success: true, Before/After, Privileges}
         └─ [dryRun=false] ──▶ shellExecWithTimeoutFn(c.command, UpdateTimeout)
                                ├─ [err != nil] ──▶ return Result{Success: false, Error: err, Privileges}
                                └─ [err == nil] ──▶ return Result{Success: true, Privileges}
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/adapters/exec.go` | Modify | Define `shellExecWithTimeoutFn`, `lookPathFn`, and `defaultShellExecWithTimeout` |
| `internal/adapters/custom.go` | Modify | Use seam variables; evaluate privileges upfront in `Update`; fail closed on missing binary |
| `internal/adapters/exec_mock_test.go` | Create | Define `fakeResult`, `execFakes`, and `setExecFakes(t, f)` helper with `t.Cleanup` |
| `internal/adapters/custom_test.go` | Modify | Convert all tests to use `setExecFakes`; test dry-run privileges; remove Windows `t.Skip` |
| `openspec/changes/upp-custom-adapter-hermeticity/design.md` | Create | Technical design artifact |

## Interfaces / Contracts

```go
// internal/adapters/exec.go
var (
    shellExecWithTimeoutFn = defaultShellExecWithTimeout
    lookPathFn             = exec.LookPath
)

func defaultShellExecWithTimeout(command string, timeout time.Duration) (string, error)

// internal/adapters/exec_mock_test.go
type fakeResult struct {
    stdout string
    err    error
}

type execFakes struct {
    shell    map[string]fakeResult
    lookPath map[string]bool
}

func setExecFakes(t *testing.T, f execFakes)
```

## Testing Strategy

| Layer | What to Test | Approach |
|-------|--------------|----------|
| Unit (Detection) | `Detect()` found vs not found | `setExecFakes` with `lookPath` map |
| Unit (Check) | `Check()` version parsing, missing binary, check command timeout | `setExecFakes` mapping `checkCmd` to fake output / `DeadlineExceeded` |
| Unit (Update) | Dry-run privileges, live update success/failure, missing binary | `setExecFakes` with mock `c.command`, assert `Result.Privileges` and `Result.Success` |
| Regression | `TestCustomAdapter_Privileges` non-interactive execution | `setExecFakes` faking `sudo` command without spawning process |
| Performance / CI | Cross-platform test execution on Linux/macOS/Windows | Run `go test -race ./internal/adapters/...` with zero skips and execution time < 1s |

## Threat Matrix

| Boundary | Applicability | Design response | Planned RED tests |
|---|---|---|---|
| Privilege Escalation Inspection | Applicable | `detectPrivileges` runs statically on `c.command` regardless of dry-run mode, ensuring `Privileges` metadata is preserved | `TestCustomAdapter_Update_DryRun_Privileges`: assert `Privileges=["sudo"]` on dry-run |
| Subprocess Execution Safety | Applicable | `Detect()` check prevents executing shell commands when base binary is missing | `TestCustomAdapter_Update_MissingBinary_FailClosed`: assert early structured error without shell call |
| Test Process Isolation | Applicable | `setExecFakes` swaps package seams and guarantees restoration via `t.Cleanup` | `TestCustomAdapter_Privileges`: assert no real `sudo` process is spawned |

## Migration / Rollout

No configuration or data migration required. Changes are internal to adapter execution and test infrastructure. Production fallback behavior is preserved via `defaultShellExecWithTimeout` and `exec.LookPath`. Rollback is a standard `git revert`.

## Open Questions

None.
