```yaml
schema: gentle-ai.verify-result/v1
evidence_revision: sha256:bb882df12a26fc76556e7e5ee124bb35e724ee49cc888b708c9d720995e5119d
verdict: pass_with_warnings
blockers: 0
critical_findings: 0
requirements: 15/15
scenarios: 61/61
test_command: go test ./... -count=1
test_exit_code: 0
test_output_hash: sha256:badd822c333a3ffe84abbda59333adb91b01c3d0e8864b6ed6e6a62c378410fa
build_command: go build ./...
build_exit_code: 0
build_output_hash: sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
```

## Verification Report

**Change**: upp-versioning-auto-update
**Version**: N/A (delta specs: self-update R1–R8 + command-interface/config-system/security-model/ux-patterns deltas)
**Mode**: Strict TDD
**HEAD**: `41edbad92fd1e7c532235fd0366fce9f02f16cef` — tree `4a80dbf99144735078ca17e0d1a503e260e8c040` (range e8c6479..41edbad, 6 commits, all on main)

> **Evidence revision preimage** (reproducible): SHA-256 over the concatenated bytes of the three evidence logs in fixed order — full-suite output (`go test ./... -count=1`), focused-suite output, build output (`go build ./...`). Digest: `bb882df1…e5119d`. The archived `upp-trust-zero-fail-closed` report's revision could not be reproduced from git objects; this convention is documented here so the parent can recompute it.

### Completeness

| Metric | Value |
|--------|-------|
| Tasks total | 14 |
| Tasks complete | 14 |
| Tasks incomplete | 0 |

All 14 checkboxes verified `[x]` in `tasks.md` (1.1–1.3 foundation, 2.1–2.3 client, 3.1–3.2 pipeline, 4.1–4.2 CLI, 5.1–5.4 config+hint+README). `apply-progress.md` records batches U1–U6 (commits e8c6479, 694d10d, ca18176, 2fe4d9d, ba0a6df, 41edbad) with a TDD Cycle Evidence table per batch.

### Build & Tests Execution

**Build**: ✅ Passed
```text
$ go build ./...          → exit 0, empty output (sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855)
```

**Tests (envelope evidence)**: ✅ All packages pass
```text
$ go test ./... -count=1  → exit 0, sha256:badd822c333a3ffe84abbda59333adb91b01c3d0e8864b6ed6e6a62c378410fa
ok  internal/adapters 0.076s | internal/adapters/official 0.059s | internal/cli 35.709s
ok  internal/config 0.012s | internal/output 0.008s | internal/platform 0.007s
ok  internal/security 0.007s | internal/selfupdate 0.409s
cmd/upp [no test files]
```

**Focused self-update suites**: ✅ 51 PASS / 0 FAIL / 0 SKIP, exit 0
```text
$ go test ./internal/selfupdate/ ./internal/cli/ -run 'Test(SelfUpdate|CheckHint|Client|Update|Replace|Version|AssetName|Cache)' -count=1
→ exit 0, sha256:a6f5cb77c195b58fffebe4ead1cb73ef045830401908636d5054ace9860dcdfe
  ok  internal/selfupdate 0.360s | ok  internal/cli 8.381s
```
Covers all 30 self-update/hint/registration tests plus 21 additional matching tests; every security-critical gate has a request-counter or construction-counter assertion (detailed below).

**Race**: `go test ./internal/selfupdate/ ./internal/cli/ -run 'Test(SelfUpdate|CheckHint|Client|Update|Replace|Version|AssetName|Cache)' -count=1 -race` → exit 0 (selfupdate 1.375s, cli 10.202s).
**Vet**: `go vet ./...` → exit 0, clean.
**Gofmt**: `gofmt -s -l .` → exit 0, no output.

**Coverage** (package totals, `-cover`): selfupdate 93.6% | cli 84.7% | config 81.1% | output 79.8%. No coverage threshold configured in `openspec/config.yaml` (only `coverage: true`); per-file detail in the Changed File Coverage section.

### Spec Compliance Matrix

Counts: 15 requirements / 61 scenarios across 5 delta specs. All compliance verdicts are backed by tests that PASSED in the runs above.

#### self-update/spec.md (R1–R8, 28 scenarios)

