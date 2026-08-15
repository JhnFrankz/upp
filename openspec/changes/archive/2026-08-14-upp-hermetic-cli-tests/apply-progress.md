# Apply Progress: Hermetic CLI Tests

**Change**: upp-hermetic-cli-tests
**Mode**: Strict TDD (test refactor — RED = baseline suite green at ~35s; success gate = final <2s)
**Batch**: All 25 tasks (single PR, user decision)
**Date**: 2026-08-14

## TDD Cycle Evidence

This is a TEST refactor: no new production behavior is introduced. The RED evidence is the pre-change baseline suite staying green through the seam introduction; GREEN is each conversion keeping the suite green with equivalent fake-based assertions; the final gate is the <2s measurement.

| Task | Test File | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|------|-----------|-------|------------|-----|-------|-------------|----------|
| 1.1 deps.go | N/A (new, var only) | N/A | N/A (new) | Baseline 33.371s green | Suite green 33.887s after seams | N/A (structural) | N/A |
| 1.2 checkDeps+buildAdapterList | check_hint_test.go, integration_test.go | Unit/Integration | ✅ full suite 33.371s | Baseline covers | ✅ green after seam | ✅ zero-value + injected | ✅ nil-default shape |
| 1.3 listDeps+runList(gf,deps) | integration_test.go | Integration | ✅ | Baseline covers | ✅ green | ✅ zero-value + injected | ✅ nil-default shape |
| 1.4 RunE wirings ×3 | integration_test.go | Integration | ✅ | Baseline covers | ✅ green | N/A (structural) | N/A |
| 1.5 Baseline guard | internal/cli suite | — | — | ✅ 33.371s recorded | ✅ 33.887s (no behavior change) | N/A | N/A |
| 1.6 setCLIDeps helper | integration_test.go | Integration | ✅ | Helper unused until 3.x | ✅ compiles | ✅ all four fields + restore | ✅ t.Cleanup |
| 2.1 fake +command/+privileges | update_test.go | Unit | ✅ | N/A (fake extension) | ✅ | ✅ Info() returns them | ✅ |
| 2.2 runUpdateWithFlags | update_test.go | Unit | ✅ | N/A | ✅ 0.008s focused run | ✅ CI + quiet flags | ✅ shared fakeAdapterList |
| 2.3 mockAdapter retired | integration_test.go | Unit | ✅ | N/A | ✅ TestAdapterIDs/ByID green | ✅ real IDs apt/brew/npm | ✅ |
| 2.4 probeSetup retired | probe_test.go | — | ✅ | N/A | ✅ probeHome kept, consumers green | N/A | ✅ |
| 3.1 ListCommand_NoConfig | integration_test.go | Integration | ✅ | N/A (conversion) | ✅ | ✅ fake+seam | ✅ |
| 3.2 CheckCommand_NoConfig | integration_test.go | Integration | ✅ | N/A | ✅ | ✅ fake+seam | ✅ |
| 3.3 CIMode_RejectsUntrusted | integration_test.go | Integration | ✅ | N/A | ✅ + updated==false | ✅ medium && untrusted | ✅ |
| 3.4 DryRun_NoCommandsExecuted | integration_test.go | Integration | ✅ | N/A | ✅ + updated==false | ✅ UpdateAvailable=true | ✅ |
| 3.5 QuietMode_SuppressesProgress | integration_test.go | Integration | ✅ | N/A | ✅ | ✅ two fakes (multi-tool) | ✅ |
| 3.6 UpdateFlow_ConfigToSummary | integration_test.go | Integration | ✅ | N/A | ✅ | ✅ fake dry-run | ✅ |
| 3.7 InitCheckUpdateLifecycle | integration_test.go | Integration | ✅ | N/A | ✅ | ✅ init real + check/update fake | ✅ |
| 3.8 CheckCommand_SummaryOutput | integration_test.go | Integration | ✅ | N/A | ✅ same assertions | ✅ | ✅ |
| 3.9 Probe CI High trusted | audit_probe_test.go | Integration | ✅ | N/A | ✅ updated==false | ✅ High via sudo | ✅ |
| 3.10 Probe Interactive High trusted | audit_probe_test.go | Integration | ✅ | N/A | ✅ deny, updated==false | ✅ | ✅ |
| 3.11 Probe Interactive High untrusted | audit_probe_test.go | Integration | ✅ | N/A | ✅ deny, updated==false | ✅ | ✅ |
| 3.12 Probe Trusted Low executes | audit_probe_test.go | Integration | ✅ | N/A | ✅ updated==true | ✅ | ✅ |
| 3.13 Probe Quiet Medium prompts | audit_probe_test.go | Integration | ✅ | N/A | ✅ "Proceed? [y/N]" + deny | ✅ | ✅ |
| 3.14 CheckHint_DefaultOff | check_hint_test.go | Unit | ✅ | N/A | ✅ zero client constructions | ✅ writeCheckConfig("") | ✅ |
| 4.1–4.5 Gates | — | — | — | — | All green (below) | — | — |

