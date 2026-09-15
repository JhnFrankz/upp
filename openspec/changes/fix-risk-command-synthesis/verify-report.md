```yaml
schema: gentle-ai.verify-result/v1
evidence_revision: sha256:482f110f75041ae74a1cd7902d591ccf6c0609acd5b0a6f54681e76424167d05
verdict: pass_with_warnings
blockers: 0
critical_findings: 0
requirements: 3/3
scenarios: 0/0
test_command: go test ./... -count=1 -race
test_exit_code: 0
test_output_hash: sha256:e8dc6e8d3bf173926715fdf50564a5d4a02d19791a2ed2c1758f730dc19e09c4
build_command: go build ./...
build_exit_code: 0
build_output_hash: sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
```

## Verification Report

**Change**: `fix-risk-command-synthesis`
**Date**: 2026-09-15
**Branch**: `feat/plan-command-derivation`
**Mode**: Strict TDD (`openspec/config.yaml` `testing.strict_tdd: true`, runner `go test ./... -count=1`)
**Artifact store**: openspec (native, authoritative) — `openspec/changes/fix-risk-command-synthesis/verify-report.md`
**Evidence Revision**: `sha256:482f110f75041ae74a1cd7902d591ccf6c0609acd5b0a6f54681e76424167d05`
**Verification scope**: proposal, both delta specs (`security-model`, `tool-adapter`), design (D1–D6), tasks (Phases 2–3), and the 8 uncommitted files in the working tree (659 changed lines: 574 insertions, 85 deletions). Phase 1 (`internal/adapters/*`) is already committed as `9074599`; its declaration surface was re-inspected as an upstream input of this verification.

> **Evidence-revision definition**: SHA-256 of the concatenation, in fixed order, of the raw hex digests of the test, build, smoke, and vet outputs: test `e8dc6e8d…` + build `e3b0c442…` (empty) + smoke `88593814…` + vet `e3b0c442…` (empty).

> **Count reconciliation (required)**: the envelope uses the native spec counts — `### Requirement:` headings (3) and `#### Scenario:` headings (0). Both delta specs express their scenarios as Given/When/Then *table rows* rather than `#### Scenario:` headings, so the native heading count for scenarios is legitimately 0. The audited table-row totals are **29 scenarios** (16 in `security-model`, 13 in `tool-adapter`): 28 satisfied, 1 pre-existing unmet (`Standard update` for npm, see W2). The prose sections below walk all 29; the envelope is not inflated beyond the native heading count.

> **Candidate scope**: the implementation under verification is UNCOMMITTED. Nothing was staged, committed, stashed, or checked out in the repository; all hunk-reversion experiments ran in a throwaway copy outside the repo (section *RED→GREEN Re-derivation*).

---

## Verification Commands (executed in the foreground, this run)

| # | Command | Observed result |
|---|---------|-----------------|
| 1 | `gofmt -s -l .` | exit 0, **no output** (clean); payload hash `sha256:e3b0c442…` |
| 2 | `go vet ./...` | exit 0, **no output** (clean); payload hash `sha256:e3b0c442…` |
| 3 | `golangci-lint run ./...` (v2.13.2, built with go1.27.0) | exit 0 — `No issues found` |
| 4 | `go test ./... -count=1 -race` | exit 0 — 11 packages `ok`, `cmd/upp` no test files, 0 failures; output hash `sha256:e8dc6e8d…` |
| 5 | `bash scripts/smoke-test.sh --skip-build` | exit 0 — `Results: 35 passed, 0 failed, 35 total` / `All tests passed!` |
| 6 | `go build ./...` | exit 0, **no output**; output hash `sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855` |

```text
$ go test ./... -count=1 -race
?   	github.com/JhnFrankz/upp/cmd/upp	[no test files]
ok  	github.com/JhnFrankz/upp/internal/adapters	1.633s
ok  	github.com/JhnFrankz/upp/internal/adapters/official	1.758s
ok  	github.com/JhnFrankz/upp/internal/cli	2.094s
ok  	github.com/JhnFrankz/upp/internal/config	1.037s
ok  	github.com/JhnFrankz/upp/internal/engine	1.424s
ok  	github.com/JhnFrankz/upp/internal/output	15.966s
ok  	github.com/JhnFrankz/upp/internal/platform	1.019s
ok  	github.com/JhnFrankz/upp/internal/security	1.045s
ok  	github.com/JhnFrankz/upp/internal/selfupdate	2.056s
ok  	github.com/JhnFrankz/upp/internal/uninstall	1.026s
```

`rules.verify.test_command` from `openspec/config.yaml` (`go test ./... -count=1 -race`) is exactly command 4. Race detector included.

---

## Completeness

