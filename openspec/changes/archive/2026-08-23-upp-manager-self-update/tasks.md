# Tasks: Package Manager Self-Update (brew/apt/winget/scoop)

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~380-420 (additions+deletions) |
| 400-line budget risk | Medium |
| Chained PRs recommended | Yes |
| Suggested split | PR1 brew+apt → PR2 winget → PR3 scoop+CLI/ux |
| Delivery strategy | ask-on-risk |
| Chain strategy | pending |

Decision needed before apply: Yes
Chained PRs recommended: Yes
Chain strategy: pending
400-line budget risk: Medium

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|----------------------|-----------------|-------------------|
| 1 | brew+apt self-only Update | PR 1 | `go test ./internal/adapters/official -run 'TestUpdate|TestCheck' -count=1` | `upp update -n` (dry-run) — ConfirmAction skipped | Revert brew/apt Update strings to `brew update && brew upgrade` / `sudo apt upgrade -y -qq` |
| 2 | winget self-only Check+Update | PR 2 | `go test ./internal/adapters/official -run 'TestCheck|TestUpdate|TestParseWinget' -count=1` | N/A (Windows-only manager; hermetic exec seam) | Revert winget.go + helper parse fn; restore `winget upgrade --all` |
| 3 | scoop self-only + CLI/ux dry-run/tests | PR 3 | `go test ./... -count=1` | `upp update -n` brew current, never pending | Revert scoop.go + CLI assertions + ux-patterns delta |

## Strict TDD

Strict short-test-first: RED (write failing test) → GREEN (minimal impl) → REFACTOR. Run `go test ./... -count=1` after every step. Helpers (`parse*Output`) are pure and tested before their adapter. Dry-run rows key the update command with `failIfRun`.

## Unit 1 — PR 1: brew + apt self-only

- [x] 1.1 RED `update_test.go`: change `brewUpdateCmd`→`brew update`, `aptUpdateCmd`→`sudo apt install --only-upgrade apt`; rows assert call + dry-run `failIfRun`. Verify: `go test ./internal/adapters/official -run TestUpdate -count=1` fails.
- [x] 1.2 GREEN `brew.go`: `Update`→`runCmd("brew update")` only + portable-ruby footgun comment; `Check` unchanged. `apt.go`: `Update`→`runCmd("sudo apt install --only-upgrade apt")` + keep `Privileges:["sudo"]`; `Check` unchanged. Verify: tests pass.
- [x] 1.3 REFACTOR rerun `go test ./... -count=1`; verify apt dry-run keeps sudo, brew no `brew update` in Check (check fakes `failIfRun`).

## Unit 2 — PR 2: winget self-only

- [x] 2.1 RED `parity_test.go`: `parseWingetUpgradeOutput` cases — leading-v 4-part (`v1.8.2311` → current tolerated), own row `Microsoft.AppInstaller`, no own row → `found=false`. Verify: fails (helper absent).
- [x] 2.2 GREEN `helper.go`: add fail-closed `parseWingetUpgradeOutput(out) (current, latest, found)`; unparseable → `found=false`, no error. Verify: run 2.1 tests.
- [x] 2.3 RED `check_test.go`: winget rows — `winget --version`→`v1.8.2311` + own row → available; old winget (no row) → `UpdateAvailable=false`, no error. Verify: fails.
- [x] 2.4 GREEN `winget.go`: `Check`→`winget --version` (via `isVersionLike`) + parse `winget upgrade` own row; <1.6 graceful; `Update`→`runCmd("winget upgrade winget")`. Verify: 2.3 passes.
- [x] 2.5 REFACTOR `update_test.go`: winget Before/After real versions (not `"unknown"`); key `winget upgrade winget`; dry-run uses `failIfRun`. `go test ./internal/adapters/official -count=1`.

## Unit 3 — PR 3: scoop self-only + CLI/ux

- [x] 3.1 RED `parity_test.go`: `parseScoopStatusOutput` cases — scoop own row, no-own-row fallback current-only, `WARN` stderr tolerated. Verify: fails.
- [x] 3.2 GREEN `helper.go`: fail-closed `parseScoopStatusOutput`; unparseable → `found=false`.
- [x] 3.3 RED `check_test.go`: scoop rows — own row → available; no row → false. Verify: fails.
- [x] 3.4 GREEN `scoop.go`: `Check`→parse `scoop status`; `Update`→`runCmd("scoop update scoop")` (never `*`). Verify: 3.3 passes.
- [x] 3.5 RED `cli/update_test.go`: brew dry-run shows current (no `-n` signal), brew never pending in TTY selector; apt/winget planned in `-n`. Verify: fails.
- [x] 3.6 REFACTOR: assert existing update.go/checkrun.go unchanged; verify brew `PolicyAlwaysUpdate` false-by-design honors sequential/`--ci`. `go test ./... -count=1`.
- [x] 3.7 Archive: apply spec-delta notes 1-5 (tool-adapter table/Gating, ux-patterns row rendering) to `openspec/specs/`.

## Risks / Gaps

- scoop `scoop status` output shape (stderr WARN + stdout table assumed) — verify in apply; fallback current-only if unstable.
- winget own-row match key (`Microsoft.AppInstaller` vs `winget`) — decide in apply; helper stays a pure func.
- Interactive brew gap (D7) accepted: brew never runs in TTY, only sequential/`--ci`.
- Bulk-upgrade removal is a behavioral break — release note + point-6 restore (proposal Rollback).
- Companion skill files (`strict-tdd.md`, `work-unit-commits`, `sdd-phase-common`) not found on disk; guidance inferred from injected SDD contract and prompt.
