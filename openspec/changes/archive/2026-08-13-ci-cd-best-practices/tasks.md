# Tasks: CI/CD Best Practices (ci-cd-best-practices)

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 40–60 (ci.yml ~30–40; branch-protection apply step; issue/PR overhead) |
| 400-line budget risk | Low |
| Chained PRs recommended | No |
| Suggested split | Single PR |
| Delivery strategy | auto-chain |
| Chain strategy | pending |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: pending
400-line budget risk: Low

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|----------------------|-----------------|-------------------|
| 1 | CI hardening + conditional script fixes (single commit) | PR 1 | `go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.7 .github/workflows/ci.yml` → exit 0, no findings; `shellcheck -S warning scripts/install.sh scripts/smoke-test.sh scripts/publish-release.sh` → exit 0; `python3 -c "import yaml; yaml.safe_load(open('.github/workflows/ci.yml'))"` → exit 0 | Real PR CI run — `lint` + `test` jobs green; post-merge `gh api` read-back confirms protection | Revert the single ci.yml commit (+ script hunks if any); `gh api -X DELETE repos/{owner}/{repo}/branches/main/protection` |

## Phase 1: CI Workflow Hardening (`.github/workflows/ci.yml`)

Tasks 1.1 → 1.4 edit the same file, in order.

- [x] 1.1 Add top-level `concurrency` after `permissions`: `group: ci-${{ github.ref }}`, `cancel-in-progress: ${{ !startsWith(github.ref, 'refs/tags/') }}`. AC: ref-scoped group; tag runs never cancelled. Evidence: actionlint + YAML parse → exit 0. Rollback: remove block.
- [x] 1.2 Add `timeout-minutes` to every job: `test` 15, `lint` 10, `release` 20. AC: all three jobs bounded. Evidence: actionlint + YAML parse → exit 0. Rollback: remove the three lines.
- [x] 1.3 Append Shellcheck step in `lint` AFTER golangci-lint: `sudo apt-get update && sudo apt-get install -y shellcheck`, then `shellcheck -S warning scripts/install.sh scripts/smoke-test.sh scripts/publish-release.sh`. AC: any warning-level finding fails `lint`. Evidence: `shellcheck -S warning scripts/install.sh scripts/smoke-test.sh scripts/publish-release.sh` → exit 0, zero findings. Runtime harness: PR CI run, `lint` job green. Rollback: remove step.
- [x] 1.4 Append Actionlint step: `go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.7 .github/workflows/ci.yml` (exact v1.7.7 pin; run from repo root; setup-go cache warms the module). AC: any workflow error fails `lint`. Evidence: same local `go run ...@v1.7.7` → exit 0, no findings. Runtime harness: PR CI run, `lint` job green. Rollback: remove step.

## Phase 2: Script Fixes (conditional, self-gating)

- [x] 2.1 If task 1.3 finds warnings: fix them in `scripts/install.sh`, `scripts/smoke-test.sh`, `scripts/publish-release.sh` (same PR; behavior identical). AC: `shellcheck -S warning` exits 0 on all three. Evidence: command → exit 0, zero findings. Rollback: revert script hunks.

## Phase 3: Delivery (issue-first, PR, out-of-band protection)

Dependency: phases 1–2 before 3.2; 3.3 after 3.2 merges.

- [x] 3.1 Create/use GitHub issue for CI hardening with repo-convention labels (bug/type). AC: issue open and referenced by the PR. Evidence: issue #27 created and closed by PR #28.
- [x] 3.2 Single commit `chore(ci): harden CI with concurrency, timeouts, and script linting`; ONE push; open single PR; pass the repo's RDD review flow (4R canonical). AC: merged PR containing exactly one commit. Evidence: PR #28 merged into main via merge commit `5b867d9` (2026-08-14, regular merge).
- [x] 3.3 After merge, apply protection: `gh api --method PUT repos/{owner}/{repo}/branches/main/protection --input -` with `required_status_checks.contexts: ["Test", "Lint"]` (EXACT job names — capitals; never lowercase), `required_pull_request_reviews.required_approving_review_count: 0`, `required_linear_history: true`, `delete_branch_on_merge: true`, enforcement enabled. AC: idempotent PUT succeeds.
- [x] 3.4 Read-back verify: `gh api repos/{owner}/{repo}/branches/main/protection --jq '{pr_required: (.required_pull_request_reviews != null), checks: .required_status_checks.contexts, linear: .required_linear_history, delete_branch: .delete_branch_on_merge}'` → `pr_required:true checks:["Test","Lint"] linear:true delete_branch:true`; RED check: direct push to `main` rejected. AC: exact match; any mismatch fails apply. Evidence: read-back `pr_required:true checks:["Test","Lint"] strict:true pr_count:0 linear:true delete_branch_on_merge:true` (repo-level).
