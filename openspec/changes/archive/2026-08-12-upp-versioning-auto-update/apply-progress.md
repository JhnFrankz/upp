# Apply Progress — upp-versioning-auto-update (Batch 1: U1)

**Date**: 2026-08-12
**Batch**: U1 only (tasks 1.1–1.2) — chained/stacked PR slice, do NOT implement beyond U1
**Delivery**: `auto-chain`, `stacked-to-main` — no branches/PRs created in this batch (created at delivery time)
**Mode**: Strict TDD (openspec/config.yaml `strict_tdd: true`; runner `go test ./... -count=1`)
**Commit**: `e8c6479 feat(selfupdate): version compare + asset mapping` (4 files, +294)

## Completed Tasks

- [x] 1.1 [U1] `internal/selfupdate/version.go`: `Version{Tag;Dirty;Dev}`, `Parse`, `Compare` (stdlib tag-prefix); tests: dev/dirty/untagged/clean.
- [x] 1.2 [U1] `internal/selfupdate/assets.go`: mapping table (macos→darwin, x86_64→amd64, aarch64→arm64, identity) → `AssetName`; tests: known + unknown fail-closed.

## TDD Cycle Evidence

| Task | Test File | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|------|-----------|-------|------------|-----|-------|-------------|----------|
| 1.1 | `internal/selfupdate/version_test.go` | Unit | N/A (new package) | ✅ Written + executed: build failed, `Parse`/`Version` undefined | ✅ Passed after minimal impl | ✅ 15 Parse rows + 12 Compare rows; caught `v0.1.0-dirty` suffix bug | ✅ gofmt -s / vet clean, tests green |
| 1.2 | `internal/selfupdate/assets_test.go` | Unit | N/A (new package) | ✅ Written + executed: build failed, `AssetName` undefined | ✅ Passed after minimal impl | ✅ 9 rows: full PLATFORMS matrix + identity + 3 fail-closed | ✅ gofmt -s / vet clean, tests green |

## Work Unit Evidence (U1)

| Evidence | Required value |
|---|---|
| Focused test command and exact result | `go test ./internal/selfupdate/ -count=1 -v` → ok, 39 PASS (3 top-level: TestParse, TestCompare, TestAssetName + 36 subtests), 0 FAIL, exit 0 |
| Runtime harness command/scenario and exact result | N/A — pure functions (`Parse`/`Compare`/`AssetName`), no I/O or runtime boundary; behavior fully proven by unit tests (per tasks.md U1 harness column) |
| Rollback boundary | Delete `internal/selfupdate/version.go`, `assets.go` + their `_test.go`; revert commit `e8c6479`; nothing else in the repo references the new package (no cli/config/output touched) |

## Files Changed

| File | Action | What Was Done |
|------|--------|---------------|
| `internal/selfupdate/version.go` | Created | `Version{Tag [3]int; Dirty, Dev bool}`, `Parse` (git-describe grammar: `vX.Y.Z`, `-N-gHASH`, `-dirty` variants, `dev`; strict fail-closed on any other shape), `Compare` (numeric 3-part tag-prefix; untagged/dirty compare tag only; `dev` below any release). stdlib only. |
| `internal/selfupdate/version_test.go` | Created | Table-driven: 15 Parse cases (all spec R1 shapes + 9 unparseable) + 12 Compare ordering cases (numeric-not-lexical, untagged/dirty tag-prefix, dev). |
| `internal/selfupdate/assets.go` | Created | `AssetName(p platform.Platform) (string, error)`: table-driven OS/arch maps (macos→darwin, x86_64→amd64, aarch64→arm64, identity for linux/darwin/amd64/arm64) → `upp-{os}-{arch}.tar.gz`; unknown OS/arch fail closed with clear error. |
| `internal/selfupdate/assets_test.go` | Created | Table-driven: full PLATFORMS matrix (linux/darwin × amd64/arm64 incl. aarch64), identity entry, unknown OS, unknown arch, windows fail-closed. |

## Decisions / Deviations from Design

- **Windows in AssetName**: decided **fail closed** (error) — design D5 table has no `windows` entry and spec R3 requires unknown OS to fail closed; the CLI (U5) will report the friendlier "not supported yet" message. Documented in `assets.go` comment and locked by test `windows fails closed`.
- **`vX.Y.Z-dirty` shape** (clean tag with uncommitted changes, real `git describe --dirty` output): supported per spec R1 ("any `-dirty` suffix"), beyond the task's 4 example shapes. Triangulation caught the initial miss.
- **Strict suffix grammar**: unparseable shapes (`v1.2`, `v1.2.3.4`, `v0.1.0-gHASH` missing count, `-19-g` missing hash, non-hex hash) error instead of being mis-parsed — fail-closed consistent with spec R1's fixed grammar.
- No other deviations — implementation matches design D2/D5 and the Interfaces/Contracts block.

## Verification (this batch)

- `go test ./internal/selfupdate/ -count=1` → ok (0.003s)
- `go test ./... -count=1` → all 9 packages ok (cli 32.9s)
- `go vet ./...` → clean
- `gofmt -s -l .` → clean
- No pre-commit gates in repo (no `.git/hooks/pre-commit`, no `core.hooksPath`) — nothing to validate the commit through.

## Remaining Tasks

- [ ] 1.3 [U2] `internal/selfupdate/cache.go`: `DetectionCache` load/save, 24h TTL; tests: fresh/stale, corrupt→miss+refetch.
- [ ] 2.1–2.3 [U3] client.go + httptest suite.
- [ ] 3.1–3.2 [U4] update.go pipeline + atomic replace.
- [ ] 4.1–4.2 [U5] CLI self-update command + parser registration.
- [ ] 5.1–5.4 [U6] config flag, output strings, check hint, README.

## Risks

- U1 commits to `main` directly (no branch yet). Later PR slices will need to branch from this commit when PRs are created at delivery time — fine for `stacked-to-main` since this commit is the chain's base.
- `git describe` hash validation requires lowercase hex; git never emits uppercase in `-gHASH`, so acceptable.

---

## Batch 2: U2 (task 1.3) — detection cache

**Date**: 2026-08-12
**Batch**: U2 only (task 1.3) — chained/stacked PR slice, do NOT implement beyond U2
**Delivery**: `auto-chain`, `stacked-to-main` — no branches/PRs created in this batch (created at delivery time)
**Mode**: Strict TDD (openspec/config.yaml `strict_tdd: true`; runner `go test ./... -count=1`)
**Commit**: `694d10d feat(selfupdate): detection cache` (2 files, +248)

### Completed Tasks (cumulative: 3/10)

- [x] 1.1 [U1] (see Batch 1 record above)
- [x] 1.2 [U1] (see Batch 1 record above)
- [x] 1.3 [U2] `internal/selfupdate/cache.go`: `DetectionCache` load/save, 24h TTL; tests: fresh/stale, corrupt→miss+refetch.

### TDD Cycle Evidence (U2)

