# Design: adapter update-correctness hardening (timeouts, linux/arm64, official gating)

## Technical Approach

Three targeted fixes on the existing subprocess seams, plus a testability seam: (1) `exec.CommandContext` + `context.WithTimeout` inside the three production execution bodies — `shellExec` (custom.go:122) and the `runCmdFn`/`runCmdArgsFn` default bodies (helper.go:19-44) — wrappers untouched; (2) go.go:57's hardcoded amd64 tarball URL extracted into `goTarballURL(goarch)`; (3) a gate in `runUpdate` after the confirm switch so official adapters update only when `Check()` said so, implemented through a new `updateDeps` seam mirroring `checkDeps`/`selfUpdateDeps`. Implements all 11 spec scenarios; zero `context` usage today, so imports are additive.

## Architecture Decisions

| # | Options | Tradeoff | Decision |
|---|---|---|---|
| D1 | Timeout placement | Wrapper-level timeouts miss the 12 direct `runCmd` update call sites (brew.go:55, go.go:72, apt.go:62…) and constrain seam fakes; `commandOutput`/`shellOutput` call seam vars directly (helper.go:134,143) | `CommandContext`+`WithTimeout` inside seam default bodies + `shellExec`; wrappers and fakes untouched |
| D2 | Timeout values | 30s kills a legitimately slow `brew update && brew upgrade`; 10m makes a hung version lookup stall the run | `CheckTimeout=30s` (version lookups finish in <2s; 15x headroom, fail fast), `UpdateTimeout=10m` (brew/apt/nvm slowest, proposal ceiling) in new `internal/adapters/timeouts.go`; shell-form checks (apt-cache, nvm ls-remote; npm/pnpm already GNU-`timeout 15`-wrapped) inherit the 10m ceiling — bounded, no false failures |
| D3 | Seam signatures | Changing them churns ~15 table rows + `exec_mock_test.go` + custom_test.go:231 | Signatures unchanged; per-op duration via internal `shellExecWithTimeout(command, timeout)` (Check→30s, Update→10m); `runCmdFn`→10m, `runCmdArgsFn`→30s. Timeout errors surface as `errors.Is(err, context.DeadlineExceeded)` (CommandContext kills; `%w` chains preserved); CLI maps via new `timeoutErr(name, op, err)` helper at update.go:91/156/177 and check.go:91 → structured message naming tool/op/limit, loop continues |
| D4 | Gating placement | Extra `Check()` call would double subprocesses | Reuse `updateInfo` from the existing check (update.go:90); insert between confirm switch (152) and `Update(false)` (155): `info.Trust == adapters.TrustOfficial && !updateInfo.UpdateAvailable` → `StatusCurrent{Version}`, continue. Custom (`TrustCustom*`) and winget/scoop (`true` by design) pass the rule naturally; dynamic adapters gated by result. Dry-run branch (102-118) unchanged |
| D5 | `runUpdateDeps` seam | Adding a param churns `NewUpdateCommand`; without it gating is untestable at CLI level | `type updateDeps struct{ buildAdapterList func(cfg *config.Config, osName string) []adapters.Adapter }`; zero value → production default (mirrors check.go:38-40); `runUpdate(gf, uf, deps)`; tests inject fakes; probe test (audit_probe_test.go:77-89) runs the production path and stays green via custom exemption |
| D6 | `goTarballURL(goarch)` | `runtime.GOARCH` is compile-time — amd64 CI can't exercise arm64 inline | `func goTarballURL(goarch string) string` in official/go.go returning the `linux-<goarch>.tar.gz` template; call site go.go:57 composes the curl\|tar pipeline with `runtime.GOARCH`; table tests amd64/arm64 |
| D7 | Scope boundaries | Creep risks review budget | No custom-id collision fix; no version-helper dedup; no dead `Update(dryRun)` cleanup; no GNU `timeout 15` wrapper changes (npm.go:31, pnpm.go:33); `lookPathFn` unbounded (no subprocess) |
| D8 | Timeout kill semantics | `CommandContext` kills only the direct child; `sh -c` grandchildren (`curl \| sh`) can linger | Accepted, documented limitation; durations as test-overridable vars enable a real kill-path integration test (`sleep 2` with 100ms timeout) |

## Data Flow

```
runUpdate (per adapter):  Check() ──err──▶ StatusFailed (timeout → timeoutErr msg)
   │ updateInfo                                  │ loop continues (spec: run continues)
   ▼
 Confirm gate (unchanged) ──Deny/Error──▶ StatusSkipped / StatusFailed
   │ Proceed
   ▼
 NEW gate: Trust==Official && !UpdateAvailable ──▶ StatusCurrent, continue
   │ pass
   ▼
 Update(false) ──timeout──▶ StatusFailed (timeoutErr msg); loop continues
```

