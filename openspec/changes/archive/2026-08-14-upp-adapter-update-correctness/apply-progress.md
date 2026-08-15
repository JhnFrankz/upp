# Apply Progress: upp-adapter-update-correctness — SLICE 2 (gating + seam, PR 2)

## Status

- **Slice**: 2 of 2 (chained PRs, stacked-to-main). PR 1 (timeouts + go arch) merged on main. PR 2 = gating + seam + timeoutErr (Phase 3/4).
- **Delivery boundary**: tasks 3.1–3.5 + 4.1–4.5 ONLY. All 11 tasks of the change COMPLETE.
- **Mode**: Strict TDD (RED first, minimal GREEN). Runner: `go test ./... -count=1`.
- **All slice-2 gates PASS**: focused, full suite, race, vet, gofmt, smoke test (23/23).

## Task Status (cumulative, both slices)

### Phase 1/2 (slice 1, on main) — merged

- [x] 1.1 RED: `custom_test.go` — `UpdateTimeout`=100ms; `shellExec` `sleep 2` → `errors.Is(err, context.DeadlineExceeded)`
- [x] 1.2 RED: `custom_test.go` — Check honors `CheckTimeout` override → DeadlineExceeded
- [x] 1.3 GREEN: `timeouts.go` — `CheckTimeout=30s`, `UpdateTimeout=10m` vars
- [x] 1.4 GREEN: `custom.go` — `shellExecWithTimeout(command, timeout)` core; `shellExec` delegates with UpdateTimeout (signature kept); Check/Update per-op variant
- [x] 1.5 RED: `official/timeout_test.go` — seam fakes return DeadlineExceeded; errors `errors.Is`-detectable (setExecFakes)
- [x] 1.6 RED: `timeout_test.go` — kill-path `sleep 2`, 100ms → DeadlineExceeded
- [x] 1.7 GREEN: `official/helper.go` — seam bodies → CommandContext+WithTimeout (runCmdFn→UpdateTimeout, runCmdArgsFn→CheckTimeout); wrappers untouched
- [x] 2.1 RED: `official/go_arch_test.go` — table: amd64→`go*linux-amd64.tar.gz`; arm64→`go*linux-arm64.tar.gz`
- [x] 2.2 GREEN: `official/go.go` — `goTarballURL(goarch)` (design block); go.go:57 → `runtime.GOARCH`

### Phase 3 (slice 2, PR 2) — implemented

- [x] 3.1 RED: `internal/cli/update_test.go` — matrix via `updateDeps` fakes: official true→called; false→StatusCurrent, NOT called; custom false→called; winget/scoop→called; dynamic false→skipped
- [x] 3.2 RED: `update_test.go` — Check timeout → tool/op/limit error; other tools still update
- [x] 3.3 GREEN: `update.go` — `updateDeps` seam (mirrors check.go:38-40, zero→prod default); `runUpdate(gf, uf, deps)` + call sites
- [x] 3.4 GREEN: `update.go` — gate between confirm (152) and `Update(false)` (155): `TrustOfficial && !UpdateAvailable` → `StatusCurrent{Version}`, continue
- [x] 3.5 GREEN: `update.go` 91/156/177 + `check.go` 91 — `timeoutErr(name, op, err)`

### Phase 4 (slice 2) — all gates pass

- [x] 4.1 Regression: `TestProbe_TrustedLowRisk_Executes` (audit_probe_test.go:77-89) stays green — custom exempt
- [x] 4.2 `go test ./... -count=1`
- [x] 4.3 `go test ./... -race`
- [x] 4.4 `go vet ./...` + `gofmt -s -w`
- [x] 4.5 `bash scripts/smoke-test.sh --skip-build`

## TDD Cycle Evidence (slice 2)