| Task | Test File | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|------|-----------|-------|------------|-----|-------|-------------|----------|
| 1.3 | `internal/selfupdate/cache_test.go` | Unit | ✅ 39/39 (package baseline `go test ./internal/selfupdate/ -count=1` → ok, 0.003s) | ✅ Written + executed: build failed, `DetectionCache`/`LoadDetectionCache`/`SaveDetectionCache`/`CacheVersion` undefined | ✅ Passed after minimal impl: 19/19 subtests green | ✅ 19 rows folded into the RED table (repo table-driven convention, same as U1): 10 load cases (fresh, stale, corrupt, empty, bad timestamp, 3 missing-field schemas, future version, missing file) + 6 freshness cases (1h, exactly-24h boundary, 24h+1s, 25h, future clock, zero value) + 3 save cases (roundtrip, nested dirs, corrupt overwrite); no Fake It possible — each row exercises real load/save/fresh logic | ✅ gofmt -s / go vet clean, code already minimal (constants extracted, config.go error style); no further refactor needed |

### Work Unit Evidence (U2)

| Evidence | Required value |
|---|---|
| Focused test command and exact result | `go test ./internal/selfupdate/ -run 'Test(Cache\|Fresh\|Load\|Save)' -count=1 -v` → ok, 19 PASS (3 top-level + 19 subtests), 0 FAIL, exit 0; package total now 58 PASS (39 U1 + 19 U2) |
| Runtime harness command/scenario and exact result | N/A — pure functions + temp-dir file I/O (`LoadDetectionCache`/`SaveDetectionCache`/`Fresh`); every case uses `t.TempDir()` fixtures, no live network or runtime boundary (per tasks.md U2 harness column) |
| Rollback boundary | Delete `internal/selfupdate/cache.go` + `cache_test.go`; revert commit `694d10d`; nothing else references the cache package (no cli/config/output touched) |

### Files Changed (this batch)

| File | Action | What Was Done |
|------|--------|---------------|
| `internal/selfupdate/cache.go` | Created | `CacheVersion` (=1) schema const, `CacheTTL` (24h), `DetectionCache{Version,Fetched,Tag}` with json tags, `Fresh(now)` TTL check (zero-value never fresh), `LoadDetectionCache(path)` → `(DetectionCache, bool)` miss-on-missing/corrupt/wrong-schema (D3), `SaveDetectionCache(path, c)` with MkdirAll + plain write (matches config.Save/ExportToFile convention). Path is caller-supplied; the U3 client will pass `{config-dir}/self-update-cache.json`. |
| `internal/selfupdate/cache_test.go` | Created | Table-driven: 10 load cases (fresh, stale-still-loads, corrupt, empty file, unparseable timestamp, missing version/fetched/tag, future schema v2, missing file) + 6 freshness cases (incl. exactly-24h `<=` boundary and future-clock) + 3 save cases (roundtrip, parent-dir creation, corrupt-file overwrite). |

### Decisions / Deviations from Design

- **Wrong-schema = miss, validated on three fields**: a load is a miss unless `Version == CacheVersion`, `Fetched` non-zero, AND `Tag` non-empty. Covers all "missing fields" variants; a schema bump (future version) is a miss, not a misread. Consistent with D3 — stricter validation only costs a re-fetch.
- **Plain write (no temp+rename)**: repo's existing writers (`config.Save`, `ExportToFile`) use `MkdirAll` + `os.Create` — no temp+rename anywhere. Followed repo convention; a torn write self-heals because the next load sees corrupt JSON → miss → re-fetch (D3). Documented in `SaveDetectionCache` comment. Atomic temp+rename is available as a hardening follow-up if desired.
- **Fresh is caller-decided**: `LoadDetectionCache` returns stale caches as valid loads; freshness is decided by the caller via `Fresh(now)` with the injected clock (`Client.Now` in U3), matching the design's `LatestCached` semantics.
- No other deviations — implementation matches design D3/D4, the `DetectionCache` contract, and the 24h TTL spec R2.

### Verification (this batch)

- `go test ./internal/selfupdate/ -count=1` → ok (0.008s), 58 PASS total
- `go test ./... -count=1` → all 9 packages ok (cli 35.6s)
- `go vet ./internal/selfupdate/` → clean
- `gofmt -s -l internal/selfupdate/` → clean (no output)
- Commit staged only U2 files (pre-existing unrelated working-tree changes in `.atl/`, `.gitignore` left untouched)

### Remaining Tasks

- [ ] 2.1–2.3 [U3] client.go + httptest suite.
- [ ] 3.1–3.2 [U4] update.go pipeline + atomic replace.
- [ ] 4.1–4.2 [U5] CLI self-update command + parser registration.
- [ ] 5.1–5.4 [U6] config flag, output strings, check hint, README.

---

## Batch 3: U3 (tasks 2.1–2.3) — HTTPS-only GitHub client

**Date**: 2026-08-12
**Batch**: U3 only (tasks 2.1–2.3) — chained/stacked PR slice, do NOT implement beyond U3
**Delivery**: `auto-chain`, `stacked-to-main` — no branches/PRs created in this batch (created at delivery time)
**Mode**: Strict TDD (openspec/config.yaml `strict_tdd: true`; runner `go test ./... -count=1`)
**Commit**: `ca18176 feat(selfupdate): HTTPS-only GitHub client` (2 files, +649)

### Completed Tasks (cumulative: 6/10)

- [x] 1.1 [U1] (see Batch 1 record above)
- [x] 1.2 [U1] (see Batch 1 record above)
- [x] 1.3 [U2] (see Batch 2 record above)
- [x] 2.1 [U3] `internal/selfupdate/client.go`: `Client{BaseURL,HTTP,CachePath,Now}`, 10s timeouts, CheckRedirect rejects off-HTTPS.
- [x] 2.2 [U3] `LatestFresh` (always fresh, write-through), `LatestCached` (silent on fail), `Download` (asset+checksums same release).
- [x] 2.3 [U3] httptest: fresh/stale cache, API 500 (hint silent, cmd error), off-HTTPS redirect, checksum match/mismatch/missing.

### TDD Cycle Evidence (U3)

| Task | Test File | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|------|-----------|-------|------------|-----|-------|-------------|----------|
| 2.1 | `internal/selfupdate/client_test.go` | Integration | ✅ 58/58 (package baseline `go test ./internal/selfupdate/ -count=1` → ok, 0.005s) | ✅ Written + executed: build failed, `Client`/`NewClient`/`checkRedirect`/`latestPath`/`downloadPath`/`Release` undefined | ✅ Passed after minimal impl: 29/29 new cases green (23 subtests + 6 top-level), race-clean | ✅ 29 cases folded into the RED table (repo table-driven convention, same as U1/U2): redirect policy 3 rows; LatestFresh 6 (ok, 500, malformed, missing tag_name, off-HTTPS redirect, nil-HTTP fallback); LatestCached 7 (fresh-cache zero-network via request counter, stale refetch+write-through, corrupt→miss→refetch, 500 silent, malformed silent, empty cache path, unwritable cache path silent); Download 5 (ok, asset 404, checksums 404, no-resolved-release, off-HTTPS redirect); http→https TLS redirect accepted; 2 timeout cases (injected 50ms client vs 300ms server) | ✅ gofmt -s / go vet clean; error-style matches config.go/cache.go; constants extracted (dialTimeout, requestTimeout, latestPath, downloadPath); no further refactor needed |
| 2.2 | `internal/selfupdate/client_test.go` | Integration | (same run as 2.1 — same file, same suite) | ✅ Written first with 2.1 (same RED build failure) | ✅ Passed: LatestFresh/LatestCached/Download behaviors all green | ✅ see rows above — every spec R2/R4 scenario for the three methods has ≥2 cases (happy + failure path) | ➖ None needed (already minimal) |
| 2.3 | `internal/selfupdate/client_test.go` | Integration | (same run) | ✅ Written first with 2.1 | ✅ Passed: full httptest suite green, zero real network (all servers are httptest; TLS via NewTLSServer) | ✅ 29 cases as listed; request-counter asserts prove "fresh cache → 0 network calls" behaviorally | ➖ None needed |

