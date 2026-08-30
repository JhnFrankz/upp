# Delta for command-interface

## MODIFIED Requirements

### Requirement: upp update

`upp update` MUST process each enabled tool, execute updates, and report results. By default (bare `upp update` without `--manager`/`--update-group`), the command MUST execute manager-group bulk package updates for all owned tools grouped under their resolving package managers, alongside standalone tool updates. `--dry-run` (with shorthand `-n`) MUST show planned update actions—including planned manager group package updates and standalone tool updates—without executing any changes. `--only` and `--skip` MUST filter which tools to process.

In TTY runs (where stdin is a TTY, and `--ci`, `--quiet`, and `--dry-run` are not set), `upp update` MUST render the interactive tool selection over the `--only`/`--skip`-filtered pending set before executing; users MUST be able to toggle individual owned tools within manager groups as well as standalone tools. The user's selection MUST narrow the update set further. Flag semantics MUST NOT change: `--only`/`--skip` filter the candidate tools prior to presentation, and `--dry-run` MUST remain strictly non-interactive (no selector rendered).

When `--manager <mgr>` or `--update-group <mgr>` is explicitly supplied, `upp update` MUST restrict execution exclusively to the specified manager's resolving owned tools (minus any `--skip`-ed tools). Execution across tools and manager groups MUST maintain per-tool error isolation, ensuring that failures in individual package updates or standalone adapters do not halt execution of remaining tools. In `--ci` mode, any failure or unconfirmed elevated risk MUST cause the command to exit with a non-zero status after completing all non-dependent updates.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Normal default update | 5 tools enabled (apt owning gh/docker, plus npm, bun, nvm) | `upp update` | gh and docker updated via apt manager-group package updates, standalone tools updated, summary shown |
| Per-tool isolated failure | Tool 3 of 5 fails (e.g. gh in apt group) | `upp update` | Tools 1-2 updated, gh fails with isolated error, docker and standalone tools 4-5 attempted and updated |
| `--ci` failure exit | Tool fails during update in CI | `upp update --ci` | Non-dependent tools complete, exit non-zero, summary shows failures |
| `--ci` elevated risk fail-closed | Sudo package update required in CI | `upp update --ci` | Fails closed non-zero immediately without prompt (`EnforceRisk: true`) |
| Dry run full flag | 3 tools have updates (2 in brew group, 1 standalone) | `upp update --dry-run` | Lists planned actions for brew group packages and standalone tools, no changes made |
| Dry run short flag | 3 tools have updates | `upp update -n` | Behaves identically to `upp update --dry-run`, no changes made |
| Selector over filtered set | TTY, `--only brew,gh,npm` where brew owns gh | `upp update --only brew,gh,npm` | Selector lists brew group containing gh and standalone npm; other tools excluded |
| Granular selection in manager group | TTY, selector shows apt group with gh and docker pre-checked | User deselects docker | Only gh is updated via apt package update; docker is skipped; summary counts match selection |
| Dry-run non-interactive | TTY, `--dry-run`, pending updates | `upp update --dry-run` | No selector rendered; planned actions listed, no changes made |
| Explicit manager filter | Linux, apt owns gh/docker and standalone tools present | `upp update --manager apt` | apt's owned group (gh, docker) bulk-updated; standalone tools excluded |
| Explicit update-group filter | macOS, brew owns gh/docker/go | `upp update --update-group brew` | brew's owned group bulk-updated; standalone tools excluded |
| Skip excludes from default group | Linux, apt owns gh/docker | `upp update --skip docker` | Only gh batch-updated via apt; docker excluded |

(Previously: bare `upp update` executed standard per-tool adapter updates without manager-group bulk package updates; group bulk updates were strictly opt-in via `--manager` or `--update-group`.)
