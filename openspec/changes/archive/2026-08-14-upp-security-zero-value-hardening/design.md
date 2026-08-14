# Design: upp security zero-value fail-closed hardening

## Technical Approach

Mirror of the archived `upp-trust-zero-fail-closed` hardening (PRs #14/#15): reorder the two fail-open security enums so each zero value is its most restrictive member, pin them with explicit contiguous constants + invariant comments, and prove fail-closed resolution with tests. Behavior for every symbolic use (production callers, `String()`, table-driven tests) is identical — only the zero-value semantic changes. Sequence: (1) RED-proof tests on current main, (2) enum reorder + comments, (3) tests go green. Spec-neutral verdict — no spec deltas.

## Architecture Decisions

### D1: Zero-value member choice

| Option | Tradeoff | Decision |
|--------|----------|----------|
| RiskLow = 0 (status quo) | Unset risk reads LOW = silent auto-proceed (fail-open) | Rejected |
| **RiskHigh = 0** | Unset = most restrictive; mirrors trust precedent; `RiskLevel(3) -> "UNKNOWN"` pin stays valid | **Chosen** |
| ConfirmProceed = 0 (status quo) | Unset decision reads PROCEED = fail-open | Rejected |
| **ConfirmError = 0** | Unset = visible failure (tool marked failed; `--ci` exits non-zero) | **Chosen** |

### D2: Explicit contiguous values vs bare iota

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Reorder + bare iota | Least diff, but invariant invisible; future insertions silently renumber | Rejected |
| **Reorder + explicit `= 0/1/2` (`= 0/1/2/3`)** | 3–4 noisier lines; self-documenting, reorder-proof; contiguity keeps out-of-range pins stable | **Chosen** |

### D3: Keep `default: // RiskHigh` with rewritten rationale

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Replace with `case RiskHigh:` | Unknown `RiskLevel(99)` exits the switch; fail-closed only by zero-value coincidence (fragile against enum edits) | Rejected |
| **Keep `default:` + new rationale** | Unknown risk resolves to High by semantics (interactive prompt / CI error), independent of the zero value | **Chosen** |

### D4: Test strategy

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Invariant tests only | Pins constants but not resolution paths | Rejected |
| **RED proof + invariant + D4 fallback cells** | RED proves fail-open on main; invariants guard future reorders; D4 cells pin zero/unknown resolution (CI + interactive) | **Chosen** |

### D5: Scope boundaries

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Add `ConfirmDecision.String()` | Nice diagnostics; out of scope | Rejected |
| Touch `output/render.go` Status enum | Unrelated iota; scope creep | Rejected |
| **Only two enums + comments + tests** | Focused diff (~60–100 lines), single PR, low review load | **Chosen** |

## Data Flow

```
ConfirmConfig{RiskLevel} ──► ConfirmAction (confirm.go:51)
  RiskLow ──► CI: Auto | interactive: info
  RiskMedium ──► CI: Auto(trusted)/Error | interactive: prompt/info
  default: // RiskHigh (RiskHigh=0 and RiskLevel(99)) ──► CI: ConfirmError | interactive: prompt
```

Zero-value `RiskLevel(0)` now equals `RiskHigh` (was `RiskLow`); zero-value `ConfirmDecision(0)` now equals `ConfirmError` (was `ConfirmProceed`). All symbolic callers observe no change.

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/security/trust.go` | Modify | Reorder `RiskLevel` → `RiskHigh = 0, RiskMedium = 1, RiskLow = 2`; explicit values + zero-value invariant comment |
| `internal/security/confirm.go` | Modify | Reorder `ConfirmDecision` → `ConfirmError = 0, ConfirmDeny = 1, ConfirmAuto = 2, ConfirmProceed = 3`; explicit values + invariant comment; rewrite R4-4 rationale (`confirm.go:83-86`) |
| `internal/security/security_expanded_test.go` | Modify | RED-proof invariants (`RiskHigh == 0`, `ConfirmError == 0`), `RiskLevel(0)` zero-risk D4 cell, `RiskLevel(99)` unknown-risk cells (CI + interactive) |

## Interfaces / Contracts

`RiskLevel` (trust.go:10-17) — invariant wording mirrors `interface.go`'s `TrustLevel` comment:

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

`ConfirmDecision` (confirm.go:16-25):

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

Exact replacement for the R4-4 rationale (`confirm.go:83-86`); `default: // RiskHigh` (line 81) stays:

```go
		// Deliberately a default, NOT `case RiskHigh:`: an unknown future
		// RiskLevel (e.g. RiskLevel(99)) MUST resolve to High by semantics —
		// interactive prompt, CI error — not by zero-value coincidence.
		// Relying on the zero value would be fragile against future enum edits.
```

`ClassifyCommand`'s terminal `return RiskLow` (trust.go:86) stays — legitimate classification, not an unset value.

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit | RED proof: `RiskHigh == 0`, `ConfirmError == 0`; `RiskLevel(0)` + untrusted CI → `ConfirmError` | `TestRiskLevel_ZeroValueIsMostRestrictive`, `TestConfirmDecision_ZeroValueIsFailure`, zero-risk D4 cell — FAIL on current main |
| Unit | Zero/unknown risk fail-closed, CI + interactive | D4 cells: `RiskLevel(0)` CI → `ConfirmError`; `RiskLevel(99)` CI → `ConfirmError`, `n` → `ConfirmDeny` |
| Regression | Symbolic uses stay green | `go test ./... -count=1` |

RED→GREEN: `go test ./internal/security/ -run 'ZeroValue|DecisionMatrix' -count=1` FAILS pre-reorder (RiskHigh==2, ConfirmError==3, `RiskLevel(0)`==Low→Auto); reorder turns it GREEN. Full gate: `go test ./... -count=1` → `-race` → `go vet ./...` → `gofmt -l internal/security/` (empty) → `bash scripts/smoke-test.sh --skip-build`.

## Threat Matrix

Applicability: hardens the confirmation gate (internal security boundary); no routing/shell/subprocess/VCS/PR/executable-classification/process-integration logic touched — standard rows N/A. Gate rows Applicable:

| Boundary | Adversarial case | Applicability | Design response | Planned RED tests |
|---|---|---|---|---|
| Gate — zero-value risk | Unset `RiskLevel` | Applicable: zero currently `RiskLow` = silent auto-proceed | `RiskHigh = 0` + invariant | `RiskLevel(0)` + untrusted CI → `ConfirmError` (RED pre-reorder) |
| Gate — unknown risk | `RiskLevel(99)` | Applicable: must fall to High, never exit switch | Retain `default:` + rewritten rationale | CI → `ConfirmError`; interactive `n` → `ConfirmDeny` (pin) |
| Gate — zero-value decision | Unset `ConfirmDecision` | Applicable: zero currently `ConfirmProceed` = fail-open | `ConfirmError = 0` + invariant | `ConfirmError == 0` invariant (RED pre-reorder) |

## Migration / Rollout

No migration required. Single-commit hardening; rollback = `git revert` of the commit (3 files, no persisted data). Fail-open is latent today, so reverting is not a regression vs main.

## Open Questions

- None — spec-neutral verdict leaves no open design decisions.
