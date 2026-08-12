# Tasks: Self-Update & Opt-in Update Detection

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~1,800–2,000 |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | 6 stacked PRs |
| Delivery strategy | auto-chain |
| Chain strategy | stacked-to-main |

Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: High

### Suggested Work Units

| Unit | Commit message | Focused test command | Runtime harness | Rollback boundary |
|------|----------------|----------------------|-----------------|-------------------|
| U1 → PR 1 | `feat(selfupdate): version compare + asset mapping` | `go test ./internal/selfupdate/ -run 'Test(Version\|AssetName)'` | N/A — pure logic | delete version.go, assets.go + tests |
| U2 → PR 2 | `feat(selfupdate): detection cache` | `go test ./internal/selfupdate/ -run TestCache` | N/A — pure logic | delete cache.go + tests |
| U3 → PR 3 | `feat(selfupdate): HTTPS-only GitHub client` | `go test ./internal/selfupdate/ -run TestClient` | httptest suite only | delete client.go + tests |
| U4 → PR 4 | `feat(selfupdate): verified atomic replace` | `go test ./internal/selfupdate/ -run 'Test(Update\|Replace)'` | temp-dir replace tests, no live release | delete update.go + tests |
| U5 → PR 5 | `feat(cli): self-update command + confirmation gate` | `go test ./internal/cli/ -run TestSelfUpdate` | `script -qec 'go run . self-update'` TTY; `--ci` deny | revert parser.go; delete selfupdate.go + tests |
| U6 → PR 6 | `feat(check): opt-in self-update hint` | `go test ./internal/cli/ ./internal/config/ -run 'Test(CheckHint\|ZeroNetwork\|CheckSelfUpdate)'` | `upp check` hint ON (read-only); E2E factory | revert config, check.go, output, README |

## Phase 1: Foundation (version, assets, cache)

- [x] 1.1 [U1] `internal/selfupdate/version.go`: `Version{Tag;Dirty;Dev}`, `Parse`, `Compare` (stdlib tag-prefix); tests: dev/dirty/untagged/clean.
- [x] 1.2 [U1] `internal/selfupdate/assets.go`: mapping table (macos→darwin, x86_64→amd64, aarch64→arm64, identity) → `AssetName`; tests: known + unknown fail-closed.
- [x] 1.3 [U2] `internal/selfupdate/cache.go`: `DetectionCache` load/save, 24h TTL; tests: fresh/stale, corrupt→miss+refetch.

## Phase 2: Client

- [x] 2.1 [U3] `internal/selfupdate/client.go`: `Client{BaseURL,HTTP,CachePath,Now}`, 10s timeouts, CheckRedirect rejects off-HTTPS.
- [x] 2.2 [U3] `LatestFresh` (always fresh, write-through), `LatestCached` (silent on fail), `Download` (asset+checksums same release).
- [x] 2.3 [U3] httptest: fresh/stale cache, API 500 (hint silent, cmd error), off-HTTPS redirect, checksum match/mismatch/missing.
## Phase 3: Pipeline

- [x] 3.1 [U4] `internal/selfupdate/update.go`: sentinels (ErrDevelopmentBuild, ErrUpToDate, ErrUnsupportedPlatform, ErrChecksumMismatch, ErrNotWritable, ErrDeniedCI, ErrNotTTY); download→verify→extract `upp`→temp.
- [x] 3.2 [U4] Atomic replace: EvalSymlinks, writability preflight (no sudo), temp 0755, backup `.backup.<ts>`, rename, restore-on-failure; tests: writable/unwritable/rename-fail/symlink.

## Phase 4: CLI

- [x] 4.1 [U5] `internal/cli/selfupdate.go`: dev/dirty→exit 0, no net; up-to-date→exit 0; Windows→unsupported; localized prompt (injectable Reader, versions+path); non-TTY/`--ci`→deny non-zero.
- [x] 4.2 [U5] `internal/cli/parser.go`: register `self-update` (Short "Update the upp binary itself", no flags; `--only`/`--skip` ignored + documented; unknown rejected); tests: `--yes` rejected, `--quiet` prompt kept.

## Phase 5: Config + hint

- [x] 5.1 [U6] `internal/config/{config,defaults}.go`: `CheckSelfUpdate` (`check_self_update`), default false; tests: absent→false, true→enabled.
- [x] 5.2 [U6] `internal/output/{language,render}.go`: en/es hint/prompt/deny/dev/up-to-date/unsupported + `SelfUpdateHint`. (Prompt/deny/dev/up-to-date/unsupported added in U5; U6 adds only the hint string + `r.SelfUpdateHint`.)
- [x] 5.3 [U6] `internal/cli/check.go`: hint post-CheckSummary (ON, not quiet, not dev, newer; offline silent); injected client factory; E2E zero-network default.
- [x] 5.4 [U6] README: document `self-update`, `check_self_update`.
