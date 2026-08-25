# Security Model Specification

## Purpose

Trust boundaries, confirmation requirements, and safe execution for official vs custom tools.

## Requirements

### Requirement: Tool Trust Levels

The system MUST distinguish three trust levels:

- **Official**: implemented and maintained by the upp project. Shipped with the binary.
- **CustomTrusted**: user-defined commands in config marked `trusted = true`.
- **CustomUntrusted**: user-defined commands in config, untrusted by default.

Config `trusted` MUST map to CustomTrusted and MUST NEVER map to Official. Trust level MUST NOT bypass the risk matrix.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Official tool | Tool is `brew` (official) | Update requested | Proceeds without extra confirmation |
| Custom untrusted | Tool is `mytool` (custom, `trusted = false`) | Update requested | Risk matrix applies; confirmation as required |
| Custom trusted | Tool is `mytool` (custom, `trusted = true`) | Update requested | Classified as CustomTrusted, never Official; risk matrix still applies |

(Previously: two levels — Official and Custom; config `trusted` promoted custom tools to Official trust, bypassing confirmation.)

### Requirement: Confirmation for Destructive Operations

The system MUST require explicit user confirmation before executing custom tool updates that involve:

- Privileged operations (sudo, admin)
- Destructive actions (rm, uninstall, overwrite)
- Network operations to untrusted sources

Confirmation MUST display: action description, tool origin (custom), and required privileges. `--ci` MUST fail high-risk custom updates with a non-zero exit, even when `trusted = true`.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Custom privileged | Custom tool uses `sudo` | Update requested | Prompt: "This will run `sudo apt install ...` for mytool. Allow? [y/N]" |
| Custom destructive | Custom tool runs `rm -rf` | Update requested | Prompt with warning, requires explicit yes |
| `--ci` high-risk | Custom tool needs confirmation | `upp update --ci` | Exits non-zero: "requires confirmation" — trust does not waive it |
| `--ci` trusted high-risk | `trusted = true`, uses `sudo` | `upp update --ci` | Exits non-zero; confirmation cannot be waived in non-interactive mode |

(Previously: `--ci` error suggested "mark as trusted in config" as a workaround; trusted high-risk custom ran without confirmation.)

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

An owned tool (gh, docker, go) MUST NOT invoke a manager command itself; its update MUST delegate to its owning manager, so the command executed and the privileges incurred are those of the manager, and the owned tool's risk derives from its manager's operation, not its own hardcoded command. A tool with no resolving owner uses its own official installer.

Official adapters MUST NOT execute arbitrary user-provided commands.

Self-update integrity MUST fail closed: the replacement archive's sha256 MUST match `checksums.txt` from the SAME release, both fetched over HTTPS with ~10s timeouts. Mismatch or missing entry MUST abort — original binary untouched, non-zero exit (stricter than install.sh's warn-and-skip). Downloaded bytes MUST be extracted, never executed.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Official brew | Platform macOS | `brew.update()` | Runs `brew update` only |
| Linux docker delegates | Platform Linux, docker owned by apt | `docker.update()` | The owning manager (apt) updates docker; no hardcoded `apt upgrade docker-ce` |
| macOS gh delegates | Platform macOS, gh owned by brew | `gh.update()` | Delegates to brew; no hardcoded `brew upgrade gh` |
| Self-update mismatch | Archive sha256 ≠ checksums.txt | Verify | Abort, binary untouched, exit non-zero |
| Self-update missing entry | checksums.txt has no asset line | Verify | Abort, binary untouched, exit non-zero |
| Self-update HTTPS-only | Asset URL over plain HTTP | Download | Refused, exit non-zero |

(Previously: `docker.update()` on Linux ran `apt upgrade docker-ce` and `gh.update()` ran its own hardcoded manager command; an owned tool's integrity and risk were independent of any manager.)

### Requirement: Output Transparency

Every update action MUST display before execution:

- Tool name and trust level
- Command to be executed
- Required privileges (if any)

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Standard update | Updating `npm` | Action displayed | Shows: "npm (official) — `npm update -g` — no privileges" |
| Custom update | Updating `mytool` | Action displayed | Shows: "mytool (custom) — `mytool --update` — sudo required" |
