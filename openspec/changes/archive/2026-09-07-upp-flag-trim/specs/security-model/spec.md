# Delta for security-model

## MODIFIED Requirements

### Requirement: Confirmation for Destructive Operations

The system MUST require explicit user confirmation before executing custom tool updates that involve:

- Privileged operations (sudo, admin)
- Destructive actions (rm, uninstall, overwrite)
- Network operations to untrusted sources

Confirmation MUST be classified by the REAL privileges and risk of the command to be executed, not by the tool's trust level alone. Owned-tool group updates MUST be reclassified by their real command risk: a manager-group bulk update that runs a privileged owned-package command (e.g. `sudo apt install --only-upgrade gh`) MUST prompt for confirmation even when each owned tool is `TrustOfficial`, because the group batch elevates real risk (sudo). A non-privileged group update (e.g. `brew upgrade gh`) MAY auto-proceed. Confirmation MUST display: action description, tool origin (custom), and required privileges. `--ci` MUST fail high-risk custom updates with a non-zero exit, even when `trusted = true`, and MUST fail a high-risk owned-tool group update that requires confirmation.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Custom privileged | Custom tool uses `sudo` | Update requested | Prompt: "This will run `sudo apt install ...` for mytool. Allow? [y/N]" |
| Custom destructive | Custom tool runs `rm -rf` | Update requested | Prompt with warning, requires explicit yes |
| `--ci` high-risk | Custom tool needs confirmation | `upp update --ci` | Exits non-zero: "requires confirmation" — trust does not waive it |
| `--ci` trusted high-risk | `trusted = true`, uses `sudo` | `upp update --ci` | Exits non-zero; confirmation cannot be waived in non-interactive mode |
| Sudo-heavy group prompts | Linux, apt owned tools (gh/docker) use `sudo apt install --only-upgrade` | `upp update` (default run, apt group in selection) | Prompts for confirmation despite TrustOfficial owned tools |
| Non-sudo group proceeds | macOS, brew owned tools use `brew upgrade` (no sudo) | `upp update` (default run, brew group in selection) | Group update proceeds without confirmation |
| `--ci` sudo group fails | Linux, `--ci`, apt group sudo package commands | `upp update --ci` | Exits non-zero for the sudo-heavy group; group not executed |

(Previously: the sudo-heavy and non-sudo group confirmation scenarios were triggered by explicitly invoking the opt-in `--manager apt` / `--manager brew` flags; confirmation semantics are unchanged and now apply to the default delegated group path.)
