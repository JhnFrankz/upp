# Apply Progress: upp-unified-update-flow

**Delivery**: Feature Branch Chain — child PRs target the immediate previous PR branch; only tracker `feature/unified-update-flow` merges to main.
**Mode**: Strict TDD (test runner: `go test ./... -count=1`)
**Attempt authority**: WU1 acq-wu1-20260821-000711 (proceed); WU2 acq-wu2-20260821-003247 (proceed, token sha256:f80520cf…8751); WU3 acq-wu3-20260821-004728 (proceed, token sha256:e35af1f…c12); WU5 acq-wu5-20260821-090458 (token sha256:c83c5206…fd21), max 3 attempts / 550–650 raw-line budget each.

---

## Work Unit 1 — Check engine relocation + onResult seam (COMPLETE)

**Branch**: `wu1/checkrun-seam` (child of tracker, based off `main` @ cfadbb4) · PR #97 → base `feature/unified-update-flow`

### Completed Tasks

- [x] **1.1** Created `internal/cli/checkrun.go`: moved `runChecks`, `safeCheck`, `checkOutcome`, `checkJob`, `calculateWorkerCount`, `defaultConcurrency` verbatim from `check.go`; new signature `runChecks(adapters []adapters.Adapter, onResult func(index int, oc checkOutcome)) []checkOutcome` (nil = silent). Stripped originals + the inline `ProgressInPlace` counter call from `check.go` (the `Renderer.ProgressInPlace` method itself remains in `internal/output/render.go`; its removal is not in any task). Callers updated: `check.go runCheck` → nil, `update.go runUpdateInteractive` → nil (interim until Unit 3 wires the board).
- [x] **1.2** Moved mechanism tests to `internal/cli/checkrun_test.go`: `TestCalculateWorkerCount_Clamping`, `TestSafeCheck_PanicRecovery`, `TestSafeCheck_TimeoutIsolation` verbatim; `TestRunChecks_CarriesUpdateInfo` rewritten to capture via callback; NEW `TestRunChecks_ReportsViaCallback` (per-index exactly-once reporting, callback outcome == returned slot outcome) and `TestRunChecks_NilCallbackSilent` (nil contract). Fakes `fakePanickingAdapter`/`fakeDelayedAdapter` moved with them. Command-level tests (`TestRunCheck_*`) remain in `check_test.go`.

### TDD Cycle Evidence (WU1)

| Task | Test File | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|------|-----------|-------|------------|-----|-------|-------------|----------|
| 1.1 | `internal/cli/checkrun_test.go` | Unit | ✅ cli pkg ok pre-change | ✅ Build fail: `not enough arguments in call to runChecks` (seam API absent) | ✅ `-run Checks -count=5` ok | ✅ 3 seam cases (report/nil/carries) | ➖ Verbatim move, none needed |
| 1.2 | `internal/cli/checkrun_test.go` | Unit | ✅ cli pkg ok pre-change | ✅ Same build fail (tests reference new signature) | ✅ Passed | ✅ Clamp table (11 cases) + panic×2 + timeout carried over | ➖ None needed |

### Work Unit Evidence (WU1)

| Evidence | Value |
|---|---|
| Focused command + result | `go test ./internal/cli/ -run Checks -count=1` → `ok github.com/JhnFrankz/upp/internal/cli` (also run `-count=5` → ok) |
| Runtime harness + result | `go test ./internal/cli/ ./internal/output/ -race -count=1` → both `ok` |
| Rollback boundary | Revert commit 8d1cb2a: restores old `runChecks(adapters, renderer, quiet, showProgress)` signature, counter UX, and original test placement atomically |

### Changed Lines (WU1)

Raw git churn: 451 insertions + 358 deletions = **809 lines** (≈660 verbatim relocations, ≈150 authored).

---

## Work Unit 2 — CheckBoard + Renderer.Color() (COMPLETE)

**Branch**: `wu2/checkboard` (based off `wu1/checkrun-seam` @ 8f11fe9; PR targets base `wu1/checkrun-seam`, NOT tracker/main)
**Scope guard**: board NOT wired into update.go here — that is task 2.1 (WU3).

