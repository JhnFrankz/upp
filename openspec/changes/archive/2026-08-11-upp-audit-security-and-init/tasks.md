# Tasks: upp audit — security confirmation & first-run init fixes

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~700–900 (2 created + 18 modified) |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | PR1 foundation → PR2 matrix → PR3 |
| Delivery strategy | auto-chain |
| Chain strategy | pending |

Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: pending
400-line budget risk: High

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|----------------------|-----------------|-------------------|
| 1 | 3-tier TrustLevel + ToolInfo fields + renames | PR 1 | `go test ./internal/adapters/... -count=1` | N/A — compile+unit gate | revert interface.go, custom.go, official/* |
| 2 | Risk before trust; real command; --ci errors; probes | PR 2 | `go test ./internal/security/... ./internal/cli/ -count=1` | probes run root.Execute w/ fake sudo+marker | revert confirm.go, update.go + tests |
| 3 | Exists() wizard + ApplyDefaults in Load + smoke | PR 3 | `go test ./internal/config/... ./internal/cli/ -run 'Init|Config' -count=1` | `bash scripts/smoke-test.sh --skip-build` | revert config.go, init.go + tests |

## Phase 1: Foundation

- [x] 1.1 RED: `internal/adapters/custom_test.go` — Info() returns TrustCustomTrusted (never Official) + Command/Privileges
- [x] 1.2 GREEN: `internal/adapters/interface.go` — 3-tier TrustLevel enum; ToolInfo +Command +Privileges
- [x] 1.3 GREEN: `internal/adapters/custom.go` — `trusted`→TrustCustomTrusted; fills Command/Privileges
- [x] 1.4 REFACTOR: rename TrustTrusted→TrustOfficial across official/*.go (12 adapters)
- [x] 1.5 Gate: `go test ./internal/adapters/... -count=1`

## Phase 2: Core

- [x] 2.1 RED: `security_expanded_test.go` — ClassifyCommand: sudo, rm -rf, curl|sh, && chains
- [x] 2.2 RED: `confirm_test.go` + `security_expanded_test.go` — CI Low→auto, Med→auto(tr)/err(untr), High→err; interactive High→prompt, Med→info(tr)/prompt(untr)
- [x] 2.3 RED: `internal/cli/audit_probe_test.go` — convert probes: sudo trusted --ci non-zero no-exec; rm -rf interactive trusted+untrusted no-exec
- [x] 2.4 RED: audit_probe_test.go — trusted low-risk executes; --quiet medium-risk still prompts
- [x] 2.5 RED: `internal/cli/init_probe_test.go` — missing → wizard; existing → overwrite prompt
- [x] 2.6 RED: `config_test.go` — Load: missing→no defaults; empty/partial→ApplyDefaults+catalog; full→as-is; Exists()
- [x] 2.7 GREEN: `internal/security/confirm.go` — typed TrustLevel; drop Trusted bool; risk before trust; CI matrix
- [x] 2.8 GREEN: `internal/cli/update.go` — pass info.Command+Privileges to Classify/Confirm; --ci high-risk errors; delete trustLevelString
- [x] 2.9 GREEN: `internal/config/config.go` — add Exists(); ApplyDefaults in Load only when file exists
- [x] 2.10 GREEN: `internal/cli/init.go` — Exists() gate; restore overwrite prompt; single Load
- [x] 2.11 Gate: `go test ./internal/security/... ./internal/config/... ./internal/cli/... -count=1`
- [x] 2.12 RED (amendment, user decision): `security_expanded_test.go` — compact pipes without spaces (`curl url|sh`, `curl url|bash`, `wget url|sh`) classify High risk
- [x] 2.13 GREEN (amendment, user decision): `internal/security/trust.go` — `hasPipeToShell` detects compact `|sh`/`|bash` variants

## Phase 3: Tests

- [x] 3.1 Update official/{info,registry,adapter_update}_test.go + custom_test.go — renames + Trust assertions
- [x] 3.2 Update confirm_test.go/security_expanded_test.go — Trusted bool→TrustLevel; CI low-risk rows (D4)
- [x] 3.3 Update config_test.go — Load-state cases + Exists()
- [x] 3.4 REFACTOR: shared probeSetup helper; drop i18n probe

## Phase 4: Verification

- [x] 4.1 `go test ./... -count=1`
- [x] 4.2 `go test ./... -race`
- [x] 4.3 `go vet ./...` + `gofmt -s -w`
- [x] 4.4 smoke-test.sh empty/partial-config → catalog-defaults assertions; run `--skip-build`