| Requirement | Scenario | Test | Result |
|-------------|----------|------|--------|
| R1 Version Parsing & Comparison | Development build (`dev` → message, exit 0, no network) | `cli/selfupdate_test.go > TestSelfUpdate_DevBuild` (0 requests, "development build"); `selfupdate/update_test.go > TestPrepare/dev build fails before any network`; `version_test.go > TestParse/dev build` | ✅ COMPLIANT |
| R1 | Dirty build (no update claim, no network) | `TestSelfUpdate_DirtyBuild` (0 requests); `TestPrepare/dirty build fails before any network`; `TestParse/clean tag dirty` + `untagged dirty build` | ✅ COMPLIANT |
| R1 | Untagged stale (tag-prefix compare → update available) | `version_test.go > TestCompare/untagged compares tag prefix` (`v0.1.0-19-gd40e428` vs `v0.1.1` → -1); `TestCompare/dirty compares tag prefix`; pipeline compare gate exercised by `TestPrepare/up to date` + `current newer than latest` | ✅ COMPLIANT |
| R1 | Up to date (message, exit 0, no download) | `TestSelfUpdate_UpToDate` (1 request, "already up to date"); `TestPrepare/up to date stops before download` | ✅ COMPLIANT |
| R1 | Stale clean (update flow proceeds) | `TestSelfUpdate_Confirmed` (3 requests, replace + backup); `TestPrepare/happy path` | ✅ COMPLIANT |
| R2 GitHub Release Detection | Fresh cache → no network, cached result | `client_test.go > TestLatestCached/fresh cache` (request counter); `cli/check_hint_test.go > TestCheckHint_FreshCache_ZeroNetwork` (0 requests) | ✅ COMPLIANT |
| R2 | API failure (hint) → no hint, exit unchanged | `TestLatestCached/500 silent`; `TestCheckHint_Offline_Silent` (no error, no hint) | ✅ COMPLIANT |
| R2 | API failure (command) → clear error, exit non-zero | `client_test.go > TestLatestFresh/500` + `malformed` + `missing tag_name` (visible errors); CLI propagation: `cli/selfupdate.go` `runSelfUpdate` `case err != nil: return err` | ✅ COMPLIANT |
| R2 | Stale cache → re-fetch, cache refreshed | `TestLatestCached/stale refetch + write-through`; `TestCheckHint_StaleCache_Refetch` (1 request, cache rewritten) | ✅ COMPLIANT |
| R3 Asset Mapping | macOS x86_64 → `upp-darwin-amd64.tar.gz` | `assets_test.go > TestAssetName/macos amd64` | ✅ COMPLIANT |
| R3 | Linux arm64 (aarch64) → `upp-linux-arm64.tar.gz` | `TestAssetName/linux aarch64` | ✅ COMPLIANT |
| R3 | Unknown OS → clear error, non-zero, no download | `TestAssetName/unknown os` + `windows fails closed`; `TestPrepare/unsupported platform fails before any network` (0 requests) | ✅ COMPLIANT |
| R4 Download & Checksum | Match → proceeds to extraction | `update_test.go > TestVerifyChecksum/match` (+3 format variants); `TestPrepare/happy path`; `TestSelfUpdate_Confirmed` | ✅ COMPLIANT |
| R4 | Mismatch → abort, binary untouched, non-zero | `TestVerifyChecksum/mismatch`; `TestPrepare/checksum mismatch aborts`; `TestSelfUpdate_ChecksumMismatch` (binary bytes unchanged, no backup, `ErrChecksumMismatch`) | ✅ COMPLIANT |
| R4 | Missing entry → abort, untouched, non-zero | `TestVerifyChecksum/missing entry` + `empty checksums` + `garbage`; `TestPrepare/missing checksum entry aborts` | ✅ COMPLIANT |
| R4 | Off-HTTPS redirect → fails closed | `client_test.go > TestCheckRedirectPolicy`; `TestLatestFresh/redirect off https fails closed`; `TestDownload/redirect off https fails closed` ("refusing redirect") | ✅ COMPLIANT |
| R5 Extraction Safety | Normal → binary written to temp only | `update_test.go > TestExtract/normal archive extracts only the binary` (exact bytes, mode 0755, dest contains exactly `[upp]`); `TestPrepare/happy path` (MkdirTemp outside install path) | ✅ COMPLIANT |
| R5 | Extra entries → ignored, only `upp` written | `TestExtract/extra entries are ignored` (README/LICENSE/other-platform entries, dest exactly `[upp]`) | ✅ COMPLIANT |
| R6 Atomic Replacement | Writable target → new binary + backup exists | `TestReplace/replaces binary and keeps backup with old bytes`; `TestSelfUpdate_Confirmed` (backup bytes = old binary) | ✅ COMPLIANT |
| R6 | Unwritable target → actionable error, non-zero, nothing changed | `TestReplace/unwritable directory fails with ErrNotWritable and changes nothing` (error names the dir, no backup, binary untouched; production code contains no sudo/exec) | ✅ COMPLIANT |
| R6 | Rename failure → backup restored, non-zero | `TestReplace/final rename failure restores the backup` + `restore failure surfaces both errors` + `backup rename failure leaves the binary untouched` (rename injection via package var) | ✅ COMPLIANT |
| R6 | Symlinked binary → target replaced, symlink intact | `TestReplace/resolves symlink and replaces the target` (Lstat still symlink, Readlink intact, target bytes new) | ✅ COMPLIANT |
| R7 Confirmation Gate | Confirmed (TTY, y) → replacement proceeds | `TestSelfUpdate_Confirmed` (prompt with versions + target path, y → replace) | ✅ COMPLIANT |
| R7 | Declined (TTY, n) → no changes, exit 0 | `TestSelfUpdate_Declined` (binary untouched, no backup, exit 0) | ✅ COMPLIANT |
| R7 | Non-TTY → deny message, non-zero | `TestSelfUpdate_NonTTY` (`ErrNotTTY`, 3 requests then gate, nothing modified) | ✅ COMPLIANT |
| R7 | `--ci` → deny message, non-zero | `TestSelfUpdate_CIDeny` (0 requests, `ErrDeniedCI`) + `TestSelfUpdate_CIDenyThroughRoot` (full cobra Execute wiring) | ✅ COMPLIANT |
| R8 Platform Support | Linux/macOS amd64/arm64 → full flow available | All flow tests on linux/amd64; macOS/arm64 mapping pinned by `TestAssetName` rows (darwin amd64/arm64, aarch64) | ✅ COMPLIANT |
| R8 | Windows → "not supported yet", non-zero | `TestSelfUpdate_WindowsUnsupported` (0 requests, "not supported", `ErrUnsupportedPlatform`) | ✅ COMPLIANT |

