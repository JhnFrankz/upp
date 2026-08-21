```yaml
schema: gentle-ai.verify-result/v1
evidence_revision: sha256:8f6ecd06d45a57f6ca0171cd70886c03bb2cd9b05955acbf989a3e346bedd2e3
verdict: pass_with_warnings
blockers: 0
critical_findings: 0
requirements: 10/10
scenarios: 53/53
test_command: go test ./... -count=1
test_exit_code: 0
test_output_hash: sha256:2d69a208610ee58f90136f3c4ee3abb332b25f9080028476ec4a0f7584252a03
build_command: go build ./...
build_exit_code: 0
build_output_hash: sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
```

# Verify Report: upp-unified-update-flow (Re-verification)

**Change**: upp-unified-update-flow
**Mode**: Strict TDD (test runner: `go test ./... -count=1`; project config `openspec/config.yaml`: `tdd: true`, verify `test_command: go test ./... -count=1`, `smoke_check: bash scripts/smoke-test.sh --skip-build`)
**HEAD**: `285fae2` on `wu5/spec-docs-e2e` (remediation commit) · change base `cfadbb4` (main, pre-WU1) · previous report at `0f51c0d` (verdict FAIL, 5 CRITICAL UNTESTED)

**Evidence payload definition**: `evidence_revision` is the SHA-256 of the concatenation (fixed order) of the SHA-256 digests of the five captured re-verification output artifacts: test, race, lint, smoke, build. Component digests:

- test  (`go test ./... -count=1`): sha256:2d69a208610ee58f90136f3c4ee3abb332b25f9080028476ec4a0f7584252a03
- race  (`go test ./internal/cli/ ./internal/output/ -race -count=1`): sha256:78b73fe46c8f933b35ef45a3cd2e4d4e1ddc6368549843a9cef2dd316507a5d7
- lint  (CI-pinned v2.12.2 `golangci-lint run ./...`): sha256:e92606b0bf483111dff0a120c315ea165821348f31365020e2468a0059095c47
- smoke (`bash scripts/smoke-test.sh --skip-build`, fresh binary): sha256:0c2307cde3e14397d6d90237f3cd329341a1199157e392dcbb67abbf9ea5e411
- build (`go build ./...`): sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855

## Overall Result

**PASS WITH WARNINGS**

The bounded remediation is verified: each of the 5 CRITICAL UNTESTED scenarios named in the previous report now has a dedicated, passing, table-driven root-Execute test pin — `TestVerifyPins_StrictTTDScenarios` (5 subtests, all green) in `internal/cli/update_test.go` — and the one production fix, `runList` in `internal/cli/list.go`, genuinely closes the `Filter round-trip` scenario because `--only`/`--skip` are now applied to the table rows using the same `ParseFilter`/`FilterTools` path as `update`. The full spec matrix re-walk now reads **53/53 scenarios COMPLIANT** (10/10 requirements: 7 MODIFIED/ADDED tables + 3 REMOVED mandates, all proven by passing tests plus source sweep).

All evidence re-executed independently at `285fae2`: full suite 329 top-level / 935 incl. subtests, 0 failed, 0 skipped (exit 0); `-race` clean on the changed packages; CI-pinned golangci-lint v2.12.2 reports 0 issues; 31/31 smoke assertions pass against a freshly built binary; scope drift is zero — `git diff --stat 0f51c0d..HEAD` touches exactly `internal/cli/list.go` (+9) and `internal/cli/update_test.go` (+38), matching commit `285fae2`'s stat. The regression spot-checks around the `list.go` change (no-flags, `--skip`, `-q`, unknown `--only`, empty-result "No tools configured.") all behave per spec and no test regressed.

Remaining items are non-blocking: WARNING-1 (local linter vacuous-pass risk) is **resolved at this re-verification** (the PATH `golangci-lint` is now the CI-pinned v2.12.2, so the skew that made v1.64.5 pass vacuously no longer exists on this machine) and is carried forward only as a resolved record; SUGGESTION-1 (stale `check` wording in the `parser.go:29` comment) and SUGGESTION-2 (stale `check` references in the `openspec/config.yaml` context) persist and are non-blocking.

## Evidence

### Test summary

**Command**: `go test ./... -count=1` → **exit 0**

| Package | Tests | Duration |
|---------|-------|----------|
| `internal/adapters` | 41 passed, 0 skipped | 0.456s |
| `internal/adapters/official` | 103 passed, 0 skipped | 0.663s |
| `internal/cli` | 33 passed, 0 skipped | 0.201s |
| `internal/config` | 60 passed, 0 skipped | 0.015s |
| `internal/output` | 9 passed, 0 skipped | 0.010s |
| `internal/platform` | 34 passed, 0 skipped | 0.004s |
| `internal/security` | 18 passed, 0 skipped | 0.008s |
| `internal/selfupdate` | 14 passed, 0 skipped | 0.412s |

