```yaml
schema: gentle-ai.verify-result/v1
evidence_revision: sha256:97db0af7d5e6f9ce9cf86e4d3b35d6637b3e5aabcfb357ba83367377cf07a53a
verdict: pass_with_warnings
blockers: 0
critical_findings: 0
requirements: 12/12
scenarios: 60/60
test_command: go test ./... -count=1 -race
test_exit_code: 0
test_output_hash: sha256:97db0af7d5e6f9ce9cf86e4d3b35d6637b3e5aabcfb357ba83367377cf07a53a
build_command: go build ./...
build_exit_code: 0
build_output_hash: sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
```

## Verification Report

**Change**: upp-ownership-model
**Version**: N/A (re-verify after remediation — resolves the WU3 grouping coverage CRITICAL from the first FAIL run)
**Mode**: Strict TDD (openspec/config.yaml `strict_tdd: true`)

### Completeness
| Metric | Value |
|--------|-------|
| Tasks total | 21 |
| Tasks complete | 21 |
| Tasks incomplete | 0 |

### Build & Tests Execution
**Build**: ✅ Passed (`go build ./...` exit 0, empty output)
**Vet**: ✅ Passed (`go vet ./...` exit 0, empty output)
**Format**: ✅ Passed (`gofmt -s -l .` empty, exit 0)
**Tests (plain)**: ✅ `go test ./... -count=1` → 8 packages ok, exit 0 (hash sha256:bcba4b51c714ffd8366a02ad03751a2642e335af62516c715e20fa9a9d9af201)
**Tests (race, config verify gate)**: ✅ `go test ./... -count=1 -race` → 8 packages ok, exit 0 (hash sha256:97db0af7d5e6f9ce9cf86e4d3b35d6637b3e5aabcfb357ba83367377cf07a53a)
**Focused (grouping wiring)**: ✅ `go test ./internal/output/... -run 'Group' -count=1` exit 0 — all GroupOrder/OwnerGroupLabel/GroupByOwner tests PASS
**Focused (adapters/cli/config)**: ✅ `go test ./internal/adapters/... ./internal/cli/... ./internal/config/... -count=1` exit 0
**Smoke**: ✅ `bash scripts/smoke-test.sh --skip-build` → 31 passed, 0 failed (hash sha256:6cbeba1217bb0bf490f31d98bfb434ca3de3ebb83c1d902fd48ceee711a9de95)
**Coverage (changed-file, informational; config threshold 0)**: adapters 91.7%, official 94.6%, cli 91.0%, config 80.6%, output 90.4%, platform 67.6%.

**Remediation applied**: `internal/output/group_test.go` added 10 tests covering `GroupOrder`, `OwnerGroupLabel`, `GroupByOwner`. Production `group.go` was NOT modified. Result: `GroupOrder` 100%, `OwnerGroupLabel` 100%, `GroupByOwner` 100% (was 0% for two of them in the first run).

**Coverage**: Informational — threshold 0 (config).

### Spec Compliance Matrix
12 requirements, 60 scenarios across 6 specs (tool-ownership-model 10, tool-adapter 17, ux-patterns 15, platform-detection 6, config-system 6, security-model 6).

#### tool-ownership-model (10/10 COMPLIANT)
| Requirement | Scenario | Test | Result |
|---|---|---|---|
| Tool Ownership Declaration | gh owner per platform | `registry_test.go > TestResolveOwner (gh-macos)` | ✅ COMPLIANT |
| Tool Ownership Declaration | docker owner per platform | `registry_test.go > TestResolveOwner (docker-windows)` | ✅ COMPLIANT |
| Tool Ownership Declaration | go Linux standalone | `registry_test.go > TestResolveOwner (go-linux-standalone)` | ✅ COMPLIANT |
| Tool Ownership Declaration | apt declares manager | `info_test.go > TestInfo (apt)` + `registry_test.go > TestKindManagerConsistency` | ✅ COMPLIANT |
| Manager Owned-Tool Cardinality | brew owns three on macOS | `registry_test.go > TestManagerOwnedToolCardinality (brew-macos)` | ✅ COMPLIANT |
| Manager Owned-Tool Cardinality | apt owns two on Linux | `registry_test.go > TestManagerOwnedToolCardinality (apt-linux)` | ✅ COMPLIANT |
| Manager Owned-Tool Cardinality | winget owns three on Windows | `registry_test.go > TestManagerOwnedToolCardinality (winget-windows)` | ✅ COMPLIANT |
| Resolved Owner Update Delegation | gh delegates on Linux | `update_test.go > TestUpdateDelegation (gh/linux-delegates-to-apt)` | ✅ COMPLIANT |
| Resolved Owner Update Delegation | go standalone on Linux | `update_test.go > TestUpdateDelegation (go/linux-standalone-keeps-own-cmd)` | ✅ COMPLIANT |
| Resolved Owner Update Delegation | docker delegates on macOS | `update_test.go > TestUpdateDelegation (docker/linux)` + `registry_test.go > TestResolveOwnerViaRuntimeGOOSToPlatform (docker-darwin)` | ✅ COMPLIANT |

