# Tasks: Simplify upp Usage

RED = failing test; GREEN = passes; VERIFY = gates.

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~330–365 (S1 155–175, S2 175–190) |
| 400-line budget risk | Medium |
| Chained PRs recommended | Yes |
| Suggested split | PR 1 (S1) then PR 2 (S2) |
| Delivery strategy | auto-chain |
| Chain strategy | stacked-to-main |

Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: Medium

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|----------------------|-----------------|-------------------|
| 1 | S1 UX: progress, dry-run, list ID, help, README, drift | PR 1 | `go test -count=1 ./internal/{output,cli}` | `upp check`: "Checking 3/10"; smoke | revert PR 1 (S1 files) |
| 2 | S2 data: skips, nvm semver, interactive | PR 2 | `go test -count=1 ./internal/{output,adapters/official,config,cli}` | `upp check` missing binary → "N up to date, M skipped"; smoke | revert PR 2; no S1 strings |

## Slice 1 (PR #1) — Presentation/UX

- [x] 1.1 RED `render_test.go`: Progress(op, cur, total, name); assert "Checking 3/10" vs "Updating 3/10" (ux).
- [x] 1.2 RED `render_test.go`: dry-run "N would update", never "All clean!" (ux).
- [x] 1.3 RED `render_test.go`: List header "ID | Name | Status | Version" + ListEntry.ID (ux).
- [x] 1.4 RED `internal/cli/help_test.go`: help groups shown, completion absent (ci).
- [x] 1.5 GREEN `render.go` (D2,D3): `Progress(op,…)`; gate `allClean := !DryRun && updated>0 && available==0 && failed==0 && skipped==0`.
- [x] 1.6 GREEN (D2): check.go:88 "Checking", update.go:94 "Updating".
- [x] 1.7 GREEN (D5): ListTools ID column, ListEntry.ID, list.go:68-72 from info.ID.
- [x] 1.8 GREEN `parser.go` (D6): AddGroup + GroupID + HiddenDefaultCmd.
- [x] 1.9 GREEN `README.md:55-72`: zero-config quickstart, `init` optional.
- [x] 1.10 GREEN (D10): ux Self-Update Prompt "localized (en/es)" → English-only aligning delta MODIFIED.
- [x] 1.11 VERIFY: `go test ./... -count=1`; smoke; no "Updating" in check output.

## Slice 2 (PR #2) — Data Quality

- [x] 2.1 RED `render_test.go`: "1 current"→"1 up to date"; skipped → "N up to date, M skipped"; check-with-skips in `integration_test.go`.
- [x] 2.2 RED `check_test.go`: nvm newer-current v26.7.0→v24.19.0 = false; `stable` unparseable = false; existing nvm pass.
- [x] 2.3 RED config tests: drop `Settings.Interactive` in config_test.go, config_expanded_test.go, integration_test.go; stray key tolerated.
- [x] 2.4 GREEN `render.go` (D4): tagline iff `current>0 && available==0 && skipped==0 && failed==0` OR empty list (preserves parser_test.go:238, integration_test.go:647).
- [x] 2.5 GREEN `nvm.go` (D7): semverCompare reusing selfupdate.Parse/Compare (replace :66); v tolerated; Dev/unparseable → false.
- [x] 2.6 GREEN `config.go` (D8): delete `Settings.Interactive` + default; never re-emitted.
- [x] 2.7 GREEN `scripts/smoke-test.sh` test 13: flip `'interactive = false'` grep → `run_test_without_output "interactive"`.
- [x] 2.8 VERIFY: `go test ./... -count=1 -race`; smoke; zero S1-string overlap.

## Final — Archive Sync

- [x] 3.1 `sdd-archive`: merge 4 deltas into `openspec/specs/{ux-patterns,command-interface,tool-adapter,config-system}`; ux merge idempotent (1.10).