# Tasks: Per-Adapter Update Policy and Check-Failure Signal

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~200–300 (14 modified, 0 created) |
| 400-line budget risk | Medium |
| Chained PRs recommended | Yes |
| Suggested split | PR 1 enum+gate+explicitness → PR 2 failure signal |
| Delivery strategy | auto-chain |
| Chain strategy | pending |

Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: pending
400-line budget risk: Medium

Recommendation: 2 chained PRs — one PR also fits, but the two concerns (metadata gating vs subprocess error signaling) are independent. Once chain strategy resolves, stacked-to-main fits both slices landing on main.

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|----------------------|-----------------|-------------------|
| 1 | UpdatePolicy enum, 13 explicit Info() sites, gate rewrite, goldens/consistency, fake policy explicitness | PR 1 | `go test ./internal/... -count=1 -race` | N/A — metadata-only behavior; gate exercised via seam fakes, no real tools | revert interface.go, 13 Info() sites, cli/update.go + their tests |
| 2 | Error-aware helpers, apt/nvm/npm/pnpm Check() propagation, npm/pnpm exit-1 interpretation | PR 2 | `go test ./internal/adapters/official/ -count=1 -race` | `bash scripts/smoke-test.sh --skip-build` (stubs → StatusCurrent; failed gated check → StatusFailed) | revert helper.go, apt.go, nvm.go, npm.go, pnpm.go + tests |

## Phase 1: UpdatePolicy Foundation (PR 1)

- [x] 1.1 RED: `official/info_test.go` goldens — each `want` row gains expected `UpdatePolicy` (apt/npm/pnpm/nvm → PolicyGated; brew/bun/docker/gh/go/opencode/winget/scoop → PolicyAlwaysUpdate); DeepEqual fails while field missing
- [x] 1.2 RED: `official/adapter_update_test.go` `TestAllAdapters_InfoConsistency` — assert `UpdatePolicy ∈ {PolicyGated, PolicyAlwaysUpdate}`; fails on undeclared values
- [x] 1.3 GREEN: `adapters/interface.go` — `UpdatePolicy` type + `PolicyGated=0`/`PolicyAlwaysUpdate=1` next to TrustLevel (:67-74), `ToolInfo` field, invariant comment citing TrustLevel zero convention (:10-12); zero MUST stay Gated
- [x] 1.4 GREEN: 13 Info() sites — apt/nvm/npm/pnpm → PolicyGated; brew/bun/docker/gh/go/opencode/winget/scoop (8 files) + custom.go:100-107 → PolicyAlwaysUpdate

## Phase 2: Gate Rewrite + Explicitness (PR 1)

- [x] 2.1 RED: `cli/update_test.go` — `fakeUpdateAdapter` gains `policy`; matrix re-keyed ID→policy (ID stays label): Gated+true→called; Gated+false→StatusCurrent, not called; AlwaysUpdate+false (stub/winget/custom)→called; failed gated check→StatusFailed, never current
- [x] 2.2 RED: `cli/update_test.go`, `integration_test.go`, `audit_probe_test.go` — all 21 `fakeUpdateAdapter` literals set `policy` explicitly (hermetic assertions unchanged)
- [x] 2.3 GREEN: `cli/update.go` — gate :183 → `info.UpdatePolicy == adapters.PolicyGated && !updateInfo.UpdateAvailable` → StatusCurrent; delete `gatedOfficialAdapters` (:43-48 + comment); dry-run branch untouched

## Phase 3: Check Failure Signal (PR 2)

- [x] 3.1 RED: `official/adapter_update_test.go` — seam tests for `commandOutputErr`/`shellOutputErr`: success passthrough; `setExecFakes` error → structured error; real `sh -c exit N` child (precedent TestRunCmd_*:268-292) → exit code via `errors.As(*exec.ExitError)`; fake DeadlineExceeded stays `errors.Is`-detectable (%w)
- [x] 3.2 RED: `official/check_test.go` — apt/nvm Check(): command-fails rows flip `wantErr=true` (tool/op/exit code); empty-output rows stay `wantErr=false`
- [x] 3.3 RED: `check_test.go` — npm/pnpm cmd constants maskless; exit-1 outdated → `update_available=true`, no error; exit-124 → structured error; other non-zero → error; timeout fake stays DeadlineExceeded-detectable
- [x] 3.4 GREEN: `official/helper.go` — `commandOutputErr`/`shellOutputErr` delegating to `runCmdArgsFn`/`runCmdFn`; failure → `"<tool> check failed (exit N): <stderr excerpt>: %w"`, exit omitted when not extractable
- [x] 3.5 GREEN: `apt.go`/`nvm.go` — Check() detection reads (installed/candidate, current/remote) via error-aware variants; display-only reads keep plain variants (D5)
- [x] 3.6 GREEN: `npm.go`/`pnpm.go` — drop `|| true` (npm.go:31, pnpm.go:33); Go exit interpretation: code 1 → valid detection (stdout decides availability), other non-zero (incl. 124) → structured failure; shell `timeout 15` kept

## Phase 4: Verification

- [x] 4.1 PR 1 gate: `go test ./internal/... -count=1 -race` — goldens, consistency, matrix green
- [x] 4.2 PR 2 gate: `go test ./internal/adapters/official/ -count=1 -race` — helper variants, check rows, exit-1/124
- [x] 4.3 `go test ./... -count=1`
- [x] 4.4 `go test ./... -race`
- [x] 4.5 `go vet ./...` + `gofmt -s -w`
- [x] 4.6 `bash scripts/smoke-test.sh --skip-build` (stubs → StatusCurrent; failed gated check → StatusFailed)
