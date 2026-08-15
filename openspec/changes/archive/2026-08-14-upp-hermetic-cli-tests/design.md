# Design: Hermetic CLI Tests

## Technical Approach

`go test ./internal/cli/` takes 35.9s (verified 2026-08-14); ~33s comes from 13 tests spawning real adapter subprocesses. Strategy, mapping to proposal/spec invariants:

1. **Seams**: `checkDeps` (check.go:38-40) gains a `buildAdapterList` field and `runList` (list.go:26-37) gains a new `listDeps` struct; both nil-default to the production builder exactly like `updateDeps` (update.go:39-41, 65-67). check.go:52 and list.go:36 become seam-driven.
2. **Cobra entry points**: package-level `cliDeps` var feeds the four `New*Command` RunE bodies (precedent: official/helper.go `runCmdFn` + `setExecFakes`, exec_mock_test.go:24-47). Zero value → production.
3. **Fakes**: `fakeUpdateAdapter` (update_test.go:19-48) gains Command/Privileges so `Info()` drives the security-classification path (update.go:146-159); `mockAdapter` (integration_test.go:20-51) and `probeSetup` (probe_test.go:25-53) are retired.
4. **Conversions**: 13 tests (8 cobra-entry, 5 probes) move to fakes; probe exec-proof becomes an `updated`-flag assertion; check_hint's DefaultOff test reuses `writeCheckConfig`.

## Architecture Decisions

### Decision: Seam shape — nil-default vs required-field

**Choice**: `checkDeps.buildAdapterList` + new `listDeps{buildAdapterList}`; nil-default fallback inside `runCheck`/`runList` (update.go:65-67 shape).
**Alternatives**: required-field (entry points construct the production default explicitly); constructor injection (`NewCheckCommand(gf, deps)`) — churns parser.go:61-64 and parser_test.go:264,274.
**Rationale**: the zero value MUST default to production `buildAdapterList` — cobra entry points are constructed with no deps in production (parser.go:61-64), so any non-production default silently ships an empty/wrong adapter list. Nil-default is the established convention (checkDeps.clientFactory check.go:129-133, selfUpdateDeps selfupdate.go:69-79) and preserves the byte-for-byte behavior the spec-neutral verdict relies on.

### Decision: Package-level injection var

**Choice**: `var cliDeps struct{ check checkDeps; update updateDeps; list listDeps; selfUpdate selfUpdateDeps }` in new `internal/cli/deps.go`; each RunE passes `cliDeps.<cmd>`; tests swap via a `setCLIDeps(t, …)` helper (homes: withCapturedStdout, integration_test.go:57-70) with `t.Cleanup` restore, mirroring `setExecFakes`.
**Alternatives**: per-command vars (more declarations, more restore points); deps as constructor params (signature churn).
**Rationale**: one declaration, one swap point. Race-safety: no `t.Parallel()` exists in internal/cli (grep-verified), so sequential mutation is safe; the var's doc comment warns that adding t.Parallel requires synchronization. Spec lists check/update/self-update; `list` is added because TestListCommand_NoConfig (integration_test.go:310) executes via `root.Execute()` and needs the seam.

### Decision: Fake design — single fakeUpdateAdapter

**Choice**: `fakeUpdateAdapter` gains `command string` and `privileges []string`; `Info()` returns `ToolInfo{ID: name, Name: name, Trust: trust, Command: command, Privileges: privileges}`. `mockAdapter` deleted; its users (TestAdapterIDs :458, TestAdapterByID :474) switch to the fake. `name` stays the ID.
**Alternatives**: keep both fakes (two contracts to keep in sync with adapters.Adapter).
**Rationale**: mockAdapter is a strict subset (no trust control, no updated recording). Gating keys on `info.ID` (gatedOfficialAdapters update.go:48, 183) — gating fakes MUST use real IDs apt/npm/nvm/pnpm, which the matrix (update_test.go:83-130) already does; name-as-ID preserves that invariant.

### Decision: Probe test conversion

**Choice**: retire `probeSetup` and marker-file proof; the 5 probes (audit_probe_test.go:28-109) call `runUpdate` with an injected fake, asserting `fake.updated` plus preserved outcomes: error in CI, no error on deny, "Proceed? [y/N]" under --quiet.
**Alternatives**: keep real-subprocess probes — rejected: that IS the ~33s cost; real exec proof stays in internal/adapters tests + scripts/smoke-test.sh.
**Rationale**: the security branches are reachable via fake Command/Privileges with identical strings — `sudo rm -rf …` → RiskHigh (HighRiskKeywords trust.go:36-47), `… && …` → RiskMedium (hasCommandChaining trust.go:92-96), privileges flow into ConfirmConfig (update.go:152-159). ConfirmAction is string-driven (confirm.go:53), so ConfirmError (CI+High), promptUser→ConfirmDeny (interactive High / Medium-untrusted; os.Stdin is /dev/null under `go test`, same deny as today), and ConfirmProceed (trusted Low) fire unchanged.

