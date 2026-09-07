# Proposal: Pacman Package Manager Adapter

## Intent

Extend `upp`'s Linux package manager coverage to Arch Linux and Arch-based distributions (CachyOS, Manjaro, EndeavourOS, Artix) by introducing an official `PacmanAdapter` in `internal/adapters/official/pacman.go`.

Currently, `upp`'s official package manager support on Linux is limited to `apt` (Debian/Ubuntu) and `brew` (Linuxbrew). On Arch Linux systems, users either configure ad-hoc custom tool commands in `config.toml` or cannot leverage `upp`'s unified update workflows for native system packages.

The `PacmanAdapter` provides a native, first-class implementation of `adapters.Adapter`, `adapters.PackageChecker`, and `adapters.PackageUpdater`. It preserves `upp`'s established core architectural tenets:
1. **Self-only manager updates**: `upp update` updates the `pacman` package manager itself (`sudo pacman -S --noconfirm pacman`), strictly avoiding uncontrolled whole-system upgrades (`pacman -Syu`).
2. **Root-free version inspection**: Detection and version queries (`pacman -Q` for installed version, `pacman -Si` for sync repository candidate) execute without root privileges and without mutating local sync databases (`pacman -Sy` is forbidden during check).
3. **Per-package bulk operations**: Implements `CheckPackage(pkg)` and `UpdatePackage(pkg)` (`sudo pacman -S --noconfirm <pkg>`), enabling manager-group bulk updates for owned packages and custom tools configured with `manager = "pacman"`.
4. **Security trust integration**: Operations requiring privilege escalation declare `Privileges: []string{"sudo"}` on their `Result`, properly prompting in interactive TTY sessions and failing closed in non-interactive `--ci` mode.

## Scope

### In Scope

- **Official Pacman Adapter (`internal/adapters/official/pacman.go`)**:
  - Implement `adapters.Adapter`:
    - `Name() string`: Returns `"pacman"`.
    - `Detect() bool`: Returns `lookPath("pacman")` on Linux.
    - `Check() (adapters.UpdateInfo, error)`: Inspects installed vs sync repository candidate versions of `pacman` using local database inspection without root.
    - `Update(dryRun bool) (adapters.Result, error)`: Self-only update (`sudo pacman -S --noconfirm pacman`), supporting dry-run simulation and returning `Privileges: []string{"sudo"}`.
    - `Info() adapters.ToolInfo`: Declares `ID: "pacman"`, `Name: "Pacman Package Manager"`, `Platforms: []string{"linux"}`, `Trust: adapters.TrustOfficial`, `UpdatePolicy: adapters.PolicyGated`, and `Kind: adapters.KindManager`.
  - Implement `adapters.PackageChecker`:
    - `CheckPackage(pkg string) (adapters.UpdateInfo, error)`: Queries installed (`pacman -Q <pkg>`) vs candidate (`pacman -Si <pkg>`) versions for any package managed by pacman without requiring root privileges.
  - Implement `adapters.PackageUpdater`:
    - `UpdatePackage(pkg string) (adapters.Result, error)`: Updates an individual owned or custom package via `sudo pacman -S --noconfirm <pkg>`, returning structured before/after versions and `Privileges: []string{"sudo"}`.
  - Version comparison:
    - Utilize `vercmp` (standard Arch Linux version comparison binary shipped with pacman) when available to accurately compare package versions with epochs and pkgrels (`1:2.0-1`), falling back to string inequality if `vercmp` is unavailable.
- **Registry & Catalog Registration**:
  - Register `&PacmanAdapter{}` in `internal/adapters/official/registry.go` (`AllAdapters()`).
  - Register `ToolEntry{ID: "pacman", Name: "Pacman Package Manager", Platforms: []string{OSLinux}, Kind: adapters.KindManager}` in `internal/platform/catalog.go` (`OfficialTools`).
- **Test Suite Updates**:
  - Update `internal/adapters/official/registry_test.go` to reflect 13 total official adapters.
  - Update `internal/adapters/official/parity_test.go` to include `pacman` in `TestManagerAdaptersImplementPackageInterfaces` and enforce parity between `AllAdapters()` and `OfficialTools`.
  - Update `internal/adapters/official/info_test.go`, `detect_test.go`, and `adapter_test.go` to include `pacman`.
  - Add comprehensive unit tests in `internal/adapters/official/` covering mock command output parsing (single-repo, multi-repo priority, missing packages, unparseable output, timeouts, dry-run, live execution, and sudo privilege reporting).
