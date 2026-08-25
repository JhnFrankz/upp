# Design: Tool→Manager Ownership Model + Grouping

## Technical Approach

Option A: add `Manager map[string]string` (platform→manager ID) and `Kind` (`KindManager`/`KindTool`) to `ToolInfo`. Each official adapter declares its per-platform owner in `Info()`; managers (apt/brew/winget/scoop) declare `KindManager`, owned tools (gh/docker/go) declare `KindTool` with a `Manager` map, standalone tools (nvm/npm/pnpm/bun/opencode/go-on-Linux) declare `KindTool` with no `Manager` entry. All 12 adapters change only their `Info()` — no new adapter methods. A new resolution helper converts `(toolID, GOOS)` → owning manager adapter, giving the registry and CLI a single source of truth. gh/docker/go `Update()` delegates to the resolved manager's `Update()`, so the hardcoded manager commands vanish. Rendering groups manager headers before owned rows.

## Architecture Decisions

| Decision | Option | Tradeoff | Choice |
|---|---|---|---|
| ToolInfo shape | `Manager map[string]string` + `Kind` vs single `Manager string` | Map is per-platform correct but heavier; single string can't express gh→brew/winget/apt | `Manager map[string]string` keyed by platform + `Kind` enum |
| Ownership source of truth | `ToolInfo` only vs catalog only vs both | `ToolInfo` is adapter-native and extensible to custom tools; catalog duplicates it for display | `ToolInfo` is canonical; catalog carries a display copy; parity_test pins both |
| Owned-tool `Update()` | Delegate to manager adapter vs run own command | Delegation centralizes logic and kills duplication; owned tool needs a manager reference | Delegate via resolved manager's `Update()` |
| Owned-tool gating | Inherit manager's policy vs own declared policy | Owned tool must NOT gate independently (spec Update Gating); manager's policy governs | Delegated path uses manager's `UpdatePolicy`; owned tool's own `UpdatePolicy` ignored on the delegated path |
| Grouping | Display-only grouping vs reordering underlying list | Reordering would break `--only`/`--skip` semantics | Group at render/selector layer only; filter iteration stays per-tool-ID |

## Data Model

```go
// Kind distinguishes manager adapters from owned/standalone tool adapters.
type Kind int
const (
    KindTool Kind = iota  // zero value: a tool with no manager
    KindManager           // apt, brew, winget, scoop
)

type ToolInfo struct {
    ID           string
    Name         string
    Platforms    []string
    Trust        TrustLevel
    UpdatePolicy UpdatePolicy
    Kind         Kind
    Manager      map[string]string // platform -> owning manager ID (nil for standalone)
    Command      string
    Privileges   []string
}
```

Catalog `ToolEntry` gains `Kind` + `Manager`:

```go
type ToolEntry struct {
    ID        string
    Name      string
    Platforms []string
    Kind      adapters.Kind
    Manager   map[string]string // display copy of adapter declaration
}
```

Config `CustomTool` gains an optional `Manager`:

```go
type CustomTool struct {
    Command  string `toml:"command"`
    CheckCmd string `toml:"check_cmd,omitempty"`
    Trusted  bool   `toml:"trusted"`
    Manager  string `toml:"manager,omitempty"` // optional owning manager ID
}
```

Owner declarations per `Info()`: gh = `{linux:apt, macos:brew, windows:winget}`; docker identical; go = `{macos:brew, windows:winget}` (no linux → standalone there); apt/brew/winget/scoop = `KindManager`; nvm/npm/pnpm/bun/opencode = `KindTool`, no `Manager`.

## Ownership Resolution

New helper in `internal/adapters/official`:

```go
// ResolveOwner returns the manager adapter owning tool on OS, or nil.
func ResolveOwner(tool, os string) adapters.Adapter {
    a := AdapterByName(tool)
    if a == nil { return nil }
    ownerID := a.Info().Manager[os]   // nil map → ""
    if ownerID == "" { return nil }   // standalone on this platform
    return AdapterByName(ownerID)
}
```