| Metric | Value |
|--------|-------|
| Tasks in scope | 2.1–2.5, 3.1–3.3 in scope; 3.4 is a carry-forward note |
| Tasks complete (`[x]`) | 8/8 in scope (2.1, 2.2, 2.3, 2.4, 2.5, 3.1, 3.2, 3.3) |
| Tasks incomplete (`[ ]`) | 1 — `tasks.md:24` (3.4), the archive-note carry-forward; intentionally unchecked, not a code task |
| Deliverable | 7 code files + `tasks.md`, 659 changed lines (574 insertions / 85 deletions) |
| Phase 1 baseline | committed at `9074599` (9 files); only declaration-only refactors, executed command strings byte-preserved |

Task boxes confirmed by inspecting the tree, not the checkmarks:

- **2.1** engine plan tests exist: `internal/engine/plan_test.go:974` (row-class pins, pacman self/owned, custom delegated/standalone, npm fallback) and `:1130` (declaration byte-equality).
- **2.2** `internal/engine/plan.go:131-176` row-class derivation; `ManagerID` on self-rows iff declared privileges (`plan.go:165-167`); `detectPrivileges` untouched (`plan.go:210-219`).
- **2.3** `UpdateCmdName` deleted from `internal/engine/resolve.go` (file now ends at `OwnedPackage`, `:172`) and its test block removed from `internal/engine/resolve_test.go`.
- **2.4** satisfied by composition, not literally — see direct verdict 1.
- **2.5** `internal/cli/update_test.go:2187` (privileged row decisions), `:2237` (`--ci` non-zero), `:2251` (approved prompt), `:2265` (custom delegated gate incl. `--ci`).
- **3.1** comment-only: `internal/cli/update.go:435-439`, `internal/security/confirm.go:37-43`; no logic change (verified by reading the diff: only comment lines).
- **3.2** battery green (commands 1, 2, 3, 4).
- **3.3** smoke green (command 5); pacman/apt/brew/winget behavior verified below.

---

## Scenario Compliance Matrix

Status legend: **PASS** = a covering test executed green in this run; **UNMET (pre-existing)** = not compliant, and not caused by this candidate.

### Delta `security-model` — Requirement: Confirmation for Destructive Operations (12 scenarios)

| Scenario | Evidence (executed this run) | Status |
|---|---|---|
| Custom privileged | `internal/security/confirm_test.go:94`; unchanged custom standalone path (`plan.go:168-172` → `info.Command`) | PASS |
| Custom destructive | `internal/security/security_expanded_test.go:48` (`rm -rf` → High) + confirm matrix | PASS |
| `--ci` high-risk | `confirm_test.go:37`, `:66`; `internal/cli/update_test.go:237` | PASS |
| `--ci` trusted high-risk | `confirm_test.go:66`; `internal/cli/update_test.go:1804` | PASS |
| Sudo-heavy group prompts | `internal/cli/update_test.go:1699`, `:1930`; gate unit `confirm_test.go:162` | PASS |
| Non-sudo group proceeds | `internal/cli/update_test.go:2030` | PASS |
| `--ci` sudo group fails | `internal/cli/update_test.go:1804` (exit non-zero, `updatePkgCount == 0`) | PASS |
| Pacman privileged update prompts | plan: `plan_test.go:1007-1015` (real pacman → `sudo pacman -S --noconfirm pacman`, `ManagerID=pacman`, `[sudo]`); gate: `update_test.go:2187`, `:2251` (pacman-faithful fake) | PASS |
| `--ci` pacman privileged fails | `update_test.go:2237` (non-zero, row not executed, `Failed: pacman`) | PASS |
| Gate input is the executed command | composed: `plan_test.go:1130` (plan == declaration) + `internal/adapters/official/update_test.go:1385` (declaration == executed) | PASS (composition) |
| Pacman package gate sees sudo | `plan_test.go:1057-1065` + `:1187-1191` (synthetic owned fixture → `sudo pacman -S --noconfirm ripgrep`, `[sudo]`, `ManagerID=pacman`); gate mapping `update.go:447-456`, classifier `trust.go:36-47` → High | PASS (composition, no CLI-level pacman-package test — W4) |
| Custom manager-delegated confirm | `plan_test.go:1066-1075` (real pacman owner); `internal/cli/update_test.go:2265` (prompt shows `sudo pacman -S --noconfirm pacman`, asserts `pacman upgrade mytool` never appears, `--ci` non-zero) | PASS |

### Delta `security-model` — Requirement: Output Transparency (4 scenarios)

| Scenario | Evidence | Status |
|---|---|---|
| Standard update (`npm`) | `internal/adapters/official/npm.go:68` executes `npm update -g`; `npm.Info()` (`npm.go:97-106`) declares no `Command`, so `plan.go:202-207` yields `npm update` — now pinned by `plan_test.go:1086-1095`. Identical to base behavior. | **UNMET (pre-existing)** — W2 |
| Custom update | unchanged custom path; `promptUser` displays name, command, privileges, risk (`confirm.go:113-123`) | PASS |
| Pacman row transparency | rendered real per-package command + `[sudo]` (`plan.go:143-147`, `plan_test.go:1057-1065`); prompt path `confirm.go:113-123` | PASS |
| Custom delegated transparency | `plan.go:148-152` → owner `SelfUpdateCommand`; CLI `update_test.go:2265` | PASS |

