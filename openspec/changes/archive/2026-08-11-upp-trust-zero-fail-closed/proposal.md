# Proposal: upp trust zero-value fail-closed hardening

## Intent

`TrustLevel`'s zero value is `TrustOfficial` (iota 0) in `internal/adapters/interface.go`, and `internal/security/confirm.go:53` auto-proceeds on it — an UNSET TrustLevel silently bypasses the risk matrix (fail-open at High risk, even in CI). Pre-change behavior was fail-closed (empty string trust != 'official'). All adapters set the field explicitly today (latent), but a security trust enum's zero value MUST be the least-privileged level. Follow-up findings R4-1 (CRITICAL), R4-3, R4-4 from review lineage review-2b2d23d991665831.

## Scope

### In Scope
- `internal/adapters/interface.go`: reorder enum so zero value = `TrustCustomUntrusted`; explicit constant values + comment documenting the zero-value fail-closed invariant (R4-1).
- Fail-closed fallback tests (R4-3): zero-value TrustLevel → risk matrix applies (untrusted semantics); unknown `TrustLevel(99)` → untrusted path; unknown `RiskLevel` → default/High branch.
- `internal/security/confirm.go`: keep `default: // RiskHigh` branch, document why it must not become an explicit `case RiskHigh:` (R4-4).

### Out of Scope
- Docs hygiene ((Previously:) annotations, verify-report |zsh claim, proposal /tmp path)
- Compact-pipe false positives, chain-before-pipe ordering
- `.atl` drift
- Any new or changed spec requirements (pure hardening of existing requirements)

## Capabilities

> Contract with sdd-spec: spec-neutral hardening. Enum ordering is an implementation detail; the security-model spec already requires "trust level MUST NOT bypass the risk matrix". No delta specs needed.

### New Capabilities
- None

### Modified Capabilities
- None

## Approach

- `internal/adapters/interface.go`: `TrustCustomUntrusted TrustLevel = iota` (zero), then `TrustCustomTrusted`, `TrustOfficial`; comment: zero value MUST stay least-privileged (fail-closed invariant). All uses are symbolic (== comparisons, `String()` switch, tests) — no numeric persistence exists.
- `internal/security/confirm.go`: behavior unchanged; retain `default: // RiskHigh` — replacing it with `case RiskHigh:` would let future risk values exit the switch as zero-value `ConfirmProceed` (fail-open).
- Tests in `internal/security/security_expanded_test.go`: extend the D4 matrix table with zero-value trust, `TrustLevel(99)`, and `RiskLevel(99)` cases; assert `TrustCustomUntrusted == 0`.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/adapters/interface.go` | Modified | Enum reorder + zero-value invariant comment (R4-1) |
| `internal/security/confirm.go` | Modified | Comment only on `default:` branch (R4-4) |
| `internal/security/security_expanded_test.go` | Modified | Fallback tests: zero/unknown trust + unknown risk (R4-3) |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Enum reorder breaks numeric assumptions | Low | No numeric persistence; all uses symbolic; full suite + race + smoke |
| Future edit makes `default` explicit `case RiskHigh:` → fail-open | Med | Documented invariant comment + fallback tests fail open on new risk values |
| Future reorder reintroduces fail-open zero value | Low | Explicit `TrustCustomUntrusted == 0` assertion in test |

## Rollback Plan

Revert the change commit (`git revert`) — affects only `internal/adapters/interface.go`, `internal/security/confirm.go`, `internal/security/security_expanded_test.go`. Pre-change behavior returns; no regression vs current main (fail-open risk is latent today).

## Dependencies

- None (self-contained hardening, no new dependencies)

## Success Criteria

- [ ] `go test ./... -count=1` green, incl. new zero-value/fallback tests
- [ ] `go test ./... -count=1 -race` green
- [ ] `go vet ./...` and `gofmt -l` clean
- [ ] `bash scripts/smoke-test.sh --skip-build` green
- [ ] `TrustCustomUntrusted == 0` asserted; `TrustLevel(99)` and `RiskLevel(99)` hit fail-closed branches
