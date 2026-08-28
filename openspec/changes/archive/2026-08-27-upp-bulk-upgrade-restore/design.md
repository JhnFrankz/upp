# Design: Restore Bulk-Upgrade per-Manager (vision point 6 deferred)

## Technical Approach

Implement the safe base + opt-in structure from the proposal. Point 6 shipped self-only delegation (gh/docker/go delegate to the manager's own Update()), so group tools are inert. This increment adds (1) a per-manager package-name mapping so an owned tool's real package is known, (2) a per-owned-tool availability check so owned tools become genuinely pending, (3) `--manager <mgr>` / `--update-group <mgr>` opt-in flags that run a manager-group bulk update, (4) a confirm-action reclassification so a sudo-heavy group prompts despite TrustOfficial, and (5) group gate inheritance from the manager's policy. Default upp update is unchanged (bulk default deferred). Consistent with Option A + Option C in the exploration.

## Architecture Decisions

| # | Decision | Options | Choice | Rationale |
|---|----------|---------|--------|-----------|
| D1 | Package-name storage | (A) ManagerPackage map[string]string on ToolInfo; (B) separate owner-package table | A | Mirrors the existing Manager map shape; platform->package unambiguous. Mirror on catalog.ToolEntry like Manager so the parity test pins both registries. |
| D2 | Availability check location | (A) owned tool's own Check(); (B) PackageChecker.CheckPackage(pkg) on manager adapters + delegated Check() | B | The package-system query (apt-cache policy, brew outdated, winget upgrade) belongs next to its manager. One helper serves both the owned-tool Check() and the group path. |
| D3 | Group UX wiring | (A) feed group through per-tool selector; (B) dedicated runUpdateGroup path | B | Bulk is explicit and opt-in; per-tool toggles add complexity. --manager takes precedence over the interactive gate. |
| D4 | Confirm reclassification | (A) ConfirmConfig.EnforceRisk bool; (B) misuse TrustCustomTrusted; (C) reclassify row's trust | A | EnforceRisk=false (default) keeps every existing decision byte-identical; only the group path sets true, bypassing the TrustOfficial->ConfirmAuto short-circuit so real command risk decides. |
| D5 | Gating inheritance | Reuse resolveEffectiveUpdatePolicy on the manager adapter | A | A group's gate IS the manager's policy. PolicyGated group gates on group availability; PolicyAlwaysUpdate runs unconditionally. |

## Data Model / Interfaces

```go
// internal/adapters/interface.go
type ToolInfo struct {
    Manager        map[string]string // platform -> owning manager ID
    ManagerPackage map[string]string // platform -> package name under that platform's manager
}
// PackageChecker implemented by manager adapters (apt/brew/winget)
type PackageChecker interface {
    CheckPackage(packageName string) (UpdateInfo, error)
}
```

Mapping: gh=gh|gh|gh; docker=docker-ce|docker|Docker.Docker; go=(standalone on Linux)|golang|GoLang.Go.

## Data Flow

`upp update --manager apt` (Linux) -> runUpdate with uf.Manager set -> runUpdateGroup(apt): enumerate KindTool AND Manager[linux]=="apt" -> {gh, docker}; drop --skip; gate (apt PolicyGated -> check group availability via CheckPackage); confirm (EnforceRisk:true, sudo -> RiskHigh prompt / --ci fails); per tool apt.Update(pkg) via manager privileged executor; render group bulk summary.

## File Changes

Modify internal/adapters/interface.go (ManagerPackage + PackageChecker); internal/adapters/official/{gh,docker,go}.go (ManagerPackage + Check delegate); internal/adapters/official/{apt,brew,winget}.go (CheckPackage impl); internal/adapters/official/parity_test.go (golden mapping parity); internal/platform/catalog.go (mirror ManagerPackage on ToolEntry); internal/cli/parser.go (UpdateFlags.Manager + --manager/--update-group alias); internal/cli/update.go (branch to runUpdateGroup); internal/security/confirm.go (ConfirmConfig.EnforceRisk); internal/output/render.go + group.go (group bulk summary); confirm_test.go/update_test.go/render_test.go (new tests).

## Testing Strategy

Unit: mapping parity golden table; apt-cache policy candidate>installed via setExecFakes; brew outdated available/current/non-zero; --skip excludes owned tool; opt-in flag parse; ConfirmConfig.EnforceRisk reclassify; gating group. E2E: go test ./... -count=1 green + smoke.

## Threat Matrix

N/A - no VCS/PR automation, git selection, commit/push state, PR command, or executable-file-classification boundary. Elevated-sudo risk governed by security-model confirm reclassification (D4), specced and tested.

## Migration / Rollout

No migration. ManagerPackage additive field; absent entries fail-closed (owned tool skipped from group batch, never guessed). Config key not added this increment.

## Open Questions

None blocking. `brew outdated` output-shape and `winget upgrade <pkg>` exit semantics re-confirm against live systems during WU2.

## Work-Unit Split (chained PRs, each < 400 lines)

WU1 ToolInfo.ManagerPackage + catalog mirror + parity test (~130, safe base). WU2 PackageChecker + CheckPackage on apt/brew/winget + owned Check() delegate + fixtures (~150). WU3 --manager/--update-group flags + runUpdateGroup (~180). WU4 ConfirmConfig.EnforceRisk + ConfirmAction reclassify + security tests (~90). WU5 group bulk summary rendering + output tests (~80). Default delivery strategy ask-on-risk; forecast chained PRs recommended: Yes, 400-line budget risk: Medium (WU2+WU3 combined would exceed; keep separate PRs).
