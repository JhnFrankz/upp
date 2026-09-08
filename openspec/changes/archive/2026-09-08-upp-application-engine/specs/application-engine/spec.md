# Delta for application-engine

## Purpose

Define the headless Application Engine architecture, lifecycle, pure domain models, adapter resolution, concurrent check execution with panic containment and context cancellation, and update plan formulation.

## NEW Requirements

### Requirement: Engine Architecture and Lifecycle

The system MUST provide an application engine instantiated via `engine.New(cfg *config.Config, osName string, opts ...Option) *Engine`. The engine instance MUST encapsulate the active configuration, the host operating system platform identifier (`osName`), and a bounded concurrency limit.

The engine MUST calculate its default concurrency worker count using `calculateWorkerCount(runtime.NumCPU())` clamped strictly within the range $[4, 8]$. The engine MUST support functional options including `WithConcurrency(workers int) Option`. If a custom concurrency value is supplied via options, worker counts below 1 MUST be clamped to 1.

The engine package MUST NOT import presentation packages (`internal/output`), command-line definition packages (`github.com/spf13/cobra`), or interactive terminal UI packages (`github.com/charmbracelet/bubbletea`). The engine MUST be capable of compiling, initializing, and executing headlessly in non-terminal, CI, and programmatic environments.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Default concurrency on 8-core CPU | Host has 8 CPU cores | `engine.New(cfg, "linux")` | Worker pool initialized with exactly 8 workers |
| Concurrency clamped on 2-core CPU | Host has 2 CPU cores | `engine.New(cfg, "linux")` | Worker pool clamped to minimum bound of 4 workers |
| Concurrency clamped on 16-core CPU | Host has 16 CPU cores | `engine.New(cfg, "linux")` | Worker pool clamped to maximum bound of 8 workers |
| Custom concurrency option | Engine initialized with `WithConcurrency(6)` | `engine.New(cfg, "darwin", WithConcurrency(6))` | Worker pool initialized with 6 workers |
| Custom concurrency lower bound | Engine initialized with `WithConcurrency(0)` | `engine.New(cfg, "windows", WithConcurrency(0))` | Worker count clamped to 1 worker |
| Headless compilation | Engine package imported without UI dependencies | Build / test execution | Package compiles cleanly without `internal/output` or BubbleTea |

### Requirement: Domain Model Isolation

The engine MUST define and operate on pure domain models representing tool states, execution outcomes, progress notifications, and update plans, decoupled from terminal presentation primitives.

The engine MUST define the following domain types:
- `CheckStatus`: typed enumeration with distinct values `StatusAvailable`, `StatusCurrent`, `StatusSkipped`, and `StatusFailed`.
- `CheckOutcome`: domain struct encapsulating tool check state:
  - `ToolID string`
  - `ToolName string`
  - `Status CheckStatus`
  - `CurrentVersion string`
  - `LatestVersion string`
  - `UpdateAvailable bool`
  - `Err error`
  - `Stderr string`
  - `RawUpdateInfo adapters.UpdateInfo`
- `CheckProgress`: progress notification struct:
  - `Index int`
  - `Total int`
  - `Outcome CheckOutcome`
- `Filter`: query criteria struct:
  - `Only []string`
- `PlannedUpdate`: action item struct for pending updates:
  - `ToolID string`
  - `ToolName string`
  - `ManagerID string`
  - `PackageName string`
  - `UpdatePolicy adapters.UpdatePolicy`
  - `CurrentVersion string`
  - `LatestVersion string`
  - `RiskCommand string`
  - `Privileges adapters.PrivilegeLevel`
  - `Trust adapters.TrustLevel`
- `UpdatePlan`: composite execution plan struct:
  - `Updates []PlannedUpdate`
  - `Skipped []CheckOutcome`
  - `Current []CheckOutcome`
  - `Failed []CheckOutcome`

Domain models MUST NOT carry ANSI escape sequences, emoji icons, terminal table layouts, or BubbleTea model interfaces.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Outcome carries version details | Tool check succeeds with update available | `Check` produces `CheckOutcome` | `StatusAvailable`, `CurrentVersion`, and `LatestVersion` populated without UI formatting |
| Zero update info on failure | Tool check encounters error | `Check` produces `CheckOutcome` | `StatusFailed`, `Err`, and `Stderr` populated; `RawUpdateInfo` is zero-value |
| Progress event structure | Check completes for index 2 of 5 | Progress callback receives `CheckProgress` | `Index=2`, `Total=5`, and `Outcome` populated with pure domain fields |
| Composite update plan structure | Plan created from check outcomes | `Plan` produces `UpdatePlan` | `Updates`, `Skipped`, `Current`, and `Failed` segregated into typed domain slices |