- **Specification Updates**:
  - `openspec/specs/tool-adapter/spec.md`: Add `pacman` to the Official Adapter Catalog, Manager Self-Update Semantics table, and Update Gating rules.
  - `openspec/specs/platform-detection/spec.md`: Add `pacman` to the Linux tool catalog.
  - `openspec/specs/tool-ownership-model/spec.md`: Declare `pacman` as a `KindManager` adapter implementing `PackageChecker` and `PackageUpdater`.
  - `openspec/specs/security-model/spec.md`: Add `pacman` to the official package managers list and sudo-gated manager execution rules.

### Out of Scope

- **AUR Helpers (`yay`, `paru`, `pikaur`, etc.)**: Support for Arch User Repository helpers is excluded. AUR packages build from source PKGBUILDs and involve interactive prompts, foreign repositories, and distinct privilege escalation flows beyond pacman's binary sync databases.
- **Dynamic Multi-Manager Ownership for Linux Built-in Tools (`gh`, `docker`)**: Currently on Linux, `gh` and `docker` declare `Manager["linux"] = "apt"`. Introducing dynamic distro-detection (e.g. routing `gh` to `pacman` when running on Arch vs `apt` on Debian) is deferred to a dedicated distro-detection / dynamic ownership change to keep this change focused and avoid regressions on Debian/Ubuntu systems.
- **System-Wide Distribution Upgrades (`pacman -Syu`)**: `upp` explicitly manages individual developer tools and packages. Whole-system rolling updates are outside `upp`'s product model.
- **Mutating Database Synchronization (`pacman -Sy`) in Check**: Running `pacman -Sy` inside `Check()` is strictly prohibited to avoid the Arch Linux partial upgrade footgun and prevent network calls during read-only inspection.
- **Non-Linux Platforms**: `pacman` is strictly scoped to the `linux` platform.

## Capabilities

### New Capabilities
None (no new OpenSpec specification domains are created).

### Modified Capabilities
- `tool-adapter`:
  - Official Adapter Catalog: Linux platform catalog gains `pacman` with native self-only command `sudo pacman -S --noconfirm pacman`.
  - Manager Self-Update Semantics: Table adds `pacman` (`check()` queries `pacman -Q pacman` installed vs `pacman -Si pacman` candidate; `update()` runs `sudo pacman -S --noconfirm pacman`; `UpdatePolicy` is `PolicyGated`).
  - Update Gating: `pacman` declared as `PolicyGated`, running self-update only when `Check()` reports `UpdateAvailable=true`.
  - Subprocess Timeouts: Standard adapter check and update timeouts apply to pacman subprocesses.
- `platform-detection`:
  - Tool Catalog: Linux catalog adds `pacman` (`ID: "pacman"`, `Kind: adapters.KindManager`, `Platforms: ["linux"]`).
- `tool-ownership-model`:
  - Tool Ownership Declaration: `pacman` declares `KindManager`.
  - Resolved Owner Update Delegation: `pacman` implements `PackageChecker` (`CheckPackage`) and `PackageUpdater` (`UpdatePackage`) to support owned tools and custom tools configured with `manager = "pacman"`.
- `security-model`:
  - Official Tool Integrity: `pacman` recognized as an official package manager adapter (`TrustOfficial`).
  - Privileged Execution: `pacman` self-update and package updates declare required privileges (`Privileges: ["sudo"]`), enforcing confirmation prompts in interactive sessions and failing closed in `--ci`.

## Approach

