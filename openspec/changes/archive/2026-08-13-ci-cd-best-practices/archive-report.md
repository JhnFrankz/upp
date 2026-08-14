# Archive Report — ci-cd-best-practices

**Archived**: 2026-08-13
**Archived to**: `openspec/changes/archive/2026-08-13-ci-cd-best-practices/`
**Artifact store mode**: hybrid (filesystem spec sync + archive move + Engram archive report)
**Archive report language**: English
**Change**: Harden the minimum viable CI/CD posture — CI run concurrency cancellation, per-job timeouts, script/workflow static checks in the `lint` job, and main branch protection (delivery requirement)

## SDD Cycle Status

| Gate | Result | Evidence |
|------|--------|----------|
| Task Completion Gate | PASS WITH RECONCILIATION — 9/9 tasks complete. Tasks 3.1–3.4 were stale unchecked boxes in `tasks.md`: they are orchestrator-owned delivery tasks (per apply-progress.md, batch 1 owned 1.1–1.4 + 2.1) proven complete by verify-report.md ("Delivery tasks (3.1–3.4, orchestrator-owned) | 4/4 complete") and by the orchestrator's final-state handoff (issue #27 closed, PR #28 merged, protection applied + read-back verified). Checkboxes reconciled at archive time per the orchestrator's explicit final-state facts; reason recorded here. | archived `tasks.md`: 9 `[x]` (1.1–1.4, 2.1, 3.1–3.4) |
| Native Review Receipt Gate | N/A — `reviewGate` structurally absent; no review was ever started for this candidate; archive proceeded under ordinary repository policy | absence is not a defect; nothing to read or block on |
| Verification | PASS — 4/4 requirements, 10/10 scenarios in the envelope, 0 blockers, 0 CRITICAL, 0 WARNING | admitted by the native validator; verify-report.md frontmatter `verdict: pass`, `requirements: 4/4`, `scenarios: 10/10` |
| CRITICAL findings gate | No CRITICAL issues in verify-report | 0 critical_findings, 0 blockers |
| Mechanical Copy Contract | PASS — `diff -r` readback EMPTY (verbatim output: no lines), status 0 | pre-move recursive snapshot vs archived tree byte-identical; baseline spec merge verified below (requirement bodies byte-identical) |

## Final-State Facts (at close)

1. **Delivery**: PR #28 merged into `main` as merge commit `5b867d9` (2026-08-14, regular merge; 2 commits: `aa93e0b` chore(ci) hardening, `a72b448` fix(ci) actionlint gate + drop redundant apt install). Issue #27 closed by the PR.
2. **CI green on the merged PR** (run 31767115742): Test pass 1m47s, Lint pass 22s — the new Shellcheck and Actionlint steps ran and passed in real CI; Release job skipped assets (no tag — expected).
3. **Branch protection for `main` ACTIVE and read-back verified**: `pr_required:true checks:["Test","Lint"] strict:true pr_count:0 linear:true`; `delete_branch_on_merge:true` (repo level); force pushes and deletions disabled (strict + linear history enforcement).
4. **Verify verdict**: PASS — 4/4 requirements, 10/10 scenarios compliant. Local gates (go test 9/9, go vet, gofmt, actionlint v1.7.7, shellcheck -S warning, PyYAML) all exit 0 on the merged tree; CI run evidence plus exact branch-protection read-back.
5. **No CRITICAL, no WARNING, no blockers** at close. All follow-ups below are non-blocking SUGGESTION-level items.

## Follow-ups (recorded, non-blocking)

1. Add comments in `.github/workflows/ci.yml` documenting (a) why `cancel-in-progress` excludes `refs/tags/*` (tag runs must never be cancelled), and (b) why actionlint is pinned to `@v1.7.7`.
2. Consider isolating `workflow_dispatch` runs from the branch concurrency group (e.g. a separate group for `github.event_name == 'workflow_dispatch'`) so a manual dispatch is never cancelled by a push — currently shares `ci-${{ github.ref }}` (spec-accepted as benign).
3. Shellcheck version drift: CI uses the runner-preinstalled shellcheck (~0.9.x on ubuntu-latest) while local verification used the v0.10.0 static binary — both clean today, but versions differ; pin if stricter equivalence is ever needed.