### Delta `tool-adapter` — Requirement: Adapter Interface (13 scenarios)

| Scenario | Evidence | Status |
|---|---|---|
| Tool installed | pre-existing adapter suite (`internal/adapters/official`), green | PASS |
| Tool missing | pre-existing adapter suite, green | PASS |
| Update available | pre-existing adapter suite, green | PASS |
| No update | pre-existing adapter suite, green | PASS |
| Update succeeds | pre-existing adapter suite, green | PASS |
| Update fails | pre-existing adapter suite, green | PASS |
| ToolInfo carries owner | `internal/adapters/official/info_test.go:36-38` (gh/docker/go) | PASS |
| Manager carries kind | `info_test.go` manager rows; `plan_test.go:158` branch exercised | PASS |
| Owned tool carries package | `info_test.go:36-38`; `plan_test.go:505`, `:553` | PASS |
| pacman declares sudo | `internal/adapters/official/pacman.go:43` + `info_test.go:106-109` | PASS |
| Declared equals executed | `internal/adapters/official/update_test.go:1385` (all 5 managers, self + package rows) ∘ `plan_test.go:1130` | PASS (composition) |
| apt/brew/winget byte-stable | `plan_test.go:1169-1185` literals (`sudo apt install --only-upgrade gh`, `brew upgrade gh`, `winget upgrade gh`); independently re-derived against base code (section below) | PASS |
| Custom delegated inherits declaration | `plan_test.go:1066-1075`; `internal/cli/update_test.go:2265` | PASS |

---

## Correctness and Design Coherence

| Decision | Implementation | Verdict |
|---|---|---|
| **D3** row-class derivation | `plan.go:131-172`: self-row → `SelfUpdateCommand`; owned → `RenderPackageCommand(owner.PackageUpdateCommand, pkg)`; custom delegated (`info.Manager == nil`) → owner `SelfUpdateCommand`; standalone → declared `Command` or `<name> update` | Coherent. Row classes match the reachable registry paths (`ResolvingOwner` `resolve.go:121-152`; custom delegation `custom.go:142-146` inline); custom delegated rows are correctly detected via `info.Manager == nil` because `CustomAdapter.Info()` never populates `Manager` (`custom.go:134-152`). |
| **D4** privileged self-row gate enablement | `plan.go:165-167` sets `ManagerID = toolID` iff `info.Privileges` non-empty; gate consumes `ManagerID != ""` as `EnforceRisk` (`update.go:455`, `confirm.go:65-67`) | Coherent and fail-safe: setting `ManagerID` can only bypass the `TrustOfficial → ConfirmAuto` short-circuit toward stricter outcomes (High → prompt/`ConfirmError`; Medium + CI → `ConfirmError`), never looser. Only pacman declares privileges today, so only pacman rows tighten. |
| **D6** synthesis retirement | `UpdateCmdName` deleted (`resolve.go`, previously `:176`); `RenderPackageCommand` (`interface.go:138-143`) replaces it; no production references remain | Coherent. Rendered output for apt/brew/winget owned rows is byte-identical to the retired synthesis. |
| Comment-only scope | `update.go:435-439`, `confirm.go:37-43` are comments; no logic lines changed | Coherent with design and proposal scope. |
| Executed commands unchanged | Phase 1 (`9074599`) replaced literals with constants; every executed string is byte-preserved (apt/brew/winget/scoop/pacman, self and package) | Verified by reading the commit diff. |

### Independent base-vs-candidate comparison (hunk reversion)

- Reverting `plan.go` + `resolve.go` to base `HEAD` while keeping the new engine tests reproduced RED for every self-row and pacman row (`APT Package Manager update` vs `sudo apt install --only-upgrade apt`, `pacman upgrade ripgrep` vs `sudo pacman -S --noconfirm ripgrep`, `pacman upgrade mytool` vs `sudo pacman -S --noconfirm pacman`) — confirming the new tests are RED-capable and the candidate turns them GREEN. Crucially, the **apt/brew/winget owned `gh` rows passed against BOTH base and candidate code**, which is the direct byte-stability proof for those rows.
- Reverting `plan.go` to base while keeping the new CLI gate tests reproduced RED for all pacman/delegated CLI scenarios (`prompt shown = false, want true`; `--ci error = nil, want non-zero`) — the gate tightening is genuinely produced by this candidate.

### User-visible behavior delta (item 3)

