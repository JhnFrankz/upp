# Delta for tool-adapter

## MODIFIED Requirements

### Requirement: Official Adapter Catalog

The system MUST ship built-in adapters for all official tools per platform (see platform-detection catalog).

Each official adapter MUST use the platform-native update mechanism. An owned tool MUST delegate its update to its owning manager rather than run its own hardcoded manager command.

The `uv` adapter MUST be registered as an official tool adapter across Linux, macOS, and Windows with `ID: "uv"`, `Kind: KindTool`, `Trust: TrustOfficial`, `UpdatePolicy: PolicyGated`, and `Platforms: ["linux", "macos", "windows"]`. As a standalone tool on all platforms, `uv` MUST NOT declare an owning manager (`Manager=nil`, `ManagerPackage=nil`).

| Tool | Linux | macOS | Windows |
|------|-------|-------|---------|
| apt | `apt install --only-upgrade apt` | N/A | N/A |
| brew | `brew update` | `brew update` | N/A |
| pacman | `sudo pacman -S --noconfirm pacman` | N/A | N/A |
| winget | N/A | N/A | `winget upgrade winget` |
| scoop | N/A | N/A | `scoop update scoop` |
| nvm | `nvm install stable` | `nvm install stable` | `nvm install stable` |
| npm | `npm update -g` | `npm update -g` | `npm update -g` |
| pnpm | `pnpm update -g` | `pnpm update -g` | `pnpm update -g` |
| bun | `bun upgrade` | `bun upgrade` | `bun upgrade` |
| uv | `uv self update` && `uv tool upgrade --all` | `uv self update` && `uv tool upgrade --all` | `uv self update` && `uv tool upgrade --all` |
| gh | → apt | → brew | → winget |
| docker | → apt | → brew | → winget |
| go | manual binary replace | → brew | → winget |
| opencode | curl installer | curl installer | curl installer |

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Linux brew adapter | Platform Linux, brew installed | `brew.update()` | Runs `brew update` (never `brew upgrade brew`) |
| Linux pacman adapter | Platform Linux, pacman installed | `pacman.update()` | Runs `sudo pacman -S --noconfirm pacman` |
| macOS docker delegates | Platform macOS, docker enabled | `docker.update()` | Delegates to `brew.update()` (owns docker on macOS) |
| Linux gh delegates | Platform Linux, gh enabled | `gh.update()` | Delegates to `apt.update()` (owns gh on Linux) |
| Windows gh delegates | Platform Windows, gh enabled | `gh.update()` | Delegates to `winget.update()` (owns gh on Windows) |
| Linux apt self-only | Platform Linux, apt installed | `apt.update()` | Runs `sudo apt install --only-upgrade apt` (never `apt upgrade`) |
| Linux uv adapter | Platform Linux, uv installed | `uv.update(false)` | Executes dual-scope `uv self update` (with external package manager bypass) and `uv tool upgrade --all` |
| macOS uv adapter | Platform macOS, uv installed | `uv.update(false)` | Executes dual-scope update as standalone tool |
| Windows uv adapter | Platform Windows, uv installed | `uv.update(false)` | Executes dual-scope update as standalone tool |

(Previously: the official adapter catalog included 13 official adapters across managers, runtimes, and tools without official Python developer tooling support; `uv` was not part of the catalog.)

### Requirement: Update Gating

Every adapter MUST declare an `UpdatePolicy` (`PolicyGated` or `PolicyAlwaysUpdate`), and the system MUST gate updates on that declaration, not on a CLI-side ID list. The system MUST run `update()` for an adapter declaring `PolicyGated` (apt, pacman, npm, pnpm, nvm, uv) only when that adapter's `check()` reported `update_available=true`. Adapters declaring `PolicyAlwaysUpdate` MUST always run their update when requested, regardless of `check()` result: official adapters without detection (brew, bun, opencode) and custom adapters report `update_available=false` by design, while winget and scoop report real self-update availability by design. An owned tool (gh, docker, go) MUST NOT be gated independently: its update delegates to its owning manager, and the manager's `UpdatePolicy` governs whether the delegated update runs; the owned tool's own `UpdatePolicy` MUST NOT apply to the delegated path. For a manager-group bulk update, the manager's `UpdatePolicy` gates the GROUP's availability (not the manager's own self-only row): a `PolicyGated` manager runs its group update only when any owned package reports availability; a `PolicyAlwaysUpdate` manager runs its group update when requested. When a `PolicyGated` adapter's `check()` fails, the system MUST report the failure for that adapter as a structured error per Adapter Error Handling and MUST NOT treat the failed check as `update_available=false` nor report the adapter as current.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Owned inherits gated | docker owned by apt (Gated) on Linux, apt check reports no update | docker delegated update | Delegated `apt.update()` skipped; docker reported current |
| Owned inherits always | gh owned by brew (AlwaysUpdate) on macOS | gh delegated update | `update()` runs (delegates to brew) |
| Stub official exempt | Adapter declaring `PolicyAlwaysUpdate` without detection (brew/bun/opencode) reports `update_available=false` | Update run | `update()` still runs |
| Gated check fails | `PolicyGated` adapter `check()` fails during update run | Update run | `update()` skipped; failure reported; adapter never reported current |
| Gated group gates on group availability | apt (Gated) group, no owned package has an update | `upp update` (default run) | Group skipped; no owned tool updated |
| AlwaysUpdate group runs | brew (AlwaysUpdate) group | `upp update` (default run) | Group update runs regardless of check result |
| Pacman gated check passes | `pacman` declares `PolicyGated`, check reports `update_available=true` | Update run | `pacman.update()` executes |
| Pacman gated check reports current | `pacman` declares `PolicyGated`, check reports `update_available=false` | Update run | `pacman.update()` skipped; reported current |
| uv gated check passes | `uv` declares `PolicyGated`, `uv.check()` reports `update_available=true` | Update run | `uv.update()` executes |
| uv gated check reports current | `uv` declares `PolicyGated`, `uv.check()` reports `update_available=false` | Update run | `uv.update()` skipped; reported current |
| uv gated check fails | `uv` declares `PolicyGated`, `uv.check()` encounters unhandled failure | Update run | `uv.update()` skipped; failure reported; not reported current |

