# Tool Adapter Interface Specification

## Purpose

Define the contract every tool adapter (official or custom) must implement. Each adapter handles one tool on one platform: detect, check, update, list.

## Requirements

### Requirement: Adapter Interface

Every adapter MUST implement four operations:

- `detect() → bool` — is this tool installed?
- `check() → UpdateInfo` — current version + latest available + update available?
- `update() → Result` — perform the update, return success/failure + details
- `list() → ToolInfo` — return installed tool info (name, version, source, owning manager, kind)

`ToolInfo` MUST carry an owning `Manager` map keyed by platform and a `Kind` (`KindManager` for manager adapters, `KindTool` otherwise). A tool with a resolving owner on the current platform reports that manager; a tool with no owner reports no manager. A `ToolInfo` whose `Kind=KindTool` and that has a resolving owner on the current platform MUST also declare a per-manager package-name entry (see Per-Manager Package Mapping), so the owned tool's package under its manager is known.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Tool installed | `nvm` is on PATH | `detect()` called | Returns `true` |
| Tool missing | `nvm` not on PATH | `detect()` called | Returns `false` |
| Update available | Node.js v18 installed, v20 latest | `check()` called | `update_available=true`, versions returned |
| No update | Node.js v20 installed, v20 latest | `check()` called | `update_available=false` |
| Update succeeds | Update command exits 0 | `update()` called | `success=true`, before/after versions returned |
| Update fails | Update command exits non-zero | `update()` called | `success=false`, error message returned |
| ToolInfo carries owner | docker installed on Linux | `list()` called | `ToolInfo` includes `Manager["linux"]="apt"`, `Kind=KindTool` |
| Manager carries kind | apt installed on Linux | `list()` called | `ToolInfo.Kind=KindManager` |
| Owned tool carries package | gh owned by apt on Linux | `list()` called | `ToolInfo` declares package `gh` under `Manager["linux"]="apt"` |

(Previously: `list() → ToolInfo` returned only name/version/source; `ToolInfo` had no `Manager` or `Kind` field, and no per-manager package-name field existed.)

### Requirement: Official Adapter Catalog

The system MUST ship built-in adapters for all official tools per platform (see platform-detection catalog).

Each official adapter MUST use the platform-native update mechanism. An owned tool MUST delegate its update to its owning manager rather than run its own hardcoded manager command.

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
| gh | → apt | → brew | → winget |
| docker | → apt | → brew | → winget |
| go | manual binary replace | → brew | → winget |
| opencode | curl installer | curl installer | curl installer |

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Linux brew adapter | Platform Linux, brew installed | `brew.update()` | Runs `brew update` (never `brew upgrade brew`) |
| macOS docker delegates | Platform macOS, docker enabled | `docker.update()` | Delegates to `brew.update()` (owns docker on macOS) |
| Linux gh delegates | Platform Linux, gh enabled | `gh.update()` | Delegates to `apt.update()` (owns gh on Linux) |
| Windows gh delegates | Platform Windows, gh enabled | `gh.update()` | Delegates to `winget.update()` (owns gh on Windows) |
| Linux apt self-only | Platform Linux, apt installed | `apt.update()` | Runs `sudo apt install --only-upgrade apt` (never `apt upgrade`) |

(Previously: gh/docker/go ran their own hardcoded manager commands (e.g. `gh.go` ran `sudo apt update && sudo apt install -y gh`) and appeared as independent update rows duplicating manager logic.)

### Requirement: Per-Manager Package Mapping

Every owned tool adapter (gh, docker, go) MUST declare, per platform, the package name under its owning manager. The system MUST NOT infer a package name from the tool ID (the names differ). Declared minimum mapping:

| Tool | Linux | macOS | Windows |
|------|-------|-------|---------|
| gh | `gh` | `gh` | `gh` |
| docker | `docker-ce` | `docker` | `Docker.Docker` |
| go | `golang` | `golang` | `GoLang.Go` |

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| docker on apt | Platform Linux, docker owned by apt | package resolved | Package = `docker-ce` |
| docker on winget | Platform Windows, docker owned by winget | package resolved | Package = `Docker.Docker` |
| go on brew | Platform macOS, go owned by brew | package resolved | Package = `golang` |
| gh on apt | Platform Linux, gh owned by apt | package resolved | Package = `gh` |

(Previously: owned tools had no per-manager package-name concept; delegated update ran the manager's self-only command, so the owned tool's package was never named.)

### Requirement: Per-Owned-Tool Availability