| Row class | Base | Candidate | Decision |
|---|---|---|---|
| apt/brew/winget owned (`gh`, `docker-ce`, …) | `sudo apt install --only-upgrade gh` / `brew upgrade gh` / `winget upgrade gh`; `ManagerID` set; privileges `[sudo]`/none/none | identical bytes, identical `ManagerID`, identical privileges | **identical** (High → prompt; Low → info+proceed; CI High → non-zero) |
| apt/brew/winget self | `<Name> update` fiction; privileges per `detectPrivileges(fiction)` | real command (`sudo apt install --only-upgrade apt`, `brew update`, `winget upgrade winget`); apt self now reports `[sudo]` | **identical outcome**: no declared privileges → `ManagerID` stays empty → `EnforceRisk=false` → `ConfirmAuto`, and `RiskCommand`/`Privileges` only surface through `printInfo`/`promptUser` (`confirm.go:107-123`), neither of which is reached on the auto-proceed path; dry-run prints no command (`update.go:218-223`). No user-visible change. |
| pacman self / pacman owned / custom `manager=pacman` | sudo-free fiction → Low → silent proceed (even under `--ci`) | real sudo command → High → prompt; `--ci` non-zero | **intended change** (mandated by both delta specs) |

---

## Orchestrator Questions — Direct Verdicts

**1. Task 2.4 not implemented literally — is the spec requirement genuinely satisfied by the composition? Is the structural claim true?**
**Verdict: the spec requirement is genuinely satisfied; a literal capture is NOT required. The structural claim is TRUE.**
- Structural claim verified: `runCmdFn`, `runCmdArgsFn`, `lookPathFn` are unexported package-level `var`s in `package official` (`helper.go:20-23`); `engine` imports `official` (`plan.go:6-7`) and `official` does not import `engine` (0 matches); there is no exported test seam (`Set…`/`ForTest`) in any non-test file of `official`. Go test files of a package are not visible to importers, so an `engine` test cannot reference the seam — including via a hypothetical `export_test.go`, whose identifiers are compiled only into `official`'s own test binary. A literal cross-package capture would therefore require adding a production exported seam purely for testing, which the design deliberately avoided (design D2/T3).
- The two proven equalities: (a) `plan.RiskCommand == declaration` for every managed row class (`plan_test.go:1130`, plus row-class pins `plan_test.go:974`); (b) `declaration == executed` for every manager/package row through the real `Update(false)`/`UpdatePackage(pkg)` path with the seam recorder (`official/update_test.go:1385`). Both halves are load-bearing, demonstrated by mutation: changing only `pacman.Info().SelfUpdateCommand` to diverge from the executed constant left the engine half GREEN and turned the `official` half RED (`TestUpdateRunsDeclaredCommand/pacman/self`); removing the `ManagerID`-on-privileged-self-row hunk left apt/brew/winget GREEN and turned exactly the pacman plan + CLI rows RED.
- The spec text constrains *behavior* ("plan.RiskCommand and plan privileges MUST byte-equal the executed command for every managed adapter, pacman included"), not the test topology. Transitivity holds because both halves read the same declaration (`Info()` of the same adapter type from the same constants). Composition is the strongest available evidence given the package boundary.
- Residual weakness (W3): the `official` half asserts the declaration is *present among* the captured commands, not that the executed command set is *exactly* the declaration, and it does not assert `Result.Privileges` equality. Acceptable for this requirement; a stronger assertion is recommended.

**2. Latent fail-open when an owner declares no `PackageUpdateCommand`.**
**Verdict: truly unreachable with the shipped registry; no delta scenario forbids it explicitly; keep as a documented latent risk with a recommended fail-closed guard.**
- Branch: `plan.go:153-156` (`packageName == ""` OR `ownerInfo.PackageUpdateCommand == ""` with `info.Manager != nil`) → `standaloneRiskCommand` → `"<name> update"`.
- Reachability: the only managers reachable as owners from the shipped registry are apt/brew/winget, via the hardcoded `Manager` maps on `gh` (`gh.go:87`), `docker` (`docker.go:86`), and `go` (`go.go:146`). All three declare `PackageUpdateCommand`. `scoop` is the only manager with no `PackageUpdateCommand` (`scoop.go:106-113`), and no adapter declares `Manager[...] = "scoop"` (0 matches). Custom tools configured with `manager = "scoop"` take the delegated branch instead (`custom.Info()` never sets `Manager`), so they cannot reach it. `pacman` also declares the template, and the pacman-owned fixture is test-only.
- If it ever became reachable, the row would classify the fiction `"<toolName> update"` while `executePlannedUpdate` runs the tool's own `a.Update(false)` (`update.go:485-486`) — the exact divergence this change removes. Design already documents this as an unreachable corner (`design.md:33`, `plan.go:154-155`). Recommend a guard (e.g. fall back to the owner's `SelfUpdateCommand` or set `ManagerID` to fail closed) plus a comment/assert so a future ownership declaration cannot silently reintroduce the bug.