## Specs Synced (delta → main)

| Domain | Action | Details |
|--------|--------|---------|
| ci-workflow | **Created** (new domain) | Delta spec merged into new baseline `openspec/specs/ci-workflow/spec.md`: 4 requirements (Run Concurrency Cancellation, Job Timeout Bounds, Script and Workflow Static Checks, Main Branch Protection (Delivery Requirement)), 10 scenarios. Requirement bodies and scenario tables preserved byte-identical from the delta (verified by `diff` of non-header lines); only the wrapper was normalized (title, purpose, `## ADDED Requirements` → `## Requirements`). `release-process/spec.md` NOT amended — the delta's own Purpose explicitly excludes it (`needs: [test, lint]`, job-scoped `contents: write`, and "no additional permissions MAY be granted" remain as specified there). |

Merge summary: **1 domain created, 0 MODIFIED, 0 appended, 0 REMOVED, 0 RENAMED** — delta maps to no existing baseline domain, so a new domain was created per repo convention (same approach as the publish-automation archive creating `release-process`). `rules.archive` ("Warn before merging destructive deltas"): N/A — non-destructive creation.

Also updated `openspec/config.yaml` context line "Specs: 8 domains (…)" → "Specs: 9 domains (command-interface, config-system, ci-workflow, platform-detection, security-model, tool-adapter, ux-patterns, self-update, release-process)" following the repo convention set by the publish-automation archive.

## Mechanical Copy Evidence

- Move mechanism: `git mv` failed (`fatal: source directory is empty` — change folder untracked, consistent with prior archives), skill-mandated fallback `mv` succeeded; source-gone check passed.
- `diff -r "$snapshot_root/source" "openspec/changes/archive/2026-08-13-ci-cd-best-practices"` → **EMPTY output, status 0** (byte-identical).
- Baseline merge fidelity: `diff` of delta vs baseline with header lines stripped → only 2 wrapper lines differ (purpose sentence and `## ADDED Requirements` vs `## Requirements`); all requirement paragraphs and scenario tables byte-identical.
- `archive-report.md` is additive-only, written after the readback, excluded from the comparison.

## Archive Contents

- `exploration.md` ✅
- `proposal.md` ✅
- `spec.md` ✅ (delta spec, 4 ADDED requirements / 10 scenarios)
- `design.md` ✅
- `tasks.md` ✅ (9/9 tasks complete; 3.1–3.4 checkboxes reconciled at archive — see Task Completion Gate)
- `apply-progress.md` ✅ (batch 1 of 1: 5/5 assigned tasks complete)
- `verify-report.md` ✅ (PASS)
- `archive-report.md` ✅ (this file, additive)

Active changes directory (`openspec/changes/`) no longer contains this change (only `archive/` remains).

## Engram Lineage

| Artifact | Engram observation |
|----------|--------------------|
| exploration | #183 (`sdd/ci-cd-best-practices/explore`) |
| proposal | #184 (`sdd/ci-cd-best-practices/proposal`) |
| spec | #185 (`sdd/ci-cd-best-practices/spec`) |
| design | #186 (`sdd/ci-cd-best-practices/design`) |
| tasks | #187 (`sdd/ci-cd-best-practices/tasks`) |
| apply-progress | #188 (`sdd/ci-cd-best-practices/apply-progress`) |
| verify-report | #191 (`sdd/ci-cd-best-practices/verify-report`) |
| archive-report | saved under topic `sdd/ci-cd-best-practices/archive-report` (this report) |

Hybrid mode archive executed from the filesystem artifacts listed above; Engram persistence covers the archive report. All artifact observations read for this report: #183–#188, #191.

## Verdict

SDD cycle complete: explored, proposed, specified (4 requirements / 10 scenarios), designed, implemented (9/9 tasks), verified (PASS, 4/4 + 10/10, admitted by the native validator), delivered (PR #28 merged, issue #27 closed, CI green, branch protection active and read-back verified), and archived. Three non-blocking follow-ups recorded. Ready for the next change.
