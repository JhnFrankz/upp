# Tasks: upp trust zero-value fail-closed hardening

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 60–100 (3 files) |
| 400-line budget risk | Low |
| Chained PRs recommended | No |
| Suggested split | Single PR |
| Delivery strategy | auto-chain |
| Chain strategy | single-pr |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: single-pr
400-line budget risk: Low

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|----------------------|-----------------|-------------------|
| 1 | Zero-value fail-closed hardening: enum reorder, default-branch comment, fallback tests | PR 1 | `go test ./internal/security/... ./internal/adapters/... ./internal/cli/... -count=1` | `bash scripts/smoke-test.sh --skip-build` | `git revert` of change commit — only `internal/adapters/interface.go`, `internal/security/confirm.go`, `internal/security/security_expanded_test.go` |

## Phase 1: RED Tests (write first — must fail on current code)

- [x] 1.1 Add D4 cells to `TestConfirmAction_DecisionMatrix` in `internal/security/security_expanded_test.go`: `TrustLevel(0)` + `RiskHigh` CI → `ConfirmError`; `TrustLevel(0)` + `RiskMedium` CI → `ConfirmError` (zero value is currently `TrustOfficial` → auto-proceeds, so RED pre-reorder)
- [x] 1.2 Add `TrustLevel(99)` + `RiskMedium` CI → `ConfirmError` and `TrustLevel(99)` + `RiskHigh` interactive → prompt cells (untrusted fail-closed pin)
- [x] 1.3 Add `RiskLevel(99)` + CI → `ConfirmError` and `RiskLevel(99)` + interactive `n\n` → `ConfirmDeny` cells (default-branch pin)
- [x] 1.4 Add `TestTrustLevel_ZeroValueIsLeastPrivileged` asserting `TrustCustomUntrusted == 0`
- [x] 1.5 Run `go test ./internal/security/... -count=1`: zero-value cells FAIL, `TrustLevel(99)`/`RiskLevel(99)` cells pass

## Phase 2: Foundation (GREEN via enum reorder)

- [x] 2.1 Reorder `TrustLevel` const block in `internal/adapters/interface.go`: `TrustCustomUntrusted TrustLevel = 0`, `TrustCustomTrusted TrustLevel = 1`, `TrustOfficial TrustLevel = 2` (explicit values, per design interface contract)
- [x] 2.2 Add invariant comment to `TrustCustomUntrusted`: zero value MUST stay least-privileged (fail-closed); never insert a tier before it

## Phase 3: Core (confirm.go comment)

- [x] 3.1 In `internal/security/confirm.go`, extend the `default: // RiskHigh` comment (line 81): it must stay a default, never become `case RiskHigh:` — an explicit case would let unknown risk values exit the switch as zero-value `ConfirmProceed` (fail-open, R4-4)

## Phase 4: Verification

- [x] 4.1 Gate: `go test ./internal/security/... ./internal/adapters/... ./internal/cli/... -count=1` green (GREEN confirmation)
- [x] 4.2 `go test ./... -count=1` green
- [x] 4.3 `go test ./... -count=1 -race` green
- [x] 4.4 `go vet ./...` clean
- [x] 4.5 `gofmt -s -l .` clean (run `gofmt -s -w` if diffs appear)
- [x] 4.6 `bash scripts/smoke-test.sh --skip-build` green