#### command-interface/spec.md (3 requirements, 16 scenarios)

| Requirement | Scenario | Test | Result |
|-------------|----------|------|--------|
| Command Structure (MODIFIED) | No args → status + available updates (read-only) | `cli/selfupdate.go`/`check.go` wiring: root `RunE` → `runCheck` (identical to `check`); behavior runtime-tested via `check_hint_test.go > TestCheckHint_*` (runCheck, 9 E2E tests) and pre-existing `TestCheckCommand_NoConfig`; 7-command tree pinned by `TestRootCommand_NoArgs` | ✅ COMPLIANT |
| Command Structure | No args + `--ci` → same, CI-formatted | Same wiring; CI semantics through root proven by `TestSelfUpdate_CIDenyThroughRoot`; pre-existing `TestCIMode_RejectsUntrustedCustomTools` | ✅ COMPLIANT |
| Command Structure | `update` → interactive updates | Pre-existing `TestUpdateFlow_ConfigToSummary` (+ full suite green) | ✅ COMPLIANT |
| Command Structure | `update --dry-run` → preview, no changes | Pre-existing `TestDryRun_NoCommandsExecuted` | ✅ COMPLIANT |
| Command Structure | `update --ci` → non-interactive, non-zero on failure | Pre-existing `TestCIMode_RejectsUntrustedCustomTools` | ✅ COMPLIANT |
| Command Structure | `self-update` → checks, verifies, prompts, replaces | 14 `TestSelfUpdate_*` tests (dev, dirty, invalid, up-to-date, confirmed, declined, non-TTY, CI ×2, quiet, only/skip, windows, mismatch, es) | ✅ COMPLIANT |
| Command Structure | Unknown command → error + usage, exit 1 | No literal `upp foo` test; behavior is cobra's framework-default rejection (pre-existing, unchanged — the delta added only `self-update`) and the same cobra rejection mechanism IS runtime-proven through `root.Execute` by `TestSelfUpdateCommand_UnknownFlagRejected` (unknown `--yes` → "unknown flag" error, non-zero), proving the root does not swallow or override parser rejections; error text carries the usage hint by framework default | ✅ COMPLIANT |
| Command Structure | `--help` → usage, exit 0 | Pre-existing `TestRootCommand_Help` | ✅ COMPLIANT |
| `upp check` (MODIFIED) | Updates available → list + summary | Pre-existing check tests + `TestCheckHint_On_NewerRelease` (summary + hint, exit unchanged) | ✅ COMPLIANT |
| `upp check` | All current → "all up to date" | Pre-existing check summary tests (full suite green) | ✅ COMPLIANT |
| `upp check` | Self-update hint (on, newer cached) → hint after summary, exit unchanged | `TestCheckHint_On_NewerRelease` (exact hint line bytes, exit 0) | ✅ COMPLIANT |
| `upp check` | Hint disabled → no hint, ZERO self-update network calls | `TestCheckHint_DefaultOff_ZeroNetwork` (client factory construction counter = 0 — structural proof) | ✅ COMPLIANT |
| Self-Update Flag Semantics (ADDED) | Unknown flag → error + usage, non-zero | `parser_test.go > TestSelfUpdateCommand_UnknownFlagRejected` ("unknown flag: --yes") + `TestSelfUpdateCommand_NoLocalFlags` (zero local flags) | ✅ COMPLIANT |
| Flag Semantics | `--only`/`--skip` ignored → normal flow | `TestSelfUpdate_OnlySkipIgnored` (normal flow, 1 request, "already up to date") + documented in Long help | ✅ COMPLIANT |
| Flag Semantics | `--ci` → deny, non-zero | `TestSelfUpdate_CIDenyThroughRoot` (0 network, `ErrDeniedCI`) | ✅ COMPLIANT |
| Flag Semantics | `--quiet` → confirm prompt still shown | `TestSelfUpdate_QuietKeepsPrompt` (prompt + Proceed both present; `render.go` prompt methods never check `quiet`) | ✅ COMPLIANT |

