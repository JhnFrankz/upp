# Tasks: Automate GitHub Release Publishing

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~120–150 (Makefile ~35, publish-release.sh ~50, ci.yml ~15, README ~15) |
| 400-line budget risk | Low |
| Chained PRs recommended | No |
| Suggested split | Single PR (U1 → U2 → U3 as work-unit commits) |
| Delivery strategy | auto-chain |
| Chain strategy | stacked-to-main |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: stacked-to-main
400-line budget risk: Low

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|----------------------|-----------------|-------------------|
| U1 | Guarded `publish` target in `Makefile` | Single PR | `bash` guard matrix (task 1.1) | `make -n publish VERSION=v9.9.9` | Delete `publish` target lines only |
| U2 | `scripts/publish-release.sh` notes + create-or-upload | Single PR | `bash -n scripts/publish-release.sh` | Scratch-repo `notes` run (task 2.4) | Delete script; Makefile/CI untouched |
| U3 | CI gate + permissions in `.github/workflows/ci.yml` | Single PR | `actionlint .github/workflows/ci.yml` (if installed) | Fork tag-push rehearsal (task 4.4) | Revert ci.yml hunk; script inert |

## Phase 1: Foundation — `Makefile` publish target (spec: Publish Guards, Tag Semantics; D1)

- [x] 1.1 RED (shell): run guard matrix — dirty tree (untracked + staged variants), non-main branch, `VERSION=1.2.0`, `VERSION=v0.2.0-rc1`, existing local tag, existing remote tag (`VERSION=v0.1.1`), and `make publish` outside a repo → each must print its exact ERROR message, exit non-zero, create no tag.
- [x] 1.2 Add `publish` to `.PHONY`; guards as POSIX-sh recipe lines in fixed D1 order 1–5 (status porcelain / branch main / `vX.Y.Z` regex / local tag absent / `git ls-remote --exit-code` absent), each failing with `ERROR: <message>` and exit 1.
- [x] 1.3 After guards: `git tag -a "$(VERSION)"` then `git push origin "refs/tags/$(VERSION)"` (explicit refspec, never `--tags`); optional `gh run watch` when `gh` available.
- [x] 1.4 GREEN (shell): `make -n publish VERSION=v9.9.9` → exact tag-a → explicit-refspec push sequence, zero side effects.

## Phase 2: Core — `scripts/publish-release.sh` (spec: Idempotent Release Creation, Asset and Checksums Contract, Curated Release Notes; D3–D4)

- [x] 2.1 Create `scripts/publish-release.sh` (`set -euo pipefail`) with `notes` and `publish` subcommands; TAG from `$GITHUB_REF_NAME`.
- [x] 2.2 `notes`: `git tag -l --format='%(contents)' "$TAG"` → first line = summary; remaining non-empty lines → `## What's new` bullets; append `## Assets` (dist file names) + warning "checksums.txt must keep shipping with every release".
- [x] 2.3 `publish`: `gh release view "$TAG"` exists → upload only missing `dist/upp-*.tar.gz`, `dist/upp-*.zip`, `dist/checksums.txt`; else `notes` → `gh release create "$TAG" ... --title "upp $TAG — $SUMMARY" --notes-file`. Never `git tag`/`git push`/`gh release delete`.
- [x] 2.4 Verify: `bash -n` + shellcheck (if available); scratch tag in scratch repo → notes contain title, What's new, Assets (6 files), checksums warning.

## Phase 3: Integration — CI release job (spec: CI Release Gate, Release Job Permissions; D2, D5)

- [x] 3.1 `.github/workflows/ci.yml`: add `needs: [test, lint]` to `release`; keep `if: startsWith(github.ref, 'refs/tags/v') || github.event_name == 'workflow_dispatch'`.
- [x] 3.2 Job-scoped `permissions: { contents: write }` on `release` only; top-level stays `contents: read`.
- [x] 3.3 Add publish step gated `if: startsWith(github.ref, 'refs/tags/v')` with `env: { GH_TOKEN: '${{ github.token }}' }` running `scripts/publish-release.sh publish`; `actions/upload-artifact` stays for all runs.
- [x] 3.4 Verify: actionlint (if available) + YAML parse; dispatch run must build/upload only (no publish step).

## Phase 4: Verification — shell-only (D6; strict TDD covers Go only — none here)

- [x] 4.1 Guard matrix (1.1) all green; record exact outputs.
- [x] 4.2 `make -n publish VERSION=v9.9.9` sequence exact (1.4).
- [x] 4.3 `actionlint ci.yml`; `shellcheck` + `bash -n` on scripts.
- [ ] 4.4 Fork rehearsal: `make publish VERSION=v0.0.99` → CI green → release with 6 assets; `sha256sum -c` local `make release` checksums vs downloaded `checksums.txt`; job re-run → completes, nothing modified; dispatch → artifacts only, no release. (DEFERRED to sdd-verify)

## Phase 5: Docs — README Release section (spec: Release Documentation; D7)

- [x] 5.1 `README.md`: replace manual attach steps with `make publish VERSION=vX.Y.Z` flow (guards listed; tag message becomes notes; CI completes release); keep `make release` docs.
