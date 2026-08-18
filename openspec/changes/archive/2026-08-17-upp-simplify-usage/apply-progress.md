# Apply Progress: upp-simplify-usage — Slice 1 (PR #1)

**Change**: upp-simplify-usage · **Phase**: apply · **Slice**: 1 (presentation/UX) · **Date**: 2026-08-16
**Slice 2 (PR #2) apply-progress will be appended/merged here by the S2 apply batch** (topic_key `sdd/upp-simplify-usage/apply-progress`).
**Mode**: Strict TDD (red/green/verify gates executed) · **Delivery**: auto-chain, stacked-to-main (PR #1 → main; PR #2 on top later) · **Review budget**: S1 ≈ 155–175 changed lines.

## Decisions Applied

D1 (slice ownership: S1 owns Progress / UpdateSummary dry-run gate / ListTools+ListEntry.ID — CheckSummary untouched), D2 (per-operation verb), D3 (dry-run "All clean!" gate), D5 (list ID column), D6 (cobra groups + hidden completion), D9 (list table home → ux-patterns), D10 (direct canonical ux-patterns drift fix).

## Slice 1 Tasks (all complete)

- [x] 1.1 RED `render_test.go`: Progress(op, cur, total, name); assert "Checking 3/10" vs "Updating 3/10".
- [x] 1.2 RED `render_test.go`: dry-run "N would update", never "All clean!".
- [x] 1.3 RED `render_test.go`: List header "ID | Name | Status | Version" + ListEntry.ID.
- [x] 1.4 RED `internal/cli/help_test.go`: help groups shown, completion absent.
- [x] 1.5 GREEN `render.go` (D2,D3): Progress(op,…); allClean gate.
- [x] 1.6 GREEN (D2): check.go "Checking", update.go "Updating".
- [x] 1.7 GREEN (D5): ListTools ID column, ListEntry.ID, list.go from info.ID.
- [x] 1.8 GREEN `parser.go` (D6): AddGroup + GroupID + HiddenDefaultCmd.
- [x] 1.9 GREEN `README.md:55-72`: zero-config quickstart, `init` optional.
- [x] 1.10 GREEN (D10): ux Self-Update Prompt "localized (en/es)" → English-only aligning delta MODIFIED.
- [x] 1.11 VERIFY: full suite, smoke, live harness — no "Updating" in check output.

**Slice 2 (not started)**: 2.1–2.8 pending. **Archive**: 3.1 pending.

## TDD Cycle Evidence

| Work unit | Task | Tests | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|---|---|---|---|---|---|---|---|---|
| AUX-1 Progress verb | 1.1, 1.6 | render_test.go (TestProgress_CheckVerb, TestProgress_UpdateVerb, TestProgress_SingleTool) + integration_test.go (TestCheckProgress_LabelsChecking) | Unit + Integration | ✅ 8/8 output, 21/21 cli | ✅ compile-fail (old signature/absent behavior) then behavior-fail | ✅ run | ✅ 3 cases (Checking, Updating, no-progress-single) + hermetic wiring | ✅ gofmt -s clean |
| AUX-2 dry-run gate | 1.2, 1.5 | render_test.go (TestUpdateSummary_DryRun updated, TestUpdateSummary_NotCleanWithSkips) + counter-case TestUpdateSummary_AllUpdated locked | Unit | ✅ baseline above | ✅ "All clean!" wrongly present at RED | ✅ run | ✅ 3 cases (dry-run pending, updated+skipped, all-updated) | ✅ gofmt -s clean |
| AUX-3 list ID column | 1.3, 1.7 | render_test.go (TestListTools updated, TestListTools_IDColumn) | Unit | ✅ baseline above | ✅ compile-fail (unknown field ID) | ✅ run | ✅ 2 entries + header + ID≠Name | ✅ gofmt -s clean |
| AUX-4 help groups | 1.4, 1.8 | help_test.go (TestHelp_ShowsGroups, TestHelp_CommandsListed, TestHelp_EqualsHelpSubcommand) | Integration (hermetic) | ✅ baseline above | ✅ group titles absent / "completion" present at RED | ✅ run | ✅ 3 cases (--help, command listing, help subcommand) | ✅ gofmt -s clean |
| AUX-5 README | 1.9 | none — docs task (no test runner applies) | Docs | n/a | n/a | ✅ write | ➖ N/A | ✅ consistent with Commands table |
| AUX-6 spec drift | 1.10 | none — canonical spec text; verified by delta-alignment diff | Docs | n/a | n/a | ✅ write | ➖ N/A | ✅ "localized (en/es)/Localized prompt" gone from canonical |

## Work Unit Evidence (Hard Gate)

| Work unit | Focused test | Result | Runtime harness | Rollback boundary |
|---|---|---|---|---|
| AUX-1 | `go test ./internal/output ./internal/cli -count=1` | ok; then `./upp check` live → "Checking X/Y", grep -c "Updating" = 0 | `upp check` (live, 10 tools): verify real progress text + zero "Updating" | revert render.go Progress fm/ call-sites in check.go/update.go + the two render_test.go cases + integration test |
| AUX-2 | `go test ./internal/output -count=1` (TestUpdateSummary_*) | ok | `upp update --dry-run` (live): summary "1 would update", no "All clean!" | revert UpdateSummary allClean gate; tests revert with it |
| AUX-3 | `go test ./internal/output -count=1` (TestList*) | ok | `upp list` (live): header `ID    Name`..., rows show apt/brew/nvm/npm (filter IDs) | revert ListEntry.ID/ListTools + render_test cases + list.go fill |
| AUX-4 | `go test ./internal/cli -count=1` (help_test.go) | ok | `upp --help` (live): Tool Commands/Config Commands/Maintenance, no "completion" | revert parser.go AddGroup/GroupID/CompletionOptions + help_test.go |
| AUX-5 | n/a (docs) | n/a | smoke test 1-2 unaffected; README quickstart commands identical | revert README quickstart block |
| AUX-6 | n/a (docs) | n/a | canonical spec diff clean vs delta MODIFIED | revert openspec/specs/ux-patterns/spec.md:102,:106 |

## Verification (task 1.11)

- `gofmt -s -w` on all touched Go files → `gofmt -l internal/ cmd/` empty.
- `go vet ./...` → clean.
- `go test ./... -count=1` → all packages ok (adapters, official, cli, config, output, platform, security, selfupdate; no test files in cmd).
- `go build ./cmd/upp` → ok.
- `bash scripts/smoke-test.sh --skip-build` → **23 passed, 0 failed**.
- Live harness: `upp check` prints "Checking 1/10" … "Checking 10/10" (read-only verb), `grep -c Updating` = 0; `upp update --dry-run` prints "Updating X/Y" and summary "1 would update" with **no** "All clean!"; `upp list` header `ID  Name  Status  Version` with filter IDs (apt/brew/nvm/npm/…).
- Marker guard: `"All tools up to date."` semantics untouched (parser_test.go:238, integration_test.go:647 still pass — CheckSummary is S2 scope).

## Files Touched (Slice 1)

| File | Action | What |
|---|---|---|
| `internal/output/render.go` | Modified | Progress(op,…) + template (D2); allClean gate (D3); ListTools ID header/row + ListEntry.ID (D5) |
| `internal/cli/check.go` | Modified | `Progress("Checking", …)` (:88) |
| `internal/cli/update.go` | Modified | `Progress("Updating", …)` (:94) |
| `internal/cli/list.go` | Modified | ListEntry.ID from `info.ID` (:68-72) |
| `internal/cli/parser.go` | Modified | CompletionOptions{HiddenDefaultCmd:true} + AddGroup(Tool/Config/Maintenance) + GroupID per command (:37-47, :58-86) |
| `README.md` | Modified | Zero-config quickstart (:55-72); `init` optional |
| `openspec/specs/ux-patterns/spec.md` | Modified | D10 drift fix: "localized (en/es)"/"Localized prompt" → English-only (:102, :106) |
| `internal/output/render_test.go` | Modified | RED/GREEN for AUX-1/2/3 |
| `internal/cli/help_test.go` | **Created** | Hermetic help-group + hidden-completion tests (AUX-4) |
| `internal/cli/integration_test.go` | Modified | TestCheckProgress_LabelsChecking (AUX-1 wiring) |
| `internal/cli/update_test.go` | Modified | `fakeAdapterList` variadic (multi-tool fakes) |
| `openspec/changes/upp-simplify-usage/tasks.md` | Modified | Slice 1 → [x] marks |
| `openspec/changes/upp-simplify-usage/apply-progress.md` | **Created** | this file |

## Deviations from Design

1. **List table icon dropped** (D5). Design maps rows to `ID | Name | Status | Version` with no room for the old leading status icon. To keep the header honest ("ID" column shows exactly the `--only`/`--skip` ID), the emoji glyph was dropped from the list table; the Status column already carries the label (e.g. "current"/"skipped"), and TabWriter columns stay aligned. No test asserted the icon.

## Issues Found / Leftover Notes for Slice 2

- **S2 heads-up (nvm)**: live `check` still reports the phantom `Node Version Manager v26.7.0 → v24.19.0` — that is exact S2 scope (2.2/2.5), untouched here.
- `update --dry-run` summary prefix in plain mode reads `[updated] 1 would update` (existing icon behavior, unchanged).
- `upp --help` renders an `Additional Commands:` section containing only cobra's builtin `help` command (its default GroupID is empty). All 7 product commands are grouped; `completion` is absent. If undesired in S2+, the fix is `root.SetHelpCommandGroupID` + a "help" group — out of S1 scope (spec has no requirement about `help` rendering).
- `cfg.Settings.Interactive` references remain in config tests / integration_test.go / smoke test 13 (`interactive = false`) — those belong to S2 (2.3/2.6/2.7) and were deliberately NOT touched.
- Permitted canonical-spec edit was exactly the D10 drift fix; no other canonical specs were modified.

## Status

Slice 1 complete (11/11 tasks, incl. verify). Ready for RDD review → PR #1 against main (stacked-to-main, PR #2 later). Next: `apply-slice-2`.

---

# Apply Progress: upp-simplify-usage — Slice 2 (PR #2)

**Change**: upp-simplify-usage · **Phase**: apply · **Slice**: 2 (data quality) · **Date**: 2026-08-17
**Mode**: Strict TDD (RED→GREEN→triangulate executed per task) · **Delivery**: auto-chain, stacked-to-main (PR #2 follows merged PR #1) · **Review budget**: S2 ≈ 175–190 changed lines (measured: 371 insertions / 52 deletions incl. tests).

## Slice 2 Tasks (all complete)

- [x] 2.1 RED `render_test.go`: "1 current"→"1 up to date"; skipped → "N up to date, M skipped"; check-with-skips in `integration_test.go`.
- [x] 2.2 RED `check_test.go`: nvm newer-current v26.7.0→v24.19.0 = false; `stable` unparseable = false; existing nvm pass.
- [x] 2.3 RED config tests: drop `Settings.Interactive` in config_test.go, config_expanded_test.go, integration_test.go; stray key tolerated.
- [x] 2.4 GREEN `render.go` (D4): tagline iff `current>0 && available==0 && skipped==0 && failed==0` OR empty list (preserves parser_test.go:238, integration_test.go:647).
- [x] 2.5 GREEN `nvm.go` (D7): semverCompare reusing selfupdate.Parse/Compare (replace :66); v tolerated; Dev/unparseable → false.
- [x] 2.6 GREEN `config.go` (D8): delete `Settings.Interactive` + default; never re-emitted.
- [x] 2.7 GREEN `scripts/smoke-test.sh` test 13: flip `'interactive = false'` grep → `run_test_without_output "interactive"`.
- [x] 2.8 VERIFY: `go test ./... -count=1 -race`; smoke; zero S1-string overlap.

## TDD Cycle Evidence (Strict TDD)

| Work unit | Task | Test File | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|---|---|---|---|---|---|---|---|---|
| S2-AUX-1 CheckSummary honesty | 2.1, 2.4 | render_test.go (TestCheckSummary_CurrentAndSkipped, _AllSkipped, _EmptyResults, _AvailableAndSkipped, _CurrentAndFailed, updated _UpdatesAvailable) + integration_test.go (TestCheckCommand_WithSkips) | Unit + Integration | ✅ output 20/20, cli 21/21 baseline | ✅ tagline wrongly present + "1 current" wording at RED | ✅ run | ✅ 6 render cases (current+skipped, all-skipped, empty, available+skipped, current+failed, available+current) + 1 integration (hermetic skip adapter) | ✅ gofmt -s clean; branch structure simplified after first GREEN |
| S2-AUX-2 nvm semver | 2.2, 2.5 | check_test.go TestCheck/nvm rows (newer-current-no-downgrade, equal-versions-v-prefix-tolerated, unparseable-stable-no-error, current-dev-no-update) | Unit (adapter, hermetic exec seam) | ✅ official 45/45 baseline | ✅ 4 new rows fail (string-inequality phantom update) | ✅ run (after v-prefix normalization fix: strip→add) | ✅ 4 cases + existing nvm/same-version + nvm/update-available (older-current) keep passing | ✅ semverCompare + normalizeVersion split; gofmt -s clean |
| S2-AUX-3 dead setting | 2.3, 2.6 | config_test.go (dropped assertions, TestLoadStrayInteractiveKey, TestExportNeverReEmitsInteractive), config_expanded_test.go, integration_test.go | Unit + Integration | ✅ config 46/46, cli baseline above | ✅ TestExportNeverReEmitsInteractive fails (export still emits `interactive = true`) | ✅ run after field+default deletion | ✅ stray-load tolerated (BurntSushi unknown-key) + export-never-re-emits + drop-all-assertions | ✅ gofmt -s clean |
| S2-AUX-4 smoke | 2.7 | scripts/smoke-test.sh test 13 | E2E | ✅ 23/23 baseline | ✅ assertion flipped (RED semantics: `'interactive = false'` grep would now fail as stale expectation) | ✅ 23/23 after flip | ➖ Single scenario | n/a (bash) |

## Work Unit Evidence (Hard Gate)

| Work unit | Focused test | Result | Runtime harness | Rollback boundary |
|---|---|---|---|---|
| S2-AUX-1 | `go test -count=1 ./internal/output -run TestCheckSummary_` (7 cases) + `go test -count=1 ./internal/cli -run TestCheckCommand_WithSkips` | ok (7/7, 1/1) | smoke test 4/6 (`upp check`, `upp check --quiet`) pass; live `upp check` semantics unchanged for all-current | revert render.go CheckSummary + render_test.go cases + TestCheckCommand_WithSkips (integration_test.go) |
| S2-AUX-2 | `go test -count=1 ./internal/adapters/official -run 'TestCheck/nvm'` (9 subtests incl. 4 new) | ok | smoke `upp check --only npm` unaffected; live nvm phantom `v26.7.0 → v24.19.0` update now correctly false | revert nvm.go semverCompare/normalizeVersion + 4 check_test.go rows |
| S2-AUX-3 | `go test -count=1 ./internal/config` (50 tests) + `go test -count=1 ./internal/cli -run 'TestExportImport_RoundTrip\|TestComplexConfigRoundTrip'` | ok | smoke test 12/13 (`upp export` with stray `interactive` key: key absent from output) | revert config.go field+default + the 3 test files' assertion removals |
| S2-AUX-4 | `bash scripts/smoke-test.sh` | 23 passed, 0 failed | full smoke run (build + all 23 tests) | revert scripts/smoke-test.sh test 13 block |

## Verification (task 2.8)

- `gofmt -s -l .` → clean (all touched Go files formatted with `-s`).
- `go vet ./...` → clean.
- `go build ./...` → ok.
- `go test ./... -count=1 -race` → all 9 packages ok (adapters, adapters/official, cli, config, output, platform, security, selfupdate; cmd/upp no test files).
- `bash scripts/smoke-test.sh` (with build) → **23 passed, 0 failed, 23 total**.
- Test counts: output 35 top-level PASS (7 CheckSummary), official TestCheck 53 subtests (9 nvm), config 50, cli 91.
- Zero S1-string overlap: check.go:88 `"Checking"` unchanged; `"would update"` (UpdateSummary) unchanged; ListTools header `ID | Name | Status | Version` unchanged; S1 files diff empty (check.go/update.go/list.go/parser.go/README.md/help_test.go/update_test.go/parser_test.go/canonical ux-patterns spec). Only S2 strings added: "N up to date", "N up to date, M skipped", "Nothing to do.".

## Deviations from Design

1. **`v`-prefix normalization direction** (D7): selfupdate.Parse REQUIRES the `v` prefix (fails closed on its absence), so "tolerate v prefix" is implemented by *adding* `v` when missing (`normalizeVersion`), not stripping it. Stripping produced a RED on `nvm/update-available` (v20.11.0 vs v22.0.0 both parsed after strip, but `v20.11.0` without prefix would fail parse for bare latest). Normalization is a tiny local helper in nvm.go; no change to selfupdate.
2. **Detail-list branch structure**: initial CheckSummary returned early in the only-skipped branch, skipping the non-quiet detail listing; restructured so the detail loop runs in every non-tagline branch (test-enforced).
3. **Icon on "N up to date, M skipped"**: uses the current icon (✔️) rather than skipped (⏭️) since tools ARE up to date; no test pins the icon.

## Issues Found / Leftover Notes for Review/Archive

- `internal/security/confirm.go` and audit test comments still use the word "Interactive" — that is the security risk-matrix concept (risk-level naming), NOT the dead config field. No change needed; archive merge of the config-system delta is unaffected.
- `upp check` with a skipped tool now prints `[current] 1 up to date, 1 skipped` in plain mode with a trailing detail line for the skipped tool — matches D4/spec "Check with skips" scenario.
- The `Additional Commands:` help note from S1 remains out of scope (no spec requirement).
- Archive (task 3.1) still pending: merge 4 deltas into canonical specs; ux merge idempotent (1.10).

## Status

Slice 2 complete (8/8 tasks, incl. verify). Ready for RDD review → PR #2 (stacked-to-main on top of merged PR #1). Next: `sdd-verify` / `review` for this slice, then archive (3.1).