It reads the adapter's own `Manager` map (canonical). Registry exposes a convenience `OwnerMetadata()` returning per-builder-kind counts; CLI uses `ResolveOwner`. A tool with no `Manager[os]` entry returns nil and stays standalone (its own `Update()` runs).

## Delegated Update Path

gh/docker/go `Update(dryRun)` becomes: resolve owner via `ResolveOwner(id, runtime.GOOS)`; if nil, run the existing standalone command (only `go` on Linux keeps one). Otherwise return `owner.Update(dryRun)` — the manager's `Update()` runs its self-only command (apt self-only, brew update, winget self-only) — never an `apt upgrade docker-ce`/`brew upgrade gh` invocation. `ToolInfo.Manager` no longer carries `Privileges`; the delegated result carries the manager's `Privileges` (e.g. apt → `["sudo"]`).

Gating: the CLI gate (`runUpdateSequential`/`processSelectedOutcome`) reads `info.UpdatePolicy`. For an owned tool that policy is now deliberately unused on the delegated path — the manager adapter's `UpdatePolicy` governs. Concretely: docker (always) owned by apt (Gated) on Linux — when apt's `check()` reports no update, the delegated `apt.Update()` result is reported as *current* via the manager's Gated gate, never force-run. gh owned by brew (AlwaysUpdate) on macOS — the delegated `brew.Update()` always runs when requested. The owned tool is therefore never a top-level gate; its row inherits the manager's Gated/Always decision.

## Rendering / Grouping

Grouping algorithm (canonical discovery order): (1) manager headers first (KindManager, in `AllAdapters` order: apt, brew, winget, scoop), (2) their owned tools (KindTool with a current-platform `Manager[os]` resolving to that manager), (3) standalone tools (KindTool, no owner on this platform). Implemented as `GroupByOwner(tools []adapters.Adapter, os string) []Group` in the `output` package.

Affected code paths:
- `output/render.go` `ListTools`: accepts `[]Group`, prints a manager header line then indented child rows. `ListEntry.ID` stays the per-tool filter ID.
- `output/checkboard.go` `NewCheckBoard`: takes a flat `[]string` of stable lines in group order (headers emitted as their own board lines); existing index-slot completion logic unchanged — grouping only changes line order/count, never per-tool completion.
- `output/selector.go` `CheckboxSelector`: `SelectOption` gains an optional `Group` field; render writes a header line before its group's options. Selection/collect logic unchanged.
- `internal/cli/list.go` `runList`: build `[]Group` from `buildAdapterList` filtered by OS, then pass to `ListTools`.
- `internal/cli/update.go` `runUpdateInteractive`: names fed to the board and `pending` to the selector are built from the group order.

Display-only: `--only`/`--skip` filter over `adapterIDs` before any grouping, exactly as today — filter semantics and round-trip IDs unchanged.

## Config

`[custom.<id>] manager = "brew"` maps to `CustomTool.Manager`. `buildAdapterList` passes manager to `NewCustomAdapter`; a custom tool with a resolvable owner that is a known official manager (`KindManager`) groups/updates under it. Validation: an unknown or non-manager value is ignored (tool proceeds standalone) and a warning is emitted to stderr — forward-compatible. `upp init` never writes `manager` (optional user-declared field), and rollback simply drops the key from existing configs with no migration.

## Security-Model Integration