The system MUST determine a real update for an owned tool by checking the owned tool's package under its manager (e.g. `apt-cache policy gh`), NOT by the manager's self check. The owned tool's delegated `check()` MUST report `update_available=true` when the owned package has a candidate newer than installed.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Owned package update | `apt-cache policy gh` candidate > installed | `gh.check()` | `update_available=true` |
| Owned package current | `apt-cache policy gh` installed == candidate | `gh.check()` | `update_available=false` |
| Owned check fails | `apt-cache policy gh` exits non-zero | `gh.check()` | Structured error; not treated as current |

(Previously: owned tools' `check()` always returned `update_available=false` because it reflected only the manager's self state, so owned tools could never be pending.)

### Requirement: Adapter Error Handling

Adapters MUST return structured errors on failure. Errors MUST include: tool name, operation attempted, exit code (if applicable), and stderr excerpt.

Adapters MUST NOT abort the entire update run on individual tool failure. The orchestrator collects results and reports in summary.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Network failure | `brew update` fails (network) | `brew.update()` | Returns error with exit code and stderr |
| Partial success | 3/5 tools updated | Full update run | Summary shows 3 updated, 2 failed |

### Requirement: Version Comparison

Adapters MUST return semver-compatible version strings when available. Adapters SHOULD normalize version formats across platforms. The nvm adapter MUST determine update availability by semantic version comparison (leading `v` prefix tolerated), not string inequality: current > latest MUST report `update_available=false` (no downgrade); when either version cannot be parsed as semver, nvm MUST NOT report an update based on string inequality alone and reports unknown (`update_available=false`) without error. Other adapters (apt, npm, pnpm, and non-semver official tools) use their own detection contract; the general semver-comparison rule is scoped to the nvm adapter.
(Previously: the requirement stated that all adapters determine update availability by semantic version comparison; in practice only the nvm adapter implements it, so the general wording is narrowed to the nvm adapter.)

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Semver version | Node v20.11.0 | `check()` | Returns `current="20.11.0"` |
| Non-semver | Docker "24.0.7" | `check()` | Returns raw version string |
| Newer current | nvm current `v26.7.0`, latest `v24.19.0` | `check()` | `update_available=false`; no downgrade |
| Older current | nvm current `v18.0.0`, latest `v20.11.0` | `check()` | `update_available=true` |
| Equal versions | nvm current `v20.11.0`, latest `20.11.0` | `check()` | `update_available=false` |
| Unparseable | nvm current `v26.7.0`, latest `stable` | `check()` | `update_available=false`, no error (unknown) |

### Requirement: Update Gating

Every adapter MUST declare an `UpdatePolicy` (`PolicyGated` or `PolicyAlwaysUpdate`), and the system MUST gate updates on that declaration, not on a CLI-side ID list. The system MUST run `update()` for an adapter declaring `PolicyGated` (apt, npm, pnpm, nvm) only when that adapter's `check()` reported `update_available=true`. Adapters declaring `PolicyAlwaysUpdate` MUST always run their update when requested, regardless of `check()` result: official adapters without detection (brew, bun, opencode) and custom adapters report `update_available=false` by design, while winget and scoop report real self-update availability by design. An owned tool (gh, docker, go) MUST NOT be gated independently: its update delegates to its owning manager, and the manager's `UpdatePolicy` governs whether the delegated update runs; the owned tool's own `UpdatePolicy` MUST NOT apply to the delegated path. When a `PolicyGated` adapter's `check()` fails, the system MUST report the failure for that adapter as a structured error per Adapter Error Handling and MUST NOT treat the failed check as `update_available=false` nor report the adapter as current.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Owned inherits gated | docker owned by apt (Gated) on Linux, apt check reports no update | docker delegated update | Delegated `apt.update()` skipped; docker reported current |
| Owned inherits always | gh owned by brew (AlwaysUpdate) on macOS | gh delegated update | `update()` runs (delegates to brew) |
| Stub official exempt | Adapter declaring `PolicyAlwaysUpdate` without detection (brew/bun/opencode) reports `update_available=false` | Update run | `update()` still runs |
| Gated check fails | `PolicyGated` adapter `check()` fails during update run | Update run | `update()` skipped; failure reported; adapter never reported current |

(Previously: docker, gh, and go were independent `PolicyAlwaysUpdate` stubs whose `update()` always ran; ownership did not exist, so gating was per-adapter with no manager inheritance.)

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

### Requirement: Subprocess Timeouts