### Decision: check_hint DefaultOff test

**Choice**: TestCheckHint_DefaultOff_ZeroNetwork (check_hint_test.go:105) replaces bare `t.Setenv("HOME", t.TempDir())` — defaults enable ALL catalog tools → real Check() subprocesses — with `writeCheckConfig(t, "")`: empty [settings] keeps check_self_update off; all catalog tools written disabled → empty hermetic loop; the zero-client-constructions assertion is unchanged.

### Decision: What stays real

**Choice**: `runInit` (init.go:27-53) and its tests (TestInitCommand_CI_Mode/DetectsTools/AlreadyExists integration_test.go:203-239, 932-951) untouched — official `Detect()` is LookPath-only (apt.go:15-17), no subprocess. TestEmptyConfig_AllToolsSkipped (:629) stays — all tools disabled → already-empty loop. TestBuildAdapterList_* (:348, :367, :737) stay — construction only. internal/adapters + internal/adapters/official untouched — real-subprocess coverage lives there by design.

### Decision: Scope boundaries

**Choice**: no dry-run/command-transparency changes; no behavior changes (nil-default seams); no full-repo hermeticity (`go test ./...` keeps internal/adapters real-subprocess tests); no changes to init.go, update.go logic, security, or adapters packages.

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/cli/deps.go` | Create | `cliDeps` holder var + doc comment incl. race warning |
| `internal/cli/check.go` | Modify | `checkDeps` + buildAdapterList; nil-default before :52; RunE → `cliDeps.check` |
| `internal/cli/list.go` | Modify | `listDeps`; `runList(gf, deps)`; nil-default; RunE → `cliDeps.list` |
| `internal/cli/update.go` | Modify | RunE → `cliDeps.update` (updateDeps unchanged) |
| `internal/cli/selfupdate.go` | Modify | RunE → `cliDeps.selfUpdate` |
| `internal/cli/update_test.go` | Modify | fake +command/+privileges, Info() returns them; runUpdateWith generalized for CI/quiet flags |
| `internal/cli/integration_test.go` | Modify | delete mockAdapter; add setCLIDeps; 8 tests → cliDeps swap + fakes; TestAdapterIDs/TestAdapterByID → fakeUpdateAdapter |
| `internal/cli/audit_probe_test.go` | Modify | 5 probes → fake-based, updated-flag assertions |
| `internal/cli/probe_test.go` | Modify | delete probeSetup; keep probeHome (used by update_test.go:54,165; init_probe_test.go:50,68,100) |
| `internal/cli/check_hint_test.go` | Modify | DefaultOff test → `writeCheckConfig(t, "")` |

## Interfaces / Contracts

```go
// internal/cli/deps.go — package-level injection for cobra entry points.
// Zero value = production. Sequential-only (no t.Parallel): adding parallel
// tests requires synchronization.
var cliDeps struct {
    check      checkDeps
    update     updateDeps
    list       listDeps
    selfUpdate selfUpdateDeps
}

// internal/cli/check.go / list.go — mirrors updateDeps (update.go:39-41, 65-67)
type listDeps struct {
    buildAdapterList func(cfg *config.Config, osName string) []adapters.Adapter
}
```

## Testing Strategy

| Layer | What | How |
|-------|------|-----|
| Unit | Gating matrix, timeout mapping (existing) | unchanged; fake gains fields |
| Integration | 8 cobra-entry tests (310, 329, 391, 433, 492, 606, 763, 906) | `setCLIDeps` swap + fakes; same output assertions; TestDryRun additionally asserts `updated == false` |
| Security | 5 probes (28-109) | fake Command/Privileges drive ClassifyCommand/ConfirmAction; `updated`-flag proof |
| E2E | `scripts/smoke-test.sh --skip-build` | unchanged real-adapter proof |
| Gates | `time go test ./internal/cli/ -count=1` < 2s (baseline 35.9s); `go test ./... -count=1`; `go test -race ./...`; `go vet ./...`; `gofmt -l .` empty | full suite |

## Threat Matrix

| Boundary | Applicability | Design response | Planned RED tests |
|---|---|---|---|
| Documentation-like paths | N/A — no doc-path classification | — | — |
| Git repository selection | N/A — no git ops | — | — |
| Commit state | N/A — no commit ops | — | — |
| Push state | N/A — no push ops | — | — |
| PR commands | N/A — no PR automation | — | — |
| Executable-file classification | Applicable — probes currently prove sudo/&& classification via real execution; fakes must drive the SAME branches | Fake Command/Privileges carry identical strings (sudo→High, &&→Medium); ConfirmAction exercised unchanged (D4) | Converted probes: CI+High trusted → error + not executed; interactive High → deny + not executed; trusted Low → executed (updated); --quiet Medium untrusted → prompt shown + deny |

## Migration / Rollout

No migration. Single PR (predicted ~200-300 lines, within the 400-line review budget); rollback is a single `git revert` — tests and nil-default seams only, no production behavior change.

## Open Questions

None.