#### tool-adapter (17/17 COMPLIANT)
| Requirement | Scenario | Test | Result |
|---|---|---|---|
| Adapter Interface | Tool installed / Tool missing | `detect_test.go > TestDetect` | ✅ COMPLIANT |
| Adapter Interface | Update available / No update | `check_test.go > TestCheck (apt/npm/pnpm/nvm/gh/docker/go)` | ✅ COMPLIANT |
| Adapter Interface | Update succeeds / Update fails | `update_test.go > TestUpdate` | ✅ COMPLIANT |
| Adapter Interface | ToolInfo carries owner | `info_test.go > TestInfo (docker/gh/go)` | ✅ COMPLIANT |
| Adapter Interface | Manager carries kind | `info_test.go > TestInfo (apt/brew/winget/scoop)` | ✅ COMPLIANT |
| Official Adapter Catalog | Linux brew adapter (runs `brew update`) | `update_test.go > TestUpdate (brew/success)`, `check_test.go > TestCheck (brew)` | ✅ COMPLIANT |
| Official Adapter Catalog | macOS docker delegates | `update_test.go > TestUpdateDelegation (gh/macos)` + `TestResolveOwnerViaRuntimeGOOSToPlatform (docker-darwin)` | ✅ COMPLIANT |
| Official Adapter Catalog | Linux gh delegates | `update_test.go > TestUpdateDelegation (gh/linux-delegates-to-apt)` | ✅ COMPLIANT |
| Official Adapter Catalog | Windows gh delegates | `update_test.go > TestUpdate (gh/windows-delegates-to-winget-success)` | ✅ COMPLIANT |
| Official Adapter Catalog | Linux apt self-only | `update_test.go > TestUpdate (apt/*)` + `check_test.go > TestCheck (apt)` | ✅ COMPLIANT |
| Update Gating | Owned inherits gated (docker owned by apt, no update) | `update_test.go > TestRunUpdate_OwnedToolInheritsGatedGate` | ✅ COMPLIANT |
| Update Gating | Owned inherits always (gh owned by brew) | `update_test.go > TestUpdateDelegation (gh/macos)` + `TestResolveEffectiveUpdatePolicy (gh-macos)` | ✅ COMPLIANT |
| Update Gating | Stub official exempt (brew/bun/opencode) | `update_test.go > TestRunUpdate_GatingMatrix (always-update brew exempt)` | ✅ COMPLIANT |
| Update Gating | Gated check fails | `update_test.go > TestRunUpdate_GatingMatrix (gated check fails)` | ✅ COMPLIANT |