## Test Summary

- **Tests written/changed**: 13 conversions (8 cobra-entry + 5 probes) + fake extension + 2 helpers; 0 test coverage removed
- **Layers**: Unit (gating matrix, hint), Integration (cobra-entry, probes), E2E (smoke unchanged)
- **Approval tests**: N/A — conversions replace real-subprocess execution with equivalent fake-based assertions; output assertions unchanged
- **Pure functions created**: `fakeAdapterList` (builder closure)

## Timing Evidence

| Milestone | Command | Result |
|-----------|---------|--------|
| Baseline (RED) | `time go test ./internal/cli/ -count=1` | ok 33.371s (real 33.777s) |
| After seams (1.5) | same | ok 33.887s (real 34.448s) — zero behavior change |
| After conversions | same | ok 0.034s |
| **Final gate (4.1)** | `time go test ./internal/cli/ -count=1` | **ok 0.036s (real 0.418s) — < 2s PASS** |

## Gate Results (Phase 4)

| Gate | Command | Result |
|------|---------|--------|
| 4.1 | `time go test ./internal/cli/ -count=1` | ✅ 0.036s test time / 0.418s real (<2s) |
| 4.2 | `go test ./... -count=1` | ✅ all packages ok |
| 4.3 | `go test -race ./...` | ✅ all packages ok |
| 4.4 | `go vet ./...`; `gofmt -l .` | ✅ vet clean; gofmt empty output |
| 4.5 | `bash scripts/smoke-test.sh --skip-build` | ✅ 23 passed, 0 failed |

## Work Unit Evidence

- **Focused test command**: `go test ./internal/cli/ -run 'RunUpdate|Adapter|Probe' -count=1` → ok 0.008s
- **Runtime harness**: `bash scripts/smoke-test.sh --skip-build` → 23/23 passed (real-adapter E2E proof intact)
- **Rollback boundary**: single revert — production touch is nil-default seams only: `internal/cli/deps.go` (new), `check.go`, `list.go` (seam + nil-default + RunE lines), `update.go`, `selfupdate.go` (RunE lines only); test files `update_test.go`, `integration_test.go`, `audit_probe_test.go`, `probe_test.go`, `check_hint_test.go`. Reverting leaves `init.go`, `internal/adapters`, `internal/security` untouched.

## Files Changed

| File | Action | What Was Done |
|------|--------|---------------|
| `internal/cli/deps.go` | Created | `cliDeps` package-level injection var + race warning doc |
| `internal/cli/check.go` | Modified | `checkDeps.buildAdapterList` field; nil-default before :52; RunE → `cliDeps.check` |
| `internal/cli/list.go` | Modified | `listDeps`; `runList(gf, deps)`; nil-default; RunE → `cliDeps.list` |
| `internal/cli/update.go` | Modified | RunE → `cliDeps.update` only |
| `internal/cli/selfupdate.go` | Modified | RunE → `cliDeps.selfUpdate` only |
| `internal/cli/update_test.go` | Modified | fake +command/+privileges (Info returns them); `fakeAdapterList`; `runUpdateWithFlags`; `runUpdateWith` refactored |
| `internal/cli/integration_test.go` | Modified | `mockAdapter` deleted; `setCLIDeps` added; 8 tests → cliDeps + fake; TestAdapterIDs/ByID → fake |
| `internal/cli/audit_probe_test.go` | Modified | 5 probes → fake-based with `updated`-flag assertions; `runUpdateCmd` deleted |
| `internal/cli/probe_test.go` | Modified | `probeSetup` deleted; `probeHome` kept |
| `internal/cli/check_hint_test.go` | Modified | DefaultOff → `writeCheckConfig(t, "")` |

## Deviations from Design

None — implementation matches design. All 7 decisions applied as specified (nil-default seams, package-level `cliDeps`, single fake, probe conversion with `updated` flag, `writeCheckConfig` reuse, what stays real, scope boundaries).

## Issues Found

None. Note: task 2.4 (delete `probeSetup`) temporarily breaks compilation until task 3.9 (probe conversions) — the probes were converted immediately after (same batch), so no intermediate state was left red.

## Status

25/25 tasks complete. Ready for verify.
