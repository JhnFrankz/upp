# Tasks: upp-pacman-adapter — Pacman Package Manager Adapter

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ≈410 (additions ≈390, deletions ≈20, net ≈+370) |
| 400-line budget risk | Medium |
| Chained PRs recommended | Yes |
| Suggested split | PR 1 Core adapter, identification & registry parity → PR 2 Check & version inspection → PR 3 Update & sudo execution → PR 4 Docs, specs & verification |
| Delivery strategy | ask-on-risk |
| Chain strategy | stacked-to-main |

Decision needed before apply: Yes
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: Medium

The changes span the official adapter layer (`pacman.go`), shared platform catalog (`catalog.go`), registry wiring (`registry.go`), and existing hermetic test suites (`info_test.go`, `detect_test.go`, `adapter_test.go`, `registry_test.go`, `parity_test.go`, `check_test.go`, `update_test.go`). A 4-part stacked PR series isolates interface compliance, read-only inspection, privileged mutating execution, and documentation/spec synchronization.

### Suggested Work Units

| Unit | Goal | PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|----|----------------------|-----------------|-------------------|
| 1 | Core `PacmanAdapter`, Name, Detect, Info, Registry & Catalog | PR 1 | `go test ./internal/adapters/official ./internal/platform -run 'TestInfo\|TestDetect\|TestAdapterNames\|TestAllAdapters\|TestAdaptersForPlatform\|TestKindManager\|TestOwnerMetadata\|TestAdapterByName\|TestManagerAdaptersImplementPackageInterfaces\|TestEveryAdapterIsInCatalog\|TestEveryCatalogEntryHasAdapter' -count=1` | N/A (static registry & lookPath seam) | Revert `pacman.go` skeleton, `registry.go` entry, `catalog.go` entry |
| 2 | Read-only inspection (`Check`, `CheckPackage`) + `vercmp` | PR 2 | `go test ./internal/adapters/official -run 'TestCheck$|TestCheckPackage' -count=1` | `upp update -n --only pacman` | Revert `Check`, `CheckPackage`, `compareVersions` in `pacman.go` |
| 3 | Mutating update (`Update`, `UpdatePackage`) + sudo privileges | PR 3 | `go test ./internal/adapters/official -run 'TestUpdate$|TestUpdatePackage' -count=1` | `upp update -n` (dry-run) — ConfirmAction skipped | Revert `Update`, `UpdatePackage` in `pacman.go` |
| 4 | Documentation, smoke tests & canonical OpenSpec sync | PR 4 | `go test ./... -count=1` | `bash scripts/smoke-test.sh --skip-build` | Revert `README.md`, `smoke-test.sh`, spec updates |

---

## Strict TDD

All implementation follows strict TDD: write or update failing tests first ([RED]), implement the minimal code required to satisfy the assertion ([GREEN]), then refactor ([REFACTOR]) and ensure test cleanliness. No real subprocesses are spawned during unit testing; all command execution is hermetically intercepted via `setExecFakes` (`runCmdFn`, `runCmdArgsFn`, `lookPathFn`).

---

## Phase 1 — Adapter Core, Identification & Detection (PR 1)

