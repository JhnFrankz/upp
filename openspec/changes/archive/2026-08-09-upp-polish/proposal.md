# Proposal: upp-polish

## Intent

Post-rewrite polish: the Bash→Go migration (upp-evolution, archived) is functionally complete, but the repo is out of sync with its new reality. README documents the removed `upp-legacy.sh`, there is no CI or release pipeline despite docs/install.sh pointing at GitHub Releases, the skill registry misses the sdd-* family, 9 files fail `gofmt`, `openspec/config.yaml` is uncommitted, and `internal/adapters/official` sits at 60.7% coverage (lowest in repo). This change closes those gaps without touching product behavior.

## Scope

### In Scope
- **Docs**: Full README rewrite for the Go CLI (install, commands init/update/check/list/export/import, config, security model, supported platforms); remove the dead "Migration from upp.sh" section.
- **CI + release pipeline**: GitHub Actions workflow (test + golangci-lint + gofmt check + smoke) and `make release` target (tag + cross-platform assets + checksums). No real release is published — the pipeline is ready; the user triggers the first tag.
- **Skill registry**: `gentle-ai skill-registry refresh` to include the 10 missing sdd-* skills.
- **Format + commits**: `gofmt -s -w` on the 9 unformatted .go files; commit `openspec/config.yaml` (Go stack + strict_tdd) and the refreshed registry.
- **Official adapter tests**: raise `internal/adapters/official` coverage from 60.7% to ≥80% with table-driven tests for the 12 adapters.

### Out of Scope
- First real release (v0.1.0 tag/assets) — triggered by the user after this change.
- Spanish CLI output changes — preserved as-is.
- Product functionality changes — parity is complete.

## Capabilities

### New Capabilities
None — no new product behavior; CI/release is delivery infrastructure, not a spec domain.

### Modified Capabilities
None — no requirement changes in the 6 existing domains (command-interface, config-system, platform-detection, security-model, tool-adapter, ux-patterns). Docs, tests, and tooling do not alter spec-level behavior.

## Approach

Docs-first: rewrite README aligned to the 6 spec domains and actual CLI surface. CI: single workflow (ubuntu-latest, Go 1.22) running `go test ./... -count=1`, golangci-lint, `gofmt -s -l` check, and `scripts/smoke-test.sh`. Release: `make release` builds assets via existing `build-all`, tags, and emits checksums — dry-run safe, no push automation. Tests: table-driven per adapter reusing existing helper patterns.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `README.md` | Modified | Rewrite for Go CLI; remove "Migration from upp.sh" |
| `.github/workflows/ci.yml` | New | test + lint + fmt-check + smoke |
| `Makefile` | Modified | Add `release` target (tag + assets + checksums) |
| `.atl/skill-registry.md` | Modified | Refresh: include sdd-* family |
| `internal/adapters/official/*_test.go` | Modified | Table-driven tests, 12 adapters, ≥80% |
| 9 .go files (cli, config, output, platform, security, tests) | Modified | `gofmt -s -w` only |
| `openspec/config.yaml` | Modified | Commit pending config |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Some adapters hard to reach 80% (docker, winget) | Med | Table-driven cases on core paths; helpers; per-adapter incremental coverage |
| CI smoke differs from local (shell/OS) | Low | Pin ubuntu-latest + Go 1.22; reuse existing smoke script |
| gofmt churn breaks blame/PR review | Low | Separate formatting commit from test commits |

## Rollback Plan

Non-functional change; every item is revertible via `git revert` per commit (README, workflow, Makefile, registry, config, tests). Deleting `.github/workflows/ci.yml` fully disables CI. No schema, state, or persisted data affected.

## Dependencies

- golangci-lint (CI only, no code dependency)
- GitHub Actions (no secrets/keys needed)

## Success Criteria

- [ ] `go test ./... -count=1` and `go test ./... -count=1 -race` green
- [ ] `gofmt -s -l .` reports no unformatted files
- [ ] Coverage `internal/adapters/official` ≥80%
- [ ] CI workflow passes on push (workflow validated on a branch)
- [ ] `make release` produces tags/assets/checksums; no release published
- [ ] `README.md` has no "Migration from upp.sh"; all 6 commands documented
- [ ] `.atl/skill-registry.md` lists the 10 sdd-* skills
- [ ] `openspec/config.yaml` committed
