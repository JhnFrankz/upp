# Design: Automate GitHub Release Publishing

## Technical Approach

Hybrid (exploration A3): `make publish` runs local guards, creates the annotated tag, pushes it; CI's hardened `release` job builds assets with the existing `make release` (unchanged) and publishes the GitHub Release idempotently on `refs/tags/v*`. `internal/selfupdate` and the asset/checksums contract are untouched. Implements all eight requirements of spec `release-process`.

## Architecture Decisions

### D1 — Makefile: single guarded `publish` target

Choice: one target, guards as POSIX-sh recipe lines, matching the Makefile's existing style. Rejected: local `gh release create` (dirty binaries, two publish paths); a separate `publish-tag` (spec only requires `make publish`).

Guards (order fixed; failure prints error, exits non-zero, creates nothing):
1. `git status --porcelain` empty → else "ERROR: working tree not clean"
2. `git branch --show-current` = `main` → else "ERROR: must run on main"
3. `echo "$(VERSION)" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+$'` → else "ERROR: version must match vX.Y.Z"
4. `git rev-parse --verify "refs/tags/$(VERSION)"` absent → else "ERROR: tag already exists"
5. `git ls-remote --exit-code origin "refs/tags/$(VERSION)"` absent → else "ERROR: tag already exists on origin"

Then `git tag -a "$(VERSION)"` (editor; first line = summary) and `git push origin "refs/tags/$(VERSION)"` (explicit refspec — never `--tags`). Optional trailing `gh run watch` when `gh` is available. Positive checks end in `|| true` (make runs each line in its own shell).

### D2 — CI release job

Choice: keep one `release` job; gate the publish step, not the job. Rejected: separate assets+release jobs (duplicate build or artifact download; more surface).

```yaml
release:
  needs: [test, lint]
  if: startsWith(github.ref, 'refs/tags/v') || github.event_name == 'workflow_dispatch'
  runs-on: ubuntu-latest
  permissions: { contents: write }
  steps:
    - uses: actions/checkout@v4
      with: { fetch-depth: 0 }            # annotated tag object for notes
    - uses: actions/setup-go@v5
      with: { go-version: '1.22.x', cache: true }
    - run: make release
    - uses: actions/upload-artifact@v4
      with: { name: upp-dist, path: dist/**, if-no-files-found: error }
    - if: startsWith(github.ref, 'refs/tags/v')
      env: { GH_TOKEN: '${{ github.token }}' }
      run: scripts/publish-release.sh publish
```

`needs` satisfies the CI Release Gate (tag push + CI red → skipped); `if` keeps dispatch build-only; PR pushes never reach the job.

### D3 — Idempotency (`scripts/publish-release.sh publish`)

Create-or-upload; tag never touched in CI (no `git tag`, `git push`, `gh release delete`):

```sh
TAG="$GITHUB_REF_NAME"
if gh release view "$TAG" >/dev/null 2>&1; then
  existing=$(gh release view "$TAG" --json assets --jq '.assets[].name')
  for f in dist/upp-*.tar.gz dist/upp-*.zip dist/checksums.txt; do
    echo "$existing" | grep -qxF "$(basename "$f")" || gh release upload "$TAG" "$f"
  done
else
  scripts/publish-release.sh notes > notes.md
  gh release create "$TAG" dist/upp-*.tar.gz dist/upp-*.zip dist/checksums.txt \
    --title "upp $TAG — $SUMMARY" --notes-file notes.md
fi
```

Re-run (UI or `gh run rerun`) uploads only missing assets; existing ones untouched.

### D4 — Curated notes (`scripts/publish-release.sh notes`)

Summary source: the annotated tag message written during `make publish` — travels with the tag, no repo files, no prompt plumbing. Rejected: NOTE file (drift), `make publish` prompt (interactive-only), auto-generated notes (spec forbids). `git tag -l --format='%(contents)' "$TAG"` → first line = summary, remaining non-empty lines = `## What's new` bullets; script appends `## Assets` (dist file names) and the warning: "checksums.txt must keep shipping with every release — self-update fails closed without it."

