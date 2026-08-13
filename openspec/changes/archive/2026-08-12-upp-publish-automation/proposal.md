# Proposal: Automate GitHub release publishing

> sdd-propose | `upp-publish-automation` | 2026-08-12

## Intent

Attaching `make release` assets to a GitHub Release is manual; without it, `upp self-update` fails closed. Automate the attach.

## Scope

### In Scope

- `make publish`: guards (clean tree, `main`, clean `vX.Y.Z`, tag absent) + `git tag -a` + push + optional `gh run watch`.
- CI `release` job: `needs: [test, lint]`, job-scoped `contents: write`, publish only on `refs/tags/v*`; idempotent create-or-upload; curated notes per repo convention (`## What's new`, `## Assets`, checksums warning).
- README Release section updated.

### Out of Scope / Non-Goals

- Changelog generation, announcements (NON-GOALS).
- Prereleases, non-`vX.Y.Z` tags, GitHub-UI or `workflow_dispatch` publishing.
- GoReleaser; `internal/selfupdate` changes.

## Locked Product Decisions

1. Only local `make publish`; CI completes release.
2. Curated notes (user summary; CI assembles).
3. `vX.Y.Z` only; dispatch publishing forbidden.
4. CI release idempotent; tag never retracted.
5. Scope v1 minimal; changelog/announcements non-goals.

## Capabilities

### New Capabilities

- `release-process`: guarded tag publishing (`make publish`) + idempotent CI release creation with checksums + curated notes.

### Modified Capabilities

- None — `self-update` contract preserved.

## Approach

Hybrid (A3, `exploration.md`): A1 local gh = dirty binaries + two paths; A2 CI-only = unguarded tags; A3 = one-command UX, CI-built clean assets, reuses `make release`.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `Makefile` | Modified | `publish`/`publish-tag` targets with guards |
| `.github/workflows/ci.yml` | Modified | Harden `release` job (needs, perms, gate, idempotency) |
| `README.md` | Modified | Document `make publish`; drop manual attach |
| `internal/selfupdate` | None | Contract preserved |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| `contents: write` escalation | Med | Job-scoped only |
| Tag before CI green | Med | `needs` gate; idempotent re-run |
| Re-run release create | Med | Create-or-upload; documented retry |
| Missed checksums asset → fail closed | Low | `make release` sole generator |

Open questions: none.

## Verification Strategy

No Go tests for Makefile/CI surface. Verify: shell-check each `make publish` guard, `make -n`, `actionlint`, one rehearsal (tag push → release → checksums match local `make release`). TDD for any Go code (none).

## Rollback Plan

Revert commit; manual `gh release create` still works. Wrong tag: delete it, push deletion, remove release.

## Dependencies

None. `gh` preinstalled on runners.

## Success Criteria

- [ ] `make publish` refuses dirty tree, non-`main`, non-`vX.Y.Z`, existing tag.
- [ ] Green CI tag push creates Release (6 assets + checksums.txt); dispatch never publishes.
- [ ] Re-run completes without touching the tag.
- [ ] `upp self-update` upgrades from a CI-published release, no manual steps; README updated.
