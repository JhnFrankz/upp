# Tasks: upp-uv-adapter — Astral uv Package & Tool Manager Adapter

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ≈380 (additions ≈365, deletions ≈15, net ≈+350) |
| 400-line budget risk | Medium |
| Chained PRs recommended | Yes |
| Suggested split | PR 1 Adapter core, detection & registry parity → PR 2 Check & root-free inspection → PR 3 Dual-scope update & error handling → PR 4 Docs, specs & verification |
| Delivery strategy | ask-on-risk |
| Chain strategy | stacked-to-main |

Decision needed before apply: Yes
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: Medium

The changes introduce first-class Python developer tooling via Astral's `uv` across the official adapter layer (`uv.go`), shared platform catalog (`catalog.go`), registry wiring (`registry.go`), and existing hermetic test suites (`info_test.go`, `detect_test.go`, `adapter_test.go`, `registry_test.go`, `parity_test.go`, `check_test.go`, `update_test.go`). A 4-part stacked PR series isolates interface compliance & registry parity, read-only inspection with exit code 2 resilience, dual-scope mutating execution, and documentation/spec synchronization.

### Suggested Work Units

| Unit | Goal | PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|----|----------------------|-----------------|-------------------|
| 1 | Core `UvAdapter`, Name, Detect, Info, Registry & Catalog | PR 1 | `go test ./internal/adapters/official ./internal/platform -run 'TestInfo\|TestDetect\|TestAdapterNames\|TestAllAdapters\|TestAdaptersForPlatform\|TestResolveOwner\|TestKindManager\|TestOwnerMetadata\|TestAdapterByName\|TestEveryAdapterIsInCatalog\|TestEveryCatalogEntryHasAdapter\|TestCatalogPlatformsMatchAdapterPlatforms\|TestCatalogOwnershipMatchesAdapter\|TestCatalogNamesMatchAdapterNames' -count=1` | N/A (static registry & lookPath seam) | Revert `uv.go` skeleton, `registry.go` entry, `catalog.go` entry |
| 2 | Read-only inspection (`Check()`, external manager bypass, outdated tool listing) | PR 2 | `go test ./internal/adapters/official -run 'TestCheck$' -count=1` | `upp update -n --only uv` | Revert `Check()`, `isExternalManagerError()`, `parseUvSelfUpdateOutput()`, `parseUvToolListOutdatedOutput()` in `uv.go` |
| 3 | Dual-scope mutating update (`Update(dryRun)`, exit code 2 bypass, `uv tool upgrade --all`) | PR 3 | `go test ./internal/adapters/official -run 'TestUpdate$' -count=1` | `upp update -n` (dry-run) / `upp update --only uv` | Revert `Update()` in `uv.go` |
| 4 | Documentation, smoke tests & canonical OpenSpec sync | PR 4 | `go test ./... -count=1` | `bash scripts/smoke-test.sh --skip-build` | Revert `README.md`, `smoke-test.sh`, canonical spec updates |

---

## Strict TDD

All implementation follows strict TDD: write or update failing tests first ([RED]), implement the minimal code required to satisfy the assertion ([GREEN]), then refactor ([REFACTOR]) and ensure test cleanliness. No real subprocesses are spawned during unit testing; all command execution is hermetically intercepted via `setExecFakes` (`runCmdFn`, `runCmdArgsFn`, `lookPathFn`).

---

## Phase 1 — Adapter Core, Identification & Detection (PR 1)

