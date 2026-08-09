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