**3. Unchanged behavior for apt/brew/winget.**
**Verdict: gate decisions byte-identical; owned-row commands byte-identical; manager self-row plan metadata changed as required by the transparency delta, with no user-visible effect.**
See the base-vs-candidate table above. Owned rows are byte-stable both as strings and as decisions; the apt owned row's privileges derive from `detectPrivileges` on the identical command in both versions. Self-rows changed from the `<Name> update` fiction to the real command (mandated), and apt's self-row plan `Privileges` went `nil → [sudo]` (derived from the now-truthful command) — but the row remains `ConfirmAuto` and silent because `ManagerID` stays empty (`EnforceRisk=false`), and `Privileges` is only rendered by `promptUser`. Verified empirically: the apt-like/brew-like/winget-like self-row CLI cases auto-proceed green (`update_test.go:2187`), and the D4-hunk mutation left them GREEN.

**4. Pacman: interactive prompt and non-zero exit under `--ci`.**
**Verdict: satisfied.** Real `PacmanAdapter.Info()` declares `Privileges: ["sudo"]` (`pacman.go:43`) and `SelfUpdateCommand = "sudo pacman -S --noconfirm pacman"` (`pacman.go:22`); `plan.go:165-167` therefore marks the pacman self-row `ManagerID="pacman"`, and the gate derives `EnforceRisk=true` (`update.go:455`). `ClassifyCommand` returns High for any command containing `sudo` (`trust.go:36-47`), so interactive → `promptUser` (`confirm.go:99-102`) and `--ci` → `ConfirmError` → `Failed: pacman` + non-zero (`update.go:464-469`, `423-425`). Proven end-to-end at the CLI layer with a pacman-faithful fake whose declaration strings are byte-identical to `pacman.go:22/27` (`update_test.go:2187`, `:2237`, `:2251`), and at the plan layer with the real pacman adapter (`plan_test.go:1007-1015`). Also confirmed by mutation: removing the D4 hunk reproduces the pre-change fail-open (`prompt shown = false`, `--ci error = nil`).

**5. Delta spec scenarios.**
**Verdict: 28/29 satisfied; 1 unmet and pre-existing.** The single unmet scenario is `security-model / Output Transparency / Standard update` (npm): the spec requires the display to show `npm update -g`, while the plan declares `npm update`. Root cause is pre-existing and out of this change's scope: `npm.Info()` declares no `Command` (`npm.go:97-106`) although `npm.Update()` runs `npm update -g` (`npm.go:68`); base `plan.go` produced the same `"npm update"` string, and `proposal.md`/`design.md:31` explicitly record standalone official adapters as residual fiction outside scope. Note the candidate *newly pins* this divergent behavior at `plan_test.go:1086-1095`; that is deliberate documentation of the pre-existing state, not a regression. All other 28 rows are satisfied as matrixed above.

**6. Review budget and the proposed two-slice split.**
**Verdict: the split as drawn is NOT cohesive/independently verifiable; the blocking delivery-plan finding D1 below applies. A three-slice stack keeps every slice ≤400 changed lines and green.**
- Arithmetic confirmed: PR2a = `plan.go` 66 + `plan_test.go` 291 + `resolve.go` 15 + `resolve_test.go` 20 = **392**; PR2b = `update_test.go` 237 + `update.go` 7 + `confirm.go` 7 = **251**; plus `tasks.md` 16 → 659 total.
- Independently verifiable? **No.** With the PR2a file set applied and the PR2b file set reverted to base, `go test ./internal/cli/...` is RED: 187 passed / 2 failed (`TestRunUpdate_CISudoFailsClosedWithEnforceRisk` — `--ci` no longer fails; `TestRunUpdate_DefaultGroupSummarySkipsDeselected` — the sudo prompt no longer appears, so the denied docker row is not skipped). Cause: the pre-existing CLI group fakes must declare `packageUpdateCommand` for declaration-driven classification to reproduce the sudo string; those lines live in the PR2b file. The intermediate commit would therefore land broken.
- Verified corrected split (each slice green in a scratch copy):
  1. **S1 — CLI test harness surface only** (~82 changed lines in `internal/cli/update_test.go`: two struct fields + `Info()` alignment, 10 fake `packageUpdateCommand:` additions, the two mirrored template constants). Green on the base implementation (189 passed).
  2. **S2 — engine derivation + synthesis retirement** (392: `plan.go`, `plan_test.go`, `resolve.go`, `resolve_test.go`). Green on top of S1 (189 passed).
  3. **S3 — gate tests + gate comments** (~169: the four new pacman/delegated CLI tests + `runUpdateDefaultList`/`fakeManagerRow`, `update.go`, `confirm.go`). Green.
  Alternatively, keep two slices by moving the harness lines into slice A, at the cost of a ~474-line slice A — over the 400-line budget, so the three-slice stack is preferable.

