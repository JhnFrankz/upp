# Design: Self-Update & Opt-in Update Detection

## Technical Approach

New `internal/selfupdate/` package contains ALL in-process network code (repo has zero today): version compare, GitHub client, 24h cache, asset mapping, checksum verify, atomic replace. `internal/cli` wires it — `selfupdate.go` command plus hint hook in `check.go`; config gains `settings.check_self_update` (default false). Implements specs self-update R1–R7 and deltas (command-interface, config-system, security-model, ux-patterns). Proposal approach B; auto-apply out of v1.

## Architecture Decisions

| # | Decision | Options | Tradeoff | Choice |
|---|----------|---------|----------|--------|
| D1 | Package layout | `internal/selfupdate/` vs inline in cli | containment vs convenience | New package = single boundary for all network/release logic; cli stays orchestration |
| D2 | Version compare | stdlib parse vs x/mod | dep vs ~30 lines | stdlib. Numeric tag-prefix compare; `dev`/`-dirty` → ErrDevelopmentBuild, no network; untagged `-N-gHASH` compares tag only |
| D3 | Corrupted cache | miss+refetch vs error | silent vs visible | **Miss**: unparseable/wrong-schema → ignore, re-fetch, overwrite. Cache is optimization, never a gate; hint stays silent (open pt 1) |
| D4 | Cache on `self-update` | honor TTL vs always fresh | rate limit vs freshness | **Always fresh** (LatestFresh), write-through. Spec requires API failure → visible non-zero error, impossible if stale cache served; confirm gate already blocks surprise replaces; explicit command is rare vs per-check hint (open pt 2) |
| D5 | Asset mapping | table in selfupdate vs platform | coupling vs cohesion | `assets.go`: canonical (`macos`→`darwin`, `x86_64`→`amd64`, `aarch64`→`arm64`, identity rest) → `upp-{os}-{arch}.tar.gz`; unknown → fail closed. Deviation from proposal's platform/detect.go entry: platform stays generic |
| D6 | Verify | fail closed vs warn-and-skip | strictness vs availability | Fail closed: sha256 vs `checksums.txt` from SAME release, HTTPS-only (CheckRedirect rejects off-https), tar/gzip extract only `upp` entry, bytes never executed |
| D7 | Replace | backup+rename vs in-place write | atomicity vs simplicity | os.Executable → EvalSymlinks → writability preflight (no sudo) → temp 0755 → backup `.backup.<ts>` → os.Rename → restore on failure, non-zero |
| D8 | Confirm gate | reuse security.ConfirmAction vs dedicated | reuse vs semantics | Dedicated prompt: ConfirmAction auto-proceeds for official tools — wrong here. TTY always prompts (versions + path, en/es); non-TTY/`--ci` deny + non-zero; `--quiet` never suppresses |
| D9 | Hint hook | after CheckSummary | — | In `runCheck` after `r.CheckSummary` (covers bare `upp`); gated on setting ON, not quiet, not dev build, newer release, offline-silent |

## Data Flow

(a) `upp self-update`:

```
Parse(version) → dev/dirty? exit 0, no network
LatestFresh() → AssetName(Detect()) → download asset + checksums.txt (HTTPS)
→ sha256 verify (mismatch/missing → abort, untouched)
→ extract "upp" to temp dir → EvalSymlinks(Executable) → writability preflight
→ confirm prompt (TTY only) → temp 0755 → backup .backup.<ts>
→ Rename → any failure → restore backup, exit non-zero
```

(b) Hint (check/bare upp): setting ON → cache <24h? use : fetch + write `{config-dir}/self-update-cache.json`; compare; newer + not dev/dirty + not quiet → one hint line; any failure → silent, exit unchanged. Setting OFF → zero network (client never constructed).

(c) Deny paths — Windows, unknown mapping, checksum mismatch, unwritable dir, non-TTY, `--ci`: clear localized error, exit non-zero, nothing modified.

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/selfupdate/version.go` | Create | Version type, Parse, Compare |
| `internal/selfupdate/client.go` | Create | GitHub client, ~10s timeouts, HTTPS-only, LatestFresh/LatestCached |
| `internal/selfupdate/cache.go` | Create | DetectionCache schema + 24h TTL, miss-on-corrupt |
| `internal/selfupdate/assets.go` | Create | AssetName mapping table, fail closed |
| `internal/selfupdate/update.go` | Create | download→verify→extract→replace pipeline, error taxonomy |
| `internal/selfupdate/*_test.go` | Create | table-driven + httptest suites |
| `internal/cli/selfupdate.go` | Create | command wiring, prompt, deny paths |
| `internal/cli/parser.go` | Modify | register `self-update` (no local flags; `--only`/`--skip` ignored; unknown flags rejected by cobra) |
| `internal/cli/check.go` | Modify | hint hook after CheckSummary |
| `internal/config/config.go`, `defaults.go` | Modify | `CheckSelfUpdate bool` (`check_self_update`), default false |
| `internal/output/language.go`, `render.go` | Modify | en/es strings: hint, prompt, deny, dev-build, up-to-date, unsupported; `r.SelfUpdateHint` |

## Interfaces / Contracts

```go
type Version struct{ Tag [3]int; Dirty, Dev bool }
func Parse(s string) (Version, error)
func (v Version) Compare(o Version) int

type Client struct { BaseURL string; HTTP *http.Client; CachePath string; Now func() time.Time }
func (c *Client) LatestFresh() (Release, error)  // self-update: network always
func (c *Client) LatestCached() (string, bool)   // hint: cache or network; false on failure
func (c *Client) Download(name string) ([]byte, []byte, error) // asset + checksums, https-only

type DetectionCache struct { Version int `json:"version"`; Fetched time.Time `json:"fetched"`; Tag string `json:"tag"` }

func AssetName(p platform.Platform) (string, error) // "upp-{os}-{arch}.tar.gz"

var ErrDevelopmentBuild, ErrUpToDate, ErrUnsupportedPlatform,
    ErrChecksumMismatch, ErrNotWritable, ErrDeniedCI, ErrNotTTY
```

## Testing Strategy

| Layer | What | How |
|-------|------|-----|
| Unit | Version compare, AssetName, cache corrupt/miss/expiry | table-driven, spec R1/R3 scenarios verbatim |
| Integration | client: fresh/stale cache, API 500, off-HTTPS redirect, checksum match/mismatch/missing | httptest.Server, zero real network |
| Integration | replace: writable/unwritable/rename-fail/symlink | temp dirs, rename failure injection; assert backup restored |
| Integration | prompt y/n, non-TTY, `--ci` deny | injectable Reader (confirm_test.go pattern) |
| E2E | hint OFF → no client constructed (zero network); ON/quiet/offline/up-to-date | runCheck with injected client factory |

## Threat Matrix

N/A — no routing, shell, subprocess, VCS/PR automation, executable-file classification, or process-integration boundary. Downloaded bytes are extracted via archive/tar, never executed; no shell commands; no git. Rows: documentation-like paths N/A (no doc-as-code execution); git selection/commit/push/PR N/A (no git or PR automation).

## Migration / Rollout

No migration: `check_self_update` absent → false (TOML zero value); cache is a new file. Releases must ship `checksums.txt` (manual today; feature fails closed otherwise — `make publish` target is a follow-up).

## Open Questions

None — open points 1–2 resolved (D3, D4).
