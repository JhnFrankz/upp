# Tasks: upp-application-engine — Application Engine Layer Extraction

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ≈1,150 (additions ≈1,000, deletions ≈150, net ≈+850) |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | PR 1 Core Engine Models & Lifecycle → PR 2 Adapter Discovery & Manager Ownership → PR 3 Bounded Concurrent Check & Cancellation → PR 4 Update Planning & Policy Gating → PR 5 CLI Bridge Refactoring & Seam Preservation → PR 6 Canonical Specs & Final Verification |
| Delivery strategy | ask-on-risk |
| Chain strategy | stacked-to-main |

Decision needed before apply: Yes
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: High

The extraction of `internal/engine` spans domain models, concurrent worker pools, cooperative context cancellation, ownership resolution, policy planning, and extensive refactoring of `internal/cli` (`checkrun.go`, `list.go`, `update.go`). A 6-part stacked PR series guarantees that each increment is focused, independently testable, maintains existing dependency seams, and does not break existing CLI behavior or hermetic tests.

### Suggested Work Units

| Unit | Goal | PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|----|----------------------|-----------------|-------------------|
| 1 | Core domain models, constructor & options (`types.go`, `engine.go`, `engine_test.go`) | PR 1 | `go test ./internal/engine -run 'TestCalculateWorkerCount|TestEngine' -count=1` | N/A (pure domain models & constructor) | Revert `internal/engine/types.go`, `engine.go`, `engine_test.go` |
| 2 | Adapter discovery, custom tool manager binding & filtering (`resolve.go`, `resolve_test.go`) | PR 2 | `go test ./internal/engine -run 'TestResolve|TestResolvingOwner' -count=1` | N/A (static registry & platform mocks) | Revert `internal/engine/resolve.go`, `resolve_test.go` |
| 3 | Bounded concurrent check, panic containment & context cancellation (`check.go`, `check_test.go`) | PR 3 | `go test ./internal/engine -run 'TestCheck' -count=1` | N/A (in-memory fake adapters) | Revert `internal/engine/check.go`, `check_test.go` |
| 4 | Update planning, policy gating & ownership inheritance (`plan.go`, `plan_test.go`) | PR 4 | `go test ./internal/engine -run 'TestPlan' -count=1` | N/A (outcome test fixtures) | Revert `internal/engine/plan.go`, `plan_test.go` |
| 5 | CLI presentation bridge refactoring & test seam preservation (`checkrun.go`, `list.go`, `update.go`, `checkrun_test.go`) | PR 5 | `go test ./internal/cli/... -count=1` | `upp list` & `upp update -n` | Revert changes in `internal/cli/` |
| 6 | Canonical OpenSpec spec sync, smoke tests & quality gates | PR 6 | `go test ./... -count=1 -race` | `bash scripts/smoke-test.sh --skip-build` | Revert canonical spec files in `openspec/specs/` and smoke test script |

---

## Strict TDD

All implementation follows strict Test-Driven Development (TDD) as defined in `openspec/config.yaml`:
1. Write or update failing tests first ([RED]) to capture requirement specifications.
2. Implement the minimal production code necessary to pass the tests ([GREEN]).
3. Refactor code and verify quality gates ([REFACTOR]).
4. No subprocesses are spawned during unit testing; all adapter operations and dependencies are hermetically isolated.

---

## Phase 1 — Core Engine Domain Models, Constructor & Lifecycle (PR 1)

