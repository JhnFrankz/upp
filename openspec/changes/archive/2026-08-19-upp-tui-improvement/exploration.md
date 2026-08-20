# Exploration: Interactive TUI selector for `upp update`

## Current State

**Rendering layer (`internal/output/render.go`)** — a single `Renderer` struct (`w io.Writer`, `color`, `emoji`, `quiet`, `verbose`, `mu sync.Mutex`) drives ALL terminal output. Color/emoji are detected once via `isTerminal(w)` (`os.File` + `ModeCharDevice`), with `NewRendererForced` for test injection. There is NO input handling anywhere in the output package — it is write-only.

**Update flow (`internal/cli/update.go`)** — `runUpdate` is fully SEQUENTIAL: for each filtered adapter: `Detect()` → progress line → `Check()` → security `ConfirmAction` (per-tool, risk-matrix driven) → policy gating → `Update()`. Confirmation today is ONLY the security-model matrix (`internal/security/confirm.go`): official tools auto-proceed, custom tools prompt per-risk-tier (`printInfo`/`promptUser` with `[y/N]` via `bufio.Scanner` on `os.Stdin`). There is NO whole-run confirmation, NO "which tools" selection UI, and `update` does NOT reuse the concurrent check engine.

**Check flow (`internal/cli/check.go`)** — `runCheck` uses a bounded worker pool (4–8, `runtime.NumCPU()` clamped) with deterministic index slotting (`results[job.index] = res`) and `Renderer.ProgressInPlace` (in-place `\r` on TTY, line-buffered fallback otherwise, mutex-synchronized). This is the ONLY interactive-feeling piece of terminal output today, and it is output-only. Its results (`StatusAvailable` + `CurrentVersion → LatestVersion`) are exactly the data a selector would need, but they are currently discarded by `update`.

**Existing interactive prompts** — three hand-rolled `[y/N]` patterns: `security.promptUser` (custom-tool risk), `cli.confirmReplace` (self-update, TTY-gated via `stdinIsTTY()`), and `cli.runInit`'s overwrite prompt (`fmt.Scanln`). All block on stdin, none are TTY-raw-mode, none render a menu.

**Dependencies (`go.mod`)** — ONLY `spf13/cobra` + `BurntSushi/toml` (+ indirect `mousetrap`, `pflag`). Zero TUI libraries, zero `x/term`, zero `isatty`. Confirmed by grep across the whole repo: no bubbletea/huh/tview/termui/promptui/survey/lipgloss/gocui/termenv anywhere.

**UX contract (`openspec/specs/ux-patterns/spec.md` + `command-interface/spec.md`)** — the feature MUST NOT break:
- **Default Interactive Mode**: official tools MUST NOT prompt; custom tools prompt only per the security-model risk matrix. `--ci` suppresses prompts; `--quiet` does NOT suppress prompts.
- **Summary Report**: deterministic canonical order, exact count/label semantics (`N updated`, `N would update` in dry-run, `N up to date, M skipped`, never "All clean!" with pending).
- **`--quiet` Verbosity**: one line per tool + summary; prompts still shown.
- **Progress Indication**: "Checking X/Y" (read-only) / "Updating X/Y" (update only), atomic under concurrency.
- **Self-Update Confirmation Prompt**: TTY-only, explicit deny, non-TTY `--ci` fails non-zero.
- **Global flags** (`--ci`, `--only`, `--skip`, `--dry-run`, `-q`, `-v`) must keep their documented semantics; `config-system` forbids any `interactive` config key or new prompt-behavior flag.

**Testing conventions** — strict TDD (`go test ./... -count=1`, `-race` in verify), table-driven, fake adapters (`fakeUpdateAdapter`, `fakeDelayedAdapter`, `fakePanickingAdapter`), `withCapturedStdout` / `probeHome` / `setCLIDeps` seams, `NewRendererForced` for output determinism. Smoke test (`scripts/smoke-test.sh`) runs non-interactively (no TTY) and must keep passing. `internal/cli` has NO `t.Parallel` tests (global `cliDeps` mutation is sequential-only).

## Affected Areas

