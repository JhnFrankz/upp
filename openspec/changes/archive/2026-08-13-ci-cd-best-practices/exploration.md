# Exploration: ci-cd-best-practices

**Change**: `ci-cd-best-practices` · **Repo**: `/home/jhan/projects/upp` (public, main branch, solo maintainer) · **Date**: 2026-08-13 · **Mode**: read-only exploration

## Current State (verified facts)

**Workflow** — `.github/workflows/ci.yml` (113 lines):
- Triggers: `push` on `main` + `tags: ['v*']`, `pull_request`, `workflow_dispatch`.
- Top-level `permissions: contents: read`; actions pinned to majors only (`actions/checkout@v4`, `actions/setup-go@v5` with `go-version: '1.22.x'` + `cache: true`, `golangci/golangci-lint-action@v6` with `version: v1.60.3`, `actions/upload-artifact@v4`).
- `test` job: vet, gofmt gate, unit, `-race`, `make build`, smoke.
- `lint` job: golangci-lint v1.60.3.
- `release` job: `needs: [test, lint]`, tag-gated, job-scoped `contents: write`, `fetch-depth: 0`, `make release`, upload-artifact (no `retention-days`), "Restore annotated tag object" step (single-quoted run, 5x retry — merged in PR #26), publish via `scripts/publish-release.sh`.
- Missing (confirmed): `concurrency`, `timeout-minutes`, `retention-days`, attestation, shellcheck/actionlint job, `.github/dependabot.yml` (never existed in git history).

**Branch protection**: confirmed absent via GitHub API (HTTP 404); no repository rulesets. The repo already works PR-first (every commit lands via "Merge pull request #NN").

**Scripts**: `bash -n` passes all three (`install.sh`, `smoke-test.sh`, `publish-release.sh`). shellcheck/actionlint not installed locally (read-only constraint); manual `-S warning`-level review of `install.sh` and `smoke-test.sh` found no violations (high confidence, empirically unverified).

**openspec**: `config.yaml` (strict_tdd: true), 8 baseline specs, 7 archived changes, no active change folder yet.

**Spec conflict discovered**: `openspec/specs/release-process/spec.md` requires the release job to run "only after the `test` and `lint` jobs pass (`needs: [test, lint]`)" and states "No additional permissions MAY be granted" beyond job-scoped `contents: write`. Adding `quality` to `needs` and `attestations`/`id-token` to the release job **requires amending this spec in the same change**.

**Other facts**: `type:chore` label exists; `go.mod` go 1.22 + cobra v1.8.0 + BurntSushi/toml v1.3.2; v0.2.0 (2026-08-13) was the first fully automated release; the prior draft is stale on the restore-step hunk (pre-PR #26) and must be rebased.

## Gap Validation vs Best Practices

| Gap (prior evaluation) | Verdict | Evidence |
|---|---|---|
| HIGH: no branch protection on main | CONFIRMED | API 404; no rulesets |
| MED: no concurrency / cancel-in-progress | CONFIRMED | absent from ci.yml |
| MED: no timeout-minutes | CONFIRMED | absent from ci.yml |
| MED: no shellcheck/actionlint job | CONFIRMED | only test/lint/release jobs exist |
| MED: no dependabot | CONFIRMED | no file, never in history |
| LOW: no retention-days | CONFIRMED | upload-artifact has none |
| LOW: no build attestation | CONFIRMED | no attest step |
| LOW: no SHA pinning (major tags + dependabot chosen) | CONFIRMED as chosen level | all actions `@v4`–`@v6` |

**Refinements**: (1) spec conflict above — must amend `release-process/spec.md`; (2) the quality job is self-gating on its own PR; (3) draft is stale on the restore-step hunk; (4) `workflow_dispatch` shares the main concurrency group (dispatch can cancel an in-progress main run — acceptable but worth deciding); (5) README line 255 says release runs "after `test` and `lint` pass" — adding `quality` makes it stale; (6) dependabot will not bump the golangci-lint `version:` string input.

## Options Considered

- **A. Concurrency**: A1 (rec) `ci-${{ github.ref }}` + `cancel-in-progress: ${{ !startsWith(github.ref, 'refs/tags/') }}`; A2 isolate dispatch via event in key; A3 status quo.
- **B. Pinning**: B1 (rec) major tags + dependabot; B2 full SHA pinning (disproportionate for solo CLI).
- **C. Release gate**: C1 (rec) `needs: [test, lint, quality]` (requires spec amendment); C2 keep `[test, lint]`; C3 rejected.
- **D. Attestation**: D1 (rec) `attest-build-provenance@v2`, `subject-path: dist/*` (includes checksums.txt — self-update trust anchor), job-scoped `attestations: write` + `id-token: write`; D2 tag pushes only.
- **E. Quality job**: E1 (rec) apt shellcheck + `go run ...actionlint@v1.7.7` (pinned, Go cache); E2 static binary; E3 none.
- **F. Branch protection** (out-of-band): F1 (rec) full (PR + checks + linear history + delete branch); F2 checks-only; F3 none. Enforcement: `gh api` vs manual — user decision.
- **G. Dependabot**: G1 (rec) actions + gomod weekly, open-pull-requests-limit 5, labels type:chore; G2 actions-only; G3 none.
- **H. Upload**: H1 (rec) `retention-days: 7` keep on tag runs; H2 gate dispatch-only.

## Recommendations

1. Adopt A1 + B1 + C1 + D1 + E1 + F1 + G1 + H1, rebased onto current ci.yml.
2. Amend `openspec/specs/release-process/spec.md` in the same change.
3. Attest `dist/*` (includes checksums.txt).
4. Plan a verify step for branch protection (gh api check) post-enablement.
5. Decide in proposal round: F1 enforcement mechanism, README one-line sync, dispatch-attest behavior, concurrency key.

## Risks and Non-goals

- Attestation OIDC transient failure → release blocked until idempotent re-run; subject-path guaranteed by `make release` ordering.
- Dispatch-vs-main cancellation race: benign, consciously accepted.
- Shellcheck version drift (apt 0.9.0 vs prior 0.10.0): contained by self-gating quality job.
- Branch protection is a GitHub settings change — risk it never gets enabled without an explicit verify step.
- Dependabot won't bump golangci-lint `version:` input — manual sync.
- Non-goals: Go version matrix, README overhaul, workflow split, self-hosted runners.
