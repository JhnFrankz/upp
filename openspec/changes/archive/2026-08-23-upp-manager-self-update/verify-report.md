```yaml
schema: gentle-ai.verify-result/v1
evidence_revision: sha256:9b1a5842cabe722833e5f7af6c31a55c5b75432ba4ae0b40703500ba2c4ca107
verdict: pass_with_warnings
blockers: 0
critical_findings: 0
requirements: 4/4
scenarios: 26/26
test_command: go test ./... -count=1
test_exit_code: 0
test_output_hash: sha256:c23b68fa5bc4df726a4f6e6afd7a75a10da40debf4ae3219417222c2aa8907f2
build_command: go build ./...
build_exit_code: 0
build_output_hash: sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
```

## Verification Report

**Change**: upp-manager-self-update
**Version**: N/A (delta specs at changeRoot; archived specs synced during apply 3.7)
**Mode**: Strict TDD (strict_tdd runner present; TDD evidence in apply-progress id 369)

### Completeness
| Metric | Value |
|--------|-------|
| Tasks total | 15 |
| Tasks complete | 15 |
| Tasks incomplete | 0 |

All 15 tasks (1.1-1.3, 2.1-2.5, 3.1-3.7) are checked. applyState=all_done. Full verification is authorized.

### Build & Tests Execution
**Build**: ✅ Passed (go build ./... exit 0)
**go vet**: ✅ Passed (exit 0, no findings)
**gofmt -s -l .**: ✅ Clean (no output)
**Tests**: ✅ `go test ./... -count=1` — all 8 packages ok, exit 0
**Focus (official)**: ✅ `go test ./internal/adapters/official -run 'TestCheck|TestUpdate|TestParseWingetUpgradeOutput|TestParseScoopStatusOutput' -count=1` — ok, exit 0
**Focus (cli)**: ✅ `go test ./internal/cli -run 'TestRunUpdate_ManagerSelfUpdate' -count=1` — ok, exit 0
**Smoke**: ✅ `bash scripts/smoke-test.sh --skip-build` — 31 passed, 0 failed, 31 total (exit 0). (Harness counts 31 checks, not the 32 the task prompt anticipated; all green.)
**Coverage**: official 93.9%, cli 91.8% (statements). Changed-file coverage: apt.go 96.3%, brew.go 100%, winget.go 95.2%, scoop.go 89.5%, helper.go (parse fns 90.9%/92.3%), all ≥80%.

### Spec Compliance Matrix
Authoritative counts from changeRoot specs: tool-adapter = 3 requirements / 20 scenarios (Official Adapter Catalog 5, Update Gating 7, Manager Self-Update Semantics 8); ux-patterns = 1 requirement / 6 scenarios (Manager Self-Update Row Rendering). Total: 4 requirements / 26 scenarios.

| Requirement | Scenario | Test | Result |
|-------------|----------|------|--------|
| Official Adapter Catalog | Linux brew adapter | `update_test.go > brew/success` (key brewUpdateCmd="brew update") | ✅ COMPLIANT |
| Official Adapter Catalog | macOS docker adapter | `update_test.go > docker/darwin-success` (pre-existing, skips on linux, runs on darwin) | ✅ COMPLIANT |
| Official Adapter Catalog | Linux apt self-only | `update_test.go > apt/success` (key aptUpdateCmd="sudo apt install --only-upgrade apt") | ✅ COMPLIANT |
| Official Adapter Catalog | Windows winget self-only | `update_test.go > winget/success` (key wingetUpdateCmd="winget upgrade winget") | ✅ COMPLIANT |
| Official Adapter Catalog | Windows scoop self-only | `update_test.go > scoop/success` (key scoopUpdateCmd="scoop update scoop") | ✅ COMPLIANT |
| Update Gating | Official update available | `cli/update_test.go > TestRunUpdate_GatingMatrix` | ✅ COMPLIANT |
| Update Gating | Official no update | `TestRunUpdate_GatingMatrix` (gated apt no update → current) | ✅ COMPLIANT |
| Update Gating | Stub official exempt | `TestRunUpdate_GatingMatrix` (brew false → still updates) | ✅ COMPLIANT |
| Update Gating | Custom exempt | `TestRunUpdate_GatingMatrix` (custom trusted/untrusted) | ✅ COMPLIANT |
| Update Gating | winget/scoop exempt | `TestRunUpdate_GatingMatrix` (winget false → still updates) | ✅ COMPLIANT |
| Update Gating | Dynamic detection | `TestRunUpdate_GatingMatrix` (gated apt without update) | ✅ COMPLIANT |
| Update Gating | Gated check fails | `TestRunUpdate_GatingMatrix` (failed → reported failed, never current) | ✅ COMPLIANT |
| Manager Self-Update Semantics | brew current-only | `check_test.go > brew/normal` (4.1.0/4.1.0, UpdateAvailable=false) | ✅ COMPLIANT |
| Manager Self-Update Semantics | brew never mutates in check | `check_test.go` brew rows (shell key brewUpdateCmd=failIfRun) | ✅ COMPLIANT |
| Manager Self-Update Semantics | brew self-update | `update_test.go > brew/success` (runs "brew update", never "brew upgrade brew") | ✅ COMPLIANT |
| Manager Self-Update Semantics | apt real detection | `check_test.go > apt/update-available` (2.4.0/2.4.5 → true) | ✅ COMPLIANT |
| Manager Self-Update Semantics | apt gated sudo | `update_test.go > apt/success` (Privileges:["sudo"], Gated) | ✅ COMPLIANT |
| Manager Self-Update Semantics | winget tolerant parse | `check_test.go > winget/update-available` + `parity_test.go > TestParseWingetUpgradeOutput` (v1.8.2311) | ✅ COMPLIANT |
| Manager Self-Update Semantics | winget old version | `check_test.go > winget/old-version-no-row` (v1.4.0 no row → false, no error) | ✅ COMPLIANT |
| Manager Self-Update Semantics | scoop parity | `update_test.go > scoop/success` (runs "scoop update scoop") | ✅ COMPLIANT |
| Row Rendering (ux-patterns) | brew current on board | `output/checkboard_test.go > TestCheckBoard_Complete_CurrentShowsUpToDate` | ✅ COMPLIANT |
| Row Rendering | brew never pending | `cli/update_test.go > TestRunUpdate_ManagerSelfUpdateBrewNeverSelector` | ✅ COMPLIANT |
| Row Rendering | brew dry-run current | `cli/update_test.go > TestRunUpdate_ManagerSelfUpdateDryRun` | ✅ COMPLIANT |
| Row Rendering | apt dry-run planned | `TestRunUpdate_ManagerSelfUpdateDryRun` (apt "would update") | ✅ COMPLIANT |
| Row Rendering | winget dry-run planned | `TestRunUpdate_ManagerSelfUpdateDryRun` (winget "would update") | ✅ COMPLIANT |
| Row Rendering | brew list version | `list.go` (CurrentVersion) + `output/render_test.go > TestListTools` (4.1.0) | ✅ COMPLIANT |

