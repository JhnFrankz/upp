# Delta for tool-adapter

## MODIFIED Requirements

### Requirement: Official Adapter Catalog

The system MUST ship built-in adapters for all official tools per platform (see platform-detection catalog).

Each official adapter MUST use the platform-native update mechanism:

| Tool | Linux | macOS | Windows |
|------|-------|-------|---------|
| apt | `apt install --only-upgrade apt` | N/A | N/A |
| brew | `brew update` | `brew update` | N/A |
| winget | N/A | N/A | `winget upgrade winget` |
| scoop | N/A | N/A | `scoop update scoop` |
| nvm | `nvm install stable` | `nvm install stable` | `nvm install stable` |
| npm | `npm update -g` | `npm update -g` | `npm update -g` |
| pnpm | `pnpm update -g` | `pnpm update -g` | `pnpm update -g` |
| bun | `bun upgrade` | `bun upgrade` | `bun upgrade` |
| gh | platform package manager | `brew upgrade gh` | `winget upgrade gh` |
| docker | `apt upgrade docker-ce` | `brew upgrade docker` | `winget upgrade docker` |
| go | manual binary replace | `brew upgrade go` | `winget upgrade go` |
| opencode | curl installer | curl installer | curl installer |

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Linux brew adapter | Platform Linux, brew installed | `brew.update()` | Runs `brew update` (self-only; never `brew upgrade brew`) |
| macOS docker adapter | Platform macOS, docker installed | `docker.update()` | Runs `brew upgrade docker` |
| Linux apt self-only | Platform Linux, apt installed | `apt.update()` | Runs `sudo apt install --only-upgrade apt` (never `apt upgrade`) |
| Windows winget self-only | Platform Windows, winget installed | `winget.update()` | Runs `winget upgrade winget` |
| Windows scoop self-only | Platform Windows, scoop installed | `scoop.update()` | Runs `scoop update scoop` |

(Previously: manager rows updated the manager AND everything it manages — brew ran `brew update && brew upgrade`, apt ran full `apt upgrade`, winget ran `winget upgrade --all`, scoop ran bare `scoop update`.)

### Requirement: Update Gating

Every adapter MUST declare an `UpdatePolicy` (`PolicyGated` or `PolicyAlwaysUpdate`), and the system MUST gate updates on that declaration, not on a CLI-side ID list. The system MUST run `update()` for an adapter declaring `PolicyGated` (apt, npm, pnpm, nvm) only when that adapter's `check()` reported `update_available=true`. Adapters declaring `PolicyAlwaysUpdate` MUST always run their update when requested, regardless of `check()` result: official adapters without detection (brew, bun, docker, gh, go, opencode) and custom adapters report `update_available=false` by design, while winget and scoop report real self-update availability by design (winget parses `winget upgrade`; scoop parses `scoop status`). When a `PolicyGated` adapter's `check()` fails, the system MUST report the failure for that adapter as a structured error per Adapter Error Handling and MUST NOT treat the failed check as `update_available=false` nor report the adapter as current.
(Previously: winget and scoop reported `update_available=true` unconditionally by design; with manager self-update semantics brew is the only always-update adapter reporting `update_available=false` by design, while winget and scoop detect real availability.)

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Official update available | Adapter declaring `PolicyGated` `check()` reports `update_available=true` | Update run | `update()` runs for that adapter |
| Official no update | Adapter declaring `PolicyGated` (apt/npm/pnpm/nvm) `check()` reports `update_available=false` | Update run | `update()` skipped; adapter reported current |
| Stub official exempt | Adapter declaring `PolicyAlwaysUpdate` without detection (brew/bun/docker/gh/go/opencode) reports `update_available=false` | Update run | `update()` still runs |
| Custom exempt | Custom adapter declaring `PolicyAlwaysUpdate` reports `update_available=false` | Update run | `update()` still runs |
| winget/scoop exempt | winget or scoop adapter declaring `PolicyAlwaysUpdate` | Update run | `update()` always runs |
| Dynamic detection | apt reports `update_available=false` | Update run | `update()` skipped |
| Gated check fails | `PolicyGated` adapter `check()` fails during update run | Update run | `update()` skipped; failure reported; adapter never reported current |

## ADDED Requirements

### Requirement: Manager Self-Update Semantics

The brew, apt, winget, and scoop adapters MUST implement self-only semantics: each row reports the manager's own version and self-update availability, and `update()` updates only the manager, never the packages it manages.

| Manager | `check()` | `update()` | Policy |
|---------|-----------|------------|--------|
| brew | `brew --version`; current-only (`Latest=Current`, `UpdateAvailable=false`) | `brew update` ONLY; MUST NOT run `brew upgrade brew` (adapter comment documents portable-ruby footgun); `brew update` is mutating (git fetch) and MUST NOT run inside `check()` | AlwaysUpdate |
| apt | `apt-cache policy apt` — Installed vs Candidate, real availability, no root | `sudo apt install --only-upgrade apt`; MUST NOT run `apt upgrade`; row means "apt package stale" (distro-managed, often intentional) | Gated |
| winget | `winget --version` + parse `winget upgrade` (no args) for winget's own row; requires winget 1.6+ (older: availability unavailable gracefully, no error); version extraction MUST tolerate leading-v 4-part (`v1.8.x`) | `winget upgrade winget` (equiv. `Microsoft.AppInstaller`) | AlwaysUpdate |
| scoop | `scoop status` own row (or `scoop --version`) | `scoop update scoop` | AlwaysUpdate |

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| brew current-only | Homebrew 4.x installed | `brew.check()` | `UpdateAvailable=false`, `Latest=Current` |
| brew never mutates in check | Homebrew installed | `brew.check()` | No `brew update` invoked (no network/git inside check) |
| brew self-update | Sequential/CI run, brew enabled | `brew.update()` | Runs `brew update` only; never `brew upgrade brew` |
| apt real detection | `apt-cache policy apt`: 2.7.3 installed, 2.7.4 candidate | `apt.check()` | `UpdateAvailable=true`, both versions returned |
| apt gated sudo | apt update available | Update run | `update()` runs `sudo apt install --only-upgrade apt` (Gated, sudo) |
| winget tolerant parse | `winget --version` → `v1.8.2311` | `winget.check()` | Current parsed tolerating leading `v`; own row from `winget upgrade` |
| winget old version | winget < 1.6 | `winget.check()` | Availability unavailable gracefully, no error |
| scoop parity | scoop outdated per `scoop status` | `scoop.update()` | Runs `scoop update scoop` |