An owned tool's risk derives from its manager. On the delegated path the executed command and privileges are the manager's (`ConfirmConfig` built from the manager's `Info().Command`/`Privileges`), so `ClassifyCommand`/`ConfirmAction` rate the manager's operation, not an `apt upgrade docker-ce` string. Owned tools carry `TrustOfficial`; the manager's risk tier decides prompting. No owned tool invokes a manager command itself (spec Official Tool Integrity).

## File Changes

| File | Action | Description |
|---|---|---|
| `internal/adapters/interface.go` | Modify | Add `Kind` enum to `ToolInfo`; add `Manager` field |
| `internal/adapters/official/*.go` (12) | Modify | Set `Kind`; gh/docker/go add `Manager` map; gh/docker/go `Update()` delegates; remove hardcoded manager commands |
| `internal/adapters/official/registry.go` | Modify | Add `ResolveOwner`, `OwnerMetadata` |
| `internal/adapters/official/ownership.go` | Create | Per-platform resolution + owner helper |
| `internal/platform/catalog.go` | Modify | `ToolEntry` gains `Kind`/`Manager`; catalog entries populated |
| `internal/adapters/custom.go` | Modify | `CustomAdapter` carries `manager`; `Info()` sets it; delegates when owner resolves |
| `internal/config/config.go` | Modify | `CustomTool.Manager` field; validate known manager in `Validate` |
| `internal/output/render.go` | Modify | `ListTools` grouped; `Group` type |
| `internal/output/checkboard.go` | Modify | Accept grouped line order |
| `internal/output/selector.go` | Modify | `SelectOption.Group`; group header render |
| `internal/cli/list.go` | Modify | Build groups, pass to `ListTools` |
| `internal/cli/update.go` | Modify | Grouped names/pending; delegation gate uses manager policy |
| `internal/cli/checkrun.go` | Modify | `buildAdapterList` threads custom manager |

## Testing Strategy

| Layer | What to Test | Approach |
|---|---|---|
| Unit | `Info()` ownership per adapter | Extend `info_test.go` golden with `Kind`/`Manager`; `UseCase` for gh/docker/go/standalone |
| Unit | Owned-tool full-interface | `registry_test` asserts Kind/Manager consistency, `ResolveOwner` per platform |
| Unit | Per-platform resolution | Table: gh/docker/go → apt/brew/winget per GOOS; go-on-Linux → nil |
| Unit | Delegated update | gh/docker go delegate to manager's `Update()`; go-on-Linux uses own path |
| Unit | Gating inheritance | Managed owned tool respects manager's `PolicyGated`/`PolicyAlwaysUpdate` |
| Unit | Grouping render | `render_test`: manager header then child rows; standalone after |
| Unit | Selector/board grouping | `selector_test`/`checkboard_test`: group headers present, stable line order |
| Integration | CLI list/update grouping | `cli` tests: `--only/--skip` round-trip untouched by grouping |
| Parity | Flat-registry sync | Extend `parity_test`: owner per platform matches adapter `Manager` |

## Threat Matrix

N/A — no routing, shell, subprocess, VCS/PR automation, executable-file classification, or new process-integration boundary beyond existing adapter exec seams (unchanged). Delegated `Update()` reuses the same `runCmd` seam; security-model delegations are covered by the existing adapter error/timeout tests.

## Migration / Rollout

No migration required. Rollback: revert `ToolInfo` `Kind`/`Manager`, the 12 `Info()` declarations, gh/docker/go delegation, grouping, and `CustomTool.Manager` (drop the key — forward-compatible). `go test ./... -count=1` is the gate.

## Open Questions

- [ ] Does an owned tool's `Check()` still report its own version for the board/selector pending row, or should it report the manager's availability? Spec shows owned tools within a manager group with per-tool pending status; design assumes owned `Check()` stays independent (version reporting) while only `Update()` delegates.

## Work Unit Split

| WU | Scope | ~Lines |
|---|---|---|
| WU1 | `ToolInfo.Kind/Manager` + 12 `Info()` declarations + `ownership.go`/`ResolveOwner` + catalog `ToolEntry` + parity/info/registry tests | ~120 |
| WU2 | gh/docker/go delegated `Update()` + gating inheritance + custom-adapter delegation + official update tests | ~110 |
| WU3 | Grouping rendering: render.go/checkboard.go/selector.go + list.go/update.go wiring + grouped render/selector/board/cli tests | ~130 |
| WU4 | Config `CustomTool.Manager` + validation/init hygiene + spec sync + `go test` green | ~60 |