### Work Unit Evidence (U3)

| Evidence | Required value |
|---|---|
| Focused test command and exact result | `go test ./internal/selfupdate/ -run 'Test(Client\|CheckRedirect\|Download\|Latest\|RedirectToHTTPS)' -count=1 -v` → ok, 29 PASS (6 top-level: TestCheckRedirectPolicy, TestLatestFresh, TestLatestCached, TestDownload, TestRedirectToHTTPSAccepted, TestClientTimeouts + 23 subtests), 0 FAIL, exit 0; package total now 87 PASS lines (58 + 29), `-race` clean |
| Runtime harness command/scenario and exact result | httptest suite only (per tasks.md U3 harness column): 4 plain `httptest.NewServer` + 1 `httptest.NewTLSServer` + 1 slow-server timeout scenario; zero real network — every request hits a loopback test server; TLS trust via `target.Client()` for the http→https redirect case |
| Rollback boundary | Delete `internal/selfupdate/client.go` + `client_test.go`; revert commit (U3); nothing else references the client package (no cli/config/output touched — verified: `go test ./...` green with only these two files added) |

### Files Changed (this batch)

| File | Action | What Was Done |
|------|--------|---------------|
| `internal/selfupdate/client.go` | Created | `Release{Tag}`, `Client{BaseURL,HTTP,CachePath,Now}` (design contract, plus unexported `release` so `Download` fetches from the same release LatestFresh/LatestCached resolved — spec R4 "SAME release"), `NewClient` + `defaultHTTPClient` (10s dial via `net.Dialer`, 10s request `Timeout`, `ProxyFromEnvironment`, `CheckRedirect: checkRedirect`), `checkRedirect` policy (any redirect whose target scheme ≠ https fails closed — security-model delta), `LatestFresh` (always network, D4; non-200/malformed/missing tag_name → visible error), `LatestCached` (fresh cache <24h via `DetectionCache.Fresh(c.Now())` → no network; else fetch + `SaveDetectionCache` write-through; ANY failure → silent `("", false)`; empty CachePath disables caching), `Download(name)` (asset + checksums.txt from resolved release, both-or-nothing on non-200). No sha256 — verification is U4. |
| `internal/selfupdate/client_test.go` | Created | httptest suite, zero real network: `newReleaseServer` route helper (latest/download/checksums with status+body knobs and `atomic.Int32` request counter), `newTestClient` (production `NewClient` wiring + `fixedNow` clock), `assertCached` write-through check. 29 cases: redirect policy unit table (https ok / http / ftp rejected); LatestFresh (ok, 500 visible, malformed, missing tag_name, off-HTTPS redirect fails closed, nil-HTTP fallback); LatestCached (fresh → 0 requests + cached tag wins over server, stale → 1 request + write-through verified, corrupt → miss → refetch + rewrite, 500 silent, malformed silent, empty cache path, unwritable cache path → silent false); Download (ok byte equality, asset 404, checksums 404, no-resolved-release, off-HTTPS redirect); `TestRedirectToHTTPSAccepted` (http origin 302 → NewTLSServer target, accepted + served); `TestClientTimeouts` (300ms server vs injected 50ms client → LatestFresh error, LatestCached silent false). |

### Decisions / Deviations from Design

- **`Download` is release-stateful**: the design signature `Download(name string)` carries no tag, but spec R4 requires fetching from the SAME release — so the client remembers the release resolved by `LatestFresh`/`LatestCached` and `Download` uses its tag for `{BaseURL}/repos/JhnFrankz/upp/releases/download/{tag}/{name}`. Download before any resolution fails with a clear error (locked by test). Documented on the struct.
- **Production download base is U4/U5's concern**: detection and download share one `BaseURL` (design contract). In production `LatestFresh` needs `https://api.github.com` but release assets live under `github.com/.../releases/download/...`; the batch that wires the CLI (U5) must pass a BaseURL that serves both, or the design needs a second base. Flagged in Risks — U3's contract is BaseURL-relative and fully test-injected.
- **`LatestCached` treats cache-write failure as a failure** (returns silent false): the orchestrator instruction "ANY failure → silent false" is applied literally — a fetch that succeeds but cannot be persisted is still silent-false, so a hint is never shown when the cache is unwritable. Locked by test `cache write failure is silent`.
- **Nil `HTTP` falls back to the production client** (`defaultHTTPClient`): keeps zero-value `Client{BaseURL: ...}` usable and mirrors the `Now` nil-fallback. Locked by test.
- **Redirect policy checks the redirect target's scheme only** (initial plain-http URLs are permitted): this is how `http.Client.CheckRedirect` works and matches the orchestrator's "http→https is fine, https→http must FAIL". Off-HTTPS hops in either direction fail closed; a plain-http BaseURL is a U5 wiring choice, not a client bypass.
- **`checksum match/mismatch/missing` rows in tasks.md 2.3 are NOT covered in U3**: verification (sha256 vs checksums.txt) is explicitly U4 work per the orchestrator's batch scope ("do NOT implement sha256 here"); U3 delivers both byte streams and the 404/error paths. U4 will add the match/mismatch/missing tests against this client.
- No other deviations — implementation matches design D4/D6, the Interfaces/Contracts block, and spec R2/R4 + security-model delta.

### Verification (this batch)

- `go test ./internal/selfupdate/ -count=1` → ok (0.38s), 29 new cases, 0 FAIL
- `go test ./internal/selfupdate/ -count=1 -race` → ok (1.43s)
- `go test ./... -count=1` → all 9 packages ok (cli 37.9s)
- `go vet ./internal/selfupdate/` → clean
- `gofmt -s -l internal/selfupdate/` → clean (no output)
- Commit staged only U3 files (pre-existing unrelated working-tree changes in `.atl/`, `.gitignore`, `.codegraph/`, `openspec/changes/` left untouched)

### Remaining Tasks

- [ ] 3.1–3.2 [U4] update.go pipeline + atomic replace.
- [ ] 4.1–4.2 [U5] CLI self-update command + parser registration.
- [ ] 5.1–5.4 [U6] config flag, output strings, check hint, README.

---

## Batch 4: U4 (tasks 3.1–3.2) — verified atomic replace

**Date**: 2026-08-12
**Batch**: U4 only (tasks 3.1–3.2) — chained/stacked PR slice, do NOT implement beyond U4
**Delivery**: `auto-chain`, `stacked-to-main` — no branches/PRs created in this batch (created at delivery time)
**Mode**: Strict TDD (openspec/config.yaml `strict_tdd: true`; runner `go test ./... -count=1`)
**Commit**: `2fe4d9d feat(selfupdate): verified atomic replace` (2 files, +969)

### Completed Tasks (cumulative: 8/10)

- [x] 1.1 [U1] (see Batch 1 record above)
- [x] 1.2 [U1] (see Batch 1 record above)
- [x] 1.3 [U2] (see Batch 2 record above)
- [x] 2.1–2.3 [U3] (see Batch 3 record above)
- [x] 3.1 [U4] `internal/selfupdate/update.go`: sentinels (ErrDevelopmentBuild, ErrUpToDate, ErrUnsupportedPlatform, ErrChecksumMismatch, ErrNotWritable, ErrDeniedCI, ErrNotTTY); download→verify→extract `upp`→temp.
- [x] 3.2 [U4] Atomic replace: EvalSymlinks, writability preflight (no sudo), temp 0755, backup `.backup.<ts>`, rename, restore-on-failure; tests: writable/unwritable/rename-fail/symlink.