### Completed Tasks

- [x] **1.3** RED `internal/output/checkboard_test.go`: 10 tests — canonical-order pending lines on `Start`; per-status flips Available (`✓ name cur → new`)/Current (`✓ name up-to-date`)/Skipped (`✓ name not installed`)/Failed (`✗ name: err`) on `Complete`; only-target-line isolation; out-of-order completion never reorders rows; idempotent `Finish`; non-color fallback (Start emits nothing, one plain line per completion, zero ANSI bytes); 8-goroutine concurrent `Complete` across all four statuses under `-race`; out-of-range index defense. Assertions run through a minimal VT100-subset terminal simulator (`\r`, `\n`, `\x1b[<n>A/B`, `\x1b[K`) asserting the settled visible frame, not raw escape bytes.
- [x] **1.4** Implemented `internal/output/checkboard.go`: `NewCheckBoard(w io.Writer, color bool, tools []string) *CheckBoard` + `Start/Complete/Finish`. Per-line state machine stores rendered row text; color mode rewrites one row via cursor-up + clear-to-end + cursor-down return (never full-board redraw, D4); private `sync.Mutex` serializes every write (D1); fallback prints one plain line per completion (D5). Added exported `Renderer.Color()` getter in `render.go` so the board reuses the renderer's single TTY-detection source (D5).

### TDD Cycle Evidence (WU2)

| Task | Test File | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|------|-----------|-------|------------|-----|-------|-------------|----------|
| 1.3 | `internal/output/checkboard_test.go` | Unit | ✅ output+cli pkgs ok pre-change (baseline captured before edits) | ✅ Build fail: `undefined: NewCheckBoard` ×10 | ✅ 10/10 pass after fix of test-side stray `buf.Reset()` | ✅ 10 cases incl. concurrency + fallback + defense | ✅ gofmt/vet clean after signature fix |
| 1.4 | `internal/output/checkboard_test.go` | Unit | same baseline | ✅ same build fail (production code absent) | ✅ `go test ./internal/output/ -run CheckBoard -count=1` → ok | covered by 1.3 matrix | ➖ helpers extracted (`boardMarkerLine`), tests still green |

### Work Unit Evidence (WU2)

| Evidence | Value |
|---|---|
| Focused command + result | `go test ./internal/output/ -run CheckBoard -count=1` → `ok github.com/JhnFrankz/upp/internal/output` (10 tests) |
| Runtime harness + result | `go test ./internal/output/ ./internal/cli/ -race -count=1` → both `ok` (concurrent `Complete` serialization is the runtime boundary; no TTY scenario in this unit — wiring arrives in WU3) |
| Rollback boundary | Delete `internal/output/checkboard.go` + `internal/output/checkboard_test.go` and revert the 5-line `Renderer.Color()` addition in `render.go` (commit 13560a7); touches nothing else — no caller exists yet by design |

### Full Gate Results (WU2)

- `go build ./...` → clean
- `go vet ./...` → clean
- `gofmt -s -l .` → no output (clean)
- `go test ./... -count=1` → all 9 packages ok

### Files Changed (WU2)

| File | Action | What Was Done |
|------|--------|---------------|
| `internal/output/checkboard.go` | Created | `CheckBoard` component: per-line state machine, mutex, cursor-up+clear redraw, color fallback |
| `internal/output/checkboard_test.go` | Created | 10 behavioral tests + VT100-subset terminal simulator harness |
| `internal/output/render.go` | Modified | Added exported `Color()` getter (+5 lines) |

### Changed Lines (WU2)

Raw git churn: **519 insertions, 0 deletions** (commit 13560a7) — within the 550 raw-line attempt budget. ≈100 lines of that are the terminal-simulator test harness.

---

## Work Unit 3 — Board wiring + up-to-date summary (COMPLETE)