#### ux-patterns (15/15 COMPLIANT — gap closed)
| Requirement | Scenario | Test | Result |
|---|---|---|---|
| List Table Output | Correct columns | `render_test.go > TestListTools_IDColumn` | ✅ COMPLIANT |
| List Table Output | Filter round-trip | `integration_test.go > TestListCommand_FilterRoundTrip_GroupingDisplayOnly` | ✅ COMPLIANT |
| List Table Output | Grouped by manager | `render_test.go > TestGroupByOwner_LinuxGroupsOwnedTools` + `TestListTools_GroupedHeaderThenChildren` | ✅ COMPLIANT |
| List Table Output | Owned tool not independent | `render_test.go > TestListTools_GroupedHeaderThenChildren` | ✅ COMPLIANT |
| List Table Output | Filters ignore grouping | `integration_test.go > TestListCommand_FilterRoundTrip_GroupingDisplayOnly` + `render_test.go > TestGroupByOwner_FilteredManagerNoPhantomHeader` | ✅ COMPLIANT |
| Live Check Board | Board renders grouped | `group_test.go > TestGroupOrder_OwnedToolGroupedUnderManager` + `TestGroupOrder_ManagersFollowCanonicalAllAdaptersOrder` + `checkboard_test.go > TestCheckBoard_GroupedOrder_Preserved` | ✅ COMPLIANT |
| Live Check Board | Owned tool in group | `group_test.go > TestGroupOrder_OwnedToolGroupedUnderManager` + `TestGroupOrder_PerPlatformResolution` + `TestGroupByOwner_PerPlatformBuckets` + `TestCheckBoard_GroupedOrder_Preserved` | ✅ COMPLIANT |
| Live Check Board | Per-tool completion flip | `checkboard_test.go > TestCheckBoard_Complete_AvailableFlipsOnlyTargetLine` | ✅ COMPLIANT |
| Live Check Board | Settled board gates selector | `update_test.go > TestRunUpdate_SelectorGateMatrix` + `TestRunUpdate_NoPendingSkipsSelector` | ✅ COMPLIANT |
| Live Check Board | Atomic concurrent rendering | `checkboard_test.go > TestCheckBoard_ConcurrentComplete_SerializesUpdates` | ✅ COMPLIANT |
| Live Check Board | Non-color fallback | `checkboard_test.go > TestCheckBoard_NonColorFallback_OnePlainLinePerCompletion` | ✅ COMPLIANT |
| Interactive Update Tool Selection | Selector groups pending | `group_test.go > TestOwnerGroupLabel_OwnedToolReturnsManagerLabel` + `TestGroupOrder_CustomToolWithInjectedManager` + `selector_test.go > TestSelector_RenderGroupHeader` | ✅ COMPLIANT |
| Interactive Update Tool Selection | Owned tool in group | `group_test.go > TestOwnerGroupLabel_OwnedToolReturnsManagerLabel` + `TestOwnerGroupLabel_FilteredManagerReturnsEmpty` + `TestGroupOrder_PerPlatformResolution` + `selector_test.go > TestSelector_GroupSelectionPreservesOrder` | ✅ COMPLIANT |
| Interactive Update Tool Selection | Bypass unchanged | `update_test.go > TestRunUpdate_SelectorGateMatrix` | ✅ COMPLIANT |
| Interactive Update Tool Selection | Not a security confirmation | `update_test.go > TestRunUpdate_InteractiveSelection` (ConfirmAction prompt for selected custom tool) | ✅ COMPLIANT |

#### platform-detection (6/6 COMPLIANT)
| Requirement | Scenario | Test | Result |
|---|---|---|---|
| Tool Catalog | Linux tool lookup | `registry_test.go > TestAdaptersForPlatformLinux` | ✅ COMPLIANT |
| Tool Catalog | macOS tool exclusion | `registry_test.go > TestAdaptersForPlatformMacOS` | ✅ COMPLIANT |
| Tool Catalog | Windows tool lookup | `registry_test.go > TestAdaptersForPlatformWindows` | ✅ COMPLIANT |
| Tool Catalog | gh owner on macOS | `parity_test.go > TestCatalogOwnershipMatchesAdapter` + `TestResolveOwner (gh-macos)` | ✅ COMPLIANT |
| Tool Catalog | docker owner on Linux | `parity_test.go > TestCatalogOwnershipMatchesAdapter` + `TestResolveOwner (docker-linux)` | ✅ COMPLIANT |
| Tool Catalog | go no owner on Linux | `parity_test.go > TestCatalogOwnershipMatchesAdapter` + `TestResolveOwner (go-linux-standalone)` | ✅ COMPLIANT |

