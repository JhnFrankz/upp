# Archive Report — upp-publish-automation

**Archived**: 2026-08-12
**Archived to**: `openspec/changes/archive/2026-08-12-upp-publish-automation/`
**Artifact store mode**: hybrid (filesystem spec sync + archive move + Engram archive report)
**Archive report language**: English
**Change**: Automate GitHub release publishing (`make publish` guarded tag target, CI release gate + idempotent create-or-upload, curated release notes)

## SDD Cycle Status

| Gate | Result | Evidence |
|------|--------|----------|
| Task Completion Gate | PASS WITH RECONCILIATION — 16/17 tasks complete; 1 unchecked (4.4 fork rehearsal) is NOT stale: it was genuinely UNEXECUTED (deferred by orchestrator, out of scope — `forkCount=0`, no fork created). Orchestrator final-state facts explicitly instruct recording "16/17 complete, 4.4 deferred" at close; rehearsal procedure recorded in verify-report.md and carried forward as a follow-up. | archived `tasks.md`: 16 `[x]` (1.1–1.4, 2.1–2.4, 3.1–3.4, 4.1–4.3, 5.1), 1 `- [ ]` (4.4, marked DEFERRED) |
| Native Review Receipt Gate | N/A — `reviewGate` structurally absent; no review was ever started for this candidate; archive proceeded under ordinary repository policy | absence is not a defect; nothing to read or block on |
| Verification | PASS WITH WARNINGS — 8/8 requirements, 25/25 scenarios in the envelope, 0 blockers, 0 CRITICAL; 17/25 scenarios fully COMPLIANT, 8 PARTIAL (all sharing one root cause: task 4.4 unexecuted) | admitted by `gentle-ai sdd-verify-validate` (`valid=true`), verdict `pass_with_warnings`, evidence_revision `sha256:56bdc0ea3286781ea53bc111ab7467600300fe616b536f9747bc3c5de70cb6d3` |
| CRITICAL findings gate | No CRITICAL issues in verify-report | 0 critical_findings, 0 blockers |
| Mechanical Copy Contract | PASS — both `diff -r` readbacks EMPTY (verbatim output: no lines), status 0 | pre-move recursive snapshot vs archived tree byte-identical; delta spec vs new main spec byte-identical |

## Final-State Facts (at close)

1. **Tasks**: 16/17 complete. Only 4.4 (fork rehearsal) is unchecked — unexecuted, not stale: `gh api repos/JhnFrankz/upp/forks` → empty, `forkCount` → 0; creating a fork was out of scope. The complete 5-step rehearsal procedure is recorded in verify-report.md ("Task 4.4 Fork Rehearsal — UNEXECUTED (honest record)").
2. **Correction to apply-progress.md**: its "15/16 tasks complete" status line is a miscount (tasks.md contains 17 tasks). verify-report.md already noted the miscount; the corrected count at close is 16/17, 4.4 deferred. apply-progress.md is archived as written (audit trail preserved) — this report is the authoritative correction.
3. **Delivery**: NOT merged at close. PR #24 (`feat/publish-automation`, +136/−6, base main, label type:feature, Closes #23) is OPEN on GitHub — merging is the maintainer's separate decision. Implementation commits on the branch: `2251205` (U1 Makefile publish target), `8424b17` (U2 scripts/publish-release.sh), `39b3f21` (U3 ci.yml hardening), `17ce7e1` (README docs). Issue #23 was created by the apply phase; it does not yet carry a maintainer-applied approval label.
4. **Verify verdict**: PASS WITH WARNINGS. 43/43 shell assertions across 3 harnesses green (guard matrix 11/11, notes matrix 22/22, ci matrix 10/10), exact `make -n` sequence, shellcheck/actionlint/go vet clean, Go suite green (0 Go files changed). The single gap is the deferred live GitHub fork rehearsal (task 4.4), reflected as 8 PARTIAL scenarios.
5. **Verify WARNING carried forward** (not fixed before archive — open at close):
   - (W1) Task 4.4 fork rehearsal UNEXECUTED: 8/25 scenarios PARTIAL (fully implemented, structurally verified, live GitHub execution unproven). The first real `make publish` (v0.2.0) will be the de facto live rehearsal; until then idempotency and release-creation paths carry residual integration risk.
