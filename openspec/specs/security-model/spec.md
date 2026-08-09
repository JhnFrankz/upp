# Security Model Specification

## Purpose

Trust boundaries, confirmation requirements, and safe execution for official vs custom tools.

## Requirements

### Requirement: Tool Trust Levels

The system MUST distinguish two trust levels:

- **Official**: implemented and maintained by the upp project. Shipped with the binary.
- **Custom**: user-defined commands in config. Treated as untrusted by default.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Official tool | Tool is `brew` (official) | Update requested | Proceeds without extra confirmation |
| Custom tool | Tool is `mytool` (custom) | Update requested | Confirmation required (unless `--ci` with explicit trust) |

### Requirement: Confirmation for Destructive Operations

The system MUST require explicit user confirmation before executing custom tool updates that involve:

- Privileged operations (sudo, admin)
- Destructive actions (rm, uninstall, overwrite)
- Network operations to untrusted sources

Confirmation MUST display: action description, tool origin (custom), and required privileges.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Custom privileged | Custom tool uses `sudo` | Update requested | Prompt: "This will run `sudo apt install ...` for mytool. Allow? [y/N]" |
| Custom destructive | Custom tool runs `rm -rf` | Update requested | Prompt with warning, requires explicit yes |
| `--ci` mode | Custom tool needs confirmation | `upp update --ci` | Error: "Custom tool X requires confirmation; run interactively or mark as trusted in config" |

### Requirement: Config Trust Override

Users MUST be able to mark custom tools as `trusted = true` in config to reduce confirmation friction. Trust level does NOT automatically skip all confirmations — confirmation behavior is risk-based:

| Risk Level | `trusted = false` | `trusted = true` |
|------------|-------------------|------------------|
| Low (non-destructive, no privileges) | Proceeds with info | Proceeds silently |
| Medium (may modify system state) | Confirmation required | Proceeds with info |
| High (destructive, privileged, network to untrusted) | Confirmation required | Confirmation required |

Trusted custom tools still display action and origin before execution. High-risk operations ALWAYS require confirmation regardless of trust level.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Trusted low-risk | `custom.mytool.trusted = true`, non-destructive | Update requested | Shows action, proceeds without prompt |
| Trusted high-risk | `custom.mytool.trusted = true`, uses `sudo` | Update requested | Confirmation still required |
| Untrusted low-risk | `custom.mytool.trusted = false`, non-destructive | Update requested | Proceeds with info |
| Untrusted high-risk | `custom.mytool.trusted = false`, destructive | Update requested | Confirmation required |

### Requirement: Official Tool Integrity

Official tool adapters MUST only invoke platform-native package managers or known official installers (brew, apt, winget, scoop, nvm, npm, pnpm, official curl installers).

Official adapters MUST NOT execute arbitrary user-provided commands.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Official brew | Platform macOS | `brew.update()` | Runs `brew upgrade` only |
| Official docker | Platform Linux | `docker.update()` | Runs `apt upgrade docker-ce` only |

### Requirement: Output Transparency

Every update action MUST display before execution:

- Tool name and trust level
- Command to be executed
- Required privileges (if any)

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Standard update | Updating `npm` | Action displayed | Shows: "npm (official) — `npm update -g` — no privileges" |
| Custom update | Updating `mytool` | Action displayed | Shows: "mytool (custom) — `mytool --update` — sudo required" |
