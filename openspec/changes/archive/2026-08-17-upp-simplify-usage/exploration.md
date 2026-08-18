# Exploration: upp-simplify-usage

Date: 2026-08-16 · Change: upp-simplify-usage · Phase: explore (read-only)
Evidence: code reading + live binary probes (built 2026-08-16, `HOME` isolated, no config and crafted configs) + archived SDD decisions. No source files modified.

## Current State

### Command surface (verified by running the built binary)

| Command | Purpose | Defined in | Interactive | Modifies system |
|---|---|---|---|---|
| `upp` (no args) | Read-only status check (same as `check`) | internal/cli/parser.go:44 | No | No |
| `check` | Query enabled tools for updates | internal/cli/check.go | No | No |
| `update` (`--dry-run`) | Apply updates to enabled tools | internal/cli/update.go | Yes | Yes |
| `init` | First-run wizard, generates config.toml | internal/cli/init.go | Yes | Yes (config) |
| `list` | Table of detected tools + status | internal/cli/list.go | No | No |
| `export` (`-o`) | Export config as TOML | internal/cli/export.go | No | No |
| `import <file>` | Replace config from TOML (confirm) | internal/cli/import.go | Yes | Yes (config) |
| `self-update` | Update the upp binary itself | internal/cli/selfupdate.go | Yes (confirm) | Yes (binary) |
| `completion`, `help` | Cobra built-ins (shown in `--help`) | cobra | No | No |

Global flags: `--quiet`, `--ci`, `--only`, `--skip` (parser.go:49-52). Command-specific: `update --dry-run`, `export -o`.

- The bare `upp` == `check` duality is intentional (spec command-interface "No args" scenario) and documented in README. Not a defect; the naming overlap with `update` is the only real ambiguity and it is already resolved by `self-update` (decision from sdd/upp-versioning-auto-update, archived).
- `--help` output lists `completion` and `help` among the 7 real commands — noise for a beginner, and the command list is flat: no grouping separates "tools" commands (`check`/`update`/`list`) from "config" commands (`init`/`export`/`import`) from `self-update`.
- `check` vs `update --dry-run` overlap: both run check() on every tool; `check` reports availability, `--dry-run` reports planned actions. Functionally near-identical; the difference is not explained anywhere in help text.

### Config (internal/config)

- TOML at `~/.config/upp/config.toml` (`%APPDATA%/upp/` on Windows), schema v1: `[settings]` (interactive, check_self_update), `[tools.*]`, `[custom.*]` (config.go:17-44).
- Zero-config behavior already works: no file → `Load()` returns defaults (config.go:114-115) → `buildAdapterList` runs ALL platform adapters (check.go:172-179: a tool is included unless config explicitly disables it). Verified by probe: bare `upp` on a fresh `HOME` checked all 10 Linux tools. **`init` is optional today** — but neither help text nor README quickstart says so; README still presents `upp init` as the required first step.
- `settings.interactive = true` is written but read nowhere in the codebase (grep: no reader) — dead setting that suggests configurability that does not exist. `check_self_update` is real (opt-in hint, default off).
- Validation is strict in odd places: an unknown enabled tool id is a hard error (config.go:175-177), while the spec says warn-and-ignore; platform-mismatch tools are silently disabled (config.go:199-204) with no warning surfaced to the user.

### Output / UX (internal/output + internal/cli) — probe findings

All probed on a piped (non-TTY) run, so emojis are plain `[status]` tags; TTY adds ✅/⏭️/❌/⬆️/✔️.

