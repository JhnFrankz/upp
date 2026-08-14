# Spec Phase — upp-security-zero-value-hardening

## Verdict: Spec-Neutral Confirmed — No Delta Specs

Verified against `openspec/specs/security-model/spec.md` and every other spec under `openspec/specs/`. No requirement text needs to change, none is contradicted, and none is newly satisfied. No `specs/` directory is created under this change folder; this artifact records the verification verdict only. Mirrors the archived `upp-trust-zero-fail-closed` precedent (Engram #84).

## Evidence

1. **No spec references the affected enums.** A sweep of `openspec/specs/` for `RiskLevel`, `ConfirmDecision`, their member names, and `zero-value`/`iota`/`fail-open`/`fail-closed` returns no match touching these enums. The only `fail-closed` mentions concern unrelated subsystems: self-update checksum integrity (`security-model/spec.md:71`), release asset checksums (`release-process/spec.md:56,62`), and unknown OS/arch asset mapping (`self-update/spec.md:34`).
2. **No requirement text changes.** Reordering `RiskLevel` (RiskHigh=0, RiskMedium=1, RiskLow=2) and `ConfirmDecision` (ConfirmError=0, ConfirmDeny=1, ConfirmAuto=2, ConfirmProceed=3) preserves the semantic mapping — same members, same `String()` labels, same behavior for every existing caller. All production uses are symbolic; the only numeric casts are test-side UNKNOWN-pin and D4 fallback cells (`security_expanded_test.go:203-205,586-587`). Requirements describe behavior, not enum layout.
3. **No requirement contradicted.** The governing security-model requirements remain satisfied unchanged:
   - `security-model/spec.md:17` — "Trust level MUST NOT bypass the risk matrix" (trust enum untouched; risk semantics unchanged).
   - `security-model/spec.md:35` — "`--ci` MUST fail high-risk custom updates with a non-zero exit, even when `trusted = true`" — remains satisfied; `ConfirmError == 0` makes fail-closed the zero-value path for future callers.
   - `security-model/spec.md:56` — "High-risk operations ALWAYS require confirmation regardless of trust level" — remains satisfied; `RiskHigh == 0` makes the most restrictive risk the zero-value.
4. **No requirement newly satisfied.** All of the above are already met on current main: the only production `ConfirmConfig{...}` literal (`internal/cli/update.go:128-135`) sets `RiskLevel` explicitly, and `ClassifyCommand` (`trust.go:65-86`) and `ConfirmAction` (`confirm.go:54-88,122-126`) always return explicit values. The fail-open trap is latent, not live; this change hardens an implementation invariant.

## Why No Delta Specs

Enum ordering is an implementation detail below the abstraction level of every requirement. Per the accepted proposal (Capabilities: New None / Modified None), no delta specs are produced; the archive phase will perform no spec sync.

## Implementation Invariants for design/apply

- `RiskLevel` (`trust.go:10-17`): explicit values RiskHigh=0, RiskMedium=1, RiskLow=2, contiguous 0-2 (keeps the `RiskLevel(3) -> "UNKNOWN"` pin); comment documents "unclassified = dangerous".
- `ConfirmDecision` (`confirm.go:16-25`): explicit values ConfirmError=0, ConfirmDeny=1, ConfirmAuto=2, ConfirmProceed=3; comment documents "unset decision = visible failure".
- `confirm.go:81-91` MUST keep `default: // RiskHigh`; rewrite the R4-4 rationale (`confirm.go:83-86`) — after reorder, the zero-value-fail-open claim becomes false; new rationale: unknown risk levels are High by semantics, not zero-value coincidence.
- `ClassifyCommand`'s terminal `return RiskLow` (`trust.go:86`) stays — legitimate classification.
- Tests MUST assert `RiskHigh == 0` and `ConfirmError == 0` and cover zero-value/unknown-risk cases resolving fail-closed (ConfirmError / High).