**Total**: 329 top-level tests passed (935 including subtests), 0 skipped, 0 failed. `cmd/upp` has no test files. Delta vs. previous report: `internal/cli` 32→33 top-level (+`TestVerifyPins_StrictTTDScenarios`), subtests 929→935 (+5 pins).

**Race**: `go test ./internal/cli/ ./internal/output/ -race -count=1` → **exit 0** (`internal/cli` 1.304s, `internal/output` 1.068s) — the two packages touched by the remediation.

### Lint summary

**Command**: `/home/linuxbrew/.linuxbrew/Cellar/golangci-lint/2.12.2/bin/golangci-lint run ./...` (the CI-pinned binary, used explicitly) → **exit 0, "0 issues."**
`golangci-lint version` on the PATH symlink now resolves to **v2.12.2** (`../Cellar/golangci-lint/2.12.2/bin/golangci-lint`), so the v1.64.5 vacuous-pass skew behind WARNING-1 is gone on this machine; the authoritative run still used the explicit Cellar path.

### E2E / smoke

**Command**: `bash scripts/smoke-test.sh --skip-build` (freshly built binary at `/tmp/opencode/upp-verify-recheck/bin/upp`) → **31 passed, 0 failed, 31 total, exit 0**. Covers: `upp check` pruned (exit 1), `upp update --dry-run`/`-n`/`-q`/`-v`/`--only`/`--skip`, `--help` without `check`/`export`/`import`/`completion`, bare dashboard with `upp update -n` guidance, `upp list`, `init --ci`, pruned `export`/`import` exit 1.

### Runtime checks (read-only, fresh binary at `/tmp/opencode/upp-verify-recheck/bin/upp`)

- bare `upp --ci` (piped) → plain-text dashboard (banner `upp dev (linux/x86_64)`, `Tools: 10 enabled`, `upp update -n` quick-reference), **exit 0** ✅
- `upp list` (no flags) → full 10-row table, `ID | Name | Status | Version`, **exit 0** ✅
- `upp list --only apt` → exactly the `apt` row (filter now honored), **exit 0** ✅
- `upp list --skip npm` → 9 rows (npm removed), **exit 0** ✅
- `upp list -q` → table still rendered (quiet suppresses only the decorative dashboard banner, not the data table — spec-compliant: "Quiet mode" suppression is defined for the bare dashboard) ✅
- `upp list --only nope` → `Warning: tool "nope" not found, ignored` on stderr, `No tools configured.` on stdout, **exit 0** ✅
- `upp check` → unknown command, exit 1; `upp --help` → no pruned commands, exit 0; `upp update -n` ≡ `upp update --dry-run` ✅

### Remediation pin confirmation (previous report's 5 CRITICAL scenarios)

Each previously UNTESTED scenario now has a dedicated passing test pin. All five are subtests of `TestVerifyPins_StrictTTDScenarios` (`internal/cli/update_test.go:969`), a table-driven root-`Execute` harness that, per case, resets `update`/`list`/`self-update` deps to two official-trust fakes (`broken` = `PolicyAlwaysUpdate` with `updateErr: "lock held"`; `apt` = `PolicyGated` with a `2.0.0` update available and a success result), runs `BuildRoot()` + `AddCommands` + `root.SetArgs(...)` + `root.Execute()` with captured stdout, and asserts exit-status parity plus positive/negative output guards. Re-executed with `-v`: all 5 subtests PASS.

| # | Scenario (spec) | Pin (subtest) | Executes | Asserts | Result |
|---|-----------------|---------------|----------|---------|--------|
| 1 | command-interface / Command Structure / **No args + `--ci`** | `bare --ci dashboard` | bare `upp --ci` via root Execute (non-TTY) | output contains `upp update -n` (dashboard rendered, not an error), no err (exit 0) | ✅ PASS |
| 2 | command-interface / Command Structure / **`update --ci` (exit non-zero on failure)** | `update --ci non-zero on failure` | `upp update --ci --only broken` (failing official adapter) | output contains `Failed: broken` (failure summary rendered), `err != nil` (non-zero exit from the `gf.CI && hasFailure` branch) | ✅ PASS |
| 3 | ux-patterns / List Table Output / **Filter round-trip (`upp list --only apt`)** | `list --only round-trip` | `upp list --only apt` end-to-end through the fixed `runList` | output contains `apt` and NOT `brew` — the `--only` filter now actually filters the table rows | ✅ PASS |
| 4 | ux-patterns / Verbose Error Diagnostics / **Short flag `-v` diagnostics** | `-v shorthand diagnostics` | `upp update -v --only broken` (failure path driven through the `-v` *shorthand*) | output contains `lock held` — the captured stderr diagnostics render through the shorthand, not just the long flag | ✅ PASS |
| 5 | ux-patterns / Verbose Error Diagnostics / **Success with verbose** | `all-success verbose clean` | `upp update -v --only apt` (all-success update under `Verbose: true`) | output contains `1 updated` (success summary) and NOT `│` (no stderr/diagnostics debug noise on the success path) | ✅ PASS |

