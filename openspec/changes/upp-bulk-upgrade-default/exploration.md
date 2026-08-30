# Exploration: Default Manager-Group Bulk Updates in 'upp update'

## 1. Executive Summary

This exploration investigates making **manager-group bulk package updates the default behavior in `upp update`** (completing the deferred scope from Point 6 of the product vision).

- **Current State**:
  - In **v0.4.0**, tool-to-manager ownership was established (`ToolInfo`, `ResolveOwner`), but owned tools (`gh`, `docker`, `go`) delegated `Update()` to `owner.Update()`, running manager self-only updates (e.g. `sudo apt install --only-upgrade apt`) without updating the owned tool packages.
  - In **v0.5.0**, manager package availability queries (`PackageChecker`) and per-package update executors (`PackageUpdater`) were implemented on manager adapters (`apt`, `brew`, `winget`), but group bulk update was kept opt-in via `--manager <mgr>` and `--update-group <mgr>`. Bare `upp update` continues to execute individual tool updates sequentially.
- **Target State (`upp-bulk-upgrade-default`)**:
  - Make `upp update` execute manager-group bulk updates by default for owned tools across both non-interactive (`runUpdateSequential`) and interactive TTY (`runUpdateInteractive`) modes.
  - Wire official owned adapters (`gh.go`, `docker.go`, `go.go`) so their `Update()` delegates to `owner.(PackageUpdater).UpdatePackage(pkg)`.
  - Enforce real command risk classification with `EnforceRisk: true` for owned packages (prompting in TTY and failing closed in `--ci`).
  - Maintain standalone/unowned tools (`npm`, `nvm`, `pnpm`, `bun`, `opencode`, `go` on Linux) in their individual execution paths with per-tool error isolation.

---

## 2. Current State Analysis

### 2.1 Adapter Delegation Gaps
In `internal/adapters/official/{gh,docker,go}.go`:
- `Check()` delegates to `owner.(adapters.PackageChecker).CheckPackage(pkg)` using platform-resolved package names (`ManagerPackage[platform]`).
- `Update(dryRun)` currently calls `owner.Update(dryRun)`, which runs manager self-only updates rather than upgrading the tool package itself.

### 2.2 CLI Execution Engine in `internal/cli/update.go`
- `runUpdate`: When `uf.Manager == ""` (default), execution falls through to `runUpdateInteractive` (TTY) or `runUpdateSequential` (non-TTY, CI, dry-run).
- `runUpdateSequential`: Iterates over tools flatly without evaluating underlying package sudo commands.
- `runUpdateInteractive`: Pre-checks all tools with `CheckBoard` and passes pending tools to `CheckboxSelector`. Selected tools run via `processSelectedOutcome`, calling `a.Update(false)` directly without manager batching or sudo risk escalation.
- `EnforceRisk`: In `runUpdateGroup`, `EnforceRisk: true` evaluates real package commands (e.g. `sudo apt install --only-upgrade gh`). In default `runUpdate`, `EnforceRisk` is not yet applied to owned tools.

---

## 3. Proposed Architecture & Seams

```
                                      ┌──────────────────────────────────────────────┐
                                      │                  upp update                  │
                                      └──────────────────────┬───────────────────────┘
                                                             │
                                   ┌─────────────────────────┴─────────────────────────┐
                                   │ Partición: Gestores vs Herramientas Standalone    │
                                   └─────────────────────────┬─────────────────────────┘
                                                             │
                         ┌───────────────────────────────────┴───────────────────────────────────┐
                         ▼                                                                       ▼
      ┌─────────────────────────────────────┐                                 ┌─────────────────────────────────────┐
      │     Grupos de Gestores (apt, brew)   │                                 │   Herramientas Standalone (npm,...) │
      ├─────────────────────────────────────┤                                 ├─────────────────────────────────────┤
      │ 1. Fila de auto-actualización gestor│                                 │ 1. Check()                          │
      │ 2. Lote de herramientas gestionadas:│                                 │ 2. ConfirmAction(Trust, Risk)       │
      │    - CheckPackage(pkg)              │                                 │ 3. Update()                         │
      │    - ConfirmAction(EnforceRisk:true)│                                 │ 4. Aislamiento de errores           │
      │    - UpdatePackage(pkg)             │                                 └──────────────────┬──────────────────┘
      │    - Aislamiento de errores         │                                                    │
      └──────────────────┬──────────────────┘                                                    │
                         │                                                                       │
                         └───────────────────────────────────┬───────────────────────────────────┘
                                                             │
                                   ┌─────────────────────────┴─────────────────────────┐
                                   │ Resumen Final Determinista (Orden Canónico)       │
                                   └───────────────────────────────────────────────────┘
```

1. **Adapter Seam**: Update `gh`, `docker`, and `go` `Update()` methods to invoke `owner.(PackageUpdater).UpdatePackage(pkg)`.
2. **CLI Runner Seam**: Refactor `runUpdate` in `internal/cli/update.go` to group owned tools by manager by default, checking package availability with `PackageChecker` and executing via `PackageUpdater`.
3. **Security Model**: Pass `EnforceRisk: true` on elevated commands, prompting in interactive TTY and aborting safely in `--ci`.
4. **Output Rendering**: Preserve deterministic summary rendering matching canonical tool discovery order.

---

## 4. Blast Radius & Affected Files

- `internal/adapters/official/{gh,docker,go}.go`: Delegate `Update` to `UpdatePackage(pkg)`.
- `internal/cli/update.go`: Default manager-group execution in `runUpdate` and `runUpdateInteractive`.
- `internal/output/render.go` & `group.go`: Batch preview and group bulk summary integration.
- `internal/adapters/official/update_test.go` & `internal/cli/update_test.go`: Hermetic unit and CLI tests.
- `openspec/specs/{bulk-update,tool-ownership-model,command-interface,ux-patterns}/spec.md`: Delta specifications.

---

## 5. Risks & Mitigations

- **Sudo elevation in CI**: Fail-closed with non-zero exit in `--ci` mode when unconfirmed elevated commands are detected (`EnforceRisk: true`).
- **Partial failures**: Isolate errors per package so failing tools do not abort sibling tools or standalone tools.
- **Standalone tools**: Preserved native execution without manager delegation when unowned.
