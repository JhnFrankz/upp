# Apply Progress: upp-security-zero-value-hardening

**Batch**: 1 (all 15 tasks, single PR) · **Mode**: Strict TDD · **Date**: 2026-08-14
**Delivery**: ask-on-risk → forecast Low → single PR, Decision needed: No — no branches, commits, or PRs created (orchestrator-owned delivery).

## Completed Tasks (cumulative, batch 1)

- [x] 1.1 `TestRiskLevel_ZeroValueIsMostRestrictive` added — asserts `RiskHigh == 0` (currently 2 → RED)
- [x] 1.2 `TestConfirmDecision_ZeroValueIsFailure` added — asserts `ConfirmError == 0` (currently 3 → RED)
- [x] 1.3 Zero-risk D4 cell added: `RiskLevel(0)` + untrusted CI → `ConfirmError` (zero is currently `RiskLow` → `ConfirmAuto` → RED)
- [x] 1.4 Unknown-risk D4 cells (`RiskLevel(99)` CI → `ConfirmError`; `RiskLevel(99)` interactive `n\n` → `ConfirmDeny`) verified pre-existing in the matrix (from trust precedent) and PASSING pre-reorder — default-branch pin confirmed, no duplication
- [x] 1.5 RED confirmed: 3 failures on current main (`ConfirmAction() = 2, want 3`; `RiskHigh = 2, want 0`; `ConfirmError = 3, want 0`); 1.4 cells pass
- [x] 2.1 `RiskLevel` const block in `internal/security/trust.go` replaced verbatim (design Interfaces/Contracts): `RiskHigh = 0, RiskMedium = 1, RiskLow = 2` + zero-value invariant comment
- [x] 3.1 `ConfirmDecision` const block in `internal/security/confirm.go` replaced verbatim: `ConfirmError = 0, ConfirmDeny = 1, ConfirmAuto = 2, ConfirmProceed = 3` + invariant comment
- [x] 3.2 `default: // RiskHigh` (confirm.go:81) kept; R4-4 rationale (lines 83-86) replaced verbatim — unknown risk is High by semantics, not zero-value coincidence
- [x] 3.3 `ClassifyCommand`'s terminal `return RiskLow` (trust.go:86) confirmed unchanged — legitimate classification
- [x] 3.4 GREEN confirmed: `go test ./internal/security/ -run 'ZeroValue|DecisionMatrix' -count=1` → `ok`
- [x] 4.1 `go test ./... -count=1` green (all 8 packages)
- [x] 4.2 `go test ./... -count=1 -race` green (all 8 packages)
- [x] 4.3 `go vet ./...` clean
- [x] 4.4 `gofmt -l internal/security/` empty (no `gofmt -s -w` needed)
- [x] 4.5 `bash scripts/smoke-test.sh --skip-build` green (23/23) — binary was missing at root, so built fresh with `go build -o upp ./cmd/upp` first (binary gitignored, tree unaffected)

## Files Changed

| File | Action | What Was Done |
|------|--------|---------------|
| `internal/security/security_expanded_test.go` | Modified | `TestRiskLevel_ZeroValueIsMostRestrictive` + `TestConfirmDecision_ZeroValueIsFailure` (invariants), 1 new zero-risk D4 cell (`RiskLevel(0)` + untrusted CI → `ConfirmError`) |
| `internal/security/trust.go` | Modified | `RiskLevel` reorder with explicit values + zero-value fail-closed invariant comment |
| `internal/security/confirm.go` | Modified | `ConfirmDecision` reorder with explicit values + invariant comment; R4-4 rationale rewrite on `default: // RiskHigh` |

## TDD Cycle Evidence

