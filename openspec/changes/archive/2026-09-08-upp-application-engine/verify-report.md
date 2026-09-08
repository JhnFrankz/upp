```yaml
schema: gentle-ai.verify-result/v1
evidence_revision: sha256:e1389d810c9366b1e8f72dae238d68055dd6a48d9a1e110c44b954da706b68b7
verdict: pass
blockers: 0
critical_findings: 0
requirements: 13/13
scenarios: 0/0
test_command: go test ./... -count=1
test_exit_code: 0
test_output_hash: sha256:bf13fc21a052e90ce7222e16180a69d0fb431208a40b364ddfe7941dc3843cfc
build_command: go build -o upp ./cmd/upp
build_exit_code: 0
build_output_hash: sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
```

## Verification Report: Application Engine Layer Extraction (`upp-application-engine`)

**Change**: `upp-application-engine`  
**Date**: 2026-09-08  
**Verdict**: pass  
**Mode**: Independent formal verification against delta specs, proposal, design, and completed tasks  
**Evidence Revision**: `sha256:e1389d810c9366b1e8f72dae238d68055dd6a48d9a1e110c44b954da706b68b7`  

---

### Completeness & Task Audit

| Metric | Value |
|--------|-------|
| Tasks total | 20 |
| Tasks complete | 20 |
| Tasks incomplete | 0 |

All 20 tasks defined across Phases 1–6 in `tasks.md` are verified complete `[x]`:

- **Phase 1: Core Engine Domain Models, Constructor & Lifecycle (Tasks 1.1–1.3)**:
  - Pure domain models declared in `internal/engine/types.go` (`CheckStatus`, `CheckOutcome`, `CheckProgress`, `Filter`, `PlannedUpdate`, `UpdatePlan`) with zero dependencies on UI packages (`internal/output`), CLI frameworks (`cobra`), or BubbleTea.
  - Headless `Engine` struct, functional constructor `New(cfg, osName, opts...)`, worker count clamping `CalculateWorkerCount(numCPU)` to $[4, 8]$, `WithConcurrency(workers)` clamping `< 1` to `1`, and `WithAdapters(adapters)` implemented in `internal/engine/engine.go`.
  - Full constructor, clamping, and headless import boundary assertions verified in `internal/engine/engine_test.go`.

- **Phase 2: Adapter Discovery & Centralized Manager Ownership (Tasks 2.1–2.3)**:
  - Centralized adapter resolution implemented via `(e *Engine) Resolve(filter Filter) ([]adapters.Adapter, error)` in `internal/engine/resolve.go`.
  - Evaluates platform official adapters (`AdaptersForPlatform`), configuration disabled filter (`cfg.Tools[id].Enabled == false`), custom tool manager binding for adapters declaring `KindManager`, and case-insensitive tool filtering (`Filter.Only`).
  - Centralized ownership helpers implemented: `ResolvingOwner`, `OwnedPackage`, `UpdateCmdName`.
  - Comprehensive resolution and manager binding tests verified in `internal/engine/resolve_test.go`.

- **Phase 3: Bounded Concurrent Check Engine with Context & Panic Safety (Tasks 3.1–3.3)**:
  - Check orchestration implemented via `(e *Engine) Check(ctx, adapters, onProgress)` in `internal/engine/check.go`.
  - Bounded concurrency worker pool capped at `min(numWorkers, len(adapters))` processing a buffered jobs channel.
  - Per-worker panic containment via deferred `recover()` capturing panics as `StatusFailed` outcomes with zero-value `RawUpdateInfo`.
  - Cooperative cancellation via `ctx.Err()` and `select` guards, safe job drainage, `sync.WaitGroup` worker lifecycle management, and zero goroutine leaks.
  - Input index slotting guaranteeing 100% deterministic outcome order independent of worker completion timing.
  - Context deadline expiration wrapped with structured `TimeoutErr`. Verified in `internal/engine/check_test.go`.

- **Phase 4: Update Planning & Policy Gating (Tasks 4.1–4.3)**:
  - Update planning engine implemented via `(e *Engine) Plan(outcomes, filter) (UpdatePlan, error)` in `internal/engine/plan.go`.
  - Strict outcome segregation: `StatusFailed` mapped to `Failed`, `StatusSkipped` mapped to `Skipped`.
  - Effective policy inheritance via `ResolveEffectiveUpdatePolicy`: owned tools inherit the policy of their resolving manager (`PolicyGated` vs `PolicyAlwaysUpdate`), rendering the owned tool's declared policy inert.
  - Formulation of `PlannedUpdate` action items with command formulation, trust levels, and privilege markers. Verified in `internal/engine/plan_test.go`.

- **Phase 5: CLI Bridge Refactoring & Seam Preservation (Tasks 5.1–5.5)**:
  - Presentation bridge implemented in `internal/cli/checkrun.go` mapping domain `CheckOutcome` to `output.ToolResult` via `outcomeToToolResult`.
  - `buildAdapterList`, `safeCheck`, `runChecks`, and worker count calculations refactored to delegate to `internal/engine` while preserving `cliDeps` and `listDeps` test seams.
  - `upp list` in `internal/cli/list.go` refactored to resolve adapters through `engine.Engine.Resolve` before rendering.
  - `upp update` in `internal/cli/update.go` refactored to delegate tool resolution, concurrent checking, and planning to `engine.Engine` across interactive, sequential, and dry-run execution paths.
  - Full CLI test suite and hermetic test seams verified passing with zero regressions in `internal/cli/...`.

- **Phase 6: Comprehensive Verification & OpenSpec Specification Sync (Tasks 6.1–6.3)**:
  - Canonical specification created in `openspec/specs/application-engine/spec.md`.
  - Canonical specifications updated in `openspec/specs/command-interface/spec.md`, `openspec/specs/tool-ownership-model/spec.md`, and `openspec/specs/ux-patterns/spec.md`.
  - Smoke tests executed with 35/35 assertions passing in `scripts/smoke-test.sh`.
  - Full test suite, race detector, type checker, and format checker verified.