### TDD Cycle Evidence (U4)

| Task | Test File | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|------|-----------|-------|------------|-----|-------|-------------|----------|
| 3.1 | `internal/selfupdate/update_test.go` | Unit + Integration | ✅ 90/90 (12 top-level + 78 subtests, package baseline `go test ./internal/selfupdate/ -count=1` → ok) | ✅ Written + executed: build failed, `verifyChecksum`/`extract`/`Prepare`/sentinels undefined | ✅ Passed after minimal impl: 37/37 new cases green | ✅ 36 cases folded into the RED table: verifyChecksum 12 rows (match, other-entries, `*`-binary-mode, uppercase hex, CRLF, mismatch, missing, empty, garbage, truncated hex, non-hex, empty-field) + extract 12 rows (normal, extra-ignored, missing binary, traversal, nested traversal, absolute, symlink-anywhere, symlink-as-binary, hardlink, binary-as-dir, not-gzip, gzip-not-tar) + Prepare 12 subtests (happy, dev 0-net, dirty 0-net, up-to-date 1-req, newer, unsupported 0-net, mismatch, missing-entry, asset 404, checksums 404, malformed tag, missing binary in archive); the `symlink entry rejected` row caught an early-return gap — dangerous entries AFTER the binary now abort too (full-scan fail-closed) | ✅ gofmt -s / go vet clean; error-style matches client.go/cache.go; constants extracted (`binarySuffix`); no further refactor needed |
| 3.2 | `internal/selfupdate/update_test.go` | Integration | (same baseline run as 3.1) | ✅ Written + executed: build failed, `Replace`/`rename` undefined | ✅ Passed after minimal impl: 8/8 Replace cases green | ✅ 8 cases: happy (new bytes + backup old bytes + 0755 + no leftovers), symlink intact/target replaced, unwritable→ErrNotWritable+actionable message+nothing changed (root-skipped), final-rename-fail→backup restored, backup-rename-fail→untouched, restore-fail→both errors surfaced+binary gone, missing binary, missing staged | ✅ gofmt -s / go vet clean; failure paths now remove the staged temp file on every branch (leftover caught by `assertNoTempLeftovers` triangulation) |

### Work Unit Evidence (U4)

| Evidence | Required value |
|---|---|
| Focused test command and exact result | `go test ./internal/selfupdate/ -run 'Test(Update\|Replace\|VerifyChecksum\|Extract\|Prepare\|Sentinels)' -count=1 -v` → ok, 49 PASS (5 top-level: TestVerifyChecksum, TestExtract, TestPrepare, TestUpdateSentinels, TestReplace + 44 subtests), 0 FAIL, exit 0; package total now 139 PASS (17 top-level + 122 subtests), `-race` clean |
| Runtime harness command/scenario and exact result | temp-dir + httptest fixtures only (per tasks.md U4 harness column: "temp-dir replace tests, no live release"): real tar.gz archives built in-test (archive/tar + gzip), real sha256 sums, `newReleaseServer` from client_test.go serving latest/asset/checksums; zero real network; chmod-0555 unwritable scenario skips as root (`os.Geteuid()==0`, chmod cannot block root) |
| Rollback boundary | Delete `internal/selfupdate/update.go` + `update_test.go`; revert commit `2fe4d9d`; nothing else references the new functions (no cli/config/output touched — verified: full suite green with only these two files added) |

### Files Changed (this batch)

| File | Action | What Was Done |
|------|--------|---------------|
| `internal/selfupdate/update.go` | Created | Sentinel vars per design contract (`ErrDevelopmentBuild`, `ErrUpToDate`, `ErrUnsupportedPlatform`, `ErrChecksumMismatch`, `ErrNotWritable`, `ErrDeniedCI`, `ErrNotTTY`); `verifyChecksum(asset, checksums, name)` — sha256 vs checksums.txt from the SAME release, sha256sum(1) grammar verified against `make release` (`<hex>  <name>`, optional `*` binary marker), missing/malformed/mismatched entry → ErrChecksumMismatch fail-closed (spec R4, security-model delta; stricter than install.sh warn-and-skip); `extract(asset, assetName, destDir)` — gzip/tar only, writes ONLY `upp-{os}-{arch}/upp` (layout verified in Makefile release target: `tar czf ... upp-{os}-{arch}`) to `destDir/upp` mode 0755, benign extra entries ignored (spec R5), dangerous entries ANYWHERE abort (absolute paths, `..` traversal, symlinks, hardlinks — task safety list); `Prepare(c, current, p)` — dev/dirty→ErrDevelopmentBuild (0 network), AssetName→ErrUnsupportedPlatform (0 network), LatestFresh, tag compare ≥0→ErrUpToDate (latest lookup only, no download), Download (same release), verify, MkdirTemp + extract outside install path, temp removed on extract error; `Replace(execPath, newPath)` — EvalSymlinks, preflight = CreateTemp in target dir (failure → ErrNotWritable with actionable no-sudo message), stage bytes 0755 + Sync, backup `{binary}.backup.<ts>`, rename over, ANY failure → restore backup + non-zero error (restore failure surfaces both), staged temp removed on every failure path; package var `rename = os.Rename` injection seam. |
| `internal/selfupdate/update_test.go` | Created | 49 cases: TestVerifyChecksum (12 rows), TestExtract (12 rows, real tar.gz fixtures via `buildArchive` helper), TestPrepare (12 httptest subtests incl. request-count proof of 0-net for dev/dirty/unsupported and 1-request for up-to-date), TestUpdateSentinels (7 non-nil distinct errors — the U5 contract), TestReplace (8 cases: happy/backup bytes/0755/no leftovers, symlink intact+target replaced, unwritable→ErrNotWritable+actionable, final-rename injection→backup restored, backup-rename injection→untouched, restore-failure→both errors surfaced, missing binary, missing staged). Helpers: `buildArchive`, `checksumLine` (Makefile format), `errName`, `injectRename` (t.Cleanup swap), `skipIfRoot`, `assertBinary`, `assertNoTempLeftovers`. |

### Decisions / Deviations from Design