- [x] 1.1 [RED][S] Define pure domain models in [`internal/engine/types.go`](file:///home/jhan/Projects/upp/internal/engine/types.go) and write failing constructor/option unit tests in [`internal/engine/engine_test.go`](file:///home/jhan/Projects/upp/internal/engine/engine_test.go) [D1, D2, D3]:
  - Author [`internal/engine/types.go`](file:///home/jhan/Projects/upp/internal/engine/types.go) with pure domain types (zero imports of `internal/output`, `github.com/spf13/cobra`, or BubbleTea):
    - `type CheckStatus int` with enum constants `StatusAvailable`, `StatusCurrent`, `StatusSkipped`, `StatusFailed`, and `String() string`.
    - `type CheckOutcome struct` containing `ToolID string`, `ToolName string`, `Status CheckStatus`, `CurrentVersion string`, `LatestVersion string`, `UpdateAvailable bool`, `Err error`, `Stderr string`, `RawUpdateInfo adapters.UpdateInfo`.
    - `type CheckProgress struct` containing `Index int`, `Total int`, `Outcome CheckOutcome`.
    - `type Filter struct` containing `Only []string`.
    - `type PlannedUpdate struct` containing `ToolID string`, `ToolName string`, `ManagerID string`, `PackageName string`, `UpdatePolicy adapters.UpdatePolicy`, `CurrentVersion string`, `LatestVersion string`, `RiskCommand string`, `Privileges []string`, `Trust adapters.TrustLevel`.
    - `type UpdatePlan struct` containing `Updates []PlannedUpdate`, `Skipped []CheckOutcome`, `Current []CheckOutcome`, `Failed []CheckOutcome`.
  - Author [`internal/engine/engine_test.go`](file:///home/jhan/Projects/upp/internal/engine/engine_test.go):
    - `TestCalculateWorkerCount_Clamping`: Table-driven tests validating CPU core counts `-1, 0, 1, 2, 3, 4, 6, 8, 9, 16, 64` clamp strictly within $[4, 8]$.
    - `TestEngine_ConstructorDefaults`: Verifies `New(cfg, "linux")` configures clamped default worker pool, platform OS name, and configuration reference.
    - `TestEngine_WithConcurrency`: Verifies `WithConcurrency(6)` sets pool to 6; `WithConcurrency(0)` and `WithConcurrency(-2)` clamp to 1.
    - `TestEngine_WithAdapters`: Verifies `WithAdapters(list)` stores custom adapter override.
    - `TestEngine_HeadlessImportIsolation`: Verifies `internal/engine` imports neither `internal/output` nor BubbleTea.
  - Verify failure: Run `go test ./internal/engine -count=1` — fails compilation due to undefined `Engine`, `New`, `CalculateWorkerCount`, and functional options (RED).
- [x] 1.2 [S] Implement `Engine` struct, constructor, options, and concurrency calculation in [`internal/engine/engine.go`](file:///home/jhan/Projects/upp/internal/engine/engine.go) [D1, D3]:
  - Define `type Engine struct` holding `cfg *config.Config`, `osName string`, `numWorkers int`, and optional `adapters []adapters.Adapter`.
  - Define `type Option func(*Engine)`.
  - Implement `CalculateWorkerCount(numCPU int) int` clamping to $[4, 8]$.
  - Implement `New(cfg *config.Config, osName string, opts ...Option) *Engine`:
    - Defaults `numWorkers` to `CalculateWorkerCount(runtime.NumCPU())`.
    - Applies functional options.
  - Implement `WithConcurrency(workers int) Option` clamping values `< 1` to `1`.
  - Implement `WithAdapters(adapters []adapters.Adapter) Option`.
  - Verify success: Run `go test ./internal/engine -run 'TestCalculateWorkerCount|TestEngine' -count=1` — all tests pass (GREEN).
- [x] 1.3 [S] REFACTOR: Run `gofmt -s -w internal/engine/` and `go vet ./internal/engine/...`.

---

## Phase 2 — Adapter Discovery & Centralized Manager Ownership (PR 2)

- [x] 2.1 [RED][M] Write failing unit tests for adapter resolution, custom manager binding, and ownership helpers in [`internal/engine/resolve_test.go`](file:///home/jhan/Projects/upp/internal/engine/resolve_test.go) [D6]:
  - `TestResolve_InjectedAdaptersOverride`: Verifies engine initialized with `WithAdapters(...)` returns injected adapters directly when `filter.Only` is empty.
  - `TestResolve_PlatformOfficialAdapters`: Verifies platform adapters for "linux" (apt, pacman, etc.) and "darwin" (brew, etc.) are enumerated in canonical discovery order.
  - `TestResolve_DisabledToolsExcluded`: Verifies `cfg.Tools["tool"].Enabled = false` excludes that tool from resolved output.
  - `TestResolve_CustomToolManagerBinding`:
    - Custom tool with `Manager: "apt"` on Linux: binds `apt` official manager adapter (`KindManager`).
    - Custom tool with `Manager: "invalid"`: falls back to standalone tool without error.
    - Custom tool with `Manager: "gh"` (which declares `KindTool`, not `KindManager`): falls back to standalone tool without error.
  - `TestResolve_FilterOnly`:
    - `Filter{Only: ["brew", "npm"]}`: filters resolved set to only matching tools.
    - `Filter{Only: ["BREW"]}`: verifies case-insensitive matching.
    - `Filter{Only: ["unknown"]}`: returns empty slice without error.
  - `TestResolvingOwner_Helpers`: Tests `ResolvingOwner`, `OwnedPackage`, `UpdateCmdName` for official and custom manager bindings across Linux, Darwin, and Windows platforms.
  - Verify failure: Run `go test ./internal/engine -run 'TestResolve|TestResolvingOwner' -count=1` — fails compilation due to undefined `Resolve`, `ResolvingOwner`, `OwnedPackage`, `UpdateCmdName` (RED).
- [x] 2.2 [M] Implement adapter discovery, manager binding, filtering, and ownership helpers in [`internal/engine/resolve.go`](file:///home/jhan/Projects/upp/internal/engine/resolve.go) [D6]:
  - Implement `(e *Engine) Resolve(filter Filter) ([]adapters.Adapter, error)`:
    - If `e.adapters != nil`, use injected adapters as source; otherwise:
      - Query `official.AdaptersForPlatform(e.osName)`.
      - Filter out tools where `e.cfg.Tools[id].Enabled == false`.
      - Instantiate custom adapters from `e.cfg.Custom`:
        - If `custom.Manager != ""`: look up manager via `official.AdapterByName(custom.Manager)` and verify `mgr.Info().Kind == adapters.KindManager`. If valid, bind manager adapter to custom adapter constructor.
        - Append valid custom adapter to resolved list.
    - Apply `filter.Only`: if `len(filter.Only) > 0`, filter case-insensitively by tool ID / name.
    - Return resolved adapters in canonical order.
  - Implement centralized ownership helper functions:
    - `ResolvingOwner(a adapters.Adapter, osName string, allAdapters ...[]adapters.Adapter) adapters.Adapter`: checks `allAdapters` override, custom adapter manager, or `official.ResolveOwner(a.Name(), osName)`.
    - `OwnedPackage(a adapters.Adapter, osName string) string`: returns `a.Info().ManagerPackage[osName]`.
    - `UpdateCmdName(manager string) string`: returns `"sudo apt install --only-upgrade"` for apt, `"brew upgrade"` for brew, `"winget upgrade"` for winget, `"<manager> upgrade"` default.
  - Verify success: Run `go test ./internal/engine -run 'TestResolve|TestResolvingOwner' -count=1` — all tests pass (GREEN).
- [x] 2.3 [S] REFACTOR: Run `gofmt -s -w internal/engine/resolve.go internal/engine/resolve_test.go` and `go vet ./internal/engine/...`.

---

## Phase 3 — Bounded Concurrent Check Engine with Context & Panic Safety (PR 3)

- [x] 3.1 [RED][L] Write failing unit tests for concurrent checking, bounded concurrency, panic containment, and cooperative context cancellation in [`internal/engine/check_test.go`](file:///home/jhan/Projects/upp/internal/engine/check_test.go) [D4, D5]:
  - Test harness fake adapters:
    - `testDelayedAdapter`: controllable sleep delay, configurable `UpdateInfo`, `checkErr`, `detectResult`.
    - `testPanickingAdapter`: panics in `Detect()` or panics in `Check()`.
    - `testConcurrencyTrackingAdapter`: tracks peak active concurrent checks via atomic counter.
  - `TestCheck_BoundedConcurrency`: 12 slow adapters passed on engine with concurrency 4; verifies peak concurrent executions never exceed 4.
  - `TestCheck_PanicContainment_Detect`: Adapter panics during `Detect()`; verifies panic caught by deferred `recover()`, outcome status is `StatusFailed`, error contains `"panic during check"`, other adapters finish successfully.
  - `TestCheck_PanicContainment_Check`: Adapter panics during `Check()`; verifies panic caught, outcome status is `StatusFailed`, `RawUpdateInfo` is zero-value.
  - `TestCheck_DetectSkipped`: Tool where `Detect() == false`; verifies status is `StatusSkipped`, `Check()` is never invoked.
  - `TestCheck_TimeoutWrapping`: Adapter check returns `context.DeadlineExceeded`; verifies error is wrapped with `TimeoutErr` naming tool and timeout duration.
  - `TestCheck_DeterministicSlotting`: Out-of-order completions (adapter 2 sleeps 5ms, adapter 0 sleeps 50ms); verifies output slice strictly matches canonical input order `[0, 1, 2]`.
  - `TestCheck_ProgressCallback`: Verifies `onProgress` is invoked exactly once per adapter with matching `Index`, `Total`, and `Outcome`.
  - `TestCheck_NilCallbackSilent`: Verifies `Check(ctx, adapters, nil)` executes cleanly without panics.
  - `TestCheck_ContextCancellation_Immediate`: Pre-canceled context (`ctx, cancel := context.WithCancel(...); cancel()`); verifies `Check` returns `context.Canceled` immediately without starting checks.
  - `TestCheck_ContextCancellation_MidFlight`: 10 slow checks; context canceled after 2 completions; verifies workers discard remaining queued tasks, active workers exit cleanly, `sync.WaitGroup` terminates, and `Check` returns `context.Canceled` with zero leaked goroutines.
  - `TestCheck_ContextDeadlineExceeded`: Context with short timeout; verifies returns `context.DeadlineExceeded`.
  - Verify failure: Run `go test ./internal/engine -run 'TestCheck' -count=1` — fails compilation due to undefined `Check`, `TimeoutErr` (RED).
- [x] 3.2 [M] Implement `Check`, worker pool, panic recovery, context observation, and `TimeoutErr` in [`internal/engine/check.go`](file:///home/jhan/Projects/upp/internal/engine/check.go) [D4, D5]:
  - Implement `TimeoutErr(name, op string, err error) error`: wraps `context.DeadlineExceeded` with tool name, op (`check`/`update`), and timeout limit (`adapters.CheckTimeout`/`adapters.UpdateTimeout`).
  - Implement `safeCheck(ctx context.Context, a adapters.Adapter) CheckOutcome`:
    - Deferred `recover()` block: catches panics in `Detect()` or `Check()`, returns `StatusFailed` with `Err: fmt.Errorf("panic during check: %v", rec)` and zero-value `RawUpdateInfo`.
    - Check `ctx.Err()` before starting work; return canceled outcome if context is done.
    - If `!a.Detect()`, return `StatusSkipped`.
    - Execute `a.Check()`:
      - If error, wrap with `TimeoutErr(info.Name, "check", err)`, return `StatusFailed` with `Err: err`, `Stderr: err.Error()`.
      - If update available, return `StatusAvailable` with version info and `RawUpdateInfo`.
      - If current, return `StatusCurrent` with current version and `RawUpdateInfo`.
  - Implement `(e *Engine) Check(ctx context.Context, adapterList []adapters.Adapter, onProgress func(CheckProgress)) ([]CheckOutcome, error)`:
    - Guard against already-canceled context: if `ctx.Err() != nil`, return `nil, ctx.Err()`.
    - Compute `workerCount = min(e.numWorkers, len(adapterList))`.
    - Pre-allocate `outcomes := make([]CheckOutcome, len(adapterList))`.
    - Create buffered jobs channel of size `len(adapterList)` and populate with input index and adapter.
    - Spawn worker pool with `sync.WaitGroup`:
      - Worker checks `select { case <-ctx.Done(): return default: }` before reading each job.
      - Worker calls `safeCheck(ctx, job.adapter)`.
      - Worker slots outcome into `outcomes[job.index] = oc`.
      - Worker calls `onProgress(CheckProgress{Index: job.index, Total: len(adapterList), Outcome: oc})` if non-nil.
    - `wg.Wait()` before returning.
    - If `ctx.Err() != nil`, return outcomes, `ctx.Err()`.
  - Verify success: Run `go test ./internal/engine -run 'TestCheck' -count=1` — all check tests pass (GREEN).
- [x] 3.3 [S] REFACTOR: Run `gofmt -s -w internal/engine/check.go internal/engine/check_test.go` and `go vet ./internal/engine/...`.

---

## Phase 4 — Update Planning & Policy Gating (PR 4)

- [x] 4.1 [RED][M] Write failing unit tests for update planning, policy gating, and ownership inheritance in [`internal/engine/plan_test.go`](file:///home/jhan/Projects/upp/internal/engine/plan_test.go) [D6]:
  - `TestPlan_OutcomeSegregation`:
    - `StatusFailed` outcome placed in `UpdatePlan.Failed` (never in `Updates`).
    - `StatusSkipped` outcome placed in `UpdatePlan.Skipped` (never in `Updates`).
  - `TestPlan_PolicyGated`:
    - Gated tool (`npm`, `apt`) with `UpdateAvailable == true`: placed in `UpdatePlan.Updates` with `PlannedUpdate` populated.
    - Gated tool with `UpdateAvailable == false`: placed in `UpdatePlan.Current` (never in `Updates`).
  - `TestPlan_PolicyAlwaysUpdate`:
    - Always-update tool (`brew`, `bun`) with `UpdateAvailable == false`: placed unconditionally in `UpdatePlan.Updates`.
  - `TestPlan_OwnedToolPolicyInheritance`:
    - Tool `gh` owned by `apt` (`PolicyGated`) on Linux: when `UpdateAvailable == false`, placed in `UpdatePlan.Current` despite `gh`'s own policy.
    - Tool `gh` owned by `brew` (`PolicyAlwaysUpdate`) on macOS: when `UpdateAvailable == false`, placed in `UpdatePlan.Updates` because `brew` policy governs.
  - `TestPlan_PlannedUpdateFields`:
    - Verifies `PlannedUpdate` has accurate `ToolID`, `ToolName`, `ManagerID`, `PackageName`, `UpdatePolicy`, `CurrentVersion`, `LatestVersion`, `RiskCommand`, `Privileges`, and `Trust`.
  - `TestResolveEffectiveUpdatePolicy`:
    - Standalone tools return declared policy.
    - Owned tools return manager's declared policy.
  - Verify failure: Run `go test ./internal/engine -run 'TestPlan' -count=1` — fails compilation due to undefined `Plan`, `ResolveEffectiveUpdatePolicy` (RED).
- [x] 4.2 [M] Implement update planning and effective policy resolution in [`internal/engine/plan.go`](file:///home/jhan/Projects/upp/internal/engine/plan.go) [D6]:
  - Implement `ResolveEffectiveUpdatePolicy(a adapters.Adapter, osName string, allAdapters ...[]adapters.Adapter) adapters.UpdatePolicy`:
    - If `owner := ResolvingOwner(a, osName, allAdapters...)`, return `owner.Info().UpdatePolicy`.
    - Otherwise return `a.Info().UpdatePolicy`.
  - Implement `(e *Engine) Plan(outcomes []CheckOutcome, filter Filter) (UpdatePlan, error)`:
    - Pre-allocate slices for `Updates`, `Skipped`, `Current`, `Failed`.
    - Resolve tool adapters for outcomes.
    - For each `CheckOutcome`:
      - If `StatusFailed`: add to `Failed`.
      - If `StatusSkipped`: add to `Skipped`.
      - If `StatusCurrent` or `StatusAvailable`:
        - Match adapter.
        - Resolve owner via `ResolvingOwner(a, e.osName)`.
        - Derive effective policy via `ResolveEffectiveUpdatePolicy(a, e.osName)`.
        - Check eligibility:
          - If effective policy is `PolicyGated` and `!oc.UpdateAvailable`: add to `Current`.
          - If effective policy is `PolicyAlwaysUpdate` or (`PolicyGated` and `oc.UpdateAvailable`):
            - Formulate `riskCommand`: if owned, `fmt.Sprintf("%s %s", UpdateCmdName(owner.Name()), pkg)`; else `info.Command` or `info.Name + " update"`.
            - Construct `PlannedUpdate` and append to `Updates`.
    - Return `UpdatePlan`.
  - Verify success: Run `go test ./internal/engine -run 'TestPlan' -count=1` — all planning tests pass (GREEN).
- [x] 4.3 [S] REFACTOR: Run `gofmt -s -w internal/engine/plan.go internal/engine/plan_test.go` and `go vet ./internal/engine/...`.

---

## Phase 5 — CLI Bridge Refactoring & Dep Preservation (PR 5)

- [x] 5.1 [RED][M] Update CLI check tests in [`internal/cli/checkrun_test.go`](file:///home/jhan/Projects/upp/internal/cli/checkrun_test.go) to test `outcomeToToolResult` mapping bridge and verify backward-compatible `runChecks` wrapper [D2, D7]:
  - Add `TestOutcomeToToolResult`: table-driven tests mapping:
    - `StatusAvailable` -> `output.StatusAvailable` with `"Current → Latest"`.
    - `StatusCurrent` -> `output.StatusCurrent` with `"Current"`.
    - `StatusSkipped` -> `output.StatusSkipped`.
    - `StatusFailed` -> `output.StatusFailed` with `Error` and `Stderr`.
  - Verify failure: Run `go test ./internal/cli -run 'TestOutcomeToToolResult' -count=1` — fails compilation due to undefined `outcomeToToolResult` (RED).
- [x] 5.2 [M] Refactor [`internal/cli/checkrun.go`](file:///home/jhan/Projects/upp/internal/cli/checkrun.go) to delegate checking to `internal/engine` while preserving presentation bridge and seams [D2, D7]:
  - Implement `outcomeToToolResult(oc engine.CheckOutcome) output.ToolResult`.
  - Refactor `calculateWorkerCount` to delegate to `engine.CalculateWorkerCount`.
  - Refactor `safeCheck` and `runChecks` to delegate to `engine.New(cfg, ...).Check(...)` and `outcomeToToolResult`, preserving the `onResult func(index int, oc checkOutcome)` callback signature for existing callers.
  - Refactor `buildAdapterList` to delegate to `engine.New(cfg, osName).Resolve(engine.Filter{})`, preserving the exact signature `func(cfg *config.Config, osName string) []adapters.Adapter` used by `integration_test.go` and `cliDeps`.
  - Verify success: Run `go test ./internal/cli -run 'TestOutcomeToToolResult|TestCalculateWorkerCount|TestSafeCheck|TestRunChecks' -count=1` — all pass (GREEN).
- [x] 5.3 [M] Refactor [`internal/cli/list.go`](file:///home/jhan/Projects/upp/internal/cli/list.go) to delegate adapter resolution to `engine.Engine` [D6, D7]:
  - In `runList`, instantiate `eng := engine.New(cfg, p.OS, ...)` (passing `WithAdapters(deps.buildAdapterList(cfg, p.OS))` if `deps.buildAdapterList != nil`).
  - Delegate adapter resolution to `eng.Resolve(engine.Filter{Only: only})`.
  - Pass resolved adapters directly to `output.GroupByOwner`.
  - Preserve `listDeps` test seam and all output messaging (`NoToolsMatchFilter`, `NoToolsConfigured`, `ListTools`).
  - Verify success: Run `go test ./internal/cli -run 'TestList' -count=1` — all pass (GREEN).
- [x] 5.4 [L] Refactor [`internal/cli/update.go`](file:///home/jhan/Projects/upp/internal/cli/update.go) to delegate checking and planning to `engine.Engine` [D6, D7]:
  - In `runUpdate`:
    - Instantiate engine `eng := engine.New(cfg, p.OS, ...)` (passing `WithAdapters(deps.buildAdapterList(cfg, p.OS))` if `deps.buildAdapterList != nil`).
    - Resolve adapters with `eng.Resolve(engine.Filter{Only: onlyList})`.
    - Retain interactive gate condition: `deps.stdinIsTTY() && !gf.CI && !gf.Quiet && !uf.DryRun`.
  - In `runUpdateSequential`:
    - Delegate check execution to `eng.Check(ctx, filteredAdapters, ...) and update planning to eng.Plan(outcomes, filter)`.
    - Loop through `UpdatePlan.Updates`, verify trust and risk with `security.ConfirmAction`, execute updates via adapter or `PackageUpdater`, and report to `output.Renderer`.
    - Map `UpdatePlan.Current`, `UpdatePlan.Skipped`, and `UpdatePlan.Failed` directly into results for `output.Summary`.
  - In `runUpdateInteractive`:
    - Reorder adapters using `output.GroupOrder(filteredAdapters, osName)`.
    - Start `board := output.NewCheckBoard(...)`.
    - Delegate pre-check to `eng.Check(ctx, grouped, func(prog engine.CheckProgress) { board.Complete(prog.Index, outcomeToToolResult(prog.Outcome)) })`.
    - Present pending updates on `output.CheckboxSelector`.
    - For selected tools, delegate planning/gating to `eng.Plan` or process selected outcomes.
    - Render `output.Summary` with deterministic ordering preserved.
  - Delegate `resolvingOwner`, `ownedPackage`, `updateCmdName`, `resolveEffectiveUpdatePolicy`, and `timeoutErr` to their counterparts in `internal/engine`.
  - Verify success: Run `go test ./internal/cli/... -count=1` — all tests pass (GREEN).
- [x] 5.5 [S] REFACTOR: Run all CLI integration tests to ensure 100% backward compatibility:
  - Run `go test ./internal/cli -run 'TestIntegration|TestUpdate|TestList' -count=1`.
  - Verify all CLI hermetic tests pass with zero regressions.
  - Run `gofmt -s -w internal/cli/` and `go vet ./internal/cli/...`.

---

## Phase 6 — Comprehensive Verification & OpenSpec Specification Sync (PR 6)

- [x] 6.1 [M] Synchronize canonical OpenSpec specifications:
  - Create canonical specification [`openspec/specs/application-engine/spec.md`](file:///home/jhan/Projects/upp/openspec/specs/application-engine/spec.md) from the delta spec.
  - Merge deltas into canonical specs:
    - [`openspec/specs/command-interface/spec.md`](file:///home/jhan/Projects/upp/openspec/specs/command-interface/spec.md): Update `upp update` and orchestration delegation requirements.
    - [`openspec/specs/tool-ownership-model/spec.md`](file:///home/jhan/Projects/upp/openspec/specs/tool-ownership-model/spec.md): Update resolved owner update delegation and centralized ownership resolution requirements.
    - [`openspec/specs/ux-patterns/spec.md`](file:///home/jhan/Projects/upp/openspec/specs/ux-patterns/spec.md): Update live check board, summary report deterministic ordering, and engine progress/outcome mapping requirements.
- [x] 6.2 [S] Verify smoke test suite compatibility in [`scripts/smoke-test.sh`](file:///home/jhan/Projects/upp/scripts/smoke-test.sh):
  - Ensure all smoke test assertions (`upp list`, `upp update -n`, `upp update -n --only ...`) continue to run cleanly against the refactored CLI binary.
- [x] 6.3 [S] Run full quality gates:
  - Full test suite: `go test ./... -count=1`.
  - Race condition detector: `go test ./... -count=1 -race`.
  - Type checker & vet: `go vet ./...`.
  - Code formatting: verify `gofmt -s -l .` produces empty output.
  - Binary compilation: `go build -o upp ./cmd/upp`.
  - Smoke tests: `bash scripts/smoke-test.sh --skip-build`.