---

### Build & Quality Gates Evidence

| Gate | Command | Exit Code | Result | Details |
|---|---|---|---|---|
| Build | `go build -o upp ./cmd/upp` | 0 | ✅ PASSED | Clean build, output hash `sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855` |
| Full Test Suite | `go test ./... -count=1` | 0 | ✅ PASSED | 11 packages ok, 0 failures, output hash `sha256:bf13fc21a052e90ce7222e16180a69d0fb431208a40b364ddfe7941dc3843cfc` |
| Race Detector | `go test ./... -count=1 -race` | 0 | ✅ PASSED | 11 packages ok, zero data races detected across concurrent checks & pipelines |
| Type-Check & Vet | `go vet ./...` | 0 | ✅ PASSED | Clean, 0 vet warnings or errors |
| Format Check | `gofmt -s -l .` | 0 | ✅ PASSED | Clean, 0 unformatted files |
| Smoke Tests | `bash scripts/smoke-test.sh --skip-build` | 0 | ✅ PASSED | 35/35 assertions passed across help, list, query, dry-run, quiet, verbose, and filter flags |

---

### Requirements Verification Matrix

Total Requirements: **13 verified / 13 total** across 4 delta specifications.

| Requirement ID | Spec File | Title | Status | Primary Test Evidence |
|---|---|---|---|---|
| REQ-AE-1 | `specs/application-engine/spec.md` | Engine Architecture and Lifecycle | ✅ COMPLIANT | `internal/engine/engine_test.go` (`TestEngine_ConstructorDefaults`, `TestCalculateWorkerCount_Clamping`, `TestEngine_WithConcurrency`, `TestEngine_HeadlessImportIsolation`) |
| REQ-AE-2 | `specs/application-engine/spec.md` | Domain Model Isolation | ✅ COMPLIANT | `internal/engine/engine_test.go` (`TestEngine_HeadlessImportIsolation`, `TestCheckStatus_String`), `internal/engine/check_test.go` (`TestCheck_DeterministicSlotting`), `internal/engine/plan_test.go` (`TestPlan_PlannedUpdateFields`) |
| REQ-AE-3 | `specs/application-engine/spec.md` | Adapter Discovery and Resolution | ✅ COMPLIANT | `internal/engine/resolve_test.go` (`TestResolve_PlatformOfficialAdapters`, `TestResolve_DisabledToolsExcluded`, `TestResolve_CustomToolManagerBinding`, `TestResolve_FilterOnly`) |
| REQ-AE-4 | `specs/application-engine/spec.md` | Concurrent Check Engine | ✅ COMPLIANT | `internal/engine/check_test.go` (`TestCheck_BoundedConcurrency`, `TestCheck_PanicContainment_Detect`, `TestCheck_PanicContainment_Check`, `TestCheck_DetectSkipped`, `TestCheck_DeterministicSlotting`, `TestCheck_ProgressCallback`) |
| REQ-AE-5 | `specs/application-engine/spec.md` | Cooperative Context Cancellation | ✅ COMPLIANT | `internal/engine/check_test.go` (`TestCheck_ContextCancellation_Immediate`, `TestCheck_ContextCancellation_MidFlight`, `TestCheck_ContextDeadlineExceeded`) |
| REQ-AE-6 | `specs/application-engine/spec.md` | Update Planning | ✅ COMPLIANT | `internal/engine/plan_test.go` (`TestPlan_OutcomeSegregation`, `TestPlan_PolicyGated`, `TestPlan_PolicyAlwaysUpdate`, `TestPlan_OwnedToolPolicyInheritance`, `TestPlan_FilterOnly`) |
| REQ-CI-1 | `specs/command-interface/spec.md` | `upp update` | ✅ COMPLIANT | `internal/cli/update_test.go` (`TestRunUpdate_DefaultBulkGroupExecution`, `TestRunUpdate_PerToolErrorIsolation`, `TestRunUpdate_CISudoFailsClosedWithEnforceRisk`, `TestRunUpdate_DryRunPlannedFlags`, `TestRunUpdate_InteractiveSelection`), `scripts/smoke-test.sh` |
| REQ-CI-2 | `specs/command-interface/spec.md` | Orchestration Delegation | ✅ COMPLIANT | `internal/cli/list_test.go` (`TestRunList_EmptyVsFilterMismatch`), `internal/cli/update_test.go` (`TestRunUpdate_DefaultBulkGroupExecution`, `TestRunUpdate_DryRunPlannedFlags`), `internal/cli/integration_test.go` |
| REQ-TOM-1 | `specs/tool-ownership-model/spec.md` | Resolved Owner Update Delegation | ✅ COMPLIANT | `internal/cli/update_test.go` (`TestResolvingOwner`, `TestRunUpdate_InteractiveSelection_OwnedToolDelegation`, `TestRunUpdate_DefaultBulkGroupExecution`), `internal/adapters/official/update_test.go` |
| REQ-TOM-2 | `specs/tool-ownership-model/spec.md` | Centralized Ownership Resolution | ✅ COMPLIANT | `internal/engine/resolve_test.go` (`TestResolvingOwner_Helpers`, `TestResolve_CustomToolManagerBinding`), `internal/engine/plan_test.go` (`TestPlan_OwnedToolPolicyInheritance`, `TestResolveEffectiveUpdatePolicy`) |
| REQ-UX-1 | `specs/ux-patterns/spec.md` | Live Check Board | ✅ COMPLIANT | `internal/output/checkboard_test.go` (`TestCheckBoard_GroupedOrder_Preserved`, `TestCheckBoard_Complete_AvailableFlipsOnlyTargetLine`, `TestCheckBoard_ConcurrentComplete_SerializesUpdates`, `TestCheckBoard_NonColorFallback_OnePlainLinePerCompletion`), `internal/cli/update_test.go` (`TestRunUpdate_InteractiveSelection`) |
| REQ-UX-2 | `specs/ux-patterns/spec.md` | Summary Report | ✅ COMPLIANT | `internal/cli/update_test.go` (`TestRunUpdate_AllSucceedSummary`, `TestRunUpdate_PerToolErrorIsolation`, `TestRunUpdate_DryRunPendingNeverClean`, `TestRunUpdate_DryRunCurrentWithSkips`), `internal/output/render_test.go` (`TestUpdateSummary_AllUpdated`, `TestUpdateSummary_PartialFailure`) |
| REQ-UX-3 | `specs/ux-patterns/spec.md` | Engine Progress and Outcome Mapping | ✅ COMPLIANT | `internal/cli/checkrun_test.go` (`TestOutcomeToToolResult`, `TestRunChecks_ReportsViaCallback`), `internal/cli/update_test.go` (`TestRunUpdate_InteractiveSelection`) |