| Task | Test File | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|------|-----------|-------|------------|-----|-------|-------------|----------|
| 3.1 | `internal/cli/update_test.go` | Unit (updateDeps fakes + captured stdout) | ✅ 8/8 cli tests pre-change | ✅ Written (compile fail: `undefined: updateDeps`, `too many arguments in call to runUpdate`) | ✅ Passed (after 3.3+3.4) | ✅ 6 cases (official T/F, custom trusted/untrusted, winget/scoop, dynamic) | ➖ None needed |
| 3.2 | `update_test.go` | Unit (fakes) | ✅ 8/8 | ✅ Written (same compile fail as 3.1) | ✅ Passed (after 3.5) | ✅ 3 cases in TestTimeoutErr (check, update, passthrough) + flow test | ➖ None needed |
| 3.3 | (proven by 3.1/3.2) | Unit | ✅ 8/8 | ✅ via 3.1/3.2 undefined-symbol RED | ✅ Seam in, matrix behaviorally RED (`Update called = true, want false`) | ✅ | ✅ Clean |
| 3.4 | (proven by 3.1) | Unit | ✅ 8/8 | ✅ via 3.1 behavioral RED | ✅ Matrix 6/6 PASS | ✅ 6 cases | ✅ Clean |
| 3.5 | `update_test.go` + sites | Unit | ✅ 8/8 | ✅ via 3.2 message assertion RED | ✅ TestTimeoutErr + flow test PASS | ✅ 3 cases | ✅ Clean |
| 4.1 | `audit_probe_test.go` (unchanged) | Integration (real subprocess) | ✅ 8/8 | N/A (regression) | ✅ PASS | N/A | ➖ None |

### RED evidence (exact)

Task 3.1/3.2 — `go test ./internal/cli/ -run 'TestRunUpdate' -count=1` after writing tests only:
```
# github.com/JhnFrankz/upp/internal/cli [github.com/JhnFrankz/upp/internal/cli.test]
internal/cli/update_test.go:53:10: undefined: updateDeps
internal/cli/update_test.go:59:66: too many arguments in call to runUpdate
	have (*GlobalFlags, *UpdateFlags, unknown type)
	want (*GlobalFlags, *UpdateFlags)
internal/cli/update_test.go:176:10: undefined: updateDeps
internal/cli/update_test.go:182:66: too many arguments in call to runUpdate
FAIL	github.com/JhnFrankz/upp/internal/cli [build failed]
```

Intermediate behavioral RED after 3.3 seam only (gate not yet implemented):
```
--- FAIL: TestRunUpdate_GatingMatrix (0.00s)
    --- FAIL: .../official_without_update_is_reported_current_and_update_is_skipped (0.00s)
        update_test.go:143: Update called = true, want false
    --- FAIL: .../dynamic_detection_without_update_is_skipped (0.00s)
        update_test.go:143: Update called = true, want false
--- FAIL: TestRunUpdate_CheckTimeoutStructuredError (0.00s)
    update_test.go:187: output lacks structured tool/op/limit timeout error for brew
```

### GREEN evidence (exact)

After 3.4 (gate) + 3.5 (timeoutErr):
```
--- PASS: TestRunUpdate_GatingMatrix (0.00s)
    --- PASS: TestRunUpdate_GatingMatrix/official_with_update_available_runs_update (0.00s)
    --- PASS: TestRunUpdate_GatingMatrix/official_without_update_is_reported_current_and_update_is_skipped (0.00s)
    --- PASS: TestRunUpdate_GatingMatrix/custom_trusted_exempt:_update_still_runs_without_update_available (0.00s)
    --- PASS: TestRunUpdate_GatingMatrix/custom_untrusted_exempt:_update_still_runs_without_update_available (0.00s)
    --- PASS: TestRunUpdate_GatingMatrix/winget_scoop_exempt:_update_always_runs (0.00s)
    --- PASS: TestRunUpdate_GatingMatrix/dynamic_detection_without_update_is_skipped (0.00s)
--- PASS: TestRunUpdate_CheckTimeoutStructuredError (0.00s)
--- PASS: TestTimeoutErr (0.00s)
    --- PASS: TestTimeoutErr/check (0.00s)
    --- PASS: TestTimeoutErr/update (0.00s)
    --- PASS: TestTimeoutErr/non-timeout_error_passes_through_unchanged (0.00s)
PASS
ok  	github.com/JhnFrankz/upp/internal/cli	0.004s
```

