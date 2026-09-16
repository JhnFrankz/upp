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

Confirmation MUST be classified by the REAL privileges and risk of the command to be executed, not by the tool's trust level alone. The command and privileges the gate classifies MUST be exactly what will execute: they MUST come from the adapter's declared real update command (see tool-adapter) and MUST byte-equal the command the update path actually runs — the system MUST NOT classify a synthesized command (e.g. `<manager> upgrade <pkg>`). `plan.RiskCommand` and plan privileges MUST byte-equal the executed command for every managed adapter, pacman included. Custom manager-delegated tools MUST be classified by the delegated manager's real update command and its declared privileges. Owned-tool group updates MUST be reclassified by their real command risk: a manager-group bulk update that runs a privileged owned-package command (e.g. `sudo apt install --only-upgrade gh` or `sudo pacman -S --noconfirm <pkg>`) MUST prompt for confirmation even when each owned tool is `TrustOfficial`, because the group batch elevates real risk (sudo). A non-privileged group update (e.g. `brew upgrade gh`) MAY auto-proceed. Confirmation MUST display: action description, tool origin (custom), and required privileges. `--ci` MUST fail high-risk custom updates with a non-zero exit, even when `trusted = true`, and MUST fail a high-risk owned-tool group update that requires confirmation.

(Previously: the gate-input source was implicit — the spec mandated real-risk classification and pacman prompting, but never bound the classified command/privileges to the adapter-declared command that actually executes; in practice pacman rows were classified from a synthesized sudo-free string and auto-proceeded, even under `--ci`. The pacman prompting and `--ci` failure outcomes below are required behavior, not regressions.)

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Custom privileged | Custom tool uses `sudo` | Update requested | Prompt: "This will run `sudo apt install ...` for mytool. Allow? [y/N]" |
| Custom destructive | Custom tool runs `rm -rf` | Update requested | Prompt with warning, requires explicit yes |
| `--ci` high-risk | Custom tool needs confirmation | `upp update --ci` | Exits non-zero: "requires confirmation" — trust does not waive it |
| `--ci` trusted high-risk | `trusted = true`, uses `sudo` | `upp update --ci` | Exits non-zero; confirmation cannot be waived in non-interactive mode |
| Sudo-heavy group prompts | Linux, apt owned tools (gh/docker) use `sudo apt install --only-upgrade` | `upp update` (default run, apt group in selection) | Prompts for confirmation despite TrustOfficial owned tools |
| Non-sudo group proceeds | macOS, brew owned tools use `brew upgrade` (no sudo) | `upp update` (default run, brew group in selection) | Group update proceeds without confirmation |
| `--ci` sudo group fails | Linux, `--ci`, apt group sudo package commands | `upp update --ci` | Exits non-zero for the sudo-heavy group; group not executed |
| Pacman privileged update prompts | Linux, pacman self-update or package update declares `sudo` privilege | `upp update` (interactive) | Prompts for confirmation before executing `sudo pacman` command |
| `--ci` pacman privileged fails | Linux, `--ci`, pacman update requires `sudo` | `upp update --ci` | Exits non-zero; privileged execution fails closed in non-interactive mode |
| Gate input is the executed command | Any managed adapter row planned (pacman included) | `upp update` plan built | `plan.RiskCommand` byte-equals the command the update path executes; plan privileges equal the adapter's declared privileges |
| Pacman package gate sees sudo | pacman package row planned (`sudo pacman -S --noconfirm <pkg>`) | Gate classifies the row | Classified high risk from the real privileged command: interactive run prompts; `--ci` exits non-zero |
| Custom manager-delegated confirm | Custom tool with `manager = "pacman"` | Update requested | Classified/confirmed by the delegated manager's real self-update command (`sudo pacman -S --noconfirm pacman`, sudo) — never a synthesized `pacman upgrade <tool>` string |

(Previously: confirmation applied only to custom tool updates; owned-tool rows were `TrustOfficial` and always auto-proceeded (`ConfirmAuto`), so a sudo-heavy manager group update would run without prompting.)

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

Official tool adapters MUST only invoke platform-native package managers or known official installers (brew, apt, pacman, winget, scoop, nvm, npm, pnpm, official curl installers).

