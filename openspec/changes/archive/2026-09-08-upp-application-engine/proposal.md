# Proposal: Application Engine Layer Extraction (`upp-application-engine`)

## Intent

Currently, `internal/cli` is overloaded with mixed responsibilities, coupling command-line parsing and terminal user interface rendering with core domain orchestration:
1. **Domain Orchestration in Presentation Layer**: `internal/cli` directly coordinates adapter discovery, custom tool manager ownership resolution (`buildAdapterList`), worker pool sizing and clamping to $[4, 8]$ (`calculateWorkerCount`, `defaultConcurrency`), panic containment (`safeCheck`), concurrent check dispatching (`runChecks`), effective update policy inheritance (`resolveEffectiveUpdatePolicy`, `resolvingOwner`), and update planning.
2. **Presentation Coupling**: `checkrun.go` directly imports `internal/output` and returns presentation types (`output.ToolResult`, `output.StatusAvailable`, `output.StatusCurrent`, `output.StatusSkipped`, `output.StatusFailed`). Core business logic and version check workflows cannot be executed or tested headlessly without instantiating UI representations.
3. **Lack of Cooperative Context Cancellation**: `runChecks` does not accept `context.Context`. If an interrupt signal (such as `SIGINT` / Ctrl+C) or an operational deadline occurs, ongoing worker goroutines cannot be canceled cooperatively and continue executing subprocesses until completion.
4. **Scattered Business Rules**: Logic for manager-tool ownership mapping, delegation, and effective policy gating (e.g., owned tools inheriting their manager's `UpdatePolicy` while the owned tool's declared policy is inert) is tightly coupled with terminal output loops in `runUpdateSequential` and `runUpdateInteractive`.

This proposal extracts a dedicated, headless Application Engine (`internal/engine`) sitting between `internal/cli` and `internal/adapters`. The engine encapsulates tool resolution, concurrent check execution with bounded concurrency, panic containment, cooperative context cancellation, and update plan formulation. The `internal/cli` package is refactored into a thin presentation and interaction bridge that delegates domain operations to `engine.Engine`.

## Scope

### In Scope

- **`internal/engine` Package**:
  - Pure domain models: `CheckStatus` enum, `CheckOutcome`, `CheckProgress`, `Filter`, `PlannedUpdate`, and `UpdatePlan`. Zero dependencies on `internal/output`, `github.com/spf13/cobra`, or BubbleTea.
  - Lifecycle & Constructor: `engine.New(cfg *config.Config, osName string, opts ...Option) *Engine`.
  - Adapter Discovery & Resolution: `Resolve(filter Filter) ([]adapters.Adapter, error)` resolving enabled platform and custom adapters, binding custom tool manager ownership, and applying filtering.
  - Concurrent Check Engine: `Check(ctx context.Context, adapters []adapters.Adapter, onProgress func(CheckProgress)) ([]CheckOutcome, error)` featuring bounded worker pooling clamped to $[4, 8]$, panic containment via `recover()`, cooperative `context.Context` cancellation, and index-based deterministic result slotting.
  - Update Planning: `Plan(outcomes []CheckOutcome, filter Filter) (UpdatePlan, error)` categorizing tools, resolving manager ownership and policy inheritance (`PolicyGated` vs `PolicyAlwaysUpdate`), and producing executable update actions.
- **Engine Unit Test Suite**:
  - Hermetic unit tests verifying worker pool bounds ($[4, 8]$ clamping), concurrency limits, panic recovery in `Detect()` and `Check()`, cooperative cancellation via `context.Context`, deterministic result ordering, and update plan formulation.
- **CLI Refactoring**:
  - `internal/cli/checkrun.go`: Refactored to act as a thin bridge converting `engine.CheckOutcome` to `output.ToolResult`.
  - `internal/cli/list.go`: Refactored to delegate adapter resolution to `engine.Resolve()`.
  - `internal/cli/update.go`: Refactored to delegate adapter resolution, checking, and planning to `engine.Engine`, while maintaining presentation rendering (`output.Renderer`, `output.CheckBoard`), TTY detection, interactive checkbox selection, and security confirmation (`security.ConfirmAction`).
  - Maintenance of test seams (`cliDeps`, `updateDeps`, `listDeps`) ensuring existing hermetic CLI tests continue to pass.