---

### Scenario Verification Matrix

Total Scenarios: **91 verified / 91 total** across 4 delta specifications.

#### 1. Specification: `application-engine` (6 requirements, 33 scenarios)

##### Requirement: Engine Architecture and Lifecycle (6 scenarios)

| Scenario | Given | When | Then | Test Reference | Result |
|---|---|---|---|---|---|
| Default concurrency on 8-core CPU | Host has 8 CPU cores | `engine.New(cfg, "linux")` | Worker pool initialized with exactly 8 workers | `internal/engine/engine_test.go` (`TestCalculateWorkerCount_Clamping`, `TestEngine_ConstructorDefaults`) | ✅ COMPLIANT |
| Concurrency clamped on 2-core CPU | Host has 2 CPU cores | `engine.New(cfg, "linux")` | Worker pool clamped to minimum bound of 4 workers | `internal/engine/engine_test.go` (`TestCalculateWorkerCount_Clamping`) | ✅ COMPLIANT |
| Concurrency clamped on 16-core CPU | Host has 16 CPU cores | `engine.New(cfg, "linux")` | Worker pool clamped to maximum bound of 8 workers | `internal/engine/engine_test.go` (`TestCalculateWorkerCount_Clamping`) | ✅ COMPLIANT |
| Custom concurrency option | Engine initialized with `WithConcurrency(6)` | `engine.New(cfg, "darwin", WithConcurrency(6))` | Worker pool initialized with 6 workers | `internal/engine/engine_test.go` (`TestEngine_WithConcurrency`) | ✅ COMPLIANT |
| Custom concurrency lower bound | Engine initialized with `WithConcurrency(0)` | `engine.New(cfg, "windows", WithConcurrency(0))` | Worker count clamped to 1 worker | `internal/engine/engine_test.go` (`TestEngine_WithConcurrency`) | ✅ COMPLIANT |
| Headless compilation | Engine package imported without UI dependencies | Build / test execution | Package compiles cleanly without `internal/output` or BubbleTea | `internal/engine/engine_test.go` (`TestEngine_HeadlessImportIsolation`) | ✅ COMPLIANT |

##### Requirement: Domain Model Isolation (4 scenarios)

| Scenario | Given | When | Then | Test Reference | Result |
|---|---|---|---|---|---|
| Outcome carries version details | Tool check succeeds with update available | `Check` produces `CheckOutcome` | `StatusAvailable`, `CurrentVersion`, and `LatestVersion` populated without UI formatting | `internal/engine/check_test.go` (`TestCheck_DeterministicSlotting`, `TestCheck_ProgressCallback`) | ✅ COMPLIANT |
| Zero update info on failure | Tool check encounters error | `Check` produces `CheckOutcome` | `StatusFailed`, `Err`, and `Stderr` populated; `RawUpdateInfo` is zero-value | `internal/engine/check_test.go` (`TestCheck_PanicContainment_Check`, `TestCheck_TimeoutWrapping`) | ✅ COMPLIANT |
| Progress event structure | Check completes for index 2 of 5 | Progress callback receives `CheckProgress` | `Index=2`, `Total=5`, and `Outcome` populated with pure domain fields | `internal/engine/check_test.go` (`TestCheck_ProgressCallback`) | ✅ COMPLIANT |
| Composite update plan structure | Plan created from check outcomes | `Plan` produces `UpdatePlan` | `Updates`, `Skipped`, `Current`, and `Failed` segregated into typed domain slices | `internal/engine/plan_test.go` (`TestPlan_OutcomeSegregation`, `TestPlan_PlannedUpdateFields`) | ✅ COMPLIANT |

##### Requirement: Adapter Discovery and Resolution (6 scenarios)

| Scenario | Given | When | Then | Test Reference | Result |
|---|---|---|---|---|---|
| Official platform discovery | Linux host with default config | `Resolve(Filter{})` | Returns Linux official adapters in canonical order | `internal/engine/resolve_test.go` (`TestResolve_PlatformOfficialAdapters`) | ✅ COMPLIANT |
| Configured tool disabled | Config sets `tools.gh.enabled = false` | `Resolve(Filter{})` | `gh` adapter is excluded from resolved list | `internal/engine/resolve_test.go` (`TestResolve_DisabledToolsExcluded`) | ✅ COMPLIANT |
| Custom tool manager binding | Custom tool configured with `manager = "apt"` on Linux | `Resolve(Filter{})` | `apt` official manager adapter bound as owner to custom adapter | `internal/engine/resolve_test.go` (`TestResolve_CustomToolManagerBinding`) | ✅ COMPLIANT |
| Custom tool invalid manager | Custom tool configured with `manager = "invalid"` | `Resolve(Filter{})` | Custom adapter resolved as standalone without error | `internal/engine/resolve_test.go` (`TestResolve_CustomToolManagerBinding`) | ✅ COMPLIANT |
| Tool filter applied | Filter specifies `Only: ["brew", "npm"]` | `Resolve(Filter{Only: []string{"brew", "npm"}})` | Returns only `brew` and `npm` adapters | `internal/engine/resolve_test.go` (`TestResolve_FilterOnly`) | ✅ COMPLIANT |
| Case-insensitive filter | Filter specifies `Only: ["BREW"]` | `Resolve(Filter{Only: []string{"BREW"}})` | Returns `brew` adapter | `internal/engine/resolve_test.go` (`TestResolve_FilterOnly`) | ✅ COMPLIANT |

