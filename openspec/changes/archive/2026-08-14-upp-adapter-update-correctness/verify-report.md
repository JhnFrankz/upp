```yaml
schema: gentle-ai.verify-result/v1
evidence_revision: sha256:a78c2cdfca4502e47438b69e5caf926fc6eaf80b16c103632d5cd1c646ecee30
verdict: pass
blockers: 0
critical_findings: 0
requirements: 3/3
scenarios: 12/12
test_command: go test ./... -count=1
test_exit_code: 0
test_output_hash: sha256:41c7c627f3f3de3bf4bb381b16f425d8755fc447375a1d3ba8255a7c0da13eba
build_command: go build -o upp ./cmd/upp && bash scripts/smoke-test.sh --skip-build
build_exit_code: 0
build_output_hash: sha256:3ead4b5007a765fa80184778c1e057ae1b8e79f56e8b4e8bd6597d1fd066f668
```

# Verification Report

**Change**: upp-adapter-update-correctness
**Version**: N/A (delta specs, not yet archived)
**Mode**: Strict TDD (RED first, minimal GREEN; runner `go test ./... -count=1`)
**Evidence revision**: main @ e758d409 (both PRs #38 timeouts+arch, #39 gating+seam merged)

## Completeness

| Metric | Value |
|--------|-------|
| Tasks total | 19 |
| Tasks complete | 19 |
| Tasks incomplete | 0 |

Task checkbox inventory verified by reading tasks.md: Phase 1 (1.1–1.7, 7/7 `[x]`), Phase 2 (2.1–2.2, 2/2 `[x]`), Phase 3 (3.1–3.5, 5/5 `[x]`), Phase 4 (4.1–4.5, 5/5 `[x]`). Apply-progress batches sum to 7+2+5+5 = 19. All phases complete — full verification run.

## Build & Tests Execution

All gates executed fresh on current main (HEAD e758d409, both PRs merged):

| Gate | Command | Result | Exit | Output hash (sha256) |
|------|---------|--------|------|----------------------|
| Full suite | `go test ./... -count=1` | 9 Go packages, 8 with test files `ok` (cmd/upp no test files; cli 36.865s) | 0 | `41c7c627f3f3de3bf4bb381b16f425d8755fc447375a1d3ba8255a7c0da13eba` |
| Race | `go test ./... -count=1 -race` | 8 packages `ok` (cli 37.620s) | 0 | `d727738066f861ca91914289785b3ee5560fe6ae58bb1b2f55147538533d8f4b` |
| Vet | `go vet ./...` | clean, no output | 0 | — |
| gofmt | `gofmt -l internal/` | empty | 0 | — |
| Smoke (fresh build) | `go build -o upp ./cmd/upp && bash scripts/smoke-test.sh --skip-build` | **23 passed, 0 failed, 23 total** | 0 | `3ead4b5007a765fa80184778c1e057ae1b8e79f56e8b4e8bd6597d1fd066f668` |
| Group-kill regression | `go test ./internal/adapters/official/ -run GroupKill -count=1 -v` | `TestRunCmd_GroupKillProvesGrandchildrenDie` PASS (0.46s) | 0 | — |
| Focused gating/timeout | `go test ./internal/cli/ -run 'TestRunUpdate|TestTimeoutErr' -count=1 -v` | 3 tests, 10 subtests PASS | 0 | — |

## Spec Compliance Matrix

Authoritative counts from the retrieved delta spec: **3 requirements / 12 scenarios** (Update Gating 6, Subprocess Timeouts 4, Go Adapter Architecture 2).

### Requirement: Update Gating

| Scenario | Test (passed at runtime) | Implementation evidence | Result |
|----------|--------------------------|-------------------------|--------|
| Official update available | `update_test.go > TestRunUpdate_GatingMatrix/gated_dynamic_apt_with_update_available_runs_update` (wantUpdated=true, StatusUpdated) | update.go:183 gate — `TrustOfficial && gatedOfficialAdapters[info.ID] && !UpdateAvailable`; true → falls through to `Update(false)` at update.go:193 | ✅ COMPLIANT |
| Official no update | `TestRunUpdate_GatingMatrix/gated_dynamic_apt_without_update_is_reported_current_and_update_is_skipped` (wantUpdated=false, StatusCurrent) | update.go:183-190 — gate → `StatusCurrent{Version}`, continue | ✅ COMPLIANT |
| Stub official exempt | `TestRunUpdate_GatingMatrix/official_stub_brew_exempt:_update_still_runs_without_update_available` (wantUpdated=true) | update.go:48-53 — `gatedOfficialAdapters` contains only apt/npm/nvm/pnpm; brew (and bun/docker/gh/go/opencode by same set membership) not gated | ✅ COMPLIANT |
| Custom exempt | `TestRunUpdate_GatingMatrix/custom_trusted_exempt...` + `custom_untrusted_exempt...` (both wantUpdated=true) | update.go:183 — gate requires `TrustOfficial`; custom adapters (TrustCustomTrusted/Untrusted) never gated | ✅ COMPLIANT |
| winget/scoop exempt | `TestRunUpdate_GatingMatrix/winget_exempt:_update_still_runs_without_update_available` (wantUpdated=true) | update.go:48-53 — winget/scoop absent from gated set | ✅ COMPLIANT |
| Dynamic detection | `TestRunUpdate_GatingMatrix/gated_dynamic_apt_without_update_is_reported_current_and_update_is_skipped` (apt false → skipped) | update.go:183-190 — apt false → skipped | ✅ COMPLIANT |

### Requirement: Subprocess Timeouts

| Scenario | Test (passed at runtime) | Implementation evidence | Result |
|----------|--------------------------|-------------------------|--------|
| Update completes in time | `TestRunUpdate_GatingMatrix` success rows (StatusUpdated) + `TestUpdate_TimeoutErrorPropagates` non-timeout branches | helper.go:24-26 `runCmdFn` bounded by `UpdateTimeout` (10m); success returns result | ✅ COMPLIANT |
| Update times out | `TestUpdate_TimeoutErrorPropagates` (apt/brew update → DeadlineExceeded, `errors.Is`), `TestRunCmd_UpdateTimeoutKills` (real `sleep 2` killed at 100ms), `TestTimeoutErr/update` (message contains "brew update timed out after 10m0s") | helper.go:28-66 — CommandContext+WithTimeout, process-group SIGKILL; custom.go:125-127 `shellExec`→UpdateTimeout; update.go:198 + 218 `timeoutErr(name,"update",err)`; loop continues (update.go:200-201) | ✅ COMPLIANT |
| Check times out | `TestRunCmdArgs_CheckTimeoutKills` (real kill), `TestRunUpdate_CheckTimeoutStructuredError` (brew check timeout → "Failed: brew", npm **still updated**), `TestTimeoutErr/check` | helper.go:67-84 `runCmdArgsFn` bounded by `CheckTimeout` (30s); custom.go:53 `shellExecWithTimeout(c.checkCmd, CheckTimeout)`; update.go:119 + check.go:94 `timeoutErr(name,"check",err)`; run continues | ✅ COMPLIANT |
| Custom times out | `TestShellExec_UpdateTimeoutKills` (real `sleep 2` at 100ms), `TestCustomAdapter_Check_CheckTimeoutKills` | custom.go:134-174 `shellExecWithTimeout` — CommandContext, Setpgid, group SIGKILL, `WaitDelay=5s` (custom.go:151); update error path update.go:198 | ✅ COMPLIANT |

### Requirement: Go Adapter Architecture

| Scenario | Test (passed at runtime) | Implementation evidence | Result |
|----------|--------------------------|-------------------------|--------|
| amd64 host | `go_arch_test.go > TestGoTarballURL/amd64` → `go*linux-amd64.tar.gz` | go.go:114-116 `goTarballURL(goarch)` interpolates arch; go.go:57 passes `runtime.GOARCH` | ✅ COMPLIANT |
| arm64 host | `TestGoTarballURL/arm64` → `go*linux-arm64.tar.gz` | go.go:114-116 + go.go:57 `runtime.GOARCH`; no hardcoded arch anywhere in go.go (grep: 0 `amd64` literals outside test) | ✅ COMPLIANT |

**Compliance summary**: 12/12 scenarios compliant — every covering test passed at runtime.

## Correctness (Static Evidence)

| Check | Status | Notes |
|-------|--------|-------|
| Every `check()`/`update()` subprocess bounded | ✅ | grep `exec.Command(` in production code: 0 matches; all 5 production sites use `exec.CommandContext` (custom.go:142,144; helper.go:32,34,73). Test-harness `exec.Command` only (custom_test.go:251 fixture, timeout_test.go:124 pgrep probe) |
| Distinct check/update timeouts | ✅ | timeouts.go:8 `CheckTimeout=30s`, :11 `UpdateTimeout=10m`; helper.go:25 runCmdFn→UpdateTimeout, :68 runCmdArgsFn→CheckTimeout; custom.go:53 Check→CheckTimeout, :126 shellExec→UpdateTimeout |
| Process-group kill + WaitDelay | ✅ | helper.go:38 Setpgid, :59 `syscall.Kill(-pid, SIGKILL)`, :43 `WaitDelay=5s`; custom.go:146,167,151 identical; proven live by `TestRunCmd_GroupKillProvesGrandchildrenDie` (pgrep: no survivor) |
| Structured timeout error | ✅ | update.go:242-251 `timeoutErr(name,op,err)` — `%w` preserves `errors.Is`; wired at update.go:119 (check), :198 (update err), :218 (update failure result), check.go:94 |
| Run continues after timeout | ✅ | update.go:121-123, 200-201, 221 `continue`; flow-proven by `TestRunUpdate_CheckTimeoutStructuredError` (npm updated after brew check timeout) |
| goTarballURL no hardcoded arch | ✅ | go.go:114-116 arch parameterized; sole caller go.go:57 uses `runtime.GOARCH` |
| Gated set exact | ✅ | update.go:48-53 — apt/npm/nvm/pnpm only; stubs brew/bun/docker/gh/go/opencode, winget/scoop, and custom exempt by predicate (TrustOfficial + set membership) |

## Coherence (Design)

| Decision | Followed? | Notes |
|----------|-----------|-------|
| D3 — `timeoutErr(name, op, err)` structured message | ✅ Yes | update.go:242-251; message shape `"%s %s timed out after %s"` proven by TestTimeoutErr; `%w` chain keeps `errors.Is` |
| D4 — gate official updates on check availability | ✅ Yes | update.go:183 predicate + placement between confirm (161-176) and `Update(false)` (193); gated set = apt/npm/nvm/pnpm (corrected in PR #39 review — see delta spec; design.md pre-correction predicate had no set) |
| D5 — `updateDeps` seam (mirrors check.go:38-40, zero→prod default) | ✅ Yes | update.go:39-41 struct, :65-67 zero-value default `buildAdapterList`, :27 call site `runUpdate(gf, uf, updateDeps{})` |
| PR-1 design — `goTarballURL(goarch)` + `runtime.GOARCH` | ✅ Yes | go.go:114-116, :57 |

Documented deviations (apply-progress "Deviations from Design"): 2, both non-structural, neither breaks a spec scenario — (1) gating matrix asserts summary labels + fake `updated` flag instead of per-tool icon lines (`runUpdate` never calls `ToolLine`; quiet mode suppresses them); (2) check.go:94 `timeoutErr` wiring has no direct unit test because `runCheck` has no adapter-list seam (design adds one only to `runUpdate`) — helper fully tested via `TestTimeoutErr` and the identical runUpdate check-error site is flow-tested.

## TDD Compliance (Strict TDD)

| Check | Result | Details |
|-------|--------|---------|
| RED evidence | ✅ | apply-progress: undefined-symbol compile RED (3.1/3.2), behavioral RED (`Update called = true, want false` ×2, structured-error message absent) |
| GREEN confirmed | ✅ | All gates re-run fresh at verify time: full suite, race, vet, gofmt, smoke 23/23, GroupKill, focused gating/timeout 10/10 subtests |
| Triangulation | ✅ | 6-row gating matrix (official T/F, brew stub, winget, custom trusted/untrusted), 3-case TestTimeoutErr, check-timeout flow test, 2 real-kill tests, group-kill survivor probe |
| Safety net | ✅ | apply-progress: 8/8 cli tests pre-change; `TestProbe_TrustedLowRisk_Executes` regression (4.1) re-run green (custom exempt) |

## Issues Found

**CRITICAL**: None

**WARNING**: None

**SUGGESTION** (recorded follow-ups):
1. `upp check` never exits non-zero on check failures — `runCheck` appends the failed result and returns nil unconditionally (check.go:114-116). Pre-existing pattern, out of this change's spec scope; a future change should map failures to exit codes.
2. `timeoutErr` reads `adapters.CheckTimeout`/`UpdateTimeout` at error-mapping time (update.go:246-248), so the reported limit reflects the var value when the error is mapped, not the value bound when the context was created. Current tests override the vars consistently (set before exec, restored via t.Cleanup), so no false result; a future refactor could capture the limit at context creation.
3. `gatedOfficialAdapters` is a hardcoded ID list (update.go:48-53), not derived from adapter metadata — an official adapter that gains real update detection later must be added manually or it silently becomes exempt.

## Verdict

**PASS** — All 19/19 tasks complete; 3/3 requirements and 12/12 scenarios compliant with passing covering tests on fresh execution (full suite exit 0, race exit 0, vet clean, gofmt clean, smoke 23/23 against a freshly built binary, GroupKill survivor probe green, focused gating/timeout 10/10). Both design deviations documented and non-structural; 3 suggestion-level follow-ups, no blockers, no CRITICAL findings, no WARNINGs.