`runList` fix confirmation (`internal/cli/list.go:49-56`): after `deps.buildAdapterList`, `runList` now runs `ParseFilter(gf.Only, gf.Skip)` + `FilterTools(adapterIDs(adapterList), only, skip, os.Stderr)` and rebuilds `adapterList` from `adapterByID` — the **same shared filter path `update` uses**, so table rows map to filter names by construction. Verified live: `upp list --only apt` → only the `apt` row; `upp list --skip npm` → npm absent; unknown `--only` → stderr warning + `No tools configured.`. No regression in the no-flags case (identity: `FilterTools` returns `tools` unchanged when both filters are empty) — full 10-row table still renders, `TestListCommand_NoConfig` and `TestEmptyConfig_AllToolsSkipped` still pass.

### Spec scenario compliance matrix

Statuses: ✅ COMPLIANT (dedicated covering test passed at runtime) · ❌ UNTESTED (no dedicated passing covering test) · ❌ FAILING (covering test failed). Under Strict TDD, UNTESTED/FAILING on a required scenario is CRITICAL.

**command-interface — Command Structure** (MODIFIED)

| Scenario | Test | Result |
|----------|------|--------|
| No args | `parser_test.go: TestBuildRoot_BareInvocationRunsDashboard`; `integration_test.go: TestRootCommand_NoArgs`; `root_test.go: TestRunDashboard_WithConfig` | ✅ COMPLIANT |
| No args + `--ci` | `update_test.go: TestVerifyPins_StrictTTDScenarios/bare_--ci_dashboard` (bare root Execute, `--ci`, dashboard guidance asserted, exit 0) + runtime piped `upp --ci`→0 | ✅ COMPLIANT *(was UNTESTED — now pinned)* |
| `update` | `update_test.go: TestRunUpdate_InteractiveSelection` (TTY, selector, confirmations, carried outcomes) | ✅ COMPLIANT |
| `update --dry-run` | `integration_test.go: TestUpdateDryRunCommand_NoConfig`, `TestDryRun_NoCommandsExecuted` (zero Update calls) | ✅ COMPLIANT |
| `update -n` | `update_test.go: TestUpdateCommand_DryRunShorthand` + runtime equivalence | ✅ COMPLIANT |
| `update --ci` (exit non-zero on failure) | `update_test.go: TestVerifyPins_StrictTTDScenarios/update_--ci_non-zero_on_failure` (failing tool through `update --ci`, `Failed: broken` asserted, non-zero return from the `gf.CI && hasFailure` branch) | ✅ COMPLIANT *(was UNTESTED — now pinned)* |
| `self-update` | `selfupdate_test.go: TestSelfUpdate_Confirmed/UpToDate/Declined` (prompt, verified replace, exactly one backup) | ✅ COMPLIANT |
| Pruned `check` command | `parser_test.go: TestUnknownCommand_Check` + smoke test 5 | ✅ COMPLIANT |
| Pruned `export` command | `parser_test.go: TestUnknownCommand_Export` | ✅ COMPLIANT |
| Pruned `import` command | `parser_test.go: TestUnknownCommand_Import` | ✅ COMPLIANT |
| Unknown command (`foo`) | `parser_test.go: TestUnknownCommand_*` (same cobra path, 3 dedicated pins) + runtime `upp foo`→1 | ✅ COMPLIANT |
| `--help` | `help_test.go: TestHelp_ShowsGroups/CommandsListed` + runtime exit 0 | ✅ COMPLIANT |

**command-interface — Help Output Grouping** (MODIFIED)

| Scenario | Test | Result |
|----------|------|--------|
| Simplified help groups | `help_test.go: TestHelp_ShowsGroups` (Commands + Maintenance present, `Tool Commands`/`Config Commands` absent) | ✅ COMPLIANT |
| Help subcommand | `help_test.go: TestHelp_EqualsHelpSubcommand` | ✅ COMPLIANT |
| Completion hidden | `help_test.go: TestHelp_CommandsListed` (rejects `completion`) + parser.go `HiddenDefaultCmd: true` | ✅ COMPLIANT |
| Pruned commands absent | `help_test.go: TestHelp_CommandsListed` (rejects `export`/`import`; runtime grep shows no `check` either) | ✅ COMPLIANT |

**command-interface — Self-Update Flag Semantics** (MODIFIED)