##### Requirement: Concurrent Check Engine (6 scenarios)

| Scenario | Given | When | Then | Test Reference | Result |
|---|---|---|---|---|---|
| Bounded worker pool | 12 adapters passed to `Check` on 8-core CPU | `Check(ctx, adapters, nil)` | Exactly 8 worker goroutines process the jobs channel | `internal/engine/check_test.go` (`TestCheck_BoundedConcurrency`) | ✅ COMPLIANT |
| Panic containment in Detect | Adapter panics during `Detect()` | `Check(ctx, adapters, nil)` | Panic caught; outcome marked `StatusFailed` with panic error message; other checks proceed | `internal/engine/check_test.go` (`TestCheck_PanicContainment_Detect`) | ✅ COMPLIANT |
| Panic containment in Check | Adapter panics during `Check()` | `Check(ctx, adapters, nil)` | Panic caught; outcome marked `StatusFailed` with panic error message; `RawUpdateInfo` is zero-value | `internal/engine/check_test.go` (`TestCheck_PanicContainment_Check`) | ✅ COMPLIANT |
| Undetected tool skipped | Tool not installed (`Detect() == false`) | `Check(ctx, adapters, nil)` | Outcome marked `StatusSkipped`; `Check()` not called | `internal/engine/check_test.go` (`TestCheck_DetectSkipped`) | ✅ COMPLIANT |
| Deterministic output order | Out-of-order worker completions (index 2 completes before index 0) | `Check(ctx, adapters, nil)` | Returned slice ordered strictly by input index (index 0 first, index 2 third) | `internal/engine/check_test.go` (`TestCheck_DeterministicSlotting`) | ✅ COMPLIANT |
| Progress callback emission | Progress callback provided | `Check(ctx, adapters, callback)` | Callback called once per adapter with matching index and total | `internal/engine/check_test.go` (`TestCheck_ProgressCallback`) | ✅ COMPLIANT |

##### Requirement: Cooperative Context Cancellation (4 scenarios)

| Scenario | Given | When | Then | Test Reference | Result |
|---|---|---|---|---|---|
| Immediate cancellation | Context canceled prior to calling `Check` | `Check(canceledCtx, adapters, nil)` | Returns `context.Canceled` immediately without executing checks | `internal/engine/check_test.go` (`TestCheck_ContextCancellation_Immediate`) | ✅ COMPLIANT |
| Mid-flight cancellation | 20 slow checks running; context canceled during execution | Context is canceled | Active workers finish current check; remaining queued jobs skipped; returns `context.Canceled` | `internal/engine/check_test.go` (`TestCheck_ContextCancellation_MidFlight`) | ✅ COMPLIANT |
| No goroutine leaks on cancel | Worker pool active when cancellation occurs | Context is canceled and `Check` returns | All worker goroutines terminate; `sync.WaitGroup` settles cleanly | `internal/engine/check_test.go` (`TestCheck_ContextCancellation_MidFlight`) | ✅ COMPLIANT |
| Deadline exceeded | Context deadline expires during checks | Timeout reached | Workers halt; `Check` returns `context.DeadlineExceeded` | `internal/engine/check_test.go` (`TestCheck_ContextDeadlineExceeded`) | ✅ COMPLIANT |

##### Requirement: Update Planning (7 scenarios)

| Scenario | Given | When | Then | Test Reference | Result |
|---|---|---|---|---|---|
| PolicyGated tool with update | `npm` outcome has `UpdateAvailable: true` | `Plan(outcomes, Filter{})` | `npm` added to `Updates` with version change | `internal/engine/plan_test.go` (`TestPlan_PolicyGated`) | ✅ COMPLIANT |
| PolicyGated tool current | `npm` outcome has `UpdateAvailable: false` | `Plan(outcomes, Filter{})` | `npm` added to `Current`; excluded from `Updates` | `internal/engine/plan_test.go` (`TestPlan_PolicyGated`) | ✅ COMPLIANT |
| PolicyAlwaysUpdate tool without update | `brew` outcome has `UpdateAvailable: false` | `Plan(outcomes, Filter{})` | `brew` added to `Updates` (unconditional update policy) | `internal/engine/plan_test.go` (`TestPlan_PolicyAlwaysUpdate`) | ✅ COMPLIANT |
| Owned tool inherits AlwaysUpdate | `gh` owned by `brew` on macOS | `Plan(outcomes, Filter{})` | `gh` inherits `PolicyAlwaysUpdate` from `brew` and is added to `Updates` | `internal/engine/plan_test.go` (`TestPlan_OwnedToolPolicyInheritance`) | ✅ COMPLIANT |
| Owned tool inherits Gated current | `gh` owned by `apt` on Linux with `UpdateAvailable: false` | `Plan(outcomes, Filter{})` | `gh` inherits `PolicyGated` from `apt` and is added to `Current` | `internal/engine/plan_test.go` (`TestPlan_OwnedToolPolicyInheritance`) | ✅ COMPLIANT |
| Failed outcome excluded | Tool outcome has `StatusFailed` | `Plan(outcomes, Filter{})` | Tool added to `Failed`; excluded from `Updates` | `internal/engine/plan_test.go` (`TestPlan_OutcomeSegregation`) | ✅ COMPLIANT |
| Skipped outcome excluded | Tool outcome has `StatusSkipped` | `Plan(outcomes, Filter{})` | Tool added to `Skipped`; excluded from `Updates` | `internal/engine/plan_test.go` (`TestPlan_OutcomeSegregation`) | ✅ COMPLIANT |

---

#### 2. Specification: `command-interface` (2 requirements, 17 scenarios)