#### config-system (6/6 COMPLIANT)
| Requirement | Scenario | Test | Result |
|---|---|---|---|
| Config Format | Custom with manager | `config_test.go > TestValidate_ValidManager` + `integration_test.go > TestBuildAdapterList_ThreadsCustomManager` | ✅ COMPLIANT |
| Config Format | Valid TOML | `config_test.go > TestLoadValidTOML` | ✅ COMPLIANT |
| Config Format | Invalid TOML | `config_test.go > TestLoadInvalidTOML` | ✅ COMPLIANT |
| Config Format | Missing fields | `config_test.go > TestLoadPartialConfig_CatalogDefaults` | ✅ COMPLIANT |
| Config Format | Unknown manager | `config_test.go > TestValidate_UnknownManagerIgnoredWarn` + `TestValidate_NonManagerWarning` + `integration_test.go > TestBuildAdapterList_UnknownManagerStaysStandalone` | ✅ COMPLIANT |
| Config Format | Init hygiene (never writes `manager`) | `config_test.go > TestSave_NeverWritesManager` | ✅ COMPLIANT |

#### security-model (6/6 COMPLIANT)
| Requirement | Scenario | Test | Result |
|---|---|---|---|
| Official Tool Integrity | Official brew (runs `brew update` only) | `update_test.go > TestUpdate (brew/success)` | ✅ COMPLIANT |
| Official Tool Integrity | Linux docker delegates (no hardcoded) | `update_test.go > TestUpdateDelegation (docker/linux-delegates-to-apt)` | ✅ COMPLIANT |
| Official Tool Integrity | macOS gh delegates (no hardcoded) | `update_test.go > TestUpdateDelegation (gh/macos-delegates-to-brew)` | ✅ COMPLIANT |
| Official Tool Integrity | Self-update mismatch → abort | `selfupdate/update_test.go > VerifyChecksum (mismatch)` (pre-existing, passes) | ✅ COMPLIANT |
| Official Tool Integrity | Self-update missing entry → abort | `selfupdate/update_test.go > VerifyChecksum (missing entry)` (pre-existing, passes) | ✅ COMPLIANT |
| Official Tool Integrity | Self-update HTTPS-only | `selfupdate/client_test.go > (redirect off https fails closed)` (pre-existing, passes) | ✅ COMPLIANT |

**Compliance summary**: 60/60 scenarios compliant (was 56/60 with 4 PARTIAL), 12/12 requirements present, 0 failing tests.

### Correctness (Static Evidence)
| Requirement | Status | Notes |
|---|---|---|
| Tool Ownership Declaration | ✅ Implemented | `ToolInfo.Kind`/`Manager`, 12 `Info()` declarations, catalog `ToolEntry` mirror. |
| Manager Owned-Tool Cardinality | ✅ Implemented | Derived from owner declarations (`TestManagerOwnedToolCardinality`, `OwnerMetadata()`). |
| Resolved Owner Update Delegation | ✅ Implemented | gh/docker/go `Update()` delegate via `ResolveOwner` + `runtimeGOOSToPlatform`. |
| Adapter Interface | ✅ Implemented | 4 ops + `ToolInfo` owner fields. |
| Official Adapter Catalog | ✅ Implemented | gh/docker→apt/brew/winget; go→brew/winget; apt self-only; no hardcoded manager cmds. |
| Update Gating | ✅ Implemented | `resolveEffectiveUpdatePolicy` inherits owner policy; owned policy INERT. |
| List Table Output | ✅ Implemented | `GroupByOwner`+`ListTools` grouped; `--only/--skip` filter before grouping. |
| Live Check Board | ✅ Implemented | `GroupOrder` (cli/update.go:303) now 100% covered; grouped board order pinned. |
| Interactive Update Tool Selection | ✅ Implemented | `OwnerGroupLabel` (cli/update.go:323) now 100% covered; selector header wiring pinned. |
| Tool Catalog | ✅ Implemented | `ToolEntry.Kind`/`Manager`, `IsManager`. |
| Config Format | ✅ Implemented | `CustomTool.Manager`, `Validate` variadic warns+ignores unknown. |
| Official Tool Integrity | ✅ Implemented | Owned tools delegate; no arbitrary user cmds; fail-closed verify. |

