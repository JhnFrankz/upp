# Proposal: Concurrent Tool Checking Engine (`upp-concurrent-check`)

## Intent

Reduce `upp check` (and bare `upp`) wall-clock execution time from $O(\sum T_i)$ to $O(\max T_i)$ by executing read-only, independent tool version checks concurrently using an automatic, bounded internal worker pool without adding any new CLI flags.

## Scope

### In Scope
- Automatic bounded worker pool in `internal/cli/check.go` with capacity $\min(8, \max(4, \text{NumCPU}))$.
- Thread-safe progress rendering in `internal/output/render.go` with mutex synchronization and atomic progress counters.
- Deterministic index-based result slotting matching canonical tool ordering.
- Per-tool failure isolation, 30s timeout enforcement, and worker panic recovery.
- Zero new flags: completely transparent optimization preserving the existing clean CLI interface.

### Out of Scope
- Adding `--concurrency` or `-j` flags (discarded to maintain a zero-config, minimal CLI surface).
- Concurrent update execution (`upp update` remains strictly sequential due to package manager locking and interactive safety).

## Capabilities

### New Capabilities
None.

### Modified Capabilities
- `ux-patterns`: Document thread-safe concurrent progress output and deterministic summary table ordering during `upp check`.

## Approach

- Maintain the existing CLI surface in `internal/cli/parser.go` completely untouched (no new flags).
- Refactor `internal/cli/check.go` to dispatch adapter checks across an internal bounded worker pool into a pre-allocated results slice keyed by canonical adapter index.
- Protect progress reporting in `internal/output/render.go` with `sync.Mutex` to prevent race conditions and garbled terminal output.
- Wrap tool checks with deferred `recover()` and `adapters.CheckTimeout` context containment.

## Affected Areas

| Area | Impact | Description |
|---|---|---|
| `internal/cli/check.go` | Modified | Implement automatic bounded worker pool and panic recovery |
| `internal/output/render.go` | Modified | Add mutex synchronization for progress rendering |
| `internal/cli/integration_test.go` | Modified | Add tests for concurrent progress and order determinism |
| `internal/cli/check_test.go` | New | Unit tests for worker pool, bounds, and race safety |

## Risks

| Risk | Likelihood | Mitigation |
|---|---|---|
| Race conditions in terminal output | Medium | Mutex-synchronized rendering; validate via `go test -race ./...` |
| Non-deterministic summary order | Low | Direct slotting into pre-allocated slice using canonical task indices |
| Host resource saturation | Low | Default workers capped at 8 based on available CPUs |
| Tool check panic crashing run | Low | Deferred `recover()` per worker returning `output.StatusFailed` |

## Rollback Plan

Revert git commits modifying `internal/cli/` and `internal/output/`. No database, configuration schema, or persistent state changes are involved.

## Dependencies

None external. Standard library only (`sync`, `sync/atomic`, `runtime`, `context`).

## Success Criteria

- [ ] `upp check` latency drops to approximately the slowest single tool check time.
- [ ] Summary table order is 100% deterministic regardless of tool completion order.
- [ ] Zero race conditions under `go test -race ./...`.
- [ ] Tool check failures, timeouts, or panics remain isolated and do not abort other workers.
- [ ] CLI surface (`upp --help`) remains 100% clean with zero added flags.