6. **No publish occurred during the cycle**: no tag was created and nothing was published. Real publishing starts with the first `make publish VERSION=vX.Y.Z` after merge.
7. **Shell verification harnesses** live in `/tmp/opencode` (`upp-guard-matrix.sh`, `upp-notes-matrix.sh`, `upp-ci-matrix.py`) — transient local state referenced in evidence, not part of the repo. shellcheck/actionlint binaries were obtained as static binaries in `/tmp/opencode/bin` (no system install).
8. **Deferred follow-ups recorded for the next change**: fork rehearsal (task 4.4 procedure from verify-report), live-first-publish monitoring (v0.2.0 as de facto rehearsal), optional shellcheck/actionlint in CI, issue #23 approval by the maintainer.

## Specs Synced (delta → main)

| Domain | Action | Details |
|--------|--------|---------|
| release-process | **Created** (new domain) | Full spec mechanically copied (shell `cp` → `diff -r` → `mv`): 8 requirements (Publish Guards, Tag Semantics, CI Release Gate, Idempotent Release Creation, Asset and Checksums Contract, Curated Release Notes, Release Job Permissions, Release Documentation), 25 scenarios. No `(Previously: ...)` delta annotations exist to carry. |

Merge summary: **1 domain created, 0 MODIFIED, 0 ADDED-appended, 0 REMOVED, 0 RENAMED** — the delta IS a full spec for a new domain. `rules.archive` ("Warn before merging destructive deltas"): N/A — non-destructive creation.

Main spec created: `openspec/specs/release-process/spec.md` (mode 644, repo convention). Also updated `openspec/config.yaml` context line "Specs: 6 domains" → "Specs: 8 domains" (was stale since the previous archive added self-update; now includes release-process).

## Mechanical Copy Evidence

- Move mechanism: `git mv` failed (`fatal: source directory is empty` — change folder untracked, consistent with prior archives), skill-mandated fallback `mv` succeeded; source-gone check passed.
- `diff -r "$snapshot_root/source" "openspec/changes/archive/2026-08-12-upp-publish-automation"` → **EMPTY output, status 0** (byte-identical).
- New-domain spec copy: `diff -r` source vs `mktemp` staging → **EMPTY output, status 0** before `mv` into `openspec/specs/release-process/`; re-verified byte-identical after `chmod 644` alignment (mktemp created the staged file as 600).
- `archive-report.md` is additive-only, written after the readback, excluded from the comparison.

## Archive Contents

- `exploration.md` ✅
- `proposal.md` ✅
- `specs/` ✅ (1 domain: release-process)
- `design.md` ✅
- `tasks.md` ✅ (16/17 tasks complete; 4.4 unchecked, genuinely deferred — not stale)
- `apply-progress.md` ✅ (single batch U1–U3 + docs, TDD Cycle Evidence tables; "15/16" status line corrected by this report to 16/17)
- `verify-report.md` ✅ (PASS WITH WARNINGS)
- `archive-report.md` ✅ (this file, additive)

Active changes directory (`openspec/changes/`) no longer contains this change (only `archive/` remains).

## Engram Lineage

| Artifact | Engram observation |
|----------|--------------------|
| archive-report | saved under topic `sdd/upp-publish-automation/archive-report` (this report) |

Hybrid mode archive executed from the filesystem artifacts listed above; Engram persistence covers the archive report. (Earlier-phase Engram observation IDs were not passed in the archive launch context and are not re-derivable from the archived folder; the filesystem audit trail is the authoritative record.)

## Verdict

SDD cycle complete: explored, proposed, specified (8 requirements / 25 scenarios), designed, implemented (16/17 tasks — 4.4 fork rehearsal deferred and honestly recorded as UNEXECUTED), verified (PASS WITH WARNINGS, 0 CRITICAL, admitted), and archived. Delivery is a single open PR #24 (Closes #23) awaiting maintainer merge — merge and the first real `make publish` are post-archive maintainer decisions. Ready for the next change.