- [x] 1.1 [RED][S] `internal/adapters/official/`: Declare test expectations for `uv` across metadata, detect, and name suites [D1, D5]:
  - [`info_test.go`](file:///home/jhan/Projects/upp/internal/adapters/official/info_test.go): Add `infoCase` table row for `"uv"` expecting `ID: "uv"`, `Name: "uv"`, `Platforms: []string{"linux", "macos", "windows"}`, `Trust: adapters.TrustOfficial`, `UpdatePolicy: adapters.PolicyGated`, `Kind: adapters.KindTool`.
  - [`detect_test.go`](file:///home/jhan/Projects/upp/internal/adapters/official/detect_test.go): Add `{"uv", &UvAdapter{}, "uv"}` to `lookPathAdapters` in `TestDetect`.
  - [`adapter_test.go`](file:///home/jhan/Projects/upp/internal/adapters/official/adapter_test.go): Add `{"uv", func() adapters.Adapter { return &UvAdapter{} }, "uv"}` to `TestAdapterNames`.
  - Verify failure: Run `go test ./internal/adapters/official -run 'TestInfo|TestDetect|TestAdapterNames' -count=1` — fails compilation due to undefined `UvAdapter` (RED).
- [x] 1.2 [S] `internal/adapters/official/uv.go`: Create adapter skeleton implementing core identity and compile-time interface assertion [D1]:
  - Define `type UvAdapter struct{}`.
  - Add compile-time interface assertion:
    ```go
    var _ adapters.Adapter = (*UvAdapter)(nil)
    ```
  - Implement `Name() string { return "uv" }`.
  - Implement `Detect() bool { return lookPath("uv") }`.
  - Implement `Info() adapters.ToolInfo` declaring `ID: "uv"`, `Name: "uv"`, `Platforms: []string{"linux", "macos", "windows"}`, `Trust: adapters.TrustOfficial`, `UpdatePolicy: adapters.PolicyGated`, `Kind: adapters.KindTool`. Standalone tool across all platforms (`Manager: nil`, `ManagerPackage: nil`, zero privileges).
  - Add compile stubs for `Check() (adapters.UpdateInfo, error)` and `Update(dryRun bool) (adapters.Result, error)`.
  - Verify success: Run `go test ./internal/adapters/official -run 'TestInfo|TestDetect|TestAdapterNames' -count=1` — all tests pass (GREEN).
- [x] 1.3 [S] REFACTOR: Run `gofmt -s -w internal/adapters/official/uv.go` and `go vet ./internal/adapters/official`.

---

## Phase 2 — Registry & Catalog Registration (PR 1)

- [x] 2.1 [RED][S] Update registry and catalog parity tests to assert 14 official adapters, platform presence, and tool cardinality [D5]:
  - [`registry_test.go`](file:///home/jhan/Projects/upp/internal/adapters/official/registry_test.go):
    - `TestAllAdaptersCount`: Update expected adapter count from 13 to 14.
    - `TestAdaptersForPlatformLinux`: Add `"uv"` to `expectedPresent` (11 → 12).
    - `TestAdaptersForPlatformMacOS`: Add `"uv"` to `expectedPresent` (9 → 10).
    - `TestAdaptersForPlatformWindows`: Add `"uv"` to `expectedPresent` (10 → 11).
    - `TestResolveOwner`: Add test rows for `uv` on Linux, macOS, and Windows confirming `wantNil: true` (standalone tool on all platforms).
    - `TestKindManagerConsistency`: Ensure `uv` is verified as `KindTool` (not present in `managers` map).
    - `TestOwnerMetadata`: Update expected totals to `Total: 14, Managers: 5, Tools: 9` (tools incremented from 8 to 9).
    - `TestAdapterByName`: Add `{"uv", false}` test case.
  - [`parity_test.go`](file:///home/jhan/Projects/upp/internal/adapters/official/parity_test.go):
    - Tests `TestEveryAdapterIsInCatalog`, `TestEveryCatalogEntryHasAdapter`, `TestCatalogPlatformsMatchAdapterPlatforms`, `TestCatalogOwnershipMatchesAdapter`, and `TestCatalogNamesMatchAdapterNames` will automatically execute against the 14 adapters.
  - Verify failure: Run `go test ./internal/adapters/official -run 'TestAllAdapters|TestAdaptersForPlatform|TestOwnerMetadata|TestAdapterByName|TestEveryAdapterIsInCatalog|TestEveryCatalogEntryHasAdapter' -count=1` — fails with adapter count (13 != 14) and missing catalog/adapter entry errors (RED).
- [x] 2.2 [S] Register `UvAdapter` in official registry and platform catalog [D5]:
  - [`registry.go`](file:///home/jhan/Projects/upp/internal/adapters/official/registry.go): Add `&UvAdapter{}` to `AllAdapters()` (after `&BunAdapter{}`).
  - [`catalog.go`](file:///home/jhan/Projects/upp/internal/platform/catalog.go): Add `{ID: "uv", Name: "uv", Platforms: []string{OSLinux, OSMacOS, OSWindows}, Kind: adapters.KindTool}` to `OfficialTools` (after `bun`).
  - Verify success: Run `go test ./internal/adapters/official ./internal/platform -run 'TestAllAdapters|TestAdaptersForPlatform|TestResolveOwner|TestKindManager|TestOwnerMetadata|TestAdapterByName|TestEveryAdapterIsInCatalog|TestEveryCatalogEntryHasAdapter|TestCatalogPlatformsMatchAdapterPlatforms|TestCatalogOwnershipMatchesAdapter|TestCatalogNamesMatchAdapterNames' -count=1` — all pass (GREEN).
- [x] 2.3 [S] REFACTOR: Run `gofmt -s -w internal/adapters/official/registry.go internal/platform/catalog.go internal/adapters/official/registry_test.go` and `go vet ./internal/adapters/... ./internal/platform/...`.

---

## Phase 3 — Root-Free Outdated Inspection (`Check()`) (PR 2)

- [x] 3.1 [RED][M] `internal/adapters/official/check_test.go`: Add table-driven check test cases covering version extraction, self-update inspection, tool outdated listing, exit code 2 bypass, and error propagation [D3, D4]:
  - Add test rows to `TestCheck`:
    - `uv/self-update-available`: `uv --version` returns `uv 0.5.0 (085c7b399 2024-12-16)`, `uv self update --dry-run` returns `Would update uv from 0.5.0 to 0.5.11`, `uv tool list --outdated` returns `No outdated tools` → `CurrentVersion: "0.5.0"`, `LatestVersion: "0.5.0"`, `UpdateAvailable: true`.
    - `uv/tool-update-available`: `uv --version` returns `uv 0.5.11`, `uv self update --dry-run` returns `uv is already up to date`, `uv tool list --outdated` returns `ruff v0.8.0 (latest: v0.9.0)` → `CurrentVersion: "0.5.11"`, `LatestVersion: "0.5.11"`, `UpdateAvailable: true`.
    - `uv/both-available`: pending self-update and outdated tools → `CurrentVersion: "0.5.0"`, `LatestVersion: "0.5.0"`, `UpdateAvailable: true`.
    - `uv/up-to-date`: self-update returns `uv is already up to date`, tools return `No outdated tools` → `CurrentVersion: "0.5.11"`, `LatestVersion: "0.5.11"`, `UpdateAvailable: false`.
    - `uv/no-tools-installed`: self-update returns `uv is already up to date`, tools return `No tools installed` → `CurrentVersion: "0.5.11"`, `LatestVersion: "0.5.11"`, `UpdateAvailable: false`.
    - `uv/external-manager-bypass-no-updates`: `uv self update --dry-run` exits code 2 with stderr/stdout `"error: uv was installed through an external package manager. Please use that package manager to update uv."`, tools return `No outdated tools` → `CurrentVersion: "0.5.11"`, `LatestVersion: "0.5.11"`, `UpdateAvailable: false` (clean bypass, no error).
    - `uv/external-manager-bypass-with-tool-updates`: self-update exits code 2 with external manager message, tools return `black v24.1.0 (latest: v24.2.0)` → `CurrentVersion: "0.5.11"`, `LatestVersion: "0.5.11"`, `UpdateAvailable: true`.
    - `uv/self-update-other-nonzero-fails`: `uv self update --dry-run` exits code 1 with network error → returns structured error containing `(exit 1)`.
    - `uv/tool-list-command-fails`: `uv tool list --outdated` exits non-zero → returns structured error.
    - `uv/not-installed-error`: `lookPath["uv"] = false` → returns error `"uv is not installed"`.
    - `uv/empty-version-unknown`: `uv --version` returns empty string → `CurrentVersion: "unknown"`, `LatestVersion: "unknown"`, `UpdateAvailable: false`.
  - Add unit tests for pure parsing and detection helper functions:
    - `TestIsExternalManagerError`: Verify exit code 2 with `"external package manager"` returns `true`; exit code 1 or other error strings return `false`; nil error returns `false`.
    - `TestParseUvSelfUpdateOutput`: Verify `"Would update uv from 0.5.0 to 0.5.11"` returns `true`; `"uv is already up to date"` returns `false`; empty string returns `false`.
    - `TestParseUvToolListOutdatedOutput`: Verify package rows return `true`; `"No tools installed"` and `"No outdated tools"` return `false`; empty string returns `false`.
  - Verify failure: Run `go test ./internal/adapters/official -run 'TestCheck$' -count=1` — fails on uv check cases (RED).
- [x] 3.2 [M] `internal/adapters/official/uv.go`: Implement `Check()` and pure parsing/bypass helpers [D3, D4]:
  - Implement `isExternalManagerError(err error, output string) bool`:
    - Returns `false` if `err == nil`.
    - Checks `isExitCode(err, 2)` or exit 2 string in `err.Error()`.
    - Checks `strings.Contains(output + " " + err.Error(), "external package manager")`.
  - Implement `parseUvSelfUpdateOutput(out string) bool`:
    - Returns `false` if empty or contains `"up to date"`.
    - Returns `true` if contains `"would update"`, `"new version"`, or `"updating"`.
  - Implement `parseUvToolListOutdatedOutput(out string) bool`:
    - Returns `false` if empty, contains `"No tools installed"`, or contains `"No outdated tools"`.
    - Returns `true` otherwise.
  - Implement `Check() (adapters.UpdateInfo, error)`:
    - Guard: If `!a.Detect()`, return `adapters.UpdateInfo{}`, error `"uv is not installed"`.
    - Version query: `extractVersion(commandOutput("uv", "--version"))`, defaulting to `"unknown"`.
    - Self-update inspection: `commandOutputErr("uv", "self", "update", "--dry-run")`. If error, evaluate `isExternalManagerError`: if true, bypass gracefully and set `selfUpdateAvailable = false`; if false, return structured error per `Adapter Error Handling`. If success, evaluate `parseUvSelfUpdateOutput`.
    - Outdated tool inspection: `commandOutputErr("uv", "tool", "list", "--outdated")`. If error, return structured error. If success, evaluate `parseUvToolListOutdatedOutput`.
    - Compute `updateAvailable = selfUpdateAvailable || toolsUpdateAvailable`.
    - Return `adapters.UpdateInfo{CurrentVersion: current, LatestVersion: current, UpdateAvailable: updateAvailable}`.
  - Verify success: Run `go test ./internal/adapters/official -run 'TestCheck$' -count=1` — all check tests pass (GREEN).
- [x] 3.3 [S] REFACTOR: Ensure check method purity and timeout compliance:
  - Verify `Check()` executes zero mutating subprocesses, requires zero superuser privileges, and is bounded by `CheckTimeout` (15s).
  - Run `gofmt -s -w internal/adapters/official/uv.go internal/adapters/official/check_test.go` and `go vet ./internal/adapters/official`.

---

## Phase 4 — Dual-Scope Execution (`Update(dryRun)`) (PR 3)

- [x] 4.1 [RED][M] `internal/adapters/official/update_test.go`: Add table-driven update test cases covering dry-run preview, live dual-scope execution, external manager bypass, and stage failure aborts [D2, D3]:
  - Define update command constants:
    ```go
    const (
        uvSelfUpdateCmd  = "uv self update"
        uvToolUpgradeCmd = "uv tool upgrade --all"
    )
    ```
  - Add test rows to `TestUpdate`:
    - `uv/not-installed-error`: `lookPath["uv"] = false` → `wantErr: true`.
    - `uv/dry-run-shortcut`: `dryRun: true`, `uvSelfUpdateCmd` and `uvToolUpgradeCmd` mapped to `failIfRun` → expects `Success: true, Before: "0.5.0", After: "0.5.0"` with zero mutating subprocesses executed.
    - `uv/success-dual-scope`: Standalone `uv`. Initial version `"0.5.0"`, post-update version `"0.5.11"`. `uv self update` succeeds; `uv tool upgrade --all` succeeds → expects `Success: true, Before: "0.5.0", After: "0.5.11"`, `Privileges: nil`.
    - `uv/external-manager-bypass-success`: `uv self update` fails with exit code 2 and `"error: uv was installed through an external package manager"`. Self-update error is bypassed gracefully; `uv tool upgrade --all` executes and succeeds → expects `Success: true, Before: "0.5.11", After: "0.5.11"`.
    - `uv/self-update-unexpected-failure`: `uv self update` fails with exit code 1 / network error → `uv tool upgrade --all` mapped to `failIfRun` (must NOT execute). Expects `Success: false, resultErr: true`.
    - `uv/tool-upgrade-failure`: `uv self update` succeeds, but `uv tool upgrade --all` fails with exit code 1 / subprocess error → expects `Success: false, resultErr: true`.
  - Verify failure: Run `go test ./internal/adapters/official -run 'TestUpdate$' -count=1` — fails on uv update cases (RED).
- [x] 4.2 [M] `internal/adapters/official/uv.go`: Implement `Update(dryRun bool)` with sequential two-stage execution and external manager bypass [D2, D3]:
  - Guard: If `!a.Detect()`, return `Result{Success: false}`, error `"uv is not installed"`.
  - Query `before := extractVersion(commandOutput("uv", "--version"))`, fallback to `"unknown"` if empty.
  - Dry-run guard: If `dryRun`, return `Result{Success: true, Before: before, After: before}` without invoking subprocesses.
  - Stage 1: Self-update binary:
    - Execute `stdout, stderr, err := runCmd("uv self update")`.
    - If `err != nil`: check `isExternalManagerError(err, stderr+" "+stdout)`. If false, abort and return `Result{Success: false, Before: before, After: before, Error: fmt.Errorf("uv self update failed: %w", err)}`. If true, bypass error gracefully and proceed to Stage 2.
  - Stage 2: Upgrade global Python tools:
    - Execute `_, stderr2, err2 := runCmd("uv tool upgrade --all")`.
    - If `err2 != nil`: abort and return `Result{Success: false, Before: before, After: before, Error: fmt.Errorf("uv tool upgrade failed: %w", err2)}`.
  - Version query: Query `after := extractVersion(commandOutput("uv", "--version"))`, fallback to `before` if empty.
  - Return `Result{Success: true, Before: before, After: after}`.
  - Verify success: Run `go test ./internal/adapters/official -run 'TestUpdate$' -count=1` — all update tests pass (GREEN).
- [x] 4.3 [S] REFACTOR: Ensure zero privilege escalation and clean error wrapping:
  - Verify `Update()` never requests or returns superuser privileges (`Privileges: nil`).
  - Verify subprocess executions are bounded by `UpdateTimeout` (120s).
  - Run `gofmt -s -w internal/adapters/official/uv.go internal/adapters/official/update_test.go` and `go vet ./internal/adapters/official`.

---

## Phase 5 — Documentation, Smoke Tests & Comprehensive Verification (PR 4)

- [x] 5.1 [S] [`README.md`](file:///home/jhan/Projects/upp/README.md): Document `uv` support:
  - Add `uv` to the official package and runtime managers list in overview (line 5) and Features list (line 10).
- [x] 5.2 [S] [`scripts/smoke-test.sh`](file:///home/jhan/Projects/upp/scripts/smoke-test.sh): Add `uv` dry-run smoke test assertion:
  - In Test 9 ("Filter flags on the dry-run query surface"), add `run_test "upp update -n --only uv" "$BINARY" update -n --only uv`.
- [x] 5.3 [M] Canonical OpenSpec specification synchronization:
  - Sync delta specifications into canonical specification files in `openspec/specs/`:
    - [`openspec/specs/tool-adapter/spec.md`](file:///home/jhan/Projects/upp/openspec/specs/tool-adapter/spec.md): Update Official Adapter Catalog table (Linux, macOS, Windows: `uv self update` && `uv tool upgrade --all`), Update Gating requirements for `uv` (`PolicyGated`), and add Dual-Scope Toolchain Execution and Resilience requirements.
    - [`openspec/specs/platform-detection/spec.md`](file:///home/jhan/Projects/upp/openspec/specs/platform-detection/spec.md): Update Tool Catalog for Linux, macOS, and Windows to include `uv` (`KindTool`).
    - [`openspec/specs/ux-patterns/spec.md`](file:///home/jhan/Projects/upp/openspec/specs/ux-patterns/spec.md): Update Live Check Board and Summary Report requirements to document `uv` rendering as a standalone tool in canonical discovery order across all platforms.
- [x] 5.4 [S] Comprehensive verification suite:
  - Run full unit tests: `go test ./... -count=1`.
  - Run race detector: `go test ./... -count=1 -race`.
  - Run type checker and vet: `go vet ./...`.
  - Check code formatting: verify `gofmt -s -l .` produces zero diffs.
  - Build binary: `go build -o upp ./cmd/upp`.
  - Run smoke tests: `bash scripts/smoke-test.sh --skip-build`.
