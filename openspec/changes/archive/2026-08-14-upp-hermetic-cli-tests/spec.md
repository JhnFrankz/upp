# Spec Phase — upp-hermetic-cli-tests

## Verdict: Spec-Neutral Confirmed — No Delta Specs

Verified against all 9 specs under `openspec/specs/` (ci-workflow, command-interface, config-system, platform-detection, release-process, security-model, self-update, tool-adapter, ux-patterns). No requirement text needs to change, none is contradicted, and none is newly satisfied. No `specs/` directory is created under this change folder; this artifact records the verification verdict only. Mirrors the archived `upp-security-zero-value-hardening` spec-neutral precedent (`openspec/changes/archive/2026-08-14-upp-security-zero-value-hardening/spec.md`).

## Evidence

1. **No spec governs CLI test execution.** A sweep of `openspec/specs/` for `hermetic`, `subprocess`, `testing.Short`, `go test`, `_test.go`, `fake`, and `suite` finds zero requirement text on how tests execute. The only `subprocess` mentions are production adapter behavior: `tool-adapter/spec.md:88` (every subprocess launched by `check()`/`update()` MUST be timeout-bounded) and `tool-adapter/spec.md:94` (scenario for the check timeout). These bind production adapters, which this change does not touch.
2. **No requirement text changes.** All 9 specs describe production behavior. The change is test-only plus injection seams with nil-defaults (`update.go:65-68` precedent), which preserves production behavior byte-for-byte; requirements describe behavior, not test mechanics.
3. **No requirement contradicted.** The governing requirements remain satisfied unchanged:
   - `tool-adapter/spec.md:88` — subprocess timeout bounding still holds; CLI tests simply stop spawning subprocesses, real adapters are unchanged.
   - `config-system/spec.md:90,96` — "test-enforced" zero-network default remains enforced by the (now faster, fake-based) suite; requirement text unchanged.
   - `ci-workflow/spec.md:21,25` — `test` job 15-min `timeout-minutes` is a ceiling contract; sub-2s CLI tests stay within it. `ci-workflow/spec.md:40` — branch protection requires check `test` to pass; unchanged.
   - `release-process/spec.md:34,39,80` — CI test/lint gating unchanged.
4. **No requirement newly satisfied.** No spec requires hermetic, fast, or deterministic CLI tests, so nothing unmet becomes satisfied. Real-adapter E2E proof stays where specs already rely on it: `scripts/smoke-test.sh` and `internal/adapters` tests (e.g. `tool-adapter/spec.md:50-51` update-mechanism scenarios).

## Why No Delta Specs

Test hermeticity is an implementation-mechanics concern below the abstraction level of every requirement. Per the accepted proposal (Capabilities: New None / Modified None), no delta specs are produced; the archive phase will perform no spec sync. A future test-strategy note under ci-workflow remains possible but is out of scope.

## Implementation Invariants for design/apply

- `checkDeps` (`check.go:38-40`): add adapter-list seam; `check.go:52` hardwires `buildAdapterList`. Mirror `updateDeps` precedent (`update.go:39-41` field, `update.go:65-67` nil-default fallback).
- `runList` (`list.go:26-37`): add deps struct with adapter-list field; `list.go:36` currently calls `buildAdapterList` directly.
- Cobra entry points `NewUpdateCommand`/`NewCheckCommand`/`NewSelfUpdateCommand`: package-level injection var, nil-default → zero production behavior change. Precedent: `official/helper.go` `runCmdFn` + `setExecFakes`.
- Extend `fakeUpdateAdapter` (`update_test.go:19-48`) with Command/Privileges fields for security-risk classification (`update.go:146-159`).
- Consolidate `mockAdapter` (`integration_test.go:20-51`) into the shared fake.
- Fakes MUST use real adapter IDs (apt, npm, nvm, pnpm): `gatedOfficialAdapters` keys on `info.ID` (`update.go:48`, `update.go:183`).
- Convert 13 tests: 8 real-adapter-loop tests (`integration_test.go:310, 329, 391, 433, 492, 606, 763, 906`) + 5 probes (`audit_probe_test.go:28-109`); probes assert an `updated` flag on the fake.
- `check_hint_test.go:105`: reuse existing `writeCheckConfig` helper (`check_hint_test.go:30-52`).
- Goal: `go test ./internal/cli/` <2s, zero real adapter subprocesses, deterministic across machines. Full gates green: `go test ./... -count=1`, vet, lint, `scripts/smoke-test.sh`.
