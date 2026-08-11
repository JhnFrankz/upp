# Design: upp audit — security confirmation & first-run init fixes

## Technical Approach

Restore the security matrix and first-run wizard: real custom commands reach `ClassifyCommand`; `TrustLevel` splits into 3 tiers so config `trusted` never aliases Official; first-run state from explicit `config.Exists()`; `Load()` applies `ApplyDefaults` for existing files. Audit probes become permanent tests. Implements the 3 delta specs.

## Architecture Decisions

| # | Options | Tradeoff | Decision |
|---|---|---|---|
| D1 | 3-tier enum vs 2-tier + bool | Renames `TrustTrusted` in 12 adapters; bool keeps the collision | `TrustOfficial`/`TrustCustomTrusted`/`TrustCustomUntrusted` |
| D2 | Typed `adapters.TrustLevel` vs string mapping | No cycle; kills stringly-typed mapping | Typed enum; drop `Trusted` bool; delete `trustLevelString()` |
| D3 | Real command via `ToolInfo.Command` vs method vs type-assert | Interface untouched; officials unaffected (Official short-circuits) | `Command`+`Privileges` on `ToolInfo`; custom fills both; runUpdate passes to Classify/Confirm |
| D4 | `--ci` low untrusted: auto vs error | Code errors (pinned); ux delta mandates auto | CI: Low→Auto; Medium→Auto(trusted)/Err(untrusted); High→Err |
| D5 | `config.Exists()` vs Load sentinel | Explicit; restores dead prompt branch | `Exists() bool` |
| D6 | `ApplyDefaults` only if file exists | Missing→empty tools; empty/partial→catalog (smoke asserts old) | Inside Load, existing files only |
| D7 | `--quiet` suppressing prompts | Prompts bypass renderer | No change; test pins prompt-under-quiet |

## Data Flow

```
config.toml ──os.Stat──▶ Exists() ──true──▶ init: overwrite prompt [y/N]
                  │       └──false──▶ init: wizard → detect → Save
Load() ──(file exists)──▶ Validate → ApplyDefaults ──▶ cfg

update: CustomAdapter.Info() ──▶ ToolInfo{Command, Privileges, Trust: Custom*}
  cmd = info.Command ──▶ ClassifyCommand ──▶ Risk ──▶ ConfirmAction{Trust, Risk, Command, Privileges, CI}
    Auto/Proceed ──▶ a.Update(false) ──▶ shellExec(cmd)
    Deny ──▶ skipped;  Error ──▶ hasFailure ──▶ --ci exits non-zero
```

## File Changes

| File | Action | Description |
|---|---|---|
| internal/adapters/interface.go | Modify | 3-tier TrustLevel; ToolInfo +Command +Privileges |
| internal/adapters/custom.go | Modify | trusted→TrustCustomTrusted; Info() fills Command/Privileges |
| internal/adapters/official/*.go (12) | Modify | Rename TrustTrusted→TrustOfficial |
| official/{info,registry,adapter_update}_test.go, custom_test.go | Modify | Renames; Trust assertions |
| internal/security/confirm.go | Modify | Typed TrustLevel; drop Trusted bool; CI matrix; prompts unchanged |
| security/{confirm,security_expanded}_test.go | Modify | CI low-risk rows; enum trust |
| internal/cli/update.go | Modify | Real command + Privileges; CI error msg; delete trustLevelString |
| internal/cli/init.go | Modify | Exists() gate; restore overwrite prompt; single Load |
| internal/config/config.go | Modify | ApplyDefaults in Load; add Exists() |
| internal/config/config_test.go | Modify | Load states; Exists() |
| internal/cli/audit_probe_test.go | Create | 2 converted probes + low-risk-success + --quiet-prompt (i18n dropped) |
| internal/cli/init_probe_test.go | Create | Wizard; overwrite prompt |

## Interfaces / Contracts

```go
// internal/adapters/interface.go
type TrustLevel int
const (
    TrustOfficial TrustLevel = iota
    TrustCustomTrusted   // trusted=true — never Official
    TrustCustomUntrusted
)
type ToolInfo struct {
    ID, Name string
    Platforms []string
    Trust     TrustLevel
    Command   string    // real command; officials empty
    Privileges []string
}
```
```go
// internal/config/config.go
func Exists() bool // os.Stat(ConfigPath()) == nil
```
`ConfirmConfig.TrustLevel` → `adapters.TrustLevel`; `Trusted` removed. Interactive: High→prompt (any); Medium→prompt (untrusted)/info (trusted); Low→info. CI: Low→auto; Medium→auto (trusted)/error (untrusted); High→error.

## Testing Strategy

| Layer | What to Test | Approach |
|---|---|---|
| Unit | ClassifyCommand classes (sudo, rm -rf, curl\|sh, &&) | Table-driven, extend chaining |
| Unit | ConfirmAction full matrix, enum trust | Table-driven, `strings.NewReader` prompts |
| Unit | Load defaults per file state; Exists() | t.TempDir + t.Setenv("HOME") |
| Unit | CustomAdapter.Info() mapping | Trust/Command/Privileges assertions |
| Integration | Security probes: high-risk custom never executes (CI non-zero; interactive trusted+untrusted); low-risk trusted executes; --quiet still prompts | probeSetup (fake sudo + marker), root.Execute, marker stat |
| Integration | init wizard / overwrite prompt | Missing → created; existing → prompt, unchanged |
| E2E | smoke-test.sh | Update empty-config assertions |

## Threat Matrix

| Boundary | Min adversarial cases | Applicability | Design response | Planned RED tests |
|---|---|---|---|---|
| Documentation-like paths | sudo, rm -rf, curl\|sh, && chains | Applicable — custom command strings are classified then executed via shellExec | Classify → gate → shellExec; Official exempt | sudo (probe: trusted CI); rm -rf (probes: interactive trusted+untrusted); curl\|sh/chain (unit); correct-pass: trusted low-risk executes |
| Git repository selection | git -C, rel/abs paths | N/A — no git ops | — | — |
| Commit state | staged, -a, empty index | N/A — no commit ops | — | — |
| Push state | tracking, first push, refspec | N/A — no push ops | — | — |
| PR commands | --head, env prefix, composed | N/A — no PR automation | — | — |

## Migration / Rollout

No migration. Behavior changes: trusted high-risk always confirms; `--ci` untrusted low-risk proceeds; empty/partial configs default to catalog (update smoke assertions). Rollback: revert branch.

## Open Questions

Resolved (user decision, 2026-08-11):

- [x] `--ci` medium-risk untrusted stays **Error** (conservative) — confirmed by user; D4 matrix unchanged.
- [x] `fmt.Scanln` stdin **accepted** for prompts until the hermetic-CLI-test follow-up (track 4, out of scope); no input seam added in this change.
