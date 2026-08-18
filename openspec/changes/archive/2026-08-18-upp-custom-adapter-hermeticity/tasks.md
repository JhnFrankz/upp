# Tasks: Hermetic CustomAdapter Execution & Privileges Consistency

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~150–200 |
| 400-line budget risk | Low |
| Chained PRs recommended | No |
| Suggested split | Single PR |
| Delivery strategy | single-pr |
| Chain strategy | stacked-to-main |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: stacked-to-main
400-line budget risk: Low

Recommendation: Single PR (~150-200 lines). Changes are self-contained in `internal/adapters/`.

### Work Units

| Unit | Goal | Likely PR | Test command | Rollback boundary |
|------|------|-----------|--------------|-------------------|
| 1 | Seams & Mock Harness | Single PR | `go test ./internal/adapters/ -run TestExecFakes -count=1` | `exec.go`, `exec_mock_test.go` |
| 2 | Privileges & Fail-Closed Logic | Single PR | `go test ./internal/adapters/ -run TestCustomAdapter_ -count=1` | `custom.go`, `custom_test.go` |
| 3 | Hermetic Test Refactor & Verification | Single PR | `go test ./internal/adapters/... -count=1 -race` | `custom_test.go` |

## Phase 1: Test Seams & Mock Harness

- [x] 1.1 RED: `internal/adapters/exec_mock_test.go` — Add `TestExecFakes_Isolation` verifying `shellExecWithTimeoutFn` and `lookPathFn` can be intercepted via `setExecFakes` and restored on `t.Cleanup`.
- [x] 1.2 GREEN: `internal/adapters/exec.go` & `internal/adapters/exec_mock_test.go` — Declare `shellExecWithTimeoutFn`, `lookPathFn`, and `defaultShellExecWithTimeout` in `exec.go`; implement `fakeResult`, `execFakes`, and `setExecFakes(t, f)` helper with `t.Cleanup` in `exec_mock_test.go`.

## Phase 2: CustomAdapter Privileges & Fail-Closed Logic

- [x] 2.1 RED: `internal/adapters/custom_test.go` — Add `TestCustomAdapter_Update_DryRun_Privileges` asserting `Update(dryRun=true)` populates `Result.Privileges` with `detectPrivileges(c.command)`.
- [x] 2.2 RED: `internal/adapters/custom_test.go` — Add `TestCustomAdapter_Check_MissingBinary` and `TestCustomAdapter_Update_MissingBinary` asserting fail-closed structured errors when `Detect() == false` without executing subshells.
- [x] 2.3 GREEN: `internal/adapters/custom.go` — Compute `privileges := detectPrivileges(c.command)` at start of `Update()`, returning `Result.Privileges` on dry-run; check `c.Detect()` via `lookPathFn` in `Check()` and `Update()` to fail closed on missing binary; delegate execution to `shellExecWithTimeoutFn`.

## Phase 3: Hermetic Test Suite Refactor

- [x] 3.1 REFACTOR: `internal/adapters/custom_test.go` — Update `TestCustomAdapter_Privileges` to use `setExecFakes` without executing real `sudo` subprocess, eliminating 10-minute hang.
- [x] 3.2 REFACTOR: `internal/adapters/custom_test.go` — Refactor `TestCustomAdapter_Update_Execute`, `TestCustomAdapter_Update_Failure`, `TestShellExec`, `TestShellExec_UpdateTimeoutKills`, `TestCustomAdapter_Check_CheckTimeoutKills`, and `TestCustomAdapter_Detect_WithRealCommand` to use `setExecFakes`, removing all `runtime.GOOS == "windows"` skips.

## Phase 4: Full Suite Verification

- [x] 4.1 VERIFY: Run `go test ./internal/adapters/... -count=1 -race` ensuring tests pass in < 1s with 0 hangs or skips.
- [x] 4.2 VERIFY: Run `go test ./... -count=1 -race` for workspace suite.
- [x] 4.3 VERIFY: Run `go vet ./...` and `gofmt -s -w internal/adapters/`.
- [x] 4.4 VERIFY: Run `bash scripts/smoke-test.sh --skip-build`.