- **Full-archive dangerous-entry scan**: task says "reject unexpected entries" while spec R5 says extra paths are "ignored". Resolution: benign extra entries are ignored (spec), but ANY dangerous entry — traversal, absolute path, symlink, hardlink — aborts the extraction even when it appears AFTER the binary entry (fail closed). The initial early-return implementation missed symlinks-after-binary; the `symlink entry rejected` RED row caught it and the scan was made exhaustive.
- **checksums.txt grammar**: `make release` emits coreutils `sha256sum` lines (`<hex>  <name>`, two spaces) or `shasum -a 256` on macOS; both formats plus the `*` binary-mode marker are accepted. Lines that cannot name a file are ignored — they cannot weaken the check because the asset's own entry must still parse and match. Malformed/missing/mismatched asset entry → ErrChecksumMismatch.
- **Rename failure injection via package var**: the design's testing strategy requires "rename failure injection; assert backup restored" but the Interfaces/Contracts block defines no interface. chmod-based final-rename failure cannot restore (the restore rename fails in the same read-only dir), so `rename = os.Rename` is a package var with `injectRename` test helper; documented on the var. Tests never run in parallel, so the mutable hook is race-safe.
- **Writability preflight = the CreateTemp itself**: D7 lists "preflight" as a distinct step; since the first mutation is creating the working temp file in the target dir, CreateTemp failure IS the preflight signal, wrapped as ErrNotWritable with an actionable message ("make {dir} writable or install upp under your home, e.g. ~/.local/bin; upp never uses sudo"). Nothing is modified before it.
- **`current >= latest` → ErrUpToDate**: spec R1 covers only equality; a locally newer build must not claim an update either, so Compare ≥ 0 returns the sentinel.
- **Up-to-date still performs the latest lookup**: spec says "no download", not "no network"; ErrUpToDate is returned after LatestFresh (exactly 1 request, proven by the request counter) and before Download.
- **Permission-based tests skip as root**: chmod 0555 cannot block root, so the unwritable scenario self-skips under `os.Geteuid()==0`; the rename-injection scenarios are permission-independent and always run.
- No other deviations — implementation matches design D6/D7, the Interfaces/Contracts block, and spec R4/R5/R6 + security-model delta.

### Verification (this batch)

- `go test ./internal/selfupdate/ -count=1` → ok (0.39s), 139 PASS total (17 top-level + 122 subtests)
- `go test ./internal/selfupdate/ -count=1 -race` → ok (1.49s)
- `go test ./... -count=1` → all 9 packages ok (cli 37.9s)
- `go vet ./internal/selfupdate/` → clean
- `gofmt -s -l internal/selfupdate/` → clean (no output)
- Commit `2fe4d9d` staged only U4 files (pre-existing unrelated working-tree changes in `.atl/`, `.gitignore`, `.codegraph/`, `openspec/changes/` left untouched)

### Remaining Tasks

- [ ] 4.1–4.2 [U5] CLI self-update command + parser registration (consumes Prepare/Replace + sentinels; must pass a BaseURL serving both latest-lookup and download paths — see U3 risk).
- [ ] 5.1–5.4 [U6] config flag, output strings, check hint, README.

### Risks

- **Backup timestamp is wall-clock**: `{binary}.backup.<ts>` uses `time.Now().Format("20060102.150405")` with no injectable clock; tests glob `*.backup.*`. Collision within the same second is practically impossible for a single rename.
- **`rename` package var is a mutable test hook**: safe because no replace tests run in parallel; if parallel tests are added later, this needs a per-call seam.
- **U5 BaseURL wiring (carried from U3)**: production `LatestFresh` needs `https://api.github.com` while assets live under `github.com/.../releases/download/...`; U5 must resolve, or the design needs a second base.
- **checksums.txt is a release-publishing requirement**: releases without it fail closed at verify (by design); `make release` already generates it.


---

## Batch 5: U5 (tasks 4.1–4.2) — CLI self-update command + confirmation gate

**Date**: 2026-08-12
**Batch**: U5 only (tasks 4.1–4.2) — chained/stacked PR slice, do NOT implement beyond U5
**Delivery**: `auto-chain`, `stacked-to-main` — no branches/PRs created (created at delivery time); this batch is the PR 5 slice
**Mode**: Strict TDD (openspec/config.yaml `strict_tdd: true`; runner `go test ./... -count=1`)
**Commit**: `ba0a6df feat(cli): self-update command with confirmation gate` (10 files, +929/−16)

### Completed Tasks (cumulative: 10/10)

- [x] 1.1–1.3 [U1–U2] (see Batch 1–2 records above)
- [x] 2.1–2.3 [U3] (see Batch 3 record above)
- [x] 3.1–3.2 [U4] (see Batch 4 record above)
- [x] 4.1 [U5] `internal/cli/selfupdate.go`: dev/dirty→exit 0, no net; up-to-date→exit 0; Windows→unsupported; localized prompt (injectable Reader, versions+path); non-TTY/`--ci`→deny non-zero.
- [x] 4.2 [U5] `internal/cli/parser.go`: register `self-update` (Short "Update the upp binary itself", no flags; `--only`/`--skip` ignored + documented; unknown rejected); tests: `--yes` rejected, `--quiet` prompt kept.

### TDD Cycle Evidence (U5)

| Task | Test File | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|------|-----------|-------|------------|-----|-------|-------------|----------|
| 4.1 BaseURL resolution | `internal/selfupdate/client_test.go` | Integration | ✅ 139/139 package baseline (`go test ./internal/selfupdate/ -count=1` → ok, 0.399s) | ✅ Written + executed: build failed, `DownloadBaseURL`/`releasePath` undefined | ✅ Passed: `TestDownloadDownloadBaseURL` green — 2-server split proof (API base 1 req, web base 2 reqs) | ✅ Both branches: field set (new test) vs empty fallback (entire existing Download suite keeps passing) | ✅ gofmt -s / go vet clean |
| 4.1 output strings | `internal/output/render_test.go` | Unit | ✅ 26/26 baseline (`go test ./internal/output/ -count=1` → ok, 0.004s) | ✅ Written + executed: build failed, `SelfUpdatePrompt`/`SelfUpdateDevBuild`/`SelfUpdateUpToDate`/`SelfUpdateDone` undefined | ✅ Passed: 5/5 new tests green | ✅ en prompt exact bytes, es prompt exact bytes, quiet-never-suppresses, en message block, es message block | ✅ gofmt -s / go vet clean; methods own formatting (output-package style) |
| 4.1 command | `internal/cli/selfupdate_test.go` | Integration | ✅ cli package baseline (`go test ./internal/cli/ -count=1` → ok, 39.4s) | ✅ Written + executed: build failed, `runSelfUpdate`/`selfUpdateDeps` undefined | ✅ Passed: 14/14 behavior tests green | ✅ 14 scenarios: dev 0-net, dirty 0-net, invalid version 0-net, up-to-date 1-req, confirmed y (3 reqs, prompt, replace + backup bytes), declined n (untouched, exit 0), non-TTY ErrNotTTY at gate (3 reqs), `--ci` ErrDeniedCI 0-req, `--ci` through root.Execute, `--quiet` prompt kept, `--only`/`--skip` ignored, windows 0-net unsupported, checksum mismatch untouched + no backup, es-config Spanish prompt | ✅ gofmt -s / go vet clean; code already minimal, constants extracted (`selfUpdateAPIBase`/`selfUpdateWebBase`) |
| 4.2 registration | `parser_test.go` + `integration_test.go` | Unit + Integration | (same cli baseline) | ✅ Written: TestAddCommands 6→7 + `self-update`, TestSelfUpdateCommand_Short/_NoLocalFlags/_UnknownFlagRejected, TestRootCommand_NoArgs 7, TestSubcommandRegistration + `self-update`; executed → FAIL "unknown command \"self-update\"" (command not registered) | ✅ Passed: parser.go `AddCommands` registers the command; all new + updated tests green | ✅ `--ci` deny through full cobra Execute (wiring proof, 0 network), `--yes` cobra rejection, help Short text | ✅ gofmt -s / go vet clean |

### Work Unit Evidence (U5)

