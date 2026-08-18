# Design: Simplify upp Usage — Honest Output and Zero-Config Story

## Technical Approach

Presentation + data-quality fixes on existing render/CLI seams, as 2 chained PRs under the 400-line budget. **S1 (PR #1, UX)**: progress verb, dry-run "All clean!" gating, `list` ID column, help grouping + hidden `completion`, README zero-config quickstart, ux-patterns drift fix. **S2 (PR #2, data)**: honest skip counts in `check` summary, nvm semver, `settings.interactive` removal. Slices touch disjoint renderer functions — no string overlap. Implements deltas ux-patterns, command-interface, tool-adapter, config-system.

## Architecture Decisions

| # | Option / Tradeoff | Decision |
|---|---|---|
| D1 | Slice overlap breaks independent revert | S1 owns `Progress` (:219-225), `UpdateSummary` (:230-278), `ListTools` (:348-364); S2 owns `CheckSummary` (:300-343). `countByStatus`/`detailSummary` untouched |
| D2 | Hardcoded verb leaks string out of renderer | `Progress(operation, current, total, name)` (:219); check.go:88 `"Checking"`, update.go:94 `"Updating"`; template `"  ⟳ %s %d/%d: %s\n"` (:223). Renderer owns all templates |
| D3 | Dry-run with pending would claim clean | `allClean := !summary.DryRun && updated>0 && available==0 && failed==0 && skipped==0` (:266-269); dry-run renders "N would update", never the tagline; all-skipped branch (:259-262) unchanged |
| D4 | "skipped" semantics must be explicit | `skipped = StatusSkipped` only (check.go:79 `!a.Detect()`; `--only/--skip` filter BEFORE the loop, never counted). "All tools up to date." ONLY when `current>0 && available==0 && skipped==0 && failed==0`, or empty list (test-enforced, parser_test.go:238 / integration_test.go:647). Skips → `"N up to date, M skipped"`; all-skipped → "Nothing to do."; non-quiet detail lists skipped |
| D5 | Filter IDs never displayed | `ListEntry` gains `ID` (:366-371); list.go:68-72 fills from `info.ID` (`--only`/`--skip` key); header (:350-351), rows (:360-361) → `ID | Name | Status | Version` |
| D6 | Manual help template vs built-ins | cobra `AddGroup` (Tool/Config/Maintenance) + per-command `GroupID` (parser.go:58-68); root `CompletionOptions{HiddenDefaultCmd: true}` (:37-47); hermetic help test |
| D7 | Reuse `selfupdate.Parse/Compare` (version.go:28,109) — needs `v`, fail-closed | `semverCompare(cur, latest string) bool` (nvm.go:66): normalize `v`; `Parse` both (`Dev`→false); `Compare(latest)<0` → true. Unparseable (e.g. `stable`) → `false`, no error — confirms adapter Unparseable scenario. Scope = comparison only; equality-based adapters untouched |
| D8 | Ignore vs delete dead setting | Delete `Settings.Interactive` (config.go:26) + default (:51). BurntSushi toml ignores unknown keys → stale key loads silently, never re-emitted by export/init/import. No validation/migration impact |
| D9 | List Table home | **ux-patterns** — rendered column format is output; command-interface owns surface/help. Matches delta placement |
| D10 | Drift fix at archive vs direct | **Direct, in S1** — precedent PR #56 (caf39b2) fixed canonical spec directly; archive merge is idempotent with the delta |

## Data Flow

```
nvm: raw cur/latest ─▶ semverCompare() ─▶ parse fail(either)▶ false (no error) │ cur<latest▶true │ else▶false
check: Detect()false ▶ StatusSkipped ▶ CheckSummary ▶ "N up to date, M skipped"
list: Info().ID ▶ ListEntry{ID} ▶ "ID | Name | Status | Version"
```

## File Changes

| File | Action | Description |
|---|---|---|
| `internal/output/render.go` | Modify | S1: Progress param (:219-225), clean-gate (:266-272), ListTools ID (:348-364), ListEntry.ID (:366-371). S2: CheckSummary skipped (:300-343) |
| `internal/cli/check.go`, `update.go`, `list.go` | Modify | S1: `Progress("Checking"/"Updating", …)` (:88, :94); `ListEntry.ID` from `info.ID` (:68-72) |
| `internal/cli/parser.go` | Modify | S1: Groups + GroupIDs + `HiddenDefaultCmd` (:37-68) |
| `internal/adapters/official/nvm.go` | Modify | S2: `semverCompare` replaces `current != latest` (:66) |
| `internal/config/config.go` | Modify | S2: drop `Settings.Interactive` (:25-31, :51) |
| `README.md` | Modify | S1: zero-config quickstart (:55-72) — install→list/check/update, `init` optional |
| `openspec/specs/ux-patterns/spec.md` | Modify | S1: "localized (en/es)" → English-only (:102) |
| `internal/output/render_test.go`, `internal/cli/parser_test.go` | Modify | S1: progress verb, no-clean dry-run, List header/ID, help-group + hidden-completion tests |
| `internal/output/render_test.go` | Modify | S2: check-with-skips summary tests |
| `internal/adapters/official/check_test.go` | Modify | S2: nvm rows — newer-current (no downgrade), unparseable `stable` (no error) |
| `internal/config/config_test.go`, `config_expanded_test.go`, `internal/cli/integration_test.go` | Modify | S2: drop `Settings.Interactive` assertions |
| `scripts/smoke-test.sh` | Modify | S2: test 13 `interactive = false` grep (:244-246) → assert key absent |

## Interfaces / Contracts

```go
func (r *Renderer) Progress(op string, cur, total int, name string)
type ListEntry struct { ID, Name string; Status Status; Version string }
func semverCompare(cur, latest string) bool // false if either unparseable/"dev"
type Settings struct { CheckSelfUpdate bool `toml:"check_self_update"` }
```

## Testing Strategy

| Layer | What to Test | Slice | Approach |
|---|---|---|---|
| Unit render | Progress verb; dry-run never "All clean!"; List ID header; CheckSummary "N up to date, M skipped" | S1+S2 | render_test.go rows (Strict TDD: RED first) |
| Unit adapters | nvm newer-current=false; unparseable=false no error; equal=false | S2 | check_test.go rows (3 existing nvm rows keep passing) |
| Unit config | default/round-trip drop Interactive; stray key tolerated | S2 | config tests assertion removal |
| Integration | help groups + completion hidden; hermetic check-with-skips | S1/S2 | parser_test.go, integration_test.go |
| E2E | smoke: S1 help strings unchanged; S2 export emits no `interactive` | both | `scripts/smoke-test.sh --skip-build` |

## Threat Matrix

| Boundary | Applicability |
|---|---|
| Shell-subprocess launch | N/A — nvm command strings unchanged static literals; only comparison of their output changes |
| VCS/PR automation / executable classification | N/A — no git/PR/exec operations; canonical spec + README are docs text |

## Migration / Rollout

No data migration. Configs with a stale `interactive` key load silently; the key is no longer re-emitted. Rollback: per-slice `git revert` (S1 ≤250 lines, S2 ≤150).

## Open Questions

- [ ] Keep `"All tools up to date."` for empty enabled-tool lists (spec-compliant, test-enforced).
- [ ] Confirm S1 carries the canonical ux-patterns drift fix (direct PR, PR #56 precedent).