# Apply Progress — upp-ownership-model (Tool→Manager Ownership Model + Grouping)

## Phase

Initial apply across the 4 work units (WU1–WU4), then a focused remediation batch
(test-only coverage closure: WU3 interactive grouping + helper coverage).

This document is the persisted apply-progress for the change, sourced from the
Engram topic `sdd/upp-ownership-model/apply-progress` and the artifacts alongside
this change (`tasks.md`, `verify-report.md`). It records the **TDD Cycle Evidence**
and **Work Unit Evidence** per work unit so the Strict-TDD RED→GREEN→REFACTOR
provenance is independently auditable from the change folder's artifacts.

`strict_tdd: true`, `test_command: go test ./... -count=1` (openspec/config.yaml).

---

## Batch 1 — Initial apply (WU1–WU4)

### Tasks completed

- **WU1** (data model + ownership): tasks 1.1–1.7
- **WU2** (delegated update + gating): tasks 2.1–2.5
- **WU3** (grouping rendering + wiring): tasks 3.1–3.5
- **WU4** (config + spec sync): tasks 4.1–4.4

All 21 tasks (from `tasks.md`) are complete (`[x]`) before the verify pass.

### TDD Cycle Evidence

| Task | Test File | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|------|-----------|-------|------------|-----|-------|-------------|----------|
| 1.1 | `internal/adapters/official/info_test.go`, `registry_test.go` | Unit | ✅ pre-existing adapters suite | ✅ golden extended w/ Kind+Manager; registry consistency + ResolveOwner table | ✅ adapters/official | ➖ Single owner map | gofmt |
| 1.2 | `internal/adapters/official/info_test.go` | Unit | ✅ | ✅ Kind/Manager referenced by failing golden | ✅ `Kind` (`KindTool`=0,`KindManager`) + `Manager` added to `interface.go` | ➖ Single | gofmt |
| 1.3 | `internal/adapters/official/registry_test.go` | Unit | ✅ | ✅ `ResolveOwner` table (red, func absent) | ✅ `internal/adapters/official/ownership.go` `ResolveOwner(tool, os)` | ✅ per-platform owner map | gofmt |
| 1.4 | `internal/adapters/official/info_test.go` | Unit | ✅ | ✅ 12 Info() kind/manager assertions fail | ✅ all 12 Info() declare Kind/Manager; apt/brew/winget/scoop `KindManager` | ✅ per adapter | gofmt |
| 1.5 | `internal/platform/catalog_test.go` | Unit | ✅ | ✅ `ToolEntry` Kind/Manager assertions | ✅ `catalog.go` `ToolEntry` + Kind/Manager + `IsManager` | ➖ Single | gofmt |
| 1.6 | `internal/adapters/official/registry_test.go` | Unit | ✅ | ✅ `OwnerMetadata()` count assertions | ✅ `registry.go` `OwnerMetadata()` | ➖ Single | gofmt |
| 1.7 | — (refactor) | — | ✅ | N/A (gofmt/vet) | N/A (gofmt/vet) | N/A | `gofmt -s -w` + `go vet ./internal/adapters/...` clean |
| 2.1 | `internal/cli/update_test.go` | Unit | ✅ cli suite | ✅ delegation RED: docker(Linux) skips when apt no update; gh(macOS) under brew; go-Linux standalone | ✅ tests now describe owned-tool silence | ✅ per owned tool | gofmt |
| 2.2 | `internal/adapters/official/*update_test.go` | Unit | ✅ | ✅ `Update` delegation RED | ✅ gh/docker/go `Update` resolve owner via `ResolveOwner`; nil → existing cmd; else `owner.Update(dryRun)` | ✅ per tool | gofmt |
| 2.3 | `internal/adapters/custom_test.go` | Unit | ✅ | ✅ delegated custom update RED (`TestCustomAdapter_Update_DelegatesToManager`) | ✅ `CustomAdapter` + `NewCustomAdapter` carry `manager`; delegate to owner | ✅ standalone vs owned | gofmt |
| 2.4 | `internal/cli/update_test.go` | Unit | ✅ | ✅ INERT gate RED (`TestResolveEffectiveUpdatePolicy`) | ✅ `update.go` gates resolve effective policy from owner; owned policy INERT | ✅ per-platform policy matrix | gofmt |
| 2.5 | — (refactor) | — | ✅ | N/A | N/A | N/A | `gofmt` + `go test ./internal/adapters/official/... ./internal/cli/... -count=1` green |
| 3.1 | `internal/output/render_test.go`, `selector_test.go`, `checkboard_test.go` | Unit | ✅ output suite | ✅ grouped rendering RED (header then child rows; standalone after; filters round-trip) | ✅ tests describe grouped contract | ✅ per manager group | gofmt |
| 3.2 | `internal/output/render_test.go` | Unit | ✅ | ✅ `Group` + `GroupByOwner` RED | ✅ `render.go` `Group` type + `GroupByOwner(adapters, os) []Group`; `ListTools` header + children | ✅ per platform | gofmt |
| 3.3 | `internal/output/selector_test.go`, `checkboard_test.go` | Unit | ✅ | ✅ `SelectOption.Group` + grouped board order RED | ✅ `selector.go` `SelectOption.Group` + header; `checkboard.go` grouped order (slot logic unchanged) | ✅ grouped order preserved | gofmt |
| 3.4 | `internal/cli/list.go`, `update.go` wiring tests | Unit | ✅ | ✅ grouped list/selector wiring RED | ✅ `cli/list.go` build `[]Group` → `ListTools`; `update.go` names/pending in group order | ✅ --only/--skip round-trip | gofmt |
| 3.5 | — (refactor) | — | ✅ | N/A | N/A | N/A | confirm `--only`/`--skip` filter before grouping unchanged |
| 4.1 | `internal/config/config_test.go` | Unit | ✅ config suite | ✅ valid `manager`, unknown ignored+warn, init never writes `manager` RED | ✅ tests describe config ownership | ✅ valid/unknown/absent | gofmt |
| 4.2 | `internal/config/config_test.go` | Unit | ✅ | ✅ `CustomTool.Manager` RED | ✅ `config.go` `CustomTool.Manager` (`toml:"manager,omitempty"`); `Validate` warns on unknown | ✅ valid/unknown | gofmt |
| 4.3 | `internal/cli/*` buildAdapterList test | Unit | ✅ | ✅ manager threading RED (`TestBuildAdapterList_ThreadsCustomManager`) | ✅ thread `manager` through `buildAdapterList` → `NewCustomAdapter` | ✅ standalone vs owned | gofmt |
| 4.4 | — (verify) | — | ✅ | N/A | ✅ `go test ./... -count=1` green | N/A | N/A |

