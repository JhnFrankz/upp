# Proposal: Astral uv Package & Tool Manager Adapter

## Intent

Extend `upp`'s developer toolchain coverage to the modern Python ecosystem by introducing an official `UvAdapter` in `internal/adapters/official/uv.go`.

Astral's `uv` has emerged as the premier next-generation Python package and tool manager, written in Rust, offering 10–100× performance gains over traditional tools (`pip`, `pipx`, `poetry`, `virtualenv`). `upp` currently provides first-class official adapters for JavaScript/TypeScript runtimes and package managers (`nvm`, `npm`, `pnpm`, `bun`), Go (`go`), and system package managers (`apt`, `brew`, `pacman`, `winget`, `scoop`), but completely lacks official support for Python developer tooling. Python developers currently rely on ad-hoc custom tool commands in `config.toml`, resulting in uncoordinated updates between the `uv` binary itself and its globally installed CLI tools.

The `UvAdapter` provides a cross-platform (`linux`, `macos`, `windows`), native implementation of `adapters.Adapter` adhering to `upp`'s core architectural tenets:
1. **User-Space & Root-Free Execution**: `uv` and its isolated tool environments operate entirely in user space (`~/.local/bin` on Linux/macOS, `%USERPROFILE%\.local\bin` or `uv\bin` on Windows). It runs without root or sudo privileges, declaring zero privilege escalations.
2. **Dual Operational Scope**: Simultaneously manages the `uv` toolchain binary itself AND upgrades all globally installed Python CLI applications via `uv tool upgrade --all`.
3. **Dynamic Self-Update Resilience**: Seamlessly handles diverse installation methods (standalone installer vs system package managers such as Homebrew, apt, pacman, winget, or scoop). When `uv` is installed via an external package manager, `uv self update` detects this (exiting code 2 with `"error: uv was installed through an external package manager"`). The adapter gracefully bypasses binary self-update without failing and proceeds immediately to upgrade global tools.
4. **Strict Gated Update Policy**: Declares `UpdatePolicy: adapters.PolicyGated`. In `upp list` and `upp update` pre-checks, it queries both `uv self update --dry-run` and `uv tool list --outdated`, marking `UpdateAvailable: true` only when an executable update or tool update is genuinely pending.
5. **Fail-Closed Diagnostics**: Network timeouts, process hangs, or unexpected runtime errors are captured into structured errors per `Adapter Error Handling`, avoiding false current reporting.

## Scope

### In Scope

- **Official uv Adapter (`internal/adapters/official/uv.go`)**:
  - Implement `adapters.Adapter`:
    - `Name() string`: Returns `"uv"`.
    - `Detect() bool`: Returns `lookPath("uv")`.
    - `Check() (adapters.UpdateInfo, error)`:
      - Extracts current version via `uv --version`.
      - Inspects self-update availability via `uv self update --dry-run`. Gracefully bypasses exit code 2 (`"error: uv was installed through an external package manager"`) without error.
      - Inspects outdated global tools via `uv tool list --outdated`.
      - Reports `UpdateAvailable: true` if either self-update or any global tool update is available.
      - Returns structured `adapters.UpdateInfo`.
    - `Update(dryRun bool) (adapters.Result, error)`:
      - In dry-run mode (`dryRun = true`), returns `Result{Success: true, Before: before, After: before}` without invoking mutating subprocesses.
      - In live mode (`dryRun = false`):
        - Executes `uv self update`; detects exit code 2 / external package manager message to bypass gracefully, propagating other non-zero exits as structured failures.
        - Executes `uv tool upgrade --all` to upgrade all global Python tools.
        - Queries updated version via `uv --version`.
        - Returns structured `adapters.Result` with before and after versions.
    - `Info() adapters.ToolInfo`:
      - Declares `ID: "uv"`, `Name: "uv"`, `Platforms: []string{"linux", "macos", "windows"}`, `Trust: adapters.TrustOfficial`, `UpdatePolicy: adapters.PolicyGated`, `Kind: adapters.KindTool`.
      - Standalone tool across all platforms (`Manager: nil`, `ManagerPackage: nil`).
- **Registry & Catalog Registration**:
  - Register `&UvAdapter{}` in `internal/adapters/official/registry.go` (`AllAdapters()`).
  - Register `ToolEntry{ID: "uv", Name: "uv", Platforms: []string{OSLinux, OSMacOS, OSWindows}, Kind: adapters.KindTool}` in `internal/platform/catalog.go` (`OfficialTools`).
