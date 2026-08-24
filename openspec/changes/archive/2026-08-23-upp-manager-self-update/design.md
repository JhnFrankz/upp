# Design: Package Manager Self-Update (brew/apt/winget/scoop)

## Technical Approach

Repurpose the four existing manager adapters **in place** — no new interface, IDs, or config (point-6 boundary). Managers are already enabled tools per platform; each is a plain Adapter row whose `Check()` reports the manager's own version/self-update availability and `Update()` runs only the self-update command. All plumbing works unchanged (board, selector, `--only`/`--skip`, policy gate, `-n`). Scoop already exists — no new files; same Windows-parity treatment.

## Architecture Decisions

| # | Decision | Alternatives | Rationale |
|---|---|---|---|
| 1 | Repurpose adapters in place | New `*-self` IDs; new Manager kind | Zero config/interface change; clean point-6 boundary; delta + release note cover behavioral change |
| 2 | brew Check current-only; `Update()` = `brew update` ONLY | `brew upgrade brew` | "Homebrew 4.x.y" → `extractVersionFromString`; `brew update` mutates — never in Check; `brew upgrade brew` non-canonical (portable-ruby footgun) |
| 3 | apt Check unchanged; `Update()` = `sudo apt install --only-upgrade apt` | Full `apt upgrade` | Check already reads Installed/Candidate; Gated + sudo stay; row = "apt package stale" |
| 4 | winget Check = version + own-row parse; `Update()` = `winget upgrade winget` | Always-available stub (today) | Real availability; AlwaysUpdate stays; `isVersionLike` strips leading letters ("v1.8.2311" ✓); < 1.6 → unavailable, no error |
| 5 | scoop Check = `scoop status` own-row parse | Current-only (`scoop --version`) | `scoop --version` reports a script commit hash; status-parse = winget parity; `Update()` = `scoop update scoop` (never `*`) |
| 6 | Interactive brew gap accepted (D7) | Force brew into selector; run `brew update` in Check | brew never available → never pending in TTY; sequential/`--ci` runs it via PolicyAlwaysUpdate |

## File Changes

| File | Action | Description |
|---|---|---|
| `official/brew.go` | Modify | Update → `brew update` only; footgun comment; Check unchanged |
| `official/apt.go` | Modify | Update → `sudo apt install --only-upgrade apt`; Check unchanged |
| `official/winget.go` | Modify | Check parses version + own row (graceful <1.6); Update → `winget upgrade winget` |
| `official/scoop.go` | Modify | Check parses `scoop status`; Update → `scoop update scoop` |
| `official/helper.go` | Modify | Extract pure `parseWingetUpgradeOutput` / `parseScoopStatusOutput` |
| `official/{check,update,parity}_test.go` | Modify | New semantics + command keys; leading-v + scoop row cases |
| `cli/update_test.go` | Modify | Gating-matrix comment; dry-run + brew-never-pending assertions |
| Unchanged | — | `cli/list.go`, `output/render.go`, `registry.go`, `catalog.go`, `checkrun.go`, `update.go` — manager rows already render Check() version |
| `openspec/specs/{tool-adapter,ux-patterns}/spec.md` | Modify | See Spec-Delta Notes |

## Interfaces / Contracts

No interface changes. Pure helpers (fail-closed: unparseable → `found=false`, `UpdateAvailable=false`): `parseWingetUpgradeOutput(out) (current, latest, found)` (matches winget/`Microsoft.AppInstaller` row); `parseScoopStatusOutput(out) (current, latest, found)` (scoop own row). Version reuse: `isVersionLike` strips leading letters ("v1.8.2311" ✓); `extractVersionFromString` handles "Homebrew 4.x.y".

## Testing Strategy

Hermetic via existing `execFakes` seam; strict TDD (`go test ./... -count=1`).

| Layer | What | Approach |
|---|---|---|
| Unit | Manager Check+Update | Table rows in `check_test.go`/`update_test.go`; `failIfRun` guards dry-run and "no `brew update` inside Check" |
| Unit | Parsing | `parity_test.go` + check rows: `v1.8.2311`, no-own-row (old winget), scoop with/without own row, apt Installed/Candidate |
| Unit/Integration | CLI `-n` + TTY | `cli/update_test.go` `fakeUpdateAdapter`: brew current in `-n` / never pending; apt/winget planned when available; `TestRunUpdate_GatingMatrix` unchanged |

Spec→test: "brew self-update" → row keyed `"brew update"`; "apt gated sudo" → `sudo apt install --only-upgrade apt` + `Privileges:["sudo"]`; "winget tolerant parse" → `v1.8.2311` + row → available; "winget old version" → no row → false; "scoop parity" → row → available + `scoop update scoop`; "brew current-only" → Check row + failIfRun in check fakes.

## Threat Matrix

N/A — no routing, shell, subprocess, VCS/PR automation, executable-file classification, or process-integration boundary; only command strings change within the existing bounded exec seam.

## Migration / Rollout

No migration. Rollback = revert four adapters to pre-change commands (proposal Rollback Plan).

## Spec-Delta Notes (for archive)

1. tool-adapter catalog table: manager rows → `apt install --only-upgrade apt` / `brew update` / `winget upgrade winget` / `scoop update scoop`; brew scenario "self-only; never `brew upgrade brew`"; add apt/winget/scoop self-only scenarios; `(Previously:)` old commands.
2. tool-adapter Update Gating: reword — brew now the only always-update adapter reporting `false` by design; winget/scoop detect real availability; Gated-check-fails kept.
3. tool-adapter ADDED "Manager Self-Update Semantics" (table + 8 scenarios).
4. ux-patterns ADDED "Manager Self-Update Row Rendering": brew current on board/`list`/`-n`, never in TTY selector; apt/winget planned in `-n`.
5. security-model line 67 unchanged — all four commands are platform-native PMs.

## Open Questions

- [ ] `scoop status` output shape on Windows (stderr WARN + stdout table assumed); fallback: current-only if unstable — confirm in apply.
- [ ] Winget own-row match key (`Microsoft.AppInstaller` vs "winget"); helper placement — decide in apply.
