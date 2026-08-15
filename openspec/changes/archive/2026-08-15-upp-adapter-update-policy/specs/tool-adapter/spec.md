# Delta for tool-adapter

## MODIFIED Requirements

### Requirement: Update Gating

Every adapter MUST declare an `UpdatePolicy` (`PolicyGated` or `PolicyAlwaysUpdate`), and the system MUST gate updates on that declaration, not on a CLI-side ID list. The system MUST run `update()` for an adapter declaring `PolicyGated` (apt, npm, pnpm, nvm) only when that adapter's `check()` reported `update_available=true`. Adapters declaring `PolicyAlwaysUpdate` MUST always run their update when requested, regardless of `check()` result: official adapters without detection (brew, bun, docker, gh, go, opencode) and custom adapters report `update_available=false` by design, while winget and scoop report `update_available=true` by design. When a `PolicyGated` adapter's `check()` fails, the system MUST report the failure for that adapter as a structured error per Adapter Error Handling and MUST NOT treat the failed check as `update_available=false` nor report the adapter as current.
(Previously: gating was decided by a CLI-side list of official adapter IDs with real update detection (apt, npm, pnpm, nvm); behavior of a failed check was undefined.)

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Official update available | Adapter declaring `PolicyGated` `check()` reports `update_available=true` | Update run | `update()` runs for that adapter |
| Official no update | Adapter declaring `PolicyGated` (apt/npm/pnpm/nvm) `check()` reports `update_available=false` | Update run | `update()` skipped; adapter reported current |
| Stub official exempt | Adapter declaring `PolicyAlwaysUpdate` without detection (brew/bun/docker/gh/go/opencode) reports `update_available=false` | Update run | `update()` still runs |
| Custom exempt | Custom adapter declaring `PolicyAlwaysUpdate` reports `update_available=false` | Update run | `update()` still runs |
| winget/scoop exempt | winget or scoop adapter declaring `PolicyAlwaysUpdate` | Update run | `update()` always runs |
| Dynamic detection | apt reports `update_available=false` | Update run | `update()` skipped |
| Gated check fails | `PolicyGated` adapter `check()` fails during update run | Update run | `update()` skipped; failure reported; adapter never reported current |

## ADDED Requirements

### Requirement: Check Failure Signal

`check()` MUST return a structured error (tool name, operation, exit code — per Adapter Error Handling) when its update-detection subprocess fails, for apt, nvm, npm, and pnpm. A subprocess failure is a non-zero exit code EXCEPT the documented npm/pnpm `outdated` convention where exit code 1 means updates are available (a valid detection, not a failure); timeout (exit 124 via the `timeout 15` wrapper) and other non-zero exits are failures. Empty subprocess output MUST NOT be treated as failure: a detection subprocess that succeeds with empty output reports unknown status (`update_available=false`) without error. The npm and pnpm adapters MUST NOT mask detection subprocess exit codes. The CLI MUST surface a failed check as `StatusFailed` for that adapter and MUST NOT report `StatusCurrent` for a failed check.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Detection fails | apt/nvm/npm/pnpm detection subprocess exits non-zero (npm/pnpm: any code except the documented exit-1-outdated) | `check()` | Structured error with tool name, operation, exit code |
| Empty output | Detection subprocess exits 0 with empty output | `check()` | Unknown status (`update_available=false`), no error |
| Gated check fails in run | `PolicyGated` adapter `check()` fails | Update run | Failure surfaced as `StatusFailed`; update skipped; not `StatusCurrent` |
| npm/pnpm maskless | npm/pnpm detection subprocess exits non-zero (incl. timeout 124) | `check()` through timeout wrapper | Failure propagates; exit code not swallowed |
| npm/pnpm exit-1 outdated | npm/pnpm `outdated` exits 1 (updates available) | `check()` | Valid detection: `update_available=true`, no error |
