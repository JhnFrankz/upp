# Tasks: upp CLI Simplification & Subcommand/Flag Pruning

## Review Workload Forecast

Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: Low per slice (~220 lines / ~280 lines)

| Field | Value |
|-------|-------|
| Estimated changed lines | ~500 total (Slice 1: ~220 lines, Slice 2: ~280 lines) |
| 400-line budget risk | Low per slice; Low/Medium overall |
| Chained PRs recommended | Yes (2 slices: PR 1 → PR 2) |
| Suggested split | 2 stacked PRs: Subcommand Pruning, Shorthands & Help Restructure (PR 1) → Bare Dashboard & Verbose Error Diagnostics (PR 2) |
| Delivery strategy | auto-chain |
| Chain strategy | **stacked-to-main** — 2 independent, atomic slices landing sequentially on main, each with a clear rollback boundary |

### Suggested Work Units

| Unit | Goal | PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|----|----------------------|-----------------|-------------------|
| 1 | Prune `export`/`import` subcommands & config helpers | PR 1 | `go test -v -count=1 ./internal/config/... ./internal/cli/...` | N/A — hermetic unit tests | Revert deleted files (`export.go`, `import.go`, `config/export.go`) |
| 2 | Add `-q`, `-n`, and `-v` flag shorthands | PR 1 | `go test -v -count=1 -run "TestFlag\|TestParse\|TestGlobalFlags" ./internal/cli/...` | N/A — hermetic flag tests | Revert `parser.go`, `update.go` |
| 3 | Restructure root `--help` into `Commands` & `Maintenance` | PR 1 | `go test -v -count=1 -run "TestHelp_" ./internal/cli/...` | N/A — hermetic help tests | Revert `parser.go`, `help_test.go` |
| 4 | Bare `upp` dashboard & welcome screen | PR 2 | `go test -v -count=1 -run "TestDashboard\|TestRoot" ./internal/output/... ./internal/cli/...` | N/A — hermetic seam tests | Revert `root.go`, `render.go`, `deps.go` |
| 5 | Verbose adapter stderr diagnostics on failure | PR 2 | `go test -v -count=1 -run "TestVerbose\|TestToolLine\|TestCheck\|TestUpdate" ./internal/output/... ./internal/cli/...` | N/A — hermetic mock adapters | Revert verbose plumbing in `check.go`, `update.go`, `render.go` |
| 6 | Integration tests, smoke tests, README & quality gates | PR 2 | `go test -v -count=1 ./... && ./scripts/smoke-test.sh` | `./scripts/smoke-test.sh` | Revert `integration_test.go`, `smoke-test.sh`, `README.md` |

---

## Slice 1: Subcommand Pruning, Shorthands & Help Restructure (PR #1) (~220 lines)

### Phase 1.1: Prune `upp export` and `upp import` Subcommands & Helpers (TDD)

- [x] 1.1.1 (RED) Update `internal/cli/parser_test.go` and `internal/cli/help_test.go` to assert that `upp export` and `upp import` are rejected as unknown commands (exit code 1) and do not appear in help output. Remove obsolete export/import test cases from `internal/config/config_test.go`, `internal/config/config_expanded_test.go`, and `internal/cli/integration_test.go`. Verify RED (test failure or compilation error on legacy calls).
- [x] 1.1.2 (GREEN) Physically delete `internal/cli/export.go`, `internal/cli/import.go`, and `internal/config/export.go`. In `internal/cli/parser.go`, remove `ExportFlags`, `NewExportCommand`, `NewImportCommand`, and their registrations from `AddCommands`.
- [x] 1.1.3 (Verify) Run `go test -v -count=1 ./internal/config/... ./internal/cli/...` to verify all config operations and pruned command rejections pass cleanly.

### Phase 1.2: Add `-q`, `-n`, and `-v` Flag Shorthands (TDD)

- [x] 1.2.1 (RED) Add unit tests in `internal/cli/parser_test.go` and `internal/cli/update_test.go` verifying:
  - `-q` sets `GlobalFlags.Quiet = true` identically to `--quiet`.
  - `-v` sets `GlobalFlags.Verbose = true` identically to `--verbose`.
  - `-n` sets `UpdateFlags.DryRun = true` identically to `--dry-run` on `upp update`.
  - Verify RED: compilation failure due to missing `Verbose` field on `GlobalFlags` or test failure due to missing flag shorthands.
- [x] 1.2.2 (GREEN) Update `GlobalFlags` in `internal/cli/parser.go` to add `Verbose bool`. Bind `-q` (`--quiet`) and `-v` (`--verbose`) in `BuildRoot()` using `BoolVarP`. Bind `-n` (`--dry-run`) in `internal/cli/update.go` using `BoolVarP`.
- [x] 1.2.3 (Verify) Run `go test -v -count=1 -run "TestFlag|TestParse|TestGlobalFlags" ./internal/cli/...` to verify all flag shorthand test cases pass.

### Phase 1.3: Restructure Help Output into Commands and Maintenance Groups (TDD)

- [x] 1.3.1 (RED) Update `internal/cli/help_test.go` to assert:
  - Root help output displays exactly two groups: `Commands` (`check`, `list`, `update`) and `Maintenance` (`init`, `self-update`).
  - The legacy `Config Commands` group is completely absent.
  - The built-in `completion` command remains hidden.
  - Verify RED: test assertion fails against the legacy 3-group layout.
