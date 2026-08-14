# Proposal: ci-cd-best-practices

## Intent

Harden the minimum viable CI/CD posture for upp without expanding the pipeline: today CI runs have no concurrency control (stale runs race and waste minutes), no per-job timeouts (a hung job blocks a release indefinitely), and the three Bash scripts plus the workflow YAML are never statically checked — broken shell syntax ships through `make release`. Scope is deliberately minimal per user decision (2026-08-13): no new jobs, no permission changes, no dependabot, no attestation.

## Scope

### In Scope

- `concurrency` in `.github/workflows/ci.yml`: `group: ci-${{ github.ref }}`, `cancel-in-progress: ${{ !startsWith(github.ref, 'refs/tags/') }}`. Dispatch shares the main group (benign, accepted); tag runs never cancelled.
- `timeout-minutes`: test 15, lint 10, release 20.
- Script lint as STEPS inside the existing `lint` job (no new jobs; `needs` and permissions untouched):
  - shellcheck `-S warning` over `scripts/install.sh`, `scripts/smoke-test.sh`, `scripts/publish-release.sh` (apt shellcheck).
  - actionlint pinned `go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.7` (lint job already has setup-go + cache).
- Branch protection for `main` — out-of-band GitHub settings, applied during APPLY via `gh api`: require PR before merging, required checks `test` + `lint`, linear history, delete branch on merge.

### Out of Scope

attest-build-provenance · dependabot · upload-artifact `retention-days` · SHA action pinning · README updates · Go version matrix · workflow split · self-hosted runners · spec amendment.

## Capabilities

> Contract with sdd-spec: **no spec changes**. `release-process/spec.md` requires `needs: [test, lint]` and "No additional permissions MAY be granted"; this change keeps both intact, so no delta spec is required. Spec phase is a no-op.

### New Capabilities

None — no spec-level behavior change (config-only change).

### Modified Capabilities

None — `release-process/spec.md` remains valid as-is.

## Approach

Single-file change to `.github/workflows/ci.yml` (only file touched in the PR):

1. Add top-level `concurrency` block (group `ci-${{ github.ref }}`, conditional `cancel-in-progress`).
2. Add `timeout-minutes` to `test` (15), `lint` (10), `release` (20).
3. Inside `lint` job, after golangci-lint: `apt-get install -y shellcheck`, `shellcheck -S warning scripts/*.sh` (all three), then `go run ...actionlint@v1.7.7` on `.github/workflows/ci.yml`.
4. APPLY phase (out-of-band): enable branch protection via `gh api` on `main`; VERIFY step confirms protection active post-apply.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `.github/workflows/ci.yml` | Modified | concurrency, timeouts, shellcheck + actionlint steps in `lint` |
| Branch protection (GitHub settings) | New | out-of-band `gh api` during apply, not a repo file |
| `scripts/*.sh` | Unchanged (fixes may be required) | shellcheck findings fixed in same PR (self-gating) |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| shellcheck first-run findings in install.sh/smoke-test.sh | Med | Same PR fixes them; quality step self-gates the PR |
| shellcheck version drift (apt 0.9.0 vs 0.10.0 in PR #24) | Low | Contained by self-gating; `-S warning` is stable across |
| actionlint pin drift (`v1.7.7`) | Low | Pinned exact version; go module cache |
| Dispatch cancels in-progress main run | Low | Benign (same pipeline); consciously accepted |
| Branch protection never activated | Med | Explicit `gh api` step + verify check in apply |
| Release blocked by failed lint | Low | Fix within same PR; release is idempotent re-runnable |

## Rollback Plan

- CI: revert the single `ci.yml` commit — pipeline returns to pre-change behavior exactly.
- Branch protection: disable via `gh api` (delete required checks/linear-history rules); no code involved.

## Dependencies

- `gh` CLI with write access to repo settings (apply phase, branch protection).
- GitHub-hosted `ubuntu-latest` runner for shellcheck/actionlint steps.

## Success Criteria

- [ ] PR #N merges with `concurrency`, `timeout-minutes`, and lint steps present in `ci.yml`.
- [ ] `lint` job passes with shellcheck (all 3 scripts) + actionlint v1.7.7 on the PR itself.
- [ ] Branch protection active on `main` (verified via `gh api` post-apply): PR required, checks `test`+`lint` required, linear history, auto-delete branches.
- [ ] Existing `needs: [test, lint]` and job-scoped `contents: write` unchanged; release-process spec untouched.
- [ ] Next tag release (`vX.Y.Z`) still publishes end-to-end.