---

## RED→GREEN Re-derivation (independent, in a throwaway copy)

All experiments ran on a `cp -a` copy under `/tmp/opencode/` (deleted afterwards). The repository working tree was never mutated: it still shows the same 8 modified files, 574 insertions / 85 deletions, no untracked files.

| # | Mutation in the copy | Observed result | Conclusion |
|---|---|---|---|
| 1 | `git checkout HEAD -- internal/engine/plan.go internal/engine/resolve.go` (base derivation), new tests kept | `TestPlan_RowClassRiskCommands` and `TestPlan_RiskCommandEqualsExecutedDeclaration` RED on all self rows + pacman owned/delegated (e.g. `RiskCommand = "APT Package Manager update", want "sudo apt install --only-upgrade apt"`); apt/brew/winget **owned `gh` rows GREEN** | New tests are RED-capable; owned-row byte stability holds against base |
| 2 | Same base revert, CLI gate tests kept | `TestRunUpdate_PrivilegedManagerRowDecisions/pacman…` RED (`prompt shown = false`, `updated = true`), `…CIFails` RED (`error = nil`), `…PromptApproved` RED, `TestRunUpdate_CustomManagerDelegatedGate` RED (both subtests) | The gate tightening is candidate-produced |
| 3 | Remove the D4 hunk (`if len(privileges) > 0 { managerID = toolID }`) | Engine: only `pacman self row … marks ManagerID` RED. CLI: the three pacman row tests RED; apt/brew/winget cases GREEN | D4 is precisely scoped; the fail-safe claim holds |
| 4 | `pacman.Info().SelfUpdateCommand` mutated to diverge from the executed constant | `TestUpdateRunsDeclaredCommand/pacman/self` RED; `TestPlan_RiskCommandEqualsExecutedDeclaration` fully GREEN | The execution half is load-bearing; neither half alone proves the property |
| 5 | `testAptPackageUpdateCommand` mutated to a non-sudo template | `TestRunUpdate_CISudoFailsClosedWithEnforceRisk` RED; `TestRunUpdate_DefaultGroupSummarySkipsDeselected` RED | The CLI group tests are bound to the declaration, not to synthesis |
| 6 | Base `plan.go` + harness-only CLI test surface (S1 emulation) | `go test ./internal/cli/...` GREEN (189 passed) | S1 is safe as a preparatory slice |

---

## Findings

> **Envelope semantics**: `critical_findings: 0` counts *implementation* non-compliances against the delta specs, design, and tasks. No such finding exists — every in-scope scenario is satisfied at runtime and the full battery is green. The delivery-plan blocker (D1) is recorded below as a separate blocking finding rather than an envelope critical, because the shipped bytes are compliant; it is nonetheless **blocking for the proposed split** and must be corrected before landing.

### BLOCKER (delivery plan — not counted in envelope `critical_findings`)

**D1 — The proposed two-slice split leaves slice PR2a with a red test suite.**
- `file:line` — split boundary between `internal/engine/plan.go:131-172` (candidate) and `internal/cli/update_test.go:2143-2146`, `:1710`, `:1766`, `:1809`, `:1851`, `:1898`, `:1934`, `:1974`, `:2006`, `:2032`, `:2065` (`packageUpdateCommand` declarations on the group-path fakes).
- Evidence — with the PR2a file set and base `internal/cli/update_test.go`: `go test ./internal/cli/... -count=1` → 187 passed / 2 failed (`TestRunUpdate_CISudoFailsClosedWithEnforceRisk`, `TestRunUpdate_DefaultGroupSummarySkipsDeselected`). Both are pre-existing tests that depend on the fakes declaring the per-package template once classification is declaration-driven.
- Classification — **candidate-caused** (the split definition, not the shipped bytes; the full working tree is green).
- Correction — adopt the verified three-slice stack (S1 harness ~82 → S2 engine 392 → S3 gate tests + comments ~169), or move the harness lines into slice A and accept a ~474-line slice A. Do not land PR2a as currently scoped.

### WARNING

**W1 — Latent fail-open branch when an owner declares no `PackageUpdateCommand`.**
- `file:line` — `internal/engine/plan.go:153-156` (with `:202-207`).
- Evidence — unreachable today: owners reachable from the registry are apt/brew/winget only (`gh.go:87`, `docker.go:86`, `go.go:146`), all of which declare the template; `scoop` declares none but owns nothing; custom `manager="scoop"` takes the delegated branch. If reached, the plan would classify `"<toolName> update"` while execution runs `a.Update(false)` (`update.go:485-486`).
- Classification — **candidate-caused** (branch introduced by D3), latent; design documents it as an unreachable corner (`design.md:33`).
- Correction — fail closed (`ownerInfo.SelfUpdateCommand` or set `ManagerID`) and/or add an assertion/comment so a future ownership declaration cannot silently reintroduce the divergence.

