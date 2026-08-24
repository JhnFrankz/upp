# Proposal: Package Manager Self-Update (brew/apt/winget/scoop)

## Intent

Vision point 4: the package manager reports its own version and self-updates ("brew updates itself with brew") as a first-class board/list row. Today brew/apt/winget adapters conflate semantics — each updates the manager AND everything it manages; redefine to self-only.

## Product Decisions (assumptions, user-confirmed 2026-08-21)

1. SELF-ONLY for brew/apt/winget + scoop (Windows parity); bulk upgrades leave `upp update` until point 6.
2. brew dry-run: NO SIGNAL — row shows "current" in `-n`; `brew update` mutates (git fetch), never inside Check().
3. Scope: all three managers + scoop.
4. Confirmation policy unchanged — brew/winget/scoop AlwaysUpdate, apt Gated (sudo).

## Behavioral Change Warning

After this change `upp update` NO LONGER bulk-upgrades packages; it self-updates managers only (brew `brew update`; apt `sudo apt install --only-upgrade apt`; winget `winget upgrade winget`; scoop `scoop update scoop`). Point 6 restores bulk upgrades per-tool.

## Point 6 Boundary

Tool→manager ownership model NOT in scope: no Kind/manager field, no new registry/config. Point 4 repurposes existing adapters; point 6 starts clean.

## Scope

**In:** repurpose the four adapters' Check/Update to self-only (`brew upgrade brew` avoided); winget Check parses `winget upgrade` tolerating "v1.8.x"; interactive brew gap ACCEPTED — never pending in TTY (D7 preserved), self-updates via sequential/CI path; spec deltas: tool-adapter table/Gating, ux-patterns "current" note.

**Out:** bulk upgrades (point 6); ownership model (point 6); new config/adapters/IDs; dnf/pacman.

## Capabilities

- **New:** None.
- **Modified:** `tool-adapter` — manager Check/Update self-only, official table commands, Update Gating wording. `ux-patterns` — brew row renders ✓ current, never in pending selector.

## Approach

Repurpose existing manager adapters in place — already enabled tools, plain Adapter rows, `--only`/`--skip` and Policy unchanged.

## Affected Areas

Modified: `official/{brew,apt,winget,scoop}.go`; their tests + parity_test.go; `specs/tool-adapter/spec.md`; `specs/ux-patterns/spec.md`. Unchanged: registry, catalog, config, cli, security-model.

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Bulk-upgrade removal surprises users | Med | Release note; point 6 restores |
| `brew upgrade brew` footgun | Low | Adapter comment documents `brew update` |
| Winget "v1.8.x" 4-part parse | Med | Tolerant extract; parity tests |
| apt "stale" often intentional | Med | Stays Gated; documented distro semantics |

## Rollback Plan

Revert adapters to pre-change commands (`brew update && brew upgrade`, `sudo apt upgrade -y -qq`, `winget upgrade --all`). Config untouched.

## Dependencies

- winget 1.6+ for `winget upgrade winget` (older: unavailable gracefully). No new deps.

## Success Criteria

- [ ] `upp update` self-updates managers only; bulk upgrades no longer run
- [ ] brew row current (never pending) in TTY/`-n`; sequential/CI still runs `brew update`
- [ ] winget/scoop/apt report real self-update availability
- [ ] `go test ./... -count=1` green; parity tests updated
- [ ] tool-adapter + ux-patterns deltas merged