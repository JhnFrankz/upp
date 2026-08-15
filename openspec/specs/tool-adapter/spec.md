# Tool Adapter Interface Specification

## Purpose

Define the contract every tool adapter (official or custom) must implement. Each adapter handles one tool on one platform: detect, check, update, list.

## Requirements

### Requirement: Adapter Interface

Every adapter MUST implement four operations:

- `detect() → bool` — is this tool installed?
- `check() → UpdateInfo` — current version + latest available + update available?
- `update() → Result` — perform the update, return success/failure + details
- `list() → ToolInfo` — return installed tool info (name, version, source)

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Tool installed | `nvm` is on PATH | `detect()` called | Returns `true` |
| Tool missing | `nvm` not on PATH | `detect()` called | Returns `false` |
| Update available | Node.js v18 installed, v20 latest | `check()` called | `update_available=true`, versions returned |
| No update | Node.js v20 installed, v20 latest | `check()` called | `update_available=false` |
| Update succeeds | Update command exits 0 | `update()` called | `success=true`, before/after versions returned |
| Update fails | Update command exits non-zero | `update()` called | `success=false`, error message returned |

### Requirement: Official Adapter Catalog

The system MUST ship built-in adapters for all official tools per platform (see platform-detection catalog).

Each official adapter MUST use the platform-native update mechanism:

| Tool | Linux | macOS | Windows |
|------|-------|-------|---------|
| apt | `apt upgrade` | N/A | N/A |
| brew | `brew upgrade` | `brew upgrade` | N/A |
| winget | N/A | N/A | `winget upgrade` |
| scoop | N/A | N/A | `scoop update` |
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
| Linux brew adapter | Platform Linux, brew installed | `brew.update()` | Runs `brew update && brew upgrade` |
| macOS docker adapter | Platform macOS, docker installed | `docker.update()` | Runs `brew upgrade docker` |

### Requirement: Adapter Error Handling

Adapters MUST return structured errors on failure. Errors MUST include: tool name, operation attempted, exit code (if applicable), and stderr excerpt.

Adapters MUST NOT abort the entire update run on individual tool failure. The orchestrator collects results and reports in summary.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Network failure | `brew update` fails (network) | `brew.update()` | Returns error with exit code and stderr |
| Partial success | 3/5 tools updated | Full update run | Summary shows 3 updated, 2 failed |

### Requirement: Version Comparison

Adapters MUST return semver-compatible version strings when available. Adapters SHOULD normalize version formats across platforms.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Semver version | Node v20.11.0 | `check()` | Returns `current="20.11.0"` |
| Non-semver | Docker "24.0.7" | `check()` | Returns raw version string |

### Requirement: Update Gating

Every adapter MUST declare an `UpdatePolicy` (`PolicyGated` or `PolicyAlwaysUpdate`), and the system MUST gate updates on that declaration, not on a CLI-side ID list. The system MUST run `update()` for an adapter declaring `PolicyGated` (apt, npm, pnpm, nvm) only when that adapter's `check()` reported `update_available=true`. Adapters declaring `PolicyAlwaysUpdate` MUST always run their update when requested, regardless of `check()` result: official adapters without detection (brew, bun, docker, gh, go, opencode) and custom adapters report `update_available=false` by design, while winget and scoop report `update_available=true` by design. When a `PolicyGated` adapter's `check()` fails, the system MUST report the failure for that adapter as a structured error per Adapter Error Handling and MUST NOT treat the failed check as `update_available=false` nor report the adapter as current.
(Previously: gating was decided by a CLI-side list of official adapter IDs with real update detection (apt, npm, pnpm, nvm); behavior of a failed check was undefined.)

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Official update available | Adapter declaring `PolicyGated` `check()` reports `update_available=true` | Update run | `update()` runs for that adapter |
| Official no update | Adapter declaring `PolicyGated` (apt/npm/pnpm/nvm) `check()` reports `update_available=false` | Update run | `update()` skipped; adapter reported current |
| Stub official exempt | Adapter declaring `PolicyAlwaysUpdate` without detection (brew/bun/docker/gh/go/opencode) reports `update_available=false` | Update run | `update()` still runs |
| Custom exempt | Custom adapter declaring `PolicyAlwaysUpdate` reports `update_available=false` | Update run | `update()` still runs |
| winget/scoop exempt | winget or scoop adapter declaring `PolicyAlwaysUpdate` | Update run | `update()` always runs |
| Dynamic detection | apt reports `update_available=false` | Update run | `update()` skipped |
| Gated check fails | `PolicyGated` adapter `check()` fails during update run | Update run | `update()` skipped; failure reported; adapter never reported current |

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
