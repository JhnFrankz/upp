# Apply Progress: CI/CD Best Practices (ci-cd-best-practices)

- Change: `ci-cd-best-practices`
- Date: 2026-08-13
- Mode: Strict TDD (config-only — RED/GREEN evidence = the local verification gate the new CI steps will enforce)
- Batch: 1 of 1 (tasks 1.1–1.4, 2.1). Delivery tasks 3.1–3.4 are owned by the orchestrator.
- Files changed:
  - `.github/workflows/ci.yml` — +15 lines (concurrency block, 3× `timeout-minutes`, 2 lint steps)
  - `scripts/install.sh` — +2/−1 (SC2155 fix, behavior identical)

## Completed Tasks

- [x] 1.1 Top-level `concurrency` after `permissions`: `group: ci-${{ github.ref }}`, `cancel-in-progress: ${{ !startsWith(github.ref, 'refs/tags/') }}`
- [x] 1.2 `timeout-minutes`: test 15, lint 10, release 20
- [x] 1.3 Shellcheck step appended in `lint` AFTER golangci-lint
- [x] 1.4 Actionlint step appended in `lint` (pinned `@v1.7.7`)
- [x] 2.1 (conditional, TRIGGERED) Fixed SC2155 in `scripts/install.sh`

## Verification Evidence (local gate — the change is not done until every check passes)

| # | Check | Command | Result |
|---|-------|---------|--------|
| 1 | Actionlint | `go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.7 .github/workflows/ci.yml` | exit 0, no findings |
| 2 | Shellcheck (before fix) | `/tmp/opencode/bin/shellcheck -S warning scripts/install.sh scripts/smoke-test.sh scripts/publish-release.sh` (v0.10.0 static binary) | exit 1 — 1 finding: SC2155 `scripts/install.sh:203` (`local backup_path="${install_path}.backup.$(date +%s)"`) |
| 3 | Shellcheck (after fix) | same command | exit 0, zero findings |
| 4 | PyYAML parse | `python3 -c "import yaml; yaml.safe_load(open('.github/workflows/ci.yml'))"` | exit 0, parse OK |
| 5 | Syntax | `bash -n scripts/install.sh scripts/smoke-test.sh scripts/publish-release.sh` | exit 0 on all three |
| 6 | Go suite (safety net, after ci.yml edit) | `go test ./... -count=1` | exit 0 — 9 packages ok (cmd/upp no test files; adapters, adapters/official, cli, config, output, platform, security, selfupdate ok) |
| 7 | Go suite (final state, after install.sh fix) | `go test ./... -count=1` | exit 0 |

### TDD Cycle Evidence

Config-only change: no Go tests are written (no Go code changes); the RED state is the lint job failing on findings (the gate CI will enforce), proven locally by the pre-fix shellcheck exit 1. GREEN is every local gate passing post-fix.

| Task | Test File | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|------|-----------|-------|------------|-----|-------|-------------|----------|
| 1.1 | N/A (config) | N/A | ✅ `go test ./... -count=1` 9/9 ok | ✅ actionlint/YAML gate (would fail on invalid config) | ✅ actionlint exit 0, YAML parse exit 0 | ➖ Single output (config block) | ➖ None needed |
| 1.2 | N/A (config) | N/A | ✅ same baseline | ✅ gate | ✅ actionlint exit 0, YAML parse exit 0 | ➖ Single output | ➖ None needed |
| 1.3 | N/A (config + scripts) | N/A | ✅ same baseline | ✅ shellcheck exit 1 (SC2155) | ✅ shellcheck exit 0 after fix | ✅ 3 scripts checked | ✅ SC2155 fix (declare/assign split) |
| 1.4 | N/A (config) | N/A | ✅ same baseline | ✅ gate | ✅ actionlint exit 0 | ➖ Single output | ➖ None needed |
| 2.1 | N/A (scripts) | N/A | ✅ `bash -n` all scripts | ✅ shellcheck exit 1 (SC2155) | ✅ shellcheck exit 0 | ✅ 3 scripts, zero findings | ✅ minimal 2-line change |

### Work Unit Evidence

| Evidence | Value |
|----------|-------|
| Focused test command + result | `shellcheck -S warning scripts/install.sh scripts/smoke-test.sh scripts/publish-release.sh` → exit 0, zero findings; `go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.7 .github/workflows/ci.yml` → exit 0 |
| Runtime harness | N/A for this batch — the runtime boundary is the PR CI run (lint job executes the new steps), which the orchestrator drives at delivery. Local gates replicate each new step's command exactly. |
| Rollback boundary | Revert the single ci.yml commit (+ `scripts/install.sh` hunk) → pipeline returns to pre-change behavior; no unrelated work touched. |

## Deviations from Design

- None of substance. Two cosmetic notes:
  - Shellcheck apt install uses `-qq` (`sudo apt-get update -qq && sudo apt-get install -y -qq shellcheck`) per the apply instructions; design.md shows the non-quiet form. Behavior identical.
  - Task 1.3 text shows `sudo apt-get update && sudo apt-get install -y shellcheck`; applied the `-qq` variant from the orchestrator instructions. No functional difference.
- Post-PR #26 "Restore annotated tag object" step (single-quoted 5x-retry run) untouched, verbatim.

## Issues Found

- Task 2.1 WAS needed: 1 shellcheck warning (SC2155 in `scripts/install.sh:203`). Fixed by splitting `local backup_path` declaration from the `date +%s` assignment. Behavior identical; re-ran shellcheck → clean.

## Remaining Tasks

- [ ] 3.1 Issue creation (orchestrator)
- [ ] 3.2 Commit + PR (orchestrator)
- [ ] 3.3 Branch protection apply (orchestrator)
- [ ] 3.4 Read-back verify (orchestrator)

## Workload / PR Boundary

- Mode: single PR (one work unit: CI hardening + conditional script fix in the same commit)
- Workload forecast: Low risk (~40–60 lines); actual diff: 17 insertions, 1 deletion (2 files) — well under budget
- Suggested commit (orchestrator-owned): `chore(ci): harden CI with concurrency, timeouts, and script linting`

## Status

5/5 assigned tasks complete (1.1–1.4, 2.1). Ready for verify.
