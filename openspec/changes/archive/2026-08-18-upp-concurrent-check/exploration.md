# Exploration: Concurrent Tool Checking Engine (`upp-concurrent-check`)

## 1. Executive Summary

`upp check` (and the bare `upp` command) currently iterates through enabled tool adapters in a strict sequential loop (`for _, a := range filteredAdapters`). For environments with 8–15 enabled package managers and developer tools, sequential checking suffers from additive latency: each tool incurs subprocess invocation and/or network queries (e.g., `apt-cache policy`, `brew --version`, `npm outdated -g`, remote registry version lookups), leading to total check times of 10–30+ seconds.

Because tool version checks are **read-only and independent**, checking is embarrassingly parallel. Transitioning to a bounded concurrent checking engine reduces execution time from the sum of all check durations ($O(\sum T_i)$) to approximately the maximum single check duration ($O(\max T_i)$), dropping check latency to ~1–2 seconds.

This document explores the architecture, compares concurrency patterns (Unbounded Goroutines vs. Bounded Worker Pool vs. Pipeline/Errgroup), establishes deterministic ordering guarantees, designs thread-safe progress tracking, ensures strict failure isolation, and details CLI integration with `--concurrency` / `-j`.

---

## 2. Current Architecture & Bottleneck Analysis

### 2.1 Sequential Execution Flow in `internal/cli/check.go`

