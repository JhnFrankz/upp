# Proposal: Restore Bulk-Upgrade per-Manager (vision point 6 deferred)

## Intent

Ownership (point 6) made gh/docker/go delegate to their manager's **self-only** `Update()` — so "update docker on Linux" runs `apt install --only-upgrade apt`, never upgrading docker. Buyers who opt into a manager group get the owned tool actually updated. Built safely and **opt-in** first; making it default is a later increment.

## Product Decisions (assumptions, product-owner-confirmed)

1. **Structure = safe base + opt-in**. First increment builds the per-manager package-name mapping, per-owned-tool availability check, and resolves the confirm-action security tension; exposes bulk as `--manager`/`--update-group`. A later increment makes it default.
2. **`--skip <owned-tool>` excludes** that tool from the manager group batch.
3. The hard blocker is the **package-name mapping** (ToolInfo knows the owning manager but not the package name under it).

## Behavioral Change Warning

`upp update --manager <mgr>` (and `--update-group`) triggers a manager-group bulk update that actually updates the manager's owned tools (not just display). Default `upp update` behavior is unchanged in this increment.

## Scope

**In:** per-manager package-name mapping on `ToolInfo`; per-owned-tool availability check (e.g. `apt-cache policy gh`); `--manager`/`--update-group` opt-in flag running each owned tool's package command (minus `--skip`-ed); resolve confirm tension (sudo-heavy group update prompts even for TrustOfficial owned tools); group gate inheritance; spec deltas.
**Out:** making bulk default (deferred); writing a per-manager "update manager's own self" (stays self-only); dnf/pacman; new adapters.

## Capabilities

### New
- `bulk-update`: manager-group bulk update — enumerate a manager's resolving owned tools, check availability per package, run each owned tool's package command, minus `--skip`-ed.

### Modified
- `tool-adapter`: `ToolInfo` gains per-manager package-name field; per-owned-tool availability check; group gate semantics.
- `tool-ownership-model`: per-manager package-name mapping + resolved-owner group bulk update.
- `security-model`: owned-tool group update respects real risk/privileges (resolves TrustOfficial→ConfirmAuto tension for sudo-heavy).
- `ux-patterns`: opt-in flag UX, group bulk summary rendering.
- `command-interface`: `--manager`/`--update-group` flag on `upp update`.
- `config-system`: optional bulk opt-in config key (if added).

## Approach

Add per-manager package-name mapping (e.g. `ManagerPackage map[platform]string` on `ToolInfo`) + a per-owned-tool availability check. Add `--manager`/`--update-group` to `upp update`: enumerate the manager's owned tools (minus `--skip`), check availability per package, run each owned tool's package command via the manager's privileged executor. Resolve confirm tension so sudo-heavy group updates prompt. Per exploration tradeoffs: Option A (manager group update method) + Option C (opt-in trigger).

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Package-name mapping has no source of truth (typos run sudo) | Med | Pin by parity test; table of records |
| Group batch multi-owned-tool semantics | Med | Per-owned-tool commands, `--skip` excludes |
| Confirm tension for sudo (TrustOfficial auto-proceed) | High | Group update reclassifies by real command risk |
| Gated group (apt) availability signal | Med | Group gate on group availability |
| `--skip` interaction with group | Med | Exclude owned tool from batch |

## Rollback Plan

Revert `ToolInfo` package-name field, `--manager`/`--update-group` flag + group path, group confirm resolution, config key (drop, forward-compatible). No migration.

## Dependencies

- None new (uses point-6 ownership model).

## Success Criteria

- [ ] `--manager apt` on Linux updates apt's owned tools (gh/docker) via their package commands, minus `--skip`
- [ ] Package mapping declared for gh/docker/go per platform (apt/brew/winget)
- [ ] Per-owned-tool availability check detects real package updates
- [ ] sudo-heavy group update prompts (confirm tension resolved)
- [ ] `--skip <owned-tool>` excludes from group
- [ ] `go test ./... -count=1` green
- [ ] 6 spec deltas merged + new `bulk-update` spec
