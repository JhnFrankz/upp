# Proposal: Self-Update & Opt-in Update Detection (upp-versioning-auto-update)

## Intent

upp cannot update its own binary; users must re-run install.sh or miss releases. Add an explicit, verified, atomic `upp self-update` command plus an opt-in (default OFF) detection hint — covering both readings of "automatic updating" without silent self-replacement.

## Scope

### In Scope
- `upp self-update` (Linux/macOS, amd64/arm64): check → download → sha256 verify → confirm → atomic replace.
- Opt-in `settings.check_self_update` (default **OFF**): hint line at end of `check`/bare `upp` output.
- New `internal/selfupdate/` package — single containment boundary for ALL in-process network code; HTTPS-only, timeouts, stdlib-only (http/json/tar/gzip, no new deps).

### Out of Scope
- Auto-apply without confirmation (user-locked: OUT of v1; rustup precedent shows complaints).
- Windows self-replace (clear "not supported yet" error only), sudo escalation, background daemon.
- `make publish` automation; `--yes`, `--check`, `--dry-run` flags (v1: one command, one prompt — user-locked).

## Capabilities

### New Capabilities
- `self-update`: command + version compare, GitHub release client, asset mapping, checksum verify, atomic replace (new `openspec/specs/self-update/spec.md`).

### Modified Capabilities (delta specs)
- `command-interface`: command table + `upp check` hint hook.
- `config-system`: new `check_self_update` setting, default false.
- `security-model`: Official Tool Integrity — self-update fails closed on checksum mismatch.
- `ux-patterns`: hint line, confirm prompt, en/es strings.

## Approach

- **Version compare**: parse `vX.Y.Z[-N-gHASH][-dirty]`/`dev`; numeric 3-part compare. `dev`/`-dirty` → "development build", never claim update; untagged `-N-gHASH` compares tag prefix only.
- **Detection**: GitHub API `releases/latest`; ~10s dial/read timeouts; 24h TTL cache (bounds 60 req/h limit); API failure → silent no-hint, exit 0.
- **Asset mapping** (the trap): map `platform.Detect()` canonical names (`macos`/`x86_64`) → release assets (`darwin`/`amd64`); dedicated table + table-driven tests; unknown mapping → fail closed with clear message.
- **Verify**: archive + `checksums.txt` from the SAME release over HTTPS; mismatch/missing → abort (deliberately stricter than install.sh warn-and-skip). Bytes extracted with tar/gzip, never executed.
- **Replace**: os.Executable → EvalSymlinks → writability preflight (actionable error, NEVER sudo) → temp file + 0755 → backup `.backup.<ts>` → os.Rename → restore on failure.
- **Confirm** (user-locked): TTY → always prompt before replace; non-TTY/`--ci` → DENY with clear message (ratified deviation from explorer's auto-proceed lean: confirmation lock + CI reproducibility).
- **UX**: Short: "Update the upp binary itself". Opt-in hint: `⬆️ upp v0.1.1 available (current v0.1.0-19-gd40e428) — run "upp self-update"`; omitted in `--quiet`; `--only`/`--skip` ignored (documented in help).

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/selfupdate/` | New | Version compare, client, asset map, verify, replace |
| `internal/cli/parser.go` | Modified | Register `self-update` (no flags in v1) |
| `internal/cli/selfupdate.go` | New | Command flow wiring |
| `internal/cli/check.go` | Modified | Opt-in hint hook |
| `internal/config/config.go`, `defaults.go` | Modified | `CheckSelfUpdate` (default false) |
| `internal/output/render.go`, `language.go` | Modified | Hint/status strings en/es |
| `internal/platform/detect.go` | Modified | Asset-name mapping helper |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Rate limit (60 req/h) | Med | 24h TTL cache; silent degradation |
| Asset-name trap → 404 | Med | Mapping table + tests; fail closed |
| TOCTOU on replace | Low | Verify before extract; backup + restore |
| Unwritable install dir | Med | Preflight, actionable error, no sudo |
| First network code | Med | One package, HTTPS-only, fail-closed checksum |

## Rollback Plan

Backup kept at `{binary}.backup.<ts>`; any failure → restore backup, non-zero exit. Manual recovery: re-run `install.sh` or rename the backup over the binary.

## Dependencies

- Releases MUST ship `checksums.txt` (manual today; feature fails closed otherwise — `make publish` target is a follow-up).

## Open Questions for Spec

- Exact hint placement in check summary; cache file location (config dir?).

## Success Criteria

- [ ] `go test ./... -count=1` green, incl. asset-mapping + version-compare table tests.
- [ ] Up-to-date binary: `self-update` → exit 0, "already up to date", no download.
- [ ] Stale binary + published release: prompts, replaces atomically, backup exists.
- [ ] Checksum mismatch: abort, no replacement, non-zero exit.
- [ ] Hint OFF: bare `upp` makes zero network calls; ON: one hint line, exit code unchanged; offline: silent omission.
- [ ] Non-TTY: deny with clear message, no silent skip or proceed.
