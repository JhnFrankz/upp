# Apply Progress: upp-adapter-update-policy — SLICE 1 (PR 1) + SLICE 2 (PR 2) COMPLETE

Status: `success` — both slices implemented under Strict TDD. PR 1 merged on main (d8d9963). Slice 2 (Phase 3: Check Failure Signal) implemented as PR 2 batch; all Phase 4 gates (4.2–4.6) green. ALL tasks `[x]` → ready for sdd-verify / sdd-archive.

## Tasks Status (tasks.md mirrors this)

- [x] 1.1 RED goldens gain UpdatePolicy
- [x] 1.2 RED consistency assertion
- [x] 1.3 GREEN enum + field
- [x] 1.4 GREEN 13 Info() sites
- [x] 2.1 RED matrix re-keyed ID→policy
- [x] 2.2 RED all 21 fake literals explicit
- [x] 2.3 GREEN gate rewrite + gatedOfficialAdapters deleted
- [x] 3.1 RED helper-variant seam tests (commandOutputErr/shellOutputErr)
- [x] 3.2 RED apt/nvm command-fails rows flip wantErr
- [x] 3.3 RED npm/pnpm maskless + exit-1/124 rows
- [x] 3.4 GREEN helper variants + structured failure builder
- [x] 3.5 GREEN apt/nvm error-aware detection reads
- [x] 3.6 GREEN npm/pnpm maskless + exit-code interpretation
- [x] 4.1–4.6 all gates green

## TDD Cycle Evidence (Strict TDD Mode)

| Task | Test File | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|------|-----------|-------|------------|-----|-------|-------------|----------|
| 1.1 | `official/info_test.go` | Unit | ✅ 26 pass | ✅ compile-fail (unknown field/constants) → runtime fail (8 AlwaysUpdate sites got 0, want 1) | ✅ 12/12 DeepEqual | ✅ 12 rows (4 Gated + 8 AlwaysUpdate) | ➖ None needed |
| 1.2 | `official/adapter_update_test.go` | Unit | ✅ 26 pass | ✅ compile-fail (same batch) | ✅ 12/12 | ➖ Single (set membership) | ➖ None needed |
| 1.3 | `adapters/interface.go` | Unit | — | ✅ (see 1.1 runtime RED) | ✅ enum + field | ✅ Gated/AlwaysUpdate both named | ✅ invariant comment mirrors TrustLevel |
| 1.4 | 13 Info() sites | Unit | — | ✅ (see 1.1) | ✅ goldens green | ✅ 13 sites, two policies | ✅ gofmt alignment |
| 2.1 | `cli/update_test.go` | Integration | ✅ cli package pass | ✅ compile-fail (policy field/constants); matrix = behavior-identity approval (D2) — passed under OLD gate, still passes under NEW gate | ✅ 7/7 rows | ✅ 7 rows: Gated±update, AlwaysUpdate ×3, custom ×2, failed-check | ✅ status assertion switch |
| 2.2 | 3 cli test files (21 literals) | Integration | ✅ | ✅ compile-fail (same batch) | ✅ hermetic assertions unchanged | ✅ 21 sites across 3 files | ➖ None needed |
| 2.3 | `internal/cli/update.go` | Integration | ✅ | — (matrix is the guard) | ✅ full suite green | ✅ gate predicate + StatusFailed row | ✅ comments updated, dry-run untouched |
| 3.1 | `official/adapter_update_test.go` | Unit | ✅ official pass (26+ tests) | ✅ compile-fail: `undefined: commandOutputErr` ×4, `undefined: shellOutputErr` ×4 | ✅ 8/8 subtests (4 per variant) | ✅ 4 cases/variant: success, structured fake failure (+stdout preserved), real `sh -c exit 7` child (ExitError code 7 via errors.As), DeadlineExceeded errors.Is | ✅ stdout-preserved-on-failure contract (D4) pinned in tests |
| 3.2 | `official/check_test.go` | Unit | ✅ | ✅ 10 rows fail: apt/command-fails (wantErr, got nil), nvm/command-fails, npm/pnpm update-available (maskless key), exit-1/124, deadline | ✅ 12/12 | ✅ apt: command-fails (err) + empty-output + "(none)" + update-available (no err); nvm: command-fails + empty-current + same-version | ➖ None needed |
| 3.3 | `official/check_test.go` | Unit | ✅ (same batch as 3.2) | ✅ exit-1-outdated: `Check() = {... UpdateAvailable:false}, want ... true` (exact: check_test.go:584) | ✅ exit-1 + exit-124 + deadline rows green | ✅ npm: exit-1+output (true), exit-1+empty (false), exit-124 (err), DeadlineExceeded (err + errors.Is); pnpm: exit-1 (true), exit-124 (err) | ✅ shared `exitErrFromChild` helper + `exitCode` harness field |
| 3.4 | `official/helper.go` | Unit | — | — (3.1 tests are the guard) | ✅ variants delegate to runCmdArgsFn/runCmdFn; `commandFailureErr` = `"<tool> check failed (exit N): <stderr excerpt>: %w"`, exit omitted when not extractable | ✅ exit present (real child), exit omitted (plain fake err, DeadlineExceeded) | ✅ empty-stderr segment omitted; `shellToolName` first-token label |
| 3.5 | `apt.go`/`nvm.go` | Unit | — | — (3.2 rows are the guard) | ✅ detection reads via shellOutputErr; display-only CurrentVersion()/currentVersion() stay plain | ✅ apt installed+candidate both error-aware; nvm current+remote both | ➖ None needed |
| 3.6 | `npm.go`/`pnpm.go` | Unit | — | — (3.3 rows are the guard) | ✅ `|| true` dropped, `timeout 15` kept; `isExitCode(err, 1)` → valid detection (stdout decides), other non-zero → structured error | ✅ npm exit-1 with/without output; pnpm exit-1 | ✅ shared `isExitCode` helper (no duplication) |