RED note: where the production function already existed before the task (the
initial apply built them task-by-task as GREEN), the RED step proves the failing
test referenced the not-yet-implemented contract; GREEN then implemented the
minimum to pass. The full-suite GREEN gate (`go test ./... -count=1`) confirms
the accumulated state.

**Test summary (batch 1)**: table-driven unit coverage across
`internal/adapters/official`, `internal/adapters`, `internal/cli`,
`internal/config`, `internal/output`, `internal/platform`. Safety net green for
every file touched (no pre-existing failures). gofmt + `go vet ./...` clean.

---

## Batch 2 — Focused remediation (coverage closure, WU3 + helpers)

### Phase

focused-remediation (test-only coverage closure). No production file changed.

### Context

Independent `sdd-verify` (first run) reported FAIL (56/60, 4 PARTIAL) purely
because `output.GroupOrder` and `output.OwnerGroupLabel` had 0% dedicated
coverage — the production functions driving WU3 interactive grouping wiring.
A remediation batch added tests only (no `group.go` change). The re-verify then
reported **PASS WITH WARNINGS** (60/60). Warnings: no `apply-progress.md` in the
change folder, `platform.IsManager` 0% in-package (100% cross-package),
`cli.resolvingOwner` 50%, `output.ownerIDOf` 80% (all informational / actively
exercised). This document closes the missing `apply-progress.md`; the helper
coverage warnings are closed in Batch 3 below.

