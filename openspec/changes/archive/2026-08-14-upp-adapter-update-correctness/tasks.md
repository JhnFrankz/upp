# Tasks: adapter update-correctness hardening (timeouts, linux/arm64, official gating)

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~150–250 (4 created, 6 modified) |
| 400-line budget risk | Medium |
| Chained PRs recommended | Yes |
| Suggested split | PR 1 timeouts+arch → PR 2 gating+seam |
| Delivery strategy | ask-on-risk |
| Chain strategy | pending |

Decision needed before apply: Yes
Chained PRs recommended: Yes
Chain strategy: pending
400-line budget risk: Medium

Recommendation: 2 chained PRs — fits 400 lines as one PR, but crosses two independent concerns. Single PR also viable.

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|----------------------|-----------------|-------------------|
| 1 | Timeouts: vars, shellExecWithTimeout, seam bodies | PR 1 | `go test ./internal/adapters/... -count=1 -race` | 100ms override, `sleep 2` kill-path (custom+official) | revert timeouts.go, custom.go, helper.go + tests |
| 2 | goTarballURL + runtime.GOARCH | PR 1 | `go test ./internal/adapters/official/ -run GoTarballURL -count=1` | N/A — pure string fn, compile+unit gate | revert go.go + go_arch_test.go |
| 3 | updateDeps seam + gating + timeoutErr | PR 2 | `go test ./internal/cli/ -run 'Update|Check' -count=1 -race` | `bash scripts/smoke-test.sh --skip-build` (six stubs → StatusCurrent) | revert update.go, check.go, update_test.go |
| 4 | Gating matrix + full gates | PR 2 | `go test ./... -count=1 -race` | `bash scripts/smoke-test.sh --skip-build` | same as unit 3 |

## Phase 1: Timeouts Foundation

- [x] 1.1 RED: `internal/adapters/custom_test.go` — `UpdateTimeout`=100ms; `shellExec` `sleep 2` → `errors.Is(err, context.DeadlineExceeded)`
- [x] 1.2 RED: `custom_test.go` — Check honors `CheckTimeout` override → DeadlineExceeded
- [x] 1.3 GREEN: `internal/adapters/timeouts.go` — `CheckTimeout=30s`, `UpdateTimeout=10m` vars
- [x] 1.4 GREEN: `custom.go` — `shellExecWithTimeout(command, timeout)` core; `shellExec` delegates with UpdateTimeout (signature kept); Check/Update per-op variant
- [x] 1.5 RED: `official/timeout_test.go` — seam fakes return DeadlineExceeded; errors `errors.Is`-detectable (setExecFakes)
- [x] 1.6 RED: `timeout_test.go` — kill-path `sleep 2`, 100ms → DeadlineExceeded
- [x] 1.7 GREEN: `official/helper.go` — seam bodies → CommandContext+WithTimeout (runCmdFn→UpdateTimeout, runCmdArgsFn→CheckTimeout); wrappers untouched

## Phase 2: Go Architecture

- [x] 2.1 RED: `official/go_arch_test.go` — table: amd64→`go*linux-amd64.tar.gz`; arm64→`go*linux-arm64.tar.gz`
- [x] 2.2 GREEN: `official/go.go` — `goTarballURL(goarch)` (design block); go.go:57 → `runtime.GOARCH`

## Phase 3: Gating + Seam

- [x] 3.1 RED: `internal/cli/update_test.go` — matrix via `updateDeps` fakes: official true→called; false→StatusCurrent, NOT called; custom false→called; winget/scoop→called; dynamic false→skipped
- [x] 3.2 RED: `update_test.go` — Check timeout → tool/op/limit error; other tools still update
- [x] 3.3 GREEN: `update.go` — `updateDeps` seam (mirrors check.go:38-40, zero→prod default); `runUpdate(gf, uf, deps)` + call sites
- [x] 3.4 GREEN: `update.go` — gate between confirm (152) and `Update(false)` (155): `TrustOfficial && !UpdateAvailable` → `StatusCurrent{Version}`, continue
- [x] 3.5 GREEN: `update.go` 91/156/177 + `check.go` 91 — `timeoutErr(name, op, err)`

## Phase 4: Verification

- [x] 4.1 Regression: `TestProbe_TrustedLowRisk_Executes` (audit_probe_test.go:77-89) stays green — custom exempt
- [x] 4.2 `go test ./... -count=1`
- [x] 4.3 `go test ./... -race`
- [x] 4.4 `go vet ./...` + `gofmt -s -w`
- [x] 4.5 `bash scripts/smoke-test.sh --skip-build`
