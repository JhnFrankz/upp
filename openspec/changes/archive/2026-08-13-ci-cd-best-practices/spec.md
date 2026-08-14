# Delta for CI Workflow Hardening (ci-cd-best-practices)

## Purpose

Config-only delta: documents the CI/CD contract requirements introduced by hardening `.github/workflows/ci.yml` (run concurrency cancellation, per-job timeouts, static checks for scripts and workflow files) plus one out-of-band delivery requirement (main branch protection). This delta does NOT amend `release-process/spec.md`: `needs: [test, lint]`, job-scoped `contents: write`, and the "no additional permissions MAY be granted" rule remain exactly as specified there.

## ADDED Requirements

### Requirement: Run Concurrency Cancellation

CI MUST declare a concurrency group `ci-${{ github.ref }}`. CI MUST set `cancel-in-progress` to `${{ !startsWith(github.ref, 'refs/tags/') }}`: a newer run on the same non-tag ref MUST cancel the superseded run, and runs on `refs/tags/*` MUST NOT be cancelled.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Superseded branch run | In-progress run on `refs/heads/feature-x` | Newer push to same ref | In-progress run cancelled, newer run proceeds |
| Tag run never cancelled | In-progress run for `v0.2.0` tag | Another run starts | No cancellation; tag run completes |
| Dispatch | Manual `workflow_dispatch` run | Run starts | Shares the main group (benign, accepted) |

### Requirement: Job Timeout Bounds

Every CI job MUST declare a `timeout-minutes` limit: `test` 15, `lint` 10, `release` 20. A job exceeding its limit MUST be terminated by GitHub and its run reported as failed.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Hung job | `test` hangs past 15 min | Limit reached | Job terminated, run fails |
| Healthy run | All jobs finish under limits | Jobs complete | No job hits its timeout |

### Requirement: Script and Workflow Static Checks

The `lint` job MUST run `shellcheck -S warning` over `scripts/install.sh`, `scripts/smoke-test.sh`, and `scripts/publish-release.sh`, and MUST run actionlint pinned to `v1.7.7` over the workflow files. Any warning-level shellcheck finding or actionlint error MUST fail the `lint` job.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Shellcheck warning | A script has an unquoted-variable warning | `lint` job runs | Job fails; PR blocked |
| Actionlint error | Workflow YAML contains an invalid key | `lint` job runs | Job fails |
| Clean scripts and workflow | No findings in scripts or YAML | `lint` job runs | Job passes |

### Requirement: Main Branch Protection (Delivery Requirement)

Applied out-of-band at apply time via `gh api` — NOT workflow behavior. `main` MUST require a pull request before merging, MUST require checks `test` and `lint` to pass, MUST require linear history, and MUST auto-delete merged branches. Post-apply verification MUST confirm the protection is active.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Protection active | Apply completed | Verify via `gh api` | PR required, checks test+lint, linear history, auto-delete confirmed |
| Direct push | Protection active | Push to `main` | Rejected |