| Scenario | Test | Result |
|----------|------|--------|
| Unknown flag | `selfupdate_test.go: TestSelfUpdate_UnknownFlagRejected` (root `self-update --yes` → cobra "unknown flag", non-zero) | ✅ COMPLIANT |
| `--only` ignored | `selfupdate_test.go: TestSelfUpdate_OnlySkipIgnored` (exactly 1 network call, normal flow) | ✅ COMPLIANT |
| `--ci` | `selfupdate_test.go: TestSelfUpdate_CIDeny` + `TestSelfUpdate_CIDenyThroughRoot` | ✅ COMPLIANT |
| `--quiet` prompt | `selfupdate_test.go: TestSelfUpdate_QuietKeepsPrompt` | ✅ COMPLIANT |

**config-system — Self-Update Detection Setting** (REMOVED, no scenario table)

| Aspect | Test | Result |
|--------|------|--------|
| Field removed; `Settings` intentionally empty | `config.go:29` `type Settings struct{}`; `config_test.go: TestDefaultConfig`, `TestLoadValidTOML` (stray key tolerated) | ✅ COMPLIANT |
| Stray `check_self_update` loads ignored, `Save` never rewrites | `config_test.go: TestLoadStrayCheckSelfUpdateKey_NeverRewritten` | ✅ COMPLIANT |

**self-update — GitHub Release Detection** (MODIFIED)

| Scenario | Test | Result |
|----------|------|--------|
| Fresh cache | `client_test.go: TestLatestCached` (fresh → cached `v0.1.1` served, **0 network requests** asserted) + `cache_test.go: TestFresh` (6 boundary cases) | ✅ COMPLIANT |
| API failure (command) | `client_test.go: TestLatestFresh` (HTTP 500 → visible error) + `selfupdate_test.go` clear-error paths; `--ci`/non-TTY deny also pinned | ✅ COMPLIANT |
| Stale cache | `client_test.go: TestLatestCached` (26h-old cache → 1 refetch, writes through) | ✅ COMPLIANT |
| No hint-path detection | Source sweep: `internal/selfupdate` client instantiated only by `cli/selfupdate.go`; hint path deleted (no `maybeShowSelfUpdateHint`, no `check_self_update` consumer); `nvm.go` uses only `selfupdate.Parse` (pure version parsing, no network) | ✅ COMPLIANT |

**ux-patterns — Live Check Board** (ADDED)

| Scenario | Test | Result |
|----------|------|--------|
| Board renders up-front | `checkboard_test.go: TestCheckBoard_Start_PaintsPendingLinesInCanonicalOrder` (canonical order + ANSI engaged) | ✅ COMPLIANT |
| Per-tool completion flip | `checkboard_test.go: TestCheckBoard_Complete_AvailableFlipsOnlyTargetLine` (only target row flips, `1.2 → 1.3`) | ✅ COMPLIANT |
| Up-to-date stays visible | `checkboard_test.go: TestCheckBoard_Complete_CurrentShowsUpToDate` | ✅ COMPLIANT |
| Failed check flips to ✗ | `checkboard_test.go: TestCheckBoard_Complete_FailedShowsInlineError` | ✅ COMPLIANT |
| Settled board gates selector | `checkboard_test.go: TestCheckBoard_Finish_SettlesFrameAndIsIdempotent` + `update_test.go: TestRunUpdate_InteractiveSelection` (selector receives exactly the pending `wantOpts`, current/failed excluded) | ✅ COMPLIANT |
| Atomic concurrent rendering | `checkboard_test.go: TestCheckBoard_ConcurrentComplete_SerializesUpdates` (8 goroutines, all 4 statuses, under `-race`) | ✅ COMPLIANT |
| Non-color fallback | `checkboard_test.go: TestCheckBoard_NonColorFallback_OnePlainLinePerCompletion` (Start silent, zero ANSI bytes, one plain line per completion) | ✅ COMPLIANT |
| Bypass modes unchanged | `update_test.go: TestRunUpdate_SelectorGateMatrix` (5 rows: TTY/non-TTY/`--ci`/`--quiet`/`--dry-run` → sequential path, no board, no selector) | ✅ COMPLIANT |

**ux-patterns — Progress Indication** (MODIFIED)

| Scenario | Test | Result |
|----------|------|--------|
| Multi-tool update | `render_test.go: TestProgress_UpdateVerb` ("Updating X/Y" during update phase) | ✅ COMPLIANT |
| Single tool | `render_test.go: TestProgress_SingleTool` (no indicator for 1 tool) | ✅ COMPLIANT |
| No checking counter | `checkboard_test.go` (board rendering) + `update_test.go:597/708` (never "Checking" in TTY pre-check output) | ✅ COMPLIANT |
| Concurrent progress atomicity | `render_test.go: TestRenderer_ConcurrentProgress_ThreadSafe` + `-race` runs | ✅ COMPLIANT |