Test Summary: 56 subtests in the official package (TestCheck ~39 rows + TestCommandOutputErr 4 + TestShellOutputErr 4 + others). All passing. Layers: Unit.

## Work Unit Evidence

### PR 1 (enum + gate + explicitness — merged)

| Evidence | Value |
|---|---|
| Focused test command and exact result | `go test ./internal/adapters/official/ -run 'TestInfo\|TestAllAdapters_InfoConsistency' -count=1` → 26 PASS; `go test ./internal/cli/ -run 'TestRunUpdate' -count=1` → 8/8 PASS |
| Runtime harness command/scenario and exact result | `bash scripts/smoke-test.sh --skip-build` → 23 passed, 0 failed |
| Rollback boundary | Revert: `interface.go`, 13 Info() sites, `cli/update.go`, `official/info_test.go`, `official/adapter_update_test.go`, `cli/{update,integration,audit_probe}_test.go` |

### PR 2 (Check Failure Signal — this batch)

| Evidence | Value |
|---|---|
| Focused test command and exact result | `go test ./internal/adapters/official/ -run 'TestCheck|TestCommandOutputErr|TestShellOutputErr' -count=1` → 56 subtests PASS (incl. npm/pnpm exit-1-outdated, exit-124-timeout, deadline-exceeded, apt/nvm command-fails) |
| Runtime harness command/scenario and exact result | `bash scripts/smoke-test.sh --skip-build` → 23 passed, 0 failed (stubs report StatusCurrent; npm/pnpm/apt exit interpretation exercised via seam fakes — real tools N/A in hermetic suite) |
| Rollback boundary | Revert exactly: `internal/adapters/official/{helper,apt,nvm,npm,pnpm}.go` + `official/{check_test,adapter_update_test}.go` (7 files). Nothing else in the tree references `commandOutputErr`/`shellOutputErr`/`isExitCode`; grep-verified. CLI untouched (check.go already mapped err → StatusFailed) |

## Files Changed (slice 2 — exact `git diff --stat`)

```
 internal/adapters/official/adapter_update_test.go | 131 ++++++++++++++++++++
 internal/adapters/official/apt.go                 |  10 +-
 internal/adapters/official/check_test.go          | 141 +++++++++++++++++++++-
 internal/adapters/official/helper.go              |  61 ++++++++++
 internal/adapters/official/npm.go                 |   9 +-
 internal/adapters/official/nvm.go                 |  10 +-
 internal/adapters/official/pnpm.go                |   9 +-
 7 files changed, 359 insertions(+), 12 deletions(-)
```

## Gates (final state — all green)

- `go test ./internal/adapters/official/ -count=1 -race` (4.2) ✅ ok
- `go test ./... -count=1` (4.3) ✅ 8 packages ok
- `go test ./... -race -count=1` (4.4) ✅ all packages ok
- `go vet ./...` (4.5) ✅ exit 0
- `gofmt -l internal/` (4.5) ✅ empty
- `bash scripts/smoke-test.sh --skip-build` (4.6) ✅ 23 passed, 0 failed

## Deviations from Design

- None — implementation matches design.md D3/D4/D5 and the spec's ADDED Check Failure Signal exactly. Two notes:
  - D3 format implemented as `"<tool> check failed (exit N): <stderr excerpt>: %w"` with the empty-stderr excerpt segment omitted (no dangling `": :"`); exit code omitted when not extractable (plain fake err, DeadlineExceeded) — matches D3's "omitted when not extractable".
  - The error-aware helpers PRESERVE stdout on failure (return trimmed stdout alongside the error). D4's "exit 1 = valid detection (stdout decides availability)" requires it; without preservation the exit-1 path could never see the outdated rows. Pinned by tests.
  - `shellToolName` derives the failure label as the first whitespace token of the shell command (the first binary in the pipeline): apt reads label as "apt-cache", nvm reads as "source". Deterministic and test-pinned.

## Notes / Risks

- check_test.go harness gained three optional fields (`wantErrContains`, `wantDeadline`, `exitCode`); `exitCode` rows run a REAL `sh -c exit N` child to inject a genuine `*exec.ExitError` into the seam fake (plain `errors.New` fakes cannot exercise `errors.As`-based exit interpretation). Skipped on windows like TestRunCmd_*.
- Changed lines 371 (359+12) for PR 2, under the 400-line budget.