| Evidence | Required value |
|---|---|
| Focused test command and exact result | `go test ./internal/cli/ -run TestSelfUpdate -count=1` → ok (0.03s), 14 PASS; full package `go test ./internal/cli/ -count=1` → ok (40.9s), all pre-existing tests green too |
| Runtime harness command/scenario and exact result | `script -qec 'go run ./cmd/upp self-update'` (TTY) → "development build; self-update is only available for release builds", exit 0, no network. `upp self-update --ci` → deny message, exit 1. `upp self-update --yes` → "unknown flag: --yes", exit 1. `--help` lists "self-update  Update the upp binary itself". NOTE: tasks.md harness column says `go run .` but main lives at `./cmd/upp` — corrected in execution. Live-release replace flow harness: N/A (no public release with checksums.txt yet); replace is proven end-to-end by the httptest confirmed/declined/mismatch tests |
| Rollback boundary | Revert commit `ba0a6df`; delete `internal/cli/selfupdate.go` + `selfupdate_test.go`; parser/output/selfupdate-client changes restore from the same revert — nothing outside the 10 U5 files changed |

### Files Changed (this batch)

| File | Action | What Was Done |
|------|--------|---------------|
| `internal/cli/selfupdate.go` | Created | `NewSelfUpdateCommand(gf)` (Use `self-update`, Short "Update the upp binary itself", Long documents `--only`/`--skip` ignored; zero local flags → cobra rejects unknowns); `runSelfUpdate(gf, version, deps)` orchestration per design D4/D7/D8: `--ci` deny FIRST (`ErrDeniedCI`, 0 network) → `Parse` → dev/dirty message + exit 0 (client never constructed) → `platform.Detect` → `Prepare` (nil client → production `NewClient(selfUpdateAPIBase, "")` with `DownloadBaseURL = selfUpdateWebBase`) → `ErrUpToDate` → "already up to date" + exit 0 → `ErrUnsupportedPlatform` → localized not-supported error → resolved target path (`os.Executable` + `EvalSymlinks` for display; Replace re-resolves) → TTY gate (`ErrNotTTY`) → dedicated `confirmReplace` prompt (current→latest + target path, y/yes only, never suppressed) → decline = exit 0 → `Replace` → success line. `selfUpdateDeps` seams: stdin, isTTY, detect, execPath, client (zero value = production). |
| `internal/cli/selfupdate_test.go` | Created | 14 tests + helpers (`selfUpdateServer` httptest routing the 3 release endpoints, `cliArchive` real tar.gz, `cliChecksumLine` Makefile format, `fakeBinary`, `backupFiles`, `newSelfUpdateDeps`): dev/dirty 0-net + untouched binary, invalid version, up-to-date 1-request, confirmed y (prompt versions+path, replace, backup bytes, 3 reqs), declined n (untouched, no backup, exit 0), non-TTY ErrNotTTY after download, `--ci` ErrDeniedCI 0-req, `--ci` through root.Execute, `--quiet` prompt kept, `--only`/`--skip` ignored, windows 0-net unsupported, checksum mismatch untouched, es-config Spanish prompt. Zero real network. |
| `internal/cli/parser.go` | Modified | `AddCommands` registers `NewSelfUpdateCommand(gf)` (7 commands). |
| `internal/cli/parser_test.go` | Modified | TestAddCommands expects 7 commands incl. `self-update`; added TestSelfUpdateCommand_Short, _NoLocalFlags, _UnknownFlagRejected. |
| `internal/cli/integration_test.go` | Modified | TestRootCommand_NoArgs 6→7; TestSubcommandRegistration adds `self-update`. |
| `internal/output/language.go` | Modified | `Strings` + 8 self-update fields (en/es): `SelfUpdatePrompt` (`%s`→`%s`), `SelfUpdateTarget`, `SelfUpdateDevBuild`, `SelfUpdateUpToDate`, `SelfUpdateDeniedCI`, `SelfUpdateDeniedNotTTY`, `SelfUpdateUnsupported`, `SelfUpdateDone`. The hint string is deliberately NOT added (U6, task 5.2). |
| `internal/output/render.go` | Modified | `SelfUpdatePrompt` (3-line prompt: versions + target + Proceed), `SelfUpdateDevBuild`, `SelfUpdateUpToDate(tag)`, `SelfUpdateDone(current, latest)` — none of them check `quiet` (never suppressed, spec flag semantics). |
| `internal/output/render_test.go` | Modified | 5 tests: prompt exact bytes en + es, quiet-never-suppresses, messages block en + es. |
| `internal/selfupdate/client.go` | Modified | **U3 BaseURL gap closed**: `Client.DownloadBaseURL` field (doc comment with the 404 finding), `releasePath = "/JhnFrankz/upp/releases/download"` const, `downloadBase()` — `DownloadBaseURL` set → `{web base}{releasePath}/{tag}/`; empty → `{BaseURL}{downloadPath}/{tag}/` (existing tests unchanged). |
| `internal/selfupdate/client_test.go` | Modified | `newReleaseServer` also routes `releasePath`; `TestDownloadDownloadBaseURL` — two servers prove the split: latest lookup hits only the API base (1 req), asset+checksums hit only the web base (2 reqs). |

### Decisions / Deviations from Design

- **BaseURL resolution (the U3-carried gap) — decided by empirical verification**: `curl -sI https://api.github.com/repos/{owner}/{repo}/releases/download/{tag}/{asset}` → **HTTP 404** (2026-08-12), while `https://github.com/.../releases/download/...` → **302** to `release-assets.githubusercontent.com` (HTTPS, allowed by the redirect policy). So the production client keeps the API base for latest lookup and adds `DownloadBaseURL = https://github.com` for assets. Empty `DownloadBaseURL` preserves the single-base behavior all tests rely on. Zero changes to existing test behavior.
- **`--ci` deny is the FIRST gate** (before version parse, platform, and any network): orchestrator instruction "deny always"; spec scenario is unconditional. Proven: 0 requests on the `--ci` path.
- **non-TTY gate stays at the prompt point** (after download/verify/extract) exactly per design data flow (a); the deny test proves 3 requests then `ErrNotTTY`. Nothing is modified either way.
- **Prompt precedes the writability preflight**: U4 implements preflight as Replace's CreateTemp (the first mutation, per U4's decision record). Hoisting it before the prompt would duplicate package logic in the CLI, which the thin-CLI constraint forbids. Observably equivalent: unwritable → `ErrNotWritable` + nothing changed + non-zero.
- **Up-to-date message shows the current version**: `Prepare` returns `Release{}` on `ErrUpToDate`, so the latest tag is unavailable to the CLI; on this path current == latest, so the value shown is identical.
- **Self-update strings live in `internal/output` NOW** (design File Changes table assigns them to language.go/render.go): U5 added prompt/deny/dev/up-to-date/unsupported/done. **U6 note**: task 5.2's wording overlaps these — 5.2 should add only the hint string + `r.SelfUpdateHint` (+ README), not duplicate the strings U5 already added.
- **Deny/unsupported error shape**: `fmt.Errorf("%s: %w", localized, sentinel)` — the user-visible text is localized (main prints the returned error) while `errors.Is` still identifies `ErrDeniedCI`/`ErrNotTTY`/`ErrUnsupportedPlatform`.
- **`CachePath ""` for the self-update client**: `LatestFresh` never reads or writes the cache (U3 implementation), so the CLI passes no cache path — no config-dir dependency for the command. D4's "write-through" for the fresh fetch remains unimplemented (U3 built `LatestFresh` cache-free); safe because `LatestCached` writes the cache on hint checks (U6).
- **tasks.md harness command corrected**: `go run .` → `go run ./cmd/upp` (main lives at cmd/upp).
- No other deviations — implementation matches design D4/D7/D8, the Interfaces/Contracts block, and spec R1/R2/R4/R6/R7 + command-interface + ux-patterns deltas.

