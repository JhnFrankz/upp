# Proposal: Tool→Manager Ownership Model + Grouping (vision point 6)

## Intent

Tools the system already updates through a package manager (docker via apt, gh via brew/winget) appear as **independent** update rows whose adapters hardcode the manager's own commands (e.g. `gh.go` runs `sudo apt update && sudo apt install -y gh`), duplicating manager logic and confusing the user. Vision point 6: such tools must **not** be duplicated — they appear **inside their manager**. This change adds a declared ownership model and manager-grouped rendering.

## Product Decisions (assumptions, user-confirmed)

1. Scope = **ownership + grouping only**. Bulk-upgrade restore is **DEFERRED** (separate change; it has a `--only` security footgun).
2. **Option A**: add `Manager`/`Kind` to `ToolInfo` — self-describing, single source of truth, extensible to custom tools.
3. Ownership is **per-platform**: gh→brew on macOS, gh→winget on Windows, gh→apt on Linux.
4. Custom tools may declare an owning manager.

## Behavioral Change Warning

`upp list` and `upp update` now **group** docker/gh/go under their owning manager (docker under apt, gh under brew/winget) instead of showing them as independent rows. Their update acts **delegate to the owning manager**; the manager row is the one the user acts on.

## Scope

**In:** `ToolInfo.Manager/Kind`; declare ownership in all 12 adapters; grouped rendering in list/update (and board/selector); delegated update path (docker/gh/go → owner); config field for custom tools to declare manager; spec deltas.
**Out:** bulk-upgrade restore (kills `--only`); any "dividio" concept (non-existent); dnf/pacman; new adapters; `go` Linux manual-binary ownership (defer).

## Capabilities

> Contract with sdd-spec. Research `openspec/specs/` first.

### New
- `tool-ownership-model`: each tool declares its owning manager per platform; a manager declares it owns N tools.

### Modified
- `tool-adapter`: `list() ToolInfo` carries manager/kind; official catalog gains owner-per-platform column; gh/docker/go rows become "delegated to owning manager".
- `ux-patterns`: List Table / Live Check Board / Interactive Update Selection gain per-manager grouping and headers.
- `platform-detection`: Tool Catalog gains owner field per platform.
- `config-system`: custom tools may declare a manager.
- `security-model`: an owned tool's risk derives from its manager; "runs `apt upgrade docker-ce`" becomes "the owning manager updates its owned tool".

## Approach

Option A (per exploration). Add to `ToolInfo` a `Manager map[platform]string` (or `Owners map[string]string`) and a `Kind` (`KindManager`/`KindTool`). Each official adapter declares its per-platform owner via `Info()`; the manager adapters (apt/brew/winget/scoop) declare `KindManager`. Rendering (list/update/board/selector) groups tools by manager then lists owned tools beneath. The update path for gh/docker/go delegates to its resolved owning manager's `update()` instead of running its own hardcoded command. Tradeoff: per-platform resolution adds a lookup; but it centralizes update logic in one place (the manager), removing duplication.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/adapters/interface.go` | Modified | `ToolInfo` gains `Manager`/`Kind` |
| `internal/adapters/official/*.go` | Modified | Declare per-platform owner; gh/docker/go delegate |
| `internal/adapters/official/registry.go` | Modified | Registry exposes owner metadata |
| `internal/platform/catalog.go` | Modified | `ToolEntry` gains owner field |
| `internal/config/config.go`, `defaults.go` | Modified | Custom tool owner field |
| `internal/cli/list.go`, `update.go` | Modified | Grouped rendering + delegation |
| `internal/output/render.go` | Modified | Group headers for list/board/selector |
| `internal/adapters/official/{info,parity,interface}_test.go` | Modified | Ownership assertions |
| `openspec/specs/*` (5) | Modified | Deltas per Capabilities |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Per-platform ownership resolution (gh→brew vs →winget) | Med | Map keyed by platform; parity tests |
| Flat-registry sync (AllAdapters ⇄ OfficialTools) | Med | Extend parity_test ownership check |
| TUI grouping complexity (board/selector) | Med | Group headers; preserve stable line order |
| Gating mismatch: Gated apt owner vs AlwaysUpdate docker | Med | Owned tool inherits owner's gating policy |
| `--only`/`--skip` interaction with grouping | Med | Grouping is display-only; filters unchanged |

## Rollback Plan

Revert `ToolInfo` Manager/Kind field + adapter owner declarations + grouped rendering/delegation. Config: if custom owner schema added, drop the key (forward-compatible); otherwise untouched. No migration needed.

## Dependencies

- None new.

## Success Criteria

- [ ] `upp list`/`update` group docker under apt, gh under brew/winget per platform
- [ ] docker/gh/go no longer independent update rows
- [ ] ownership declared in all 12 adapters
- [ ] custom tools can declare a manager
- [ ] `go test ./... -count=1` green
- [ ] 5 spec deltas merged + new `tool-ownership-model` spec
