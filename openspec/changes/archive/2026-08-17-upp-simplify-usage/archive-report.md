# Archive Report: upp-simplify-usage

**Change**: upp-simplify-usage — Simplify upp Usage: Honest Output and Zero-Config Story
**Archived**: 2026-08-17
**Delivery**: auto-chain, stacked-to-main — 2 chained PRs
- PR #64 (slice 1, presentation/UX): merged to main as `766e9bb` (feat(cli): honest output and zero-config story)
- PR #65 (slice 2, data quality): merged to main as `f49014e` (feat(adapters): honest data quality — skipped counts, nvm semver, drop dead interactive setting)
- Issue #63: CLOSED (`gh issue view 63` → state CLOSED, closedAt 2026-08-17T15:24:18Z)

**Reviewed HEAD**: `f49014e` (both PRs merged to main).
**RDD review receipts**: lineages `review-ca7d43a46d8edc05` (S1) and `review-879fd87b35a59c15` (S2) — passed with no BLOCKER/CRITICAL.

## Purpose

Close the SDD cycle: merge the 4 delta specs into the canonical specs (source of truth), mark the archive task complete, and record the final state of the change.

## Task Completion Gate

Task 3.1 was the single remaining unchecked implementation task (archive phase's own job). The Task Completion Gate passed: all implementation tasks 1.1–1.11 and 2.1–2.8 are marked `[x]`, and `verify-report.md` exists with verdict PASS. Task 3.1 is now marked `[x]` by this archive phase (see tasks.md). No stale checkboxes remain.

## Per-Domain Merge Results

All four delta specs were merged into existing canonical specs. Merges followed the OpenSpec delta convention: ADDED requirements appended to the Requirements section, MODIFIED requirements replaced the full matching block (including unchanged scenarios preserved and the delta's `(Previously: ...)` note carried through, per repo precedent in archived changes). Requirements not present in the deltas were preserved untouched.

| Domain | Delta action | Merge result | Final canonical state |
|--------|--------------|--------------|------------------------|
| ux-patterns | 1 ADDED (List Table Output) + 3 MODIFIED (Summary Report, Progress Indication, Self-Update Confirmation Prompt) | **Clean / idempotent** | 10 requirements (9 pre-existing + 1 new). The Self-Update Confirmation Prompt MODIFIED was **idempotent**: the D10 drift fix ("localized (en/es)" → English-only, 2-line/2-scenario change) was already applied directly to the canonical spec in S1 (commit `766e9bb`), so this delta's English-only requirement text and scenarios matched the canonical spec exactly — verified no duplication (heading count = 1) and no conflicting text. The `(Previously: ...)` note is the only delta-only addition in that requirement. Summary Report gained 2 scenarios (Check with skips, Dry-run pending); Progress Indication gained the "Checking X/Y" vs "Updating X/Y" wording and 1 scenario (Multi-tool check). |
| command-interface | 1 ADDED (Help Output Grouping) | **Clean** | 8 requirements (7 pre-existing + 1 new). Help Output Grouping appended after `upp export`/`upp import` with its 3 scenarios. |
| tool-adapter | 1 MODIFIED (Version Comparison) | **Clean** | 8 requirements (unchanged count). Version Comparison replaced the 2-scenario block with the full semver-comparison contract (no-downgrade rule, fail-closed unparseable, nvm MUST use semver) + 4 new scenarios (Newer current, Older current, Equal versions, Unparseable). Pre-existing scenarios Semver version and Non-semver preserved. |
| config-system | 1 MODIFIED (Config Format) | **Clean** | 6 requirements (unchanged count). Config Format: minimum-structure TOML dropped the `[settings] interactive = true` block (now starts with `[tools]`), added the `settings.interactive` MUST-NOT-exist paragraph, and gained 2 scenarios (Stray interactive, Init/export hygiene). Pre-existing language-key paragraph merged with the new interactive paragraph; old `interactive = true` text survives only in the `(Previously: ...)` note. |

**Merge verification**: post-merge grep confirms all 10 delta requirements appear exactly once in their canonical spec (no duplicates) and all 22 delta scenarios (8 ux-patterns, 6 tool-adapter, 5 config-system, 3 command-interface) are present exactly once. Requirements not in the deltas are preserved. The ux-patterns merge is confirmed idempotent: `grep -c "localized (en/es)"` = 1 (only the `(Previously: ...)` note references it; the requirement body and TTY-prompt scenario are English-only) and the Self-Update Confirmation Prompt heading count = 1.

## Verification Summary

Source: `verify-report.md` (verdict PASS, reviewed HEAD `f49014e`, written 2026-08-17 at verification time).

- **Verdict**: PASS — implementation matches all 4 delta specs (14/14 requirements, 22/22 scenarios compliant with passing covering tests), honors design D1–D10, all 19 apply tasks complete.
- **Runtime**: `go test ./... -count=1 -race` → exit 0, all 9 packages ok, 0 failures; `go build ./...` → exit 0; `gofmt -s -l` clean; `go vet ./...` clean; smoke `scripts/smoke-test.sh` → 23/23 passed.
- **E2E evidence** (verify-report): `upp check` prints "Checking 9/10" with zero "Updating"; `upp update --dry-run` reports pending updates with no "All clean!"; `upp list` header `ID  Name  Status  Version`; `upp --help` grouped Tool/Config/Maintenance with `completion` absent.
- **Domain status**: command-interface PASS (3/3 scenarios), config-system PASS (5/5), tool-adapter PASS (6/6), ux-patterns PASS (8/8).
- **Design conformance**: D1–D10 honored, including two documented deviations: D5 list-table status icon dropped (header honesty, no test pinned the icon) and D7 v-prefix normalization implemented by *adding* `v` (selfupdate.Parse requires the prefix) instead of stripping — direction swap only, test-enforced.

## Residual Notes / Follow-ups (recorded, no action taken)

Recorded per the handoff and apply-progress/verify-report; no source code was modified by this archive phase.

1. **nvm multi-LTS `tail -1` extraction (pre-existing, out of scope)**: noted in the handoff as a pre-existing observation carried from prior review. Not touched in this change; semver comparison scope was comparison-only (D7).
2. **External-parser wording (pre-existing)**: noted in the handoff; no spec change made.
3. **Branch-comment maintainability (pre-existing)**: noted in the handoff; no code change made.
4. **SUGGESTION findings from verify-report (informational, no action needed)**:
   - `internal/security/confirm.go` comments and `internal/cli/import.go:31` still use the word "Interactive" — this is the security risk-matrix *concept* (risk-level naming), NOT the dead config field. Confirmed pre-existing, no code change warranted.
   - `upp --help` renders an `Additional Commands:` section containing only cobra's builtin `help` (default empty GroupID). No spec requirement covers this; out of scope.
   - nvm phantom-downgrade live check (`v26.7.0 → v24.19.0`) now correctly `false` — verified by `TestCheck/nvm/newer-current-no-downgrade`.
5. **Documented deviations (candidate-caused, non-blocking)**: S1 list-table icon drop (D5), S2 v-prefix normalization direction (D7) — both covered by passing tests; record, no remediation.

All findings are class **pre-existing** or **candidate-caused but non-blocking** with disposition "record, no remediation". **CRITICAL: None. WARNING: None.**

## Artifacts

- Merged canonical specs:
  - `openspec/specs/ux-patterns/spec.md` (clean/idempotent merge)
  - `openspec/specs/command-interface/spec.md` (clean merge)
  - `openspec/specs/tool-adapter/spec.md` (clean merge)
  - `openspec/specs/config-system/spec.md` (clean merge)
- `openspec/changes/upp-simplify-usage/archive-report.md` (this file)
- `openspec/changes/upp-simplify-usage/tasks.md` (task 3.1 marked `[x]`)

**No source code was modified** in this archive phase. Delivery of the canonical spec changes (review + commit) is handled by the orchestrator per repo policy; this phase did not commit, branch, or open a PR.

## Intentional Archive Actions

No destructive deltas were encountered (no REMOVED/RENAMED requirements, no large-section removals). The ux-patterns merge was idempotent with the S1 direct canonical fix — no duplicate or conflicting requirement text remains. No intentional-partial or stale-checkbox-reconciliation overrides were needed.
