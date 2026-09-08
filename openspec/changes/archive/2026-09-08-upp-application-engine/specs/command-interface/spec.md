# Delta for command-interface

## MODIFIED Requirements

### Requirement: `upp update`

`upp update` MUST process each enabled tool, execute updates, and report results. The command MUST delegate tool adapter resolution, concurrent version checking, and update plan formulation to `engine.Engine`. By default (bare `upp update`), the command MUST execute manager-group bulk package updates for all owned tools grouped under their resolving package managers, alongside standalone tool updates. `--dry-run` (with shorthand `-n`) MUST show planned update actions—including planned manager group package updates and standalone tool updates—without executing any changes. `--only` MUST filter which tools to process.

In TTY runs (where stdin is a TTY, and `--ci`, `--quiet`, and `--dry-run` are not set), `upp update` MUST render the interactive tool selection over the `--only`-filtered pending set before executing; users MUST be able to toggle individual owned tools within manager groups as well as standalone tools. The user's selection MUST narrow the update set further. Flag semantics MUST NOT change: `--only` filters the candidate tools prior to presentation, and `--dry-run` MUST remain strictly non-interactive (no selector rendered).

Execution across tools and manager groups MUST maintain per-tool error isolation, ensuring that failures in individual package updates or standalone adapters do not halt execution of remaining tools. In `--ci` mode, any failure or unconfirmed elevated risk MUST cause the command to exit with a non-zero status after completing all non-dependent updates. `--manager` and `--update-group` MUST NOT exist as flags; supplying either MUST produce the default unknown-flag rejection (error + usage, non-zero exit).

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
| `--manager` rejected | Update running | `upp update --manager apt` | Error: unknown flag "manager", usage hint, exit non-zero |
| `--update-group` rejected | Update running | `upp update --update-group brew` | Error: unknown flag "update-group", usage hint, exit non-zero |
| Engine delegation seam | Update invoked | `upp update` | Resolves adapters, executes concurrent pre-checks, and builds update plan via `engine.Engine` |

(Previously: `upp update` directly coordinated discovery, concurrent check execution, and planning within `internal/cli` loops; it now delegates resolution, checking, and planning to `engine.Engine`.)

## NEW Requirements

### Requirement: Orchestration Delegation

The `list` and `update` commands MUST delegate tool adapter discovery, resolution, check dispatching, and update plan formulation to `engine.Engine`.

1. `upp list` MUST invoke `engine.Resolve(filter)` to obtain active, enabled adapters for the current platform and render the resulting groups via `output.GroupByOwner`.
2. `upp update` MUST invoke `engine.Resolve(filter)` for tool discovery, `engine.Check(ctx, adapters, onProgress)` for concurrent version checking, and `engine.Plan(outcomes, filter)` for update action formulation.
3. The presentation layer (`internal/cli`) MUST consume pure domain models (`CheckProgress`, `CheckOutcome`, `UpdatePlan`) emitted by the engine and map them to presentation renderers (`output.CheckBoard`, `output.Renderer`, `output.CheckboxSelector`) without altering command flags (`--quiet`/`-q`, `--verbose`/`-v`, `--ci`, `--only`, `--dry-run`/`-n`), visual layouts, exit codes, or interactive selection workflows.
4. Tool execution and security confirmation (`security.ConfirmAction`) MUST remain in the CLI presentation layer.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| List command delegation | Host platform with enabled tools | `upp list` | Invokes `engine.Resolve(filter)` to obtain active adapters and renders table via `output.GroupByOwner` |
| Update command delegation | Host platform with enabled tools | `upp update` | Invokes `engine.Resolve()`, `engine.Check()`, and `engine.Plan()`, bridging progress to `output.CheckBoard` |
| Flag preservation | Command passed `-q`, `-v`, `--ci`, `--only` | Command execution | Flags parsed by CLI and passed as domain `Filter` to engine; visual flags applied to CLI renderer |
| Exit code preservation | Update encounters failed check or unconfirmed CI risk | Command execution | Command exits with non-zero status identical to previous behavior |
| Headless engine compatibility | CLI presentation disabled or piped stdout | Command execution | Engine executes headless resolution, checking, and planning without presentation dependencies |
