# Delta for Command Interface

## ADDED Requirements

### Requirement: Canonical Tool Selection Identity

In TTY `upp update`, the selection-to-adapter mapping MUST use the canonical stable tool identity `adapters.ToolInfo.ID`, falling back to `adapters.Adapter.Name()` ONLY when `Info().ID` is empty. Selector option identifiers MUST carry this same canonical identity. The selector display label MUST remain the human-facing `ToolInfo.Name` and MUST NOT be used as the selection key. Every selected option MUST resolve to exactly one adapter, and an option that resolves to no adapter MUST be surfaced with an explicit failure result rather than silently dropped.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Distinct ID and label | Adapter `Info().ID="gh"`, `Info().Name="GitHub CLI"` | Selector renders | Option `ID="gh"`, `Label="GitHub CLI"` |
| Selection maps to adapter | Option `ID="gh"` selected | Update executes | `gh` adapter resolved and updated; no silent drop |
| Empty ID fallback | Adapter `Info().ID=""` | Selection mapping | Identity falls back to `Adapter.Name()` |
| Unresolvable selection | Selected ID matches no adapter | Update executes | Tool surfaced as an explicit failure result, never a silent skip |

### Requirement: Deselected Pending Tools Reporting

In TTY `upp update`, pending tools presented in the selector but not selected by the user MUST be reported with a distinct status in the final summary, separate from `StatusSkipped` (not-installed / confirm-deny) and from `StatusCurrent`. Deselected pending tools MUST NOT be silently dropped, MUST NOT be counted as skipped, and MUST NOT be counted as updated.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Deselected tool reported | 3 pending tools, user selects 2 | Update finishes | Summary reports the deselected tool under a distinct deselected status with its own count |
| Distinct from skipped | 1 deselected pending tool, 1 not-installed tool | Update finishes | Skipped count includes only the not-installed tool; deselected counted separately |
| Not silently dropped | Any pending tool deselected | Update finishes | The tool appears in the summary detail; no omission |
| All pending deselected | User deselects every pending tool | Update finishes | Summary reports all pending tools as deselected; nothing updated |

## MODIFIED Requirements

### Requirement: `upp update`

`upp update` MUST process each enabled tool, execute updates, and report results. The command MUST delegate tool adapter resolution, concurrent version checking, and update plan formulation to `engine.Engine`. By default (bare `upp update`), the command MUST execute manager-group bulk package updates for all owned tools grouped under their resolving package managers, alongside standalone tool updates. `--dry-run` (with shorthand `-n`) MUST show planned update actions—including planned manager group package updates and standalone tool updates—without executing any changes. `--only` MUST filter which tools to process.

In TTY runs (where stdin is a TTY, and `--ci`, `--quiet`, and `--dry-run` are not set), `upp update` MUST derive the pending set from `engine.Plan` (`plan.Updates`) — not from per-tool availability alone — so `PolicyAlwaysUpdate` tools appear even when currently up to date, and MUST render the interactive tool selection over that set before executing. Selection identifiers MUST use the canonical stable identity (see Canonical Tool Selection Identity); labels are display-only. Users MUST be able to toggle individual owned tools within manager groups as well as standalone tools. The user's selection MUST narrow the update set further; selected tools MUST execute from the same `PlannedUpdate` metadata (risk, policy, privileges, manager) the sequential path consumes, and the interactive path MUST NOT re-implement that policy. Pending tools left deselected MUST be reported per Deselected Pending Tools Reporting. Flag semantics MUST NOT change: `--only` filters the candidate tools prior to presentation, and `--dry-run` MUST remain strictly non-interactive (no selector rendered).

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
| Plan-derived pending set | TTY, brew current (`UpdateAvailable=false`), npm has an update | `upp update` | Selector lists both brew (AlwaysUpdate) and npm; pending set equals `plan.Updates` |
| Canonical identity selection | TTY, adapter `Info().ID="gh"`, `Info().Name="GitHub CLI"` | User selects gh and confirms | gh adapter resolved by ID and updated; no silent drop |
| Granular selection in manager group | TTY, selector shows apt group with gh and docker pre-checked | User deselects docker | Only gh is updated via apt package update; docker is reported under the distinct deselected status; summary counts match selection |
| Dry-run non-interactive | TTY, `--dry-run`, pending updates | `upp update --dry-run` | No selector rendered; planned actions listed, no changes made |
| `--manager` rejected | Update running | `upp update --manager apt` | Error: unknown flag "manager", usage hint, exit non-zero |
| `--update-group` rejected | Update running | `upp update --update-group brew` | Error: unknown flag "update-group", usage hint, exit non-zero |
| Engine delegation seam | Update invoked | `upp update` | Resolves adapters, executes concurrent pre-checks, and builds update plan via `engine.Engine` |

(Previously: the interactive path built its pending set from `oc.Status == StatusAvailable` only, keyed selection by display name (`oc.ToolName`) while the adapter map keyed by short ID (`Adapter.Name()`), silently dropped on mismatch, and re-implemented risk/policy in `processSelectedOutcome`; deselected pending tools were dropped with no status. It now derives the pending set and per-tool execution metadata from `engine.Plan` and reports deselection distinctly.)