- [x] 1.3.2 (GREEN) Update `AddCommands()` in `internal/cli/parser.go`:
  - Define groups `&cobra.Group{ID: "commands", Title: "Commands"}` and `&cobra.Group{ID: "maintenance", Title: "Maintenance"}`.
  - Assign `check.GroupID = "commands"`, `update.GroupID = "commands"`, `list.GroupID = "commands"`.
  - Assign `init.GroupID = "maintenance"`, `selfUpdate.GroupID = "maintenance"`.
- [x] 1.3.3 (Verify & Slice 1 Gate) Run `go test -v -count=1 ./internal/cli/...`, verify formatting with `test -z "$(gofmt -s -l .)"`, and run `go vet ./...`. Verify net line diff is within ~220 lines.

---

## Slice 2: Bare Dashboard & Verbose Error Diagnostics (PR #2) (~280 lines)

### Phase 2.1: Implement Bare Dashboard Welcome Screen (TDD)

- [x] 2.1.1 (RED) Add unit tests for dashboard presentation and execution:
  - In `internal/output/render_test.go`: test `Renderer.Dashboard` and `Renderer.DashboardNoConfig` (validates banner, platform string, tools count formatting, quickstart command list, `--quiet` suppression, and pipe/non-TTY plain mode without ANSI codes).
  - In `internal/cli/root_test.go`: test `runDashboard` using mocked `dashboardDeps` (validates missing config prompts `upp init`, valid config shows enabled vs available tool count, quiet flag suppresses banner, strictly read-only with no network calls).
  - Verify RED: compilation failure on missing methods/types or assertion failures.
- [x] 2.1.2 (GREEN) Implement dashboard logic:
  - In `internal/output/render.go`: define `DashboardData` struct; implement `Dashboard(data DashboardData)` and `DashboardNoConfig(version, platform string)` on `Renderer`.
  - Create `internal/cli/root.go`: define `dashboardDeps` struct and implement `runDashboard(gf *GlobalFlags, version string, w io.Writer, deps dashboardDeps) error`.
  - In `internal/cli/deps.go`: add `dashboard dashboardDeps` to `cliDeps`.
  - In `internal/cli/parser.go` / `internal/cli/root.go`: update `BuildRoot().RunE` to invoke `runDashboard(gf, cmd.Root().Version, os.Stdout, cliDeps.dashboard)`.
- [x] 2.1.3 (Verify) Run `go test -v -count=1 ./internal/output/... ./internal/cli/... -run "TestDashboard|TestRoot"` to verify all dashboard unit and hermetic tests pass.

### Phase 2.2: Implement Verbose Subprocess Stderr Diagnostics on Adapter Failure (TDD)

- [x] 2.2.1 (RED) Add tests for verbose failure diagnostics:
  - In `internal/output/render_test.go`: test that when `verbose` is true and status is `StatusFailed`, `Renderer` renders indented subprocess stderr lines beneath the failed tool entry; test that stderr is suppressed when `verbose` is false or when `quiet` is true (quiet precedence).
  - In `internal/cli/check_test.go` and `internal/cli/update_test.go`: test with failing mock adapters to verify captured stderr is stored in `ToolResult.Stderr` and rendered when `-v` is active.
  - Verify RED: compilation failure or missing stderr lines in test output.
- [x] 2.2.2 (GREEN) Implement verbose diagnostic rendering and pipeline capture:
  - In `internal/output/render.go`: add `verbose bool` field to `Renderer`, add `NewRendererVerbose` constructor, update `ToolResult` struct to include `Stderr string`, and update `verboseToolLine` / `detailSummary` to print indented stderr lines when `verbose && !quiet && result.Stderr != ""`.
  - In `internal/cli/check.go` and `internal/cli/update.go`: instantiate renderer with `gf.Verbose`, capture adapter error/stderr output into `ToolResult.Stderr` on failure.
- [x] 2.2.3 (Verify) Run `go test -v -count=1 ./internal/output/... ./internal/cli/... -run "TestVerbose|TestToolLine|TestCheck|TestUpdate"` to verify verbose error diagnostics behavior.

### Phase 2.3: Integration Tests, Smoke Tests, Documentation & Quality Gates

- [x] 2.3.1 (Integration Tests) Update `internal/cli/integration_test.go` to test bare `upp` dashboard execution, `-q`, `-n`, `-v` flag interactions, 2-group help, and unknown command rejection for pruned `export`/`import`.
- [x] 2.3.2 (Smoke Tests) Update `scripts/smoke-test.sh` to remove legacy `export`/`import` checks, add tests for bare `upp` dashboard, test `-q`, `-n`, and `-v` flags, and verify unknown command exit 1 for `export`/`import`.
- [x] 2.3.3 (Documentation) Update `README.md` to reflect the simplified CLI command catalog (`check`, `list`, `update`, `init`, `self-update`), 2-group help structure, flag shorthands (`-q`, `-n`, `-v`), bare `upp` dashboard welcome screen, and standard dotfiles config management.
- [x] 2.3.4 (Final Quality Gate Verification)
  - Run full test suite: `go test -v -count=1 ./...`
  - Run race detector: `go test -race -count=1 ./...`
  - Verify formatting and vet: `test -z "$(gofmt -s -l .)" && go vet ./...`
  - Run end-to-end smoke test script: `./scripts/smoke-test.sh`