## File Changes

| File | Action | Description |
|---|---|---|
| `internal/adapters/timeouts.go` | Create | `CheckTimeout`/`UpdateTimeout` vars (single source, test-overridable) |
| `internal/adapters/custom.go` | Modify | `shellExecWithTimeout` core; `shellExec` delegates with UpdateTimeout (signature kept); Check/Update call per-op variant |
| `internal/adapters/official/helper.go` | Modify | Seam default bodies → `CommandContext`+`WithTimeout` (runCmdFn→Update, runCmdArgsFn→Check); wrappers untouched |
| `internal/adapters/official/go.go` | Modify | `goTarballURL(goarch)`; call site with `runtime.GOARCH` |
| `internal/cli/update.go` | Modify | `updateDeps` seam; `runUpdate(gf, uf, deps)`; gating block; `timeoutErr` mapping |
| `internal/cli/check.go` | Modify | `timeoutErr` mapping at check-error site (91) |
| `internal/cli/update_test.go` | Create | Gating matrix via `updateDeps` fakes |
| `internal/adapters/official/go_arch_test.go` | Create | amd64/arm64 URL table |
| `internal/adapters/official/timeout_test.go` | Create | DeadlineExceeded fake propagation + kill-path test |
| `internal/adapters/custom_test.go` | Modify | shellExec kill-path timeout test |
| `internal/cli/audit_probe_test.go` | Unchanged | Preserved — custom exempt from gating |

## Interfaces / Contracts

```go
// internal/adapters/timeouts.go
var (
    CheckTimeout  = 30 * time.Second  // version lookups; fail fast
    UpdateTimeout = 10 * time.Minute  // brew/apt/nvm slowest
)
```
```go
// internal/cli/update.go — mirrors checkDeps (check.go:38-40)
type updateDeps struct {
    buildAdapterList func(cfg *config.Config, osName string) []adapters.Adapter
}
```
```go
// gating block, after Confirm switch, before a.Update(false)
if info.Trust == adapters.TrustOfficial && !updateInfo.UpdateAvailable {
    results = append(results, output.ToolResult{Name: info.Name,
        Status: output.StatusCurrent, Version: updateInfo.CurrentVersion})
    continue
}
```
```go
// internal/adapters/official/go.go
func goTarballURL(goarch string) string {
    return fmt.Sprintf("https://go.dev/dl/$(curl -fsSL https://go.dev/VERSION?m=text | head -1).linux-%s.tar.gz", goarch)
}
```

## Testing Strategy

| Layer | What to Test | Approach |
|---|---|---|
| Unit (official) | Timeout propagation: seam fakes return `context.DeadlineExceeded`; adapter `Result.Error`/`UpdateInfo` error `errors.Is`-detectable | `setExecFakes` + new timeout_test.go |
| Unit (kill path) | Real `sleep 2` via seam body with overridden 100ms timeout → `DeadlineExceeded` | Var override + `t.Cleanup` restore (official + custom) |
| Unit (custom) | shellExec per-op timeout paths | custom_test.go additions |
| Unit (go arch) | `goTarballURL` amd64/arm64 | Table test, go_arch_test.go |
| Integration (gating) | Matrix: official true→Update called; official false→`StatusCurrent`, Update NOT called; custom false→Update called; winget/scoop true→called; dynamic false→skipped; Check timeout→structured error, other tools still update | `updateDeps` fakes + `withCapturedStdout`, update_test.go |
| Regression | `TestProbe_TrustedLowRisk_Executes` still executes (custom exempt) | audit_probe_test.go unchanged |
| E2E | `scripts/smoke-test.sh --skip-build` | Full gates: `go test ./... -count=1 -race`, `go vet ./...`, gofmt |

## Threat Matrix

| Boundary | Applicability | Design response | Planned RED tests |
|---|---|---|---|
| Documentation-like paths | N/A — no classification or command-string changes; execution surface only gains bounds and gating | — | timeout kill-path per seam |
| Git repository selection | N/A — no git ops | — | — |
| Commit state | N/A — no commit ops | — | — |
| Push state | N/A — no push ops | — | — |
| PR commands | N/A — no PR automation | — | — |

## Migration / Rollout

No migration. Documented behavior change: the six stubs (brew, bun, docker, gh, go, opencode) now report `StatusCurrent` and stop updating — intended per spec (their `UpdateAvailable: false` is by design; winget/scoop and custom are exempt). Rollback: single PR revert (~150-250 lines, under the 400-line review budget) restores unconditional updates, no timeouts, amd64-only URL.

## Open Questions

None.