### Requirement: Adapter Discovery and Resolution

The engine MUST provide `Resolve(filter Filter) ([]adapters.Adapter, error)` to discover, bind, and filter active adapters for the configured platform.

Discovery MUST proceed as follows:
1. The engine MUST enumerate built-in official adapters for the platform identified by `osName`.
2. The engine MUST enumerate custom adapters configured in `cfg.Custom`.
3. If a tool is explicitly disabled in configuration (`cfg.Tools[id].Enabled == false`), the engine MUST exclude that adapter from resolution.
4. For custom adapters declaring an owning manager (`custom.Manager != ""`), the engine MUST check if the declared manager exists as a platform official adapter declaring `KindManager`. If valid, the engine MUST bind the manager adapter as the custom adapter's owner. If invalid or not declaring `KindManager`, the custom adapter MUST remain standalone.
5. If `filter.Only` is non-empty, the engine MUST retain only adapters matching the filter list (case-insensitively by tool ID).

`Resolve` MUST return the filtered slice of adapters in deterministic canonical discovery order.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Official platform discovery | Linux host with default config | `Resolve(Filter{})` | Returns Linux official adapters in canonical order |
| Configured tool disabled | Config sets `tools.gh.enabled = false` | `Resolve(Filter{})` | `gh` adapter is excluded from resolved list |
| Custom tool manager binding | Custom tool configured with `manager = "apt"` on Linux | `Resolve(Filter{})` | `apt` official manager adapter bound as owner to custom adapter |
| Custom tool invalid manager | Custom tool configured with `manager = "invalid"` | `Resolve(Filter{})` | Custom adapter resolved as standalone without error |
| Tool filter applied | Filter specifies `Only: ["brew", "npm"]` | `Resolve(Filter{Only: []string{"brew", "npm"}})` | Returns only `brew` and `npm` adapters |
| Case-insensitive filter | Filter specifies `Only: ["BREW"]` | `Resolve(Filter{Only: []string{"BREW"}})` | Returns `brew` adapter |

### Requirement: Concurrent Check Engine

The engine MUST provide `Check(ctx context.Context, adapters []adapters.Adapter, onProgress func(CheckProgress)) ([]CheckOutcome, error)` to execute version checks concurrently across the provided adapter slice.

The check engine MUST execute with the following behaviors:
1. **Bounded Concurrency**: Concurrency MUST be limited by a worker pool whose size equals `min(numWorkers, len(adapters))`, where `numWorkers` is clamped to $[4, 8]$.
2. **Panic Containment**: Every worker goroutine MUST execute each check inside a deferred `recover()` block. If `Detect()` or `Check()` panics, the panic MUST NOT terminate the worker or engine; the engine MUST record a `CheckOutcome` with `Status: StatusFailed`, `Err: fmt.Errorf("panic during check: %v", rec)`, and a zero-value `RawUpdateInfo`.
3. **Detection Gating**: If `a.Detect()` returns `false`, the engine MUST record a `CheckOutcome` with `Status: StatusSkipped` and MUST NOT invoke `a.Check()`.
4. **Error Handling**: If `a.Check()` returns an error, the engine MUST record a `CheckOutcome` with `Status: StatusFailed`, wrapping the error with timeout context and capturing `err.Error()` in `Stderr`.
5. **Deterministic Ordering**: Check outcomes MUST be slotted into a pre-allocated slice using the adapter's input index (`outcomes[job.index] = oc`), ensuring the returned outcome slice preserves the canonical input order regardless of worker completion order.
6. **Progress Notification**: If `onProgress` is non-nil, workers MUST invoke `onProgress(CheckProgress{Index: job.index, Total: total, Outcome: oc})` upon each check completion. If `onProgress` is nil, execution MUST complete silently.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Bounded worker pool | 12 adapters passed to `Check` on 8-core CPU | `Check(ctx, adapters, nil)` | Exactly 8 worker goroutines process the jobs channel |
| Panic containment in Detect | Adapter panics during `Detect()` | `Check(ctx, adapters, nil)` | Panic caught; outcome marked `StatusFailed` with panic error message; other checks proceed |
| Panic containment in Check | Adapter panics during `Check()` | `Check(ctx, adapters, nil)` | Panic caught; outcome marked `StatusFailed` with panic error message; `RawUpdateInfo` is zero-value |
| Undetected tool skipped | Tool not installed (`Detect() == false`) | `Check(ctx, adapters, nil)` | Outcome marked `StatusSkipped`; `Check()` not called |
| Deterministic output order | Out-of-order worker completions (index 2 completes before index 0) | `Check(ctx, adapters, nil)` | Returned slice ordered strictly by input index (index 0 first, index 2 third) |
| Progress callback emission | Progress callback provided | `Check(ctx, adapters, callback)` | Callback called once per adapter with matching index and total |