- **Test Suite Updates**:
  - Update `internal/adapters/official/registry_test.go`:
    - Increment adapter count from 13 to 14 in `TestAllAdaptersCount`.
    - Add `uv` to `expectedPresent` in `TestAdaptersForPlatformLinux` (11 → 12), `TestAdaptersForPlatformMacOS` (9 → 10), and `TestAdaptersForPlatformWindows` (10 → 11).
    - Update `TestOwnerMetadata`: Total 14, Managers 5, Tools 9 (8 → 9).
    - Add `uv` to `TestAdapterByName`.
  - Update `internal/adapters/official/parity_test.go`:
    - Verify catalog ⇄ adapter synchronization (`TestEveryAdapterIsInCatalog`, `TestEveryCatalogEntryHasAdapter`, `TestCatalogPlatformsMatchAdapterPlatforms`, `TestCatalogOwnershipMatchesAdapter`, `TestCatalogNamesMatchAdapterNames`).
  - Update `internal/adapters/official/info_test.go`:
    - Add golden metadata assertion for `uv`.
  - Update `internal/adapters/official/detect_test.go` and `adapter_test.go`:
    - Add `uv` to `lookPathAdapters` and `TestAdapterNames`.
  - Add comprehensive unit tests in `internal/adapters/official/` (`check_test.go`, `update_test.go`, or `uv_test.go`):
    - Clean version extraction from `uv --version` (`uv 0.12.10 (...)` → `0.12.10`).
    - Detection success/failure via `lookPath`.
    - `Check()` self-update detection, external package manager exit code 2 bypass, and outdated tool parsing.
    - `Check()` clean up-to-date states (`"No tools installed"`, `"No outdated tools"`).
    - `Check()` failure propagation and timeout handling.
    - `Update()` dry-run simulation and live dual-scope execution (with and without external package manager bypass).
    - `Update()` failure handling on self-update and tool upgrade.
- **Specification Updates**:
  - `openspec/specs/tool-adapter/spec.md`: Add `uv` to the Official Adapter Catalog, Update Gating rules (`PolicyGated`), and Dual-Scope Execution semantics.
  - `openspec/specs/platform-detection/spec.md`: Add `uv` to Linux, macOS, and Windows official catalogs as standalone `KindTool`.
  - `openspec/specs/ux-patterns/spec.md`: Document `uv` rendering on the Live Check Board and Summary Report as a standalone tool across all platforms.

### Out of Scope

- **Project-Specific Virtual Environments (`uv venv`, `uv sync`, `uv pip`)**: `upp` explicitly manages machine-wide developer tools and runtimes. Repository-local virtualenvs and lockfile synchronizations are managed by project workflows, not `upp`.
- **Python Interpreter Installation & Upgrades (`uv python install / upgrade`)**: Installing or upgrading system-level or uv-managed CPython interpreters is excluded to prevent unexpected ABI breaks for existing Python environments.
- **Static Ownership Delegation (e.g. `uv` owned by `brew` or `winget`)**: `uv` is declared as a standalone `KindTool` on all platforms. Dynamic self-update detection cleanly handles externally managed installations at runtime without requiring brittle OS package manager coupling.

## Capabilities

### New Capabilities
None (no new OpenSpec specification domains are created).

### Modified Capabilities
- `tool-adapter`:
  - Official Adapter Catalog: Linux, macOS, and Windows platforms gain `uv` with dual update execution (`uv self update` and `uv tool upgrade --all`).
  - Update Gating: `uv` declared as `PolicyGated`, running `update()` only when `Check()` reports `UpdateAvailable=true`.
  - External Package Manager Bypass: Documented resilience rule: `uv self update` exit code 2 with external package manager message is bypassed gracefully during both `Check()` and `Update()`.
  - Subprocess Timeouts: Standard adapter check and update timeouts apply to `uv` subprocess invocations.
- `platform-detection`:
  - Tool Catalog: Linux, macOS, and Windows catalogs add `uv` (`ID: "uv"`, `Name: "uv"`, `Platforms: ["linux", "macos", "windows"]`, `Kind: adapters.KindTool`).
- `ux-patterns`:
  - Live Check Board & Summary Report: `uv` rendered as a standalone tool under the standalone tools section across all platforms.

## Approach

