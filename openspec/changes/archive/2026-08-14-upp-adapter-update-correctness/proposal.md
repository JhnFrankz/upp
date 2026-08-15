# Proposal: adapter update-correctness hardening (timeouts, linux/arm64, official gating)

## Intent

Three correctness gaps in tool update execution:

- **No timeouts anywhere.** Every subprocess launch can hang the run indefinitely. All execution lives in exactly two production files — `shellExec` (internal/adapters/custom.go:121-137) and the official seam bodies `runCmdFn`/`runCmdArgsFn` (internal/adapters/official/helper.go:19-44); zero `context` usage repo-wide.
- **linux/arm64 broken.** The Go adapter's Linux branch hardcodes `.linux-amd64.tar.gz` (internal/adapters/official/go.go:57). `runtime.GOARCH` is compile-time — an amd64 CI host cannot exercise the arm64 branch unless URL construction becomes a parameterized helper.
- **Official adapters update unconditionally.** `runUpdate` calls `a.Update(false)` without consulting `Check()` (internal/cli/update.go:154-155). Six official adapters (brew, bun, docker, gh, go, opencode) always run their update command despite reporting `UpdateAvailable: false` (brew.go:34, bun.go:33, docker.go:32, gh.go:32, go.go:32, opencode.go:30). With timeouts added, a slow `brew` would also block every run.

## Scope

### In Scope
- **Timeouts**: `exec.CommandContext` + `context.WithTimeout` in `shellExec` AND both seam default bodies. CRITICAL: timeouts MUST live in the seam bodies, not the wrappers — `commandOutput`/`shellOutput` (helper.go:133-148) call the seam variables directly. Shared context; per-op timeouts (check ~30s, update ~10min; brew/nvm/apt slowest — exact values in design).
- **linux/arm64**: extract go.go:54-70 tarball URL construction into an arch-parameterized testable helper.
- **Official gating**: `runUpdate` calls `Update()` only when `Check()` returned `UpdateAvailable=true`; the six stubs map to `StatusCurrent`. **Custom adapters EXEMPT** — `UpdateAvailable` always false (custom.go:57-61), confirm gate already protects them. winget.go:30/scoop.go:30 hardcode `UpdateAvailable: true` — stay always-update by design (document). apt/npm/pnpm/nvm keep dynamic detection (apt.go:43, npm.go:37, pnpm.go:39, nvm.go:62).
- **runUpdate deps seam**: small refactor mirroring `checkDeps` (check.go:34-40) / `selfUpdateDeps` (selfupdate.go:49-56) so gating is testable (probe-style harness fallback — design picks).
- **Tests**: timeouts via existing seam fakes (keep seam signatures; skippable real-command integration alternative), gating via mocked adapters through the new seam, go-arch URL table tests.

### Out of Scope
- Custom-id collision warning; duplicated version helpers; dead `Update(dryRun)` branches; i18n; hermetic CLI tests.
- GNU `timeout 15` wrappers in npm.go:31/pnpm.go:33 (left as-is); winget/scoop semantics change.

## Capabilities

### New Capabilities
- None

### Modified Capabilities
- `tool-adapter`: `update()` for official adapters MUST run only when `check()` reported `update_available=true` (custom, winget, scoop exempt); `update()`/`check()` subprocesses MUST be timeout-bounded. Delta spec: `tool-adapter`.

## Approach

1. `context.WithTimeout` at the three execution sites (custom.go seam; helper.go seam bodies — wrappers untouched, zero churn).
2. Refactor go.go Linux URL into `goTarballURL(goarch string)`; call with `runtime.GOARCH`.
3. Add `runUpdateDeps` seam (mirror `checkDeps`); gate official adapters on `updateInfo.UpdateAvailable`, custom exempt.
4. Table-driven tests: fake `context.DeadlineExceeded`, gating matrix, URL-per-arch.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| internal/adapters/custom.go:121-137 | Modified | `shellExec` timeout |
| internal/adapters/official/helper.go:19-44 | Modified | seam body timeouts (wrappers untouched) |
| internal/adapters/official/go.go:54-70 | Modified | arch-parameterized tarball URL |
| internal/cli/update.go:34-198 | Modified | gating + `runUpdateDeps` seam |
| internal/cli/audit_probe_test.go:77-89 | Unchanged | preserved by custom exemption |
| openspec/specs/tool-adapter | Modified | delta spec (gating + timeouts) |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Six stubs stop updating (regression if `Check()` lies) | Med | False is by design; gating tests + manual smoke; real-detection paths unaffected |
| Timeout value wrong → false failures | Med | Per-op values; brew/nvm/apt slowest; confirm in design |
| arm64 still unexercised on CI | Med | Parameterized helper → amd64 CI covers logic; arch table tests |
| Gating breaks trusted-custom probe test | Low | Custom exemption preserves `TestProbe_TrustedLowRisk_Executes` |

## Rollback Plan

Single PR (est. ~150-250 changed lines, medium forecast, review budget 400). `git revert` of the merge commit restores unconditional updates, no timeouts, amd64-only URL. No data/config migration.

## Dependencies

- None. stdlib `context`/`os/exec` only.

## Success Criteria

- [ ] `go test ./... -count=1 -race` and `bash scripts/smoke-test.sh --skip-build` green.
- [ ] Every subprocess site timeout-bounded; seam signatures unchanged.
- [ ] Go Linux URL table-tested for amd64/arm64 via parameterized helper.
- [ ] Six stubs report `StatusCurrent` when no update; winget/scoop and custom still update.
- [ ] Tool-adapter delta spec documents gating + timeouts.