| Task | Test File | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|------|-----------|-------|------------|-----|-------|-------------|----------|
| 1.1 | `security_expanded_test.go` | Unit | ✅ security pkg green (baseline) | ✅ Written (got `RiskHigh = 2`, want 0) | ✅ Passed post-reorder | ➖ Skipped: structural invariant, single possible output | ➖ None needed |
| 1.2 | `security_expanded_test.go` | Unit | ✅ security pkg green | ✅ Written (got `ConfirmError = 3`, want 0) | ✅ Passed post-reorder | ➖ Skipped: structural invariant, single possible output | ➖ None needed |
| 1.3 | `security_expanded_test.go` | Unit | ✅ security pkg green | ✅ Written (got `ConfirmAction() = 2, want 3`) | ✅ Passed post-reorder | ✅ 2 paths: zero-risk CI (new) + unknown-risk CI/interactive (pre-existing pins) | ➖ None needed |
| 1.4 | `security_expanded_test.go` | Unit | ✅ security pkg green | ✅ Already passing pre-reorder — pins default branch | ✅ Passed | ✅ 2 cells: CI error + interactive deny | ➖ None needed |
| 2.1 | — (GREEN impl) | Unit | ✅ baseline | — | ✅ `go test ./internal/security/ -run 'ZeroValue|DecisionMatrix' -count=1` ok | — | ➖ None needed |
| 3.1 | — (GREEN impl) | Unit | ✅ baseline | — | ✅ same focused command ok | — | ➖ None needed |
| 3.2 | — (comment only) | — | ✅ baseline | — | ✅ same focused command ok | — | ➖ None needed |
| 3.3 | — (verification) | — | — | — | ✅ `return RiskLow` untouched (diff-verified) | — | ➖ None needed |

### Test Summary

- **Total tests written**: 3 (2 invariant tests + 1 zero-risk D4 cell)
- **Total tests passing**: 3 (+ all pre-existing, incl. 19-cell D4 matrix now 20 cells)
- **Layers used**: Unit (3)
- **Approval tests**: None — no behavior refactoring (enum reorder preserves all symbolic uses; full suite + smoke prove it)
- **Pure functions created**: 0

## Work Unit Evidence

| Evidence | Required value |
|---|---|
| Focused test command and exact result | `go test ./internal/security/ -run 'ZeroValue\|DecisionMatrix' -count=1` → RED: `FAIL` exit 1 with exactly 3 failures — `security_expanded_test.go:611: ConfirmAction() = 2, want 3`, `:631: RiskHigh = 2, want 0 (zero value MUST be most-restrictive)`, `:640: ConfirmError = 3, want 0 (zero value MUST be the failure outcome)`; the `RiskLevel(99)` cells pass. GREEN after reorder: `ok github.com/JhnFrankz/upp/internal/security 0.004s` exit 0 |
| Runtime harness command/scenario and exact result | `go build -o upp ./cmd/upp && bash scripts/smoke-test.sh --skip-build` (binary was absent; built fresh — gitignored, tree unaffected) → `Results: 23 passed, 0 failed, 23 total — All tests passed!` exit 0 |
| Rollback boundary | Revert exactly 3 files: `internal/security/trust.go` (enum), `internal/security/confirm.go` (enum + comment), `internal/security/security_expanded_test.go` (tests). No other files touched; no numeric persistence anywhere (`RiskLevel(`/`ConfirmDecision(` casts exist only in tests); openspec change folder is orchestrator-owned planning artifact, not implementation. |

## Deviations from Design

None — implementation matches design.md verbatim (explicit `= 0/1/2` and `= 0/1/2/3` values + invariant comments; `default: // RiskHigh` retained with new R4-4 rationale; D4 fallback cells + invariant tests).

Notes (within design latitude): task 1.4's `RiskLevel(99)` cells already existed in the matrix from the trust hardening precedent (lines 586-587) — verified passing pre-reorder instead of duplicating them; their pin role is recorded in evidence. Smoke gate required a fresh binary build (`go build -o upp ./cmd/upp`) because the root binary was absent; the gate command itself ran as spec'd with `--skip-build`.

## Issues Found

None.

## Remaining Tasks

None — all 15 tasks complete. Ready for sdd-verify.

## Workload / PR Boundary

- Mode: single PR
- Current work unit: Unit 1 — zero-value fail-closed hardening
- Boundary: batch 1 starts at RED tests (task 1.1) and ends at smoke (task 4.5); all 3 files changed, no branches/commits/PRs (orchestrator-owned delivery gates)
- Estimated review budget impact: ~90 changed lines (well under 400)
