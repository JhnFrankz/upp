# Proposal: upp security zero-value fail-closed hardening

## Intent

Mirror of the archived `upp-trust-zero-fail-closed` hardening (PRs #14/#15): two security enums still fail open at zero:
- `RiskLevel` (`internal/security/trust.go:10-17`): `RiskLow = iota` — unset classification reads as LOW.
- `ConfirmDecision` (`internal/security/confirm.go:16-25`): `ConfirmProceed = iota` — unset decision reads as PROCEED.

Latent, not live: all uses symbolic (no numeric zero uses, int conversions, or serialization); all 21 `ConfirmConfig{...}` literals set `RiskLevel` explicitly; `ClassifyCommand` (trust.go:65-86) and `ConfirmAction` (confirm.go:54-88,122-126) always return explicit values. A security enum's zero value MUST be its most restrictive member.

## Scope

### In Scope
- `trust.go:10-17`: reorder `RiskLevel` → `RiskHigh = 0, RiskMedium = 1, RiskLow = 2`; explicit values, contiguous 0-2 (keeps the `RiskLevel(3) -> "UNKNOWN"` pin), invariant comment (unclassified = dangerous).
- `confirm.go:16-25`: reorder `ConfirmDecision` → `ConfirmError = 0, ConfirmDeny = 1, ConfirmAuto = 2, ConfirmProceed = 3`; explicit values + invariant comment (unset decision = visible failure: tool marked failed, `--ci` exits non-zero).
- `confirm.go:81-91`: keep `default:`, rewrite its R4-4 rationale — after reorder the claim that an unhandled exit "would return the zero-value ConfirmProceed (fail-open)" becomes false. New rationale: unknown risk levels (e.g. `RiskLevel(99)`) are High by semantics, not zero-value coincidence.
- Tests in `internal/security/security_expanded_test.go` per the trust precedent: RED proof (assert fail-closed zeros on current main first), invariant tests (`RiskHigh == 0`, `ConfirmError == 0`), D4 fallback cells (zero-value/unknown risk → ConfirmError / High).

### Out of Scope
- `ConfirmDecision.String()`, i18n, adapter semantics
- linux/arm64 tarball, audit hygiene tracks
- `output/render.go` Status enum (unrelated iota)
- Spec-level changes (enum ordering is an implementation detail)

## Capabilities

> Contract with sdd-spec: spec-neutral hardening; security-model already requires trust MUST NOT bypass the risk matrix. No delta specs.

### New Capabilities
- None

### Modified Capabilities
- None

## Approach

1. RED proof: add zero-value assertions + fallback cells; `go test` fails on current main.
2. Reorder both enums with explicit values + invariant comments.
3. Rewrite the R4-4 rationale; `ClassifyCommand`'s terminal `return RiskLow` (trust.go:86) stays — legitimate classification.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/security/trust.go` | Modified | RiskLevel reorder + invariant comment |
| `internal/security/confirm.go` | Modified | ConfirmDecision reorder + invariant comment; R4-4 rationale rewrite |
| `internal/security/security_expanded_test.go` | Modified | RED proof + invariant tests + D4 fallback cells |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Reorder breaks numeric assumptions | Low | Symbolic uses only; suite + race + smoke |
| Rationale rewrite loses R4-4 intent | Med | Keep `default:`; rewrite explains semantic (not zero-value) High treatment |
| Future reorder reintroduces fail-open zero | Low | `RiskHigh == 0` / `ConfirmError == 0` assertions |

## Rollback Plan

`git revert` the single PR (only `trust.go`, `confirm.go`, `security_expanded_test.go`; no numeric persistence, configs, or state). Fail-open is latent today, so reverting is not a regression vs main.

## Dependencies

None. Self-contained; ~60-100 changed lines (2 enums + comments + tests) → single PR, low review load.

## Success Criteria

- [ ] `go test ./... -count=1` green incl. new invariant/fallback tests
- [ ] `-race` green; `go vet ./...` and `gofmt -l` clean
- [ ] `bash scripts/smoke-test.sh --skip-build` green
- [ ] `RiskHigh == 0` and `ConfirmError == 0` asserted; zero/unknown cases fail closed
