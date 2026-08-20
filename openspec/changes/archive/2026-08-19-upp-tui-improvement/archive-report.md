# Archive Report: upp-tui-improvement

**Change**: upp-tui-improvement
**Archived**: 2026-08-19 (ISO prefix `2026-08-19-upp-tui-improvement`)
**Mode**: hybrid (OpenSpec filesystem + Engram)
**Project**: upp — Go binary, cobra CLI, TOML config
**Verdict at close**: PASS WITH WARNINGS — archived as intentional-with-warnings (non-blocking follow-ups, see below)

## Final State (at close)

- **Delivery**: 4 PRs merged to `main` — #91 selector (78b4b08), #92 runChecks (a2ae8eb), #93 interactive update (b4aec36), #94 docs (1c6b016). Main HEAD at archive time: `accd1f1`.
- **Tasks**: 21/21 complete (`tasks.md` has zero unchecked implementation tasks; 21 `[x]`).
- **Verification**: `pass_with_warnings`, 0 CRITICAL, 0 blockers, 2/2 requirements, 18/18 scenarios compliant, evidence `sha256:4d83880e…` (validated by `gentle-ai sdd-verify-validate`). Runtime evidence: `go test ./... -count=1` 9/9 packages exit 0; `-race` exit 0; `go vet ./...` exit 0; golangci-lint exit 0 (misspell locale US); smoke 31/31 non-TTY byte-identical; `make build-all` exit 0 (linux/darwin/windows amd64+arm64, Windows x/term raw-mode compiles).
- **Design conformance**: all 9 architecture decisions (D1–D9) followed, per `verify-report.md`.

## Warnings and Follow-ups (non-blocking, recorded for future work)

1. **Deferred manual TTY check (verify-report WARNING 1, task 4.4)**: task 4.4 is marked `[x]` WITH CAVEAT — zero pending updates on the dev machine at verification time, so the live selector (arrow keys, Space, `a`/`n`, Enter, Esc/`q`, raw-mode restore) could not be exercised end-to-end by a human. Covered by 27 unit tests + smoke non-TTY byte-identical (31/31) + build-all Windows raw-mode compile. **Open follow-up**: confirm live behavior on the first real pending update. NOT an incomplete task.
2. **Coverage suggestion (verify-report SUGGESTION 1)**: `processSelectedOutcome` interactive-path branches (ConfirmDeny/ConfirmError/policy-gate-bypass/update-error, update.go:402-414, 422-429, 433-441, 452-462) at 53.6% coverage; identical logic covered via `runUpdateSequential` (84.6%) and the gating matrix. **Open follow-up**: add a direct interactive test (selected tool denied/errored) to harden the carried-outcome loop.
3. **Composition test suggestion (verify-report SUGGESTION 2)**: no single end-to-end test combines `--only`/`--skip` with the TTY seam (filter + selector composition); component tests cover both halves separately. **Open follow-up**: add `runUpdate(&GlobalFlags{Only: ...})` with `stdinIsTTY: true` asserting selector options.
4. **Cosmetic doc inconsistency (verify-report SUGGESTION 3)**: `design.md` and `tasks.md` use British "cancelled" while code/renderer use US "canceled" (repo lint enforces misspell locale US). Left as archived — the archive is an audit trail and is not modified; code is correct.

## Spec Sync (delta → main specs)

| Domain | Action | Result |
|--------|--------|--------|
| `ux-patterns` | ADDED Requirement "Interactive Update Tool Selection" (10 scenarios) | Appended to `openspec/specs/ux-patterns/spec.md` (12 → 13 requirements); all other requirements preserved byte-identical |
| `command-interface` | MODIFIED Requirement "`upp update`" (3 scenarios added, 1 previously-note updated) | Replaced block in `openspec/specs/command-interface/spec.md` (7 → 7 requirements, same count); all other requirements preserved byte-identical |

Both merges performed mechanically (shell byte assembly from delta files, no model reproduction of content) and verified: merged delta blocks are byte-identical to the delta spec files (`diff` empty), surrounding context preserved.

## Archive Move

`openspec/changes/upp-tui-improvement/` → `openspec/changes/archive/2026-08-19-upp-tui-improvement/` via `mv` (fallback: change folder untracked, `git mv` refused). Pre-move recursive snapshot compared with `diff -r` after the move: **EMPTY** (byte-identical). Archive contains all artifacts: proposal.md, exploration.md, specs/ux-patterns/spec.md, specs/command-interface/spec.md, design.md, tasks.md, verify-report.md. Active changes directory no longer contains this change.

## Native Review Receipt Gate

`reviewGate` is structurally ABSENT: `gentle-ai review status` returned `applicability: unrelated` with `paths: []` (no pending code changes; all 4 PRs merged). Archive proceeded under ordinary repository policy. No review topics exist for this candidate.

## Task Completion Gate / Stale-Checkbox Reconciliation

- Persisted filesystem `tasks.md` (source of truth for hybrid): 0 unchecked, 21/21 `[x]`, 4.4 checked with caveat. Gate passed without repair.
- Engram observation #324 (`sdd/upp-tui-improvement/tasks`, rev 5, written 2026-08-19 15:35:54) was an intermediate snapshot showing task 4.4 unchecked. **Reconciliation**: updated to final state (4.4 `[x]` with caveat, matching filesystem `tasks.md`) — sanctioned by the orchestrator's explicit final-state facts plus `verify-report.md` (21/21) and `apply-progress` (Engram #325) proving completion. Engram preserves prior revisions, so the intermediate snapshot remains in history.

## Engram Observations Read (traceability)

| Observation ID | Topic | Role |
|----------------|-------|------|
| 320 | `sdd/upp-tui-improvement/explore` | Exploration (read for context) |
| 321 | `sdd/upp-tui-improvement/proposal` | Proposal |
| 322 | `sdd/upp-tui-improvement/spec` | Delta specs (ux-patterns + command-interface) |
| 323 | `sdd/upp-tui-improvement/design` | Design |
| 324 | `sdd/upp-tui-improvement/tasks` | Tasks (stale snapshot; reconciled — see above) |
| 325 | `sdd/upp-tui-improvement/apply-progress` | Apply progress (intermediate snapshot; completion proven by higher-ranked sources) |
| 327 | `sdd/upp-tui-improvement/verify-report` | Verify report (pass_with_warnings) |

Per the Final-State Authority, intermediate snapshots (apply-progress #325, tasks #324 rev 5) describe state at their write time; the final-state facts above come from the orchestrator's launch prompt, the persisted `tasks.md`, and `verify-report.md` (validated), which outrank them.

## Archive Persistence

- Filesystem: `openspec/changes/archive/2026-08-19-upp-tui-improvement/archive-report.md` (additive, excluded from `diff -r` readback)
- Engram: `sdd/upp-tui-improvement/archive-report` (type architecture, capture_prompt false)

## SDD Cycle Status

**Complete.** The change was planned (explore/propose), specified (2 delta specs), designed (9 decisions), implemented (21 tasks, 4 chained PRs merged), verified (pass_with_warnings, 0 CRITICAL), and archived. Source of truth (`openspec/specs/`) reflects the new behavior. Ready for the next change.
