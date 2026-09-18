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

Confirmation MUST be classified by the REAL privileges and risk of the command to be executed, not by the tool's trust level alone. The command and privileges the gate classifies MUST be exactly what will execute: they MUST come from the adapter's declared real update command (see tool-adapter) and MUST byte-equal the command the update path actually runs — the system MUST NOT classify a synthesized command (e.g. `<manager> upgrade <pkg>`). `plan.RiskCommand` and plan privileges MUST byte-equal the executed command for every managed adapter, pacman included. Custom manager-delegated tools MUST be classified by the delegated manager's real update command and its declared privileges. Owned-tool group updates MUST be reclassified by their real command risk: a manager-group bulk update that runs a privileged owned-package command (e.g. `sudo apt install --only-upgrade gh` or `sudo pacman -S --noconfirm <pkg>`) MUST prompt for confirmation even when each owned tool is `TrustOfficial`, because the group batch elevates real risk (sudo). A non-privileged group update (e.g. `brew upgrade gh`) MAY auto-proceed. Confirmation MUST display: action description, tool origin (custom), and required privileges. Under `--ci`, which cannot prompt, the tier depends on who ships the command: a high-risk **custom** update MUST fail with a non-zero exit even when `trusted = true`, while a command that is an **official** declaration — including an owned-tool group's manager package command — proceeds. See Official Command Versus Custom Command Under `--ci`.

(Previously: the gate-input source was implicit — the spec mandated real-risk classification and pacman prompting, but never bound the classified command/privileges to the adapter-declared command that actually executes; in practice pacman rows were classified from a synthesized sudo-free string and auto-proceeded, even under `--ci`. The pacman prompting outcome below is required behavior, not a regression. Later: `--ci` also failed every privileged official row, which made it unusable on Debian/Ubuntu hosts with pending apt updates; the Official Command Versus Custom Command requirement replaces that with the shipped-command distinction.)

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Custom privileged | Custom tool uses `sudo` | Update requested | Prompt: "This will run `sudo apt install ...` for mytool. Allow? [y/N]" |
| Custom destructive | Custom tool runs `rm -rf` | Update requested | Prompt with warning, requires explicit yes |
| `--ci` high-risk | Custom tool needs confirmation | `upp update --ci` | Exits non-zero: "requires confirmation" — trust does not waive it |
| `--ci` trusted high-risk | `trusted = true`, uses `sudo` | `upp update --ci` | Exits non-zero; confirmation cannot be waived in non-interactive mode |
| Sudo-heavy group prompts | Linux, apt owned tools (gh/docker) use `sudo apt install --only-upgrade` | `upp update` (default run, apt group in selection) | Prompts for confirmation despite TrustOfficial owned tools |
| Non-sudo group proceeds | macOS, brew owned tools use `brew upgrade` (no sudo) | `upp update` (default run, brew group in selection) | Group update proceeds without confirmation |
| `--ci` sudo group proceeds | Linux, `--ci`, apt group sudo package commands | `upp update --ci` | Group update proceeds: the package command is an official, shipped declaration |
| Pacman privileged update prompts | Linux, pacman self-update or package update declares `sudo` privilege | `upp update` (interactive) | Prompts for confirmation before executing `sudo pacman` command |
| `--ci` pacman privileged proceeds | Linux, `--ci`, pacman update requires `sudo` | `upp update --ci` | Proceeds: the command is an official, shipped declaration |
| Apt privileged update prompts | Linux, apt self-update (`sudo apt install --only-upgrade apt`) declares no `Privileges` | `upp update` (interactive) | Prompts for confirmation — classified by the REAL command, not by the adapter's declarations |
| Gate input is the executed command | Any managed adapter row planned (pacman included) | `upp update` plan built | `plan.RiskCommand` byte-equals the command the update path executes; plan privileges equal the adapter's declared privileges |
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

### Requirement: Custom Check-Command Gate

A custom tool's `check_cmd` is arbitrary shell. The system MUST classify it by its real risk — never by the tool's trust level alone — before allowing it to run without consent, exactly as it does for a custom update command.

A check command classified above `RiskLow` MUST NOT run from a read-only surface (`upp list`, the bare dashboard), whose contract is that it does not modify the system. In an interactive `update`, the user MUST be prompted through the standard confirmation gate before the check runs. `--ci` MUST deny with a non-zero exit. A denial drops only that check: the tool still reports as detected, with an empty version.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Benign check command | `check_cmd = "mytool --version"` | Any command | Runs unchanged, no prompt |
| No check command | `check_cmd` absent | Any command | No gate applies |
| Dangerous check in list | `check_cmd` uses `sudo` | `upp list` | Check is not run; tool listed as detected with an empty version |
| Dangerous check interactive | `check_cmd` uses `sudo` | `upp update` (TTY, user answers y) | Prompted before running the check, then runs |
| Dangerous check denied | `check_cmd` uses `sudo` | `upp update` (TTY, user answers n) | Check skipped; the run continues for the other tools |
| Dangerous check `--ci` | `check_cmd` uses `sudo` | `upp update --ci` | Deny message naming the tool, exit non-zero |

(Previously: `CustomAdapter.Check()` ran `check_cmd` with no trust check, no risk classification, and no privilege declaration — `Info()` declared only the update command's privileges. `upp list` therefore executed arbitrary shell despite its "Modifies System: No" contract.)

(Previously: the `--ci` scenarios required a privileged OFFICIAL row to fail closed — pacman's self-update and the apt sudo package group both exited non-zero, because the adapter was TrustOfficial and `EnforceRisk` was derived from the declaration. Two consequences followed from classifying by the declaration instead of the command: apt's self-update ran `sudo apt install --only-upgrade apt` without ever prompting (it declared no `Privileges`), while pacman's identical-shaped privileged row did prompt; and `--ci` was unusable on any Debian/Ubuntu host with pending apt updates. Confirmation now classifies by the real command for every row, and `--ci` distinguishes only by who ships the command.)

### Requirement: Official Command Versus Custom Command Under `--ci`

Interactive confirmation MUST classify every row by the real risk of the command it will execute, whatever the row's origin or trust: any command above RiskLow MUST prompt before running.

`--ci` cannot prompt, so it MUST distinguish by who ships the command:

- An **official** adapter's command is a fixed string inside the binary — written, reviewed and versioned by upp. A row whose `RiskCommand` is an official declaration proceeds under `--ci` regardless of its risk tier.
- A **custom** command comes from the user's config and cannot be vouched for. A custom row whose real risk is above `RiskLow` MUST fail closed with a non-zero exit, even when `trusted = true`.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Official privileged self-row in `--ci` | pacman or apt self-update, `sudo` command | `upp update --ci` | Proceeds; the command is shipped by upp |
| Official privileged owned group in `--ci` | apt group sudo package commands | `upp update --ci` | Proceeds; the command is shipped by upp |
| Custom privileged in `--ci` | `[custom.x]` whose update resolves to a `sudo` command | `upp update --ci` | Deny message, exit non-zero, not executed |
| Custom trusted privileged in `--ci` | same, `trusted = true` | `upp update --ci` | Still denied; trust does not waive the risk matrix |
| Official privileged interactive | apt self-update declaring no `Privileges` | `upp update` (TTY) | Prompts; classified by the real command, not the declaration |
| Custom low-risk in `--ci` | `[custom.x]` with a non-destructive command | `upp update --ci` | Proceeds unchanged |

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