**Branch**: `wu3/wire-board` (based off `wu2/checkboard` @ 1195e06; PR #99 → base `wu2/checkboard`, NOT tracker/main)
**Attempt**: acq-wu3-20260821-004728 (token sha256:e35af1f…c12), 550 raw-line budget — used 252. Attempt resumed after an interrupted mid-TDD-cycle return; commit 6877372 was already on disk and was re-verified, not redone.

### Completed Tasks

- [x] **2.1** Commit 6877372: `runUpdateInteractive` builds `output.NewCheckBoard(os.Stdout, r.Color(), names)` in canonical filtered order; `Start()` before the `runChecks` pool; `board.Complete` as `onResult`; `Finish()` before the pending-only (`StatusAvailable`) CheckboxSelector. The interim "engine must be silent" locks became positive board assertions (exactly one plain fallback line per filtered tool) plus never-return "Checking X/Y" guards in `TestRunUpdate_InteractiveSelection` and `TestRunUpdate_SelectorCancel`. Wiring intent re-verified against design D4/D5/D9 at resume before continuing.
- [x] **2.2** RED→GREEN: `UpdateSummary` now counts `StatusCurrent` (previously counted nowhere — all-current and current+skipped runs wrongly hit "All tools not installed. Nothing to do."); new "N up to date" part inserted between updated/would-update and skipped (mirrors `CheckSummary` part order); all-skipped special branch requires `current == 0`; `detailSummary` lists current tools under "Up to date:". Tests: `TestUpdateSummary_AllCurrentDryRun`, `TestUpdateSummary_CurrentWithSkipsDryRun` ("8 up to date, 2 skipped", never "All tools up to date."), `TestUpdateSummary_UpdatedAndCurrent`.
- [x] **2.3** RED→GREEN: CLI-level pins `TestRunUpdate_DryRunCurrentWithSkips` (8 current + 2 not-installed via dry-run → "8 up to date, 2 skipped"; never "All tools up to date." / "All clean!" / "not installed") and `TestRunUpdate_DryRunPendingNeverClean` ("would update", never "All clean!"); `TestRunUpdate_NoPendingSkipsSelector` re-pinned from the old wrong all-skipped summary to "1 up to date, 1 skipped" (D6 correction, documented in PR body).

### TDD Cycle Evidence (WU3)

| Task | Test File | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|------|-----------|-------|------------|-----|-------|-------------|----------|
| 2.1 | `internal/cli/update_test.go` | Unit | ✅ cli pkg ok pre-change (baseline re-verified at resume) | ✅ committed in 6877372 (interim locks fail against wiring absence) | ✅ `-run Update -count=1` ok | board line count==1 per tool + never-"Checking" guards | ➖ none needed |
| 2.2 | `internal/output/render_test.go` | Unit | ✅ output+cli pkgs ok pre-change | ✅ 3 new tests fail: "All tools not installed. Nothing to do." on an 8-current run; no up-to-date part/listing | ✅ `-run TestUpdateSummary` ok | all-current / 8+2 skips / mixed updated+current | ➖ mirrors CheckSummary conventions |
| 2.3 | `internal/cli/update_test.go` | Unit | ✅ same baseline | ✅ both dry-run tests fail on the "not installed" claim | ✅ `-run TestRunUpdate` ok after render fix + NoPendingSkipsSelector re-pin | pending-only never-clean guard added | ➖ `dryRunFakes` helper extracted |

### Work Unit Evidence (WU3)

| Evidence | Value |
|---|---|
| Focused command + result | `go test ./internal/cli/ -run Update -count=1` → ok; `go test ./internal/output/ -run TestUpdateSummary -count=1` → ok |
| Runtime harness + result | `go test ./internal/output/ ./internal/cli/ -race -count=1` → both ok (board mutex serialization under the concurrent pool is the runtime boundary; TTY visual path stays covered by WU2 simulator tests) |
| Rollback boundary | Revert 6877372 + 5fe1eff + a6d8ac2: unwinds wiring, summary counting, and their tests atomically; nothing downstream depends on them yet |

### Full Gate Results (WU3)

- `go build ./...` → clean · `go vet ./...` → clean · `gofmt -s -l .` → clean
- `go test ./... -count=1` → all 9 packages ok
- PR #99 opened → base `wu2/checkboard`

### Files Changed (WU3)

| File | Action | What Was Done |
|------|--------|---------------|
| `internal/cli/update.go` | Modified | Board wiring in `runUpdateInteractive` (6877372) |
| `internal/output/render.go` | Modified | Up-to-date count part + Current listing + all-skipped branch guard (5fe1eff) |
| `internal/output/render_test.go` | Modified | 3 D6 summary tests (+86) |
| `internal/cli/update_test.go` | Modified | Board assertions (6877372), dry-run pins + NoPendingSkipsSelector re-pin, `dryRunFakes` helper (+95/−2 net across commits) |

### Changed Lines (WU3)

Raw git churn: 6877372 (+45/−10) + 5fe1eff (+98/−2) + a6d8ac2 (+95/−2) = **+238/−14 = 252 raw lines** — within the 550 attempt budget.

---

## Work Unit 4 — Removals: `upp check`, hint, `check_self_update` (COMPLETE)

**Branch**: `wu4/removals` (based off `wu3/wire-board` @ a6d8ac2; PR → base `wu3/wire-board`, NOT tracker/main)
**Attempt**: acq-wu4-20260821-084011 (token sha256:d8a69a0c…81ce), stated budget 650 raw lines — actual raw churn exceeded it (see Changed Lines); all changes map 1:1 to tasks 3.1–3.4, no scope creep.

### Completed Tasks

- [x] **3.1** Commit 9aa957a: deleted `internal/cli/check.go` (`NewCheckCommand`, `runCheck`, `checkDeps`, `maybeShowSelfUpdateHint`, `selfUpdateCacheFile`); relocated `buildAdapterList`/`adapterIDs`/`adapterByID` verbatim into `checkrun.go` (still shared by update.go/list.go — design D3 lists only command/runCheck/checkDeps/hint for deletion); dropped `check` registration in `parser.go` (grouping comment now Commands=list/update) and the `cliDeps.check` slot in `deps.go`; `setCLIDeps` lost its checkDeps parameter. `checkrun.go` untouched except imports + the relocated helpers.
- [x] **3.2** Deleted `Renderer.SelfUpdateHint` (render.go) and its two render tests; deleted `Settings.CheckSelfUpdate` + its default — `Settings` is now an intentionally empty struct documented as forward-compatible (unknown keys ignored on load, never rewritten by Save).
- [x] **3.3** Deleted `check_hint_test.go` (8 hint tests; `writeCheckConfig` helper moved into parser_test.go — still used there), `TestCheckProgress_LabelsChecking`, and the runCheck-level `check_test.go`. Fixed expectations: help_test.go wanted-list, parser_test.go `expectedCommands`, integration_test.go subcommand count 5→4 / help list / registration map. Ported onto `upp update --dry-run`: NoConfig, QuietMode_SuppressesProgress, EmptyConfig_AllToolsSkips ("All tools not installed. Nothing to do."), Init→DryRun lifecycle, WithSkips ("1 up to date, 1 skipped"), SummaryOutput, DeterministicOrderUnderConcurrency ("3 would update"/"2 up to date", alpha<gamma<epsilon). New `TestUpdateDryRun_MixedStatusCounts` ports the mixed-status counts coverage (1 would update / 2 up to date / 2 failed, zero Update calls) — the panicking adapter from the old test is replaced by a second check-error adapter because the sequential dry-run path has no safeCheck containment (pre-existing behavior, unchanged).
- [x] **3.4** RED-first `TestLoadStrayCheckSelfUpdateKey_NeverRewritten`: config with `check_self_update = true` loads without error and `Save` never re-emits the key (failed pre-change because Save re-encoded the struct field). TestLoadValidTOML keeps the key in its TOML as a live tolerance case; TestLoadCheckSelfUpdate and all field references deleted.

### TDD Cycle Evidence (WU4)

| Task | Test File | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|------|-----------|-------|------------|-----|-------|-------------|----------|
| 3.1 | `internal/cli/parser_test.go` | Integration | ✅ cli/config/output ok @ a6d8ac2 | ✅ `TestUnknownCommand_Check` fails: check still executes | ✅ unknown-command error, exit 1 | ➖ mirrors Export/Import pins (single possible outcome) | ➖ none needed |
| 3.2 | `internal/output/render_test.go` | Unit | ✅ same baseline | ✅ N/A — pure deletion; compile failure of hint callers is the forcing function | ✅ output pkg ok after method+tests removed | ➖ deletion, no new behavior | ➖ none needed |
| 3.3 | `internal/cli/integration_test.go` | Integration | ✅ same baseline | ✅ ports fail against old surface (wrong command/summary text) until removal lands | ✅ `-run 'TestUpdateDryRun|TestQuietMode|TestInitUpdate' -count=1` ok | ✅ counts/order/skips/quiet/mixed-status variants | ✅ stale checkDeps comments cleaned, suite green |
| 3.4 | `internal/config/config_test.go` | Unit | ✅ same baseline | ✅ `TestLoadStrayCheckSelfUpdateKey_NeverRewritten` fails: Save rewrote the key | ✅ config pkg ok after field removal | ✅ stray-key load + never-rewritten round-trip | ➖ none needed |

### Work Unit Evidence (WU4)

| Evidence | Value |
|---|---|
| Focused command + result | `go test ./internal/cli/ ./internal/config/ ./internal/output/ -count=1` → all 3 ok |
| Runtime harness + result | Built `/tmp/opencode/upp-wu4`: `upp check` → `unknown command "check"`, exit 1; `upp update --dry-run` → exit 0. Plus `go test ./internal/{cli,config,output} -race -count=1` → ok |
| Rollback boundary | Revert 9aa957a: restores check.go, both deleted test files, SelfUpdateHint, CheckSelfUpdate, and the pre-port test expectations atomically; nothing outside internal/ touched |

### Full Gate Results (WU4)

- `go build ./...` clean · `go vet ./...` clean · `gofmt -s -l .` clean
- `go test ./... -count=1` → all 9 packages ok

### Files Changed (WU4)

| File | Action | What Was Done |
|------|--------|---------------|
| `internal/cli/check.go` | Deleted | Command, runCheck, checkDeps, hint logic (shared helpers relocated first) |
| `internal/cli/check_hint_test.go` | Deleted | 8 hint tests; writeCheckConfig helper preserved in parser_test.go |
| `internal/cli/check_test.go` | Deleted | runCheck-level tests; coverage ported to dry-run |
| `internal/cli/checkrun.go` | Modified | +buildAdapterList/adapterIDs/adapterByID (verbatim) + official/config imports |
| `internal/cli/parser.go` / `deps.go` | Modified | Dropped check registration + cliDeps.check slot |
| `internal/output/render.go` / `render_test.go` | Modified | Deleted SelfUpdateHint + its 2 tests |
| `internal/config/config.go` / `config_test.go` | Modified | Settings emptied; compat table test added; dead setting tests removed |
| `internal/cli/{help,integration,parser}_test.go`, `update.go`, `list.go` | Modified | Expectations fixed, dry-run ports, stale comment refs |

### Changed Lines (WU4)

Raw git churn: commit 9aa957a **+271/−847 = 1118 raw lines** — exceeds the 650-line attempt budget. Cause: mandated deletions alone (check.go 187 + check_hint_test.go 345 + check_test.go 96 = 628) plus required edits/ports leave no smaller atomic slice; deleting runCheck breaks three test files at once, so the unit cannot be split without a red intermediate state. Every line maps to tasks 3.1–3.4.

---

## Work Unit 5 — Spec verification + docs/E2E (COMPLETE)

**Branch**: `wu5/spec-docs-e2e` (based off `wu4/removals` @ 95d2641; PR → base `wu4/removals`, NOT tracker/main)
**Attempt**: acq-wu5-20260821-090458 (token sha256:c83c5206…fd21), 650 raw-line budget — used ≈390 code + ≈100 bookkeeping.
**Maintainer decision (authorized 2026-08-21)**: task 3.5 added and completed — delete orphaned `Renderer.CheckSummary` + its 8 render tests.

### Completed Tasks

- [x] **3.5** Commit f8422b1: deleted `Renderer.CheckSummary` (render.go, ~80 lines) + `TestCheckSummary_*` (8 tests, render_test.go). Zero production callers confirmed via codegraph blast radius after 9aa957a; the update path renders through `UpdateSummary`. Pure deletion; output+cli packages green immediately after gofmt.
- [x] **4.1** RED→GREEN commit 0cbe869: spec ux-patterns "All succeed" demands explicit zero counts — strengthened `TestUpdateSummary_AllUpdated` to `"2 updated, 0 failed. All clean!"` (RED: got "2 updated. All clean!"); GREEN: allClean branch now prints `, 0 failed. All clean!`. Added CLI-level `TestRunUpdate_AllSucceedSummary` end-to-end through `runUpdateSequential` (PolicyAlwaysUpdate fake → "1 updated, 0 failed. All clean!"). Strengthened `TestUpdateSummary_PartialFailure` to exact "1 updated, 1 failed. Review errors above." (approval pin, passed immediately). No-tools + deterministic-order scenarios already pinned (WU3/WU4: `TestUpdateSummary_AllSkipped`, `TestEmptyConfig_AllToolsSkipped`, `TestUpdateDryRun_DeterministicOrderUnderConcurrency`, `TestUpdateDryRun_MixedStatusCounts`).
- [x] **4.2** Commit 9ecd80a: strengthened `TestToolLine_VerboseFailureDiagnostics` with the indented `"    │ <line>"` prefix assertion beneath the failed tool line (approval pin — renderer already indents). Full scenario coverage pre-existing: `-v`/default/`-q -v` via `TestRunUpdate_VerboseFailureDiagnostics` (CLI), `TestToolLine_NonVerboseFailureSuppressed`, `TestToolLine_QuietOverridesVerbose`, `TestUpdateSummary_VerboseFailureDiagnostics`.
- [x] **4.3** Commit 9ecd80a: new `TestSelfUpdate_UnknownFlagRejected` — root Execute with `self-update --yes` errors with cobra's "unknown flag" rejection (non-zero exit proxy; approval pin on cobra default). Already pinned: Short text (parser_test.go:389), `--ci` deny (`TestSelfUpdate_CIDeny`/`CIDenyThroughRoot`), `--only/--skip` ignored (`TestSelfUpdate_OnlySkipIgnored`), `--quiet` keeps prompt (`TestSelfUpdate_QuietKeepsPrompt`).
- [x] **4.4** RED→GREEN commit 5379d62: dashboard quick-reference still advertised removed `upp check`; updated `TestDashboard_Formatting` + `TestRunDashboard_WithConfig` to require `upp update -n` and reject `upp check` (both RED against old render.go); GREEN: Dashboard lines now `upp update -n` ("Preview pending updates (--dry-run)") / `upp update` / `upp list` / `upp --help` per ux-patterns Bare Dashboard requirement. Swept stale `upp check` comment in checkrun.go runChecks doc. `go build ./... && go vet ./...` clean.
- [x] **5.1** Commit 1b2ec3d: smoke-test.sh Tests 2/5/7/8/9 rewritten off the removed command — Test 2 drops `upp check --help`; Test 5 becomes "Read-only query surface" (`upp update --dry-run` header assert + `upp check` pruned-exit-1); Tests 7/8/9 route `-q`/`--quiet`, `-v`/`--verbose`, `--only/--skip/--only+--skip` through `upp update -n`. Harness: `bash scripts/smoke-test.sh --skip-build` → **31 passed, 0 failed, exit 0**.
- [x] **5.2** Commit 27c21ca: README — quick-start block loses the check entry (`upp update -n` labeled read-only query surface), Commands table row removed, opt-in hint bullet deleted (feature removed with check; kills the false bare-`upp` hint claim), `check_self_update = false` line dropped from the config example. Repo sweep: no non-test `upp check` refs remain outside smoke's intentional exit-1 assertion.

### TDD Cycle Evidence (WU5)

| Task | Test File | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|------|-----------|-------|------------|-----|-------|-------------|----------|
| 3.5 | `internal/output/render_test.go` | Unit | ✅ 9 pkgs ok @ 95d2641 | ➖ N/A — pure deletion; test removal is part of the authorized unit | ✅ output+cli ok post-delete | ➖ deletion, no behavior | ➖ none needed |
| 4.1 | `render_test.go` + `update_test.go` | Unit + Integration | ✅ same baseline | ✅ both new/strengthened tests fail ("2 updated. All clean!" lacks "0 failed") | ✅ `-run TestUpdateSummary` + `-run TestRunUpdate` ok | ✅ renderer all-succeed + CLI end-to-end + partial-fail composition | ➖ none needed |
| 4.2 | `render_test.go` | Unit | ✅ same baseline | ➖ N/A — approval pin on existing indenting renderer | ✅ passed first run | ➖ single scenario shape | ➖ none needed |
| 4.3 | `selfupdate_test.go` | Integration | ✅ same baseline | ➖ N/A — approval pin on cobra unknown-flag default | ✅ passed first run | covered by existing ci/quiet/filter pins | ➖ none needed |
| 4.4 | `render_test.go` + `root_test.go` | Unit + Integration | ✅ same baseline | ✅ both dashboard tests fail while render.go prints "upp check" | ✅ after quick-reference rewrite | ✅ positive (update -n) + negative (no check) assertions | ➖ none needed |
| 5.1 | `scripts/smoke-test.sh` | E2E harness | ✅ prior smoke green | ➖ N/A — script harness, not Go TDD | ✅ 31/31 pass, exit 0 | ✅ -q/-v/--only/--skip variants | ➖ none needed |
| 5.2 | N/A (docs) | Docs | ✅ N/A | ➖ N/A | ✅ grep sweep clean | ➖ N/A | ➖ N/A |

### Work Unit Evidence (WU5)

| Evidence | Value |
|---|---|
| Focused command + result | `go test ./internal/output/ ./internal/cli/ -count=1` → both ok; smoke `bash scripts/smoke-test.sh --skip-build` → 31 passed / 0 failed / exit 0 |
| Runtime harness + result | Built `./upp`: smoke suite incl. rewritten dry-run query tests and `upp check` → exit 1. Plus `go test ./internal/{cli,output} -race -count=1` → ok |
| Rollback boundary | Revert f8422b1..27c21ca individually: each commit is one deliverable (deletion / summary counts / pins / dashboard / smoke / docs); none depends on another's partial state |

### Full Gate Results (WU5)

- `go build ./...` clean · `go vet ./...` clean · `gofmt -s -l .` clean
- `go test ./... -count=1` → all 9 packages ok · `-race` cli+output → ok
- PR opened → base `wu4/removals` (feature-branch-chain)

### Files Changed (WU5)

| File | Action | What Was Done |
|------|--------|---------------|
| `internal/output/render.go` | Modified | Deleted CheckSummary; all-clean summary gains ", 0 failed"; dashboard quick-reference → `upp update -n` |
| `internal/output/render_test.go` | Modified | −8 CheckSummary tests; strengthened AllUpdated/PartialFailure/verbose-indent/Dashboard assertions |
| `internal/cli/update_test.go` | Modified | `TestRunUpdate_AllSucceedSummary` end-to-end pin |
| `internal/cli/selfupdate_test.go` | Modified | `TestSelfUpdate_UnknownFlagRejected` pin |
| `internal/cli/root_test.go` / `checkrun.go` | Modified | Dashboard guidance expectations; stale comment sweep |
| `scripts/smoke-test.sh` | Modified | Tests 2/5/7/8/9 onto list/dry-run surfaces + pruned-check exit 1 |
| `README.md` | Modified | Dropped check/hint refs; documented `-n` query surface |

### Changed Lines (WU5)

Raw git churn (code): **+100/−290 = 390 raw lines** across f8422b1…27c21ca; plus SDD bookkeeping ≈100 → within the 650-line attempt budget.

---

## Remaining Tasks

- [ ] None — all 18 tasks complete (17 planned + maintainer-authorized 3.5).

## Status

18/18 tasks complete. WU5 done (PR → `wu4/removals`). Chain ready for verify/archive once child PRs merge.