An owned tool (gh, docker, go) MUST NOT invoke a manager command itself; its update MUST delegate to its owning manager, so the command executed and the privileges incurred are those of the manager, and the owned tool's risk derives from its manager's operation, not its own hardcoded command. A tool with no resolving owner uses its own official installer.

Official adapters MUST NOT execute arbitrary user-provided commands.

The pacman adapter MUST strictly invoke self-only updates (`sudo pacman -S --noconfirm pacman`) or targeted package updates (`sudo pacman -S --noconfirm <pkg>`), and MUST NOT invoke whole-system upgrades (`pacman -Syu`) or mutating sync operations (`pacman -Sy`) during read-only version checks.

Self-update integrity MUST fail closed: the replacement archive's sha256 MUST match `checksums.txt` from the SAME release, both fetched over HTTPS with ~10s timeouts. Mismatch or missing entry MUST abort — original binary untouched, non-zero exit (stricter than install.sh's warn-and-skip). Downloaded bytes MUST be extracted, never executed.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Official brew | Platform macOS | `brew.update()` | Runs `brew update` only |
| Official pacman self-update | Platform Linux | `pacman.update()` | Runs `sudo pacman -S --noconfirm pacman` only |
| Pacman never runs whole system upgrade | Platform Linux | `pacman.update()` | MUST NOT invoke `pacman -Syu` |
| Pacman check never syncs DB | Platform Linux | `pacman.check()` | Reads local sync DB; MUST NOT invoke `pacman -Sy` |
| Linux docker delegates | Platform Linux, docker owned by apt | `docker.update()` | The owning manager (apt) updates docker; no hardcoded `apt upgrade docker-ce` |
| macOS gh delegates | Platform macOS, gh owned by brew | `gh.update()` | Delegates to brew; no hardcoded `brew upgrade gh` |
| Self-update mismatch | Archive sha256 ≠ checksums.txt | Verify | Abort, binary untouched, exit non-zero |
| Self-update missing entry | checksums.txt has no asset line | Verify | Abort, binary untouched, exit non-zero |
| Self-update HTTPS-only | Asset URL over plain HTTP | Download | Refused, exit non-zero |

(Previously: official package managers listed were brew, apt, winget, scoop, nvm, npm, and pnpm; pacman was not included, and no pacman-specific command restrictions existed.)

(Previously: `docker.update()` on Linux ran `apt upgrade docker-ce` and `gh.update()` ran its own hardcoded manager command; an owned tool's integrity and risk were independent of any manager.)

### Requirement: Output Transparency

Every update action MUST display before execution:

- Tool name and trust level
- Command to be executed
- Required privileges (if any)

The displayed command and privileges MUST be the adapter-declared command that will actually execute — for managed adapters and custom manager-delegated tools alike — never a synthesized string.

(Previously: the requirement listed what to display but did not bind the displayed command and privileges to the adapter-declared command that actually executes.)

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Standard update | Updating `npm` | Action displayed | Shows: "npm (official) — `npm update -g` — no privileges" |
| Custom update | Updating `mytool` | Action displayed | Shows: "mytool (custom) — `mytool --update` — sudo required" |
| Pacman row transparency | pacman package update planned | Action displayed | Shows `sudo pacman -S --noconfirm <pkg>` with sudo privileges |
| Custom delegated transparency | Custom tool with `manager = "pacman"` | Action displayed | Shows the manager's real self-update command (`sudo pacman -S --noconfirm pacman`, sudo), not a synthesized string |
### Requirement: Zero-Sudo Uninstallation Policy

`upp uninstall` MUST NEVER invoke `sudo` or attempt automatic privilege escalation. If any binary, backup, configuration, or cache directory cannot be removed due to insufficient filesystem permissions, the command MUST perform best-effort removal of all accessible targets, emit actionable manual remediation commands (e.g. `sudo rm -rf <path>`), and exit with status 1.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Unwritable binary | `/usr/local/bin/upp` owned by root, user is non-root | `upp uninstall` | Deletes user config/cache, prints warning for `/usr/local/bin/upp` with manual sudo command, exits 1 |
| Full permission | All targets writable | `upp uninstall` | Deletes all targets cleanly, exits 0 |
| Simulation mode | Root binary and user config | `upp uninstall --dry-run` | Lists all planned target deletions without modifying disk, exits 0 |