### D5 — Permissions

Job-scoped `permissions: contents: write` on `release` only; top-level and all other jobs stay `contents: read`. Auth is the automatic `github.token` via `GH_TOKEN` — no PATs, no secrets in the repo or Makefile. On dispatch runs the write permission is inert (no write call executes) — accepted over two-job complexity.

### D6 — Verification (shell-only; strict TDD covers Go, none here)

1. Guard matrix: dirty tree (untracked and staged variants), wrong branch, `VERSION=1.2.0`, `VERSION=v0.2.0-rc1`, existing local tag, existing remote tag (`VERSION=v0.1.1`) → assert exact message, non-zero exit, no tag created.
2. `make -n publish VERSION=v9.9.9` → assert exact sequence (tag -a → explicit-refspec push), zero side effects.
3. `actionlint ci.yml`; `shellcheck` + `bash -n` on scripts.
4. `scripts/publish-release.sh notes` against a scratch tag in a scratch repo → assert title and all sections.
5. Rehearsal on a fork: `make publish VERSION=v0.0.99` → CI green → release with 6 assets; `sha256sum -c` local `make release` checksums vs downloaded `checksums.txt`; job re-run → no changes; dispatch → artifacts only.

### D7 — README

Replace the manual-attach instructions in the Release section: `make publish VERSION=vX.Y.Z` (guards listed; tag message becomes release notes; CI completes the release). Keep `make release` documentation. No install.sh changes.

## Data Flow

```
maintainer ─ make publish ─ guards ─ git tag -a ─ push refs/tags/vX.Y.Z
                                                │
   push(tags v*) ─ CI: test ─┐                 ▼
                  lint ──────┴─ release job ─ make release ─ dist/ (5 archives + checksums.txt)
                                                        │
                             publish-release.sh: notes (tag msg) + create-or-upload (gh)
                                                        │
                                         GitHub Release ── upp self-update (unchanged)
```

## File Changes

| File | Action | Before → After |
|---|---|---|
| `Makefile` | Modify | no publish target → guarded `publish` |
| `.github/workflows/ci.yml` | Modify | release: no needs, no perms, build+upload only → needs gate, job-scoped write, step-gated publish |
| `scripts/publish-release.sh` | Create | — → notes assembly + idempotent create-or-upload |
| `README.md` | Modify | manual attach steps → `make publish` flow |
| `internal/selfupdate/*`, `scripts/install.sh` | None | contract preserved |

## Threat Matrix

| Boundary | Applicability | Design response | RED test |
|---|---|---|---|
| Documentation-like paths | N/A — no doc file is executed | — | — |
| Git repository selection | Applicable | plain `git`, cwd authority (repo root, as `make release` already assumes); no `-C` | `make publish` outside a repo → non-zero |
| Commit state | Applicable | `git status --porcelain` empty covers staged + unstaged | dirty and staged variants abort |
| Push state | Applicable | `git ls-remote --exit-code` pre-check; push explicit `refs/tags/$(VERSION)`, never `--tags` | remote-exists aborts; `make -n` shows explicit refspec |
| PR commands | N/A — no PR automation | — | — |

## Failure Modes (→ spec scenarios)

- Guard failure (dirty/branch/version/tag) → abort, no tag created; fix and re-run.
- CI red on tag push → `needs` skips release; tag is unpublished → retract it (`git tag -d` + `git push origin :refs/tags/vX.Y.Z`), fix, re-publish.
- Push fails after `git tag -a` → local tag only; finish with `git push origin refs/tags/vX.Y.Z` or `git tag -d` and retry.
- Partial upload → re-run uploads only missing assets; tag and existing assets untouched.
- Dispatch → builds and uploads artifacts, never creates a release.

## Migration / Rollout

No migration. Rollback: revert commit; manual `gh release create` path still works. Wrong tag: delete tag + remote ref + release (only if unpublished — published tags are never retracted).

## Open Questions

None.
