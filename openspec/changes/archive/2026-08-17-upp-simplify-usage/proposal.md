# Proposal: Simplify upp Usage — Honest Output and Zero-Config Story

## Intent

Make upp trustworthy for beginners: read-only commands must never claim to update, summaries must never claim "all clean" or "all up to date" when tools were skipped or updates are pending, and README must document the zero-config reality that `init` is optional. Fix the nvm phantom-update bug (semver comparison) so `check` output and gated `update` runs only report real updates.

## Scope

### Slice 1 — Presentation/UX (PR #1)
- Per-operation progress label: "Checking X/Y" for `check`/bare `upp`; "Updating X/Y" only for `update` (render.go:223).
- Dry-run summary wording: no "All clean!" when updates are pending (render.go:266-272).
- `list` table: real ID column + correct header mapping (render.go:348-364).
- Help: cobra `CommandGroups` grouping, hide `completion` (parser.go).
- README zero-config quickstart (README.md:55-72); ux-patterns spec drift fix (:102 "localized (en/es)" → English-only).

### Slice 2 — Data quality (PR #2)
- `check` summary counts skipped tools honestly: "N up to date, M skipped" — never "All tools up to date." when enabled tools were skipped/unchecked (render.go:300-343).
- nvm update detection via semver comparison, reusing `internal/selfupdate/version.go` (nvm.go:66) — no phantom downgrade "updates".

### Out of Scope
- Command removals/renames (bare `upp`==`check`, `self-update` naming settled).
- Security model, policy gating, English-only decision (PR #54/#56) — except the :102 spec-text fix.
- `init` ask-before-redetect optimization, `install.sh`, unknown-tool warn-vs-error alignment (config.go:175-177, deferred).

## Business Rules

1. Keep both commands: `check` = state (read-only), `update` = action; wording MUST make the distinction explicit.
2. Summary MUST count explicitly (up to date / skipped / failed); "All tools up to date." MUST NOT print when any enabled tool was skipped/unchecked.
3. Output strings are not a contract: tests update together with text (Strict TDD: spec deltas first).
4. Both slices this cycle, delivered as 2 chained PRs.

## Current-State Gap

Probe-verified: `check` prints "Updating 1/10"; dry-run prints "✅ … All clean!" with pending updates; skipped tools vanish from counts; `list` headers mislabel data; nvm compares `v26.7.0 != v24.19.0` as "update available"; README presents `init` as required although zero-config works.

## Approach

- **Slice 1**: presentation-only; each fix independently revertible; spec deltas (ux-patterns, command-interface) + hermetic test updates first.
- **Slice 2**: summary-honesty counting + nvm semver compare (narrow scope: comparison only, not version parsing). Slice 2 must not touch slice 1 strings.

## Capabilities

> Contract with sdd-spec. Researched `openspec/specs/`.

### New Capabilities
- None

### Modified Capabilities
- `ux-patterns`: Summary Report (explicit skipped counts; no "All clean!" with pending updates); Progress Indication (check vs update wording); Self-Update Confirmation Prompt (remove stale "localized (en/es)").
- `command-interface`: `--help` grouping + `completion` hidden.
- `tool-adapter`: Version Comparison (nvm MUST use semver compare; no update when current > latest).
- `config-system`: only if Decision Gap 1 resolves to remove `settings.interactive` (conditional).

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| internal/output/render.go | Modified | Progress label, summaries, list table |
| internal/cli/check.go, update.go, parser.go | Modified | Call sites, help groups |
| internal/adapters/official/nvm.go | Modified | Semver-based update detection |
| internal/selfupdate/version.go | Reused | Existing version parse/compare |
| openspec/specs/ux-patterns, command-interface, tool-adapter | Modified | Delta specs (strict TDD) |
| README.md | Modified | Zero-config quickstart |
| internal/**/*_test.go, scripts/smoke-test.sh | Modified | Output-text assertions |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Text changes ripple into hermetic tests/smoke | High | Spec deltas + tests first; mechanical |
| nvm parse gap (e.g., `v26.7.0`) | Med | Scope to comparison only, reuse proven parser |
| Users scripted around "All clean"/"All up to date" | Low | Note in changelog; wording stays grep-stable ("clean" removed only when pending) |
| Slice 2 touches slice 1 strings | Med | Explicit string-ownership boundary in tasks |

## Rollback Plan

Each fix is a small independent commit; revert per-item. Both PRs target the feature branch chain — revert = `git revert` of the slice PR; no schema/migration involved. Spec deltas revert with the code.

## Dependencies

- Exploration: `openspec/changes/upp-simplify-usage/exploration.md` (done).
- Slice 2 depends on slice 1 merge (chained PRs, review budget 400 lines).

## Success Criteria

- [ ] `upp check` prints "Checking X/Y" and never "Updating X/Y"; dry-run never pairs "All clean!" with pending updates.
- [ ] `check` with skipped tools prints explicit counts incl. "M skipped"; never "All tools up to date." with skips.
- [ ] `list` table shows tool IDs usable with `--only`/`--skip`.
- [ ] nvm on newer node reports no update; semver tests green.
- [ ] README quickstart works from scratch without `init`.
- [ ] `go test ./... -count=1` and smoke script pass.

## Decision Gaps (name explicitly)

1. `settings.interactive` (dead setting, config.go:26): **remove** (recommended — prompts are risk-matrix-driven) or wire as prompt toggle. Default to remove; confirm before sdd-spec writes the config-system delta.
2. `check` vs `update --dry-run`: docs-only clarification — RESOLVED by Business Rule 1 (no convergence).
3. Slice split: 2 chained PRs — RESOLVED (Business Rule 4).

## Non-Goals

Security model, policy-driven update gating, English-only output (PR #54/#56), RDD delivery, config schema v1, command removal/renames, `init` wizard redesign.
