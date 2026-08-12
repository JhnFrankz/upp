# Exploration: upp versioning & automatic updating

Date: 2026-08-12 · Change: `upp-versioning-auto-update` · Phase: explore (read-only)
Status: fresh exploration — prior change `upp-versioning-self-update` (same topic family) was CANCELLED by the user; its locked decisions are DEAD and are NOT inherited. Verified facts from that exploration (Engram #108) were reused where re-verified against the live tree.

## Current State (verified 2026-08-12 against source, Makefile, git)

### Versioning
- `cmd/upp/main.go:12` — `var version = "dev"`; injected via Makefile `LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"`, `VERSION = git describe --tags --always --dirty`. `root.Version = version` (main.go:16) → cobra prints `upp version {version}`.
- Current describe: `v0.1.0-19-gd40e428-dirty` (19 commits past tag v0.1.0, dirty tree). Only tag: `v0.1.0`.
- NO semver parsing or version comparison anywhere in the codebase. No `version` subcommand (only the flag).
- Runtime version string is parseable: clean release builds (built AT the tag) produce exactly `v0.1.0`; untagged/dev builds carry `-N-gHASH` and/or `-dirty` suffixes, or `dev` when ldflags are absent.

### Command surface (verified: internal/cli/parser.go:58-67 AddCommands)
- Root RunE → `runCheck` (bare `upp` = status). Subcommands: `init`, `update` (TAKEN — updates TOOLS, has `--dry-run`), `check`, `list`, `export`, `import`. Persistent flags inherited by all: `--quiet`, `--ci`, `--only`, `--skip`.
- Free names for a self-update command: `self-update`, `upgrade`, `version` (redundant with flag). `update` must NOT be reused (collision with tool updates).

### Network / release reality
- ZERO in-process network code: `grep net/http` over all .go files → none. `self-update` would be the FIRST network code in the binary.
- Dependencies: cobra 1.8.0 + BurntSushi/toml 1.3.2 only (Go 1.22). stdlib `net/http`, `encoding/json`, `archive/tar`, `compress/gzip` suffice — no new deps needed.
- `make release` builds `dist/upp-{os}-{arch}.tar.gz|.zip` + `checksums.txt` (sha256sum). NO tag/publish target — releases are published manually. v0.1.0 IS published (5 archives + checksums.txt).
- `scripts/install.sh` is the verified reference recipe: GitHub API `releases/latest` → tag → `releases/download/{tag}/{asset}` + `checksums.txt` → sha256 verify (currently **warn-and-skip**) → backup + `mv`. In-process version must be stricter: **fail closed**.
- Asset-name trap: Makefile assets use GOOS names (`darwin`, `linux`, `windows`, `amd64`/`arm64`) but `platform.Detect()` (internal/platform/detect.go) returns canonical names `macos` (not `darwin`) and `x86_64` (not `amd64`). Self-update needs its own asset mapping — do NOT pass `platform.Detect()` output into asset names.
- Security model (internal/security/confirm.go): official tools auto-proceed; risk matrix with fail-closed zero-value trust; `--ci` errors on untrusted/medium-or-high. Self-update is "official" (the tool itself) but executes no shell command — checksum verification is the integrity gate.

## The "automatic" fork (the user's ambiguity — explored honestly)

"Actualización automática" has two readings; they are different products with different risk profiles:

### (a) User-invoked self-update — `upp self-update`
Detect → download → verify → replace, only when the user asks. Standard, safe, universal precedent.

### (b) AUTOMATIC updating — two sub-levels
- **(b1) Automatic DETECTION**: a "new upp available" hint surfaced while running other commands. Safe, common.
- **(b2) Automatic APPLY**: the binary replaces itself without an explicit self-update command. Footgun.

### Precedent (researched 2026-08-12)
| Tool | Explicit command | Automatic behavior |
|---|---|---|
| rustup | `rustup self update` | `auto-self-update` setting: `enable` (default) / `disable` / `check-only`. Auto-applies ONLY during `rustup update`/`toolchain install`, suppressible per-invocation via `--no-self-update`. Known complaints (rust-lang/rustup#2576): replaced expected version mid-command; users asked for check-only. |
| mise | `mise self-update` (GitHub API, `-y`/`-f`) | No silent auto-apply. Disables the command when installed via a package manager (path-based detection). |
| gh (GitHub CLI) | `gh upgrade` | Periodic cached check (~24h) prints "new release available" notice; NEVER auto-applies. |
| chezmoi | `chezmoi upgrade` | Explicit only. |

**Conclusion: no mainstream CLI silently replaces its own binary unprompted.** The one auto-apply precedent (rustup) is opt-in-configurable, fires only inside an explicit user update command, and still generated complaints. Detection-with-notice is the standard "automatic" behavior (gh, mise, rustup check-only). Automatic apply also breaks: CI reproducibility (binary changes mid-pipeline), running-process semantics (Windows cannot rename a running exe), and the audit posture this repo just adopted.

## Approaches

1. **A. Command-only self-update** — `upp self-update` (with `--check`, `--dry-run`); no automatic anything.
   - Pros: smallest surface; zero surprise; first network code fully user-triggered; matches rustup/mise/gh explicit-command pattern.
   - Cons: user must remember to run it; "automatic" part of the request unaddressed.
   - Effort: Medium

2. **B. Self-update + opt-in detection hint (RECOMMENDED)** — A, plus `settings.check_self_update` (default **OFF**) that adds one hint line at the end of `check`/bare `upp` output when a newer release exists, with a 24h TTL cache (gh-style) so it never hits the network per invocation.
   - Pros: covers BOTH readings of "actualización automática" safely; hint is read-only (exit code unchanged); opt-in respects the audit culture (new network behavior is never silently on); TTL cache bounds rate-limit exposure; offline/API failure degrades to "no hint".
   - Cons: one more config key; hint machinery must be carefully no-op in `--quiet`/`--ci`/offline; "automatically knowing" is not "automatically updating" — user wanting b2 must be told it's out of scope.
   - Effort: Medium

3. **C. B + auto-apply (rustup-style)** — config-gated auto-update when running `upp update` (never bare commands), plus check-only/disable modes.
   - Pros: true "actualización automática"; rustup precedent exists.
   - Cons: self-replacement mid-tool-update run can break the very command that triggered it; Windows impossible; surprises users (rustup#2576 lesson); contradicts fail-closed security posture; complexity High for marginal v1 value.
   - Effort: High — **rejected for v1**, revisit only on explicit user demand.

### Shared mechanics (all approaches)
- **Detection**: `GET https://api.github.com/repos/JhnFrankz/upp/releases/latest` (JSON, tag + assets; unauthenticated limit 60 req/h; matches install.sh). Degradation: API failure → "could not check" (exit 0), never break the main flow.
- **Comparison**: parse `vX.Y.Z[-N-gHASH][-dirty]`/`dev`; numeric 3-part compare of the tag prefix (stdlib, ~30 lines — no `golang.org/x/mod` dependency). Rules: `-dirty` or `dev` → "development build", NO update claim, exit 0; `-N-gHASH` (untagged build) → compare tag prefix only, a newer tag IS an update.
- **Download/verify**: `releases/download/{tag}/{asset}` + `checksums.txt` from the SAME release over HTTPS; checksum mismatch or missing entry → **abort (fail closed)** — deliberately stricter than install.sh's warn-and-skip.
- **Replace** (Linux/macOS): `os.Executable()` → `filepath.EvalSymlinks` (users symlink `~/.local/bin/upp`) → writability preflight (clear error, NEVER sudo from inside the tool) → temp file + chmod 0755 → backup current (`.backup.<ts>`, install.sh pattern) → `os.Rename` (atomic on POSIX) → restore on failure. Windows: explicit "not supported yet" error in v1 (rename-over-running-exe).
- **Security**: HTTPS only, dial/read timeouts (~10s), no redirect off https, stdlib honors HTTP(S)_PROXY; downloaded bytes are extracted with `archive/tar` — never executed as a shell command (keeps the risk matrix untouched).

### UX integration sketch
- `upp self-update` — Short: "Update the upp binary itself". Flags: `--check` (detect only), `--dry-run` (download+verify, no replace). Interactive prompt before replace; non-TTY → default deny with clear message.
- Hint line (opt-in): after the check summary — `⬆️ upp v0.1.1 available (current v0.1.0-19-gd40e428) — run "upp self-update"`. Omitted in `--quiet`; never errors in `--ci`; silently omitted offline or when the TTL cache hasn't expired.
- `--only`/`--skip` are tool filters — self-update ignores them (document in help).
- Output via existing `output.Renderer` + localized `Strings` (en/es) — new statuses, same style.

## Affected Areas

- `cmd/upp/main.go` — version var unchanged; optional richer `--version` template (out of core scope).
- `internal/cli/parser.go` — register new subcommand in `AddCommands`; new flags struct.
- `internal/cli/selfupdate.go` (NEW) — command wiring (check/download/verify/replace flow).
- `internal/selfupdate/` (NEW package) — version parse/compare, GitHub API client, asset mapping (darwin/macos + amd64/x86_64 translation), download+verify, atomic replace; THE containment boundary for first network code.
- `internal/cli/check.go` — opt-in hint hook after `CheckSummary` (approach B).
- `internal/config/config.go` + `defaults.go` — `Settings` gains `CheckSelfUpdate bool` (default false).
- `internal/output/render.go` + `language.go` — hint/status lines (en/es).
- `internal/platform/detect.go` — canonical OS/arch exist but asset names need separate mapping (see trap above); likely a small helper, not a change to Detect.
- `internal/security/` — no change to the matrix; spec note that self-update integrity is checksum-based (fail closed).
- `openspec/specs/` — delta specs: command-interface, config-system, security-model, ux-patterns.
- `scripts/install.sh`, `Makefile` — unchanged (reference only).

## Open Questions for Proposal

1. Command name: `self-update` (recommended; rustup/mise precedent) vs `upgrade` (gh/chezmoi) — user-facing surface, orchestrator should confirm.
2. Hint default: OFF (recommended — opt-in network behavior in an audited CLI) vs ON with kill-switch.
3. Auto-apply (b2): confirm definitively OUT of v1? (Recommended: yes.)
4. `--ci` behavior for `self-update`: auto-proceed after checksum verification (official-tool semantics, recommended) vs error.
5. Windows: defer entirely (recommended) — rename-over-running-exe semantics.
6. Release discipline: add a `make publish` target (tag + `gh release upload`) in this change or a follow-up? Feature fails closed when releases lack checksums.txt.
7. Non-TTY behavior: deny-with-message (recommended) vs require an explicit flag.

## Recommendation

**Approach B**: `upp self-update` command + opt-in (default OFF) detection hint in `check`/bare `upp` output. It answers BOTH readings of "actualización automática" at the safe level: the tool tells you automatically when a new version exists, and updating is one explicit, verified command. Auto-apply (b2) stays out of v1 — no mainstream CLI does silent self-replacement without complaints, and this repo just hardened its security posture. Release-asset + checksums.txt infrastructure already exists; stdlib covers all needs; install.sh proves the recipe; the previous cancelled change's verified facts (command map, version mechanics, asset layout) all hold.

## Risks

- First in-process network code in a just-audited, security-sensitive CLI → contained in one new package; HTTPS-only, timeouts, no off-https redirects, checksum fail-closed (stricter than install.sh), downloaded bytes never executed.
- False update claims on dev builds → `-dirty`/`dev` must report "development build" and never claim; compare only clean tag prefixes.
- Asset-name mapping trap (`macos` vs `darwin`, `x86_64` vs `amd64`) → dedicated mapping + table-driven tests; wrong name = 404, must fail closed with a clear message.
- Rate limits (60 req/h unauthenticated) → 24h TTL cache for the hint; manual `--check` unaffected; shared-IP environments degrade silently.
- Write permissions: `/usr/local/bin` (sudo) vs `~/.local/bin` — writability preflight with actionable error; never sudo from inside the tool.
- Supply chain: checksums.txt fetched from the same release over GitHub TLS; residual risk is GitHub compromise — acceptable and standard for this asset model.
- Release discipline: manual publish today; self-update silently unavailable (fail closed) if a release ships without checksums.txt — `make publish` target recommended as part of or right after this change.
- Windows self-replace: deferred; must error clearly, not silently skip.
- TOCTOU on replace → verify archive BEFORE extraction, backup before rename, restore on failure; POSIX rename is atomic.

## Ready for Proposal

Yes. Codebase facts and mechanics are verified; precedent research is done; the fork is mapped. The orchestrator should confirm with the user: (1) command name `self-update` vs `upgrade`, (2) hint default OFF vs ON, (3) auto-apply definitively out of v1 — then launch sdd-propose.