### Requirement: Cooperative Context Cancellation

The check engine MUST accept a `context.Context` and observe cancellation signals cooperatively during execution.

1. Worker goroutines MUST check `ctx.Err()` before reading pending jobs and before executing adapter operations.
2. When `ctx` is canceled (e.g., via `context.Canceled` or `context.DeadlineExceeded`), worker goroutines MUST immediately stop consuming new jobs and exit their execution loops.
3. The engine MUST drain or close internal job channels and wait for all active worker goroutines to exit using a `sync.WaitGroup` before returning from `Check`.
4. The engine MUST NOT leak goroutines, leave open unbuffered channels, or deadlock upon cancellation.
5. If canceled, `Check` MUST return `ctx.Err()`. Completed outcomes slotted prior to cancellation MAY be returned alongside the error.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Immediate cancellation | Context canceled prior to calling `Check` | `Check(canceledCtx, adapters, nil)` | Returns `context.Canceled` immediately without executing checks |
| Mid-flight cancellation | 20 slow checks running; context canceled during execution | Context is canceled | Active workers finish current check; remaining queued jobs skipped; returns `context.Canceled` |
| No goroutine leaks on cancel | Worker pool active when cancellation occurs | Context is canceled and `Check` returns | All worker goroutines terminate; `sync.WaitGroup` settles cleanly |
| Deadline exceeded | Context deadline expires during checks | Timeout reached | Workers halt; `Check` returns `context.DeadlineExceeded` |

### Requirement: Update Planning

The engine MUST provide `Plan(outcomes []CheckOutcome, filter Filter) (UpdatePlan, error)` to formulate an executable update plan from check outcomes.

Planning MUST evaluate outcomes and apply update policies as follows:
1. **Outcome Categorization**:
   - Outcomes with `StatusFailed` MUST be added to `UpdatePlan.Failed` and MUST NOT be added to `UpdatePlan.Updates`.
   - Outcomes with `StatusSkipped` MUST be added to `UpdatePlan.Skipped` and MUST NOT be added to `UpdatePlan.Updates`.
2. **Policy Gating**:
   - For standalone tools declaring `PolicyGated` (e.g., apt, pacman, npm, pnpm, nvm, uv): if `UpdateAvailable == true`, the tool MUST be added to `UpdatePlan.Updates`; if `UpdateAvailable == false`, the tool MUST be added to `UpdatePlan.Current`.
   - For standalone tools declaring `PolicyAlwaysUpdate` (e.g., brew, bun, opencode, winget, scoop): the tool MUST be added to `UpdatePlan.Updates` unconditionally, regardless of whether `UpdateAvailable` is true or false.
   - For owned tools (e.g., gh, docker, go, or custom tools declaring a manager): the tool MUST inherit the effective `UpdatePolicy` of its resolved owning manager adapter. The owned tool's declared `UpdatePolicy` MUST be inert.
3. **Planned Action Formulation**:
   - Each item in `UpdatePlan.Updates` MUST be represented as a `PlannedUpdate` containing `ToolID`, `ToolName`, `ManagerID` (empty if standalone), `PackageName` (empty if standalone), effective `UpdatePolicy`, `CurrentVersion`, `LatestVersion`, `RiskCommand`, `Privileges`, and `Trust`.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| PolicyGated tool with update | `npm` outcome has `UpdateAvailable: true` | `Plan(outcomes, Filter{})` | `npm` added to `Updates` with version change |
| PolicyGated tool current | `npm` outcome has `UpdateAvailable: false` | `Plan(outcomes, Filter{})` | `npm` added to `Current`; excluded from `Updates` |
| PolicyAlwaysUpdate tool without update | `brew` outcome has `UpdateAvailable: false` | `Plan(outcomes, Filter{})` | `brew` added to `Updates` (unconditional update policy) |
| Owned tool inherits AlwaysUpdate | `gh` owned by `brew` on macOS | `Plan(outcomes, Filter{})` | `gh` inherits `PolicyAlwaysUpdate` from `brew` and is added to `Updates` |
| Owned tool inherits Gated current | `gh` owned by `apt` on Linux with `UpdateAvailable: false` | `Plan(outcomes, Filter{})` | `gh` inherits `PolicyGated` from `apt` and is added to `Current` |
| Failed outcome excluded | Tool outcome has `StatusFailed` | `Plan(outcomes, Filter{})` | Tool added to `Failed`; excluded from `Updates` |
| Skipped outcome excluded | Tool outcome has `StatusSkipped` | `Plan(outcomes, Filter{})` | Tool added to `Skipped`; excluded from `Updates` |
