# Delta for tool-adapter

## ADDED Requirements

### Requirement: Update Gating

The system MUST run `update()` for an official adapter with real update detection (apt, npm, pnpm, nvm) only when that adapter's `check()` reported `update_available=true`. Official adapters WITHOUT update detection (brew, bun, docker, gh, go, opencode) MUST NOT be gated: they report `update_available=false` by design and MUST still run their update when requested, as documented per adapter. Custom adapters MUST NOT be gated: they report `update_available=false` by design and MUST still run their update when requested. winget and scoop MUST NOT be gated: they report `update_available=true` by design and MUST always run. Adapters with dynamic detection (apt, npm, pnpm, nvm) MUST respect their `check()` result.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Official update available | Gated official adapter `check()` reports `update_available=true` | Update run | `update()` runs for that adapter |
| Official no update | Gated official adapter (apt/npm/pnpm/nvm) `check()` reports `update_available=false` | Update run | `update()` skipped; adapter reported current |
| Stub official exempt | Official adapter without detection (brew/bun/docker/gh/go/opencode) reports `update_available=false` | Update run | `update()` still runs |
| Custom exempt | Custom adapter reports `update_available=false` | Update run | `update()` still runs |
| winget/scoop exempt | winget or scoop adapter | Update run | `update()` always runs |
| Dynamic detection | apt reports `update_available=false` | Update run | `update()` skipped |

### Requirement: Subprocess Timeouts

Every subprocess launched by `check()` or `update()` MUST be bounded by a timeout, for both custom and official adapters. When a subprocess exceeds its timeout, the system MUST terminate the subprocess, MUST return a structured timeout error for that tool, and MUST NOT block the remainder of the update run. `check()` and `update()` MUST use distinct timeouts appropriate to each operation; exact durations are a design decision, and slow package managers (brew, nvm, apt) MAY warrant longer update timeouts.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Update completes in time | `brew update` finishes within the update timeout | `update()` | Success result returned |
| Update times out | `brew update` hangs beyond the update timeout | `update()` | Structured timeout error; other tools still update |
| Check times out | `check()` subprocess hangs beyond the check timeout | `check()` | Structured timeout error; update run continues |
| Custom times out | Custom shell command exceeds the timeout | `update()` | Structured timeout error returned |

### Requirement: Go Adapter Architecture

The go adapter's Linux download MUST fetch the Go toolchain tarball matching the running process architecture: `linux-amd64` on amd64 hosts and `linux-arm64` on arm64 hosts. The adapter MUST NOT hardcode a single architecture.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| amd64 host | Process runs on linux/amd64 | go `check()` or `update()` | Downloads `go*linux-amd64.tar.gz` |
| arm64 host | Process runs on linux/arm64 | go `check()` or `update()` | Downloads `go*linux-arm64.tar.gz` |