### Verification (this batch)

- `go test ./internal/selfupdate/ -count=1` → ok (0.40s); `-race` → ok (1.53s)
- `go test ./internal/output/ -count=1` → ok; `-race` → ok
- `go test ./internal/cli/ -run TestSelfUpdate -count=1` → ok; `-race` → ok (1.08s)
- `go test ./internal/cli/ -count=1` → ok (40.9s, full package)
- `go test ./... -count=1` → all 9 packages ok (cli 34.4s)
- `go vet ./internal/cli/ ./internal/output/ ./internal/selfupdate/` → clean
- `gofmt -s -l internal/` → clean (no output)
- Runtime harness: TTY dev-build exit 0; `--ci` deny exit 1; `--yes` rejected exit 1; `--help` lists the command
- Commit staged only the 10 U5 files (pre-existing unrelated working-tree changes in `.atl/`, `.gitignore`, `.codegraph/`, `openspec/changes/` left untouched)

### Remaining Tasks

- [ ] 5.1–5.4 [U6] config flag `CheckSelfUpdate` (`check_self_update`), hint string + `r.SelfUpdateHint`, check.go hook, README. U6 must NOT re-add the prompt/deny/dev/up-to-date/unsupported strings (added in U5).

### Risks

- **Slice size above the 400-line budget** (+929/−16, of which 494 are tests): expected for PR 5; the chained/stacked split was pre-resolved by the orchestrator (auto-chain), and this batch implements only the assigned U5 slice.
- **`Prepare` temp dir leaks after the gate**: `/tmp/upp-selfupdate-*` is not removed after a successful Replace or after a non-TTY/declined gate (pre-existing U4 behavior, now reachable from the CLI). Hardening follow-up: remove the temp dir after Replace consumes it.
- **D4 "write-through" for `LatestFresh` unimplemented** (U3 built it cache-free): safe because the hint path writes the cache on its own checks (U6).
- **U6 task 5.2 string overlap**: prompt/deny/dev/up-to-date/unsupported strings already exist in `internal/output` — 5.2 must not duplicate them.

---

## Batch 6: U6 (tasks 5.1–5.4) — opt-in self-update hint (FINAL)

**Date**: 2026-08-12
**Batch**: U6 only (tasks 5.1–5.4) — chained/stacked PR slice, do NOT implement beyond U6
**Delivery**: `auto-chain`, `stacked-to-main` — no branches/PRs created (created at delivery time); this batch is the PR 6 slice
**Mode**: Strict TDD (openspec/config.yaml `strict_tdd: true`; runner `go test ./... -count=1`)
**Commit**: `41edbad feat(check): opt-in self-update hint` (9 files, +529/−7)

### Completed Tasks (cumulative: 14/14 — ALL TASKS DONE)