**W2 — Pre-existing unmet scenario: `security-model / Output Transparency / Standard update` (npm).**
- `file:line` — `internal/adapters/official/npm.go:68` (`npm update -g` executed) vs `npm.go:97-106` (no `Command` declared) → `internal/engine/plan.go:202-207` → pinned at `internal/engine/plan_test.go:1086-1095`.
- Evidence — plan/display command is `npm update`, the scenario requires `npm update -g`.
- Classification — **pre-existing** (identical at base `HEAD`; `proposal.md:16` and `design.md:31` scope standalone official fiction out).
- Correction — none required for this change; carry the archive note (`tasks.md:42`) into the archive phase.

**W3 — The execution-half test proves presence, not exact equality, and omits privilege equality.**
- `file:line` — `internal/adapters/official/update_test.go:1446-1456` (`found` flag over captured commands); design T3 asked for "privileges equal declared".
- Evidence — an adapter that executed the declaration *plus* another divergent command (e.g. `PnpmAdapter.Update`'s prune-retry, `pnpm.go:84-85`) would still pass. The composition remains sufficient for the delta's equality requirement, but the assertion is weaker than the design's stated test.
- Classification — **candidate-caused** (test strength).
- Correction — assert the exact mutating command set (or assert `captured` has exactly one entry for managers) and add `res.Privileges == info.Privileges`.

**W4 — No end-to-end CLI coverage with a real pacman adapter; the pacman *package* gate scenario is proven only compositionally.**
- `file:line` — `internal/cli/update_test.go:2150-2162` (fake), `internal/engine/plan_test.go:1057-1065` (synthetic fixture).
- Evidence — the delta's `Pacman package gate sees sudo` scenario is evidenced by a plan-level synthetic fixture plus a gate-level fake self-row; the registry has no pacman-owned tool (`Manager[...] = "pacman"`: 0 matches), so the production path is unreachable today.
- Classification — **candidate-caused** (coverage gap).
- Correction — optionally add a CLI test with a pacman-owned fake (Manager map + `ManagerPackage`) asserting prompt/`--ci` failure for the package row. The fake declaration strings are already byte-identical to `pacman.go:22/27`, so the current evidence is faithful.

**W5 — No `apply-progress` artifact exists for this change, so the Strict-TDD "TDD Cycle Evidence" table could not be checked.**
- `file:line` — `openspec/changes/fix-risk-command-synthesis/` contains only `proposal.md`, `design.md`, `tasks.md`, `specs/`.
- Evidence — the strict-TDD module's primary artifact is absent; the per-task RED/GREEN annotations live only in `tasks.md` lines 5-24.
- Mitigation — RED→GREEN was independently re-derived by hunk reversion (section above), which is stronger evidence than the table; no test file listed in `tasks.md` is missing, and all run green.
- Classification — **unknown** (persistence/orchestration gap; not a code defect).
- Correction — persist `apply-progress.md` for future changes so the module can be executed as designed.

### SUGGESTION

**S1 — Declaration sweep can pass vacuously if the registry is empty.** `internal/adapters/official/info_test.go:77-104` loops over `AllAdapters()` with no companion non-empty assertion. Phase 1 (committed); consider `if len(AllAdapters()) == 0 { t.Fatal(...) }`. — **pre-existing pattern**.

**S2 — Hand-mirrored templates in the CLI fakes can drift.** `internal/cli/update_test.go:2143-2146` duplicates the real apt/brew templates. Deriving them from `official.AdapterByName("apt").Info().PackageUpdateCommand` would make drift impossible. — **candidate-caused**.

**S3 — Task 2.4 traceability.** The literal task text (stub `runCmdFn` from an `engine` test) was replaced by the composition, documented at `tasks.md:16`. Recommend keeping that note verbatim in the archive so the substitution is auditable. — **candidate-caused**, accepted.

**S4 — `tasks.md:24` (3.4) remains unchecked by design.** It is the archive-note carry-forward, not code; verify the note reaches the archive phase (`tasks.md:42`). — **candidate-caused**, expected.

---

## Strict TDD Compliance

### TDD Compliance

| Check | Result | Details |
|-------|--------|---------|
| TDD evidence reported | ⚠️ | No `apply-progress` artifact in the openspec store (W5); RED/GREEN annotations exist in `tasks.md:5-24` |
| All tasks have tests | ✅ | 8/8 in-scope tasks map to test files; all exist |
| RED confirmed (tests exist and are RED-capable) | ✅ | 6/8 verified by hunk reversion (experiments 1-5); 2.3 is a deletion verified by inspection; 3.x are battery/comments |
| GREEN confirmed (tests pass on execution) | ✅ | 8/8; full suite green (command 4), targeted run: 44 relevant tests/subtests passed |
| Triangulation adequate | ✅ | Table-driven multi-case tests: `plan_test.go:974` (9 cases), `:1130` (9 cases), `update_test.go:2187` (4 cases), `official/update_test.go:1385` (9 cases), delegated gate (2 subtests) |
| Safety net for modified files | ✅ | `plan_test.go`, `update_test.go`, `resolve_test.go`, `official/update_test.go`, `info_test.go` pre-existed; new cases were appended and existing cases re-run green |

**TDD Compliance**: 5/6 checks confirmed (the single ⚠️ is the missing evidence artifact, mitigated by independent re-derivation).

### Test Layer Distribution

| Layer | Tests | Files | Tools |
|-------|-------|-------|-------|
| Unit | `TestPlan_RowClassRiskCommands`, `TestPlan_RiskCommandEqualsExecutedDeclaration`, `TestManagerCommandDeclarations`, `TestConfirmAction_*`, `TestClassifyCommand_*` | `internal/engine/plan_test.go`, `internal/adapters/official/{info,update}_test.go`, `internal/security/{confirm,security_expanded}_test.go` | `go test` |
| Integration | `TestRunUpdate_PrivilegedManagerRowDecisions`, `…CIFails`, `…PromptApproved`, `TestRunUpdate_CustomManagerDelegatedGate`, `TestRunUpdate_DefaultBulkGroupExecution`, `…CISudoFailsClosedWithEnforceRisk`, `…GroupNonSudoProceeds`, `…DefaultGroupSummarySkipsDeselected`, `TestUpdateRunsDeclaredCommand` (exec-seam) | `internal/cli/update_test.go`, `internal/adapters/official/update_test.go` | `go test` |
| E2E | smoke suite (35 checks) | `scripts/smoke-test.sh` | bash harness |
| **Total** | 44 relevant tests/subtests + 35 smoke checks | 5 changed test files | |

### Changed File Coverage (function level, changed symbols)

| File | Symbol | Coverage | Rating |
|------|--------|----------|--------|
| `internal/engine/plan.go` | `Plan` | 94.0% | ✅ Excellent |
| `internal/engine/plan.go` | `standaloneRiskCommand` | 100.0% | ✅ Excellent |
| `internal/engine/resolve.go` | `ResolvingOwner` / `OwnedPackage` | 95.0% / 100.0% | ✅ Excellent |
| `internal/cli/update.go` | `executePlannedUpdate` | 90.6% | ✅ Excellent |
| `internal/security/confirm.go` | `ConfirmAction` / `promptUser` | 100.0% / 93.8% | ✅ Excellent |
| `internal/adapters/interface.go` | `RenderPackageCommand` (Phase 1) | 100.0% | ✅ Excellent |
| `internal/adapters/official/pacman.go` | `Info` / `Update` / `UpdatePackage` (Phase 1) | 100.0% / 100.0% / 100.0% | ✅ Excellent |

Cumulative statement coverage of the measured packages: **92.0%**.

### Assertion Quality

| File | Line | Assertion | Issue | Severity |
|------|------|-----------|-------|----------|
| `internal/adapters/official/update_test.go` | 1446-1456 | `found` loop over captured commands | Presence, not exact executed-set equality | WARNING |

**Assertion quality**: 0 CRITICAL, 1 WARNING. No tautologies, no assertions without production calls, no ghost loops, no smoke-only tests, no mock-heavy files. The engine tests compare plan output against the declaration computed from the same `Info()` source deliberately — they are the detector for reintroduced synthesis, and were proven load-bearing by experiments 1 and 4.

### Quality Metrics

**Linter**: ✅ No issues (`golangci-lint v2.13.2`, `run ./...`)
**Type Checker**: ✅ No issues (`go vet ./...`)
**Formatter**: ✅ Clean (`gofmt -s -l .` no output)

---

## Verdict

**PASS WITH WARNINGS.** The implementation satisfies both delta specs at runtime: `plan.RiskCommand`/plan privileges derive from adapter declarations and compose to the executed command; pacman rows now prompt and fail closed under `--ci`; apt/brew/winget owned-row commands and gate decisions are byte-identical to base; `UpdateCmdName` synthesis is retired; the full battery (gofmt, vet, golangci-lint, `go test -race`, smoke 35/35) is green on the uncommitted tree.

Warning set: one pre-existing unmet delta scenario (npm, W2), one documented latent unreachable fail-open branch (W1), a test-strength gap in the execution-half equality (W3), a pacman-package CLI coverage gap (W4), and a missing `apply-progress` artifact (W5). One **blocking delivery-plan finding** (D1): the proposed two-slice split would land a red intermediate commit; the verified three-slice stack fixes it. Zero implementation-critical findings; no implementation blocker.