- **Preservation of Behavior**:
  - 100% preservation of all existing CLI flags, commands, visual output, exit codes, and test assertions across all integration and unit tests.

### Out of Scope

- Modifying user-facing CLI flags, command hierarchy, or help text (`upp list`, `upp update`, `upp init`, `upp self-update`, `upp uninstall`).
- Modifying terminal presentation, ANSI styling, emojis, tables, or progress rendering in `internal/output`.
- Moving interactive terminal prompts (the interactive checkbox selector loop in `runUpdateInteractive`) into `internal/engine` (interactive prompts remain in `internal/cli`).
- Altering the low-level adapter interface (`adapters.Adapter`) or official adapter implementations in `internal/adapters/official/`.
- Changing adapter execution seams (`CommandRunner`, `runCmdFn`).
- Exposing concurrency flags (e.g. `--concurrency` or `-j`) to users; worker pool clamping remains automatic.

## Capabilities

### New Capabilities

- `application-engine`: Introduces a headless Application Engine (`internal/engine`) that MUST encapsulate tool resolution, concurrent checking with bounded worker pooling, cooperative cancellation, panic containment, and update planning with policy gating. The engine MUST operate completely independently of CLI presentation, output formatting, or interactive I/O.

### Modified Capabilities

- `ux-patterns`: Terminal UI components (`output.CheckBoard`, `output.Renderer`) MUST consume engine progress events (`engine.CheckProgress`) and map pure domain outcomes (`engine.CheckOutcome`) to presentation models (`output.ToolResult`). Deterministic ordering guarantees in summary reports MUST be maintained by the engine's index-slotted result gathering.
- `command-interface`: Commands (`list`, `update`) MUST delegate orchestration, discovery, and planning to `engine.Engine` while maintaining identical flag semantics, output formatting, error isolation, and exit codes.
- `tool-ownership-model`: Manager resolution, ownership mapping, and effective policy inheritance MUST be centralized in the application engine during discovery and planning rather than evaluated ad-hoc within CLI loops.
- `tool-adapter`: Adapter check dispatch MUST support cooperative cancellation via `context.Context` and panic containment within the engine boundary, ensuring worker panics or canceled contexts do not crash or hang the process.

## Approach

### D1: Layered Engine Package (`internal/engine`)

Establish a clean layered architecture where `internal/engine` sits between `internal/cli` and `internal/adapters`:
- **Dependency Hierarchy**:
  ```
  cmd/upp/main.go
         │
         ▼
    internal/cli ──────────┐
         │                 ▼
         │          internal/output
         ▼
   internal/engine
         │
         ├──────────────┬──────────────┬──────────────┐
         ▼              ▼              ▼              ▼
  internal/adapters  internal/    internal/      internal/
  (official/custom)  platform     config         security
  ```
- **Strict Boundary**: `internal/engine` MUST NOT import `internal/cli` or `internal/output`. Domain logic remains completely free of presentation dependencies.

### D2: Pure Domain Models in `engine`

Define domain types in `internal/engine` that encapsulate state without UI primitives:
- `CheckStatus`: Typed enumeration for check states (`StatusAvailable`, `StatusCurrent`, `StatusSkipped`, `StatusFailed`).
- `CheckOutcome`: Pure domain result of an adapter check:
  - `ToolID string`
  - `ToolName string`
  - `Status CheckStatus`
  - `CurrentVersion string`
  - `LatestVersion string`
  - `UpdateAvailable bool`
  - `Err error`
  - `Stderr string`
  - `RawUpdateInfo adapters.UpdateInfo`
- `CheckProgress`: Progress event struct emitted during concurrent execution:
  - `Index int`
  - `Total int`
  - `Outcome CheckOutcome`