**Compliance summary**: 26/26 scenarios compliant.

### Correctness (Static Evidence)
| Requirement | Status | Notes |
|------------|--------|-------|
| brew self-only | ✅ Implemented | Update→"brew update" only; Check() current-only; footgun comment present; no `brew upgrade brew` in code/tests |
| apt self-only | ✅ Implemented | Update→"sudo apt install --only-upgrade apt"; Priveleges:["sudo"] on all real paths; Gated preserved; no `apt upgrade` |
| winget self-only | ✅ Implemented | Check=version+own-row (Microsoft.AppInstaller); <1.6 graceful; Update→"winget upgrade winget" |
| scoop self-only | ✅ Implemented | Check parses "scoop status" fail-closed; Update→"scoop update scoop" (never `*`) |
| Gating (brew only false; winget/scoop real) | ✅ Implemented | update.go sequential+interactive policy gate confirmed unchanged; no CLI-side ID gating |
| CLI render (brew current; apt/winget planned) | ✅ Implemented | update.go/checkrun.go unchanged; delegates row state to adapter Check() |

### Coherence (Design)
| Decision | Followed? | Notes |
|----------|-----------|-------|
| D1 Repurpose adapters in place | ✅ Yes | No new files/IDs; brew/apt/winget/scoop modified in place |
| D2 brew Check current-only; Update=brew update only | ✅ Yes | Matches brew.go; failIfRun guard in check fakes |
| D3 apt unchanged Check; gated sudo self-only | ✅ Yes | Matches apt.go |
| D4 winget version+own-row; Update=winget upgrade winget | ✅ Yes | wingetSelfID=Microsoft.AppInstaller; isVersionLike tolerates leading v |
| D5 scoop status own-row; Update=scoop update scoop | ✅ Yes | Matches scoop.go; fail-closed fallback |
| D7 interactive brew gap accepted | ✅ Yes | brew never pending in TTY (confirmed by selector test); sequential/--ci runs via PolicyAlwaysUpdate |
| Open Q: scoop status shape | ⚠️ Accepted risk | Not validated on real Windows; fail-closed parse + current-only fallback tested hermetically; design explicitly accepts fallback |
| Open Q: winget own-row key | ✅ Resolved | Microsoft.AppInstaller (unambiguous vs display/Source "winget"); pure helper |

### Issues Found
**CRITICAL**: None
**WARNING**: 
- Real-Windows `scoop status` / `winget upgrade` output shape is untested on real hardware (hermetic fakes only). The fail-closed parse + current-only fallback is tested and the design explicitly accepts this risk → WARNING, not a CRITICAL. No spec scenario is left uncovered.
- darwin/windows per-GOOS update rows (docker/gh/go, plus the winget/scoop rows) cannot execute on the linux host (rows skip via GOOS guard); they pass on their native GOOS.
**SUGGESTION**: 
- smoke harness counts 31 checks (task prompt said 32); the harness is authoritative and green.
- `scoop.Update` coverage 77.8% (before/after status-re-parse branch less exercised); acceptable.

### Verdict
PASS WITH WARNINGS
All 4 requirements / 26 scenarios are covered by passing runtime tests; the only WARNINGs are the real-Windows `scoop status`/`winget upgrade` shape (accepted design risk with a tested fail-closed fallback) and natively-skipped per-GOOS rows on linux. No blockers, no CRITICAL, no UNTESTED/FAILING scenario.
