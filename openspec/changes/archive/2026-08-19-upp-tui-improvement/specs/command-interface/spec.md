# Delta for command-interface

## MODIFIED Requirements

### Requirement: `upp update`

`upp update` MUST process each enabled tool, execute updates, and report results. `--dry-run` (with single-letter shorthand `-n`) MUST show planned actions without executing. `--only`/`--skip` filter which tools to process.

In TTY runs (stdin is a TTY, and `--ci`, `--quiet`, and `--dry-run` are not set), `upp update` MUST render the interactive tool selection over the `--only`/`--skip`-filtered pending set before executing; the user's selection MUST narrow the update set further. Flag semantics MUST NOT change: `--only`/`--skip` filter exactly as before, and `--dry-run` MUST remain non-interactive (no selector rendered).

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Normal update | 5 tools enabled | `upp update` | Each tool updated, summary shown |
| Partial failure | Tool 3 of 5 fails | `upp update` | 1-2 updated, 3 failed, 4-5 attempted |
| `--ci` failure | Tool fails in CI | `upp update --ci` | Exit non-zero, summary shows failures |
| Dry run full flag | 3 tools have updates | `upp update --dry-run` | Lists planned actions, no changes |
| Dry run short flag | 3 tools have updates | `upp update -n` | Behaves identically to `--dry-run`, no changes |
| Selector over filtered set | TTY, `--only brew,npm`, 2 pending | `upp update --only brew,npm` | Selector lists only brew and npm; other tools not shown |
| Selection narrows further | TTY, selector shows 3 pre-checked | User deselects 1 tool | Only the 2 selected tools updated; summary counts match selection |
| Dry-run non-interactive | TTY, `--dry-run`, pending updates | `upp update --dry-run` | No selector; planned actions listed, no changes |

(Previously: `--dry-run` had no single-letter shorthand `-n`; TTY runs processed the filtered set directly with no interactive selection step.)
