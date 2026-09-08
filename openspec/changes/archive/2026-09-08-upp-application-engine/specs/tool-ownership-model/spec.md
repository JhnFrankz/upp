# Delta for tool-ownership-model

## MODIFIED Requirements

### Requirement: Resolved Owner Update Delegation

Given an owned tool (`gh`, `docker`, `go`, or custom tool declaring `manager`) and host platform, the application engine MUST resolve the owning manager adapter during adapter discovery (`Resolve`) and update planning (`Plan`); the owned tool's `Update()` method MUST delegate execution to the resolved manager adapter's `PackageUpdater` interface via `UpdatePackage(pkg)`, supplying the platform-resolved package name mapped for that manager (e.g. `gh` on apt/brew/winget, `docker-ce-cli`/`docker`/`Docker.DockerCLI`, `go`/`golang-go`/`GoLang.Go`, or configured custom package name). Package manager adapters declaring `KindManager` (apt, brew, pacman, winget, scoop) MUST implement `PackageChecker` (`CheckPackage(pkg)`) and `PackageUpdater` (`UpdatePackage(pkg)`). A tool with no resolving owner on the host platform (such as `go` on Linux, or standalone tools like `nvm`, `pnpm`, `bun`) MUST use its own adapter's update path.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| gh delegates on Linux | Platform Linux, gh enabled, owned by apt | `gh.Update()` | Delegates to `apt.(PackageUpdater).UpdatePackage("gh")` with package name `gh` |
| docker delegates on macOS | Platform macOS, docker enabled, owned by brew | `docker.Update()` | Delegates to `brew.(PackageUpdater).UpdatePackage("docker")` with formula `docker` |
| docker delegates on Windows | Platform Windows, docker enabled, owned by winget | `docker.Update()` | Delegates to `winget.(PackageUpdater).UpdatePackage("Docker.DockerCLI")` with package ID `Docker.DockerCLI` |
| go delegates on macOS | Platform macOS, go enabled, owned by brew | `go.Update()` | Delegates to `brew.(PackageUpdater).UpdatePackage("go")` with formula `go` |
| go standalone on Linux | Platform Linux, go enabled (no owner on Linux) | `go.Update()` | Uses native Go adapter update path without manager delegation |
| PackageUpdater interface assertion | Owned tool resolved to manager adapter | `tool.Update()` | Asserts manager implements `PackageUpdater` and executes `UpdatePackage(pkg)`, returning error if assertion fails or update errors |
| pacman implements PackageUpdater | Custom tool configured with `manager = "pacman"` and package `ripgrep` | `tool.Update()` | Asserts pacman implements `PackageUpdater` and delegates to `pacman.UpdatePackage("ripgrep")` |
| pacman implements PackageChecker | Custom tool configured with `manager = "pacman"` and package `ripgrep` | `tool.Check()` | Asserts pacman implements `PackageChecker` and delegates to `pacman.CheckPackage("ripgrep")` |
| Engine centralized resolution | Owned tool evaluated during `Resolve` and `Plan` | `engine.Resolve()` / `engine.Plan()` | Owning manager and effective policies resolved centrally by the application engine |

(Previously: manager resolution and effective update policy derivation were performed ad-hoc across `internal/cli` functions `buildAdapterList`, `resolvingOwner`, and `resolveEffectiveUpdatePolicy` during CLI update execution loops; resolution is now centralized within `internal/engine`.)

## NEW Requirements

### Requirement: Centralized Ownership Resolution

The application engine MUST centralize custom tool manager binding, owner resolution, and effective update policy inheritance within `internal/engine`.

1. **Custom Tool Manager Binding**: During `engine.Resolve`, when a configured custom tool declares an owning manager (`custom.Manager != ""`), the engine MUST look up the declared manager name among platform official adapters. If a matching official adapter exists and declares `KindManager`, the engine MUST bind the manager adapter as the custom tool's owner (`managerArgs`). If no match exists or the adapter does not declare `KindManager`, the custom tool MUST resolve as standalone.
2. **Effective Policy Inheritance**: During `engine.Plan`, for every owned tool (official or custom) that resolves to an owning manager on the host platform, the engine MUST apply the owning manager's `UpdatePolicy` to govern update planning:
   - If the owning manager declares `PolicyGated`, the owned tool MUST be planned for update (`Updates`) only when its check reported `UpdateAvailable == true`; otherwise it MUST be categorized as `Current`.
   - If the owning manager declares `PolicyAlwaysUpdate`, the owned tool MUST be planned for update (`Updates`) unconditionally.
   - The owned tool's own declared `UpdatePolicy` MUST be inert.
3. **Decoupled Presentation**: The CLI presentation layer MUST NOT implement independent manager ownership resolution or policy inheritance logic, delegating resolution and planning entirely to `engine.Engine`.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Custom tool manager bound in Resolve | Custom tool with `manager = "brew"` on macOS | `engine.Resolve(Filter{})` | `brew` manager adapter bound to custom tool as owner |
| Custom tool unknown manager in Resolve | Custom tool with `manager = "unknown"` | `engine.Resolve(Filter{})` | Custom tool resolves as standalone |
| Owned tool inherits Gated policy in Plan | `gh` owned by `apt` (`PolicyGated`) on Linux with candidate available | `engine.Plan(outcomes, Filter{})` | `gh` planned for update in `UpdatePlan.Updates` |
| Owned tool inherits Gated policy current | `gh` owned by `apt` (`PolicyGated`) on Linux with candidate current | `engine.Plan(outcomes, Filter{})` | `gh` categorized into `UpdatePlan.Current` |
| Owned tool inherits AlwaysUpdate in Plan | `gh` owned by `brew` (`PolicyAlwaysUpdate`) on macOS | `engine.Plan(outcomes, Filter{})` | `gh` planned for update unconditionally in `UpdatePlan.Updates` |
| Owned declared policy inert | Owned tool declares `PolicyAlwaysUpdate` but manager is `PolicyGated` without updates | `engine.Plan(outcomes, Filter{})` | Manager's `PolicyGated` takes precedence; tool categorized as `Current` |