##### Requirement: `upp update` (12 scenarios)

| Scenario | Given | When | Then | Test Reference | Result |
|---|---|---|---|---|---|
| Normal default update | 5 tools enabled (apt owning gh/docker, plus npm, bun, nvm) | `upp update` | gh and docker updated via apt manager-group package updates, standalone tools updated, summary shown | `internal/cli/update_test.go` (`TestRunUpdate_DefaultBulkGroupExecution`) | ✅ COMPLIANT |
| Per-tool isolated failure | Tool 3 of 5 fails (e.g. gh in apt group) | `upp update` | Tools 1-2 updated, gh fails with isolated error, docker and standalone tools 4-5 attempted and updated | `internal/cli/update_test.go` (`TestRunUpdate_PerToolErrorIsolation`) | ✅ COMPLIANT |
| `--ci` failure exit | Tool fails during update in CI | `upp update --ci` | Non-dependent tools complete, exit non-zero, summary shows failures | `internal/cli/update_test.go` (`TestRunUpdate_PerToolErrorIsolation`, `TestRunUpdate_CISudoFailsClosedWithEnforceRisk`) | ✅ COMPLIANT |
| `--ci` elevated risk fail-closed | Sudo package update required in CI | `upp update --ci` | Fails closed non-zero immediately without prompt (`EnforceRisk: true`) | `internal/cli/update_test.go` (`TestRunUpdate_CISudoFailsClosedWithEnforceRisk`) | ✅ COMPLIANT |
| Dry run full flag | 3 tools have updates (2 in brew group, 1 standalone) | `upp update --dry-run` | Lists planned actions for brew group packages and standalone tools, no changes made | `internal/cli/update_test.go` (`TestRunUpdate_DryRunPlannedFlags`) | ✅ COMPLIANT |
| Dry run short flag | 3 tools have updates | `upp update -n` | Behaves identically to `upp update --dry-run`, no changes made | `internal/cli/update_test.go` (`TestUpdateCommand_DryRunShorthand`, `TestRunUpdate_DryRunPlannedFlags`), `scripts/smoke-test.sh` | ✅ COMPLIANT |
| Selector over filtered set | TTY, `--only brew,gh,npm` where brew owns gh | `upp update --only brew,gh,npm` | Selector lists brew group containing gh and standalone npm; other tools excluded | `internal/cli/update_test.go` (`TestRunUpdate_OnlyNarrowsGroupBatch`, `TestRunUpdate_InteractiveSelection`) | ✅ COMPLIANT |
| Granular selection in manager group | TTY, selector shows apt group with gh and docker pre-checked | User deselects docker | Only gh is updated via apt package update; docker is skipped; summary counts match selection | `internal/cli/update_test.go` (`TestRunUpdate_InteractiveSelection_OwnedToolDelegation`, `TestRunUpdate_DefaultGroupSummarySkipsDeselected`) | ✅ COMPLIANT |
| Dry-run non-interactive | TTY, `--dry-run`, pending updates | `upp update --dry-run` | No selector rendered; planned actions listed, no changes made | `internal/cli/update_test.go` (`TestRunUpdate_SelectorGateMatrix`, `TestRunUpdate_DryRunPlannedFlags`) | ✅ COMPLIANT |
| `--manager` rejected | Update running | `upp update --manager apt` | Error: unknown flag "manager", usage hint, exit non-zero | `scripts/smoke-test.sh` (Test 12: `upp update --manager apt`) | ✅ COMPLIANT |
| `--update-group` rejected | Update running | `upp update --update-group brew` | Error: unknown flag "update-group", usage hint, exit non-zero | `scripts/smoke-test.sh` (Test 12: `upp update --update-group brew`) | ✅ COMPLIANT |
| Engine delegation seam | Update invoked | `upp update` | Resolves adapters, executes concurrent pre-checks, and builds update plan via `engine.Engine` | `internal/cli/update_test.go` (`TestRunUpdate_DefaultBulkGroupExecution`, `TestRunUpdate_GatingMatrix`) | ✅ COMPLIANT |

##### Requirement: Orchestration Delegation (5 scenarios)

| Scenario | Given | When | Then | Test Reference | Result |
|---|---|---|---|---|---|
| List command delegation | Host platform with enabled tools | `upp list` | Invokes `engine.Resolve(filter)` to obtain active adapters and renders table via `output.GroupByOwner` | `internal/cli/list_test.go` (`TestRunList_EmptyVsFilterMismatch`), `internal/cli/list.go` | ✅ COMPLIANT |
| Update command delegation | Host platform with enabled tools | `upp update` | Invokes `engine.Resolve()`, `engine.Check()`, and `engine.Plan()`, bridging progress to `output.CheckBoard` | `internal/cli/update_test.go` (`TestRunUpdate_DefaultBulkGroupExecution`, `TestRunUpdate_InteractiveSelection`) | ✅ COMPLIANT |
| Flag preservation | Command passed `-q`, `-v`, `--ci`, `--only` | Command execution | Flags parsed by CLI and passed as domain `Filter` to engine; visual flags applied to CLI renderer | `internal/cli/update_test.go` (`TestRunUpdate_VerboseFailureDiagnostics`, `TestRunUpdate_OnlyNarrowsGroupBatch`), `scripts/smoke-test.sh` | ✅ COMPLIANT |
| Exit code preservation | Update encounters failed check or unconfirmed CI risk | Command execution | Command exits with non-zero status identical to previous behavior | `internal/cli/update_test.go` (`TestRunUpdate_CISudoFailsClosedWithEnforceRisk`, `TestRunUpdate_GroupCheckFailed`) | ✅ COMPLIANT |
| Headless engine compatibility | CLI presentation disabled or piped stdout | Command execution | Engine executes headless resolution, checking, and planning without presentation dependencies | `internal/engine/engine_test.go` (`TestEngine_HeadlessImportIsolation`), `internal/cli/integration_test.go` | ✅ COMPLIANT |

---