1. **Adapter Architecture (`internal/adapters/official/pacman.go`)**:
   - Model structure directly after `internal/adapters/official/apt.go`.
   - **Detection**:
     ```go
     func (a *PacmanAdapter) Detect() bool {
         return lookPath("pacman")
     }
     ```
   - **Version Checking (`Check` and `CheckPackage`)**:
     - Query installed version:
       `bash -o pipefail -c 'pacman -Q <pkg> 2>/dev/null | awk "{print \$2}"'`
       If package is not installed or command fails, version resolves to `"unknown"`.
     - Query latest available candidate version:
       `bash -o pipefail -c 'pacman -Si <pkg> 2>/dev/null | grep -E "^Version" | head -1 | awk "{print \$3}"'`
       Extracts the first matching `Version` field from sync databases. In multi-repository configurations (e.g. CachyOS + Arch core/extra), taking the first repository match respects pacman's configured repository priority order in `/etc/pacman.conf`.
     - Evaluate update availability:
       When `current != "unknown"` and `latest != "unknown"`:
       If `vercmp` is available on PATH, execute `vercmp <latest> <current>`. An exit or output > 0 indicates `UpdateAvailable = true`. If `vercmp` is unavailable, fallback to string inequality `current != latest`.
     - Fails closed: Non-zero exit codes from pacman (other than package-not-found) propagate as structured errors per `Adapter Error Handling`.
   - **Self-Update (`Update`)**:
     - In dry-run mode (`dryRun = true`), returns `Result{Success: true, Before: before, After: before}` without invoking subprocesses.
     - In live mode (`dryRun = false`), executes `sudo pacman -S --noconfirm pacman` bounded by the adapter update timeout.
     - Evaluates stderr for `error:` strings, returning `Result` with `Success: false` and formatted error if update fails.
     - Always populates `Privileges: []string{"sudo"}` on the returned `Result`.
   - **Per-Package Update (`UpdatePackage`)**:
     - Executes `sudo pacman -S --noconfirm <pkg>` bounded by adapter timeout.
     - Returns `Result` with before and after versions and `Privileges: []string{"sudo"}`.
   - **Metadata (`Info`)**:
     ```go
     func (a *PacmanAdapter) Info() adapters.ToolInfo {
         return adapters.ToolInfo{
             ID:           "pacman",
             Name:         "Pacman Package Manager",
             Platforms:    []string{"linux"},
             Trust:        adapters.TrustOfficial,
             UpdatePolicy: adapters.PolicyGated,
             Kind:         adapters.KindManager,
         }
     }
     ```

2. **Registry & Catalog Wiring**:
   - In `internal/adapters/official/registry.go`:
     - Add `&PacmanAdapter{}` to `AllAdapters()`.
   - In `internal/platform/catalog.go`:
     - Add `{ID: "pacman", Name: "Pacman Package Manager", Platforms: []string{OSLinux}, Kind: adapters.KindManager}` to `OfficialTools`.

3. **Parity and Golden Tests**:
   - `internal/adapters/official/registry_test.go`: Update `TestAllAdaptersCount` to expect 13 adapters; verify pacman is registered and consistent.
   - `internal/adapters/official/parity_test.go`: Update `TestManagerAdaptersImplementPackageInterfaces` to include `info.ID == "pacman"`; verify catalog ⇄ adapter sync.
   - `internal/adapters/official/info_test.go`: Add golden metadata expectation for `pacman`.
   - `internal/adapters/official/detect_test.go` and `adapter_test.go`: Add `pacman` test cases.

