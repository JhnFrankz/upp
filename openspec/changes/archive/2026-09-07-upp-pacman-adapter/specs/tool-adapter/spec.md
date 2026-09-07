# Delta for tool-adapter

## MODIFIED Requirements

### Requirement: Official Adapter Catalog

The system MUST ship built-in adapters for all official tools per platform (see platform-detection catalog).

Each official adapter MUST use the platform-native update mechanism. An owned tool MUST delegate its update to its owning manager rather than run its own hardcoded manager command.

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

(Previously: the catalog contained 12 official tools without pacman; Linux package manager support was limited to apt and brew.)

### Requirement: Manager Self-Update Semantics

The brew, apt, pacman, winget, and scoop adapters MUST implement self-only semantics: each row reports the manager's own version and self-update availability, and `update()` updates only the manager, never the packages it manages.

| Manager | `check()` | `update()` | Policy |
|---------|-----------|------------|--------|
| brew | `brew --version`; current-only (`Latest=Current`, `UpdateAvailable=false`) | `brew update` ONLY; MUST NOT run `brew upgrade brew` (adapter comment documents portable-ruby footgun); `brew update` is mutating (git fetch) and MUST NOT run inside `check()` | AlwaysUpdate |
| apt | `apt-cache policy apt` — Installed vs Candidate, real availability, no root | `sudo apt install --only-upgrade apt`; MUST NOT run `apt upgrade`; row means "apt package stale" (distro-managed, often intentional) | Gated |
| pacman | `pacman -Q pacman` installed vs `pacman -Si pacman` candidate — real availability, no root | `sudo pacman -S --noconfirm pacman`; MUST NOT run `pacman -Syu` or `pacman -Sy`; row means "pacman package stale" | Gated |
| winget | `winget --version` + parse `winget upgrade` (no args) for winget's own row; requires winget 1.6+ (older: availability unavailable gracefully, no error); version extraction MUST tolerate leading-v 4-part (`v1.8.x`) | `winget upgrade winget` (equiv. `Microsoft.AppInstaller`) | AlwaysUpdate |
| scoop | `scoop status` own row (or `scoop --version`) | `scoop update scoop` | AlwaysUpdate |

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| brew current-only | Homebrew 4.x installed | `brew.check()` | `UpdateAvailable=false`, `Latest=Current` |
| brew never mutates in check | Homebrew installed | `brew.check()` | No `brew update` invoked (no network/git inside check) |
| brew self-update | Sequential/CI run, brew enabled | `brew.update()` | Runs `brew update` only; never `brew upgrade brew` |
| apt real detection | `apt-cache policy apt`: 2.7.3 installed, 2.7.4 candidate | `apt.check()` | `UpdateAvailable=true`, both versions returned |
| apt gated sudo | apt update available | Update run | `update()` runs `sudo apt install --only-upgrade apt` (Gated, sudo) |
| pacman real detection | `pacman -Q pacman`: 6.1.0 installed, 7.0.0 candidate | `pacman.check()` | `UpdateAvailable=true`, both versions returned without root |
| pacman self-update sudo | pacman update available | Update run | `update()` runs `sudo pacman -S --noconfirm pacman` (Gated, sudo) |
| pacman check never mutates | pacman installed | `pacman.check()` | Reads local sync DB; MUST NOT run `pacman -Sy` |
| winget tolerant parse | `winget --version` → `v1.8.2311` | `winget.check()` | Current parsed tolerating leading `v`; own row from `winget upgrade` |
| winget old version | winget < 1.6 | `winget.check()` | Availability unavailable gracefully, no error |
| scoop parity | scoop outdated per `scoop status` | `scoop.update()` | Runs `scoop update scoop` |

(Previously: manager self-update covered only brew, apt, winget, and scoop; pacman was not included.)

### Requirement: Update Gating

Every adapter MUST declare an `UpdatePolicy` (`PolicyGated` or `PolicyAlwaysUpdate`), and the system MUST gate updates on that declaration, not on a CLI-side ID list. The system MUST run `update()` for an adapter declaring `PolicyGated` (apt, pacman, npm, pnpm, nvm) only when that adapter's `check()` reported `update_available=true`. Adapters declaring `PolicyAlwaysUpdate` MUST always run their update when requested, regardless of `check()` result: official adapters without detection (brew, bun, opencode) and custom adapters report `update_available=false` by design, while winget and scoop report real self-update availability by design. An owned tool (gh, docker, go) MUST NOT be gated independently: its update delegates to its owning manager, and the manager's `UpdatePolicy` governs whether the delegated update runs; the owned tool's own `UpdatePolicy` MUST NOT apply to the delegated path. For a manager-group bulk update, the manager's `UpdatePolicy` gates the GROUP's availability (not the manager's own self-only row): a `PolicyGated` manager runs its group update only when any owned package reports availability; a `PolicyAlwaysUpdate` manager runs its group update when requested. When a `PolicyGated` adapter's `check()` fails, the system MUST report the failure for that adapter as a structured error per Adapter Error Handling and MUST NOT treat the failed check as `update_available=false` nor report the adapter as current.

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

