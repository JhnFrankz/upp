# Delta for tool-ownership-model

## MODIFIED Requirements

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