- `internal/output/render.go` — add selector rendering (checkbox list, cursor, per-tool status/version, diff-ish lines), keep all existing methods byte-identical for existing tests.
- `internal/output/selector.go` (NEW, likely) — input handling: raw-mode key reading, ANSI escapes, non-TTY fallback; or delegate to a TUI lib.
- `internal/cli/update.go` — new pre-update phase: run concurrent checks (reuse `safeCheck`/worker pool from check.go), then if TTY && !`--ci` && !`--quiet` && !`--dry-run`, render selector and apply the chosen subset; wire chosen IDs into the existing per-tool loop (skipping unchanged). Decide how selection interacts with `--only`/`--skip` (filter is a pre-existing constraint; selection is a further narrowing).
- `internal/cli/check.go` — expose `safeCheck` + worker-pool runner as a reusable function (e.g. `runChecks(adapters) []output.ToolResult`) instead of duplicating it.
- `internal/cli/parser.go` — possibly a new flag (e.g. `--select`/`--interactive`? — see Risks: config-system forbids `interactive` KEY in config, not a flag; but flag semantics are spec'd in command-interface, so ANY new flag is a spec delta).
- `internal/cli/update_test.go`, `internal/cli/check_test.go`, `internal/output/render_test.go` — new table-driven tests: selection parsing, key handling via injectable reader, non-TTY fallback, `--ci`/`--quiet` bypass, determinism.
- `openspec/specs/ux-patterns/spec.md` (+ possibly `command-interface/spec.md`) — delta spec: new Requirement for the interactive selector, MUST NOT modify existing requirements.
- `go.mod` — ONLY if a TUI lib is adopted (currently not needed for the recommended approach).

## Approaches

1. **Hand-rolled ANSI checkbox selector inside `internal/output`** — extend the existing Renderer pattern: a `Selector` type reading raw key events from an injectable `io.Reader` (via `golang.org/x/term`'s `MakeRaw`/`ReadPassword`-style helper OR plain bufio for non-raw fallback), rendering a checkbox list with `↑/↓`, `Space` toggle, `a` select-all, `Enter` confirm, `Esc`/`q` cancel. Reuse the concurrent check results (with `CurrentVersion → LatestVersion` shown per row). Non-TTY/`--ci`/`--quiet`/`--dry-run` bypass the selector entirely and fall back to today's exact behavior.
   - Pros: zero new dependencies; reuses existing Renderer color/emoji/TTY detection and `NewRendererForced` test seam; fits package-per-layer layout; full control of output (byte-exact testable); matches the project's zero-dep philosophy; the "Updating X/Y" progress + Summary Report contracts stay untouched.
   - Cons: hand-rolled raw-mode input is fiddly (terminal state restore on panic/interrupt, Windows console caveats); needs `golang.org/x/term` for `MakeRaw` (a stdlib-adjacent dep, but still a new one); no free scrolling/paging polish — must build it; risk of subtle escape-code bugs on exotic terminals.
   - Effort: Medium.

2. **Add `charmbracelet/bubbletea` (+ lipgloss) and build a full TUI** — a real event-loop TUI with checkbox list, paging, maybe a detail/diff pane showing `current → latest` per tool.
   - Pros: industry-standard, actively maintained, gorgeous results, handles raw mode/alt-screen/resize/Windows for free; fast to build a rich UI; the repo's `ProgressInPlace` proves appetite for richer interactive output.
   - Cons: heavy dependency tree (bubbletea pulls `x/term`, `x/sys`, `lipgloss`, etc.) into a deliberately tiny `go.mod`; event-loop architecture is a big architectural step for a write-only Renderer; testing requires `tea.Program` simulation or model-unit tests that don't fit the existing `withCapturedStdout` pattern; cross-compile surface grows (Makefile `build-all` targets); overkill for a checkbox list + versions.
   - Effort: Medium-High.

3. **Minimal improvement to the existing confirm prompt** — no selector: change the sequential per-tool flow to show a compact numbered preview of pending updates (from concurrent pre-checks) and ask ONE `[y/N]` "Update N tools (brew, npm, ...)?" with a `n` + numbered-subset escape hatch (e.g. "enter numbers to pick, or y for all").
   - Pros: smallest change; zero new deps; preserves all existing contracts trivially; still kills the "confirm all or nothing" gap.
   - Cons: no arrow keys/checkbox UX the user asked for; subset selection via numbers is clunkier; less "wow"; still sequential update execution.
   - Effort: Low.

4. **Two-phase: concurrent pre-check + numbered list picker (no raw mode)** — reuse the check engine, print numbered rows of pending tools, ask "Which tools? (e.g. 1,3,5 or all):" reading a plain line; then run updates for the chosen subset.
   - Pros: zero new deps, no raw-mode risk, trivially testable (line-based input like existing prompts), reuses `safeCheck` directly, cross-platform safe (Windows cmd/ps1 line input), still a real improvement.
   - Cons: not a "real" TUI (no arrow keys); less discoverable than a checkbox; user asked explicitly for arrow-key checkbox UX.
   - Effort: Low-Medium.

## Recommendation

**Approach 1 — hand-rolled ANSI checkbox selector inside `internal/output`, gated on TTY && !`--ci` && !`--quiet` && !`--dry-run`, fed by a reused concurrent pre-check.** This is the only option that delivers the requested UX (arrow keys, per-tool toggling, versions inline) while honoring every hard constraint: `--ci` must stay non-interactive, `--quiet` must not add prompts, official tools must not be re-prompted (the selector is a user-initiated tool-choice UI, NOT a security confirmation — security.ConfirmAction still runs per-tool afterwards), Summary Report and Progress contracts stay intact, and go.mod stays lean (one stdlib-adjacent `golang.org/x/term` for `MakeRaw` at most, or even zero deps if we accept a read-byte loop on a TTY fd).

Structural shape: extract the concurrent check runner from `check.go` into a shared function (e.g. `runChecks(adapters) []output.ToolResult` in a new `internal/cli/checkrun.go` or inside check.go) so `update` can pre-check, render the selector from `StatusAvailable` results (`Name`, `CurrentVersion → LatestVersion`), and then run its existing per-tool update loop ONLY over the selected IDs — skipping tools the user deselected with `StatusSkipped`-like "not selected" reporting or silently, per spec decision. The security matrix and policy gating stay exactly as-is inside the loop. `--only`/`--skip` remain pre-filters; selection narrows further.

The selector itself: new `internal/output/selector.go` with a `CheckboxSelector` type taking `(w io.Writer, r io.Reader, options []SelectOption)`, injectable for tests exactly like `ConfirmConfig.Reader`. Key handling: `↑/↓`/`j`/`k` move, `Space` toggle, `a` all, `n` none, `Enter` confirm, `Esc`/`q` cancel (cancel = run nothing, exit 0, matching "decline" semantics of existing prompts). Non-TTY fallback = the selector never renders (callers gate on the same `isTerminal`/`stdinIsTTY` check self-update already uses).

## Risks

- **Raw-mode terminal handling**: must restore terminal state on every exit path (panic, signal, error). Mitigation: `defer` restore + `golang.org/x/term.MakeRaw`; add a RED test for terminal-state restore via injectable seam. Windows: `x/term` handles console raw mode, but arrow-key bytes on Windows consoles need `\x1b[A`/`\x1b[B` handling identical to Unix in practice — verify via a Windows CI target or document as best-effort.
- **Spec contract collisions**: any new flag or changed default needs a delta spec (command-interface flags are spec'd; ux-patterns Default Interactive Mode says prompts are driven by risk matrix + TTY). The selector MUST be framed as a TTY-driven user-choice UI, NOT a new prompt requirement, and MUST NOT be configurable via config key (config-system forbids `interactive` key). Esc/decline MUST NOT violate "never auto-proceed": declining runs nothing, exit 0.
- **Existing golden/summary tests**: update flow tests assert exact output; the pre-check phase adds new output lines in non-quiet TTY mode. Tests run non-TTY (captured stdout), so the selector branch is naturally skipped — but the pre-check "Checking X/Y" progress lines WILL appear in captured output unless gated on TTY too. Decision needed: pre-check progress shown only when the selector will actually render (TTY), or always shown (then existing tests must be updated deliberately).
- **Double check cost**: `update` today checks inside its loop; a pre-check phase runs `Check()` twice for gated tools (once for the selector, once in the loop). For gated tools (apt/npm/pnpm/nvm) the loop's check decides gating; pre-check results are advisory. Mitigation: carry pre-check results into the loop to skip the second `Check()` for selected tools (design decision; must not change gating semantics).
- **Dry-run interplay**: `update --dry-run` today shows planned actions WITHOUT executing — the selector must not appear in dry-run (it's a no-op preview; spec says dry-run is non-interactive). Decision documented above.
- **`internal/cli` sequential-only test constraint**: no `t.Parallel`; new selector tests must follow the same pattern (injectable reader, no global mutation).
- **Makefile build-all surface**: if `x/term` is added, cross-compilation targets must still pass (`go build` for all GOOS/GOARCH); `x/term` is pure-Go with build tags, low risk but must be verified in the build matrix.

## Ready for Proposal

**Yes** — the topic is well-scoped and the codebase is fully understood. The orchestrator should tell the user: a hand-rolled ANSI checkbox selector (Approach 1) is feasible within the existing Renderer + concurrent-check architecture with zero heavyweight deps; it will be gated on TTY + non-CI + non-quiet + non-dry-run so every existing contract (`--ci`, `--quiet`, dry-run, Summary Report, security matrix) stays intact; a concurrent pre-check phase reused from `check.go` feeds the selector; expect a delta spec for a new ux-patterns requirement plus possibly a command-interface note, ~Medium effort, with the two open design decisions (pre-check progress visibility in captured output, and double-check avoidance for gated tools) to resolve during proposal/design. The "diff view" desire maps to per-row `CurrentVersion → LatestVersion` inline (already available from `UpdateInfo`); a full side-by-side diff pane would push toward Approach 2 (bubbletea) and is NOT recommended for v1.
