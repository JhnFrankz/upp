# Delta for tool-adapter

## MODIFIED Requirements

### Requirement: Update Gating

Every adapter MUST declare an `UpdatePolicy` (`PolicyGated` or `PolicyAlwaysUpdate`), and the system MUST gate updates on that declaration, not on a CLI-side ID list. The system MUST run `update()` for an adapter declaring `PolicyGated` (apt, npm, pnpm, nvm) only when that adapter's `check()` reported `update_available=true`. Adapters declaring `PolicyAlwaysUpdate` MUST always run their update when requested, regardless of `check()` result: official adapters without detection (brew, bun, opencode) and custom adapters report `update_available=false` by design, while winget and scoop report real self-update availability by design. An owned tool (gh, docker, go) MUST NOT be gated independently: its update delegates to its owning manager, and the manager's `UpdatePolicy` governs whether the delegated update runs; the owned tool's own `UpdatePolicy` MUST NOT apply to the delegated path. For a manager-group bulk update, the manager's `UpdatePolicy` gates the GROUP's availability (not the manager's own self-only row): a `PolicyGated` manager runs its group update only when any owned package reports availability; a `PolicyAlwaysUpdate` manager runs its group update when requested. When a `PolicyGated` adapter's `check()` fails, the system MUST report the failure for that adapter as a structured error per Adapter Error Handling and MUST NOT treat the failed check as `update_available=false` nor report the adapter as current.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Owned inherits gated | docker owned by apt (Gated) on Linux, apt check reports no update | docker delegated update | Delegated `apt.update()` skipped; docker reported current |
| Owned inherits always | gh owned by brew (AlwaysUpdate) on macOS | gh delegated update | `update()` runs (delegates to brew) |
| Stub official exempt | Adapter declaring `PolicyAlwaysUpdate` without detection (brew/bun/opencode) reports `update_available=false` | Update run | `update()` still runs |
| Gated check fails | `PolicyGated` adapter `check()` fails during update run | Update run | `update()` skipped; failure reported; adapter never reported current |
| Gated group gates on group availability | apt (Gated) group, no owned package has an update | `upp update` (default run) | Group skipped; no owned tool updated |
| AlwaysUpdate group runs | brew (AlwaysUpdate) group | `upp update` (default run) | Group update runs regardless of check result |

(Previously: the two group-level gating scenarios were invoked via the opt-in `--manager apt` / `--manager brew` flags; gating semantics are unchanged — Gated groups still gate on group availability, AlwaysUpdate groups still run — and the scenarios now invoke the default update path, which is the only update path.)
