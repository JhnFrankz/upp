# Design: CI/CD Best Practices (ci-cd-best-practices)

## Technical Approach

Config-only change to `.github/workflows/ci.yml` plus one out-of-band delivery step (branch protection via `gh api` at apply). No restructure, permissions, or new jobs; `needs: [test, lint]` and the release-process spec untouched. The post-PR #26 "Restore annotated tag object" step (single-quoted 5x-retry run) stays verbatim.

Three additive blocks in ci.yml:

1. **Top-level `concurrency`** (after `permissions`):

   ```yaml
   concurrency:
     group: ci-${{ github.ref }}
     cancel-in-progress: ${{ !startsWith(github.ref, 'refs/tags/') }}
   ```

   Non-tag runs on the same ref cancel superseded peers; tag runs and dispatch never cancel.

2. **`timeout-minutes`** per job: `test: 15`, `lint: 10`, `release: 20`.

3. **Static-check steps appended inside the existing `lint` job**, AFTER the `golangci-lint` step:

   - `Shellcheck`: `sudo apt-get update && sudo apt-get install -y shellcheck`, then `shellcheck -S warning scripts/install.sh scripts/smoke-test.sh scripts/publish-release.sh`. apt is a single uncached install, ~10s warm-up inside the lint timeout.
   - `Actionlint`: `go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.7 .github/workflows/ci.yml`. Exact pin; setup-go `cache: true` already warms the module fetch — no wrapper action to pin.

   Both steps: default `bash`, no `if:` guards — any finding fails the job (per spec).

## Architecture Decisions

| Decision | Options | Tradeoff | Decision |
|---|---|---|---|
| Concurrency group | Per-branch vs `ci-${{ github.ref }}` | Per-branch over-groups push+PR; ref-scope: one active run per ref | `ci-${{ github.ref }}` (per proposal) |
| Protection API | Classic branch protection (PUT `/branches/main/protection`) vs rulesets | Rulesets: modern but POST `rules[]` payload and ordering; classic: one idempotent PUT, enough for a solo repo | Classic branch protection |
| Review requirement | 1 approving review vs 0 | 1 blocks the solo maintainer (cannot approve own PR); 0 = PR + checks only | PR required, `required_approving_review_count: 0` |
| Check contexts | lowercase `test`/`lint` vs job names | GitHub matches contexts against the job `name:`; lowercase never matches, blocks every merge | `["Test", "Lint"]` (job names) — flag to apply/verify |
| `enforce_admins` | true vs false | true blocks admin bypass but risks self-lockout; false keeps a recovery path | `false` |

## Data Flow

    push/PR/tag ──▶ concurrency gate ──▶ test (15m) + lint (10m, +shellcheck/actionlint)
      ──▶ needs ──▶ release (20m) ──▶ GitHub Release
    apply ──▶ gh api PUT protection(main) ──▶ read-back verify

## File Changes

| File | Action | Description |
|---|---|---|
| `.github/workflows/ci.yml` | Modify | +concurrency block, +3 `timeout-minutes`, +2 steps in `lint` |
| `scripts/*.sh` | Modify (only on findings) | shellcheck fixes in same PR (self-gating) |
| Branch protection on `main` | New (out-of-band) | `gh api` at apply; not a repo file |

## Interfaces / Contracts

Apply-time delivery step:

```bash
gh api --method PUT repos/{owner}/{repo}/branches/main/protection --input - <<'EOF'
{
  "required_status_checks": {"strict": true, "contexts": ["Test", "Lint"]},
  "enforce_admins": false,
  "required_pull_request_reviews": {"required_approving_review_count": 0},
  "restrictions": null,
  "required_linear_history": true,
  "allow_force_pushes": false,
  "allow_deletions": false,
  "delete_branch_on_merge": true
}
EOF
```

Post-apply read-back:

```bash
gh api repos/{owner}/{repo}/branches/main/protection \
  --jq '{pr_required: (.required_pull_request_reviews != null),
         checks: .required_status_checks.contexts,
         linear: .required_linear_history,
         delete_branch: .delete_branch_on_merge}'
```

Expected: `pr_required: true, checks: ["Test", "Lint"], linear: true, delete_branch: true`; any mismatch fails apply.

## Verification Strategy

| Layer | What to Verify | Approach |
|---|---|---|
| Local, pre-push | actionlint, shellcheck (if installed), YAML validity | `go run ...actionlint@v1.7.7 .github/workflows/ci.yml`; `shellcheck -S warning scripts/*.sh`; `python3 -c "import yaml; yaml.safe_load(open('.github/workflows/ci.yml'))"` |
| CI self-gate | New lint steps pass on the PR | lint runs shellcheck+actionlint; `needs` unchanged |
| Post-apply | Protection active | `gh api` read-back; direct push to `main` rejected |

## Threat Matrix

| Boundary | Applicability | Design response | Planned RED tests |
|---|---|---|---|
| Documentation-like paths | N/A — no executable docs | — | none |
| Git repository selection | N/A — checkout unchanged | — | none |
| Commit state | N/A — no new commit logic | — | none |
| Push state | Applicable — protection rejects direct pushes to `main` | `enforce_admins: false`; read-back asserts rules | apply verify: read-back proves protection; direct push rejected |
| PR commands | Applicable — merges require PR with Test+Lint checks | contexts = job names; PR-only merge path | apply verify: read-back asserts contexts |

## Migration / Rollout

No migration. Rollback boundary: revert the single `ci.yml` commit (pipeline returns to pre-change behavior exactly); disable protection via `gh api -X DELETE repos/{owner}/{repo}/branches/main/protection` — no code involved.

## Risks

| Risk | Likelihood | Mitigation |
|---|---|---|
| shellcheck first-run findings in the 3 scripts | Med | Fixed in same PR; lint job self-gates |
| apt shellcheck 0.9.0 vs 0.10.0 drift (PR #24) | Low | Self-gating; `-S warning` stable across versions |
| actionlint pin drift | Low | Exact `@v1.7.7` pin + Go module cache |
| Dispatch cancels in-progress main run | Low | Benign (same pipeline); consciously accepted |
| Protection never activated / wrong check contexts | Med | Explicit apply step + read-back verify; contexts are job names |
| Release blocked by failing lint | Low | Fix in same PR; release idempotent, re-runnable |

## Open Questions

None blocking. Apply note: confirm contexts resolve to job names via the read-back.