### Coherence (Design)
| Decision | Followed? | Notes |
|---|---|---|
| Option A: `Manager map[platform]string` + `Kind` enum | ✅ Yes | `interface.go` `Kind`/`Manager`. |
| `ToolInfo` canonical; catalog display copy; parity pins both | ✅ Yes | `TestCatalogOwnershipMatchesAdapter`. |
| Owned-tool `Update()` delegates to manager | ✅ Yes | gh/docker/go `Update()` → `ResolveOwner` → owner `Update()`. |
| Owned-tool gating inherits manager policy (INERT own) | ✅ Yes | `resolveEffectiveUpdatePolicy` in `cli/update.go`. |
| Grouping display-only; filter before grouping | ✅ Yes | `list.go`/`update.go` filter per-ID before `GroupByOwner`/`GroupOrder`. |
| WU3 deviation: checkboard uses manager OWN row as group anchor (no separate text header), preserving NewCheckBoard signature + index-slot completion | ✅ Acceptable | Satisfies grouping intent; stable line order preserved (`TestCheckBoard_GroupedOrder_Preserved`). |
| Config `CustomTool.Manager` (toml omitempty); valid known manager only | ✅ Yes | `config.go` Validation + `checkrun.go` `buildAdapterList` resolve via `official.AdapterByName`. |

### Remediation Verification (the reason for this re-run)
The first verify reported FAIL (56/60, 4 PARTIAL) purely because `output.GroupOrder` and `output.OwnerGroupLabel` had 0% dedicated coverage — the production functions that drive WU3 interactive grouping wiring. Remediation added `internal/output/group_test.go` (10 tests) without touching production `group.go`. Re-run evidence:
- `GroupOrder` → **100%** (was 0%): `TestGroupOrder_OwnedToolGroupedUnderManager`, `TestGroupOrder_ManagersFollowCanonicalAllAdaptersOrder`, `TestGroupOrder_PerPlatformResolution` (macos/linux/windows), `TestGroupOrder_FilteredManagerFallsToStandalone`, `TestGroupOrder_CustomToolWithInjectedManager`.
- `OwnerGroupLabel` → **100%** (was 0%): `TestOwnerGroupLabel_OwnedToolReturnsManagerLabel`, `TestOwnerGroupLabel_StandaloneToolReturnsEmpty`, `TestOwnerGroupLabel_FilteredManagerReturnsEmpty`, `TestOwnerGroupLabel_CustomToolWithInjectedManager`.
- `GroupByOwner` → **100%**: `TestGroupByOwner_PerPlatformBuckets` (macos/linux/windows), `TestGroupByOwner_CustomToolBucketedUnderManager`.
- Display-only confirmed: `TestGroupOrder_FilteredManagerFallsToStandalone` proves a filtered-out manager yields no phantom group and the owned tool falls to standalone, preserving `--only`/`--skip` round-trip. `TestGroupByOwner_FilteredManagerNoPhantomHeader` + `TestListCommand_FilterRoundTrip_GroupingDisplayOnly` pin the list-side round-trip.
- Interactive wiring confirmed real: `cli/update.go:303` calls `output.GroupOrder(filteredAdapters, osName)` for the board, and `cli/update.go:323` calls `output.OwnerGroupLabel(grouped[i], osName, grouped)` for the selector group header. All four previously-PARTIAL scenarios now have direct passing covering tests on that production wiring.

### Issues Found
**CRITICAL**: None.

**WARNING**:
1. No `apply-progress.md` exists in the change folder → no TDD Cycle Evidence table, so Strict-TDD RED/GREEN per-task provenance cannot be independently audited from the artifact. Runtime evidence (all gates green, all scenario tests pass) is the substitute. (This is a process/documentation gap, not a spec-scenario compliance failure; the prior baseline graded it WARNING, which is maintained.)
2. `platform.IsManager` shows 0% in-package coverage but is exercised cross-package (config `Validate` unknown/non-manager tests) at 100% with `-coverpkg`. Informational; config threshold 0.
3. `cli.resolvingOwner` 50%, `output.ownerIDOf` 80% — partial helper coverage, but the custom-adapter owner branch is now directly exercised at the grouping layer (`TestGroupOrder_CustomToolWithInjectedManager`, `TestOwnerGroupLabel_CustomToolWithInjectedManager`, `TestGroupByOwner_CustomToolBucketedUnderManager`) and adapter layer (`TestCustomAdapter_Update_DelegatesToManager`) and CLI layer (`TestBuildAdapterList_ThreadsCustomManager`).