(Previously: `PolicyGated` adapters included only apt, npm, pnpm, and nvm; pacman was not defined.)

### Requirement: Version Comparison

Adapters MUST return semver-compatible version strings when available. Adapters SHOULD normalize version formats across platforms. The nvm adapter MUST determine update availability by semantic version comparison (leading `v` prefix tolerated), not string inequality: current > latest MUST report `update_available=false` (no downgrade); when either version cannot be parsed as semver, nvm MUST NOT report an update based on string inequality alone and reports unknown (`update_available=false`) without error. The pacman adapter SHOULD utilize `vercmp` when available on PATH to compare Arch package versions (accounting for epochs and package releases); if `vercmp` is unavailable, the pacman adapter MUST fall back to string inequality. Other adapters (apt, npm, pnpm, and non-semver official tools) use their own detection contract; the general semver-comparison rule is scoped to the nvm adapter.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Semver version | Node v20.11.0 | `check()` | Returns `current="20.11.0"` |
| Non-semver | Docker "24.0.7" | `check()` | Returns raw version string |
| Newer current | nvm current `v26.7.0`, latest `v24.19.0` | `check()` | `update_available=false`; no downgrade |
| Older current | nvm current `v18.0.0`, latest `v20.11.0` | `check()` | `update_available=true` |
| Equal versions | nvm current `v20.11.0`, latest `20.11.0` | `check()` | `update_available=false` |
| Unparseable | nvm current `v26.7.0`, latest `stable` | `check()` | `update_available=false`, no error (unknown) |
| Pacman vercmp epoch order | pacman current `1:6.1.0-1`, candidate `7.0.0-1`, `vercmp` on PATH | `check()` | `UpdateAvailable=false` (current epoch takes precedence) |
| Pacman vercmp newer candidate | pacman current `6.1.0-1`, candidate `6.1.0-2`, `vercmp` on PATH | `check()` | `UpdateAvailable=true` |
| Pacman fallback inequality | pacman current `6.1.0-1`, candidate `7.0.0-1`, `vercmp` missing | `check()` | `UpdateAvailable=true` (via string inequality) |

(Previously: version comparison addressed semver for nvm and general normalization without specifying Arch Linux `vercmp` support for pacman.)

### Requirement: Check Failure Signal

`check()` MUST return a structured error (tool name, operation, exit code — per Adapter Error Handling) when its update-detection subprocess fails, for apt, pacman, nvm, npm, and pnpm. A subprocess failure is a non-zero exit code EXCEPT the documented npm/pnpm `outdated` convention where exit code 1 means updates are available (a valid detection, not a failure); timeout (exit 124 via the `timeout 15` wrapper) and other non-zero exits are failures. Empty subprocess output MUST NOT be treated as failure: a detection subprocess that succeeds with empty output reports unknown status (`update_available=false`) without error. The npm and pnpm adapters MUST NOT mask detection subprocess exit codes. The CLI MUST surface a failed check as `StatusFailed` for that adapter and MUST NOT report `StatusCurrent` for a failed check.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Detection fails | apt/pacman/nvm/npm/pnpm detection subprocess exits non-zero (npm/pnpm: any code except the documented exit-1-outdated) | `check()` | Structured error with tool name, operation, exit code |
| Empty output | Detection subprocess exits 0 with empty output | `check()` | Unknown status (`update_available=false`), no error |
| Gated check fails in run | `PolicyGated` adapter `check()` fails | Update run | Failure surfaced as `StatusFailed`; update skipped; not `StatusCurrent` |
| npm/pnpm maskless | npm/pnpm detection subprocess exits non-zero (incl. timeout 124) | `check()` through timeout wrapper | Failure propagates; exit code not swallowed |
| npm/pnpm exit-1 outdated | npm/pnpm `outdated` exits 1 (updates available) | `check()` | Valid detection: `update_available=true`, no error |
| Pacman detection fails | `pacman -Si pacman` exits non-zero (database corruption or sync error) | `pacman.check()` | Structured error; surfaced as `StatusFailed` |

(Previously: detection failure error enforcement covered apt, nvm, npm, and pnpm; pacman was not included.)

## NEW Requirements (if any)