Note: the update flow's summary renders only tool names ("Failed: brew"), not error text — the structured message content is proven at the helper level (TestTimeoutErr) and the site mapping at the flow level (failed result + run continues). Pre-existing rendering limitation, out of scope.

## Work Unit Evidence (slice 2)

| Evidence | Required value |
|---|---|
| Focused test command and exact result | `go test ./internal/cli/ -run 'TestRunUpdate\|TestTimeoutErr' -count=1 -v` → 3 tests, 10 subtests PASS, exit 0 |
| Runtime harness command/scenario and exact result | `bash scripts/smoke-test.sh --skip-build` → 23 passed, 0 failed, 23 total, exit 0 (six stubs behavior unchanged: check returns UpdateAvailable=false; update run reports StatusCurrent per new gate — covered by unit matrix; smoke exercises real binary end-to-end) |
| Rollback boundary | Revert exactly: `internal/cli/update.go`, `internal/cli/check.go`, delete `internal/cli/update_test.go` → restores pre-slice behavior; no other production/test file touched |

## Deviations from Design

1. **Assertion-layer adjustment (not a design deviation)**: the gating matrix asserts the summary labels ("Updated: tool" / absence of "Updated:"/"Failed:") plus the fake's `updated` flag instead of per-tool icon lines — `runUpdate` never calls `ToolLine`, and quiet mode suppresses per-tool lines entirely, so icons are not rendered in this flow. Design's planned RED shape (matrix via updateDeps fakes + withCapturedStdout) preserved.
2. **check.go:91 timeoutErr wiring has no direct unit test**: `runCheck` has no adapter-list seam (design D5 adds one only to `runUpdate`), so the check-error site cannot be exercised with fakes without out-of-scope seam work; the helper is fully tested via TestTimeoutErr and the identical runUpdate check-error site is flow-tested. Wiring mirrors the tested site exactly.

Everything else matches design.md D3/D4/D5 and the Interfaces/Contracts blocks exactly (updateDeps struct shape, zero-value production default, gating predicate/placement, timeoutErr message shape).

## Files Changed (slice 2)

| File | Action | What Was Done |
|------|--------|---------------|
| `internal/cli/update.go` | Modified | `updateDeps` seam (zero value → buildAdapterList); `runUpdate(gf, uf, deps)`; caller passes `updateDeps{}`; gating block after confirm switch before `Update(false)`; `timeoutErr` helper + wiring at check-error (91), update-error, update-failure sites |
| `internal/cli/check.go` | Modified | `timeoutErr(info.Name, "check", err)` at check-error site (91) |
| `internal/cli/update_test.go` | Created | `fakeUpdateAdapter` (records Update calls), gating matrix (6 cases), check-timeout flow test, `TestTimeoutErr` (3 cases) |

## Gates (slice 2)

- `go test ./internal/cli/ -count=1` → PASS (34.2s)
- `go test ./internal/cli/ -run 'TestProbe_TrustedLowRisk_Executes' -count=1` → PASS (4.1 regression)
- `go test ./... -count=1` → PASS (all 8 packages; cli 34.1s)
- `go test ./... -race -count=1` → PASS (all 8 packages; cli 34.6s)
- `go vet ./...` → clean (exit 0)
- `gofmt -l internal/cli/` → clean (no output)
- `bash scripts/smoke-test.sh --skip-build` → 23 passed, 0 failed, exit 0

## Remaining

None — all 11 tasks complete. Next: sdd-verify.