**SUGGESTION**:
1. Optionally add an apply-progress.md with a TDD Cycle Evidence table to make Strict-TDD provenance fully auditable (not required for archive; runtime gates are green).
2. Optionally add a CLI integration test driving the `upp update` interactive path with a manager+owned fake pair to close the WU3 end-to-end grouping gap beyond the unit wiring (the unit wiring now has direct coverage).

### Verdict
PASS WITH WARNINGS — the WU3 interactive grouping coverage CRITICAL is resolved: `GroupOrder`/`OwnerGroupLabel`/`GroupByOwner` are 100% covered by direct passing tests, all 60/60 scenarios are compliant (was 56/60 with 4 PARTIAL), all source gates green (test, race, vet, gofmt, smoke 31/31), 0 failing tests, 21/21 tasks complete. Remaining items are WARNING/SUGGESTION only (no apply-progress TDD evidence table; partial helper coverage that is actively exercised; informational coverage-threshold 0). Archive is appropriate.

---

### TDD Compliance
| Check | Result | Details |
|---|---|---|
| TDD Evidence reported | ❌ | No `apply-progress.md` in change folder → no TDD Cycle Evidence table. |
| All tasks have tests | ⚠️ | 21/21 tasks complete; per-task RED/GREEN evidence not auditable (no apply-progress). |
| RED confirmed (tests exist) | ⚠️ | Test files exist and pass; per-task RED provenance unverifiable. |
| GREEN confirmed (tests pass) | ✅ | `go test ./... -count=1 -race` exit 0; all test files pass. |
| Triangulation adequate | ✅ | WU3 grouping wiring (GroupOrder/OwnerGroupLabel) now multi-scenario triangulated (ownership, per-platform, filtered-manager, custom-injected). |
| Safety Net for modified files | ⚠️ | N/A (no apply-progress) |

**TDD Compliance**: 3/6 checks fully confirmed (GREEN, triangulation, runtime gates); 3 ⚠️/❌ due to missing apply-progress provenance (documentation gap, not a compliance failure).

---

### Test Layer Distribution
| Layer | Tests | Files | Tools |
|-------|-------|-------|-------|
| Unit | majority | ~21 files | go stdlib |
| Integration | update/cli flows, smoke | `cli/integration_test.go`, `cli/update_test.go`, smoke-test.sh | go stdlib + bash |
| E2E | smoke (upp binary) | `scripts/smoke-test.sh` | bash |
| **Total** | **~319 test funcs** | **~21** | |

---

### Changed File Coverage
| File | Line % | Rating |
|------|--------|--------|
| `internal/adapters/official/ownership.go` | 85.7% | ⚠️ Acceptable |
| `internal/adapters/official/registry.go` | 100% | ✅ Excellent |
| `internal/output/group.go` (GroupOrder 100%, OwnerGroupLabel 100%, GroupByOwner 100%, ownerIDOf 80%) | 100% core | ✅ Excellent |
| `internal/output/render.go` (ListTools 100%) | 100% | ✅ Excellent |
| `internal/output/checkboard.go` | 88–100% | ✅ Excellent |
| `internal/output/selector.go` (Run 95%) | 95% | ✅ Excellent |
| `internal/cli/update.go` (runUpdateInteractive 86%, runUpdate 92%) | 86–92% | ✅ Excellent |
| `internal/cli/list.go` (runList 72%) | 72% | ⚠️ Low |
| `internal/config/config.go` (Validate 91%) | 91% | ✅ Excellent |
| `internal/platform/catalog.go` (IsManager 0% in-pkg, 100% cross-pkg) | mixed | ⚠️ Low |

**Coverage analysis**: Informational; config threshold 0. The primary gap (`GroupOrder`/`OwnerGroupLabel` 0%) is now closed at 100%.

---

### Assertion Quality
**Assertion quality**: ✅ All assertions verify real behavior — no tautologies, no type-only-assertion-only tests, no ghost loops over possibly-empty collections. The new `group_test.go` assertions compare concrete ordered slices (`want` arrays) and exact display labels ("Homebrew", "APT Package Manager"), and the filtered-manager tests assert the negative (empty label / unchanged order) to prove no phantom group.

---

### Quality Metrics
**Linter**: ➖ not invoked (golangci-lint optional; `go vet` clean)
**Type Checker**: ✅ No errors (`go vet ./...` exit 0)