- `Filter`: Query criteria containing `Only []string`.
- `PlannedUpdate`: Structured update action containing:
  - `ToolID string`
  - `ToolName string`
  - `ManagerID string` (empty if standalone)
  - `PackageName string` (package name under manager, if owned)
  - `UpdatePolicy adapters.UpdatePolicy`
  - `CurrentVersion string`
  - `LatestVersion string`
  - `RiskCommand string`
  - `Privileges adapters.PrivilegeLevel`
  - `Trust adapters.TrustLevel`
- `UpdatePlan`: Composite plan containing:
  - `Updates []PlannedUpdate`
  - `Skipped []CheckOutcome`
  - `Current []CheckOutcome`
  - `Failed []CheckOutcome`

### D3: Engine Lifecycle & Constructor

The engine instance maintains configuration and platform context:
```go
type Engine struct {
    cfg        *config.Config
    osName     string
    numWorkers int
}

type Option func(*Engine)

func New(cfg *config.Config, osName string, opts ...Option) *Engine
func WithConcurrency(workers int) Option
```
- Defaults `numWorkers` to `calculateWorkerCount(runtime.NumCPU())` clamped within $[4, 8]$.

### D4: First-Class `context.Context` Support & Safe Execution

`Check` accepts `context.Context` for cooperative cancellation:
```go
func (e *Engine) Check(ctx context.Context, adapters []adapters.Adapter, onProgress func(CheckProgress)) ([]CheckOutcome, error)
```
- **Cancellation**: Workers verify `ctx.Err()` before picking up tasks from the jobs channel. If the context is canceled, unstarted tasks are marked canceled/skipped or returned with `ctx.Err()`.
- **Worker Pool**: Clamped to $[4, 8]$ workers (or total adapters if smaller).
- **Panic Containment**: Each worker wraps execution in a deferred `recover()` block. If an adapter panics during `Detect()` or `Check()`, the panic is contained and returned as `StatusFailed` with a descriptive error (`panic during check: ...`).
- **Deterministic Ordering**: Outcomes are slotted into a pre-allocated slice using the adapter's canonical input index, guaranteeing output order matches input order regardless of worker completion order.

### D5: Update Planning

`Plan` separates planning from execution:
```go
func (e *Engine) Plan(outcomes []CheckOutcome, filter Filter) (UpdatePlan, error)
```
- Evaluates gating policy: `PolicyGated` tools update only when `UpdateAvailable == true`; `PolicyAlwaysUpdate` tools update unconditionally.
- For owned tools, resolves effective update policy and risk commands from the owning manager adapter.
- Returns a structured `UpdatePlan` ready for CLI confirmation and execution.

### D6: Thin CLI Adapter Bridge

`internal/cli` delegates orchestration to `engine.Engine`:
- `internal/cli/checkrun.go` becomes a translation bridge mapping `engine.CheckOutcome` to `output.ToolResult`.
- `internal/cli/list.go` invokes `engine.Resolve()` to fetch adapters and passes them to `output.GroupByOwner`.
- `internal/cli/update.go` invokes `engine.Resolve()`, `engine.Check()`, and `engine.Plan()`:
  - In interactive mode: passes `engine.CheckProgress` to `output.CheckBoard`, presents pending updates via `output.CheckboxSelector`, and executes selected updates.
  - In sequential mode: iterates through planned updates, verifies trust and risk with `security.ConfirmAction`, executes updates, and renders `output.Summary`.
- Existing test injection points (`cliDeps`) are preserved so all hermetic CLI tests remain functional.

## Affected Areas

