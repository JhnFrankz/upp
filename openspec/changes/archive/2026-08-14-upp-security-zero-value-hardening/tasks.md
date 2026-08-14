# Tasks: upp security zero-value fail-closed hardening

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 60–100 (3 files) |
| 400-line budget risk | Low |
| Chained PRs recommended | No |
| Suggested split | Single PR |
| Delivery strategy | ask-on-risk |
| Chain strategy | pending |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: pending
400-line budget risk: Low

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|----------------------|-----------------|-------------------|
| 1 | Fail-closed zero values: RED tests → enum reorders + invariant comments → R4-4 rationale rewrite | PR 1 | `go test ./internal/security/ -run 'ZeroValue\|DecisionMatrix' -count=1` | `bash scripts/smoke-test.sh --skip-build` | `git revert` of change commit — only `internal/security/trust.go`, `internal/security/confirm.go`, `internal/security/security_expanded_test.go`; no persisted data |

## Phase 1: RED Proof Tests (write first — must fail on current main)

- [x] 1.1 Add `TestRiskLevel_ZeroValueIsMostRestrictive` to `internal/security/security_expanded_test.go` asserting `RiskHigh == 0` (currently 2 → RED)
- [x] 1.2 Add `TestConfirmDecision_ZeroValueIsFailure` asserting `ConfirmError == 0` (currently 3 → RED)
- [x] 1.3 Add zero-risk D4 cell to `TestConfirmAction_DecisionMatrix`: `RiskLevel(0)` + untrusted CI → `ConfirmError` (zero is currently `RiskLow` → Auto → RED)
- [x] 1.4 Add unknown-risk D4 cells: `RiskLevel(99)` CI → `ConfirmError`; `RiskLevel(99)` interactive `n\n` → `ConfirmDeny` (pass pre-reorder; default-branch pin)
- [x] 1.5 Run `go test ./internal/security/ -run 'ZeroValue|DecisionMatrix' -count=1`: 1.1–1.3 FAIL, 1.4 passes — RED confirmed on current main

## Phase 2: Foundation — RiskLevel reorder (GREEN 1)

- [x] 2.1 Replace the `RiskLevel` const block in `internal/security/trust.go:10-17` verbatim (design Interfaces/Contracts):

```go
const (
	// RiskHigh is the ZERO value on purpose: an unset RiskLevel MUST resolve to
	// the most restrictive member so unclassified risk fails closed. The zero
	// value MUST stay the most restrictive tier — never insert a tier before it.
	RiskHigh RiskLevel = 0
	// RiskMedium may modify system state.
	RiskMedium RiskLevel = 1
	// RiskLow is non-destructive, no privileges required.
	RiskLow RiskLevel = 2
)
```

## Phase 3: Core — ConfirmDecision reorder + R4-4 rationale (GREEN 2)

- [x] 3.1 Replace the `ConfirmDecision` const block in `internal/security/confirm.go:16-25` verbatim (design Interfaces/Contracts):

```go
const (
	// ConfirmError is the ZERO value on purpose: an unset decision MUST fail
	// visibly — tool marked failed, --ci exits non-zero — never auto-proceed.
	// The zero value MUST stay the failure outcome — never insert before it.
	ConfirmError ConfirmDecision = 0
	// ConfirmDeny means the user denied the action.
	ConfirmDeny ConfirmDecision = 1
	// ConfirmAuto means --ci mode auto-proceeded (no prompt shown).
	ConfirmAuto ConfirmDecision = 2
	// ConfirmProceed means the user approved the action.
	ConfirmProceed ConfirmDecision = 3
)
```

- [x] 3.2 In `internal/security/confirm.go`, keep `default: // RiskHigh` (line 81) and replace the R4-4 rationale comment (lines 83-86) verbatim (design Interfaces/Contracts):

```go
		// Deliberately a default, NOT `case RiskHigh:`: an unknown future
		// RiskLevel (e.g. RiskLevel(99)) MUST resolve to High by semantics —
		// interactive prompt, CI error — not by zero-value coincidence.
		// Relying on the zero value would be fragile against future enum edits.
```

- [x] 3.3 Confirm `ClassifyCommand`'s terminal `return RiskLow` (`trust.go:86`) unchanged — legitimate classification
- [x] 3.4 Run `go test ./internal/security/ -run 'ZeroValue|DecisionMatrix' -count=1`: all green — GREEN confirmed

## Phase 4: Verification Gates

- [x] 4.1 `go test ./... -count=1` green
- [x] 4.2 `go test ./... -count=1 -race` green
- [x] 4.3 `go vet ./...` clean
- [x] 4.4 `gofmt -l internal/security/` empty (run `gofmt -s -w` on any diffs)
- [x] 4.5 `bash scripts/smoke-test.sh --skip-build` green
