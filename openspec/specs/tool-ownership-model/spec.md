# Tool Ownership Model Specification

## Purpose

Define how each tool adapter declares its owning manager per platform, and how a manager adapter reports the set and count of tools it owns. This lets owned tools (gh, docker, go) group under their owning manager instead of appearing as independent update rows, and delegate their update to the owning manager.

## Requirements



### Requirement: Tool Ownership Declaration

Every tool adapter MUST declare its owning manager per platform through `ToolInfo`. A tool's `ToolInfo` MUST carry a `Manager` map keyed by platform and a `Kind` (`KindManager` for manager adapters, `KindTool` for owned tools). Ownership is per-platform: the same tool MAY be owned by different managers on different platforms.

| Tool | Linux | macOS | Windows |
|------|-------|-------|---------|
| gh | apt | brew | winget |
| docker | apt | brew | winget |
| go | (none) | brew | winget |

Manager adapters (apt, brew, winget, scoop) MUST declare `KindManager`. Tools with no resolving owner on a platform (nvm, npm, pnpm, bun, opencode, go-on-Linux) MUST remain standalone (`KindTool`, no `Manager` entry).

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| gh owner per platform | Platform is macOS | `gh.list()` | `Manager["macos"]="brew"` |
| docker owner per platform | Platform is Windows | `docker.list()` | `Manager["windows"]="winget"` |
| go Linux standalone | Platform is Linux | `go.list()` | `Kind=KindTool`; no `Manager["linux"]` |
| apt declares manager | apt adapter queried | `apt.list()` | `Kind=KindManager` |

### Requirement: Manager Owned-Tool Cardinality

A manager adapter MUST report the set and count of tools it owns on the current platform from per-platform owner declarations. The owned set MUST be derived from owner declarations, not hardcoded per platform.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| brew owns three on macOS | Platform macOS | brew ownership resolved | Owns gh, docker, go |
| apt owns two on Linux | Platform Linux | apt ownership resolved | Owns gh, docker |
| winget owns three on Windows | Platform Windows | winget ownership resolved | Owns gh, docker, go |

### Requirement: Resolved Owner Update Delegation

Given a tool and platform, the system MUST resolve the owning manager; the owned tool's update MUST delegate to the resolved manager's `update()` for the owned tool's package under that manager (per the per-manager package-name mapping). A tool with no resolving owner MUST use its own adapter's update path.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| gh delegates on Linux | Platform Linux, gh enabled, owned by apt | `gh.update()` | Delegates to `apt.update()` for package `gh` |
| go standalone on Linux | Platform Linux, go enabled | `go.update()` | Uses go adapter (no owner on Linux) |
| docker delegates on macOS | Platform macOS, docker enabled, owned by brew | `docker.update()` | Delegates to `brew.update()` for package `docker` |

(Previously: the delegated `update()` ran the manager's self-only command; the owned tool's package under the manager was never named, so the owned tool was never actually upgraded.)