1. **Adapter Architecture (`internal/adapters/official/uv.go`)**:
   - Structure follows standalone official tool adapters (`bun.go`, `pnpm.go`, `opencode.go`).
   - **Detection**:
     ```go
     func (a *UvAdapter) Detect() bool {
         return lookPath("uv")
     }
     ```
   - **External Package Manager Handling**:
     Helper function `isExternalManagerError(err error, output string) bool`:
     Returns `true` when `err` reflects exit code 2 and `output` contains `"external package manager"`.
   - **Version Checking (`Check`)**:
     - Query installed version:
       `extractVersion(commandOutput("uv", "--version"))`.
     - Inspect self-update availability:
       Executes `commandOutputErr("uv", "self", "update", "--dry-run")`.
       - If exit code is 2 and output matches external package manager message: self-update is bypassed (`selfUpdateAvailable = false`).
       - If another error occurs: returns structured error per `Adapter Error Handling`.
       - If command succeeds: parses output; if output indicates an update is pending (e.g., contains `"Would update"` or does not report already up-to-date), sets `selfUpdateAvailable = true`.
     - Inspect global tool updates:
       Executes `commandOutputErr("uv", "tool", "list", "--outdated")`.
       - If command fails: returns structured error.
       - If output contains outdated package entries (non-empty and not containing `"No tools installed"` or `"No outdated tools"`): sets `toolsUpdateAvailable = true`.
     - Gating evaluation:
       `UpdateAvailable = selfUpdateAvailable || toolsUpdateAvailable`.
     - Returns `adapters.UpdateInfo{CurrentVersion: current, LatestVersion: current, UpdateAvailable: updateAvailable}`.
   - **Update Execution (`Update`)**:
     - In dry-run mode (`dryRun = true`), returns `Result{Success: true, Before: before, After: before}` without invoking commands.
     - In live mode (`dryRun = false`):
       - Step 1: Self-update.
         Executes `runCmd("uv self update")`. If it fails with exit code 2 and external package manager message, gracefully ignore and proceed. If it fails with any other error, return `Result{Success: false, Error: err}`.
       - Step 2: Global tool upgrade.
         Executes `runCmd("uv tool upgrade --all")`. If it fails, return `Result{Success: false, Error: err}`.
       - Query updated version: `after := extractVersion(commandOutput("uv", "--version"))`.
       - Returns `Result{Success: true, Before: before, After: after}`.
   - **Static Metadata (`Info`)**:
     ```go
     func (a *UvAdapter) Info() adapters.ToolInfo {
         return adapters.ToolInfo{
             ID:           "uv",
             Name:         "uv",
             Platforms:    []string{"linux", "macos", "windows"},
             Trust:        adapters.TrustOfficial,
             UpdatePolicy: adapters.PolicyGated,
             Kind:         adapters.KindTool,
         }
     }
     ```

2. **Registry & Catalog Wiring**:
   - In `internal/adapters/official/registry.go`:
     - Add `&UvAdapter{}` to `AllAdapters()`.
   - In `internal/platform/catalog.go`:
     - Add `{ID: "uv", Name: "uv", Platforms: []string{OSLinux, OSMacOS, OSWindows}, Kind: adapters.KindTool}` to `OfficialTools`.

3. **Parity and Golden Tests**:
   - `internal/adapters/official/registry_test.go`:
     - Update `TestAllAdaptersCount` to expect 14 adapters.
     - Update `TestAdaptersForPlatformLinux` (12 tools), `TestAdaptersForPlatformMacOS` (10 tools), `TestAdaptersForPlatformWindows` (11 tools).
     - Update `TestOwnerMetadata` (Total: 14, Managers: 5, Tools: 9).
     - Add `"uv"` to `TestAdapterByName`.
   - `internal/adapters/official/parity_test.go`:
     - Confirm all parity assertions pass for `uv`.
   - `internal/adapters/official/info_test.go`:
     - Add golden metadata assertion for `uv`.
   - `internal/adapters/official/detect_test.go` and `adapter_test.go`:
     - Add `uv` test cases.

