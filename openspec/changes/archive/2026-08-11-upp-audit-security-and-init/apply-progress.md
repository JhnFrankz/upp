# Apply Progress — upp-audit-security-and-init (Cumulative: Batch 1 + Batch 2 + Batch 3 + Batch 4)

**Change**: upp-audit-security-and-init
**Mode**: Strict TDD
**Slices**: Batch 1 = Phase 1, tasks 1.1–1.5 (PR 1, work unit 1). Batch 2 = Phase 2 core, tasks 2.1–2.4, 2.7, 2.8 (PR 2, work unit 2). Batch 3 = Phase 2 remaining + amendment, tasks 2.5, 2.6, 2.9–2.13 (PR 3, work unit 3). Batch 4 = Phase 3 + Phase 4 finalization, tasks 3.1–3.4, 4.1–4.4 (rides on PR #3 delivery, work unit 4). No commits/branches created — delivery gates are orchestrator-owned; changes left in working tree.
**Delivery**: auto-chain · stacked-to-main (PR #1 targets main; PR #2 stacks after it; PR #3 stacks after PR #1+#2).

## Batch 1 (PR 1 — Foundation) — TDD Cycle Evidence

| Task | Test File | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|------|-----------|-------|------------|-----|-------|-------------|----------|
| 1.1 | internal/adapters/custom_test.go | Unit | ✅ 24/24 (adapters pkg) | ✅ Written — build fails: undefined TrustCustomTrusted/TrustCustomUntrusted/TrustOfficial, ToolInfo.Command/Privileges | ✅ Passed (via 1.2+1.3) | ✅ 2 cases (trusted sudo→[sudo]; untrusted plain→[]) | ➖ N/A (RED task) |
| 1.2 | internal/adapters/interface.go (no own test file; verified by 1.1 tests) | Unit | ✅ above | ✅ Written (1.1) | ✅ Passed at 1.3 gate — interlock: 1.2 removes constants still referenced by custom.go, so GREEN confirmed once 1.3 compiles; test file compiled clean after 1.2 alone | ➖ Single — structural type change, one output set | ➖ None needed |
| 1.3 | internal/adapters/custom_test.go | Unit | ✅ above | ✅ Written (1.1) | ✅ Passed — go test -run TestCustomAdapter_Info: 2/2 PASS | ✅ 2 paths: trusted=true→TrustCustomTrusted+[sudo]; untrusted→TrustCustomUntrusted+[] | ➖ None needed |
| 1.4 | official/{info,registry,adapter_update}_test.go (approval) | Unit | ✅ 24/24 | ✅ Approval tests renamed first (pin TrustOfficial) | ✅ Passed after rename — all official tests green | ➖ Single — mechanical rename | ✅ Rename across 12 adapters + 3 official test files + cli/update.go + cli/integration_test.go; tests green after EACH refactor step |
| 1.5 | Gate run | — | — | — | ✅ go test ./internal/adapters/... -count=1 → ok (both packages), exit 0 | — | — |

**Interlock note (1.2 GREEN)**: task 1.2 removes TrustTrusted/TrustUntrusted which custom.go still references, so the package cannot compile until 1.3 lands. 1.2's GREEN gate is therefore confirmed by the focused test run after 1.3 (production code only — test file itself compiled immediately after 1.2). No behavior was implemented in 1.2 beyond the type-level contract.

## Batch 2 (PR 2 — Security Matrix) — TDD Cycle Evidence

| Task | Test File | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|------|-----------|-------|------------|-----|-------|-------------|----------|
| 2.1 | internal/security/security_expanded_test.go | Unit | ✅ 39/39 (security pkg, baseline at batch-2 start) | ✅ Written (pre-existing pinned tests) — all threat-matrix cases already present: sudo (HighRiskEdgeCases/CaseInsensitive), rm -rf (nested/root/uppercase), curl\|sh (bare/bash/-fsSL/wget + HighRiskKeywords_AllCovered), && chains (MediumRiskEdgeCases: &&, \|\|, ;) | ✅ Passed at baseline run | ✅ Multi-case tables per class (9 high / 9 medium / 9 low) | ➖ None needed — no production classifier change in slice |
| 2.2 | internal/security/confirm_test.go + security_expanded_test.go | Unit | ✅ above | ✅ Written — compile fails: ConfirmConfig.TrustLevel is string; adapters.TrustLevel not assignable; Trusted field removed in tests (16 rows affected) | ✅ Passed at 2.7 gate | ✅ Full 18-row DecisionMatrix incl. D4 rows (untrusted CI low→Auto) — caught real bug: CI low returned Proceed instead of Auto (fixed in GREEN) | ➖ None needed |
| 2.3 | internal/cli/audit_probe_test.go | Integration | ✅ 51/51 (cli pkg baseline) | ✅ Converted 3 probes (i18n probe dropped per design): sudo trusted --ci, rm -rf interactive trusted, rm -rf interactive untrusted — RED run: 2/3 FAIL (interactive bypasses EXECUTE under old code, marker written) | ✅ Passed at 2.8 gate | ✅ 3 cases: trusted CI / trusted interactive / untrusted interactive | ➖ None needed |
| 2.4 | internal/cli/audit_probe_test.go | Integration | ✅ above | ✅ 2 new probes: trusted low-risk executes (correct-pass — green at RED run by design); --quiet medium-risk still prompts (FAIL at RED run: executed + no prompt) | ✅ Passed at 2.8 gate | ✅ 2 cases: low-risk success vs medium-risk prompt-under-quiet | ➖ None needed |
| 2.7 | internal/security/confirm.go | Unit (via 2.2 tests) | ✅ above | ✅ Written (2.2) | ✅ Passed — first GREEN attempt failed 1 row (trusted CI low got ConfirmProceed, want ConfirmAuto); fixed: CI low returns Auto before info print; 37/37 PASS | ✅ DecisionMatrix 18 rows + edge cases (empty tool/command, privileges, no-privileges, uppercase/full-word inputs) | ✅ Risk-before-trust switch (Low/Medium/High) extracted; printInfo helper extracted; promptUser unchanged |
| 2.8 | internal/cli/update.go | Integration (via 2.3/2.4 probes) | ✅ above | ✅ Written (2.3/2.4) | ✅ Passed — ClassifyCommand(info.Command) real command; TrustLevel: info.Trust typed; Privileges passed; trustLevelString deleted; TestTrustLevelString removed; TestCIMode_RejectsUntrustedCustomTools command converted low→medium (D4: CI low auto now); 55/55 cli PASS incl. 5/5 probes | ✅ Probes 1–5 all green; TestCIMode_RejectsUntrustedCustomTools pins CI medium-untrusted error | ✅ gofmt -s, go vet clean; go build ./... clean |

**2.1 evidence note**: the classifier already existed with full threat-matrix test coverage (audit-era tests), and the audit root cause was upstream (update.go passed `info.Name + " update"` instead of the real command), not the classifier itself. 2.1 therefore required no new test content — verified all planned cases are pinned; recorded as RED-satisfied by existing coverage.

**2.2 RED→GREEN detail**: the 16 ConfirmConfig usages across confirm_test.go + security_expanded_test.go were rewritten to `adapters.TrustLevel` (TrustOfficial/TrustCustomTrusted/TrustCustomUntrusted), `Trusted bool` removed everywhere, and D4 expectations applied: untrusted CI Low changed ConfirmError→ConfirmAuto (confirm_test.go table + DecisionMatrix + TestConfirmAction_CustomUntrusted_CI_AllRisks). The old matrix pinned "untrusted CI low → error"; D4 + ux-patterns delta ("`--ci` low-risk run → auto-proceed") mandate the change.

## Batch 3 (PR 3 — Init Wizard + Config Load States + Compact-Pipe Amendment) — TDD Cycle Evidence

| Task | Test File | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|------|-----------|-------|------------|-----|-------|-------------|----------|
| 2.5 | internal/cli/init_probe_test.go | Integration | ✅ 55/55 cli + config/security green (baseline before batch 3) | ✅ Written — 3 probes; RED run 3/3 FAIL: first-run wizard never ran (no config created, "Config already exists. Use --ci to overwrite." emitted), no overwrite prompt shown | ✅ Passed at 2.10 gate — 3/3 PASS via root.Execute | ✅ 3 probes: missing→wizard creates / existing→prompt+preserve (byte-identical) / existing+"y"→regenerates (old settings gone) | ➖ None needed |
| 2.6 | internal/config/config_test.go | Unit | ✅ above | ✅ Written — compile fails: undefined Exists (2 refs); then with Exists() added only: TestLoadEmptyFile_AppliesDefaults + TestLoadPartialConfig_CatalogDefaults FAIL (no ApplyDefaults in Load); missing→no-defaults and full→as-is pins PASS (spec pins) | ✅ Passed at 2.9 gate | ✅ 5 distinct Load states: missing (0 tools), empty (catalog + en), partial (es preserved + catalog), full (as-is, no additions), Exists true/false | ➖ None needed |
| 2.9 | internal/config/config.go | Unit (via 2.6 tests) | ✅ above | ✅ Written (2.6) | ✅ Passed — Exists() (os.Stat on ConfigPath) + ApplyDefaults(cfg) inside Load after Validate, existing files only; full config pkg green | ✅ above (5 states) | ✅ Minimal diff; docstring kept accurate; ApplyDefaults stays in defaults.go (no import churn) |
| 2.10 | internal/cli/init.go | Integration (via 2.5 probes) | ✅ above | ✅ Written (2.5) | ✅ Passed — Exists() gate replaces `Version > 0` inference; dead prompt branch restored and merged with the gate (single prompt, delete early-return warning + dead second prompt); single config.Load() (lang only); CI overwrite path unchanged | ✅ Probes 3/3 + pre-existing init tests (TestInitCommand_CI_Mode/DetectsTools/AlreadyExists, TestInitCheckUpdateLifecycle) all green | ✅ Duplicate Load removed; dead branch deleted; prompt logic lives in exactly one place |
| 2.11 | Gate run | — | — | — | ✅ `go test ./internal/security/... ./internal/config/... ./internal/cli/... -count=1` → ok ×3, exit 0 | — | — |
| 2.12 | internal/security/security_expanded_test.go | Unit | ✅ above | ✅ Written — 3 rows added to TestHasPipeToShell_EdgeCases (compact `\|sh`/`\|bash`) + 3 rows to TestClassifyCommand_HighRiskEdgeCases; RED run: all 6 FAIL (compact pipes classified Low / hasPipeToShell=false) | ✅ Passed at 2.13 gate | ✅ 6 compact cases (curl\|sh, curl\|bash, wget\|sh at both detection levels) + all pre-existing spaced-pipe/URL/double-pipe rows still green (spaced behavior preserved) | ➖ None needed |
| 2.13 | internal/security/trust.go | Unit (via 2.12 tests) | ✅ above | ✅ Written (2.12) | ✅ Passed — hasPipeToShell adds compact `\|sh` and `\|bash` substrings alongside spaced `\| sh`/`\| bash` (case-insensitive); full security pkg green | ✅ above (6 cases) | ➖ None needed — 2-line addition, existing structure kept |

**2.6 RED note**: the Exists() tests were RED by compile failure (undefined symbol — strict-tdd: test references non-existent production code). To capture the Load-state tests' RED explicitly, Exists() was added first (its own GREEN), then the 5 Load-state tests were run against the still-unchanged Load(): empty-file and partial-config FAILED (no ApplyDefaults), missing-file and full-config pins passed as designed. Then ApplyDefaults-in-Load landed (2.9 GREEN).

## Batch 4 (PR 3 finalization — Phase 3 Tests + Phase 4 Verification) — TDD Cycle Evidence

| Task | Test File | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|------|-----------|-------|------------|-----|-------|-------------|----------|
| 3.1 | official/{info,registry,adapter_update}_test.go + custom_test.go | Unit | ✅ full suite green at batch-4 start (adapters 0.071s, official 0.047s, security 0.009s, config 0.009s, cli 31.6s) | ➖ N/A — work already present from PR1 | ✅ Verified — TrustOfficial in all 12 golden info rows + registry/all-adapters + adapter_update consistency; custom_test.go pins TrustCustomTrusted/Untrusted + Command/Privileges | ➖ N/A — mechanical rename already applied in B1 | ✅ One residue fixed: stale comment "Trust must be Trusted" → "Trust must be TrustOfficial" (adapter_update_test.go:152); gofmt clean after |
| 3.2 | confirm_test.go + security_expanded_test.go | Unit | ✅ above | ➖ N/A — work already present from PR2 | ✅ Verified — ConfirmConfig.TrustLevel is typed adapters.TrustLevel everywhere; Trusted bool gone; DecisionMatrix 18 rows incl. D4 ("untrusted CI low" → ConfirmAuto, line 552) | ➖ N/A — D4 rows already landed in 2.2/2.7 | ➖ None needed |
| 3.3 | config_test.go | Unit | ✅ above | ➖ N/A — work already present from PR3 | ✅ Verified — TestExists_MissingFile/ExistingFile; TestLoadMissingFile_NoDefaults (0 tools); TestLoadEmptyFile_AppliesDefaults; TestLoadPartialConfig_CatalogDefaults; TestLoadFullConfig_AsIs | ➖ N/A — 5 states already landed in 2.6/2.9 | ➖ None needed |
| 3.4 | internal/cli/probe_test.go (new), audit_probe_test.go, init_probe_test.go | Integration | ✅ cli 58 tests green before refactor | ✅ Approval: baseline captured before move (58/58 + all packages green) | ✅ Passed — after extraction `go test ./internal/cli/ ./internal/adapters/official/ -count=1` → ok (cli 30.5s), exit 0; first attempt failed build (unused tmpDir in overwrite probe — fixed by using probeHome(t) without assignment) | ✅ 8 probes (5 audit + 3 init) + all pre-existing cli tests still green after the move | ✅ probeSetup + new probeHome extracted to shared probe_test.go; init probes' 3× duplicated tmpDir/Setenv blocks replaced with probeHome(t); i18n probe remnants: NONE existed (grep i18n/TrustTrusted/trustLevelString = 0 matches) — verified and noted, nothing to drop |

## Batch 4 — Test Summary

- Tests added: 0 new (3.1–3.3 verified as already complete; 3.4 pure helper extraction, 0 behavior change)
- Tests in cli pkg: 58 (unchanged — probe_test.go hosts shared helpers only)
- Layers used: none new (Unit + Integration from prior batches)
- Approval tests: the 58-test cli baseline + full suite served as approval for the 3.4 move
- Pure functions: 0 new (probeHome is a 3-line t.Helper wrapper)

## Work Unit Evidence (Batch 4 — PR 3, work unit 4)

| Evidence | Required value |
|---|---|
| Focused test command and exact result | `go test ./... -count=1` → all 7 test packages `ok`, exit 0 (cli 32.9s); `go test ./... -race` → all `ok`, exit 0 (cli 33.5s); `go vet ./...` → clean, exit 0; `gofmt -s -w .` then `gofmt -s -l .` → EMPTY (clean), exit 0 |
| Runtime harness | `bash scripts/smoke-test.sh` (built fresh — root binary was stale from 11:59, predating batches 1–3; then re-run `--skip-build` after the fresh build) → **23 passed, 0 failed, 23 total, exit 0**. New D6 assertions: export with empty config → contains `tools.apt` (catalog defaults); partial config → `tools.apt` + `language = "es"` preserved; NO config → export output does NOT contain `tools.apt` (first-run path, no catalog defaults). First run FAILED 1/22 (test 14 asserted `upp list` → "No tools configured." — wrong seam: buildAdapterList adds all catalog tools when cfg.Tools has no entry, so list output is identical for missing vs empty config); fixed by asserting via `upp export` output, adding a `run_test_without_output` helper for the negative no-config case. |
| Rollback boundary | Revert internal/cli/probe_test.go (delete) + revert audit_probe_test.go/init_probe_test.go helper hunks; revert scripts/smoke-test.sh tests 12–14 + run_test_without_output helper. Test-file updates revert with their production counterparts (no production file changed in this batch — pure test/script finalization). No new production code in batch 4. |

## Batch 3 — Test Summary

- Tests added: 3 integration probes (cli) + 9 unit tests (config: TestExists ×2, TestLoad* ×4 states + TestLoadMissingFile_NoDefaults pin) + 6 compact-pipe table rows (security, 2 tables)
- Tests in cli pkg: 58 (was 55); config pkg: unchanged function count +9 new; security pkg: unchanged count, +6 table rows
- Full suite `go test ./... -count=1` → all packages ok, exit 0
- Layers used: Unit (config states, security classifier) + Integration (3 init probes via root.Execute with real CLI + stdin/stdout capture)
- Approval tests: none needed (no refactor of existing behavior — new behavior only)
- Pure functions: 0 new (Exists() is a trivial os.Stat wrapper; hasPipeToShell extended in place)

## Work Unit Evidence (Batch 3 — PR 3, work unit 3)

| Evidence | Required value |
|---|---|
| Focused test command and exact result | `go test ./internal/security/... ./internal/config/... ./internal/cli/... -count=1` → `ok github.com/JhnFrankz/upp/internal/security`, `ok .../internal/config`, `ok .../internal/cli`, exit 0. Per-package GREEN runs: config pkg `ok` (0.009s) after 2.9; cli pkg `ok` (27.5s, 58 tests) after 2.10; security pkg `ok` after 2.13. |
| Runtime harness | Init wizard probes via root.Execute with t.TempDir HOME + temp-file os.Stdin replacement — `go test ./internal/cli/ -run TestInitProbe -count=1 -v` → 3/3 PASS: TestInitProbe_MissingConfig_WizardCreates (config file created at $HOME/.config/upp/config.toml, "Config written to" in output), TestInitProbe_ExistingConfig_PromptsAndPreserves (output contains "Overwrite with new detection?", file byte-identical after "n"), TestInitProbe_ExistingConfig_ConfirmedOverwrites ("y" → "Config written to", loaded language no longer "es"). At RED run: 3/3 FAILED (wizard never ran; prompt never shown — dead first-run path). |
| Rollback boundary | Revert internal/config/config.go (Exists + ApplyDefaults-in-Load) + internal/config/config_test.go; revert internal/cli/init.go + internal/cli/init_probe_test.go — each pair reverts independently (config change breaks init probes; init change alone breaks nothing else). SEPARATELY, the amendment reverts independently: internal/security/trust.go + the 6 compact rows in internal/security/security_expanded_test.go. No smoke-test.sh, no Phase 3/4 files touched. |

## Files Changed (cumulative all batches)

| File | Action | What Was Done |
|------|--------|---------------|
| internal/adapters/interface.go | Modified | B1: 3-tier TrustLevel enum (TrustOfficial, TrustCustomTrusted, TrustCustomUntrusted); ToolInfo +Command +Privileges. B2: TrustLevel.String() method (official/custom-trusted/custom-untrusted) — collateral for typed-trust prompt/info rendering |
| internal/adapters/custom.go | Modified | B1: Info(): trusted→TrustCustomTrusted (never Official); fills Command + Privileges via detectPrivileges |
| internal/adapters/custom_test.go | Modified | B1: Info tests: untrusted→TrustCustomUntrusted; trusted→TrustCustomTrusted + never-Official + Command + Privileges assertions |
| internal/adapters/official/*.go (12) | Modified | B1: mechanical rename adapters.TrustTrusted→adapters.TrustOfficial (apt, brew, bun, docker, gh, go, npm, nvm, opencode, pnpm, scoop, winget) |
| internal/adapters/official/info_test.go | Modified | B1: mechanical rename in golden ToolInfo rows |
| internal/adapters/official/registry_test.go | Modified | B1: mechanical rename + error message text |
| internal/adapters/official/adapter_update_test.go | Modified | B1: mechanical rename + error message text |
| internal/security/confirm.go | Modified | B2: ConfirmConfig.TrustLevel → adapters.TrustLevel (typed); Trusted bool dropped; risk-before-trust switch; D4 CI matrix (Low→Auto any trust; Medium→Auto trusted/Err untrusted; High→Err even trusted); interactive (High→prompt, Medium→info trusted/prompt untrusted, Low→info); printInfo extracted; promptUser unchanged |
| internal/security/trust.go | Modified | B3: hasPipeToShell detects compact `\|sh` / `\|bash` variants in addition to spaced `\| sh` / `\| bash` (case-insensitive) — closes the batch-2 issue #1 (amendment tasks 2.12/2.13, user decision) |
| internal/security/confirm_test.go | Modified | B2: full rewrite — typed trust tables; D4 untrusted-CI-Low→Auto; high/medium/low interactive rows |
| internal/security/security_expanded_test.go | Modified | B2: classifier tests untouched; 14 ConfirmAction tests migrated to typed TrustLevel; DecisionMatrix 18 rows with D4 expectations; TestConfirmAction_CustomUntrusted_CI_AllRisks now low→Auto/med→Err/high→Err. B3: +3 compact rows in TestHasPipeToShell_EdgeCases + 3 in TestClassifyCommand_HighRiskEdgeCases |
| internal/cli/update.go | Modified | B1: mechanical rename (TrustOfficial). B2: ClassifyCommand(info.Command) real command; ConfirmConfig gets TrustLevel: info.Trust, Command: info.Command, Privileges: info.Privileges; CI error msg → "CI mode: custom tool requires confirmation"; trustLevelString deleted |
| internal/cli/init.go | Modified | B3: runInit gates on config.Exists() (D5 — first-run from explicit file existence, never Version-inference); interactive overwrite prompt "Config already exists. Overwrite with new detection? [y/N]" restored as THE single gate branch (dead second prompt + "Use --ci to overwrite" early-return deleted); single config.Load() call; CI overwrite path unchanged |
| internal/cli/audit_probe_test.go | Created | B2: probeSetup (fake sudo/evil-tool/harmless-tool + marker, PATH/HOME isolation, config save) + runUpdateCmd helper; 5 probes: trusted CI high-risk non-zero no-exec, trusted interactive high-risk no-exec, untrusted interactive high-risk no-exec, trusted low-risk executes, --quiet medium-risk prompts (i18n probe dropped per design) |
| internal/cli/init_probe_test.go | Created | B3: runInitCmd helper (root.Execute + temp-file os.Stdin replacement); 3 probes: missing→wizard creates config; existing→overwrite prompt + byte-identical file on deny; existing+"y"→regenerates. B4: 3× duplicated tmpDir/Setenv(HOME) replaced with shared probeHome(t) |
| internal/cli/probe_test.go | Created | B4 (3.4 REFACTOR): shared probe helpers — probeHome(t) (HOME isolation for all CLI probes) + probeSetup(t, tool) (fake sudo/evil-tool/harmless-tool + marker, PATH/HOME isolation, config save), moved verbatim from audit_probe_test.go |
| internal/cli/integration_test.go | Modified | B1: mechanical rename in mockAdapter Trust. B2: TestTrustLevelString removed (function deleted); TestCIMode_RejectsUntrustedCustomTools command "untrusted-tool --update"→"untrusted-tool --update && echo done" (medium risk — D4 CI low auto) |
| internal/config/config.go | Modified | B3: Exists() (os.Stat on ConfigPath, false on any error); Load() calls ApplyDefaults(cfg) after Validate ONLY when the file exists (D6) — missing file returns base config with empty tools |
| internal/config/config_test.go | Modified | B3: TestExists_MissingFile/ExistingFile; TestLoadMissingFile_NoDefaults (pin: 0 tools); TestLoadEmptyFile_AppliesDefaults (catalog + en); TestLoadPartialConfig_CatalogDefaults (es preserved + catalog); TestLoadFullConfig_AsIs (no additions) |
| scripts/smoke-test.sh | Modified | B4 (4.4): +run_test_without_output helper; +tests 12–14 (D6 config load states) — empty config → `upp export` contains tools.apt (catalog defaults); partial config → tools.apt + `language = "es"` preserved; NO config → export does NOT contain tools.apt (first-run path stays). Initial assertion attempt via `upp list` was wrong (buildAdapterList adds catalog tools when cfg.Tools lacks an entry — same output for missing and empty config); export output is the correct observable seam |
| openspec/changes/upp-audit-security-and-init/tasks.md | Modified | Tasks 1.1–1.5, 2.1–2.13, 3.1–3.4, 4.1–4.4 marked [x] (B4 completed Phase 3 + Phase 4) |

## Deviations from Design

None structural. Three documented notes (B1+B2 carried forward, B3, B4 new):

1. **TrustLevel.String() added to adapters/interface.go** (design D2 deletes `trustLevelString()` in cli). The typed enum needs a renderer for %s in confirm.go's info/prompt output — placed on the enum type itself (single source of truth, both packages use it). Mechanical collateral within the rollback boundary.
2. **TestCIMode_RejectsUntrustedCustomTools converted to medium-risk command** instead of updating its expectation to the new D4 low-risk-auto behavior. Preserves the integration-level "CI rejects confirmable custom tools" pin at the correct risk tier.
3. **B3 — CI+existing behavior kept as overwrite-without-prompt** (pre-existing "Use --ci to overwrite" semantics). The D5 design flow diagram only specifies interactive prompt; keeping CI-overwrite preserves TestInitCommand_AlreadyExists ("Config written to" under --ci with existing config) and the documented `--ci` overwrite contract. If interactive-only overwrite is desired in CI, that's a separate decision.
4. **B4 — smoke no-config assertion seam**: design D6 says smoke asserts config-load states, but `upp list` cannot distinguish missing vs empty config (buildAdapterList treats absent cfg.Tools entries as enabled and lists the whole catalog). The D6 assertions therefore use `upp export` output: `[tools.apt]` present for empty/partial existing files, absent for a missing file. Same behavior asserted at the unit level (TestLoadMissingFile_NoDefaults pins 0 tools); smoke now pins the user-visible TOML.
5. **B4 — shared probe helper shape**: 3.4's "shared probeSetup across both probe files" resolved as probeHome(t) + probeSetup(t, tool) in a shared probe_test.go. Init probes need HOME isolation but manage their own config and need no fake tools, so they consume probeHome; audit probes consume probeSetup (which now calls probeHome). The fake-tool/config part stays init-agnostic — no forced reuse where it doesn't fit.

## Issues Found

1. ~~**Pre-existing classifier gap**: compact pipe `curl https://x.com|sh` classified RiskLow — hasPipeToShell only matched spaced variants.~~ **RESOLVED in Batch 3** via amendment tasks 2.12/2.13 (user decision): compact `|sh`/`|bash` now High risk; spaced behavior preserved (all pre-existing pipe rows still green). Note: `|shell` also matches compact `|sh` (conservative false positive in the safe direction — heuristic documented, no behavioral impact on legit commands).
2. **D4 bug caught by triangulation during 2.7 GREEN**: first implementation returned ConfirmProceed for CI low-risk (info path before CI check); DecisionMatrix row "trusted CI low" caught it; fixed to return ConfirmAuto in CI before printing info. Resolved in-slice.
3. **Trusted bool was dead code pre-change**: old `Trusted: info.Trust == adapters.TrustOfficial` was always false for custom tools (post-PR1 semantics), silently treating all custom tools as untrusted in confirm. Typed TrustLevel (2.7/2.8) removes the class of bug.
4. **B3 — init probe stdin seam**: the overwrite prompt reads os.Stdin directly (fmt.Scanln), so the probes replace os.Stdin with a temp file during root.Execute. This is the design's accepted stdin behavior (open question resolved: no input seam in this change; hermetic-CLI-test follow-up tracked out of scope).

## Remaining Tasks

None — all 26 tasks complete (Phase 1–4, incl. amendment tasks 2.12/2.13). Ready for sdd-verify.

## Status

Batch 1: 5/5 complete. Batch 2: 6/6 complete. Batch 3: 7/7 complete. Batch 4: 8/8 complete (3.1–3.4, 4.1–4.4). **Overall 26/26 tasks complete (incl. amendment tasks 2.12/2.13) — ALL phases done.** Work unit 4 finished: full suite + race + vet + gofmt clean, smoke 23/23 green (new D6 load-state assertions). Tree left dirty for orchestrator-owned delivery (PR #3 rides finalization).

## Collateral verification (Batch 4)

- go build ./... → OK (smoke build)
- go test ./... -count=1 → all 7 packages ok (cli 32.9s), exit 0
- go test ./... -race → all 7 packages ok (cli 33.5s), exit 0
- go vet ./... → clean, exit 0
- gofmt -s -w . → applied; gofmt -s -l . → empty (clean)
- bash scripts/smoke-test.sh → 23 passed, 0 failed, 23 total, exit 0
