# Tasks: RiskCommand From the Real Executed Command

## Phase 1: Adapter Declaration Surface (Work Unit 1)

- [x] 1.1 RED `internal/adapters/official/info_test.go`: table sweep over `official.AllAdapters()` — each `KindManager` declares non-empty `SelfUpdateCommand`; apt/brew/winget/pacman declare `PackageUpdateCommand` with `PackagePlaceholder` (scoop self-only exempt); pacman `Privileges: ["sudo"]`.
- [x] 1.2 GREEN `internal/adapters/interface.go`: add `ToolInfo.SelfUpdateCommand`, `ToolInfo.PackageUpdateCommand`, const `PackagePlaceholder = "<pkg>"`, `RenderPackageCommand` (replace first `<pkg>`, else append `" "+pkg`).
- [x] 1.3 GREEN `internal/adapters/interface_test.go` (new): `RenderPackageCommand` table, placeholder present/absent (T1).
- [x] 1.4 GREEN `internal/adapters/official/{apt,brew,winget,scoop,pacman}.go`: extract executed commands into package constants; `Info()` declares, `Update()`/`UpdatePackage()` run the same constant (D2). Pin pacman `sudo pacman -S --noconfirm <pkg>` / `sudo pacman -S --noconfirm pacman`, sudo.
- [x] 1.5 GREEN `internal/adapters/official/update_test.go`: exec-capture via `runCmdFn` seam — `Update(false)`/`UpdatePackage(pkg)` runs byte-exactly the declared constant.

## Phase 2: Plan Derivation + Gate Truth (Work Unit 2)

- [x] 2.1 RED `internal/engine/plan_test.go`: pacman self row → `sudo pacman -S --noconfirm pacman`, sudo, `ManagerID="pacman"` (D4); pacman owned render via synthetic fixture (path unreachable today); custom `manager="pacman"` inherits self command; custom standalone keeps `info.Command`; official standalone fallback unchanged; apt/brew/winget owned pins byte-identical.
- [x] 2.2 GREEN `internal/engine/plan.go`: D3 row-class derivation (self → `SelfUpdateCommand`; owned → `RenderPackageCommand(owner.PackageUpdateCommand, pkg)`; custom delegated → `owner.SelfUpdateCommand`; standalone unchanged); `ManagerID` on self-rows iff declared privileges (D4). `detectPrivileges` untouched.
- [x] 2.3 GREEN `internal/engine/resolve.go`: delete `UpdateCmdName`; drop its tests from `internal/engine/resolve_test.go`.
- [x] 2.4 RED→GREEN `internal/engine/plan_test.go`: `plan.RiskCommand` byte-equals the executed command declaration (`SelfUpdateCommand` / `RenderPackageCommand(PackageUpdateCommand, pkg)`); privileges equal declared. The execution half (declaration == the command Update/UpdatePackage runs, captured through the `runCmdFn` seam) lives in `internal/adapters/official/update_test.go:TestUpdateRunsDeclaredCommand`; the seam is package-private (engine imports official, so official cannot import engine) — the two equalities are composed rather than stubbed cross-package.
- [x] 2.5 RED→GREEN `internal/cli/update_test.go`: privileged-manager row prompts; `--ci` exits non-zero; custom `manager="pacman"` classified by real self command; apt/brew/winget decisions unchanged (spec "Pacman privileged update prompts", "--ci pacman privileged fails").

## Phase 3: Comments, Battery, Archive Handoff

- [x] 3.1 Comment-only: correct stale EnforceRisk doc (`internal/cli/update.go:435-436`, `internal/security/confirm.go:37-41`) to cover ManagerID-marked privileged self-rows (D4). No logic change.
- [x] 3.2 Battery: `gofmt -s -l .` clean, `go vet ./...`, `go test ./... -count=1 -race` green.
- [x] 3.3 E2E `bash scripts/smoke-test.sh --skip-build`; verify: pacman prompts, `--ci` non-zero on pacman rows, apt/brew/winget byte-stable.
- [ ] 3.4 Carry the archive note below into verify/archive.

## Review Workload Forecast

Estimated changed lines: ~420–480. Split: PR 1 = Phase 1 → PR 2 = Phases 2–3. Delivery: auto-chain.

Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: Medium

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|----------------------|-----------------|-------------------|
| 1 | Adapter declarations: ToolInfo fields, `RenderPackageCommand`, constants, pacman sudo | PR 1 | `go test ./internal/adapters/... -count=1` | N/A — declarations only | Revert PR 1; no consumer changed |
| 2 | D3/D4 derivation, synthesis retirement, byte-equality + gate tests, comments | PR 2 (base: PR 1) | `go test ./internal/engine/... ./internal/cli/... -count=1 -race` | `bash scripts/smoke-test.sh --skip-build` | Revert PR 2; gate inputs restore byte-for-byte |

**Archive note (pre-existing, not a regression):** the delta's "Standard update" scenario (`npm update -g`) was already unmet before this change — `npm.go:68` runs `npm update -g`, no adapter declares `Command`. Record as pre-existing; npm untouched here.