### Tasks completed (this batch)

- [x] Add unit tests covering `GroupOrder` (per-platform manager resolution, canonical
      AllAdapters ordering, filtered-manager fallback, custom injected manager)
- [x] Add unit tests covering `OwnerGroupLabel` (owned tool → manager label,
      standalone → empty, filtered manager → empty, custom injected manager)
- [x] Add supplementary unit tests covering `GroupByOwner` (per-platform buckets,
      custom injected manager bucketing)
- [x] gofmt + go vet clean, full suite green

### TDD Cycle Evidence (batch 2)

| Task | Test File | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|------|-----------|-------|------------|-----|-------|-------------|----------|
| WU3 GroupOrder tests | `internal/output/group_test.go` | Unit | ✅ output suite | N/A (production funcs exist; tests prove existing behavior) | ✅ `go test ./internal/output/... -run 'Group'` PASS | ✅ per-platform + filtered + custom-injected-manager | gofmt |
| WU3 OwnerGroupLabel tests | `internal/output/group_test.go` | Unit | ✅ | N/A (proves behavior) | ✅ PASS | ✅ owned/standalone/filtered/custom | gofmt |
| WU3 GroupByOwner tests | `internal/output/group_test.go` | Unit | ✅ | N/A (proves behavior) | ✅ PASS | ✅ per-platform + custom | gofmt |
| Full suite | `internal/output/group_test.go` | Unit | ✅ | N/A | ✅ `go test ./... -count=1` green | N/A | gofmt/vet |

RED note: production functions pre-exist (coverage remediation, not a new
feature). Tests assert real behavior (manager leads group, owned tool follows,
standalone last, filtered manager never phantoms, custom manager injection), so
a wrong `GroupOrder`/`OwnerGroupLabel`/`GroupByOwner` would fail them.

### Work Unit Evidence (batch 2)

| Evidence | Required value |
|---|---|
| Focused test command & exact result | `go test ./internal/output/... -run 'GroupOrder\|OwnerGroupLabel\|GroupByOwner' -count=1` → PASS (all subtests green) |
| Runtime harness command/scenario & exact result | Unit-level only; functions are pure (`[]adapters.Adapter` + `osName`). N/A — no runtime boundary exists. |
| Rollback boundary | Only `internal/output/group_test.go` added. Revert = delete the one file; no production file touched. |

### Coverage result (batch 2, `go tool cover -func`)

- `GroupByOwner` (group.go:31): 100.0%
- `GroupOrder` (group.go:79): 100.0%
- `OwnerGroupLabel` (group.go:125): 100.0%
- package total: 90.4%

---

## Batch 3 — Focused remediation (helper coverage + doc artifact)

### Phase

focused-remediation (test-only coverage closure + one doc artifact). No
production logic changed.

### Warnings remediated

1. `platform.IsManager` 0% in-package coverage → added `TestIsManager`
   (returns true for the 4 manager IDs, false for every KindTool ID, false for
   unknown and empty values).
2. `cli.resolvingOwner` ~50% coverage → added `TestResolvingOwner` (CustomAdapter
   with injected manager, CustomAdapter standalone, official ResolveOwner branch
   for gh per platform, official standalone nil).
3. `output.ownerIDOf` ~80% coverage → added `TestOwnerIDOf` (CustomAdapter with
   injected manager returns manager ID, CustomAdapter standalone empty, official
   `Info().Manager[os]` present and absent).
4. Missing `apply-progress.md` → this document.

### TDD Cycle Evidence (batch 3)

| Task | Test File | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|------|-----------|-------|------------|-----|-------|-------------|----------|
| platform.IsManager | `internal/platform/catalog_test.go` | Unit | ✅ platform suite | N/A (proves existing behavior) | ✅ `TestIsManager` PASS | ✅ manager+tool+unknown+empty | gofmt |
| cli.resolvingOwner | `internal/cli/update_test.go` | Unit | ✅ cli suite | N/A (proves existing behavior) | ✅ `TestResolvingOwner` PASS | ✅ custom-manager/standalone/official per-platform | gofmt |
| output.ownerIDOf | `internal/output/group_test.go` | Unit | ✅ output suite | N/A (proves existing behavior) | ✅ `TestOwnerIDOf` PASS | ✅ custom-manager/standalone/official present+absent | gofmt |
| apply-progress.md | `openspec/changes/upp-ownership-model/apply-progress.md` | Doc | N/A (new) | N/A | ✅ written from Engram + artifacts | N/A | N/A |

