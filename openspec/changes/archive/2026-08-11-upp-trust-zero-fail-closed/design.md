# Design: upp trust zero-value fail-closed hardening

## Technical Approach

Make the security trust enum fail closed by construction: reorder `TrustLevel` in `internal/adapters/interface.go` so the zero value is `TrustCustomUntrusted` (least privileged), pin it with explicit constant values and an invariant comment, and add fallback tests proving that unset/unknown trust and unknown risk all route to the risk matrix / High branch. `internal/security/confirm.go` keeps its `default: // RiskHigh` branch with a comment explaining why it must not become `case RiskHigh:`. Pure hardening per the spec-neutral verdict (Engram #84) — no spec deltas, no behavior change beyond the zero-value semantic. All existing uses are symbolic (`==` comparisons, `String()` switch, table-driven tests); grep confirms no numeric persistence (only `%d` in test error messages).

## Architecture Decisions

### Decision: Enum reorder with explicit values

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Reorder + bare iota | Least diff, but invariant invisible in source; future insertions silently renumber | **Explicit `= 0 / = 1 / = 2` values + invariant comment** — the invariant is self-documenting and reorder-proof |
| Reorder + explicit values + comment | Slightly noisier diff (3 lines) | Chosen |

### Decision: Retain `default: // RiskHigh` in ConfirmAction

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Keep `default: // RiskHigh` + comment | A future unknown `RiskLevel` (e.g. `RiskLevel(99)`) falls into High = prompt/error = fail-closed | **Chosen** |
| Replace with `case RiskHigh:` | Unknown risk values would exit the switch unhandled, returning zero-value `ConfirmProceed` = fail-open | Rejected — this is the R4-4 trap |

### Decision: Test placement

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Extend D4 table + one invariant test | Zero/unknown trust + unknown risk fit the existing matrix cells; the `== 0` pin is a different assertion (compile-time invariant), deserves its own test | **Both** — D4 table gains fallback cells; `TestTrustLevel_ZeroValueIsLeastPrivileged` pins the invariant |

## Data Flow

```
internal/cli/update.go:73  info := a.Info()          // ToolInfo.Trust (TrustLevel)
internal/cli/update.go:128 ConfirmAction(ConfirmConfig{TrustLevel: info.Trust, RiskLevel: riskLevel, CI: gf.CI})
internal/security/confirm.go:53  TrustOfficial? → ConfirmAuto (only explicit official)
internal/security/confirm.go:57  switch RiskLevel → Low/Medium/default(RiskHigh) → ConfirmError | prompt | info
```

Zero-value `TrustLevel` no longer equals `TrustOfficial` (post-reorder), so it falls through to the risk matrix as untrusted: High → CI `ConfirmError` / interactive prompt; Medium → CI error (untrusted) / prompt; Low → auto/info. Unknown `TrustLevel(99)` behaves identically (not `== TrustOfficial`, not `== TrustCustomTrusted`).

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/adapters/interface.go` | Modify | Reorder enum: `TrustCustomUntrusted = 0`, `TrustCustomTrusted = 1`, `TrustOfficial = 2`; explicit values + zero-value invariant comment (R4-1) |
| `internal/security/confirm.go` | Modify | Comment only on `default: // RiskHigh` — why it must stay a default, not `case RiskHigh:` (R4-4) |
| `internal/security/security_expanded_test.go` | Modify | D4 table fallback cells + `TrustCustomUntrusted == 0` invariant test (R4-3) |

## Interfaces / Contracts

`TrustLevel` (internal/adapters/interface.go) — the invariant pattern:

```go
const (
	// TrustCustomUntrusted is for custom adapters, untrusted by default.
	// It is the ZERO value on purpose: an unset TrustLevel MUST resolve to the
	// least-privileged level so unset trust fails closed. The zero value MUST
	// stay the least-privileged tier — never insert a new level before it.
	TrustCustomUntrusted TrustLevel = 0
	// TrustCustomTrusted is for custom adapters marked trusted=true in config.
	// It must never alias TrustOfficial: trust level MUST NOT bypass the risk matrix.
	TrustCustomTrusted TrustLevel = 1
	// TrustOfficial is for official, built-in adapters.
	TrustOfficial TrustLevel = 2
)
```

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit | Zero-value trust: `adapters.TrustLevel(0)` + High/Medium risk → CI `ConfirmError`, interactive prompt (untrusted semantics) | Extend `TestConfirmAction_DecisionMatrix` table (D4) |
| Unit | `TrustLevel(99)` → not official, not trusted → untrusted path (CI medium error; interactive high prompt) | D4 table cells |
| Unit | `RiskLevel(99)` → `default` branch: CI `ConfirmError`; interactive prompt (`n\n` → `ConfirmDeny`) | D4 table cells |
| Unit | Invariant: `TrustCustomUntrusted == 0` | New `TestTrustLevel_ZeroValueIsLeastPrivileged` |
| Regression | Existing 18-cell D4 matrix + all suites stay green (symbolic refs) | `go test ./... -count=1`, `-race`, `go vet`, `gofmt -l`, smoke |

RED→GREEN (strict TDD): the zero-value trust tests are RED against current code (zero = `TrustOfficial` → auto-proceeds); enum reorder turns them GREEN. The `TrustLevel(99)` / `RiskLevel(99)` cells already pass today and pin the fail-closed behavior against future edits.

## Threat Matrix

Applicability: this change hardens the confirmation gate — an internal security boundary, not routing/shell/process automation. Standard rows are N/A; gate rows are Applicable.

| Boundary | Minimum adversarial cases | Applicability | Design response | Planned RED tests |
|---|---|---|---|---|
| Documentation-like paths | `requirements.txt`, `README.sh` | N/A: no file-classification logic touched | — | — |
| Git repository selection | `git -C`, relative paths | N/A: no git invocation touched | — | — |
| Commit state | staged, `commit -a`, empty index | N/A: no commit automation touched | — | — |
| Push state | tracking branch, first push | N/A: no push automation touched | — | — |
| PR commands | `--head`, composed commands | N/A: no PR automation touched | — | — |
| Confirmation gate — zero-value trust | Unset `TrustLevel` (struct literal) at High/Medium risk | Applicable: zero value currently `TrustOfficial` = silent auto-proceed (fail-open) | Reorder enum; zero = `TrustCustomUntrusted` | `TrustLevel(0)` + High CI → `ConfirmError` (RED pre-reorder) |
| Confirmation gate — unknown trust value | `TrustLevel(99)` | Applicable: must behave as untrusted, never bypass matrix | No code path change; risk matrix applies | `TrustLevel(99)` + Medium CI → `ConfirmError`; + High interactive → prompt |
| Confirmation gate — unknown risk value | `RiskLevel(99)` | Applicable: must fall to High/default, never exit switch | Retain `default: // RiskHigh`; document R4-4 | `RiskLevel(99)` + CI → `ConfirmError`; + interactive → `ConfirmDeny` on `n` |

## Migration / Rollout

No migration required. Single-commit hardening; rollback = `git revert` of the commit (3 files, no data).

## Open Questions

- None — spec-neutral verdict (Engram #84) leaves no open design decisions.