**ux-patterns — List Table Output** (MODIFIED)

| Scenario | Test | Result |
|----------|------|--------|
| Correct columns | `render_test.go: TestListTools` + `TestListTools_IDColumn` (header `ID | Name | Status | Version`, no `Tool` mislabel) | ✅ COMPLIANT |
| Filter round-trip (`upp list --only apt`) | `update_test.go: TestVerifyPins_StrictTTDScenarios/list_--only_round-trip` (end-to-end `upp list --only apt` via root Execute: `apt` present, `brew` absent) + the `runList` filter fix (`list.go:49-56`, same `ParseFilter`/`FilterTools` path as `update`) + live binary check | ✅ COMPLIANT *(was UNTESTED — now pinned and the production filter path fixed)* |

**ux-patterns — Bare Dashboard Welcome Screen** (MODIFIED)

| Scenario | Test | Result |
|----------|------|--------|
| Interactive dashboard | `render_test.go: TestDashboard_Formatting` (banner, tools count, `upp update -n` guidance, rejects `upp check`) + `root_test.go: TestRunDashboard_WithConfig` | ✅ COMPLIANT |
| Non-TTY / Pipe | `render_test.go: TestDashboard_PlainNonTTY` (no ANSI) + runtime piped `upp --ci`→0 | ✅ COMPLIANT |
| Quiet mode | `render_test.go: TestDashboard_QuietSuppresses` (zero bytes) + `root_test.go: TestRunDashboard_QuietSuppresses` | ✅ COMPLIANT |
| Missing config | `root_test.go: TestRunDashboard_NoConfig` + `render_test.go: TestDashboardNoConfig` (directs to `upp init`) | ✅ COMPLIANT |

**ux-patterns — Summary Report** (MODIFIED)

| Scenario | Test | Result |
|----------|------|--------|
| All succeed | `render_test.go: TestUpdateSummary_AllUpdated` ("2 updated, 0 failed. All clean!") + `update_test.go: TestRunUpdate_AllSucceedSummary` (end-to-end) | ✅ COMPLIANT |
| Partial fail | `render_test.go: TestUpdateSummary_PartialFailure` + `integration_test.go: TestUpdateDryRun_MixedStatusCounts` | ✅ COMPLIANT |
| No tools | `render_test.go: TestUpdateSummary_AllSkipped` + `integration_test.go: TestEmptyConfig_AllToolsSkipped` | ✅ COMPLIANT |
| Up-to-date with skips | `render_test.go: TestUpdateSummary_CurrentWithSkipsDryRun` ("8 up to date, 2 skipped") + `update_test.go: TestRunUpdate_DryRunCurrentWithSkips` (never "All tools up to date.") | ✅ COMPLIANT |
| Dry-run pending | `update_test.go: TestRunUpdate_DryRunPendingNeverClean` ("would update", never "All clean!") + `TestUpdateDryRun_DeterministicOrderUnderConcurrency` | ✅ COMPLIANT |
| Concurrent deterministic order | `integration_test.go: TestUpdateDryRun_DeterministicOrderUnderConcurrency` (alpha<gamma<epsilon under reversed completion) | ✅ COMPLIANT |

**ux-patterns — Verbose Error Diagnostics Rendering** (MODIFIED)

| Scenario | Test | Result |
|----------|------|--------|
| Verbose failure diagnostics | `render_test.go: TestToolLine_VerboseFailureDiagnostics` (indented `    │ <line>` per stderr line) + `update_test.go: TestRunUpdate_VerboseFailureDiagnostics` (end-to-end `-v`) | ✅ COMPLIANT |
| Short flag `-v` diagnostics | `update_test.go: TestVerifyPins_StrictTTDScenarios/-v_shorthand_diagnostics` (failure path driven through the `-v` shorthand; `lock held` stderr rendered) + `parser_test.go: TestBuildRoot_FlagShorthands` (`-v`→`gf.Verbose` mapping) | ✅ COMPLIANT *(was UNTESTED — now pinned)* |
| Default non-verbose failure | `render_test.go: TestToolLine_NonVerboseFailureSuppressed` + `update_test.go: TestRunUpdate_VerboseFailureDiagnostics` (default run suppresses stderr) | ✅ COMPLIANT |
| Success with verbose | `update_test.go: TestVerifyPins_StrictTTDScenarios/all-success_verbose_clean` (all-success update under `Verbose: true`; success summary rendered, zero `│` diagnostics noise) | ✅ COMPLIANT *(was UNTESTED — now pinned)* |
| Quiet takes precedence | `render_test.go: TestToolLine_QuietOverridesVerbose` + `update_test.go: TestRunUpdate_VerboseFailureDiagnostics` (`-q -v` suppresses) | ✅ COMPLIANT |

