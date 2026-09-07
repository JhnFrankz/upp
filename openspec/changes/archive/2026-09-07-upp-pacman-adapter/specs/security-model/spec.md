# Delta for security-model

## MODIFIED Requirements

### Requirement: Confirmation for Destructive Operations

The system MUST require explicit user confirmation before executing custom tool updates that involve:

- Privileged operations (sudo, admin)
- Destructive actions (rm, uninstall, overwrite)
- Network operations to untrusted sources

Confirmation MUST be classified by the REAL privileges and risk of the command to be executed, not by the tool's trust level alone. Owned-tool group updates MUST be reclassified by their real command risk: a manager-group bulk update that runs a privileged owned-package command (e.g. `sudo apt install --only-upgrade gh` or `sudo pacman -S --noconfirm <pkg>`) MUST prompt for confirmation even when each owned tool is `TrustOfficial`, because the group batch elevates real risk (sudo). A non-privileged group update (e.g. `brew upgrade gh`) MAY auto-proceed. Confirmation MUST display: action description, tool origin (custom), and required privileges. `--ci` MUST fail high-risk custom updates with a non-zero exit, even when `trusted = true`, and MUST fail a high-risk owned-tool group update that requires confirmation.

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

(Previously: confirmation scenarios explicitly cited apt for Linux sudo-heavy groups; pacman's sudo privilege requirements were not specified.)

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

## NEW Requirements (if any)
