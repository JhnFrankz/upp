# Tasks: Default Manager-Group Bulk Updates (`upp-bulk-upgrade-default`)

## Review Workload Forecast & Budget Analysis

| Metric | Target / Forecast | Status |
| :--- | :--- | :--- |
| **Total Changed Lines Forecast** | **~340 lines** (excluding generated files/fixtures) | **Within 400-line Budget** |
| **400-Line Budget Review Risk** | **Low** | ✅ Compliant |
| **Work Units (WU)** | 4 Atomic Work Units | ✅ Isolated Seams |
| **Chained PRs Recommended** | Optional / Can be applied as a coherent atomic feature | Flexible |
| **Delivery Strategy** | Direct atomic delivery or step-by-step WU PR chain | Ready |

---

## Work Unit (WU) Mapping

| Unit | Focus / Scope | Target Files | Forecast Lines | Review Risk |
| :--- | :--- | :--- | :--- | :--- |
| **WU1** | **Adapter Layer Delegation** | [`gh.go`](file:///home/jhan/projects/upp/internal/adapters/official/gh.go), [`docker.go`](file:///home/jhan/projects/upp/internal/adapters/official/docker.go), [`go.go`](file:///home/jhan/projects/upp/internal/adapters/official/go.go), [`update_test.go`](file:///home/jhan/projects/upp/internal/adapters/official/update_test.go), [`parity_test.go`](file:///home/jhan/projects/upp/internal/adapters/official/parity_test.go) | ~75 lines | Low |
| **WU2** | **CLI Runner & Execution Engine** | [`update.go`](file:///home/jhan/projects/upp/internal/cli/update.go), [`update_test.go`](file:///home/jhan/projects/upp/internal/cli/update_test.go) | ~125 lines | Low |
| **WU3** | **Output & TTY Selector Integration** | [`selector.go`](file:///home/jhan/projects/upp/internal/output/selector.go), [`render.go`](file:///home/jhan/projects/upp/internal/output/render.go), [`group.go`](file:///home/jhan/projects/upp/internal/output/group.go), [`selector_test.go`](file:///home/jhan/projects/upp/internal/output/selector_test.go), [`render_test.go`](file:///home/jhan/projects/upp/internal/output/render_test.go) | ~95 lines | Low |
| **WU4** | **Quality & Verification** | Regression suite, linters, delta spec audit | ~45 lines | Low |
| **Total** | **All 4 Work Units** | **Whole feature surface** | **~340 lines** | **Low** |

---

## Task Groups & Action Items

### Group 1: Adapter Layer Delegation (WU1)

- [x] **1.1 RED**: Write adapter unit tests in [`internal/adapters/official/update_test.go`](file:///home/jhan/projects/upp/internal/adapters/official/update_test.go) asserting that `GhAdapter.Update()`, `DockerAdapter.Update()`, and `GoAdapter.Update()` delegate to `PackageUpdater.UpdatePackage(pkg)` with platform-mapped package names, verify dry-run safety, and verify Linux `go` executes standalone update.
- [x] **1.2 GREEN**: Update [`GhAdapter.Update`](file:///home/jhan/projects/upp/internal/adapters/official/gh.go) to resolve owning manager via `ResolveOwner("gh", platform)`, assert `owner.(adapters.PackageUpdater)`, look up `a.Info().ManagerPackage[platform]`, and call `updater.UpdatePackage(pkg)`.
- [x] **1.3 GREEN**: Update [`DockerAdapter.Update`](file:///home/jhan/projects/upp/internal/adapters/official/docker.go) to resolve owning manager via `ResolveOwner("docker", platform)`, assert `owner.(adapters.PackageUpdater)`, look up `a.Info().ManagerPackage[platform]`, and call `updater.UpdatePackage(pkg)`.
- [x] **1.4 GREEN**: Update [`GoAdapter.Update`](file:///home/jhan/projects/upp/internal/adapters/official/go.go) to delegate to `updater.UpdatePackage(pkg)` when owned (macOS `brew` -> `golang`/`go`, Windows `winget` -> `GoLang.Go`), preserving the native standalone tarball install path on Linux (`ResolveOwner` returns `nil`).
- [x] **1.5 REFACTOR**: Extend [`internal/adapters/official/parity_test.go`](file:///home/jhan/projects/upp/internal/adapters/official/parity_test.go) to verify all owned tools declare consistent `ManagerPackage` mappings and all manager adapters implement `adapters.PackageUpdater`.

**Group 1 Verification Commands**:
```bash
go test -v -count=1 ./internal/adapters/official/... -run "Test.*Update"
go test -v -count=1 ./internal/adapters/...
```

---

### Group 2: CLI Runner & Execution Engine (WU2)

- [x] **2.1 RED**: Add CLI unit tests in [`internal/cli/update_test.go`](file:///home/jhan/projects/upp/internal/cli/update_test.go) testing bare `upp update` default group bulk execution, `--manager`/`--update-group` filtering, `--skip` exclusion within groups, per-tool error isolation, and `--ci` fail-closed behavior with `EnforceRisk: true`.
- [x] **2.2 GREEN**: Refactor [`runUpdate`](file:///home/jhan/projects/upp/internal/cli/update.go) and [`runUpdateSequential`](file:///home/jhan/projects/upp/internal/cli/update.go) so that bare `upp update` partitions filtered adapters into manager groups and standalone tools, checking per-package availability via `PackageChecker.CheckPackage(pkg)` and executing via `PackageUpdater.UpdatePackage(pkg)` in canonical discovery order.
- [x] **2.3 GREEN**: Apply `EnforceRisk: true` on [`security.ConfirmAction`](file:///home/jhan/projects/upp/internal/security/confirm.go) invocations for package updates in `runUpdateSequential` and `processSelectedOutcome`, prompting for `sudo` in interactive TTY and returning `ConfirmError` in `--ci` mode.
- [x] **2.4 GREEN**: Implement per-tool error boundaries around `PackageUpdater.UpdatePackage(pkg)` calls so that a failure in one owned tool records `StatusFailed` and diagnostic stderr without terminating remaining sibling or standalone tool updates.
- [x] **2.5 REFACTOR**: Unify routing in `runUpdate` so explicit `--manager`/`--update-group` flags act as narrow filters over candidate manager groups while bare `upp update` runs all active manager groups and standalone tools seamlessly.

**Group 2 Verification Commands**:
```bash
go test -v -count=1 ./internal/cli/... -run "TestUpdate.*"
go test -v -count=1 ./internal/cli/...
```

---

### Group 3: Output & TTY Selector Integration (WU3)

- [x] **3.1 RED**: Add output unit tests in [`internal/output/selector_test.go`](file:///home/jhan/projects/upp/internal/output/selector_test.go) and [`internal/output/render_test.go`](file:///home/jhan/projects/upp/internal/output/render_test.go) for granular per-tool toggling under manager headers in `CheckboxSelector`, and test `GroupBatchPreview` / `GroupBulkSummary` with updated, current, skipped, and failed tools in canonical order.
- [x] **3.2 GREEN**: Ensure [`CheckboxSelector`](file:///home/jhan/projects/upp/internal/output/selector.go) correctly renders manager group headers while allowing granular toggling and selection of individual owned tool items.
- [x] **3.3 GREEN**: Update [`GroupBatchPreview`](file:///home/jhan/projects/upp/internal/output/render.go) and [`GroupBulkSummary`](file:///home/jhan/projects/upp/internal/output/render.go) / [`UpdateSummary`](file:///home/jhan/projects/upp/internal/output/render.go) to format per-tool outcomes within manager groups and explicitly report counts ("N updated, M skipped, K failed") in deterministic canonical discovery order.
- [x] **3.4 GREEN**: Connect TTY interactive selection to the carried-outcome execution loop in [`runUpdateInteractive`](file:///home/jhan/projects/upp/internal/cli/update.go) so that deselected owned tools are skipped and selected tools execute via `PackageUpdater.UpdatePackage(pkg)`.
- [x] **3.5 REFACTOR**: Audit color, plain text, and ANSI cursor handling in [`internal/output/`](file:///home/jhan/projects/upp/internal/output/) to prevent formatting artifacts or race conditions across different terminal environments.

**Group 3 Verification Commands**:
```bash
go test -v -count=1 ./internal/output/...
go test -v -count=1 ./internal/cli/... -run "TestInteractive.*"
```

---

### Group 4: Quality & Verification (WU4)

- [x] **4.1 Full Regression Suite**: Run entire repository test suite with race detection (`go test -count=1 -race ./...`).
- [x] **4.2 Static Analysis & Linters**: Run `golangci-lint run ./...` and `go vet ./...` to guarantee zero lint or style regressions.
- [x] **4.3 Delta Spec Compliance Audit**: Verify 100% scenario coverage against delta specs in `openspec/changes/upp-bulk-upgrade-default/specs/`:
  - `bulk-update`: Default group trigger, per-tool command execution, canonical ordering, error isolation, sudo CI fail-closed.
  - `tool-ownership-model`: Resolved owner delegation (`gh`, `docker`, `go`), Linux standalone `go` preservation.
  - `command-interface`: Default `upp update`, dry-run (`-n`, `--dry-run`), filtered flags (`--manager`, `--only`, `--skip`), `--ci` non-zero exit on failure.
  - `ux-patterns`: Granular per-tool selection in TTY CheckboxSelector, accurate summary counts without misleading "All clean!" / "All tools up to date." messages.
- [x] **4.4 Review Budget Verification**: Confirm total line diff is within the 400-line budget and document low review risk.

**Group 4 Verification Commands**:
```bash
go test -count=1 -race ./...
golangci-lint run ./...
git diff --stat
```

---

## Traceability Matrix (Specs to Tasks)

| Spec Requirement & Scenarios | Primary Task Group | Primary Work Unit |
| :--- | :--- | :--- |
| **tool-ownership-model**: `Resolved Owner Update Delegation` | Group 1 | WU1 |
| **bulk-update**: `Default Group Bulk Trigger` | Group 2 | WU2 |
| **bulk-update**: `Per-Owned-Tool Command Execution` | Group 1 & 2 | WU1 & WU2 |
| **bulk-update**: `Elevated sudo fails closed in CI` | Group 2 | WU2 |
| **command-interface**: `Default upp update execution` | Group 2 | WU2 |
| **command-interface**: `Dry run full and short flag` | Group 2 & 3 | WU2 & WU3 |
| **command-interface**: `Granular selection in manager group` | Group 3 | WU3 |
| **command-interface**: `Per-tool isolated failure & CI exit` | Group 2 | WU2 |
| **ux-patterns**: `Summary Report with explicit counts` | Group 3 | WU3 |
| **ux-patterns**: `Concurrent deterministic order` | Group 3 | WU3 |
| **Quality & Regression**: `Race detection & lint compliance` | Group 4 | WU4 |