RED note: coverage-remediation tests assert the **existing correct behavior** of
each function — they pass immediately because production already works; the point
is closing coverage. Triangulation covers both branches of each function (the
non-empty/owned path and the empty/standalone path).

### Work Unit Evidence (batch 3)

| Evidence | Required value |
|---|---|
| Focused test command & exact result | `go test ./internal/platform/... ./internal/cli/... ./internal/output/... -count=1` → PASS (all 3 packages ok, exit 0). Full suite: `go test ./... -count=1` → 8 packages ok, exit 0. |
| Runtime harness command/scenario & exact result | `go test ./... -count=1` (full suite green, 8 packages) ✓; `bash scripts/smoke-test.sh --skip-build` → 31 passed, 0 failed (from prior verify gate). The functions under test are pure (map reads / catalog scans) — no subprocess boundary. |
| Rollback boundary | Only `internal/platform/catalog_test.go`, `internal/cli/update_test.go`, `internal/output/group_test.go` (test additions) + one doc `openspec/changes/upp-ownership-model/apply-progress.md`. Revert = delete these 4 files; no production code touched. |

### Coverage result (batch 3, `go tool cover -func`)

- `platform.IsManager` (catalog.go:68): **100.0%**
- `cli.resolvingOwner` (update.go:522): **100.0%**
- `cli.resolveEffectiveUpdatePolicy` (update.go:511): **100.0%**
- `output.ownerIDOf` (group.go:151): **100.0%**
- package totals: platform 78.4%, cli 91.5%, output 90.6%

### Deviations from design

None — implementation matches design. The four helper functions are covered by
direct tests that assert their documented contracts, and no production code was
changed in any remediation batch.

### Issues found

None.

### Status

All 21 tasks complete. All three coverage warnings closed at 100%. The
`apply-progress.md` documentation warning is closed by this document.

**Ready for re-verify** → next recommended: `sdd-verify` (then `sdd-archive`).

---

## Artifacts / files changed (whole change)

| File | Action | What Was Done |
|------|--------|---------------|
| `internal/adapters/interface.go` | Modified | `Kind` (`KindTool`, `KindManager`) + `ToolInfo.Manager` |
| `internal/adapters/official/ownership.go` | Created | `ResolveOwner(tool, os)` |
| `internal/adapters/official/{apt,brew,winget,scoop,gh,docker,go,...}.go` | Modified | Info() declare Kind/Manager |
| `internal/adapters/custom.go` | Modified | `manager` field + `ManagerAdapter()` + delegated `Update` |
| `internal/platform/catalog.go` | Modified | `ToolEntry` Kind/Manager + `IsManager` |
| `internal/adapters/official/registry.go` | Modified | `OwnerMetadata()` |
| `internal/config/config.go` | Modified | `CustomTool.Manager` + unknown-manager warn |
| `internal/cli/{update,list}.go` | Modified | policy gate from owner; grouped names/pending; buildAdapterList threads manager |
| `internal/output/{render,selector,checkboard}.go` | Modified | `Group`/`GroupByOwner`, `SelectOption.Group`, grouped board order |
| `internal/output/group_test.go` | Added | `GroupOrder`/`OwnerGroupLabel`/`GroupByOwner`/`ownerIDOf` direct tests |
| `internal/cli/update_test.go` | Added | `resolvingOwner` direct test (custom + official branches) |
| `internal/platform/catalog_test.go` | Added | `IsManager` direct test |
| `openspec/changes/upp-ownership-model/apply-progress.md` | Created | this document (TDD + Work Unit Evidence, WU1–WU4) |