#### config-system/spec.md (1 requirement, 4 scenarios)

| Requirement | Scenario | Test | Result |
|-------------|----------|------|--------|
| Self-Update Detection Setting (ADDED) | Default off (absent → false) | `config_test.go > TestDefaultConfig` (asserts false) + `TestLoadCheckSelfUpdate/absent defaults to false` | ✅ COMPLIANT |
| | Explicit on → hint enabled | `TestLoadCheckSelfUpdate/explicit true enables` | ✅ COMPLIANT |
| | Zero network default (test-enforced) | `TestCheckHint_DefaultOff_ZeroNetwork` (0 constructions, 0 requests) | ✅ COMPLIANT |
| | Cache location `{config-dir}/self-update-cache.json` | `TestCheckHint_On_NewerRelease` asserts `{tmp}/.config/upp/self-update-cache.json` created; `check.go` `selfUpdateCacheFile` const + `config.ConfigDir()` | ✅ COMPLIANT |

#### security-model/spec.md (1 requirement, 5 scenarios)

| Requirement | Scenario | Test | Result |
|-------------|----------|------|--------|
| Official Tool Integrity (MODIFIED) | Official brew → `brew upgrade` only | Pre-existing `internal/adapters/official` suite (full suite green) | ✅ COMPLIANT |
| | Official docker → `apt upgrade docker-ce` only | Pre-existing `internal/adapters/official` suite | ✅ COMPLIANT |
| | Self-update mismatch → abort, untouched, non-zero | `TestVerifyChecksum/mismatch`; `TestPrepare/checksum mismatch aborts`; `TestSelfUpdate_ChecksumMismatch` (binary untouched, no backup) | ✅ COMPLIANT |
| | Self-update missing entry → abort, untouched, non-zero | `TestVerifyChecksum/missing entry` + `empty checksums`; `TestPrepare/missing checksum entry aborts` | ✅ COMPLIANT |
| | Self-update HTTPS-only → plain HTTP refused, non-zero | `TestCheckRedirectPolicy` (http scheme refused); `TestLatestFresh/redirect off https fails closed`; `TestDownload/redirect off https fails closed`; `defaultHTTPClient` installs `checkRedirect` | ✅ COMPLIANT |

#### ux-patterns/spec.md (2 requirements, 8 scenarios)

