# Proposal: Per-Adapter Update Policy and Check-Failure Signal

## Intent

Two silent-degradation bugs. (1) Update gating lives in a CLI-side ID list (`gatedOfficialAdapters`, update.go:48-53) instead of per-adapter metadata. (2) `check()` swallows subprocess failures: `shellOutput`/`commandOutput` return `""` on error (helper.go:173-188), mapped to "unknown" → `update_available=false` → StatusCurrent (check.go:111-116) — a failed check reads as "current".

## Scope

### In Scope

- `ToolInfo.UpdatePolicy`: `PolicyGated=0` (zero, fail-closed convention) / `PolicyAlwaysUpdate=1`; explicit at all 13 Info() sites (apt/npm/nvm/pnpm → Gated; rest → AlwaysUpdate).
- Gate: update.go:183 → `info.UpdatePolicy == adapters.PolicyGated && !updateInfo.UpdateAvailable`; delete `gatedOfficialAdapters` (:48-53, single consumer:183).
- `shellOutputErr`/`commandOutputErr` on same seams; apt/nvm/npm/pnpm error ONLY on real subprocess failure (exit ≠ 0) — empty output stays "unknown"; npm/pnpm drop `|| true` (npm.go:31, pnpm.go:33).
- Explicitness: `TestAllAdapters_InfoConsistency` asserts non-zero policy; goldens info_test.go:22-33, `fakeUpdateAdapter` (update_test.go:46-54) + ~10 hermetic fake sites set policy explicitly.
- Spec delta `tool-adapter`: MODIFIED Update Gating (ID-list → per-adapter policy; scenario: gated check FAILS → update reports failure, never "current") + ADDED Check Failure Signal (check() MUST error on detection-subprocess failure; CLI MUST surface StatusFailed, never StatusCurrent on failed check).

### Out of Scope

- Failure-aware `upp check` exit code (flag only); list.go:66-69 swallow (minor).
- Renderer changes — CLI reuses StatusFailed.
- winget/scoop stay pure stubs (AlwaysUpdate; discarded output; failures unreported — by design).
- npm/pnpm `timeout 15` bound decision → design phase.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `tool-adapter`: Update Gating requirement rewritten from ID-list wording to per-adapter `UpdatePolicy`; new Check Failure Signal requirement added.

## Approach

Contract already returns error (interface.go:43) — no UpdateInfo change. Add enum (interface.go:67-74), set at 13 sites, replace gate, delete ID list, add error-aware helper variants, propagate errors in apt/nvm/npm/pnpm Check(), drop `|| true`. CLI already maps err → StatusFailed (check.go:95-103, update.go:114-123) — no CLI change; fix is adapters returning real errors. Update goldens/fakes.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| interface.go:67-74 | Modified | UpdatePolicy enum |
| 12 official Info() + custom.go:94-107 | Modified | explicit policy, 13 sites |
| update.go:48-53, :183 | Removed/Modified | gate; ID list deleted |
| helper.go:173-188 | Modified | error-aware variants |
| apt.go, nvm.go, npm.go, pnpm.go Check() | Modified | structured errors; drop `\|\| true` |
| check.go, update.go (CLI) | None | StatusFailed already handled |
| info_test.go, adapter_update_test.go, update_test.go + fakes | Modified | goldens, explicitness |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| npm/pnpm timeout layering (10m helper bound vs shell `timeout 15`) | Med | Bound → design; keep 15s shell bound |
| apt "(none)" vs failure ambiguity | Low | Only exit ≠ 0 signals failure; "(none)" stays unknown |
| Missed site defaults to Gated (zero) | Low | 13 explicit sites + non-zero assertion |
| Hermetic fakes defaulting to Gated flip gating matrix | Med | Every fake sets policy explicitly |

## Rollback Plan

`git revert` of the PR. Behavior change: failed checks surface as StatusFailed instead of silent StatusCurrent; revert restores prior behavior. No migration (in-memory metadata).

## Dependencies

None.

## Success Criteria

- [ ] `go test ./... -count=1` green; vet/lint clean.
- [ ] `gatedOfficialAdapters` deleted; gate reads UpdatePolicy only.
- [ ] All 13 Info() sites explicit; consistency test asserts non-zero policy.
- [ ] Seam-faked: apt/nvm/npm/pnpm Check() errors on subprocess failure, never on empty output; npm/pnpm maskless.
- [ ] Failed gated check → failure, never "current"; `upp check` shows StatusFailed.