In the current implementation ([`runCheck`](file:///home/jhan/projects/upp/internal/cli/check.go#L45-L119)):

```go
for i, a := range filteredAdapters {
    info := a.Info()
    if !a.Detect() {
        results = append(results, output.ToolResult{
            Name:   info.Name,
            Status: output.StatusSkipped,
        })
        continue
    }

    if !gf.Quiet && total > 1 {
        r.Progress("Checking", i+1, total, info.Name)
    }

    updateInfo, err := a.Check()
    if err != nil {
        results = append(results, output.ToolResult{
            Name:   info.Name,
            Status: output.StatusFailed,
            Error:  timeoutErr(info.Name, "check", err),
        })
        continue
    }

    if updateInfo.UpdateAvailable {
        results = append(results, output.ToolResult{
            Name:    info.Name,
            Status:  output.StatusAvailable,
            Version: fmt.Sprintf("%s → %s", updateInfo.CurrentVersion, updateInfo.LatestVersion),
        })
    } else {
        results = append(results, output.ToolResult{
            Name:    info.Name,
            Status:  output.StatusCurrent,
            Version: updateInfo.CurrentVersion,
        })
    }
}
```

### 2.2 Invocations & Entry Points
1. `upp check`: invoked via [`NewCheckCommand`](file:///home/jhan/projects/upp/internal/cli/check.go#L18-L28).
2. `upp` (bare command): invoked in [`BuildRoot`](file:///home/jhan/projects/upp/internal/cli/parser.go#L37-L52) (`RunE: func(cmd *cobra.Command, args []string) error { return runCheck(...) }`).

### 2.3 Key Observations & Constraints
- **Subprocess Timeout Safety**: Each adapter's check call is bounded by [`adapters.CheckTimeout`](file:///home/jhan/projects/upp/internal/adapters/timeouts.go#L8) (30s) through [`adapters.RunCommandWithTimeout`](file:///home/jhan/projects/upp/internal/adapters/exec.go#L74-L118) or [`shellExecWithTimeout`](file:///home/jhan/projects/upp/internal/adapters/custom.go#L142-L144).
- **Stateless Adapters**: Official adapter structs (`AptAdapter`, `BrewAdapter`, `NpmAdapter`, etc.) and `CustomAdapter` instances maintain no mutable state across calls. `Detect()` and `Check()` are safe for concurrent invocation across distinct adapter instances.
- **Output Rendering**: [`output.Renderer`](file:///home/jhan/projects/upp/internal/output/render.go#L41-L47) writes directly to `os.Stdout` without internal locking. In non-quiet mode, `r.Progress("Checking", current, total, name)` is called before checking each tool, and `r.CheckSummary(results)` is called at the end with the full results slice.

---

## 3. Concurrency Pattern Comparison

We evaluated three concurrency architectures for the checking engine:

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                               Concurrency Options                               │
├─────────────────────────┬─────────────────────────────┬─────────────────────────┤
│ Approach A: Unbounded   │ Approach B: Bounded Worker  │ Approach C: Pipeline /  │
│ Goroutines (1 per tool) │ Pool (Channel / Semaphore)  │ errgroup                │
├─────────────────────────┼─────────────────────────────┼─────────────────────────┤
│ • 1 goroutine per tool  │ • Bounded workers (W size)  │ • golang.org/x/sync     │
│ • Direct index write    │ • Configurable via -j/--flag│ • Context cancellation  │
│ • No worker pool limit  │ • Default: min(8,max(4,CPU))│ • Stage-based channels  │
└─────────────────────────┴─────────────────────────────┴─────────────────────────┘
```

### Approach A: Unbounded Goroutines per Tool with Direct Index Mapping

#### Architecture
Spawn one goroutine for every enabled adapter in `filteredAdapters`. Each goroutine writes its result directly into a pre-allocated slice `results := make([]output.ToolResult, total)` at index `i`. Synchronize with `sync.WaitGroup`.

```go
results := make([]output.ToolResult, len(filteredAdapters))
var wg sync.WaitGroup
for i, a := range filteredAdapters {
    wg.Add(1)
    go func(idx int, adapter adapters.Adapter) {
        defer wg.Done()
        results[idx] = checkSingleTool(adapter)
    }(i, a)
}
wg.Wait()
```

#### Evaluation
- **Pros**:
  - Simplest possible implementation; zero queue/channel management.
  - Natural $O(1)$ deterministic index placement without post-sorting.
- **Cons**:
  - **Unbounded resource consumption**: Spawning 20–30 simultaneous subprocesses (e.g. on setups with many custom tools) causes high fork/exec pressure, file descriptor exhaustion, and memory spikes.
  - **Local contention & rate limits**: Multiple concurrent package manager queries (e.g., `apt-cache`, `brew`, `nvm`) can hit registry rate limits, trigger disk I/O thrashing, or contend for local lockfiles/sockets.
  - **No user control**: Cannot be tuned or throttled via `--concurrency` / `-j`.

---

### Approach B: Bounded Worker Pool with Configurable Concurrency (Recommended)

#### Architecture
Set worker pool capacity $W = \text{concurrency}$, with default calculated as:
$$\text{defaultWorkers} = \min(8, \max(4, \text{runtime.NumCPU()}))$$

Supports user override via `--concurrency <N>` / `-j <N>` (persistent flag on root or check command). Setting `-j 1` guarantees fully deterministic serial execution.

#### Implementation Variants

##### Variant B1: Fixed Worker Goroutines Consuming a Job Channel
```go
type checkJob struct {
    index   int
    adapter adapters.Adapter
}

type checkTaskResult struct {
    index  int
    result output.ToolResult
}

func runCheckPool(adapters []adapters.Adapter, workers int, onProgress func(name string)) []output.ToolResult {
    total := len(adapters)
    results := make([]output.ToolResult, total)
    if total == 0 {
        return results
    }

    if workers > total {
        workers = total
    }
    if workers < 1 {
        workers = 1
    }

    jobs := make(chan checkJob, total)
    out := make(chan checkTaskResult, total)

    var wg sync.WaitGroup
    for w := 0; w < workers; w++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for job := range jobs {
                if onProgress != nil {
                    onProgress(job.adapter.Info().Name)
                }
                res := checkSingleAdapter(job.adapter)
                out <- checkTaskResult{index: job.index, result: res}
            }
        }()
    }

    for i, a := range adapters {
        jobs <- checkJob{index: i, adapter: a}
    }
    close(jobs)

    wg.Wait()
    close(out)

    for r := range out {
        results[r.index] = r.result
    }
    return results
}
```

##### Variant B2: Semaphore-Bounded Goroutines with WaitGroup
```go
sem := make(chan struct{}, workers)
results := make([]output.ToolResult, total)
var wg sync.WaitGroup

for i, a := range filteredAdapters {
    wg.Add(1)
    sem <- struct{}{} // acquire token
    go func(idx int, adapter adapters.Adapter) {
        defer wg.Done()
        defer func() { <-sem }() // release token
        results[idx] = checkSingleAdapter(adapter)
    }(i, a)
}
wg.Wait()
```

#### Evaluation
- **Pros**:
  - **Controlled concurrency**: Prevents system overload and OS process table saturation.
  - **Dynamic flag configuration**: Respects `--concurrency` / `-j` and settings.
  - **Deterministic indexing**: Direct index mapping guarantees canonical ordering.
  - **Graceful fallback**: `-j 1` provides zero-concurrency sequential execution for debugging and constrained environments.
  - **Zero external dependencies**: Implemented purely with Go standard library (`sync`, `runtime`, `chan`).
- **Cons**:
  - Slightly more orchestration logic than unbounded goroutines.

---

### Approach C: Pipeline / `golang.org/x/sync/errgroup` Pattern

#### Architecture
Use Go's `errgroup.Group` with `g.SetLimit(workers)` to manage parallel execution and error handling, or construct a multi-stage channel pipeline (`Detect` stage $\rightarrow$ `Check` stage $\rightarrow$ `Collect` stage).

#### Evaluation
- **Pros**:
  - Standard Go concurrency library pattern with built-in `SetLimit`.
- **Cons**:
  - **Violation of Failure Isolation**: `errgroup` cancels the shared context on the *first* returned error. In `upp check`, a failure on one tool (e.g., `apt` check failing) MUST NOT cancel or abort checks on other tools; all tools must complete and be reported in the summary.
  - Using `errgroup` while suppressing errors (returning `nil` from all worker functions) reduces it to a standard `sync.WaitGroup`, but requires pulling in `golang.org/x/sync` (which is not currently in `go.mod`).
  - Pipeline stages (separating `Detect()` and `Check()`) introduce unnecessary channel synchronization overhead because `Detect()` (usually `exec.LookPath`) takes $<0.1\text{ms}$ and is inherently tied to each tool's lifecycle.

---

### Concurrency Matrix Summary

| Criterion | Approach A: Unbounded Goroutines | Approach B: Bounded Worker Pool (Recommended) | Approach C: errgroup / Pipeline |
| :--- | :--- | :--- | :--- |
| **Resource Safety** | ❌ Low (unbounded forks) | ✅ High (strictly bounded $W$) | ⚠️ Medium |
| **CLI Flag `-j` / `--concurrency`** | ❌ Not supported | ✅ Fully configurable | ⚠️ Requires custom wrapper |
| **Deterministic Ordering** | ✅ Yes ($O(1)$ index placement) | ✅ Yes ($O(1)$ index placement) | ⚠️ Requires collector sorting |
| **Failure Isolation** | ✅ High | ✅ High (isolated worker per job) | ❌ Poor (`errgroup` cancels on error) |
| **External Dependencies** | ✅ None (stdlib) | ✅ None (stdlib) | ❌ Requires `golang.org/x/sync` |
| **Implementation Complexity** | Low | Low–Medium | Medium–High |

---

## 4. Key Requirements & Architectural Solutions

### 4.1 Deterministic Result Ordering
- **Requirement**: `results` slice passed to `r.CheckSummary(results)` must match the exact canonical order defined by `buildAdapterList` and filtered by `FilterTools`, regardless of which check finishes first.
- **Solution**:
  1. Input adapters form an ordered slice `filteredAdapters` of length $N$.
  2. Each check task carries its canonical index $i \in [0, N-1]$.
  3. Pre-allocated slice `results := make([]output.ToolResult, N)`.
  4. Worker writes to `results[task.index]` (either directly or via result channel collector).
  5. Because each task writes to a unique, non-overlapping index in `results`, slice assignment is data-race-free without mutex contention once `wg.Wait()` finishes.

```
Canonical Adapter Order:  [0: apt]      [1: brew]      [2: npm]      [3: docker]
                              │             │              │              │
Workers Check in Parallel:  (Task 0)      (Task 1)       (Task 2)       (Task 3)
                              │             │              │              │
Completion Order (random):    └──────────────┼──────────────┼──────────────┘
                                             │ (e.g. npm finishes first, then apt...)
                                             ▼
Deterministic Results Slice: [0: aptRes]   [1: brewRes]   [2: npmRes]   [3: dockerRes]
                                             ▼
                                     r.CheckSummary(results)
```

---

### 4.2 Thread-Safe Progress Tracking

#### Challenges
- `r.Progress("Checking", current, total, name)` prints `  ⟳ Checking X/Y: <name>\n` to `os.Stdout`.
- In a concurrent execution model, multiple workers running simultaneously would race on `os.Stdout`, causing garbled multi-byte ANSI sequences or corrupted lines.
- In sequential execution, `current` was the loop index $i+1$. In concurrent execution, progress events occur as tasks are dispatched or completed.

#### Solutions
1. **Atomic Progress Counter**:
   Maintain an atomic counter `var progressCounter atomic.Int32`. As each tool check is initiated (or completed), increment the counter:
   $$\text{curr} = \text{progressCounter.Add}(1)$$
2. **Synchronized Output**:
   Protect calls to `r.Progress` with a mutex or encapsulate progress reporting inside a dedicated progress reporter / thread-safe renderer method:
   ```go
   type SafeRenderer struct {
       r  *output.Renderer
       mu sync.Mutex
   }

   func (s *SafeRenderer) Progress(op string, current, total int, name string) {
       s.mu.Lock()
       defer s.mu.Unlock()
       s.r.Progress(op, current, total, name)
   }
   ```
3. **Preserving Test Contracts**:
   Tests like [`TestCheckProgress_LabelsChecking`](file:///home/jhan/projects/upp/internal/cli/integration_test.go#L508-L531) assert that stdout contains `"Checking 1/2"` and `"Checking 2/2"`. By incrementing the atomic counter from 1 to $N$ as checks begin, output will deterministically contain all values `1/N` through `N/N`.
4. **Quiet Mode Optimization**:
   When `gf.Quiet` is true, progress reporting is bypassed entirely, eliminating mutex locking overhead.

---

### 4.3 Failure Isolation

#### Requirements
- If an adapter hangs, times out, throws an error, or panics, it must NOT crash or abort checks on any other adapter.
- The summary at the end must report the failed tool cleanly as `output.StatusFailed` with its structured error message.

#### Defensive Layers
1. **Timeout Containment**:
   All adapter operations are wrapped by `context.WithTimeout(ctx, adapters.CheckTimeout)` (30s) inside `adapters.RunCommandWithTimeout`. If a command hangs, its process group is killed and `context.DeadlineExceeded` is returned, mapped via `timeoutErr(info.Name, "check", err)`.
2. **Worker Panic Recovery**:
   Every worker goroutine includes a deferred `recover()` block:
   ```go
   func safeCheck(a adapters.Adapter) (res output.ToolResult) {
       defer func() {
           if r := recover(); r != nil {
               res = output.ToolResult{
                   Name:   a.Info().Name,
                   Status: output.StatusFailed,
                   Error:  fmt.Errorf("panic during check: %v", r),
               }
           }
       }()

       info := a.Info()
       if !a.Detect() {
           return output.ToolResult{
               Name:   info.Name,
               Status: output.StatusSkipped,
           }
       }

       updateInfo, err := a.Check()
       if err != nil {
           return output.ToolResult{
               Name:   info.Name,
               Status: output.StatusFailed,
               Error:  timeoutErr(info.Name, "check", err),
           }
       }

       if updateInfo.UpdateAvailable {
           return output.ToolResult{
               Name:    info.Name,
               Status:  output.StatusAvailable,
               Version: fmt.Sprintf("%s → %s", updateInfo.CurrentVersion, updateInfo.LatestVersion),
           }
       }
       return output.ToolResult{
           Name:    info.Name,
           Status:  output.StatusCurrent,
           Version: updateInfo.CurrentVersion,
       }
   }
   ```

---

### 4.4 CI & Quiet Mode Compatibility

- **Quiet Mode (`--quiet` / `gf.Quiet`)**:
  - Suppresses all progress lines.
  - Summary table / lines printed at the end remain compact.
  - Self-update hint suppressed.
- **CI Mode (`--ci` / `gf.CI`)**:
  - Non-interactive line-by-line output.
  - No ANSI cursor manipulation or spinner escapes that break CI logs.
  - In `runCheck`, exit status remains 0 when check succeeds (even if some tools have updates or failed checks, matching current check behavior), while `r.CheckSummary` lists failures clearly.

---

## 5. CLI Flags & Configuration Design

### 5.1 Concurrency Flag Specification

Add `--concurrency` (shorthand `-j`) to `GlobalFlags` in `internal/cli/parser.go`:

```go
type GlobalFlags struct {
    Quiet       bool
    CI          bool
    Only        string
    Skip        string
    Concurrency int
}
```

Flag Registration in `BuildRoot()`:
```go
root.PersistentFlags().IntVarP(
    &gf.Concurrency,
    "concurrency",
    "j",
    defaultConcurrency(),
    "maximum number of concurrent checks",
)
```

### 5.2 Default Concurrency Calculation

```go
// defaultConcurrency calculates the default worker pool size:
// min(8, max(4, runtime.NumCPU()))
func defaultConcurrency() int {
    cpus := runtime.NumCPU()
    if cpus < 4 {
        return 4
    }
    if cpus > 8 {
        return 8
    }
    return cpus
}
```

#### Calculated Values by CPU Count
- 1 or 2 CPU cores (low-end VMs, CI containers): `max(4, 1) = 4` $\rightarrow$ `min(8, 4) = 4` workers
- 4 CPU cores: `max(4, 4) = 4` $\rightarrow$ `min(8, 4) = 4` workers
- 6 CPU cores: `max(4, 6) = 6` $\rightarrow$ `min(8, 6) = 6` workers
- 8 CPU cores: `max(4, 8) = 8` $\rightarrow$ `min(8, 8) = 8` workers
- 16+ CPU cores (high-end workstations): `max(4, 16) = 16` $\rightarrow$ `min(8, 16) = 8` workers

---

## 6. Proposed Engine Architecture

### 6.1 Architecture Overview

```
                      ┌───────────────────────────┐
                      │    filteredAdapters       │
                      │  [apt, brew, npm, docker] │
                      └─────────────┬─────────────┘
                                    │
                                    ▼
                      ┌───────────────────────────┐
                      │    Job Channel Queue      │
                      │   buffered: len(tools)    │
                      └─────────────┬─────────────┘
                                    │
            ┌───────────────────────┼───────────────────────┐
            ▼                       ▼                       ▼
   ┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
   │    Worker 1     │     │    Worker 2     │     │    Worker W     │
   │ safeCheck(tool) │     │ safeCheck(tool) │     │ safeCheck(tool) │
   └────────┬────────┘     └────────┬────────┘     └────────┬────────┘
            │                       │                       │
            │ Progress Callback     │ Progress Callback     │ Progress Callback
            ▼ (SafeRenderer.mu)     ▼ (SafeRenderer.mu)     ▼ (SafeRenderer.mu)
      "Checking 1/4"          "Checking 2/4"          "Checking 3/4"
            │                       │                       │
            └───────────────────────┼───────────────────────┘
                                    ▼
                      ┌───────────────────────────┐
                      │   Result Channel / Wait   │
                      └─────────────┬─────────────┘
                                    │
                                    ▼
                      ┌───────────────────────────┐
                      │  results[job.index] = res │
                      │  (Canonical Tool Order)   │
                      └─────────────┬─────────────┘
                                    │
                                    ▼
                      ┌───────────────────────────┐
                      │  r.CheckSummary(results)  │
                      └───────────────────────────┘
```

### 6.2 Seam & Dependency Injection Compatibility

`checkDeps` in `internal/cli/check.go` maintains its existing injectable seams:
```go
type checkDeps struct {
    clientFactory    func(cachePath string) *selfupdate.Client
    buildAdapterList func(cfg *config.Config, osName string) []adapters.Adapter
}
```
The concurrent checking engine takes `filteredAdapters` and runs the checks concurrently without altering the interface between `runCheck` and `buildAdapterList` or `selfupdate.Client`.

---

## 7. Affected Files & Implementation Scope

| File | Change Scope | Description |
| :--- | :--- | :--- |
| [`internal/cli/parser.go`](file:///home/jhan/projects/upp/internal/cli/parser.go) | Modified | Add `Concurrency int` to `GlobalFlags`; bind `--concurrency` / `-j` with `defaultConcurrency()`. |
| [`internal/cli/check.go`](file:///home/jhan/projects/upp/internal/cli/check.go) | Modified | Implement worker pool check runner; replace sequential `for` loop; thread-safe progress calling. |
| [`internal/output/render.go`](file:///home/jhan/projects/upp/internal/output/render.go) | Modified (or helper) | Add mutex synchronization to `r.Progress` or wrap renderer for concurrent progress calls. |
| [`internal/cli/integration_test.go`](file:///home/jhan/projects/upp/internal/cli/integration_test.go) | Modified / Extended | Verify concurrent check behavior, progress output, deterministic ordering, and flag parsing. |
| [`internal/cli/check_test.go`](file:///home/jhan/projects/upp/internal/cli/) | New / Extended | Unit tests for worker pool, concurrency bounds, panic recovery, and timeout isolation. |

---

## 8. Risk Assessment & Mitigation

| Risk | Severity | Likelihood | Mitigation |
| :--- | :--- | :--- | :--- |
| **Race Conditions in Output** | High | Medium | Synchronize progress rendering via `sync.Mutex` on `Renderer` or thread-safe progress emitter. Run `go test -race ./...`. |
| **Non-Deterministic Summary Order** | High | Low | Indexed result collection (`results[job.index] = res`) guarantees canonical ordering regardless of execution duration. |
| **Package Manager Lock Conflicts** | Medium | Low | `upp check` only runs read-only queries (e.g. `apt-cache policy`, `brew --version`), which do not acquire exclusive database locks. |
| **High Subprocess Overhead on Low-End Systems** | Medium | Low | Concurrency is bounded to $\le 8$ by default, clamped to $\ge 1$, and user-tunable via `-j 1`. |
| **Hanging Check Blocks Run** | Medium | Low | Each check execution is bounded by `adapters.CheckTimeout` (30s) process group kill; failures do not block other workers. |

---

## 9. Recommendations for OpenSpec Artifacts

When proceeding to proposal and design phases for `upp-concurrent-check`:
1. **Proposal**: Scope to `upp check` and bare `upp` read-only check operations. Explicitly keep `upp update` sequential to prevent package manager write-lock contention.
2. **Specs**: Update `command-interface/spec.md` with `--concurrency` / `-j` flag requirements, and `ux-patterns/spec.md` with concurrent progress reporting specifications.
3. **Design**: Document the Bounded Worker Pool architecture, `SafeRenderer` synchronization, index-mapped result preservation, and panic recovery isolation.
4. **Verification**: Include race detector validation (`go test -race ./...`) and benchmark assertions demonstrating parallel check speedups.
