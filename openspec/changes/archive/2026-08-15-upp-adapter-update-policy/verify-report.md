```yaml
schema: gentle-ai.verify-result/v1
evidence_revision: sha256:e541af869028913e216db64592a5badc2f2b613b0736e25eab05224e07764d1e
verdict: pass
blockers: 0
critical_findings: 0
requirements: 2/2
scenarios: 12/12
test_command: go test ./... -count=1
test_exit_code: 0
test_output_hash: sha256:a4dd21c6071b82e1df981fa17d0a63cf8d384043ea74f0dff322edaa6ed91611
build_command: bash scripts/smoke-test.sh --skip-build
build_exit_code: 0
build_output_hash: sha256:cc64cd3e7dd67d45f6154354d0cd6a2e302aaf0d6c5318d46e4618c01a810fef
```

# Verification Report

**Change**: upp-adapter-update-policy
**Version**: N/A (delta specs, not yet archived)
**Mode**: Strict TDD (RED first, minimal GREEN; runner `go test ./... -count=1`)
**Evidence revision**: main @ 7f8ad023 (PR #45 enum+gate, PR #46 failure signal merged)

## Completeness

| Metric | Value |
|--------|-------|
| Tasks total | 19 |
| Tasks complete | 19 |
| Tasks incomplete | 0 |

Task checkbox inventory verified by reading tasks.md: Phase 1 (1.1–1.4, 4/4 `[x]`), Phase 2 (2.1–2.3, 3/3 `[x]`), Phase 3 (3.1–3.6, 6/6 `[x]`), Phase 4 (4.1–4.6, 6/6 `[x]`). Apply-progress batches sum to 4+3+6+6 = 19. All phases complete — full verification run.

## Build & Tests Execution

All gates executed fresh on current main (HEAD 7f8ad023, PRs #45 + #46 merged):

| Gate | Command | Result | Exit | Output hash (sha256) |
|------|---------|--------|------|----------------------|
| Full suite | `go test ./... -count=1` | 9 packages — 8 `ok` (adapters 0.285s, official 0.768s, cli 0.042s, config, output, platform, security, selfupdate 0.402s); cmd/upp no test files | 0 | `a4dd21c6071b82e1df981fa17d0a63cf8d384043ea74f0dff322edaa6ed91611` |
| Race | `go test ./... -count=1 -race` | 8 packages `ok` (official 1.774s, cli 1.124s) | 0 | `f6152d9654756cd726b01de4627c73c4422b0ba22bb38dc01a0d48c750687411` |
| Vet | `go vet ./...` | clean, no output | 0 | `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855` |
| gofmt | `gofmt -l internal/` | empty | 0 | `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855` |
| Smoke | `bash scripts/smoke-test.sh --skip-build` | **23 passed, 0 failed, 23 total** | 0 | `cc64cd3e7dd67d45f6154354d0cd6a2e302aaf0d6c5318d46e4618c01a810fef` |
| Focused new rows | `go test ./internal/adapters/official/ -run 'TestCheck|TestCommandOutputErr|TestShellOutputErr' -count=1` | **57/57 subtests PASS** (TestCheck 49 rows + TestCommandOutputErr 4 + TestShellOutputErr 4; 0 FAIL) | 0 | `14af88aba633496792d81a0574f96478c28d676424b1cb81f92b373a9f00e4da` |

## Spec Compliance Matrix

Authoritative counts from the retrieved delta spec: **2 requirements / 12 scenarios** (Update Gating 7, Check Failure Signal 5).

### Requirement: Update Gating (MODIFIED)

| Scenario | Test (passed at runtime) | Implementation evidence | Result |
|----------|--------------------------|-------------------------|--------|
| Official update available | `update_test.go > TestRunUpdate_GatingMatrix/gated_apt_with_update_available_runs_update` (wantUpdated=true, StatusUpdated) | update.go:171 gate — `info.UpdatePolicy == adapters.PolicyGated && !updateInfo.UpdateAvailable` → StatusCurrent; false falls through to `Update(false)` at update.go:181; PolicyGated declared at apt.go:106, npm.go:103, pnpm.go:117, nvm.go:123 | ✅ COMPLIANT |
| Official no update | `TestRunUpdate_GatingMatrix/gated_apt_without_update_is_reported_current_and_update_is_skipped` (wantUpdated=false, StatusCurrent) | update.go:171-178 — gate → `StatusCurrent{Version}`, continue; Update not called | ✅ COMPLIANT |
| Stub official exempt | `TestRunUpdate_GatingMatrix/always-update_brew_exempt:_update_still_runs_without_update_available` (wantUpdated=true) | PolicyAlwaysUpdate declared at brew.go:90, bun.go:88, docker.go:107, gh.go:107, go.go:108, opencode.go:85 — gate predicate only matches PolicyGated; stubs always update | ✅ COMPLIANT |
| Custom exempt | `TestRunUpdate_GatingMatrix/custom_trusted_exempt...` + `custom_untrusted_exempt...` (both wantUpdated=true) | custom.go:105 — `UpdatePolicy: PolicyAlwaysUpdate`; trust no longer participates in the gate (design D2) | ✅ COMPLIANT |
| winget/scoop exempt | `TestRunUpdate_GatingMatrix/always-update_winget_exempt:_update_still_runs_without_update_available` (wantUpdated=true) | winget.go:79, scoop.go:79 — `PolicyAlwaysUpdate` (update_available=true by design) | ✅ COMPLIANT |
| Dynamic detection | `TestRunUpdate_GatingMatrix/gated_apt_without_update_is_reported_current_and_update_is_skipped` (apt false → skipped) | update.go:171 predicate on the live `check()` result — apt `UpdateAvailable=false` → skipped | ✅ COMPLIANT |
| Gated check fails | `TestRunUpdate_GatingMatrix/gated_check_fails:_reported_failed,_never_current,_update_skipped` (wantUpdated=false, StatusFailed) | update.go:102-111 — check() error → `StatusFailed` + continue BEFORE the gate; `StatusCurrent` is only reachable at update.go:172-177; check.go:95-103 same mapping | ✅ COMPLIANT |

### Requirement: Check Failure Signal (ADDED)

| Scenario | Test (passed at runtime) | Implementation evidence | Result |
|----------|--------------------------|-------------------------|--------|
| Detection fails | `check_test.go` rows: `apt/command-fails` (wantErrContains "apt check failed"), `nvm/command-fails` ("nvm check failed"), `npm/other-nonzero-exit` ("(exit 2)"), `pnpm/other-nonzero-exit` ("(exit 2)") | helper.go:196-233 — `commandOutputErr`/`shellOutputErr`/`commandFailureErr` build `"<tool> check failed (exit N): <stderr excerpt>: %w"` (exit via `errors.As(*exec.ExitError)`, omitted when not extractable); wired at apt.go:27,37, nvm.go:46,57, npm.go:37, pnpm.go:39 | ✅ COMPLIANT |
| Empty output | `check_test.go` rows: `apt/empty-output`, `nvm/empty-current-unknown`, `npm/empty-version-unknown`, `pnpm/empty-version-unknown` — no error, `UpdateAvailable=false` | helpers return nil error on exit 0; empty/"(none)" maps to "unknown" (apt.go:32-34, nvm.go:50-53, npm.go:26-28, pnpm.go:28-30) — never a failure | ✅ COMPLIANT |
| Gated check fails in run | `TestRunUpdate_GatingMatrix/gated_check_fails...` (StatusFailed, Update not called, not StatusCurrent) | update.go:102-111 — failed check → `StatusFailed` + `continue` before the gate; check.go:95-103 identical; failure never reaches `StatusCurrent` | ✅ COMPLIANT |
| npm/pnpm maskless | `npm/exit-1-empty-output` ("(exit 1)"), `npm/other-nonzero-exit` ("(exit 2)"), `npm/deadline-exceeded` (`errors.Is` DeadlineExceeded), pnpm equivalents; `TestCommandOutputErr/real-exit-code` (REAL `sh -c exit 7` child → `*exec.ExitError` code 7, message "(exit 7)") | npm.go:37 / pnpm.go:39 — direct exec, no `|| true`, no shell wrapper; exit codes propagate via `errors.As` (helper.go:223-233, 238-241); deadline `%w`-chained (helper.go:79-82) so `errors.Is` survives | ✅ COMPLIANT |
| npm/pnpm exit-1 outdated | `npm/exit-1-outdated` (real exit-1 child + stdout → `UpdateAvailable=true`, no error), `pnpm/exit-1-outdated` (real exit-1 child → `UpdateAvailable=true`) | npm.go:38-43, pnpm.go:40-45 — `isExitCode(err, 1)` + non-empty stdout → valid detection; stdout decides availability (stdout preserved on failure, helper.go:199,213) | ✅ COMPLIANT |

**Compliance summary**: 12/12 scenarios compliant — every covering test passed at runtime.

## Correctness (Static Evidence)

| Check | Status | Notes |
|-------|--------|-------|
| UpdatePolicy declared at all 13 Info() sites | ✅ | grep: 4 Gated (apt.go:106, npm.go:103, pnpm.go:117, nvm.go:123) + 9 AlwaysUpdate (brew.go:90, bun.go:88, docker.go:107, gh.go:107, go.go:108, opencode.go:85, winget.go:79, scoop.go:79, custom.go:105); goldens pin every value (info_test.go:24-35) |
| Enum zero-value fail-closed | ✅ | interface.go:36-49 — `PolicyGated=0` with invariant comment mirroring the TrustLevel convention (interface.go:10-12); `ToolInfo.UpdatePolicy` at interface.go:88 |
| gatedOfficialAdapters deleted | ✅ | grep: 0 matches in production/tests; only archived docs and proposal mention it |
| Gate reads policy, not ID list | ✅ | update.go:171 — single predicate `info.UpdatePolicy == adapters.PolicyGated && !updateInfo.UpdateAvailable`; trust removed from gating (D2) |
| Failed check never current | ✅ | update.go:102-111 (update) and check.go:95-103 (check): error → `StatusFailed`, `continue` precedes the gate at update.go:171 |
| npm/pnpm maskless | ✅ | npm.go:37, pnpm.go:39 — `commandOutputErr("npm|pnpm", "outdated", ...)`; no `|| true` anywhere in production code |
| Exit codes not swallowed | ✅ | helper.go:223-233 `errors.As` extraction; proven by real-child rows (exit 1/2/7) and composite-key fake |
| Composite-key fake | ✅ | exec_mock_test.go:42-44 — `key = name + " " + strings.Join(args, " ")` distinguishes `npm --version` from `npm outdated -g --depth=0` |
| 21 fake literals explicit | ✅ | awk scan of update_test.go + integration_test.go + audit_probe_test.go: 21/21 `fakeUpdateAdapter{...}` blocks contain `policy:` (0 missing); matrix re-keyed ID→policy (update_test.go:99-172); consistency assertion adapter_update_test.go:293-295 |

## Coherence (Design)

| Decision | Followed? | Notes |
|----------|-----------|-------|
| D1 — enum, zero=Gated fail-closed | ✅ Yes | interface.go:36-49 + ToolInfo field :88; invariant comment cites TrustLevel convention (:10-12) |
| D2 — gate rewrite, trust removed | ✅ Yes | update.go:171 predicate; `gatedOfficialAdapters` deleted; dry-run branch untouched (:113-130) |
| D3 — helper variants + structured failure | ✅ Yes | helper.go:196-233; shape `"<tool> check failed (exit N): <stderr excerpt>: %w"`; exit omitted when not extractable; empty-stderr excerpt segment omitted; `%w` keeps `errors.Is` (proven by `npm/deadline-exceeded` + helper deadline subtests) |
| D4 — npm/pnpm timeout layering | ⚠️ Yes, with deviation | Design kept shell `timeout 15`; implementation REMOVED the shell wrapper — npm.go:30-36 / pnpm.go:32-38: direct exec bounded by `runCmdArgsFn`'s `CheckTimeout` (30s, helper.go:69), portable (no GNU `timeout` dependency — macOS). Spec scenario THENs still hold: non-zero exits propagate with exit codes (real exit 1/2/7 rows), timeouts are failures (DeadlineExceeded → structured error → StatusFailed). Literal GNU-timeout exit-124 is no longer observable; timeouts surface as DeadlineExceeded. apply-progress.md:38 ("`timeout 15` kept") is stale vs the code — see SUGGESTION 1 |
| D5 — error-aware detection reads only | ✅ Yes | apt.go:27,37 / nvm.go:46,57 detection via `shellOutputErr`; display-only reads keep plain variants (apt.go:112, nvm.go:128) |
| D6 — fake/policy explicitness | ✅ Yes | 21/21 literals explicit; matrix re-keyed to policy; goldens DeepEqual; declared-value assertion |
| D7 — scope boundaries | ✅ Yes | No CLI changes for the failure signal (check.go/update.go mapping pre-existing); winget/scoop untouched |

Documented deviations: 1 (D4 mechanism) — non-breaking, spec-compliant, arguably an improvement (portability). apply-progress "Deviations from Design" section does not list it; flagged as docs drift.

## TDD Compliance (Strict TDD)

| Check | Result | Details |
|-------|--------|---------|
| TDD Evidence reported | ✅ | apply-progress TDD Cycle Evidence table (tasks 1.1–3.6, 15 rows with RED/GREEN/TRIANGULATE/REFACTOR) |
| RED confirmed (test files exist) | ✅ | info_test.go, adapter_update_test.go, update_test.go, check_test.go all present; compile-fail + behavioral REDs documented per task |
| GREEN confirmed (tests pass) | ✅ | All gates re-run fresh at verify time: full suite exit 0, race exit 0, vet clean, gofmt clean, smoke 23/23, focused 57/57 subtests |
| Triangulation adequate | ✅ | 7-row gating matrix (Gated±update, AlwaysUpdate×3, custom×2, failed check); 49-row TestCheck (incl. exit-1/2 real children, deadline); 4+4 helper subtests (success/structured/real-exit-7/deadline) |
| Safety Net for modified files | ✅ | apply-progress: 26 official tests + cli package pass pre-change; hermetic assertions unchanged (task 2.2) |
| Assertion quality | ✅ | Behavioral assertions only: `Update called` flag, output status strings, UpdateInfo DeepEqual, error-message content, `errors.As`/`errors.Is`; no tautologies, no ghost loops (fixed 12-adapter list, concrete matrix rows) |
| Changed-file coverage | ✅ | helper.go 100% incl. all new symbols (`commandOutputErr`/`shellOutputErr`/`commandFailureErr`/`isExitCode` 100%); apt Check 93.8%; nvm Check 87.5%; npm Check 100%; pnpm Check 100%; cli update.go runUpdate 81.3% — all ≥80% |

Test layer distribution: Unit — official package (helper variants, check rows, goldens, consistency); Integration — cli package (gating matrix, hermetic flows, audit probe). No E2E (hermetic seam suite, consistent with precedent).

## Issues Found

**CRITICAL**: None

**WARNING**: None

**SUGGESTION** (recorded follow-ups):
1. Docs drift on the D4 mechanism: design.md D4 and apply-progress.md:38 claim shell `timeout 15` is kept; the code removed it in favor of `runCmdArgsFn`'s `CheckTimeout` (npm.go:30-36, pnpm.go:32-38). Spec-compliant and more portable (no GNU `timeout` on macOS), but archive-report should record the actual mechanism. Consequence: a literal GNU-timeout exit-124 can no longer occur; check timeouts now surface as DeadlineExceeded-structured errors.
2. nvm Update still runs `source ~/.nvm/nvm.sh` hardcoded (nvm.go:90), ignoring `NVM_DIR` — while Check() honours `NVM_DIR` (nvm.go:46,57). Pre-existing; an NVM_DIR-only install would update against the wrong root. Out of this change's scope.
3. `runCmdArgsFn` (helper.go:68-85) lacks the process-group kill + `WaitDelay` treatment `runCmdFn` has (helper.go:36-44, 56-65): on CheckTimeout it kills only the direct child. Now that npm/pnpm Check() routes through `runCmdArgsFn`, a grandchild-spawning `outdated` process would not be group-killed. Pre-existing gap; a future change should mirror `Setpgid`/negative-pid SIGKILL there.
4. check_test.go exit-code harness wires the real child error AFTER `setExecFakes` (check_test.go:591-608) — order-dependent because the fakes' map is captured by reference; works today, brittle to a future fake redesign.
5. apt Check() detection depends on bash (`bash -o pipefail -c ...`, apt.go:27,37) — pre-existing shell dependency; Debian/Ubuntu targets ship bash, but a dash-only environment fails detection with a structured error (now visible thanks to this change).
6. npm exit-1 availability is decided by stdout shape (npm.go:39,43): exit 1 + empty stdout is treated as an operational failure (EACCES/unreachable registry) rather than "no updates". Matches spec intent; worth a comment note that `--depth=0` stdout carries the outdated table.

## Verdict

**PASS** — All 19/19 tasks complete; 2/2 requirements and 12/12 scenarios compliant with passing covering tests on fresh execution (full suite exit 0, race exit 0, vet clean, gofmt clean, smoke 23/23, focused 57/57 subtests incl. real-child exit-code rows). One documented, non-breaking, spec-compliant design deviation (D4 timeout mechanism: shell `timeout 15` → Go `CheckTimeout`); 6 suggestion-level follow-ups, no blockers, no CRITICAL findings, no WARNINGs.