**ux-patterns — Self-Update Detection Hint** (REMOVED, no scenario table)

| Aspect | Test | Result |
|--------|------|--------|
| No code path renders the hint | Source sweep (zero `SelfUpdateHint`/`maybeShowSelfUpdateHint` in `cmd`/`internal`/`scripts`/`README`) + `check_hint_test.go` deleted; `render.go` has no hint method | ✅ COMPLIANT |

**Compliance summary**: **53/53 scenarios ✅ COMPLIANT · 0 UNTESTED · 0 FAILING**. The 3 REMOVED requirements (command-interface `upp check`, config-system Self-Update Detection Setting, ux-patterns Self-Update Detection Hint) carry no scenario tables; their removal mandates remain proven by passing tests + source sweep. All 5 rows that were UNTESTED in the previous report are now pinned; no other row changed status, and the `list.go` change introduced no regression anywhere in the matrix (List Table Output, Summary "No tools", dashboard quiet/missing-config, and the `--skip`/`--only` update-path pins all re-verified green).

> Every scenario **introduced or altered by this change** (Live Check Board ×8, pruned `check`, removed hint, `-n` dry-run query surface, all-clean zero-failure count, dashboard quick-reference, `check_self_update` forward-compat, release-detection exclusivity) has a dedicated passing test, and the 5 previously carried-over UNTESTED scenarios are now pinned too. 53/53.

### Completeness

| Metric | Value |
|--------|-------|
| Tasks total | 18 |
| Tasks complete | 18 |
| Tasks incomplete | 0 |

