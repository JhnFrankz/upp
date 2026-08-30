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

Given an owned tool (`gh`, `docker`, `go`) and host platform, the system MUST resolve the owning manager adapter; the owned tool's `Update()` method MUST delegate execution to the resolved manager adapter's `PackageUpdater` interface via `UpdatePackage(pkg)`, supplying the platform-resolved package name mapped for that manager (e.g. `gh` on apt/brew/winget, `docker-ce-cli`/`docker`/`Docker.DockerCLI`, `go`/`golang-go`/`GoLang.Go`). A tool with no resolving owner on the host platform (such as `go` on Linux, or standalone tools like `nvm`, `pnpm`, `bun`) MUST use its own adapter's update path.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| gh delegates on Linux | Platform Linux, gh enabled, owned by apt | `gh.Update()` | Delegates to `apt.(PackageUpdater).UpdatePackage("gh")` with package name `gh` |
| docker delegates on macOS | Platform macOS, docker enabled, owned by brew | `docker.Update()` | Delegates to `brew.(PackageUpdater).UpdatePackage("docker")` with formula `docker` |
| docker delegates on Windows | Platform Windows, docker enabled, owned by winget | `docker.Update()` | Delegates to `winget.(PackageUpdater).UpdatePackage("Docker.DockerCLI")` with package ID `Docker.DockerCLI` |
| go delegates on macOS | Platform macOS, go enabled, owned by brew | `go.Update()` | Delegates to `brew.(PackageUpdater).UpdatePackage("go")` with formula `go` |
| go standalone on Linux | Platform Linux, go enabled (no owner on Linux) | `go.Update()` | Uses native Go adapter update path without manager delegation |
| PackageUpdater interface assertion | Owned tool resolved to manager adapter | `tool.Update()` | Asserts manager implements `PackageUpdater` and executes `UpdatePackage(pkg)`, returning error if assertion fails or update errors |

(Previously: the delegated `update()` ran the manager's self-only command or generic `manager.update()`; the owned tool's package under the manager was never named, so the owned tool was never actually upgraded.)

### Requirement: Resolved-Owner Group Bulk Update

Given a manager and platform, the system MUST be able to update that manager's resolving owned set as one group. The group update MUST enumerate the manager's owned tools (from owner declarations), MUST exclude any owned tool named by `--skip`, MUST check each owned tool's package availability, and MUST run each owned tool's per-manager package command. The manager's own self-only row MUST remain distinct from the group.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| brew group on macOS | Platform macOS, brew owns gh, docker, go | Group update for brew | Runs brew's package commands for gh, docker, go |
| apt group skip | Platform Linux, apt owns gh, docker | `--skip docker` group update for apt | Group updates only gh; docker excluded |
| Manager self distinct | Platform Linux, apt group update | Group update for apt | Owned tools updated via package commands; apt self handled separately |