4. **Unit Tests**:
   - Mock command runner table tests in `internal/adapters/official/check_test.go` and `update_test.go` (or `pacman_test.go`) covering:
     - Detection success/failure via `lookPath`.
     - Clean installed version extraction (`pacman 7.1.0.r9.g54d9411-4`).
     - Sync candidate extraction with single and multiple repository outputs.
     - Version comparison via `vercmp` and fallback inequality.
     - Missing package detection (`error: package '...' was not found`).
     - Command timeout and error propagation.
     - Dry-run update and live update execution with sudo privilege attachment.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/adapters/official/pacman.go` | Created | Implements `PacmanAdapter` (`Adapter`, `PackageChecker`, `PackageUpdater`) |
| `internal/adapters/official/registry.go` | Modified | Registers `&PacmanAdapter{}` in `AllAdapters()` |
| `internal/platform/catalog.go` | Modified | Adds `pacman` entry to `OfficialTools` |
| `internal/adapters/official/registry_test.go` | Modified | Updates adapter count (12 → 13) and adapter consistency tests |
| `internal/adapters/official/parity_test.go` | Modified | Includes `pacman` in manager package interface verification and catalog parity |
| `internal/adapters/official/info_test.go` | Modified | Adds golden metadata assertion for `pacman` |
| `internal/adapters/official/detect_test.go` | Modified | Adds `pacman` detection test case |
| `internal/adapters/official/adapter_test.go` | Modified | Adds `pacman` adapter name verification |
| `internal/adapters/official/check_test.go` | Modified | Adds unit tests for pacman check and package check parsing |
| `internal/adapters/official/update_test.go` | Modified | Adds unit tests for pacman self-update and package update execution |
| `openspec/specs/tool-adapter/spec.md` | Modified | Adds pacman to official catalog, self-update semantics, and gating tables |
| `openspec/specs/platform-detection/spec.md` | Modified | Adds pacman to Linux official catalog specification |
| `openspec/specs/tool-ownership-model/spec.md` | Modified | Adds pacman as a declared manager adapter with package updater/checker capabilities |
| `openspec/specs/security-model/spec.md` | Modified | Adds pacman to official tool integrity list and sudo elevation specifications |

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| **Arch Linux Partial Upgrade Footgun** (`pacman -Sy` without upgrade causes broken system libraries) | High (if misused) | High | `upp` NEVER runs `pacman -Sy` inside `Check()` or `Update()`. Version queries strictly read local databases (`pacman -Q`, `pacman -Si`), and package updates invoke `sudo pacman -S --noconfirm <pkg>`. |
| **Multi-Repository Candidate Parsing** (multiple repos define the same package name with different versions) | Medium | Medium | Extracted candidate queries parse only the first matching `Version` field (`head -1`), adhering to pacman's configured repository precedence order in `/etc/pacman.conf`. |
| **Sudo Elevation Blocking Unattended CI** (`sudo pacman` prompts for password in non-interactive runs) | Medium | Medium | Live update results declare `Privileges: []string{"sudo"}`. upp's security model automatically prompts in interactive TTY and fails closed with non-zero exit in `--ci`. |
| **Registry & Catalog Parity Drift** (AllAdapters and OfficialTools fall out of sync) | Low | Medium | Enforced by existing `parity_test.go` assertions (`TestEveryAdapterIsInCatalog`, `TestEveryCatalogEntryHasAdapter`, `TestCatalogOwnershipMatchesAdapter`). |
| **Arch Version Syntax Nuances** (pkgrel, epoch like `1:7.1.0-2`) | Low | Low | Uses `vercmp` binary when present on the system to compare Arch package versions accurately, falling back to string inequality. |

## Rollback Plan

This change is strictly additive to the official adapter catalog and does not alter existing adapters (`apt`, `brew`, `winget`, `scoop`) or change configuration schemas.

If any regression occurs:
1. Revert the commit(s) introducing `internal/adapters/official/pacman.go`, registry entry, catalog entry, test additions, and spec deltas.
2. The codebase returns to 12 official adapters with zero migrations or residual state.
3. Any custom tools in `config.toml` that declared `manager = "pacman"` will gracefully fallback to standalone operation per existing forward-compatibility rules (`IsManager("pacman") == false`).

## Success Criteria

- [ ] `PacmanAdapter` implemented in `internal/adapters/official/pacman.go` implementing `Adapter`, `PackageChecker`, and `PackageUpdater`.
- [ ] Detection returns `true` on Linux when `pacman` is present on PATH.
- [ ] `Check()` queries installed vs candidate version of `pacman` root-free without network or sync DB mutation.
- [ ] `Update()` performs self-only upgrade (`sudo pacman -S --noconfirm pacman`) reporting `Privileges: ["sudo"]`.
- [ ] `CheckPackage()` and `UpdatePackage()` accurately inspect and upgrade individual packages.
- [ ] `AllAdapters()` returns 13 adapters; `platform.OfficialTools` includes `pacman` on Linux.
- [ ] Parity tests (`parity_test.go`, `registry_test.go`, `info_test.go`) pass.
- [ ] Spec deltas in `tool-adapter`, `platform-detection`, `tool-ownership-model`, and `security-model` documented.
