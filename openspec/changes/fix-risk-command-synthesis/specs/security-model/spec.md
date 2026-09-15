# Delta for Security Model

## MODIFIED Requirements

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
