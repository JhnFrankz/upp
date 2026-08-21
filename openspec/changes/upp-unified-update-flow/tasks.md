# Tasks: Unified Update Flow

## Review Workload Forecast

|Field|Value|
|---|---|
|Estimated changed lines|900–1200|
|400-line budget risk|High|
|Chained PRs recommended|Yes|
|Suggested split|PR1 seam → PR2 board → PR3 wiring → PR4 removals → PR5 docs/E2E|
|Delivery strategy|ask-on-risk|
|Chain strategy|pending|

Decision needed before apply: Yes
Chained PRs recommended: Yes
Chain strategy: feature-branch-chain
400-line budget risk: High

### Suggested Work Units

|Unit|Goal|Likely PR|Focused test command|Runtime harness|Rollback boundary|
|---|---|---|---|---|---|
|1|Relocate check engine + `onResult` seam|PR 1|`go test ./internal/cli/ -run Checks`|`-race` package tests|Revert unit files|
|2|`CheckBoard` + `Renderer.Color()`|PR 2|`go test ./internal/output/ -run CheckBoard`|`-race` package tests|Delete new files|
|3|Wire board into update pre-check; up-to-date summary count|PR 3|`go test ./internal/cli/ -run Update`|TTY `upp update --dry-run`|Revert wiring edits|
|4|Delete `upp check`, hint, `check_self_update`|PR 4|`go test ./internal/cli/ ./internal/config/`|`upp check` exits 1|Revert restores command+hint|
|5|smoke-test.sh + README refresh|PR 5|`bash scripts/smoke-test.sh`|Local smoke run|Revert script/README|

## Phase 1: Foundation

- [x] 1.1 Create `internal/cli/checkrun.go`: move check engine (`runChecks`, `safeCheck`, `checkOutcome`, `checkJob`, worker/concurrency consts) verbatim from `check.go`; add `onResult(index, checkOutcome)` callback (nil = silent); strip originals + `ProgressInPlace`; `update.go` passes nil interim.
- [x] 1.2 Move mechanism tests (worker clamp, `safeCheck` panic isolation) to `internal/cli/checkrun_test.go`, asserting via callback capture.
- [ ] 1.3 RED `internal/output/checkboard_test.go`: canonical-order pending lines (`Start`), per-status flips Available/Current/Skipped/Failed (`Complete`), idempotent `Finish`, non-color fallback, concurrent `Complete` under `-race`.
- [ ] 1.4 Implement `internal/output/checkboard.go`: `NewCheckBoard(w, color, tools)` + `Start/Complete/Finish`, per-line state machine, private mutex, cursor-up+clear redraw; add exported `Renderer.Color()` getter in `render.go`.

## Phase 2: Update-flow wiring

- [ ] 2.1 `update.go` `runUpdateInteractive`: build board (canonical filtered order); `Start()` before pool, `board.Complete` as `onResult`, `Finish()` before pending-only (`StatusAvailable`) CheckboxSelector.
- [ ] 2.2 `render.go`: `UpdateSummary`/`detailSummary` gain "N up to date" count + Current listing (D6).
- [ ] 2.3 `update_test.go`: replace "Checking X/Y" (:577/:680) with board assertions; dry-run 8 current+2 skipped → "8 up to date, 2 skipped", never "All tools up to date."; pending never "All clean!".

## Phase 3: Removals

- [ ] 3.1 Delete rest of `internal/cli/check.go` (command, `runCheck`, `checkDeps`, `maybeShowSelfUpdateHint`); drop registration in `parser.go` and `check` slot in `deps.go`.
- [ ] 3.2 Delete `SelfUpdateHint` (`render.go` :571); delete `Settings.CheckSelfUpdate` + default (`internal/config/config.go` :29).
- [ ] 3.3 Delete `check_hint_test.go`, `TestCheckProgress_LabelsChecking`; fix `help_test.go` :50, `parser_test.go` :265; port 8 `SetArgs({"check"})` integrations onto `upp update --dry-run`.
- [ ] 3.4 Extend config unknown-key tables (`config_test.go` :266/:347): `check_self_update = true` loads silently ignored, `Save` never rewrites.

## Phase 4: Spec verification

- [ ] 4.1 Summary Report scenarios: all-succeed/partial-fail/no-tools counts; deterministic canonical order via `upp update --dry-run`.
- [ ] 4.2 Verbose diagnostics: `-v` indented stderr beneath failed tool; default concise only; `-q -v` suppressed.
- [ ] 4.3 Self-update flags: unknown flag rejected non-zero; `--ci` denies; `--only/--skip` ignored; `--quiet` still prompts; Short "Update the upp binary itself".
- [ ] 4.4 Sweep leftover `upp check` refs (dashboard quick-reference → `upp update -n`); `go build ./... && go vet ./...`.

## Phase 5: Docs & E2E

- [ ] 5.1 Rewrite `scripts/smoke-test.sh` Tests 2/5/7/8/9 onto `list` / `upp update --dry-run` with `-q/-v/--only/--skip`; assert `upp check` exits 1.
- [ ] 5.2 Update `README.md` :65/:91/:109/:144: drop check + hint, document `-n` query surface, remove false bare-`upp` claim.
