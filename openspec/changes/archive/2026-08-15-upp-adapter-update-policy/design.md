# Design: Per-Adapter Update Policy and Check-Failure Signal

## Technical Approach

Two mechanics on existing seams. (1) Update gating moves from the CLI-side `gatedOfficialAdapters` ID list (update.go:48-53) into per-adapter metadata: an `UpdatePolicy` enum in the adapters package, a field on `ToolInfo`, explicit at all 13 `Info()` sites, consumed by a rewritten gate at update.go:183 — the list is deleted. (2) `check()` stops swallowing subprocess failures: error-aware helper variants `commandOutputErr`/`shellOutputErr` (helper.go, next to the originals, delegating to the SAME seam vars) make apt/nvm/npm/pnpm `Check()` return structured errors (tool/op/exit-code, `%w`-chained) on real subprocess failure; empty output stays "unknown"; npm/pnpm drop `|| true` with Go-level exit-code interpretation. The CLI already maps errors to `StatusFailed` (check.go:95-103, update.go:114-123, render.go:194/220) — zero CLI changes. Implements delta spec tool-adapter: MODIFIED Update Gating (7 scenarios) + ADDED Check Failure Signal (4 scenarios).

## Architecture Decisions

| # | Options | Tradeoff | Decision |
|---|---|---|---|
| D1 | Enum shape | A third "Dynamic" value is a synonym: gating by the live `check()` result IS the dynamic policy; `PolicyGated`=0 fail-closed mirrors the TrustLevel zero convention (interface.go:10-12: "unset resolves to least-privileged") so a missed site never updates unrequested | Two values: `PolicyGated UpdatePolicy = 0` / `PolicyAlwaysUpdate UpdatePolicy = 1`, next to TrustLevel; invariant comment citing the TrustLevel convention; zero MUST stay PolicyGated |
| D2 | Gate rewrite | `info.Trust == adapters.TrustOfficial &&` (update.go:183) is now redundant: every official adapter declares its policy explicitly and custom.go declares AlwaysUpdate — trust no longer participates in gating | `if info.UpdatePolicy == adapters.PolicyGated && !updateInfo.UpdateAvailable` → StatusCurrent; `gatedOfficialAdapters` (:48-53) deleted (single consumer :183); dry-run branch (:126-142) untouched; behavior-identical for all 12 official + custom |
| D3 | Helper variants | New signatures call the same seam vars → `exec_mock_test.go` and all seam fakes keep working; error shape must survive CLI rendering (render.go prints `Error.Error()` inline) and `timeoutErr` (update.go:242, `errors.Is` DeadlineExceeded) | `commandOutputErr(name, args...) (string, error)` / `shellOutputErr(command) (string, error)` in helper.go delegating to `runCmdArgsFn`/`runCmdFn`; failure → `"<tool> check failed (exit N): <stderr excerpt>: %w"` — exit code via `errors.As(*exec.ExitError)` (omitted when not extractable), seam error `%w`-chained so timeout mapping survives |
| D4 | npm/pnpm timeout layering | Literal "drop `|| true`" breaks the adapters: `npm outdated`/`pnpm outdated` exit 1 when outdated packages exist (documented convention) — a maskless error-on-non-zero check would report failure exactly when an update is available and never update. Alternatives: `pnpm --no-exit-code` (version-dependent flag risk), shell `[ $? -le 1 ]` (opaque) | Keep shell `timeout 15` as the effective check bound (helper ceiling is 10m via runCmdFn); drop `|| true`; interpret exit codes in Go: `ExitError` code 1 = valid detection (stdout decides availability), any other non-zero — incl. GNU timeout's 124 — = structured failure → StatusFailed. Interplay documented: 15s shell bound fires first; seam 30s CheckTimeout only bounds runCmdArgs version reads; shell-form check timeout message may claim 30s while the real ceiling is 10m (pre-existing, accepted) |
| D5 | apt "(none)" | apt-cache policy output "(none)"/empty means "no candidate" not "failed"; only a non-zero exit is a failure signal | Error-aware variants used for the four `Check()` detection reads (apt installed/candidate, nvm current/remote); display-only reads (`CurrentVersion()`, `CurrentVersion`/before/after in Update) keep the plain variants — failures there fall back to "unknown", never gate-relevant |
| D6 | Fake/policy explicitness | zero=Gated means a fake that omits policy silently becomes gated — the gating matrix flips (exempt rows skip updates). A missed PRODUCTION site for an AlwaysUpdate adapter is caught by goldens (DeepEqual); a missed Gated site is indistinguishable from explicit Gated — mitigated by zero fail-closed (safe direction) | `fakeUpdateAdapter` gains `policy` field (update_test.go:21-54); the matrix re-keys from ID to policy (rows carry `policy`; ID remains for labels); `TestAllAdapters_InfoConsistency` asserts policy ∈ {Gated, AlwaysUpdate} (any other value fails); every hermetic fake site (update_test.go, integration_test.go, audit_probe_test.go, ~19 literals) sets policy explicitly; matrix behavioral assertions (wantUpdated) guard fakes |
| D7 | Scope boundaries | Creep risks review budget | No failure-aware `upp check` exit code (flag only); no list.go:66-69 swallow fix; no renderer changes; winget/scoop stubs untouched (AlwaysUpdate, discarded output); no CheckTimeout-class migration for npm/pnpm |