| Area | Impact | Description |
|---|---|---|
| `internal/engine/engine.go` | New | Engine struct, constructor `New`, and options |
| `internal/engine/types.go` | New | Domain types: `CheckStatus`, `CheckOutcome`, `CheckProgress`, `Filter`, `PlannedUpdate`, `UpdatePlan` |
| `internal/engine/resolve.go` | New | Adapter discovery and custom tool manager binding (`Resolve`) |
| `internal/engine/check.go` | New | Worker pool, `safeCheck` panic containment, and context-cancellable `Check` |
| `internal/engine/plan.go` | New | Update planning, manager ownership resolution, and policy gating (`Plan`) |
| `internal/engine/*_test.go` | New | Unit tests for engine: worker limits, panic isolation, context cancellation, and planning |
| `internal/cli/checkrun.go` | Modified | Delegate checking and resolution to `engine`, map domain types to `output.ToolResult` |
| `internal/cli/list.go` | Modified | Delegate tool discovery to `engine.Resolve()` |
| `internal/cli/update.go` | Modified | Delegate pre-checks and planning to `engine.Check()` and `engine.Plan()` |
| `internal/cli/deps.go` | Modified | Update dependency injection seams to support engine delegation |
| `internal/cli/*_test.go` | Modified | Verify CLI integration and hermetic unit tests with engine delegation |
| `openspec/specs/ux-patterns/spec.md` | Modified | Document engine progress callback seam and deterministic ordering |
| `openspec/specs/command-interface/spec.md` | Modified | Clarify CLI orchestration delegation to application engine |
| `openspec/specs/tool-adapter/spec.md` | Modified | Document context cancellation and panic recovery around adapter checks |
| `openspec/specs/tool-ownership-model/spec.md` | Modified | Document engine-managed ownership resolution and policy inheritance |

## Risks

| Risk | Likelihood | Mitigation |
|---|---|---|
| Presentation or CLI test regression | Medium | Maintain existing `cliDeps` seams; ensure `engine.CheckOutcome` maps 1:1 to `output.ToolResult`; all existing integration tests MUST pass byte-for-byte. |
| Race conditions in concurrent check execution | Low | Slot results by index into pre-allocated slice; synchronize progress callbacks; validate with `go test -race ./...`. |
| Context cancellation deadlocks or leaks | Low | Ensure job channels are closed properly; `sync.WaitGroup` waits for all active workers before returning; write dedicated unit tests for canceled contexts. |
| Divergence in ownership policy inheritance | Medium | Centralize `resolvingOwner` and `resolveEffectiveUpdatePolicy` inside `internal/engine`; verify against existing parity and update tests. |
| Breaking test hermeticity in CLI tests | Low | Retain fake adapter list builder seams in `cliDeps`; ensure engine accepts injected adapters or fakes without spawning subprocesses. |

## Rollback Plan

If regressions occur during implementation or deployment:
1. Revert the git commits introducing `internal/engine/` and refactoring `internal/cli/`.
2. Because this is a pure internal code refactoring with no external database, file format, configuration schema, or CLI flag changes, `git revert` cleanly restores the previous inline implementation in `internal/cli/`.
3. Verify the rollback by executing `go test ./... -count=1 -race` and `bash scripts/smoke-test.sh --skip-build`.

## Dependencies

- **Go Standard Library**: `context`, `sync`, `sync/atomic`, `runtime`, `fmt`, `errors`.
- **Internal Packages**:
  - `internal/adapters`
  - `internal/adapters/official`
  - `internal/config`
  - `internal/platform`
  - `internal/security`
- **Zero External Dependencies**: No third-party packages added to `go.mod`.

## Success Criteria

- [ ] `internal/engine` package created with pure domain models (`CheckOutcome`, `CheckStatus`, `CheckProgress`, `UpdatePlan`, `PlannedUpdate`, `Filter`).
- [ ] Zero dependencies from `internal/engine` to `internal/output`, `internal/cli`, or BubbleTea.
- [ ] `engine.Check` accepts `context.Context` and cancels gracefully when the context is canceled.
- [ ] Bounded worker pool clamping ($[4, 8]$) and panic containment verified via hermetic table-driven unit tests in `internal/engine`.
- [ ] `internal/cli/checkrun.go`, `internal/cli/list.go`, and `internal/cli/update.go` refactored to delegate to `engine.Engine`.
- [ ] 100% preservation of all existing CLI flags, commands, visual output, and exit codes.
- [ ] All existing integration and hermetic CLI tests pass without regression.
- [ ] `go test ./... -count=1 -race` passes cleanly.
- [ ] `bash scripts/smoke-test.sh --skip-build` passes cleanly.
- [ ] Spec deltas documented for `ux-patterns`, `command-interface`, `tool-adapter`, and `tool-ownership-model`.