1. **`check` prints "Updating X/Y" progress** (render.go:223 hardcodes `"  %s Updating %d/%d: %s\n"`; callers check.go:88, update.go:94). Verified: `upp check` on a fresh HOME printed `⟳ Updating 1/10: APT Package Manager ...` — read-only command claiming it is updating. The single most visible wording bug.
2. **`check` summary silently drops skipped tools** (render.go:300-343 CheckSummary counts only available/current/failed). Probe with a custom tool whose binary is missing: `[available] 1 available, 9 current` — 11 adapters existed, the custom one was skipped (not installed) and vanished from the counts; with all tools skipped it would print `All tools up to date.` despite checking nothing. Beginner can't tell which tools were actually checked.
3. **Dry-run summary contradicts itself** (render.go:266-272): `[updated] 1 would update. All clean!` — "✅ ... All clean!" alongside a pending update (icon/label mismatch; "would update" counted as `updated`).
4. **`list` table columns are mislabeled** (render.go:348-364): header `Tool | Name | Status | Version`, but the data rows place the status icon under "Tool" and the display name ("APT Package Manager") under "Name". The IDs that `--only`/`--skip`/config actually use (`apt`) never appear, so the user cannot map table rows to filter names.
5. **nvm phantom update: `Node Version Manager v26.7.0 → v24.19.0`** (internal/adapters/official/nvm.go:66): `updateAvailable := current != "unknown" && latest != "unknown" && current != latest` — pure string inequality, no semver comparison, and it compares the current node (non-LTS) against the latest LTS line. A user on a NEWER node version is told an "update" to an OLDER one is available, and `upp update` (PolicyGated) would actually run `nvm install stable`. Check output is the tool's primary surface; this makes it untrustworthy.
6. **`--quiet` output keeps icons/colors and error details** (render.go:198-214) — spec says quiet = one line per tool, which is roughly honored; not a defect, but `quietToolLine` still prefixes `[skipped]`/`[failed]` tags — acceptable.
7. **security prompts bypass the renderer** (internal/security/confirm.go:98, 109-113 use `fmt.Printf` directly): raw text, no color/emoji consistency with the rest of the CLI; `--quiet` doesn't affect them (correct per spec, but stylistically inconsistent).
8. **self-update Long help is jargon-heavy**: "verify its sha256 checksum" (selfupdate.go:40-41) — fine for power users, noise for beginners; also `--only/--skip ignored` note is long.

### Onboarding

- README quickstart (README.md:55-72) presents a 4-step flow (`init` → `list` → `check` → `update`). Verified the CLI needs none of it: `upp update` works on a fresh machine (all tools enabled by default). The README contradicts actual behavior; the "Zero config" story is undocumented.
- First-run `upp init` (init.go:27-95) works post-audit: detects, generates config, and on a second run re-detects everything BEFORE asking "Config already exists. Overwrite with new detection?" (init.go:41-65) — wasted work and the prompt doesn't tell the user what changed (e.g., "2 new tools detected").
- `curl | bash` install (scripts/install.sh) is a separate shell recipe; not part of this change.

### Existing simplicity signals (prior decisions to respect)

- Spec command-interface: bare `upp` == check; `self-update` must not collide with `update`; `--only` wins over `--skip`; unknown filter names warn-and-ignore. Do not rename/merge commands without a spec delta and user confirmation.
- PR #54/#56 (commits a139ef6, caf39b2, 2026-08-15): **output is English-only**; `settings.language` was dropped. Respect fully — no language work.
- **Spec drift found**: openspec/specs/ux-patterns/spec.md:102 (Self-Update Confirmation Prompt) still says "localized (en/es)" — stale text from before the language drop; the implementation is English-only. Should be corrected by this change (spec hygiene, not re-adding languages).
- Archived changes: upp-evolution (2026-08-09) chose cobra + flat commands; upp-versioning-auto-update (2026-08-12) fixed the `self-update` naming; upp-hermetic-cli-tests (2026-08-14) made CLI tests hermetic — output-text changes must keep those tests (and the smoke script) in mind.
- Config-system spec D6: missing config = defaults, existing-but-empty = catalog defaults, first-run = explicit file existence. Preserve.

## Affected Areas

| Area | Why |
|---|---|
| internal/output/render.go | Progress label (223), CheckSummary counts (300-343), dry-run summary wording (241-272), list table (348-364) |
| internal/cli/check.go | check summary + skipped-tool honesty; progress call site (88) |
| internal/cli/update.go | progress call site (94); summary reuse |
| internal/cli/parser.go | help grouping (cobra CommandGroups), hide `completion`, "no config needed" hint |
| internal/cli/init.go | ask-before-redetect (41-65) |
| internal/adapters/official/nvm.go | phantom-update detection (66) — semver compare (internal/selfupdate/version.go already has parsing/compare logic to reuse) |
| internal/config/config.go | dead `interactive` setting (26) — remove or wire; unknown-tool warn-vs-error (175-177) |
| openspec/specs/ux-patterns/spec.md | spec drift at :102; Summary Report scenarios if wording changes |
| README.md | zero-config quickstart; list/flags tables |
| internal/**/*_test.go, scripts/smoke-test.sh | output-text assertions will need updating (strict TDD) |

