```yaml
schema: gentle-ai.verify-result/v1
evidence_revision: sha256:a90c997bc131f434875a296dd6d2bbebba0e0973978a156873ab533f3deec102
verdict: pass
blockers: 0
critical_findings: 0
requirements: 0/0
scenarios: 0/0
test_command: go test ./... -count=1
test_exit_code: 0
test_output_hash: sha256:36804fd3f158c1f108ca46e7539a30794ec6d8c4e38201d24d07176271812ae1
build_command: go build -o upp ./cmd/upp && bash scripts/smoke-test.sh --skip-build
build_exit_code: 0
build_output_hash: sha256:581e082832e13df89150a27320f0435f51f7d7461de46f25daffdaed1c5a7331
```

# Verification Report

**Change**: upp-hermetic-cli-tests
**Version**: N/A (spec-neutral, no delta specs)
**Mode**: Strict TDD (test refactor — RED = baseline suite green at 33.371s; success gate = final <2s)
**Evidence revision**: main @ fe79f8f (PR #42 merged), worktree clean except untracked `openspec/changes/upp-hermetic-cli-tests/`

## Completeness

| Metric | Value |
|--------|-------|
| Tasks total | 25 |
| Tasks complete | 25 |
| Tasks incomplete | 0 |

Task checkbox inventory verified by reading tasks.md: Phase 1 (1.1–1.6, 6/6 `[x]`), Phase 2 (2.1–2.4, 4/4 `[x]`), Phase 3 (3.1–3.14, 14/14 `[x]`), Phase 4 (4.1–4.5, 5/5 `[x]`). Phases sum to 6+4+14+5 = 25. All phases complete — full verification run.

## Build & Tests Execution

All gates executed fresh on current main (HEAD fe79f8f, PR #42 merged). Outputs captured to `/tmp/opencode/verify-upp/`:

| Gate | Command | Result | Exit | Output hash (sha256) |
|------|---------|--------|------|----------------------|
| **CLI timing gate** | `time go test ./internal/cli/ -count=1` | `ok 0.039s` test time; **REAL=0.458s — <2s criterion PASS** (baseline was 33.371s) | 0 | `14ce24854ab0b1f959aceda1de59bf450e61307a04d8ed8cdb3dc60d08d4bda7` |
| Full suite | `go test ./... -count=1` | 8 packages with test files `ok` (cli 0.036s) | 0 | `36804fd3f158c1f108ca46e7539a30794ec6d8c4e38201d24d07176271812ae1` |
| Race | `go test ./... -count=1 -race` | 8 packages `ok` (cli 1.126s) | 0 | `925e8e08ebe5c93b269ffec13b16d27fdd10c90d0b20b6bd37f1edd399222acd` |
| Vet | `go vet ./...` | clean, no output | 0 | `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855` (empty) |
| gofmt | `gofmt -l internal/cli/` | empty (0 bytes) | 0 | — |
| Smoke (fresh build) | `go build -o upp ./cmd/upp && bash scripts/smoke-test.sh --skip-build` | **23 passed, 0 failed, 23 total** — real-adapter E2E proof intact | 0 | `581e082832e13df89150a27320f0435f51f7d7461de46f25daffdaed1c5a7331` |

## Spec Compliance

Spec-neutral confirmed: **0 requirements / 0 scenarios** (authoritative counts from `spec.md`). The implementation invariants from the spec were verified directly against main with file:line evidence:

| Implementation Invariant | Evidence |
|--------------------------|----------|
| `cliDeps` package-level var feeding the 4 RunE bodies | deps.go:12-17 (struct: check/update/list/selfUpdate); check.go:25 `cliDeps.check`; list.go:22 `cliDeps.list`; update.go:27 `cliDeps.update`; selfupdate.go:44 `cliDeps.selfUpdate` |
| `checkDeps` nil-default | check.go:38-43 (struct: clientFactory + buildAdapterList), check.go:55-57 `if deps.buildAdapterList == nil { deps.buildAdapterList = buildAdapterList }`; nil-default clientFactory check.go:135-139 |
| `listDeps` nil-default | list.go:30-32 (struct), list.go:44-46 nil-default before use at :47 |
| `fakeUpdateAdapter` Command/Privileges | update_test.go:21-31 (fields `command`/`privileges`), update_test.go:46-54 `Info()` returns `ToolInfo{ID, Name, Trust, Command, Privileges}` |
| Fakes use real adapter IDs | Gating matrix rows: `apt` (update_test.go:106,114 — gated both directions), `brew` (:122 exempt), `winget` (:130 exempt), `custom` (:138,146 exempt); `npm` in flow tests (update_test.go:193, integration_test.go:490); TestAdapterIDs/ByID use apt/brew/npm (integration_test.go:449-451, 465-466). No synthetic IDs anywhere — the gate predicate `gatedOfficialAdapters[info.ID]` (update.go:183) is exercised with a real gated member |
| No `os/exec` in cli `*_test.go` | grep over internal/cli test files: 0 matches |
| 13 conversions present | 8 cobra-entry: TestListCommand_NoConfig (integration_test.go:290), TestCheckCommand_NoConfig (:316), TestCIMode_RejectsUntrustedCustomTools (:385), TestDryRun_NoCommandsExecuted (:415), TestQuietMode_SuppressesProgress (:483), TestUpdateFlow_ConfigToSummary (:601), TestInitCheckUpdateLifecycle (:764), TestCheckCommand_SummaryOutput (:922). 5 probes: audit_probe_test.go:24, 43, 62, 80, 100 — all assert `updated` flag |
| Probes assert `updated` flag | audit_probe_test.go:35-37 (CI: must not execute), :54-56 (interactive High deny), :73-75 (untrusted High deny), :91-93 (trusted Low `updated == true`), :113-115 (quiet Medium deny) |
| `writeCheckConfig` reuse | check_hint_test.go:105-108 `TestCheckHint_DefaultOff_ZeroNetwork` → `writeCheckConfig(t, "")` (helper at :30) |
| <2s goal | **0.458s real** (0.039s test time) vs 33.371s baseline |

## Correctness (Static Evidence)

| Check | Status | Notes |
|-------|--------|-------|
| Zero production behavior change (nil-default seams) | ✅ | All seams fall back to the production `buildAdapterList`/`clientFactory` on zero value; RunE bodies pass `cliDeps.<field>` whose zero value is production. Spec-neutral verdict holds |
| Injection is sequential-safe | ✅ | deps.go:9-11 — no `t.Parallel` in internal/cli (grep-verified); `setCLIDeps` restores via `t.Cleanup` (integration_test.go:23-31) |
| Gating set exact | ✅ | update.go:48-53 — apt/npm/nvm/pnpm only; brew/winget/custom exempt by predicate (`TrustOfficial && gatedOfficialAdapters[info.ID] && !UpdateAvailable`, update.go:183) |
| Real-subprocess coverage retained | ✅ | internal/adapters + internal/adapters/official untouched (their real-subprocess tests still run in `go test ./...`: adapters 0.263s, official 0.706s); smoke-test.sh 23/23 against fresh build |
| Probe security branches equivalent | ✅ | Fake Command strings drive `ClassifyCommand` (sudo → High, `&&` → Medium) and Privileges flow into `ConfirmConfig` (update.go:146-159); `os.Stdin` is /dev/null under `go test`, so interactive prompts deny identically (audit_probe_test.go:11-19 rationale) |
| 0 test coverage removed | ✅ | 13 conversions keep equivalent assertions; output assertions unchanged (per apply-progress); TestDryRun/TestCIMode gained `updated == false` |

## Coherence (Design)

| Decision | Followed? | Notes |
|----------|-----------|-------|
| D1 — nil-default seam shape (`checkDeps.buildAdapterList` + `listDeps`) | ✅ Yes | check.go:55-57, list.go:44-46 mirror update.go:65-67 exactly |
| D2 — package-level `cliDeps` var + `setCLIDeps(t, …)` helper with `t.Cleanup` | ✅ Yes | deps.go:12-17; integration_test.go:23-31; race-safety warning documented |
| D3 — single fake; name-as-ID | ✅ Yes | mockAdapter deleted (grep: 0 matches); all fake IDs are real domain IDs |
| D4 — probe conversion with `updated`-flag proof | ✅ Yes | all 5 probes assert `fake.updated`; `probeSetup` deleted (probe_test.go is now the `probeHome` helper only, probe_test.go:1-14) |
| D5 — check_hint DefaultOff reuses `writeCheckConfig` | ✅ Yes | check_hint_test.go:108 |
| D6 — what stays real (init tests, EmptyConfig, BuildAdapterList, adapters packages) | ✅ Yes | untouched per apply-progress; verified by fresh full-suite run |
| Scope boundaries (no behavior changes, no init.go/security/adapters changes) | ✅ Yes | diff limited to seams + tests; smoke 23/23 unchanged |

## Issues Found

**CRITICAL**: None

**WARNING**: None

**SUGGESTION** (recorded follow-ups + new observations):
1. **Gated-set catalog parity in fakes**: the gating matrix proves the gate with `apt` only (update_test.go:106,114); `npm` appears in flow tests, but `nvm`/`pnpm` never appear as fake IDs in any cli test. The parenthetical in the spec invariant ("apt, npm, nvm, pnpm") is thus only partially exercised — set membership for nvm/pnpm rests on production-code read of update.go:48-53. No synthetic IDs are used, so there is no silent gate bypass; a future test could add one nvm/pnpm gated row for direct catalog parity.
2. **Bare-upp seam gap** (recorded follow-up, archived `2026-08-12-upp-versioning-auto-update/verify-report.md`): no test executes bare `root.Execute()` with no args/unknown args through the seam'd RunE bodies — `TestRootCommand_NoArgs`/`TestRootCommand_Help` (integration_test.go:121,135) are untouched by this change and do not assert seam wiring; the seam path is only exercised by the 8 converted tests.
3. **Probe wiring coverage**: the 5 probes call `runUpdate` directly with `updateDeps` (unit-level), not via the cobra RunE → `cliDeps.update` path; the `--ci`/`--quiet` flag plumbing through `NewUpdateCommand` is covered by the converted cobra-entry tests, but probe+flag interactions are not jointly tested through the root command.

## Verdict

**PASS** — All 25/25 tasks complete; spec-neutral change with 0/0 requirements/scenarios; all 10 implementation invariants hold on main with file:line evidence. Fresh execution on fe79f8f: CLI suite 0.039s test / 0.458s real (baseline 33.371s — the <2s success criterion passes by ~73×), full suite exit 0, race exit 0, vet clean, gofmt empty, smoke 23/23 against a freshly built binary. No blockers, no CRITICAL findings, no WARNINGs; 3 suggestion-level follow-ups.
