# Tasks: Tool→Manager Ownership Model + Grouping

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~420 (WU1 120 + WU2 110 + WU3 130 + WU4 60) |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | PR1 WU1 → PR2 WU2 → PR3 WU3 → PR4 WU4 |
| Delivery strategy | ask-on-risk |
| Chain strategy | feature-branch-chain |

Decision needed before apply: Yes
Chained PRs recommended: Yes
Chain strategy: feature-branch-chain
400-line budget risk: High

### Suggested Work Units

| Unit | Goal | PR | Focused test | Runtime harness | Rollback |
|------|------|----|--------------|-----------------|----------|
| 1 | Kind/Manager+owners | PR1 | `go test ./internal/adapters/official/... -run 'Info\|Parity\|Registry\|Owner' -count=1` | N/A (static) | Revert Kind/Manager, Info(), catalog.go |
| 2 | Delegated update+gating | PR2 | `go test ./internal/adapters/official/... -run 'Update\|Delegat\|Gate' -count=1` | `go test ./internal/cli/... -run 'Update\|Gate' -count=1` | Restore gh/docker/go cmds |
| 3 | Grouping render+wiring | PR3 | `go test ./internal/output/... -run 'Group\|Render\|Selector\|Board' -count=1` | N/A (no TTY) | Revert grouping + wiring |
| 4 | Config+validation | PR4 | `go test ./internal/config/... -run 'Custom\|Manager' -count=1` | `bash scripts/smoke-test.sh --skip-build` | Drop `manager`; revert buildAdapterList |

Feature Branch Chain (tracker `tool/ownership-model`): PR1 base=tracker; PR2 base=PR1; PR3 base=PR2; PR4 base=PR3.

## WU1 — Data model + ownership (RED→GREEN→REFACTOR)

- [x] 1.1 RED: extend `info_test.go` golden w/ Kind+Manager; `registry_test` consistency + `ResolveOwner` table.
- [x] 1.2 GREEN: `internal/adapters/interface.go` add `Kind` (`KindTool`=0,`KindManager`) + `Manager`.
- [x] 1.3 GREEN: create `internal/adapters/official/ownership.go` — `ResolveOwner(tool, os)` reads `Manager[os]`.
- [x] 1.4 GREEN: declare all 12 `Info()` — gh/docker `{linux:apt,macos:brew,windows:winget}`, go `{macos:brew,windows:winget}`; apt/brew/winget/scoop `KindManager`; others `KindTool`.
- [x] 1.5 GREEN: `internal/platform/catalog.go` `ToolEntry` + `Kind`/`Manager`.
- [x] 1.6 GREEN: `internal/adapters/official/registry.go` add `OwnerMetadata()`.
- [x] 1.7 REFACTOR: `gofmt -s -w` + `go vet ./internal/adapters/...`.

## WU2 — Delegated update + gating

- [x] 2.1 RED: `update_test.go` delegation — docker(Linux) skips when apt no update; gh(macOS) under brew; go-Linux standalone.
- [x] 2.2 GREEN: gh/docker/go `Update` resolve owner via `ResolveOwner(id, runtime.GOOS)`; nil → existing cmd (go-Linux); else `return owner.Update(dryRun)`.
- [x] 2.3 GREEN: `internal/adapters/custom.go` `CustomAdapter`+`NewCustomAdapter` carry `manager`; delegate to owner.
- [x] 2.4 GREEN: `internal/cli/update.go` gates resolve effective policy from owner; owned policy INERT.
- [x] 2.5 REFACTOR: `gofmt`; `go test ./internal/adapters/official/... ./internal/cli/... -count=1`.

## WU3 — Grouping rendering + wiring

- [x] 3.1 RED: `render_test.go`/`selector_test.go`/`checkboard_test.go` grouped — header then child rows; standalone after; filters round-trip.
- [x] 3.2 GREEN: `internal/output/render.go` — `Group` type + `GroupByOwner(adapters, os) []Group`; `ListTools` header + children.
- [x] 3.3 GREEN: `internal/output/selector.go` `SelectOption.Group` + header; `checkboard.go` grouped order (slot logic unchanged).
- [x] 3.4 GREEN: `internal/cli/list.go` build `[]Group` → `ListTools`; `update.go` names/pending in group order.
- [x] 3.5 REFACTOR: confirm `--only`/`--skip` filter before grouping unchanged.

## WU4 — Config + spec sync

- [x] 4.1 RED: `config_test.go` — valid `manager`, unknown ignored+warn, init never writes `manager`.
- [x] 4.2 GREEN: `internal/config/config.go` `CustomTool.Manager` (`toml:"manager,omitempty"`); `Validate` warns on unknown.
- [x] 4.3 GREEN: thread `manager` through `buildAdapterList` → `NewCustomAdapter`.
- [x] 4.4 VERIFY: `go test ./... -count=1` green.
