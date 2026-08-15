# Proposal: Hermetic CLI Tests

## Intent

`go test ./internal/cli/` takes 34.3s (2026-08-14); ~33s comes from 13 tests spawning real adapter subprocesses. Make `internal/cli` tests hermetic: zero real subprocesses, injected fakes, <2s, deterministic.

## Scope

### In Scope
- Convert 8 real-adapter-loop tests (integration_test.go:310, 329, 391, 433, 492, 606, 763, 906) and 5 probe tests (audit_probe_test.go:28-109) to injected fakes.
- Add adapter-list injection to `checkDeps` (check.go:38-40); check.go:52 hardwires `buildAdapterList`. Mirror updateDeps (update.go:39-41, 65-67).
- Add a deps struct to `runList` (list.go:26-37) — currently none.
- Package-level injection var for Cobra entry points (`NewUpdateCommand`/`NewCheckCommand`/`NewSelfUpdateCommand`); precedent `official/helper.go` `runCmdFn` + `setExecFakes`.
- Extend `fakeUpdateAdapter` (update_test.go:19-48) with Command/Privileges (security-risk classification, update.go:146-159).
- Consolidate `mockAdapter` (integration_test.go:20-51) into the shared fake.
- check_hint_test.go:105 to use existing `writeCheckConfig` helper (check_hint_test.go:30-52).

### Out of Scope
- Dry-run/command transparency; any behavior change.
- Real-adapter E2E — stays in scripts/smoke-test.sh + internal/adapters tests.
- Full-repo hermeticity (`go test ./...` keeps real-subprocess tests in internal/adapters).
- runInit (init.go:27-53): no shell subprocess — unchanged.

## Capabilities

None of the 9 specs under openspec/specs/ cover test hermeticity.

### New Capabilities
None.

### Modified Capabilities
None — spec-neutral (test-only refactor). Future test-strategy note under ci-workflow possible; out of scope.

## Approach

1. Seams: adapter-list injection into checkDeps/runList; package-level injection for Cobra entry points. Nil-defaults → zero production behavior change.
2. Fakes: extend fakeUpdateAdapter (Command/Privileges), consolidate mockAdapter. Fakes MUST use real IDs (apt/npm/nvm/pnpm) — gatedOfficialAdapters keys on info.ID.
3. Convert 13 tests to fake-based assertions; probe tests assert an `updated` flag on fakeUpdateAdapter.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| internal/cli/check.go | Modified | adapterList injection via deps |
| internal/cli/list.go | Modified | deps struct + injection |
| internal/cli/update.go, init.go | Unmodified | seam exists / no subprocess |
| internal/cli/integration_test.go | Modified | 8 tests → fakes |
| internal/cli/audit_probe_test.go | Modified | 5 probes → fakeUpdateAdapter |
| internal/cli/check_hint_test.go | Modified | reuse writeCheckConfig |
| internal/cli/update_test.go | Modified | extend/consolidate fake |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| gatedOfficialAdapters ID mismatch | Med | fakes use real adapter IDs |
| Probe conversion loses exec proof | Low | covered by adapters package + smoke |
| Fake wiring drifts from prod behavior | Low | nil-default seams; full gates green |

## Rollback Plan

`git revert` the single PR. Tests + seams only; no production behavior change.

## Dependencies

None.

## Success Criteria

- [ ] `go test ./internal/cli/` <2s, deterministic across machines
- [ ] Zero real adapter subprocesses in cli tests
- [ ] Full gates green: `go test ./... -count=1`, vet, lint, smoke-test.sh
- [ ] ~200-300 changed lines, mostly tests; fits 400-line PR budget