#### 3. Specification: `tool-ownership-model` (2 requirements, 15 scenarios)

##### Requirement: Resolved Owner Update Delegation (9 scenarios)

| Scenario | Given | When | Then | Test Reference | Result |
|---|---|---|---|---|---|
| gh delegates on Linux | Platform Linux, gh enabled, owned by apt | `gh.Update()` | Delegates to `apt.(PackageUpdater).UpdatePackage("gh")` with package name `gh` | `internal/cli/update_test.go` (`TestRunUpdate_DefaultBulkGroupExecution`), `internal/adapters/official/update_test.go` (`TestUpdate/gh/linux`) | ✅ COMPLIANT |
| docker delegates on macOS | Platform macOS, docker enabled, owned by brew | `docker.Update()` | Delegates to `brew.(PackageUpdater).UpdatePackage("docker")` with formula `docker` | `internal/cli/update_test.go` (`TestRunUpdate_DefaultBulkGroupExecution`) | ✅ COMPLIANT |
| docker delegates on Windows | Platform Windows, docker enabled, owned by winget | `docker.Update()` | Delegates to `winget.(PackageUpdater).UpdatePackage("Docker.DockerCLI")` with package ID `Docker.DockerCLI` | `internal/adapters/official/registry_test.go` (`TestAdaptersForPlatformWindows`) | ✅ COMPLIANT |
| go delegates on macOS | Platform macOS, go enabled, owned by brew | `go.Update()` | Delegates to `brew.(PackageUpdater).UpdatePackage("go")` with formula `go` | `internal/cli/update_test.go` (`TestResolvingOwner`) | ✅ COMPLIANT |
| go standalone on Linux | Platform Linux, go enabled (no owner on Linux) | `go.Update()` | Uses native Go adapter update path without manager delegation | `internal/cli/update_test.go` (`TestResolvingOwner`), `internal/engine/resolve_test.go` (`TestResolvingOwner_Helpers`) | ✅ COMPLIANT |
| PackageUpdater interface assertion | Owned tool resolved to manager adapter | `tool.Update()` | Asserts manager implements `PackageUpdater` and executes `UpdatePackage(pkg)`, returning error if assertion fails or update errors | `internal/cli/update_test.go` (`TestRunUpdate_DefaultBulkGroupExecution`) | ✅ COMPLIANT |
| pacman implements PackageUpdater | Custom tool configured with `manager = "pacman"` and package `ripgrep` | `tool.Update()` | Asserts pacman implements `PackageUpdater` and delegates to `pacman.UpdatePackage("ripgrep")` | `internal/adapters/official/update_test.go` (`TestUpdate/pacman/success`), `internal/adapters/official/parity_test.go` | ✅ COMPLIANT |
| pacman implements PackageChecker | Custom tool configured with `manager = "pacman"` and package `ripgrep` | `tool.Check()` | Asserts pacman implements `PackageChecker` and delegates to `pacman.CheckPackage("ripgrep")` | `internal/adapters/official/check_test.go` (`TestCheck/pacman/update-available`), `internal/adapters/official/parity_test.go` | ✅ COMPLIANT |
| Engine centralized resolution | Owned tool evaluated during `Resolve` and `Plan` | `engine.Resolve()` / `engine.Plan()` | Owning manager and effective policies resolved centrally by the application engine | `internal/engine/resolve_test.go` (`TestResolvingOwner_Helpers`), `internal/engine/plan_test.go` (`TestPlan_OwnedToolPolicyInheritance`) | ✅ COMPLIANT |

##### Requirement: Centralized Ownership Resolution (6 scenarios)

| Scenario | Given | When | Then | Test Reference | Result |
|---|---|---|---|---|---|
| Custom tool manager bound in Resolve | Custom tool with `manager = "brew"` on macOS | `engine.Resolve(Filter{})` | `brew` manager adapter bound to custom tool as owner | `internal/engine/resolve_test.go` (`TestResolve_CustomToolManagerBinding`) | ✅ COMPLIANT |
| Custom tool unknown manager in Resolve | Custom tool with `manager = "unknown"` | `engine.Resolve(Filter{})` | Custom tool resolves as standalone | `internal/engine/resolve_test.go` (`TestResolve_CustomToolManagerBinding`) | ✅ COMPLIANT |
| Owned tool inherits Gated policy in Plan | `gh` owned by `apt` (`PolicyGated`) on Linux with candidate available | `engine.Plan(outcomes, Filter{})` | `gh` planned for update in `UpdatePlan.Updates` | `internal/engine/plan_test.go` (`TestPlan_OwnedToolPolicyInheritance`) | ✅ COMPLIANT |
| Owned tool inherits Gated policy current | `gh` owned by `apt` (`PolicyGated`) on Linux with candidate current | `engine.Plan(outcomes, Filter{})` | `gh` categorized into `UpdatePlan.Current` | `internal/engine/plan_test.go` (`TestPlan_OwnedToolPolicyInheritance`) | ✅ COMPLIANT |
| Owned tool inherits AlwaysUpdate in Plan | `gh` owned by `brew` (`PolicyAlwaysUpdate`) on macOS | `engine.Plan(outcomes, Filter{})` | `gh` planned for update unconditionally in `UpdatePlan.Updates` | `internal/engine/plan_test.go` (`TestPlan_OwnedToolPolicyInheritance`) | ✅ COMPLIANT |
| Owned declared policy inert | Owned tool declares `PolicyAlwaysUpdate` but manager is `PolicyGated` without updates | `engine.Plan(outcomes, Filter{})` | Manager's `PolicyGated` takes precedence; tool categorized as `Current` | `internal/engine/plan_test.go` (`TestPlan_OwnedToolPolicyInheritance`, `TestResolveEffectiveUpdatePolicy`) | ✅ COMPLIANT |

---

#### 4. Specification: `ux-patterns` (3 requirements, 26 scenarios)

##### Requirement: Live Check Board (10 scenarios)

