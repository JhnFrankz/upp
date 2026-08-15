# Tasks: Hermetic CLI Tests

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~200–300 |
| 400-line budget risk | Medium |
| Chained PRs recommended | No |
| Suggested split | Single PR; optional 2-PR split |
| Delivery strategy | ask-on-risk |
| Chain strategy | pending |

Decision needed before apply: Yes
Chained PRs recommended: No
Chain strategy: pending
400-line budget risk: Medium

Recommendation: single PR (test-only, one revert); split only if tracking toward 400.

### Work Units

| Unit | Goal | Likely PR | Test command | Harness | Rollback |
|------|------|-----------|--------------|---------|----------|
| 1 | Seams: deps.go, checkDeps/listDeps, RunE wirings | Single PR c1 | `time go test ./internal/cli/ -count=1` | `bash scripts/smoke-test.sh --skip-build` | check.go, list.go, deps.go, update.go, selfupdate.go |
| 2 | Fakes: fake extension; retire mockAdapter/probeSetup | Single PR c2 | `go test ./internal/cli/ -run 'RunUpdate|Adapter|Probe' -count=1` | N/A — helpers only | update_test.go, probe_test.go |
| 3 | 8 cobra-entry conversions | Single PR c3 | `go test ./internal/cli/ -run 'ListCommand|CheckCommand|CIMode|DryRun|QuietMode|UpdateFlow|Lifecycle|SummaryOutput' -count=1` | N/A — fake integration; E2E in smoke | integration_test.go |
| 4 | Probes + check_hint + timing gate | Single PR c4 | `go test ./internal/cli/ -run TestProbe -count=1`; `time go test ./internal/cli/ -count=1` <2s | `bash scripts/smoke-test.sh --skip-build` | audit_probe_test.go, check_hint_test.go; one revert |

## Phase 1: Seams (nil-default, behavior-neutral)

- [x] 1.1 Create `internal/cli/deps.go`: `var cliDeps struct{check checkDeps; update updateDeps; list listDeps; selfUpdate selfUpdateDeps}`; zero=production, sequential-only
- [x] 1.2 `check.go`: `checkDeps` + `buildAdapterList`; nil-default before :52
- [x] 1.3 `list.go`: `listDeps`; `runList(gf, deps)`; nil-default; RunE → `cliDeps.list`
- [x] 1.4 RunE wirings: `NewCheckCommand`/`NewUpdateCommand`/`NewSelfUpdateCommand` → `cliDeps.check/update/selfUpdate`
- [x] 1.5 Baseline guard (RED): suite green ~35s, record timing
- [x] 1.6 `integration_test.go`: `setCLIDeps(t, …)` helper, `t.Cleanup` restore

## Phase 2: Fakes

- [x] 2.1 `update_test.go`: `fakeUpdateAdapter` + `command`/`privileges`; `Info()` returns them; `updated` flag
- [x] 2.2 `update_test.go`: generalize `runUpdateWith` for CI/quiet
- [x] 2.3 `integration_test.go`: delete `mockAdapter`; `TestAdapterIDs`/`TestAdapterByID` → fake, real IDs (apt/npm/nvm/pnpm)
- [x] 2.4 `probe_test.go`: delete `probeSetup`; keep `probeHome`

## Phase 3: Conversions (all green)

- [x] 3.1 `TestListCommand_NoConfig`: `setCLIDeps` + fake
- [x] 3.2 `TestCheckCommand_NoConfig`: `setCLIDeps` + fake
- [x] 3.3 `TestCIMode_RejectsUntrustedCustomTools`: fake, real custom ID
- [x] 3.4 `TestDryRun_NoCommandsExecuted`: fake; `updated == false`
- [x] 3.5 `TestQuietMode_SuppressesProgress`: fake
- [x] 3.6 `TestUpdateFlow_ConfigToSummary`: fake
- [x] 3.7 `TestInitCheckUpdateLifecycle`: fake
- [x] 3.8 `TestCheckCommand_SummaryOutput`: fake; output assertions unchanged
- [x] 3.9 `TestProbe_TrustedCustomHighRisk_CI`: High (`sudo rm -rf`); `updated == false`
- [x] 3.10 `TestProbe_TrustedCustomHighRisk_Interactive`: High; deny, not executed
- [x] 3.11 `TestProbe_UntrustedCustomHighRisk_Interactive`: deny, not executed
- [x] 3.12 `TestProbe_TrustedLowRisk_Executes`: trusted Low; `updated == true`
- [x] 3.13 `TestProbe_QuietMediumRisk_StillPrompts`: `&&` Medium, --quiet; prompt+deny
- [x] 3.14 `check_hint_test.go` `TestCheckHint_DefaultOff_ZeroNetwork` → `writeCheckConfig(t, "")`

## Phase 4: Verification

- [x] 4.1 `time go test ./internal/cli/ -count=1` < 2s (record)
- [x] 4.2 `go test ./... -count=1`
- [x] 4.3 `go test -race ./...`
- [x] 4.4 `go vet ./...`; `gofmt -l .` empty
- [x] 4.5 `bash scripts/smoke-test.sh --skip-build`