Re-confirmed at `285fae2`: all 18/18 tasks remain checked in tasks.md; the remediation commit added no new tasks and broke none. Spot-check of the remediation: `runList` now applies `ParseFilter`/`FilterTools` (the `list` task's filter contract, previously only exercised on the `update` path) — verified in source and live; the 5 verify pins are in a single additive test function with no production-code coupling.

### Correctness (Static Evidence)

| Requirement | Status | Notes |
|------------|--------|-------|
| Live per-tool check board in TTY `upp update` | ✅ Implemented | `update.go:284-293`: board built in canonical filtered order, `Start()` before pool, `board.Complete` as `onResult`, `Finish()` before pending-only selector. Unchanged by remediation. |
| Full removal of `upp check` | ✅ Implemented | `check.go` deleted; `parser.go AddCommands` registers only init/update/self-update/list; `deps.go` has no `check` slot; runtime `upp check`→exit 1. Unchanged. |
| Full removal of self-update hint | ✅ Implemented | `maybeShowSelfUpdateHint`, `Renderer.SelfUpdateHint`, hint tests, `Settings.CheckSelfUpdate` all deleted; no residual consumers. Unchanged. |
| `-n` dry-run as sole read-only query surface | ✅ Implemented | `update --dry-run`/`-n` prints planned actions, executes zero updates (`TestDryRun_NoCommandsExecuted`); documented in README + dashboard quick-reference. Unchanged. |
| All-clean summary explicitly counts zero failures | ✅ Implemented | `render.go:328` → `"N updated, 0 failed. All clean!"`; pinned at renderer and CLI level. Unchanged. |
| Up-to-date counting + listing (D6) | ✅ Implemented | `render.go:296-298` "N up to date" part; `detailSummary` "Up to date:" listing; all-skipped branch guarded by `current == 0` (`render.go:310`). Unchanged. |
| `check_self_update` forward-compat | ✅ Implemented | `Settings struct{}`; non-strict decode tolerates the key; `Save` never re-emits it. Unchanged. |
| `upp list` honors `--only`/`--skip` (filter round-trip) | ✅ Implemented (remediation) | `list.go:49-56` applies the shared `ParseFilter`/`FilterTools` path to the table rows; `--only` wins over `--skip` (parser.go:88-97); unknown names warn to stderr; empty result → `No tools configured.` (list.go:85-88); no-flags path is identity. |

### Coherence (Design)

| Decision | Followed? | Notes |
|----------|-----------|-------|
| D1 — `CheckBoard` owns multi-line ANSI behind its own `sync.Mutex` | ✅ Yes | `checkboard.go`: private mutex, cursor-up+clear+down single-row rewrite, no full-board redraw. |
| D2 — `runChecks` seam replaces renderer/flags with `onResult` (nil=silent) | ✅ Yes | `checkrun.go:114`; nil callback path pinned (`TestRunChecks_NilCallbackSilent`). |
| D3 — machinery moved verbatim to `checkrun.go`; `check.go` deleted | ✅ Yes | All six symbols present in `checkrun.go`; `check.go` removed; shared helpers `buildAdapterList/adapterIDs/adapterByID` relocated and still used by update/list. |
| D4 — Start→Complete→Finish lifecycle, per-line flip, pending-only selector | ✅ Yes | `update.go:288-293`; selector input filters `StatusAvailable` only. |
| D5 — fallback via `Renderer.Color()`, one plain line per completion, no ANSI | ✅ Yes | `render.go:83` getter; `checkboard.go:76-80` fallback; `update.go:288` passes `r.Color()`. |
| D6 — summary gains up-to-date count + listing | ✅ Yes | `render.go:296-298`, `detailSummary` Current section. |
| Carried-outcome loop + bypass contracts untouched | ✅ Yes | Sequential path byte-identical in behavior; all four bypass gates pinned by `TestRunUpdate_SelectorGateMatrix`. |
| Remediation — `list` filter reuses the update filter path | ✅ Yes | `runList` calls the same `ParseFilter` + `FilterTools` (parser.go) that `update` uses, preserving `--only`-wins semantics and stderr warnings — the round-trip is structural, not a second implementation. |

### TDD Compliance

| Check | Result | Details |
|-------|--------|---------|
| TDD Evidence reported | ✅ | "TDD Cycle Evidence" tables present for all 5 work units in apply-progress.md. |
| All tasks have tests | ✅ | 18/18 tasks reference test files that exist in the codebase. |
| RED confirmed (tests exist) | ✅ | `checkrun_test.go`, `checkboard_test.go`, `update_test.go`, `render_test.go`, `parser_test.go`, `config_test.go`, `selfupdate_test.go`, `smoke-test.sh` all present; the 5 remediation pins exist in `update_test.go` and are scenario-named. |
| GREEN confirmed (tests pass) | ✅ | Every reported test file passes on re-execution (`go test ./... -count=1` exit 0; `TestVerifyPins_StrictTTDScenarios` 5/5 subtests PASS). |
| Triangulation adequate | ✅ | The 5 new pins are table-driven rows with distinct (args, want, not, wantErr) tuples — each pin proves a different exit status and output guard, not one shared expectation. |
| Safety Net for modified files | ✅ | `internal/cli` (33 tests) and `internal/output` (9 tests) passed before and after the remediation; `-race` green on both changed packages; `list.go` went from no dedicated filter pin to a covered one (`runList` 84.8%, package 91.8%). |

**TDD Compliance**: 6/6 checks passed. The prior coverage gap (5 unpinned spec scenarios) is closed by dedicated passing pins; no new gap was introduced.

### Test Layer Distribution

| Layer | Tests | Files | Tools |
|-------|-------|-------|-------|
| Unit | ~152 top-level | `checkboard_test` (10), `checkrun_test` (6), `render_test` (44), `config_test` (17), `parser_test` (24, unit portion), `root_test` (3), `help_test` (3) | `go test` |
| Integration | ~85 top-level | `update_test` (14, incl. the 5-pin table), `integration_test` (33), `selfupdate_test` (14), `parser_test`/`help_test`/`root_test` (root-Execute portion) | `go test` |
| E2E | 31 assertions | `scripts/smoke-test.sh` | `bash` + built binary |
| **Total** | **329 Go + 31 smoke** | 11 Go test files + 1 shell harness | — |

### Changed File Coverage

From `-coverprofile` + `go tool cover -func` (remediation-touched + key changed production files, re-measured at `285fae2`):

| File | Coverage | Rating |
|------|----------|--------|
| `internal/cli/list.go` (remediation) | `runList` 84.8%, `NewListCommand` 100% (package total 91.8%) | ✅ Acceptable (the low spot is the unreachable-in-tests `config.Load`/`platform.Detect` error wrapping, not the new filter code) |
| `internal/output/checkboard.go` | 88.9–100% per func (`Complete` 91.7%, `boardResultLine` 88.9%) | ✅ Excellent |
| `internal/cli/checkrun.go` | 92.9–100% (`safeCheck` 95.2%, `buildAdapterList` 92.9%) | ✅ Excellent |
| `internal/cli/update.go` | 86.0–100% (`runUpdateInteractive` 86.0%, `runUpdateSequential` 90.4%, `timeoutErr` 100%) | ✅ Acceptable |
| `internal/cli/parser.go` | 88.9–100% | ✅ Acceptable |
| `internal/cli/selfupdate.go` | 86.4–100% | ✅ Acceptable |
| `internal/output/render.go` | `UpdateSummary`/`detailSummary`/`ToolLine` covered; gaps in `Color()` 0% (exercised via update path, not directly), `isTerminal` 42.9% (env-dependent) | ⚠️ Acceptable |
| `internal/config/config.go` | 50–100% (`Save` 73.3%, `ConfigDir` 50% — env-dependent) | ⚠️ Acceptable |

No changed file is below ~80% at the behavioral level; the low spots are environment-dependent helpers, not change logic. Coverage threshold per `openspec/config.yaml`: 0 (informational).

### Assertion Quality

**Assertion quality**: ✅ All assertions verify real behavior. Zero tautologies, ghost loops, type-only-without-value, or smoke-only assertions found in the new/modified test files. The new pin table pairs every positive guard (`want`) with an explicit negative guard (`not`) or exit-status expectation (`wantErr`): the `--ci` dashboard row asserts guidance text **and** `wantErr=false`; the `update --ci` row asserts the failure summary **and** `wantErr=true`; the `list --only` row asserts presence of `apt` **and** absence of `brew`; the `-v` row asserts the stderr text actually rendered; the all-success row asserts the summary **and** absence of the `│` diagnostics prefix. The `checkboard_test.go` harness continues to assert *settled visible frames* via a minimal VT100-subset simulator (not raw escape bytes), idempotency by byte-length, and explicit negative guards.

### Quality Metrics

**Linter**: ✅ 0 issues on the CI-pinned v2.12.2 binary (authoritative Cellar path) and the PATH default now also resolves to v2.12.2 (WARNING-1 resolved at this re-verification).
**Type Checker**: ✅ No errors (`go build ./...` exit 0, clean output; `go vet` re-run at apply WU gates).
**Formatter**: ✅ `gofmt -s -l .` clean at apply WU gates (no source formatting change in the remediation — both added files diff clean under gofmt).

### Scope Drift Check

- **Remediation commit surface**: `git show --stat 285fae2` → exactly `internal/cli/list.go` (+9) and `internal/cli/update_test.go` (+38); nothing else. `git diff --stat 0f51c0d..HEAD` (previous report HEAD → current HEAD) shows the identical 2 files — **zero drift beyond the remediation since the previous report**.
- **Working tree**: only untracked SDD bookkeeping under `openspec/changes/upp-unified-update-flow/` (design.md, explore.md, proposal.md, specs/, verify-report.md) — no stray source/test/config modifications.
- **Residual `upp check` references**: unchanged from previous report — only `scripts/smoke-test.sh:188` (the intentional pruned-command exit-1 assertion), historical archives under `openspec/changes/archive/` (immutable), and the two documented SUGGESTION-1/2 wording sites. No code, docs, e2e test, completion file, or dashboard reference remains.
- **Residual hint references**: none in `cmd`/`internal`/`scripts`/`README`; `check_self_update` appears only as a documented removal and forward-compat test pins; the still-valid `self-update-cache.json` release cache remains under the self-update capability.
- **Shell completions**: no completion files reference `check`; the cobra `completion` built-in is hidden (`parser.go:41-43`).

## Findings

**CRITICAL**: none. All 5 CRITICAL UNTESTED scenarios from the previous report are closed by dedicated passing test pins (see "Remediation pin confirmation"); the full matrix re-walk is 53/53 COMPLIANT with 0 FAILING.

**WARNING**:
- **WARNING-1** — Lint toolchain version skew: **resolved at this re-verification**. Previously the PATH default was golangci-lint v1.64.5, which exited 0 without enforcing the v2 config (vacuous pass). The PATH `golangci-lint` now resolves to the CI-pinned v2.12.2 (`~/.linuxbrew/bin/golangci-lint` → `../Cellar/golangci-lint/2.12.2/bin/golangci-lint`, `version` = v2.12.2), and the authoritative run (explicit Cellar path) reports 0 issues. Carried forward as a resolved record; remaining risk: a future environment without the v2.12.2 install would reintroduce the skew — keep the CI pin / explicit path as the source of truth. Non-blocking.

**SUGGESTION** (carried forward, persist, non-blocking):
- **SUGGESTION-1** — Stale comment: `internal/cli/parser.go:29` still reads `Running upp with no args shows status (like check), read-only.` — `check` no longer exists; reword to reference the dashboard.
- **SUGGESTION-2** — `openspec/config.yaml` context block still lists `check` among the cobra subcommands ("cli (cobra subcommands check/list/update/init/self-update)") and the apply guideline still says "Keep upp check/list/update/… working" — stale after this change's removals; refresh at archive.

## Follow-ups

1. **Archive**: the change is verified 53/53. Merge WU1–WU5 child PRs onto the tracker branch, then run `sdd-archive` (spec deltas apply at archive).
2. Optional at archive: fix the stale `check` wording in `parser.go:29` and the `openspec/config.yaml` context (SUGGESTION-1/2) — non-blocking, cosmetic/docs only.
3. Keep linting via the CI-pinned v2.12.2 binary (or a version gate) to prevent a regression to the vacuous-lint skew (WARNING-1, resolved).