## Approaches

1. **A — Presentation-only polish (no behavior change)**
   - Progress label "Checking X/Y" vs "Updating X/Y" per operation; dry-run summary wording/icon; `list` table (ID column + correct header); help-text grouping via cobra CommandGroups + hide `completion`; fix ux-patterns spec drift; README zero-config quickstart.
   - Pros: pure UX, no security/config semantics touched, smallest review; every item independently revertible.
   - Cons: none structural; output-text changes ripple into tests/smoke (mechanical).
   - Effort: Low–Medium.

2. **B — A + data-quality fixes**
   - Adds: `check` summary counts skipped tools honestly (no more silent "All tools up to date."); nvm semver-based update detection (reuse internal/selfupdate version compare); drop or wire the dead `interactive` setting.
   - Pros: output becomes trustworthy — the core of "simple to use" is "says the truth"; adapters change is small and test-covered.
   - Cons: nvm behavior change (no more phantom updates → fewer `update` runs); needs spec deltas (ux-patterns Summary/check scenarios, config-system settings) before code per strict TDD.
   - Effort: Medium.

3. **C — Surface reduction (merge/rename commands, remove init-as-required)**
   - e.g., drop `check` in favor of bare `upp`+`update --dry-run`; make `init` purely optional and self-hint; group or hide export/import.
   - Pros: smallest possible command surface.
   - Cons: removes documented commands (spec command-interface deltas + user decision required); scripts/smoke/tests churn; `check` is the one command that is unambiguous and cheap — deleting it saves little and risks confusing existing users.
   - Effort: Medium–High.

## Recommendation

**Approach B, delivered as two review slices** (per the 400-line review budget): slice 1 = A (presentation + spec drift + README), slice 2 = B-only items (summary honesty, nvm semver, dead setting). Rationale: the biggest beginner blockers found in probes are *misleading output* (progress label, dry-run "All clean!", phantom nvm update, silent skips) — not command count. C is not recommended now: `check`/`self-update` naming was already settled by prior decisions, and removing commands is higher churn for marginal gain; keep C items (hide `completion`, group help) inside slice 1 instead.

## Non-Goals (must NOT change)

- Security model: trust levels, risk classification, fail-closed zero values, `--ci` deny semantics, official no-prompt rule (security-model spec).
- English-only output (PR #54/#56). The ux-patterns "en/es" drift is a spec-text fix only.
- Policy-driven update gating (adapter UpdatePolicy, 2026-08-15 change) — behavior stays.
- RDD delivery, commit conventions, review budget, config schema v1 and file locations.
- No command removals/renames without explicit user confirmation.

## Risks

- Output-text changes touch many hermetic tests + smoke greps; strict TDD means spec deltas and tests come first — small but real scope creep in slice 1.
- nvm semver change could surface a real adapter parsing gap (e.g., "v26.7.0" vs "v24.19.0" parse) — keep the change narrowly scoped to comparison, not version-string parsing.
- "Count skipped in check" changes the meaning of existing ux-patterns scenarios ("All tools up to date." text) — must be a deliberate spec delta, and could annoy users who want the terse message.
- Removing/rewording summary strings ("All clean!") is cosmetically visible; users may have scripted around `grep "All clean"` — low likelihood, note in proposal.

## Open Questions for Proposal

1. `settings.interactive` — wire it (prompt toggle) or remove it? (Recommend remove: prompts are already risk-matrix-driven; the setting is misleading.)
2. `check` vs `update --dry-run` — clarify docs only, or converge behavior?
3. Slice split for delivery — confirm two PRs (A then B) fits the review budget.

## Ready for Proposal

Yes. The exploration found concrete, evidence-backed simplification opportunities (probe-verified), all within presentation/data-quality scope, with the security model, English-only, update gating, and RDD untouched.
