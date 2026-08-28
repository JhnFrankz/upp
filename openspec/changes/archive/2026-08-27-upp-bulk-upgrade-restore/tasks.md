# Tasks: Restore Bulk-Upgrade per-Manager (opt-in)

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~630 |
| 400-line budget risk | Medium |
| Chained PRs recommended | Yes |
| Delivery strategy | ask-on-risk |
| Chain strategy | feature-branch-chain |

Decision needed before apply: Yes
Chained PRs recommended: Yes
Chain strategy: feature-branch-chain
400-line budget risk: Medium

## Suggested Work Units (each < 400 lines, one PR each)

| Unit | Goal | PR |
|------|------|----|
| WU1 | ToolInfo.ManagerPackage + catalog mirror + parity | PR1 |
| WU2 | PackageChecker + CheckPackage apt/brew/winget + owned Check delegate | PR2 |
| WU3 | --manager/--update-group flags + runUpdateGroup | PR3 |
| WU4 | ConfirmConfig.EnforceRisk reclassify | PR4 |
| WU5 | Group bulk summary rendering | PR5 |

## Phase 1 WU1 (RED->GREEN->REFACTOR)

- [x] 1.1 RED parity_test.go every Manager[p] has ManagerPackage[p]
- [x] 1.2 GREEN add ManagerPackage map[string]string to ToolInfo
- [x] 1.3 GREEN declare packages gh/docker/go Info()
- [x] 1.4 GREEN mirror ManagerPackage on ToolEntry
- [x] 1.5 REFACTOR golden table pins both registries

## Phase 2 WU2 (RED->GREEN->REFACTOR)

- [x] 2.1 RED gh.Check() update_available=true when package candidate>installed
- [x] 2.2 GREEN interface.go add PackageChecker interface CheckPackage(pkg)(UpdateInfo,error)
- [x] 2.3 GREEN apt CheckPackage apt-cache policy <pkg>
- [x] 2.4 GREEN brew CheckPackage brew outdated --json <pkg>
- [x] 2.5 GREEN winget CheckPackage winget upgrade <pkg> (generalize parseWingetUpgradeOutput)
- [x] 2.6 GREEN gh/docker/go Check delegate to manager CheckPackage(ManagerPackage[platform])
- [x] 2.7 REFACTOR hermetic setExecFakes fixtures

## Phase 3 WU3 (RED->GREEN->REFACTOR)

- [x] 3.1 RED parser flags set uf.Manager, bare leaves empty
- [x] 3.2 GREEN UpdateFlags.Manager + register --manager/--update-group
- [x] 3.3 GREEN runUpdate branches to runUpdateGroup
- [x] 3.4 GREEN runUpdateGroup enumerate owned minus --skip, per-package availability, gate, run manager executor
- [x] 3.5 GREEN --ci sudo group fails non-zero
- [x] 3.6 REFACTOR table fakes

## Phase 4 WU4 (RED->GREEN->REFACTOR)

- [x] 4.1 RED EnforceRisk:true + TrustOfficial + RiskHigh -> prompt/CI-error
- [x] 4.2 GREEN ConfirmConfig.EnforceRisk bool default false, bypass TrustOfficial->ConfirmAuto
- [x] 4.3 REFACTOR EnforceRisk:false byte-identical

## Phase 5 WU5 (RED->GREEN->REFACTOR)

- [x] 5.1 RED group bulk summary updated/skipped/current/failed
- [x] 5.2 GREEN render.go/group.go group bulk summary canonical order
- [x] 5.3 REFACTOR deterministic discovery order

## Deferred (disabled this increment)

Making bulk default; dnf/pacman adapters; manager self stays self-only.