- [x] 1.1–1.3 [U1–U2] (see Batch 1–2 records above)
- [x] 2.1–2.3 [U3] (see Batch 3 record above)
- [x] 3.1–3.2 [U4] (see Batch 4 record above)
- [x] 4.1–4.2 [U5] (see Batch 5 record above)
- [x] 5.1 [U6] `internal/config/config.go`: `CheckSelfUpdate` (`check_self_update`), default false; tests: absent→false, true→enabled.
- [x] 5.2 [U6] `internal/output/{language,render}.go`: hint string (en/es) + `r.SelfUpdateHint`. (Prompt/deny/dev/up-to-date/unsupported strings were added in U5 — NOT re-added; per U5's decision record.)
- [x] 5.3 [U6] `internal/cli/check.go`: hint post-CheckSummary (ON, not quiet, not dev, newer; offline silent); injected client factory; E2E zero-network default.
- [x] 5.4 [U6] README: `self-update` in Commands table + "Self-update" section + `check_self_update` in the config sample.

### TDD Cycle Evidence (U6)

| Task | Test File | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|------|-----------|-------|------------|-----|-------|-------------|----------|
| 5.1 config flag | `internal/config/config_test.go` | Unit | ✅ config package baseline (`go test ./internal/config/ -count=1` → ok, 0.008s) | ✅ Written + executed: build failed, `Settings.CheckSelfUpdate undefined` | ✅ Passed after minimal impl (field + explicit default false) | ✅ 3 load rows (absent→false, explicit false, explicit true) + TestDefaultConfig assertion | ✅ gofmt -s / go vet clean; zero-value bool documented on the field |
| 5.2 hint string + renderer | `internal/output/render_test.go` | Unit | ✅ output package baseline (`go test ./internal/output/ -count=1` → ok, 0.002s) | ✅ Written + executed: build failed, `SelfUpdateHint undefined` | ✅ Passed: 3/3 new tests green | ✅ 3 cases: en exact bytes, es exact bytes, quiet-suppresses; triangulation caught TWO bugs — (1) template arg order (latest must lead the line per spec), (2) double-`v` (`vv0.1.1`) — versions arrive pre-formatted so the template embeds no `v` | ✅ gofmt -s / go vet clean; template comment documents the leading-latest layout |
| 5.3 check hint hook + E2E | `internal/cli/check_hint_test.go` | E2E (runCheck-level, injected factory) | ✅ cli package baseline (`go test ./internal/cli/ -count=1` → ok, 35.7s) | ✅ Written + executed: build failed, `checkDeps undefined` + `too many arguments to runCheck` | ✅ Passed: 9/9 behavior tests green (7 top-level + 2 dev/dirty subtests) | ✅ 9 scenarios: default-off 0 constructions, ON newer (hint + cache created + 1 req), fresh cache 0 reqs, stale refetch + write-through, offline 500 silent, quiet 0 constructions, dev + dirty 0 constructions, up-to-date 1 req no hint | ✅ gofmt -s / go vet clean; test config now disables all catalog tools → hermetic and fast (9 tests in 4.8s vs 75s) |
| 5.4 README | N/A (docs) | N/A | N/A | N/A — docs travel with code (work-unit-commits); no behavior to test | N/A | N/A | ✅ README matches existing tone; commands table + section + config sample |

### Work Unit Evidence (U6)

| Evidence | Required value |
|---|---|
| Focused test command and exact result | `go test ./internal/cli/ -run TestCheckHint -count=1 -v` → ok, 9 PASS (7 top-level + dev/dirty subtests), 0 FAIL, exit 0 (4.8s); `go test ./internal/config/ ./internal/output/ -count=1` → ok; focused aggregate `go test ./internal/cli/ ./internal/config/ -run 'Test(CheckHint\|ZeroNetwork\|CheckSelfUpdate)' -count=1` → ok |
| Runtime harness command/scenario and exact result | Real binary against the real production client: `HOME=$(mktemp -d) go run ./cmd/upp check` with `check_self_update = true` → exit 0, summary shown, NO hint, NO error (api.github.com latest 404 — no release yet — exercises the offline-silent path against the real client; no cache written because the fetch failed, consistent with LatestCached write-on-success); `--quiet check` → exit 0, no hint. Zero-network default is proven structurally by the E2E factory tests (client never constructed), per tasks.md "E2E factory" harness |
| Rollback boundary | Revert commit `41edbad`; delete `internal/cli/check_hint_test.go`; config/output/parser/README changes restore from the same revert — nothing outside the 9 U6 files changed |

### Files Changed (this batch)

| File | Action | What Was Done |
|------|--------|---------------|
| `internal/config/config.go` | Modified | `Settings.CheckSelfUpdate bool` (`toml:"check_self_update"`), explicit `CheckSelfUpdate: false` in `DefaultConfig()` (absent → TOML zero value; spec config-system "Default off"). |
| `internal/config/config_test.go` | Modified | `TestDefaultConfig` asserts default false; new `TestLoadCheckSelfUpdate` (3 rows: absent→false, explicit false, explicit true). |
| `internal/output/language.go` | Modified | `Strings.SelfUpdateHint` en: `⬆️ upp %s available (current %s) — run "upp self-update"`, es: `⬆️ upp %s disponible (actual %s) — ejecuta "upp self-update"`. Versions arrive pre-formatted (v0.1.1) — template embeds no `v` (U5 string convention). ONLY the hint string added — U5's prompt/deny/dev/up-to-date/unsupported strings untouched. |
| `internal/output/render.go` | Modified | `SelfUpdateHint(current, latest)` — template leads with latest (spec: `v{latest} available (current {current})`); `if r.quiet { return }` — the ONE self-update output quiet suppresses (hint is informational output, unlike the confirm prompt). |
| `internal/output/render_test.go` | Modified | 3 tests: en exact bytes, es exact bytes, quiet-suppresses (empty output). |
| `internal/cli/check.go` | Modified | `selfUpdateCacheFile = "self-update-cache.json"` const; `checkDeps{clientFactory func(cachePath string) *selfupdate.Client}` seam (mirrors selfUpdateDeps; zero value = production `NewClient(selfUpdateAPIBase, cachePath)`); `runCheck(gf, version, deps)` — both `NewCheckCommand` and bare `upp` (parser.go root RunE) pass `cmd.Root().Version`; `maybeShowSelfUpdateHint(gf, r, cfg, version, deps)` after `r.CheckSummary` (design D9): gates — setting ON, not quiet (both skip BEFORE client construction → zero network), version parses + not dev/dirty, `{config-dir}/self-update-cache.json` via `config.ConfigDir()`, `LatestCached()`, parse latest, `current.Compare(latestV) >= 0` → no hint, else ONE line via `r.SelfUpdateHint`. ANY failure silent; exit unchanged. |
| `internal/cli/parser.go` | Modified | Root `RunE` passes `cmd.Root().Version` + `checkDeps{}` (bare `upp` inherits the hint). |
| `internal/cli/check_hint_test.go` | Created | 9 E2E tests + helpers (`writeCheckConfig` disables all catalog tools for hermetic fast runs, `writeDetectionCache`/`readCachedTag` RFC3339 JSON fixtures, `hintFactory` construction counter + cachePath recorder, `selfUpdateServer` reuse from selfupdate_test.go, 404/500 handler servers): default-off 0 constructions + no hint; ON newer → exact hint line `⬆️ upp v0.1.1 available (current v0.1.0) — run "upp self-update"` + cache created at `{config-dir}/self-update-cache.json` + 1 req; fresh cache → 0 reqs; stale → refetch + write-through (cache rewritten v0.1.1); offline 500 → silent exit 0; quiet → 0 constructions; dev + dirty → 0 constructions; up-to-date → 1 req, no hint. Zero real network. |
| `README.md` | Modified | Commands table + `upp self-update` row; new "Self-update" section (check → verify → confirm → atomic replace with backup; opt-in hint default OFF + 24h cache + quiet/offline semantics; non-TTY/`--ci` deny; no flags in v1; dev/dirty never claim updates; Windows not supported; `checksums.txt` required); config sample + `check_self_update = false`. |

### Decisions / Deviations from Design

- **`r.SelfUpdateHint(current, latest)` signature per task; template leads with latest**: the spec line is `⬆️ upp v{latest} available (current {current})` — the renderer swaps so `%s` #1 is latest. Triangulation caught the initial wrong order (test expected the spec line).
- **No `v` in the template**: versions arrive pre-formatted (`formatVersion` → `v0.1.0`), matching the U5 string convention (`SelfUpdatePrompt`/`UpToDate`/`Done` embed no `v`). The initial `v%s` template rendered `vv0.1.1` — triangulation caught it; final line is exactly the spec bytes.
- **Quiet gates BEFORE client construction**: spec requires the hint omitted under `--quiet`; D9's gate list ("ON, not quiet, ...") is applied at the hook level so quiet mode also performs zero network. Renderer-level `r.quiet` check is defense in depth (and the documented "one place quiet applies").
- **Dev/dirty gate before any network**: unparseable versions are treated like dev builds — no client construction. `dev` and `v0.1.0-dirty` both proven 0 constructions.
- **Up-to-date still performs the lookup** (1 request, then no hint): mirrors U4's ErrUpToDate decision — the comparison decides, the hint never claims an update that isn't there; a locally NEWER build (`current > latest`) also shows no hint (`Compare >= 0`).
- **`checkDeps` seam shape**: `clientFactory func(cachePath string) *selfupdate.Client` — the hook computes the cache path (`config.ConfigDir()` + `self-update-cache.json`), so the test factory proves the exact spec path is passed AND the real write-through lands there. Zero-value deps → production client (U5 pattern).
- **Test hermiticity**: `writeCheckConfig` disables every catalog tool for the current platform so the adapter loop is empty — the hint tests are 100% hermetic (9 tests in 4.8s vs 75s with live adapters) and assert only hint behavior.
- **tasks.md 5.2 wording**: it listed "hint/prompt/deny/dev/up-to-date/unsupported" — prompt/deny/dev/up-to-date/unsupported were added in U5 (per U5's decision record); U6 added ONLY the hint string + `r.SelfUpdateHint`. tasks.md updated with a note.
- No other deviations — implementation matches design D9, the Interfaces/Contracts delta, and spec config-system + ux-patterns + command-interface deltas.

### Verification (this batch)

- `go test ./internal/cli/ -run TestCheckHint -count=1` → ok (4.8s), 9 PASS
- `go test ./internal/cli/ ./internal/config/ ./internal/output/ -count=1` → ok (cli 37.9s)
- `go test ./... -count=1` → all 9 packages ok (cli 37.7s)
- `go test ./internal/cli/ ./internal/config/ ./internal/output/ -count=1 -race` → ok (cli 41.2s)
- `go vet ./...` → clean; `gofmt -s -l .` → clean
- Runtime harness: hint ON real binary → exit 0 silent (offline/404 path), no cache written; `--quiet` → no hint, exit 0
- Commit `41edbad` staged only the 9 U6 files (pre-existing unrelated working-tree changes in `.atl/`, `.gitignore`, `.codegraph/`, `openspec/changes/` left untouched)

### Remaining Tasks

None — all 14 tasks complete. Ready for `sdd-verify`.

### Risks

- **Runtime hint needs a real release to be observable**: with no GitHub release yet, the hint's happy path is only proven by the E2E factory tests; the real-client harness exercised the silent-failure path. Once the first release ships, `upp check` with the setting ON will show the hint.
- **Hint cache write races with concurrent upp processes** (two checks within the same second both write `self-update-cache.json`): plain write per repo convention (U2 decision); a torn write self-heals via miss-on-corrupt (D3).
- **Full change is now 6 stacked commits on main**: PR creation at delivery time must branch from `41edbad` and stack PRs 1→6 per `stacked-to-main`.