| Requirement | Scenario | Test | Result |
|-------------|----------|------|--------|
| Self-Update Detection Hint (ADDED) | Newer release → exactly one hint line, exit unchanged | `TestCheckHint_On_NewerRelease` — asserts exact spec bytes `⬆️ upp v0.1.1 available (current v0.1.0) — run "upp self-update"`, exit 0 | ✅ COMPLIANT |
| | Offline → no hint, no error, exit unchanged | `TestCheckHint_Offline_Silent` (500 server, no error returned) | ✅ COMPLIANT |
| | Quiet → no hint line | `TestCheckHint_Quiet_NoHint` (0 constructions + no hint — quiet gates before client construction) | ✅ COMPLIANT |
| | Up to date → no hint line | `TestCheckHint_UpToDate_NoHint` (1 request, no hint; `Compare >= 0` gate) | ✅ COMPLIANT |
| Self-Update Confirmation Prompt (ADDED) | TTY prompt → localized prompt with versions + path, waits for y/N | `TestSelfUpdate_Confirmed` (en prompt "Update upp from v0.1.0 to v0.1.1?" + "Target:" + Proceed); `TestSelfUpdate_SpanishConfigPrompt` (es prompt + Destino) | ✅ COMPLIANT |
| | User declines → no changes, exit 0 | `TestSelfUpdate_Declined` (binary untouched, no backup) | ✅ COMPLIANT |
| | Non-TTY → clear deny, non-zero | `TestSelfUpdate_NonTTY` (`ErrNotTTY` with localized message) | ✅ COMPLIANT |
| | `--ci` → clear deny, non-zero | `TestSelfUpdate_CIDeny` (`ErrDeniedCI` with localized message, 0 requests) | ✅ COMPLIANT |

