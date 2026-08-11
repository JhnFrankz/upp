# Delta for security-model

## MODIFIED Requirements

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
