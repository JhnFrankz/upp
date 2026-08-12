# Archive Report — upp-versioning-auto-update

**Archived**: 2026-08-12
**Archived to**: `openspec/changes/archive/2026-08-12-upp-versioning-auto-update/`
**Artifact store mode**: hybrid (filesystem spec sync + archive move + Engram archive report)
**Archive report language**: English
**Change**: Self-Update & Opt-in Update Detection (`upp self-update` command, `check_self_update` hint, `internal/selfupdate/` package)

## SDD Cycle Status

| Gate | Result | Evidence |
|------|--------|----------|
| Task Completion Gate | PASS — 14/14 tasks complete, 0 unchecked | archived `tasks.md`: 14 `[x]` (1.1–1.3, 2.1–2.3, 3.1–3.2, 4.1–4.2, 5.1–5.4), 0 `- [ ]` |
| Native Review Receipt Gate | N/A — `reviewGate` structurally absent; no review was ever started for this candidate; archive proceeded under ordinary repository policy | absence is not a defect; nothing to read or block on |
| Verification | PASS WITH WARNINGS — 15/15 requirements, 61/61 scenarios, 0 blockers, 0 CRITICAL, 4 WARNINGs | admitted by `gentle-ai sdd-verify-validate` (`valid: true`), evidence_revision `sha256:bb882df12a26fc76556e7e5ee124bb35e724ee49cc888b708c9d720995e5119d` (per verify-report observation #132 + orchestrator final-state handoff) |
| CRITICAL findings gate | No CRITICAL issues in verify-report | 0 critical_findings |
| Mechanical Copy Contract | PASS — `diff -r` empty (verbatim output: no lines), status 0 | pre-move recursive snapshot vs archived tree byte-identical |

## Final-State Facts (at close)

1. **Tasks**: 14/14 complete in the persisted tasks artifact (0 unchecked). No archive-time stale-checkbox reconciliation needed.
2. **Delivery**: COMPLETED — 6 stacked PRs (#17–#22) MERGED to main, tip `77c385a` (merge of #22, 2026-08-12). Issue #16 (enhancement) tracks the change.
   - PRs #17/#18 = original commits `e8c6479` (U1 version compare + asset mapping), `694d10d` (U2 detection cache).
   - PRs #19/#20 = U3 client / U4 replace commits plus lint-fix commits `950d5e5`, `e734103` (errcheck/SA1019 golangci-lint CI-only findings).
   - PRs #21/#22 = U5 command / U6 hint commits rebased onto updated main: `dd2a700`, `a234c35` — identical trees to the settled candidates `ba0a6df` / `41edbad` (the verify-phase HEAD).
3. **Verify verdict**: PASS WITH WARNINGS, `valid: true`, evidence revision `bb882df1…e5119d`. Full suite `go test ./... -count=1` green on merged main (9/9 packages) — re-verified after delivery.
4. **Verify WARNINGs carried forward** (pre-disclosed hardening items, NOT fixed before archive — recorded as open at close):
   - (W1) No literal unknown-command / bare-root execution test — cobra framework-default rejection, mechanism proven through `root.Execute`; pre-existing test hole, not a regression.
   - (W2) `Prepare` temp-dir leak after replace/decline (`/tmp/upp-selfupdate-*` not removed) — hardening follow-up; no correctness impact.
   - (W3) Cache plain-write race — self-heals via miss-on-corrupt (D3); negligible frequency.
   - (W4) `LatestFresh` not write-through (D4 carried) — no spec requires it; hint path writes its own cache.
5. **Native attempt ledger**: all `sdd-attempt` attempts settled (U1–U6 + verify). Two maintainer resets recorded (U2, U3, U4 budget overruns — re-acquired with correct budgets). No unresolved attempts.
6. **Deferred follow-ups recorded for the next change**: `make publish` automation (releases still manual; self-update fails closed without `checksums.txt`); Windows self-replace support; temp-dir leak hardening.

## Specs Synced (delta → main)

| Domain | Action | Details |
|--------|--------|---------|
| self-update | **Created** (new domain) | Full spec mechanically copied (shell `cp` → `diff -r` → `mv`): 8 requirements R1–R8 (Version Parsing, GitHub Release Detection, Asset Mapping, Download & Checksum, Extraction Safety, Atomic Replacement, Confirmation Gate, Platform Support), 28 scenarios. |
| command-interface | Updated | 2 MODIFIED requirements replaced with full delta blocks: "Command Structure" (7 commands incl. `self-update`; no alias with `update`), "`upp check`" (hint after summary when `check_self_update` on; zero self-update network otherwise). 1 ADDED requirement appended after `upp check`: "Self-Update Flag Semantics" (no flags in v1, cobra rejection of unknowns, `--ci` denies, `--only`/`--skip` ignored, `--quiet` never suppresses prompt/deny). 4 non-delta requirements preserved (Global Flags, `upp init`, `upp update`, `upp export`/`upp import`). |
| config-system | Updated | 1 ADDED requirement appended at end of Requirements: "Self-Update Detection Setting" (`settings.check_self_update`, default `false`, zero network test-enforced, cache at `{config-dir}/self-update-cache.json`). 5 non-delta requirements preserved. |
| security-model | Updated | 1 MODIFIED requirement replaced with full delta block: "Official Tool Integrity" (self-update integrity fails closed: sha256 vs `checksums.txt` from SAME release over HTTPS, mismatch/missing abort untouched, extracted never executed). 4 non-delta requirements preserved (Tool Trust Levels, Confirmation for Destructive Operations, Config Trust Override, Output Transparency). |
| ux-patterns | Updated | 2 ADDED requirements appended at end of Requirements: "Self-Update Detection Hint" (exactly one line `⬆️ upp v{latest} available (current {current}) — run "upp self-update"`, exit unchanged, silent offline/quiet/up-to-date) and "Self-Update Confirmation Prompt" (localized en/es, versions + path, non-TTY/`--ci` deny non-zero). 7 non-delta requirements preserved. |

Merge summary: **1 domain created, 3 MODIFIED replaced, 4 ADDED appended, 0 REMOVED, 0 RENAMED**. All 15 delta requirements (8 self-update + 3 command-interface + 1 config-system + 1 security-model + 2 ux-patterns) now live in the main specs. `rules.archive` ("Warn before merging destructive deltas"): N/A — non-destructive merge, no sections removed.

Main specs updated: `openspec/specs/self-update/spec.md`, `openspec/specs/command-interface/spec.md`, `openspec/specs/config-system/spec.md`, `openspec/specs/security-model/spec.md`, `openspec/specs/ux-patterns/spec.md`.

## Mechanical Copy Evidence

- Move mechanism: `git mv` failed (`fatal: source directory is empty` — change folder untracked, consistent with prior archives), skill-mandated fallback `mv` succeeded; source-gone check passed.
- `diff -r "$snapshot_root/source" "openspec/changes/archive/2026-08-12-upp-versioning-auto-update"` → **EMPTY output, status 0** (byte-identical).
- New-domain spec copy: `diff -r` source vs `mktemp` staging → **EMPTY output, status 0** before `mv` into `openspec/specs/self-update/`; re-verified identical after permission alignment (`chmod 644` to repo convention).
- `archive-report.md` is additive-only, written after the readback, excluded from the comparison.

## Archive Contents

- `exploration.md` ✅
- `proposal.md` ✅
- `specs/` ✅ (5 domains: command-interface, config-system, security-model, self-update, ux-patterns)
- `design.md` ✅
- `tasks.md` ✅ (14/14 tasks complete, 0 unchecked)
- `apply-progress.md` ✅ (6 batches U1–U6, TDD Cycle Evidence tables)
- `verify-report.md` ✅ (PASS WITH WARNINGS)
- `archive-report.md` ✅ (this file, additive)

Active changes directory (`openspec/changes/`) no longer contains this change (only `archive/` remains).

## Engram Lineage (observation IDs read)

| Artifact | Engram observation |
|----------|--------------------|
| explore | #113 |
| proposal | #115 |
| spec | #116 |
| design | #117 |
| tasks | #120 |
| apply-progress | #123 |
| verify-report | #132 |
| delivery decision (6 PRs merged #17–#22) | #133 |
| chain strategy decision | #114 |
| archive-report | saved under topic `sdd/upp-versioning-auto-update/archive-report` |

## Verdict

SDD cycle complete: explored, proposed, specified (15 requirements / 61 scenarios), designed, implemented (14/14 tasks, 6 stacked PRs #17–#22 MERGED to main tip 77c385a), verified (PASS WITH WARNINGS, 0 CRITICAL, admitted), and archived. Ready for the next change.
