# Design: Concurrent Tool Checking Engine (`upp-concurrent-check`)

## Technical Approach

`upp check` and bare `upp` currently execute tool checks sequentially in a single-threaded loop, suffering additive latency across independent, read-only tools.

This design introduces a zero-config, concurrent checking engine:
1. An automatic bounded worker pool in `internal/cli/check.go` sized dynamically via `runtime.NumCPU()`.
2. Deterministic result placement via pre-allocated index slotting matching canonical tool discovery order.
3. Thread-safe single-line in-place progress rendering (`\r`) with mutex synchronization and atomic counters, gracefully degrading on non-TTY/CI outputs.
4. Comprehensive per-tool isolation safeguarding against timeouts (30s `CheckTimeout`) and worker panics via deferred `recover()`.

## Architecture Decisions

### Decision D1: Automatic Bounded Worker Pool (Zero-Config)
- **Choice**: Dispatch jobs across a bounded worker pool sized to $\min(8, \max(4, \text{runtime.NumCPU()}))$, with zero new CLI flags (no `--concurrency` / `-j`).
- **Alternatives**: Unbounded goroutines; configurable `--concurrency` flag.
- **Rationale**: Keeps CLI surface minimal and zero-config while protecting host resources (file descriptors, process forks) on both low-end VMs and high-core workstations.

### Decision D2: Deterministic Result Ordering via Direct Index Slotting
- **Choice**: Pre-allocate `results := make([]output.ToolResult, len(adapters))`. Each job struct carries its canonical discovery index `i`; workers assign directly to `results[job.index] = res`.
- **Alternatives**: Unordered results channel with post-sorting.
- **Rationale**: Provides $O(1)$ zero-overhead deterministic ordering matching canonical adapter sequence, eliminating sorting keys or mutex contention on result collection.

### Decision D3: Single-Line In-Place Progress Rendering
- **Choice**: Synchronize progress output using `sync.Mutex` on `output.Renderer` and an atomic counter (`atomic.Int32`). In interactive TTYs, render single-line in-place updates using `\r` (e.g. `\r  ⟳ Checking tools [X/Y]...`). For non-TTY / CI / pipes, emit clean line-buffered progress without ANSI/CR escapes.
- **Alternatives**: Multi-line terminal redraws; async event bus.
- **Rationale**: Prevents garbled interleaved stdout output during concurrent execution while keeping progress clean and responsive across TTY and CI environments.

### Decision D4: Per-Tool Failure and Panic Isolation
- **Choice**: Wrap each worker execution in a deferred `recover()` block and enforce 30s `adapters.CheckTimeout` context containment. Failures and panics record `output.StatusFailed` on the individual tool.
- **Alternatives**: Abort execution on first error (`errgroup`).
- **Rationale**: Ensures one faulty, hanging, or panicking tool adapter never interrupts or corrupts checks for other tools.

## Data Flow

```
filteredAdapters [N]
       │
       ▼
 [Job Channel] ── (capacity N, carries adapter + index i)
       │
 ┌─────┴───────────────────────┐
 ▼                             ▼
Worker 1 (1..W)            Worker W
 ├─ safeCheck(adapter)      ├─ safeCheck(adapter)
 │   ├─ recover() guard     │   ├─ recover() guard
 │   ├─ 30s CheckTimeout    │   ├─ 30s CheckTimeout
 │   └─ atomic counter++    │   └─ atomic counter++
 ├─ Renderer.mu.Lock()      ├─ Renderer.mu.Lock()
 │   └─ Progress("\r...")   │   └─ Progress("\r...")
 └─ results[i] = res        └─ results[i] = res
 └─────┬───────────────────────┘
       │ sync.WaitGroup.Wait()
       ▼
 r.CheckSummary(results)  (Deterministic canonical order)
```

## File Changes

| File | Action | Description |
|---|---|---|
| `internal/cli/check.go` | Modify | Implement bounded worker pool, `safeCheck` with panic recovery, direct index slotting |
| `internal/output/render.go` | Modify | Add `sync.Mutex` to `Renderer`; support synchronized single-line in-place progress and non-TTY fallback |
| `internal/cli/check_test.go` | Create | Unit tests for worker pool sizing, race conditions (`-race`), panic recovery, and timeout isolation |
| `internal/cli/integration_test.go` | Modify | Integration tests for concurrent check deterministic order and progress formatting |

## Interfaces / Contracts

```go
// internal/cli/check.go
type checkJob struct {
    index   int
    adapter adapters.Adapter
}

func defaultConcurrency() int {
    cpus := runtime.NumCPU()
    if cpus < 4 { return 4 }
    if cpus > 8 { return 8 }
    return cpus
}

func safeCheck(a adapters.Adapter) (res output.ToolResult)

// internal/output/render.go
type Renderer struct {
    w     io.Writer
    color bool
    emoji bool
    quiet bool
    mu    sync.Mutex
}

func (r *Renderer) Progress(op string, current, total int, name string)
```

## Testing Strategy

| Layer | What to Test | Approach |
|---|---|---|
| Unit (`check_test.go`) | Worker pool concurrency bounds & sizing | Verify `defaultConcurrency()` clamping logic [4, 8] |
| Unit (`check_test.go`) | Panic recovery isolation | Inject panicking fake adapter; verify `StatusFailed` recorded and run completes |
| Unit (`check_test.go`) | Deterministic ordering | Run mock adapters with inverse/random delays; verify output matches canonical index order |
| Concurrency / Race | Thread safety of progress and result writes | Run full suite with `go test -race ./...` |
| Integration | TTY in-place vs non-TTY/CI progress | Verify clean progress rendering and summary output across environments |

## Threat Matrix

| Boundary | Applicability | Design Response | Planned RED Tests |
|---|---|---|---|
| Subprocess Resource Exhaustion | Applicable | Strict worker pool bound ($\le 8$) prevents OS fork/file descriptor starvation | `TestCheck_WorkerPoolBounds`: verify active workers $\le W$ |
| Panic Containment | Applicable | Deferred `recover()` catches unexpected adapter panics, returning `StatusFailed` | `TestCheck_PanicRecovery`: mock panic adapter returns clean failure |
| Stdout Race / Corruption | Applicable | `Renderer.mu` ensures atomic line/in-place progress emissions | `TestCheck_ConcurrentProgressRace`: run under `-race` with concurrent output |
| Hung Check Deadlock | Applicable | 30s `CheckTimeout` kills runaway subprocesses | `TestCheck_TimeoutIsolation`: slow adapter times out without blocking pool |

## Migration / Rollout

- **Migration**: Zero configuration or CLI changes required. Fully backward-compatible drop-in optimization.
- **Rollback**: Revert `internal/cli/check.go` and `internal/output/render.go` commits (`git revert`).