## Data Flow

```
Check(): detection subprocess ──exit 0 + output──▶ parse → UpdateInfo
              │ exit 0, empty/"(none)"                (update_available=false, no error)
              ▼
          "unknown" status
              │ non-zero exit (incl. 124 timeout)
              ▼
     structured error (tool/op/exit) ──%w──▶ CLI StatusFailed (timeout → timeoutErr msg)

runUpdate per adapter:  UpdatePolicy==Gated && !UpdateAvailable ──▶ StatusCurrent, continue
                        else ──▶ Update(false)
```

## File Changes

| File | Action | Description |
|---|---|---|
| `internal/adapters/interface.go` | Modify | `UpdatePolicy` enum + ToolInfo field (:67-74), invariant comment mirroring TrustLevel |
| `internal/adapters/official/helper.go` | Modify | `commandOutputErr`/`shellOutputErr` + exit-code extraction + structured failure builder |
| `internal/adapters/official/apt.go`, `nvm.go`, `npm.go`, `pnpm.go` | Modify | Info(): PolicyGated; Check(): error-aware detection reads; npm/pnpm maskless + exit-1 interpretation |
| `internal/adapters/official/brew.go`, `bun.go`, `docker.go`, `gh.go`, `go.go`, `opencode.go`, `winget.go`, `scoop.go` | Modify | Info(): PolicyAlwaysUpdate (8 sites) |
| `internal/adapters/custom.go` | Modify | Info(): PolicyAlwaysUpdate (:100-107) |
| `internal/cli/update.go` | Modify | Gate :183 → policy predicate; delete `gatedOfficialAdapters` :48-53 + comment |
| `internal/adapters/official/info_test.go` | Modify | Goldens gain per-adapter policy; consistency test adds declared-value assertion |
| `internal/adapters/official/check_test.go` | Modify | npm/pnpm cmd constants maskless; error rows flip `wantErr`; new exit-1-outdated row; new structured-exit-code row |
| `internal/adapters/official/adapter_update_test.go` | Modify | Helper-variant seam tests (success / failure / exit-code rows) |
| `internal/cli/update_test.go` | Modify | `fakeUpdateAdapter.policy`; matrix re-keyed to policy |
| `internal/cli/integration_test.go`, `audit_probe_test.go` | Modify | Every `fakeUpdateAdapter` site sets policy explicitly |

## Interfaces / Contracts

```go
// internal/adapters/interface.go — zero value MUST stay PolicyGated (fail-closed,
// mirrors TrustLevel convention at interface.go:10-12).
const (
    PolicyGated        UpdatePolicy = 0 // update() only when check() reported an update
    PolicyAlwaysUpdate UpdatePolicy = 1 // update() always runs when requested
)
```
```go
// internal/adapters/official/helper.go — delegate to the seam vars (fakeable);
// failure returns "<tool> check failed (exit N): <stderr>: %w" (%w preserves
// errors.Is(err, context.DeadlineExceeded) for the CLI timeoutErr mapping).
func commandOutputErr(name string, args ...string) (string, error)
func shellOutputErr(command string) (string, error)
```

## Testing Strategy

| Layer | What to Test | Approach |
|---|---|---|
| Unit (helpers) | Variant success/failure/exit-code via seams; exit code extraction (real `sh -c exit N` child — precedent: TestRunCmd_*, adapter_update_test.go:268-292) | adapter_update_test.go additions; exec_mock_test.go unchanged |
| Unit (adapters) | Check() failure rows flip wantErr (apt/nvm/npm/pnpm command-fails rows); empty-output rows stay error-free; npm/pnpm exit-1-outdated = no error + availability; exit-124 = error; timeout via DeadlineExceeded fake stays errors.Is-detectable | check_test.go rows + new rows |
| Unit (goldens) | Per-adapter policy exact (reflect.DeepEqual); consistency test declared-value assertion | info_test.go |
| Integration (gating) | Matrix re-keyed to policy: Gated+true→updated; Gated+false→current; AlwaysUpdate+false→updated (stub/winget/custom); failed gated check→StatusFailed, never current | update_test.go, seam fakes |
| Regression | Hermetic suite (integration_test.go, audit_probe_test.go) with explicit fake policies; probe tests still deny/execute via confirm, not the gate | fake literals updated |
| E2E / Gates | `go test ./... -count=1 -race`, `go vet ./...`, gofmt, `bash scripts/smoke-test.sh --skip-build` | full gates |

## Threat Matrix

| Boundary | Applicability | Design response | Planned RED tests |
|---|---|---|---|
| Documentation-like paths | N/A — command strings unchanged static literals (only `|| true` removed); no classification or new launch sites | — | — |
| Git repository selection | N/A — no git ops | — | — |
| Commit state | N/A — no commit ops | — | — |
| Push state | N/A — no push ops | — | — |
| PR commands | N/A — no PR automation | — | — |

## Migration / Rollout

No migration (in-memory metadata). Documented behavior change: failed checks surface as StatusFailed instead of silent StatusCurrent; npm/pnpm detection with outdated packages keeps reporting updates (exit-1 convention preserved). Rollback: single `git revert` of the PR (~250 lines, under the 400-line budget).

## Open Questions

None.