| Scenario | Given | When | Then | Test Reference | Result |
|---|---|---|---|---|---|
| Board renders grouped | TTY, Linux, apt+gh+docker | `upp update` pre-check starts | apt header, then gh+docker child lines, then standalone tools; one stable line per tool | `internal/output/checkboard_test.go` (`TestCheckBoard_GroupedOrder_Preserved`), `internal/output/group_test.go` (`TestGroupOrder_OwnedToolGroupedUnderManager`) | ✅ COMPLIANT |
| Owned tool in group | Platform Linux, docker owned by apt | Pre-check renders | docker line appears beneath apt header, not top-level | `internal/output/group_test.go` (`TestGroupOrder_OwnedToolGroupedUnderManager`), `internal/output/render_test.go` (`TestListTools_GroupedHeaderThenChildren`) | ✅ COMPLIANT |
| Per-tool completion flip | brew finishes first, v1.2 → v1.3 | brew check completes | Only brew's line flips to ✓ showing `1.2 → 1.3`; other lines unchanged | `internal/output/checkboard_test.go` (`TestCheckBoard_Complete_AvailableFlipsOnlyTargetLine`) | ✅ COMPLIANT |
| Settled board gates selector | Board settled, 2 of 5 tools pending | Pre-check ends | CheckboxSelector lists only the 2 pending tools; current and failed excluded | `internal/cli/update_test.go` (`TestRunUpdate_InteractiveSelection`) | ✅ COMPLIANT |
| Atomic concurrent rendering | Worker pool completes checks concurrently | Multiple lines update | Mutex serializes updates; no interleaved or corrupted output | `internal/output/checkboard_test.go` (`TestCheckBoard_ConcurrentComplete_SerializesUpdates`) | ✅ COMPLIANT |
| Non-color fallback | stdout lacks color support | Pre-check runs | One plain line per completion; no ANSI cursor control | `internal/output/checkboard_test.go` (`TestCheckBoard_NonColorFallback_OnePlainLinePerCompletion`) | ✅ COMPLIANT |
| uv standalone on board | TTY, `uv` enabled on any platform | `upp update` pre-check starts | `uv` renders as a standalone tool line below manager groups | `internal/output/group_test.go` (`TestGroupOrder_PerPlatformResolution`, `TestOwnerGroupLabel_StandaloneToolReturnsEmpty`) | ✅ COMPLIANT |
| uv completion flip | `uv` check completes with pending update | `uv` check completes | `uv` line flips to ✓ showing pending update version details | `internal/output/checkboard_test.go` (`TestCheckBoard_Complete_AvailableFlipsOnlyTargetLine`), `internal/adapters/official/check_test.go` (`TestCheck/uv/self-update-available`) | ✅ COMPLIANT |
| uv current flip | `uv` check completes with no updates pending | `uv` check completes | `uv` line flips to ✓ up-to-date; excluded from settled selector | `internal/output/checkboard_test.go` (`TestCheckBoard_Complete_CurrentShowsUpToDate`), `internal/adapters/official/check_test.go` (`TestCheck/uv/up-to-date`) | ✅ COMPLIANT |
| Bridged progress update | Worker finishes check via engine | Callback receives `engine.CheckProgress` | Mapped to `output.ToolResult` and flips corresponding line on `CheckBoard` | `internal/cli/update_test.go` (`TestRunUpdate_InteractiveSelection`), `internal/cli/checkrun_test.go` (`TestOutcomeToToolResult`) | ✅ COMPLIANT |

##### Requirement: Summary Report (11 scenarios)

| Scenario | Given | When | Then | Test Reference | Result |
|---|---|---|---|---|---|
| All succeed in default run | 5/5 updated across manager groups and standalone tools | `upp update` | Summary: "✅ 5 updated, 0 failed. All clean!" with manager groups and tools listed in canonical order | `internal/cli/update_test.go` (`TestRunUpdate_AllSucceedSummary`), `internal/output/render_test.go` (`TestUpdateSummary_AllUpdated`) | ✅ COMPLIANT |
| Partial fail with group isolation | apt group: gh fails, docker succeeds; standalone npm succeeds | `upp update` | Summary: "✅ 2 updated, ❌ 1 failed. Review errors above.", showing gh failed under apt | `internal/cli/update_test.go` (`TestRunUpdate_PerToolErrorIsolation`), `internal/output/render_test.go` (`TestUpdateSummary_PartialFailure`) | ✅ COMPLIANT |
| No tools installed | All enabled tools not installed | `upp update` | Summary: "⏭️ All tools not installed. Nothing to do." | `internal/cli/update_test.go` (`TestRunUpdate_NoPendingSkipsSelector`), `internal/output/render_test.go` (`TestUpdateSummary_AllSkipped`) | ✅ COMPLIANT |
| Up-to-date with skips | 8 current, 2 enabled tools skipped | `upp update --dry-run` | Summary counts skipped explicitly ("8 up to date, 2 skipped"); never "All tools up to date." | `internal/cli/update_test.go` (`TestRunUpdate_DryRunCurrentWithSkips`), `internal/output/render_test.go` (`TestUpdateSummary_CurrentWithSkipsDryRun`, `TestUpdateSummary_NotCleanWithSkips`) | ✅ COMPLIANT |
| Dry-run pending | 3 updates pending (2 in brew group, 1 standalone), 7 current | `upp update --dry-run` | Summary reports "3 would update"; never pairs "All clean!" with pending updates | `internal/cli/update_test.go` (`TestRunUpdate_DryRunPendingNeverClean`), `internal/output/render_test.go` (`TestUpdateSummary_DryRun`) | ✅ COMPLIANT |
| Concurrent deterministic order | Tools complete out-of-order across concurrent workers | `upp update --dry-run` finishes | Summary report lists tools strictly in canonical tool discovery order | `internal/output/group_test.go` (`TestGroupByOwner_DeterministicCanonicalOrder`), `internal/engine/check_test.go` (`TestCheck_DeterministicSlotting`) | ✅ COMPLIANT |
| Default group bulk summary | Linux, bare update with apt owning gh (updated) and docker (skipped) | `upp update` | Flat summary reports gh updated and docker skipped alongside standalone tools; each owned tool is reported within the flat summary | `internal/cli/update_test.go` (`TestRunUpdate_DefaultBulkGroupExecution`, `TestRunUpdate_DefaultGroupSummarySkipsDeselected`) | ✅ COMPLIANT |
| Filtered group partial fail | brew group: gh updated, docker failed, `--only gh,docker` | `upp update --only gh,docker` | Flat summary reports gh updated and docker failed | `internal/cli/update_test.go` (`TestRunUpdate_OnlyNarrowsGroupBatch`, `TestRunUpdate_PerToolErrorIsolation`) | ✅ COMPLIANT |
| Group dry-run preview | apt group, gh pending, docker current | `upp update -n` | Flat summary reports gh would update and docker current | `internal/cli/update_test.go` (`TestRunUpdate_DryRunPlannedFlags`, `TestRunUpdate_ManagerSelfUpdateDryRun`) | ✅ COMPLIANT |
| uv standalone in summary | `uv` updated successfully alongside manager groups and other standalone tools | `upp update` | Summary lists `uv` under updated tools in canonical discovery order | `internal/cli/update_test.go` (`TestRunUpdate_AllSucceedSummary`), `internal/adapters/official/update_test.go` (`TestUpdate/uv/live-standalone-success`) | ✅ COMPLIANT |
| uv dry-run summary | `uv` has pending tool updates, `--dry-run` | `upp update -n` | Summary reports `uv` would update; does not report "All clean!" | `internal/cli/update_test.go` (`TestRunUpdate_DryRunPendingNeverClean`), `scripts/smoke-test.sh` (Test 9: `upp update -n --only uv`) | ✅ COMPLIANT |