(Previously: `uv` was not an official adapter; the list of `PolicyGated` adapters included apt, pacman, npm, pnpm, and nvm.)

## NEW Requirements

### Requirement: Dual-Scope Toolchain Execution and Resilience

The `uv` adapter MUST implement dual-scope inspection and execution across Linux, macOS, and Windows.

During inspection (`Check()`):
1. The adapter MUST retrieve the installed version via `uv --version`.
2. The adapter MUST inspect binary self-update availability by executing `uv self update --dry-run`.
   - If the command exits with code 2 and standard error or output indicates that uv was installed through an external package manager (`"error: uv was installed through an external package manager"`), the adapter MUST gracefully bypass binary self-update inspection without error, treating self-update availability as false.
   - If the command exits non-zero with any other error code, the adapter MUST fail closed and return a structured error per Adapter Error Handling.
   - If the command succeeds, the adapter MUST mark self-update as available (`true`) if output indicates a pending update.
3. The adapter MUST inspect globally installed tool updates by executing `uv tool list --outdated`.
   - If the command exits non-zero, the adapter MUST fail closed and return a structured error.
   - If the command succeeds, the adapter MUST inspect output: clean states such as `"No tools installed"` or `"No outdated tools"`, or empty output, MUST report tool updates as not available (`false`). Any non-empty output listing outdated packages MUST mark tool updates as available (`true`).
4. The adapter MUST report `UpdateAvailable: true` if either binary self-update or any global tool update is available.

During update execution (`Update(dryRun)`):
1. When `dryRun` is `true`, the adapter MUST NOT execute mutating commands and MUST return a successful `Result` with identical before and after versions.
2. When `dryRun` is `false`:
   - Step 1: The adapter MUST execute `uv self update`. If execution fails with exit code 2 and external package manager output, the adapter MUST gracefully ignore the error and proceed to tool upgrades. If execution fails with any other non-zero exit code or error, the adapter MUST abort and return a structured failure `Result`.
   - Step 2: The adapter MUST execute `uv tool upgrade --all`. If execution fails, the adapter MUST abort and return a structured failure `Result`.
   - Step 3: The adapter MUST query the updated version via `uv --version` and return a successful `Result` reporting before and after versions.

Subprocess invocations during `Check()` and `Update()` MUST be bounded by the standard adapter timeouts (`CheckTimeout` of 15s and `UpdateTimeout` of 120s).

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Check with self-update available | `uv self update --dry-run` reports an update is available | `uv.check()` called | Returns `UpdateAvailable=true` |
| Check with outdated tools | `uv tool list --outdated` lists outdated CLI packages | `uv.check()` called | Returns `UpdateAvailable=true` |
| Check up to date | Both binary and tools report current ("No outdated tools") | `uv.check()` called | Returns `UpdateAvailable=false`, `CurrentVersion` set |
| Check external manager bypass | `uv` installed via package manager; `uv self update --dry-run` exits code 2 with external package manager message, tools up-to-date | `uv.check()` called | Bypasses self-update without error; returns `UpdateAvailable=false` |
| Check external manager with outdated tools | `uv` installed via package manager; self-update exits code 2, `uv tool list --outdated` lists outdated packages | `uv.check()` called | Bypasses self-update without error; returns `UpdateAvailable=true` |
| Check subprocess error | `uv tool list --outdated` exits with unexpected non-zero code | `uv.check()` called | Fails closed returning structured error per Adapter Error Handling |
| Check subprocess timeout | Subprocess hangs beyond `CheckTimeout` (15s) | `uv.check()` called | Terminated; returns structured timeout error |
| Update dry run | Pending updates detected | `uv.update(true)` called | Returns `Success=true` with `Before` and `After` versions identical; no subprocess executed |
| Update live dual-scope succeeds | Standalone `uv` installation; update available | `uv.update(false)` called | Executes `uv self update` then `uv tool upgrade --all`; returns `Success=true` with before/after versions |
| Update live external manager bypass | `uv` managed by brew/winget/apt; `uv self update` exits code 2 | `uv.update(false)` called | Bypasses self-update gracefully; executes `uv tool upgrade --all`; returns `Success=true` |
| Update live tool upgrade fails | `uv tool upgrade --all` exits non-zero | `uv.update(false)` called | Aborts and returns `Success=false` with structured error details |
| Update live self-update fails unexpectedly | `uv self update` exits non-zero (non-code-2) | `uv.update(false)` called | Aborts before tool upgrade; returns `Success=false` with structured error |
| Update subprocess timeout | `uv tool upgrade --all` hangs beyond `UpdateTimeout` (120s) | `uv.update(false)` called | Terminated; returns structured timeout error |
