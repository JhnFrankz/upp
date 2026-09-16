# Proposal: RiskCommand From the Real Executed Command

## Intent

The confirmation gate evaluates a synthesized command, not the executed one. `UpdateCmdName` (resolve.go:176) lacks a pacman case and builds `"<manager> upgrade <pkg>"`; pacman's `Info()` declares no privileges — so `plan.RiskCommand` is sudo-free fiction while `UpdatePackage` executes `sudo pacman -S --noconfirm <pkg>` (pacman.go:156). `ClassifyCommand` on the fiction yields RiskLow: silent auto-proceed, even under `--ci`. Custom tools with `manager=pacman/scoop` confirm a string that never runs (custom.go:90 delegates to the manager's self-update). The archived pacman verify-report "proved" compliance via a synthetic-string unit test (verify-report.md:194), never via the real plan→execute path. This repairs the gate to consume the command that actually executes, per confirm.go's EnforceRisk intent and the security-model spec.

## Scope

### In Scope
- Adapter-declared source of truth: managers expose their real per-package update command; `plan.RiskCommand`/`Privileges` derive from it, replacing `UpdateCmdName` synthesis on the managed path.
- pacman declares `Privileges: ["sudo"]` so the gate sees the real privileged command.
- Custom manager-delegated tools: risk/privileges derive from the delegated manager's real command.
- Integration-style test asserting `plan.RiskCommand` byte-equals the command `UpdatePackage`/`Update` executes (pacman included).

### Out of Scope
- Audit fixes #2-#5 (`init --ci` gate, go installer checksum, `check_cmd` classification, dead-wood cleanup incl. `detectPrivileges` dedup and doas/pkexec classifier keywords) — each its own change.
- Any behavior change where the synthesized command is already byte-identical to execution: apt/brew/winget rows keep today's exact prompt behavior.

## Capabilities

> Contract with sdd-spec. Verified against `openspec/specs/`.

### New Capabilities
None — defect repair inside existing capabilities.

### Modified Capabilities
- `security-model`: "Confirmation for Destructive Operations" and "Output Transparency" gain the rule that the gate's input command/privileges MUST be what will actually execute (adapter-declared), with pacman and custom-manager scenarios.
- `tool-adapter`: adapter contract gains declaration of the real update command (pacman sudo declaration included) so plan metadata cannot diverge from execution.

## Approach

Managers declare their real per-package update command (mechanism — `PackageUpdater` method vs `ToolInfo` template — is sdd-design's call); pacman adds `Privileges: ["sudo"]` to `Info()`. plan.go derives `RiskCommand`/`Privileges` from declared commands; custom delegated rows inherit the manager's real command. The CLI gate (update.go:444) keeps consuming `plan.RiskCommand` — only its input becomes truthful; EnforceRisk semantics unchanged.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/engine/resolve.go` | Modified | Retire `UpdateCmdName` synthesis on managed path |
| `internal/engine/plan.go` | Modified | Derive RiskCommand/Privileges from adapter-declared command |
| `internal/adapters/interface.go` | Modified | Real-command declaration surface |
| `internal/adapters/official/pacman.go` | Modified | `Info()` declares `Privileges: ["sudo"]` |
| `internal/adapters/custom.go` | Modified | Delegated rows inherit manager's real command |
| `internal/cli/update.go` | Verified | Gate contract unchanged; input now real |
| tests (engine/adapters/cli) | New | Integration equality test + regression pins |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| UX change: pacman rows prompt / `--ci` fails where they silently ran | Certain (intended) | Mandated by existing security-model scenarios; changelog note |
| Declared command drifts from executed again | Low | Integration equality test; table-driven adapter tests |
| Prompt churn on apt/brew/winget | Low | Byte-identical regression pins on current RiskCommands |

## Rollback Plan

Revert the change's commits. No persisted state, config, or disk migration exists — plan metadata is in-memory, so a revert restores prior gate inputs and prompt behavior byte-for-byte. Added tests revert with the code.

## Dependencies

None external; builds on existing `PackageUpdater`/`ConfirmAction` contracts.

## Success Criteria

- [ ] `plan.RiskCommand` equals the exact executed command for every managed adapter (integration test, pacman included).
- [ ] pacman rows carry `Privileges: ["sudo"]`; interactive run prompts; `--ci` exits non-zero.
- [ ] apt/brew/winget RiskCommands byte-identical to today (regression pins pass).
- [ ] Battery green: `gofmt -s`, `go vet`/golangci-lint, `go test ./... -count=1 -race`.
