# Delta for tool-adapter

## MODIFIED Requirements

### Requirement: Adapter Interface

Every adapter MUST implement four operations:

- `detect() → bool` — is this tool installed?
- `check() → UpdateInfo` — current version + latest available + update available?
- `update() → Result` — perform the update, return success/failure + details
- `list() → ToolInfo` — return installed tool info (name, version, source, owning manager, kind)

`ToolInfo` MUST carry an owning `Manager` map keyed by platform and a `Kind` (`KindManager` for manager adapters, `KindTool` otherwise). A tool with a resolving owner on the current platform reports that manager; a tool with no owner reports no manager.

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

(Previously: `list() → ToolInfo` returned only name/version/source; `ToolInfo` had no `Manager` or `Kind` field.)

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

### Requirement: Update Gating

Every adapter MUST declare an `UpdatePolicy` (`PolicyGated` or `PolicyAlwaysUpdate`), and the system MUST gate updates on that declaration, not on a CLI-side ID list. The system MUST run `update()` for an adapter declaring `PolicyGated` (apt, npm, pnpm, nvm) only when that adapter's `check()` reported `update_available=true`. Adapters declaring `PolicyAlwaysUpdate` MUST always run their update when requested, regardless of `check()` result: official adapters without detection (brew, bun, opencode) and custom adapters report `update_available=false` by design, while winget and scoop report real self-update availability by design. An owned tool (gh, docker, go) MUST NOT be gated independently: its update delegates to its owning manager, and the manager's `UpdatePolicy` governs whether the delegated update runs; the owned tool's own `UpdatePolicy` MUST NOT apply to the delegated path. When a `PolicyGated` adapter's `check()` fails, the system MUST report the failure for that adapter as a structured error per Adapter Error Handling and MUST NOT treat the failed check as `update_available=false` nor report the adapter as current.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Owned inherits gated | docker owned by apt (Gated) on Linux, apt check reports no update | docker delegated update | Delegated `apt.update()` skipped; docker reported current |
| Owned inherits always | gh owned by brew (AlwaysUpdate) on macOS | gh delegated update | `update()` runs (delegates to brew) |
| Stub official exempt | Adapter declaring `PolicyAlwaysUpdate` without detection (brew/bun/opencode) reports `update_available=false` | Update run | `update()` still runs |
| Gated check fails | `PolicyGated` adapter `check()` fails during update run | Update run | `update()` skipped; failure reported; adapter never reported current |

(Previously: docker, gh, and go were independent `PolicyAlwaysUpdate` stubs whose `update()` always ran; ownership did not exist, so gating was per-adapter with no manager inheritance.)