- [x] 1.1 [RED][S] `internal/adapters/official/`: Declare test expectations for `pacman` across metadata, detect, and name suites [D1, D5]:
  - [`info_test.go`](file:///home/jhan/Projects/upp/internal/adapters/official/info_test.go): Add `infoCase` table row for `"pacman"` expecting `ID: "pacman"`, `Name: "Pacman Package Manager"`, `Platforms: []string{"linux"}`, `Trust: adapters.TrustOfficial`, `UpdatePolicy: adapters.PolicyGated`, `Kind: adapters.KindManager`.
  - [`detect_test.go`](file:///home/jhan/Projects/upp/internal/adapters/official/detect_test.go): Add `{"pacman", &PacmanAdapter{}, "pacman"}` to `lookPathAdapters` in `TestDetect`.
  - [`adapter_test.go`](file:///home/jhan/Projects/upp/internal/adapters/official/adapter_test.go): Add `{"pacman", func() adapters.Adapter { return &PacmanAdapter{} }, "pacman"}` to `TestAdapterNames`.
  - Verify failure: Run `go test ./internal/adapters/official -run 'TestInfo|TestDetect|TestAdapterNames' -count=1` — fails compilation due to undefined `PacmanAdapter`.
- [x] 1.2 [S] `internal/adapters/official/pacman.go`: Create adapter skeleton implementing core identity and compile-time interface assertions [D1]:
  - Define `type PacmanAdapter struct{}`.
  - Add compile-time interface assertions:
    ```go
    var (
        _ adapters.Adapter        = (*PacmanAdapter)(nil)
        _ adapters.PackageChecker = (*PacmanAdapter)(nil)
        _ adapters.PackageUpdater = (*PacmanAdapter)(nil)
    )
    ```
  - Implement `Name() string { return "pacman" }`.
  - Implement `Detect() bool { return lookPath("pacman") }`.
  - Implement `Info() adapters.ToolInfo` declaring `ID: "pacman"`, `Name: "Pacman Package Manager"`, `Platforms: []string{"linux"}`, `Trust: adapters.TrustOfficial`, `UpdatePolicy: adapters.PolicyGated`, `Kind: adapters.KindManager`.
  - Add compile stubs for `Check() (adapters.UpdateInfo, error)`, `Update(dryRun bool) (adapters.Result, error)`, `CheckPackage(pkg string) (adapters.UpdateInfo, error)`, and `UpdatePackage(pkg string) (adapters.Result, error)`.
  - Verify success: Run `go test ./internal/adapters/official -run 'TestInfo|TestDetect|TestAdapterNames' -count=1` — all tests pass (GREEN).
- [x] 1.3 [S] REFACTOR: Run `gofmt -s -w internal/adapters/official/pacman.go` and `go vet ./internal/adapters/official`.

---

## Phase 2 — Registry & Catalog Registration (PR 1)

- [x] 2.1 [RED][S] Update registry and catalog parity tests to assert 13 official adapters and manager package interfaces [D5]:
  - [`registry_test.go`](file:///home/jhan/Projects/upp/internal/adapters/official/registry_test.go):
    - `TestAllAdaptersCount`: Update expected adapter count from 12 to 13.
    - `TestAdaptersForPlatformLinux`: Add `"pacman"` to `expectedPresent`.
    - `TestKindManagerConsistency`: Add `"pacman": true` to `managers` map.
    - `TestManagerOwnedToolCardinality`: Add `{"pacman-linux", "pacman", "linux", []string{}}` (pacman owns 0 official tools on Linux).
    - `TestOwnerMetadata`: Update expected totals to `Total: 13, Managers: 5, Tools: 8`.
    - `TestAdapterByName`: Add `{"pacman", false}` test case.
  - [`parity_test.go`](file:///home/jhan/Projects/upp/internal/adapters/official/parity_test.go):
    - `TestManagerAdaptersImplementPackageInterfaces`: Update condition to check `info.Kind == adapters.KindManager && (info.ID == "apt" || info.ID == "brew" || info.ID == "winget" || info.ID == "pacman")`.
  - Verify failure: Run `go test ./internal/adapters/official -run 'TestAllAdapters|TestAdaptersForPlatform|TestKindManager|TestOwnerMetadata|TestAdapterByName|TestManagerAdaptersImplementPackageInterfaces|TestEveryAdapterIsInCatalog|TestEveryCatalogEntryHasAdapter' -count=1` — fails with adapter count and missing catalog entry errors (RED).
- [x] 2.2 [S] Register `PacmanAdapter` in official registry and platform catalog [D5]:
  - [`registry.go`](file:///home/jhan/Projects/upp/internal/adapters/official/registry.go): Add `&PacmanAdapter{}` to `AllAdapters()` (after `&BrewAdapter{}`).
  - [`catalog.go`](file:///home/jhan/Projects/upp/internal/platform/catalog.go): Add `{ID: "pacman", Name: "Pacman Package Manager", Platforms: []string{OSLinux}, Kind: adapters.KindManager}` to `OfficialTools` (after `brew`).
  - Verify success: Run `go test ./internal/adapters/official ./internal/platform -run 'TestAllAdapters|TestAdaptersForPlatform|TestKindManager|TestOwnerMetadata|TestAdapterByName|TestManagerAdaptersImplementPackageInterfaces|TestEveryAdapterIsInCatalog|TestEveryCatalogEntryHasAdapter|TestCatalogPlatformsMatchAdapterPlatforms|TestCatalogOwnershipMatchesAdapter' -count=1` — all pass (GREEN).
- [x] 2.3 [S] REFACTOR: Run `gofmt -s -w internal/adapters/official/registry.go internal/platform/catalog.go` and `go vet ./internal/adapters/... ./internal/platform/...`.

---

## Phase 3 — Version Inspection (`Check` & `CheckPackage`) (PR 2)

- [x] 3.1 [RED][M] `internal/adapters/official/check_test.go`: Add table-driven check and package check test cases [D2, D3]:
  - Define query command templates for pacman:
    ```go
    pacmanInstalledCmd = "bash -o pipefail -c 'pacman -Q pacman 2>/dev/null | awk \"{print \\$2}\"'"
    pacmanCandidateCmd = "bash -o pipefail -c 'pacman -Si pacman 2>/dev/null | grep -E \"^Version\" | head -1 | awk \"{print \\$3}\"'"
    ```
  - Add test rows to `TestCheck`:
    - `pacman/update-available`: installed `6.1.0-1`, candidate `7.0.0-1`, `vercmp 7.0.0-1 6.1.0-1` returns `1` → `UpdateAvailable: true`.
    - `pacman/current`: installed `7.0.0-1`, candidate `7.0.0-1` → `UpdateAvailable: false` (no `vercmp` called when versions match).
    - `pacman/epoch-precedence`: installed `1:6.1.0-1`, candidate `7.0.0-1`, `vercmp 7.0.0-1 1:6.1.0-1` returns `-1` → `UpdateAvailable: false` (epoch in installed package takes precedence).
    - `pacman/vercmp-fallback`: `lookPath["vercmp"] = false`, installed `6.1.0-1`, candidate `7.0.0-1` → `UpdateAvailable: true` (string inequality fallback).
    - `pacman/candidate-command-fails`: `pacman -Si` exits non-zero → returns structured error containing `"pacman check failed"`.
    - `pacman/not-installed-error`: `lookPath["pacman"] = false` → returns error `"pacman is not installed"`.
  - Add test rows to `TestCheckPackage`:
    - `pacman/available`: package `ripgrep`, installed `14.1.0-1`, candidate `14.1.1-1`, `vercmp 14.1.1-1 14.1.0-1` returns `1` → `UpdateAvailable: true`.
    - `pacman/current`: package `ripgrep`, installed `14.1.1-1`, candidate `14.1.1-1` → `UpdateAvailable: false`.
    - `pacman/not-installed`: package `ripgrep`, installed `""` resolves to `"unknown"`, candidate `14.1.1-1` → `UpdateAvailable: false`.
    - `pacman/multi-repo-priority`: candidate output contains multi-repo `Version : ...` strings, verifies `head -1` prioritizes the first repository candidate.
  - Verify failure: Run `go test ./internal/adapters/official -run 'TestCheck$|TestCheckPackage' -count=1` — fails on pacman check cases (RED).
- [x] 3.2 [M] `internal/adapters/official/pacman.go`: Implement root-free database queries and `vercmp` comparison [D2, D3]:
  - Implement `CurrentVersion() (string, error)`: Query `bash -o pipefail -c 'pacman -Q pacman 2>/dev/null | awk \"{print \\$2}\"'` via `shellOutput`. If empty, return `"unknown", nil`.
  - Implement `compareVersions(current, latest string) bool`:
    - If `current == latest`, return `false`.
    - If `lookPath("vercmp")` is true, invoke `commandOutput("vercmp", latest, current)`. If output parses as integer `n`, return `n > 0`.
    - Fallback: return `current != latest`.
  - Implement `CheckPackage(pkg string) (adapters.UpdateInfo, error)`:
    - Guard: If `!a.Detect()`, return error `"pacman is not installed"`.
    - Installed query: `bash -o pipefail -c 'pacman -Q <pkg> 2>/dev/null | awk \"{print \\$2}\"'` via `shellOutput`. Empty becomes `"unknown"`.
    - Candidate query: `bash -o pipefail -c 'pacman -Si <pkg> 2>/dev/null | grep -E \"^Version\" | head -1 | awk \"{print \\$3}\"'` via `shellOutputErr(cmd, "pacman")`. Propagate non-zero exit as structured error. Empty becomes `"unknown"`.
    - When `current != "unknown"` and `latest != "unknown"`, calculate `updateAvailable = a.compareVersions(current, latest)`.
    - Return `adapters.UpdateInfo{CurrentVersion: current, LatestVersion: latest, UpdateAvailable: updateAvailable}`.
  - Implement `Check() (adapters.UpdateInfo, error)`: Delegate to `a.CheckPackage("pacman")`.
  - Verify success: Run `go test ./internal/adapters/official -run 'TestCheck$|TestCheckPackage' -count=1` — all check tests pass (GREEN).
- [x] 3.3 [S] REFACTOR: Ensure check method purity:
  - Verify `Check()` and `CheckPackage()` execute no mutating operations (`pacman -Sy` is forbidden) and require no superuser privileges.
  - Run `gofmt -s -w internal/adapters/official/pacman.go internal/adapters/official/check_test.go`.

---

## Phase 4 — Mutating Execution (`Update` & `UpdatePackage`) (PR 3)

- [x] 4.1 [RED][M] `internal/adapters/official/update_test.go`: Add table-driven update and package update test cases [D4]:
  - Define `pacmanUpdateCmd = "sudo pacman -S --noconfirm pacman"`.
  - Add test rows to `TestUpdate`:
    - `pacman/not-installed-error`: `lookPath["pacman"] = false` → `wantErr: true`.
    - `pacman/dry-run-shortcut`: `dryRun: true`, `pacmanUpdateCmd` mapped to `failIfRun` → expects `Success: true, Before: "6.1.0-1", After: "6.1.0-1"`.
    - `pacman/update-command-error`: `pacmanUpdateCmd` returns command error → expects `Success: false, Privileges: ["sudo"], resultErr: true`.
    - `pacman/stderr-marker-fails`: `pacmanUpdateCmd` returns `stderr: "error: failed to commit transaction"` → expects `Success: false, Privileges: ["sudo"], resultErr: true`.
    - `pacman/success`: live command succeeds, before `"6.1.0-1"`, after `"7.0.0-1"` → expects `Success: true, Privileges: ["sudo"]`.
  - Add test rows to `TestUpdatePackage`:
    - `pacman/ripgrep-updates-owned-package`: `sudo pacman -S --noconfirm ripgrep` succeeds → expects `Success: true, Privileges: ["sudo"]`.
    - `pacman/package-command-fails`: command error → expects `Success: false, Privileges: ["sudo"]`.
    - `pacman/stderr-marker-fails`: `stderr: "error: target not found: badpkg"` → expects `Success: false, Privileges: ["sudo"]`.
    - `pacman/not-installed-error`: `lookPath["pacman"] = false` → `wantErr: true`.
  - Verify failure: Run `go test ./internal/adapters/official -run 'TestUpdate$|TestUpdatePackage' -count=1` — fails on pacman update cases (RED).
- [x] 4.2 [M] `internal/adapters/official/pacman.go`: Implement self-only and package update execution with sudo privileges [D4]:
  - Implement `Update(dryRun bool) (adapters.Result, error)`:
    - Guard: If `!a.Detect()`, return `Result{Success: false}`, error `"pacman is not installed"`.
    - Retrieve `before, _ := a.CurrentVersion()`.
    - If `dryRun`, return `Result{Success: true, Before: before, After: before}` without invoking subprocesses.
    - Execute `sudo pacman -S --noconfirm pacman` via `runCmd`.
    - On subprocess error, return `Result{Success: false, Before: before, After: before, Error: fmt.Errorf("pacman update failed: %w", err), Privileges: []string{"sudo"}}`.
    - On stderr containing `"error:"`, return `Result{Success: false, Before: before, After: before, Error: fmt.Errorf("pacman update error: %s", truncate(stderr, 200)), Privileges: []string{"sudo"}}`.
    - Retrieve `after, _ := a.CurrentVersion()`.
    - Return `Result{Success: true, Before: before, After: after, Privileges: []string{"sudo"}}`.
  - Implement `UpdatePackage(pkg string) (adapters.Result, error)`:
    - Guard: If `!a.Detect()`, return `Result{Success: false}`, error `"pacman is not installed"`.
    - Retrieve `before, _ := a.CurrentVersion()`.
    - Execute `sudo pacman -S --noconfirm <pkg>` via `runCmd(fmt.Sprintf("sudo pacman -S --noconfirm %s", pkg))`.
    - Handle subprocess error and stderr `"error:"` marker, returning `Result` with `Privileges: []string{"sudo"}`.
    - Retrieve `after, _ := a.CurrentVersion()`.
    - Return `Result{Success: true, Before: before, After: after, Privileges: []string{"sudo"}}`.
  - Verify success: Run `go test ./internal/adapters/official -run 'TestUpdate$|TestUpdatePackage' -count=1` — all update tests pass (GREEN).
- [x] 4.3 [S] REFACTOR: Privilege verification:
  - Verify that both `Update` and `UpdatePackage` consistently return `Privileges: []string{"sudo"}` on all execution paths.
  - Run `gofmt -s -w internal/adapters/official/pacman.go internal/adapters/official/update_test.go`.

---

## Phase 5 — Documentation, Smoke Tests & OpenSpec Specification Sync (PR 4)

- [x] 5.1 [S] [`README.md`](file:///home/jhan/Projects/upp/README.md): Document Pacman support:
  - Add `pacman` to official package and runtime managers list in overview (line 5) and Features list (line 10).
- [x] 5.2 [S] [`scripts/smoke-test.sh`](file:///home/jhan/Projects/upp/scripts/smoke-test.sh): Add pacman smoke test assertion:
  - In Test 9 ("Filter flags on the dry-run query surface"), add `run_test "upp update -n --only pacman" "$BINARY" update -n --only pacman`.
- [x] 5.3 [M] Canonical OpenSpec specification synchronization:
  - Sync delta specifications into canonical specification files in `openspec/specs/`:
    - [`openspec/specs/tool-adapter/spec.md`](file:///home/jhan/Projects/upp/openspec/specs/tool-adapter/spec.md): Update Official Adapter Catalog table, Manager Self-Update Semantics table, Update Gating rules, Version Comparison (`vercmp`), and Check Failure Signal requirements.
    - [`openspec/specs/platform-detection/spec.md`](file:///home/jhan/Projects/upp/openspec/specs/platform-detection/spec.md): Update Linux Tool Catalog to include `pacman` (`KindManager`).
    - [`openspec/specs/tool-ownership-model/spec.md`](file:///home/jhan/Projects/upp/openspec/specs/tool-ownership-model/spec.md): Update Tool Ownership Declaration, Manager Owned-Tool Cardinality (pacman owns 0 official tools on Linux), Resolved Owner Update Delegation (`PackageChecker`/`PackageUpdater`), and Resolved-Owner Group Bulk Update.
    - [`openspec/specs/security-model/spec.md`](file:///home/jhan/Projects/upp/openspec/specs/security-model/spec.md): Update Official Tool Integrity (self-only `sudo pacman -S --noconfirm pacman`, prohibit `pacman -Syu` and `pacman -Sy`) and Confirmation for Destructive Operations (`sudo` privilege classification).
- [x] 5.4 [S] Final verification suite:
  - Run full test suite: `go test ./... -count=1`.
  - Run race detector: `go test ./... -count=1 -race`.
  - Run type checker and vet: `go vet ./...`.
  - Check code formatting: verify `gofmt -s -l .` produces empty output.
  - Build binary: `go build -o upp ./cmd/upp`.
  - Run smoke tests: `bash scripts/smoke-test.sh --skip-build`.