4. **Unit Tests**:
   - Hermetic table-driven tests in `check_test.go` and `update_test.go` (or `uv_test.go`):
     - Detection on PATH vs missing binary.
     - Version string extraction from realistic `uv --version` outputs.
     - `Check()` with self-update available.
     - `Check()` with external package manager exit code 2 and outdated tools.
     - `Check()` with clean/up-to-date state (`"No tools installed"`, `"No outdated tools"`).
     - `Check()` command failure and timeout handling.
     - `Update()` dry-run and live dual-scope execution.
     - `Update()` error handling on tool upgrade failures.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/adapters/official/uv.go` | Created | Implements `UvAdapter` (`adapters.Adapter`) with dual self-update and tool upgrade logic |
| `internal/adapters/official/registry.go` | Modified | Registers `&UvAdapter{}` in `AllAdapters()` |
| `internal/platform/catalog.go` | Modified | Adds `uv` entry to `OfficialTools` across Linux, macOS, Windows |
| `internal/adapters/official/registry_test.go` | Modified | Updates adapter counts (13 → 14 total, 8 → 9 tools) and platform presence assertions |
| `internal/adapters/official/parity_test.go` | Modified | Verifies catalog ⇄ adapter synchronization for `uv` |
| `internal/adapters/official/info_test.go` | Modified | Adds golden metadata assertion for `uv` |
| `internal/adapters/official/detect_test.go` | Modified | Adds `uv` detection test case |
| `internal/adapters/official/adapter_test.go` | Modified | Adds `uv` adapter name verification |
| `internal/adapters/official/check_test.go` | Modified | Adds unit tests for `uv` check and outdated detection |
| `internal/adapters/official/update_test.go` | Modified | Adds unit tests for `uv` self-update and tool upgrade execution |
| `openspec/specs/tool-adapter/spec.md` | Modified | Adds `uv` to official catalog, update gating, and dual-scope update semantics |
| `openspec/specs/platform-detection/spec.md` | Modified | Adds `uv` to Linux, macOS, and Windows official catalog specification |
| `openspec/specs/ux-patterns/spec.md` | Modified | Documents `uv` display as a standalone tool on live board and summary |

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| **External Package Manager Collision** (`uv self update` fails when `uv` was installed via brew, apt, pacman, etc.) | High | Medium | `uv self update` exit code 2 and `"installed through an external package manager"` output is explicitly handled: both `Check()` and `Update()` gracefully bypass self-update without error and proceed to tool upgrades. |
| **Outdated Tools Output Parsing Instability** (subtle output changes across uv versions) | Low | Low | Checks are anchored on standard flags (`uv tool list --outdated`) and resilient output checks: empty output, `"No tools installed"`, and `"No outdated tools"` map cleanly to `UpdateAvailable = false`. |
| **Hanging Subprocesses during Tool Upgrades** (`uv tool upgrade --all` downloading large wheels) | Medium | Medium | All subprocess executions are strictly bounded by `adapters.CheckTimeout` (15s) and `adapters.UpdateTimeout` (120s), utilizing process groups to ensure prompt termination on timeout. |
| **Registry & Catalog Parity Drift** (`AllAdapters` and `OfficialTools` falling out of sync) | Low | Medium | Bidirectional compile/test-time parity assertions in `parity_test.go` enforce catalog and registry synchronization. |

## Rollback Plan

This change is purely additive to the official tool catalog and does not modify any existing tool adapters or change configuration file schemas.

If any regression or failure occurs:
1. Revert the commit(s) introducing `internal/adapters/official/uv.go`, registry and catalog registrations, test updates, and spec deltas.
2. The codebase returns to 13 official adapters with zero migrations and no residual state.
3. Any custom tools in `config.toml` naming `uv` will continue running as custom tools without interference.

## Success Criteria

- [ ] `UvAdapter` implemented in `internal/adapters/official/uv.go` conforming to `adapters.Adapter`.
- [ ] Detection returns `true` on all platforms when `uv` is found on PATH.
- [ ] `Check()` accurately inspects installed version and reports `UpdateAvailable: true` if self-update is pending or global tools are outdated.
- [ ] `Check()` and `Update()` gracefully bypass exit code 2 when `uv` is managed by an external package manager.
- [ ] `Update(dryRun=true)` simulates updates without making changes.
- [ ] `Update(dryRun=false)` executes self-update (if supported) and `uv tool upgrade --all`.
- [ ] `AllAdapters()` returns 14 adapters; `platform.OfficialTools` includes `uv` on Linux, macOS, and Windows.
- [ ] Parity tests (`parity_test.go`, `registry_test.go`, `info_test.go`) pass.
- [ ] Comprehensive unit tests for `uv` check and update workflows pass with zero race conditions (`go test -race ./...`).
- [ ] Spec deltas in `tool-adapter`, `platform-detection`, and `ux-patterns` documented.