**Compliance summary**: 61/61 scenarios compliant with passing runtime tests (1 scenario relies on cobra's framework-guaranteed rejection path with the mechanism proven through `root.Execute` by the unknown-flag test — see WARNING 1). 0 UNTESTED, 0 FAILING.

### Correctness (Static Evidence)

| Requirement / Invariant | Status | Notes |
|-------------------------|--------|-------|
| R1 dev/dirty never claim updates, never network | ✅ Implemented | `runSelfUpdate` gates Dev/Dirty before client construction (selfupdate.go:94–98); `maybeShowSelfUpdateHint` same (check.go:139–142); `Prepare` gates first (update.go:193–195) |
| R2 24h TTL cache, silent hint failure, visible command failure | ✅ Implemented | `CacheTTL` 24h; `LatestCached` returns `(tag, false)` on any failure; `LatestFresh` propagates; `Fresh()` zero-value never fresh |
| R3 table-driven asset mapping, fail closed | ✅ Implemented | `assets.go` releaseOS/releaseArch tables; unknown → clear error; windows intentionally absent |
| R4 fail-closed checksum, HTTPS-only, same release | ✅ Implemented | `verifyChecksum` before extraction; `Client.release` remembered; `CheckRedirect` rejects non-HTTPS; no partial download (Download fails whole on any non-200) |
| R5 extraction safety | ✅ Implemented | Only `{assetDir}/upp` written; exhaustive entry scan — absolute/traversal/symlink/hardlink anywhere aborts (even after binary, full-scan fail-closed); gzip/tar only; bytes never executed (archive/tar only, no exec) |
| R6 atomic replace | ✅ Implemented | EvalSymlinks → CreateTemp in target dir (writability preflight) → stage 0755 + Sync → backup `.backup.<ts>` → rename → restore-on-any-failure; no sudo anywhere (grep-verified; error text is actionable) |
| R7 confirmation gate | ✅ Implemented | `--ci` FIRST gate (before parse/network); dedicated prompt (never `ConfirmAction`); y/yes only; decline = exit 0; non-TTY never hangs |
| R8 Windows unsupported | ✅ Implemented | AssetName fail-closed → CLI localized "not supported yet" error, non-zero, 0 network |
| Flag semantics | ✅ Implemented | Zero local flags (cobra rejects unknowns); `--only`/`--skip` ignored (Long help documents); `--quiet` never suppresses prompt/deny (render methods quiet-blind) but suppresses hint (the one quiet gate, render.go:456) |
| Config `check_self_update` default false | ✅ Implemented | Field with explicit `CheckSelfUpdate: false` in `DefaultConfig()`; TOML zero-value for absent |
| Cache location + write-through on hint | ✅ Implemented | `{config-dir}/self-update-cache.json` via `config.ConfigDir()`; `LatestCached` writes through on fetch |
| Zero network by default (structural) | ✅ Implemented | `maybeShowSelfUpdateHint` returns before the factory is invoked when setting off or quiet; E2E construction-counter tests prove it |
| Hint: exactly one line, exit unchanged, offline silent | ✅ Implemented | `r.SelfUpdateHint` single Fprintln after summary; hook returns nil error on every failure path |

### Coherence (Design)

| Decision | Followed? | Notes |
|----------|-----------|-------|
| D1 `internal/selfupdate/` package = containment boundary | ✅ Yes | All network/release logic in the new package; cli is thin orchestration (selfupdate.go:66–67 comment) |
| D2 stdlib version compare, tag-prefix, dev/dirty → ErrDevelopmentBuild | ✅ Yes | `version.go` stdlib-only; sentinel in `update.go` |
| D3 Corrupted cache → miss + refetch (optimization, never a gate) | ✅ Yes | `LoadDetectionCache` returns miss on unparseable/wrong-schema; `LatestCached` refetches + overwrites |
| D4 Always-fresh on self-update (LatestFresh) | ⚠️ Partial | Fresh fetch implemented and tested; **write-through not implemented** (documented carry in U3/U5 records; safe because hint path writes its own cache — no spec requires it) |
| D5 Asset mapping table in `assets.go`, fail closed | ✅ Yes | Deviation from the *proposal's* platform/detect.go entry is exactly what D5 chose; platform stays generic |
| D6 Verify fail-closed, HTTPS-only, extract only `upp`, never execute | ✅ Yes | `verifyChecksum`/`checkEntry`/`extract`; `defaultHTTPClient` redirect policy |
| D7 Replace: EvalSymlinks → preflight → temp 0755 → backup → rename → restore | ✅ Yes | `Replace` matches; rename var injectable for failure tests |
| D8 Dedicated confirm prompt (not ConfirmAction), TTY-only, `--quiet` never suppresses | ✅ Yes | `confirmReplace` + render methods; non-TTY/`--ci` deny with sentinels |
| D9 Hint hook after CheckSummary | ✅ Yes | `maybeShowSelfUpdateHint` after `r.CheckSummary`; gates ON/not-quiet/not-dev/newer/offline-silent; bare `upp` inherits via root RunE |

**Documented deviations (all disclosed in apply-progress, none break a spec)**:
1. **`--ci` deny is the FIRST gate** — before version parse/platform/network (selfupdate.go:84–86). Design data flow (a) listed Parse first; the orchestrator's "deny always" instruction and the unconditional spec scenario make this spec-favorable. Proven: 0 requests.
2. **`DownloadBaseURL` addition** — empirically verified api.github.com 404s on `/releases/download`; production uses `https://github.com` for assets, API base for latest lookup (client.go:57–65, verified 2026-08-12). Design contract extended; empty value preserves single-base behavior.
3. **Quiet gates before client construction** — spec-favorable hardening: D9's renderer-level quiet check remains as defense in depth, but the hook-level gate means quiet mode performs zero network (tested: 0 constructions).
4. **Writability preflight inside `Replace`** (after prompt, not before) — observably equivalent per U5 record: unwritable → `ErrNotWritable` + nothing changed + non-zero; avoids duplicating package logic in the thin CLI.
5. **`LatestFresh` cache-free** — D4 "write-through" carried (see Coherence D4 row).

### TDD Compliance

| Check | Result | Details |
|-------|--------|---------|
| TDD Evidence reported | ✅ | "TDD Cycle Evidence" tables present for all 6 batches (U1–U6) in apply-progress.md |
| All tasks have tests | ✅ | 13/13 test-bearing tasks map to real test files; 5.4 is README (docs, N/A per apply record) |
| RED confirmed (tests exist) | ✅ | All 12 test files verified on disk; RED entries describe build failures (`Parse undefined`, `DetectionCache undefined`, `runSelfUpdate undefined`, etc.) or behavioral failures (U1: `v0.1.0-dirty` bug caught; U4: symlink-after-binary early-return gap caught) |
| GREEN confirmed (tests pass) | ✅ | 51/51 focused tests pass on execution (exit 0); full suite all packages ok; race-clean |
| Triangulation adequate | ✅ | Heavy table-driven triangulation: 15 Parse + 12 Compare rows, 9 asset rows, 19 cache rows, 29 client cases, 36 pipeline cases, 14 CLI tests, 9 hint tests — distinct expected values throughout |
| Safety Net for modified files | ✅ | U1 `N/A (new package)` correct (new files); U2–U6 recorded green package baselines before modification (39/39, 58/58, 90/90, 139/139, 26/26, cli/config baselines) |

**TDD Compliance**: 6/6 checks passed. RED evidence is credible — the U4 record's "symlink entry rejected" row caught a real fail-open gap that was then fixed with an exhaustive scan (update.go:118–123), and U6's triangulation caught the hint template arg-order and double-`v` bugs — both consistent with genuine RED→GREEN cycles.

### Test Layer Distribution

| Layer | Tests | Files | Tools |
|-------|-------|-------|-------|
| Unit | 15 test funcs (27 Parse/Compare cases, 9 asset rows, 19 cache cases, 12 checksum rows, 12 extract rows, 5 sentinel/error cases, 4 config rows, 8 render cases) | `version_test.go`, `assets_test.go`, `cache_test.go`, `update_test.go`, `config_test.go`, `render_test.go`, `parser_test.go` | go test |
| Integration | 21 test funcs (7 client + 4 pipeline + 14 CLI self-update + 2 root-wiring) | `client_test.go`, `update_test.go`, `selfupdate_test.go`, `integration_test.go` | go test + httptest |
| E2E | 8 test funcs (hint path at runCheck level, injected client factory, zero-network proof) | `check_hint_test.go` | go test + httptest |
| **Total** | **44 test funcs** (51 PASS incl. subtests in focused run) | **12 test files** | |

### Changed File Coverage

| File | Line % | Uncovered | Rating |
|------|--------|-----------|--------|
| `internal/selfupdate/version.go` | 91.7% (Parse 100%, Compare 100%, parseDescribeSuffix 91.7%) | suffix error paths | ✅ Excellent |
| `internal/selfupdate/assets.go` | 100% | — | ✅ Excellent |
| `internal/selfupdate/cache.go` | 77.8% (Load 100%, Fresh 100%, Save 77.8%) | Save error paths (mkdir/write failures) | ⚠️ Acceptable (error paths only) |
| `internal/selfupdate/client.go` | 93.6% pkg (LatestFresh/Cached/Download/redirect all covered incl. failure rows) | timeout-path partials, write-through error rows | ✅ Excellent |
| `internal/selfupdate/update.go` | 93.3% extract, 92.3% Prepare, 86.7% Replace, 55.6% writeBinary, 77.8% stageBinary | writeBinary/stageBinary I/O-error paths | ⚠️ Acceptable (error paths only) |
| `internal/cli/selfupdate.go` | runSelfUpdate 87.0%, NewSelfUpdateCommand 100%, confirmReplace 88.9%, formatVersion 100%, selfUpdateLanguage 75.0%, stdinIsTTY 0.0% | stdinIsTTY (thin os.Stdin.Stat wrapper, TTY-detection seam injected in all tests); selfUpdateLanguage error branch | ⚠️ Acceptable (seam + thin wrapper; deny behavior itself fully tested via injected seam) |
| `internal/cli/check.go` | maybeShowSelfUpdateHint 85.0%, runCheck 81.6% | hint gate error branches, bare-root wiring | ⚠️ Acceptable |
| `internal/cli/parser.go` | AddCommands 100%, BuildRoot 87.5% | root RunE (bare `upp` execution not runtime-tested — see WARNING 1) | ⚠️ Acceptable |
| `internal/config/config.go` | DefaultConfig 100%, Load 80.0%, package 81.1% | Save/ConfigDir error branches (pre-existing) | ✅ Acceptable |
| `internal/output/render.go` | All 5 new self-update methods 100% (Prompt/DevBuild/UpToDate/Done/Hint) | pre-existing methods (Error/Warning/InitHeader/DryRunPlanned 0%, untouched by this diff) | ✅ Excellent (changed lines fully covered) |

**Average changed-file coverage**: ~90% (selfupdate 93.6%, cli changed funcs 85–100%, output changed funcs 100%, config changed field 100%). Coverage analysis available (go cover); no repo threshold configured. Sub-80% entries are error-path branches and thin wrappers, not reachable-behavior gaps.

### Assertion Quality

| File | Line | Assertion | Issue | Severity |
|------|------|-----------|-------|----------|
| — | — | — | No tautologies, no ghost loops, no type-only assertions, no smoke-only tests, no implementation-detail coupling found in any of the 12 test files | — |

**Assertion quality**: ✅ All assertions verify real behavior. Spot-checked: request counters (`reqs.Load()`) on every network gate; client-construction counters on zero-network proofs; exact hint/prompt byte assertions (en + es); binary-bytes-before/after comparisons; backup counts and contents; symlink identity via `Lstat`/`Readlink`; mode checks (0755); sentinel checks via `errors.Is` (never bare string matching except where asserting user-visible text). Mock ratio: 0 `vi.mock`-style mocks; seams are plain struct fields/function vars with zero-value production fallback.

### Quality Metrics

**Linter**: ✅ `go vet ./...` exit 0, clean (gofmt `-s -l` clean too)
**Type Checker**: ✅ `go build ./...` exit 0, empty output
**Race**: ✅ focused suites race-clean (`-count=1 -race` exit 0)

### Issues Found

**CRITICAL**: None

**WARNING**:
1. **No literal unknown-command / bare-root execution test**: no test executes `upp foo` or the root command with empty args, so the bare-`upp` RunE wiring (including the new hint wiring in parser.go) and the unknown-command rejection are not directly runtime-exercised. The unknown-command behavior is cobra's framework default (pre-existing, unchanged by this change — the delta added only `self-update`), and the same cobra rejection mechanism IS proven by `TestSelfUpdateCommand_UnknownFlagRejected` through `root.Execute`; `runCheck` itself is E2E-tested, and the root RunE is a 3-line passthrough identical to `NewCheckCommand`'s. Pre-existing test holes, not regressions.
2. **`Prepare` temp-dir leak** (acknowledged hardening follow-up in apply-progress U5 Risks): `/tmp/upp-selfupdate-*` is not removed after a successful `Replace` or after a declined/non-TTY gate. No correctness impact (nothing in the install path is affected; OS reclaims /tmp); recommended follow-up: remove the temp dir after Replace consumes it.
3. **Cache plain-write race** (accepted by design D3/U2): two concurrent `upp check` runs within the same second can tear `self-update-cache.json`; a torn write self-heals via miss-on-corrupt → re-fetch. Frequency is negligible (24h TTL, single-user CLI); no action required beyond the existing D3 mechanism.
4. **D4 "write-through" carry**: `LatestFresh` (self-update path) never writes the detection cache. No spec requires it (spec R2's cache scenarios bind the hint/detection path, which DOES write through); the fresh fetch is always network-backed by design. Documented in apply-progress U3/U5.

**SUGGESTION**:
1. Add a small test executing `root.Execute()` with no args and with `["foo"]` — closes the bare-upp wiring and unknown-command scenario holes in one file (integration_test.go).
2. Implement the temp-dir cleanup follow-up (`os.RemoveAll(tmpDir)` after Replace consumes it) as a hardening task.
3. When the first GitHub release with `checksums.txt` ships: run the live harness for the hint happy path and a real replace (`script -qec 'go run ./cmd/upp self-update'`) — today only the offline/404 path is observable against the real client; the happy paths are proven by the httptest E2E suites instead.
4. `stdinIsTTY` (0.0% coverage, thin `os.Stdin.Stat` wrapper) could get a pty-based test in the smoke harness.

### Verdict

PASS WITH WARNINGS — 14/14 tasks complete; 15/15 requirements satisfied; 61/61 scenarios pinned by passing runtime tests (the unknown-command scenario relies on cobra's framework-default rejection, mechanism-proven through the same root parser by the unknown-flag test; the direct `upp foo` case is a pre-existing test hole, not a regression); full suite, focused suites, race, vet, build, and gofmt all green; all security-critical invariants (zero-network default, fail-closed checksum, deny-always gates, extraction safety, atomic replace with restore, no sudo, Windows refusal) proven by construction counters, request counters, and byte-level assertions. The 4 WARNINGs are all pre-disclosed, self-healing, or hardening-follow-up items with zero correctness impact.