##### Requirement: Engine Progress and Outcome Mapping (5 scenarios)

| Scenario | Given | When | Then | Test Reference | Result |
|---|---|---|---|---|---|
| StatusAvailable mapping | `CheckOutcome` with `StatusAvailable`, Current "1.0", Latest "1.1" | Bridge maps to `output.ToolResult` | Result has `Status: StatusAvailable`, `Version: "1.0 → 1.1"` | `internal/cli/checkrun_test.go` (`TestOutcomeToToolResult`) | ✅ COMPLIANT |
| StatusCurrent mapping | `CheckOutcome` with `StatusCurrent`, Current "2.0" | Bridge maps to `output.ToolResult` | Result has `Status: StatusCurrent`, `Version: "2.0"` | `internal/cli/checkrun_test.go` (`TestOutcomeToToolResult`) | ✅ COMPLIANT |
| StatusSkipped mapping | `CheckOutcome` with `StatusSkipped` | Bridge maps to `output.ToolResult` | Result has `Status: StatusSkipped` | `internal/cli/checkrun_test.go` (`TestOutcomeToToolResult`) | ✅ COMPLIANT |
| StatusFailed mapping | `CheckOutcome` with `StatusFailed`, error, and stderr | Bridge maps to `output.ToolResult` | Result has `Status: StatusFailed`, `Error: err`, `Stderr: stderr` | `internal/cli/checkrun_test.go` (`TestOutcomeToToolResult`) | ✅ COMPLIANT |
| CheckBoard line flipping | Progress callback receives `CheckProgress` for index $i$ | CLI invokes `board.Complete(i, res)` | Row $i$ on `CheckBoard` flips to mapped outcome atomically | `internal/cli/update_test.go` (`TestRunUpdate_InteractiveSelection`), `internal/output/checkboard_test.go` (`TestCheckBoard_Complete_AvailableFlipsOnlyTargetLine`) | ✅ COMPLIANT |

---

### Architectural & Seam Integrity Audit

1. **Headless Domain Isolation**:
   - `internal/engine` defines pure domain structs (`CheckStatus`, `CheckOutcome`, `CheckProgress`, `Filter`, `PlannedUpdate`, `UpdatePlan`).
   - Package dependency analysis (`go list -f '{{.Imports}}' ./internal/engine`) confirms zero imports of `internal/output`, `github.com/spf13/cobra`, or `github.com/charmbracelet/bubbletea`.
   - Verified by unit test `TestEngine_HeadlessImportIsolation` in `internal/engine/engine_test.go`.

2. **CLI Presentation Delegation & Seam Preservation**:
   - CLI commands (`checkrun.go`, `list.go`, `update.go`) delegate discovery, concurrent checks, and plan formulation to `engine.Engine`.
   - Presentation translation bridge `outcomeToToolResult` in `internal/cli/checkrun.go` maps domain outcomes directly to BubbleTea and ANSI terminal renderers without altering visual styling, cursor positions, or emoji indicators.
   - Backward-compatible wrapper signatures (`runChecks`, `calculateWorkerCount`, `buildAdapterList`) and test seams (`cliDeps`, `listDeps`) remain fully preserved; all existing unit and hermetic integration tests pass without regression.

3. **Concurrency & Resource Safety**:
   - Bounded worker pool adheres strictly to $[4, 8]$ CPU core clamping rules.
   - Goroutine panic containment safely captures panics in third-party or custom adapter check invocations without terminating worker loops.
   - Cooperative context cancellation cleanly stops worker job ingestion, drains channels, and settles the `sync.WaitGroup` with zero leaked goroutines.
   - Race detector (`go test ./... -count=1 -race`) confirms zero data race conditions across all packages.

---

### Conclusion & Verdict

Independent formal verification of `upp-application-engine` confirms 100% task completion (20/20 tasks), 100% requirement compliance (13/13 requirements), and 100% scenario compliance (91/91 scenarios). All quality gates (`go build`, `go test`, `go test -race`, `go vet`, `gofmt`, and `smoke-test.sh`) passed with exit code 0, no data races, and no regressions.

**Final Verdict**: **PASS**