Every subprocess launched by `check()` or `update()` MUST be bounded by a timeout, for both custom and official adapters. When a subprocess exceeds its timeout, the system MUST terminate the subprocess, MUST return a structured timeout error for that tool, and MUST NOT block the remainder of the update run. `check()` and `update()` MUST use distinct timeouts appropriate to each operation; exact durations are a design decision, and slow package managers (brew, nvm, apt) MAY warrant longer update timeouts.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Update completes in time | `brew update` finishes within the update timeout | `update()` | Success result returned |
| Update times out | `brew update` hangs beyond the update timeout | `update()` | Structured timeout error; other tools still update |
| Check times out | `check()` subprocess hangs beyond the check timeout | `check()` | Structured timeout error; update run continues |
| Custom times out | Custom shell command exceeds the timeout | `update()` | Structured timeout error returned |

### Requirement: Go Adapter Architecture

The go adapter's Linux download MUST fetch the Go toolchain tarball matching the running process architecture: `linux-amd64` on amd64 hosts and `linux-arm64` on arm64 hosts. The adapter MUST NOT hardcode a single architecture.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| amd64 host | Process runs on linux/amd64 | go `check()` or `update()` | Downloads `go*linux-amd64.tar.gz` |
| arm64 host | Process runs on linux/arm64 | go `check()` or `update()` | Downloads `go*linux-arm64.tar.gz` |


### Requirement: Check Failure Signal

`check()` MUST return a structured error (tool name, operation, exit code — per Adapter Error Handling) when its update-detection subprocess fails, for apt, nvm, npm, and pnpm. A subprocess failure is a non-zero exit code EXCEPT the documented npm/pnpm `outdated` convention where exit code 1 means updates are available (a valid detection, not a failure); timeout (exit 124 via the `timeout 15` wrapper) and other non-zero exits are failures. Empty subprocess output MUST NOT be treated as failure: a detection subprocess that succeeds with empty output reports unknown status (`update_available=false`) without error. The npm and pnpm adapters MUST NOT mask detection subprocess exit codes. The CLI MUST surface a failed check as `StatusFailed` for that adapter and MUST NOT report `StatusCurrent` for a failed check.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Detection fails | apt/nvm/npm/pnpm detection subprocess exits non-zero (npm/pnpm: any code except the documented exit-1-outdated) | `check()` | Structured error with tool name, operation, exit code |
| Empty output | Detection subprocess exits 0 with empty output | `check()` | Unknown status (`update_available=false`), no error |
| Gated check fails in run | `PolicyGated` adapter `check()` fails | Update run | Failure surfaced as `StatusFailed`; update skipped; not `StatusCurrent` |
| npm/pnpm maskless | npm/pnpm detection subprocess exits non-zero (incl. timeout 124) | `check()` through timeout wrapper | Failure propagates; exit code not swallowed |
| npm/pnpm exit-1 outdated | npm/pnpm `outdated` exits 1 (updates available) | `check()` | Valid detection: `update_available=true`, no error |

### Requirement: Custom Adapter Privileges & Execution

Custom adapters MUST analyze configured command strings statically for privilege escalation tokens (e.g., `sudo`, `doas`, `runas`, `admin`). Custom adapter `Update(dryRun)` MUST populate `Result.Privileges` with detected privileges for both dry-run (`dryRun=true`) and live execution (`dryRun=false`).

Custom adapter `Check()` and `Update()` MUST fail closed with a structured error when the base command executable is not detected on PATH (`Detect() == false`), avoiding unnecessary subshell invocations. When the base binary is present on PATH:
- `Update(true)` MUST return a success `Result` with before/after command strings and detected privileges without invoking any subprocess.
- `Update(false)` MUST execute the update command via the platform shell bounded by timeout and return the execution `Result` with detected privileges.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Dry-run with sudo | Custom tool configured with `sudo apt upgrade` and binary present on PATH | `Update(dryRun=true)` called | Returns success `Result` with `Privileges=["sudo"]`, before/after set, no subprocess spawned |
| Live update with sudo | Custom tool configured with `sudo apt upgrade` and binary present on PATH | `Update(dryRun=false)` called | Executes command via shell, returns `Result` with `Privileges=["sudo"]` |
| Missing binary on check | Custom tool whose base command binary is missing from PATH (`Detect() == false`) | `Check()` called | Fails closed with structured error without invoking check subshell |
| Missing binary on update | Custom tool whose base command binary is missing from PATH (`Detect() == false`) | `Update(dryRun=false)` called | Fails closed returning `Result` with `Success=false` and structured error without invoking update subshell |
| Present binary executes | Custom tool with base binary present on PATH | `Update(dryRun=false)` called | Executes shell command bounded by timeout, returns exit status and detected privileges |
| Present binary dry-run | Custom tool with base binary present on PATH | `Update(dryRun=true)` called | Returns preview `Result` with `Success=true`, before/after commands, and detected privileges |

