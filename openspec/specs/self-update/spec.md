# Self-Update Specification

## Purpose

`upp self-update` replaces the running upp binary with a sha256-verified release build: detect → compare → download → verify → confirm → atomic replace. It is the only self-modifying command and the containment boundary for all in-process network code (HTTPS-only, stdlib only, no new dependencies).

## Requirements

### Requirement: Version Parsing and Comparison

The system MUST parse version strings `vX.Y.Z`, `vX.Y.Z-N-gHASH`, `vX.Y.Z-N-gHASH-dirty`, or `dev` and MUST compare releases numerically on the 3-part tag prefix (`X.Y.Z`). `dev` or any `-dirty` suffix MUST be treated as a development build: no update claim, no network calls. Untagged `-N-gHASH` builds MUST compare the tag prefix only. A clean tag MUST compare fully.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Development build | Running `dev` | `upp self-update` | "development build" message, exit 0, no network |
| Dirty build | Running `v0.1.0-19-gd40e428-dirty` | `upp self-update` | "development build" message, no update claim |
| Untagged stale | Running `v0.1.0-19-gd40e428`, latest v0.1.1 | Compare | Update available (tag prefix compare) |
| Up to date | Running v0.1.1, latest v0.1.1 | `upp self-update` | "already up to date", exit 0, no download |
| Stale clean | Running v0.1.0, latest v0.1.1 | `upp self-update` | Update flow proceeds |

### Requirement: GitHub Release Detection

The system MUST query `https://api.github.com/repos/JhnFrankz/upp/releases/latest` over HTTPS with ~10s dial/read timeouts and MUST use a 24h TTL cache (bounds the 60 req/h unauthenticated limit). A fresh cache MUST be reused without network. API failure on `upp self-update` MUST report a clear error and exit non-zero. Release detection runs ONLY as part of `upp self-update`: no other command MAY trigger release detection or any self-update network call, and self-update remains manual-only.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Fresh cache | Cache < 24h old | Detection | No network call, cached result used |
| API failure (command) | API unreachable | `upp self-update` | Clear error, exit non-zero |
| Stale cache | Cache > 24h old | Detection | Re-fetch, cache refreshed |
| No hint-path detection | Any command other than `upp self-update` runs | Execution | Zero release-detection network calls from that command |

(Previously: an API failure on the opt-in hint path (`upp check`) had to stay silent with the exit code unchanged; the hint path no longer exists and detection is exclusive to `upp self-update`.)

### Requirement: Asset Mapping

The system MUST map detected OS/arch to release asset names through a dedicated table: `macos`→`darwin`, `x86_64`→`amd64`, `aarch64`→`arm64` (identity for `linux`, `darwin`, `amd64`, `arm64`), producing `upp-{os}-{arch}.tar.gz`. Unknown OS/arch MUST fail closed with a clear message and non-zero exit. The mapping MUST be table-driven and covered by table-driven tests.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| macOS x86_64 | Detected macos/x86_64 | Map | `upp-darwin-amd64.tar.gz` |
| Linux arm64 | Detected linux/aarch64 | Map | `upp-linux-arm64.tar.gz` |
| Unknown OS | Detected freebsd | Map | Clear error, exit non-zero, no download |

### Requirement: Download and Checksum Verification

The system MUST download the asset and `checksums.txt` from the SAME release over HTTPS only (no off-HTTPS redirects, ~10s timeouts) and MUST verify the archive sha256 against `checksums.txt` before extraction. Mismatch or missing entry MUST abort: binary untouched, no backup created, non-zero exit.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Match | sha256 matches checksums.txt | Verify | Proceeds to extraction |
| Mismatch | sha256 differs from checksums.txt | Verify | Abort, binary untouched, exit non-zero |
| Missing entry | No line for asset in checksums.txt | Verify | Abort, binary untouched, exit non-zero |
| Off-HTTPS redirect | Download redirects to http | Download | Fails closed, exit non-zero |

### Requirement: Extraction Safety

The system MUST extract only `tar.gz` archives, MUST write only the known binary path (`upp`) from the archive, and MUST NOT execute downloaded bytes. Extraction MUST target a temp location outside the install path.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Normal | Archive contains `upp` | Extract | Binary written to temp location only |
| Extra entries | Archive contains other paths | Extract | Extra paths ignored, only `upp` written |

### Requirement: Atomic Replacement

On Linux/macOS amd64/arm64, the system MUST resolve `os.Executable()` via `EvalSymlinks`, preflight target writability, write the verified binary to a temp file (mode 0755), create backup `{binary}.backup.<ts>`, then `os.Rename` over the binary. Any failure MUST restore the backup and exit non-zero. An unwritable target MUST produce an actionable error WITHOUT attempting sudo.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Writable target | `~/.local/bin` writable | Replace | New binary in place, backup exists |
| Unwritable target | `/usr/local/bin` not writable | Preflight | Actionable error, exit non-zero, nothing changed |
| Rename failure | Rename fails mid-replace | Replace | Backup restored, exit non-zero |
| Symlinked binary | `upp` is a symlink | Resolve | Symlink target replaced, symlink intact |

### Requirement: Confirmation Gate

`upp self-update` MUST require explicit user confirmation before replacing the binary. Non-TTY stdin or `--ci` MUST deny with a clear message and non-zero exit — never hang, never auto-proceed, never silently skip.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Confirmed | TTY, user answers y | Prompt | Replacement proceeds |
| Declined | TTY, user answers n | Prompt | No changes, exit 0 |
| Non-TTY | stdin is piped | `upp self-update` | Deny message, exit non-zero |
| `--ci` | `upp self-update --ci` | Execution | Deny message, exit non-zero |

### Requirement: Platform Support

`upp self-update` MUST support Linux and macOS on amd64/arm64. On Windows it MUST print a clear "not supported yet" error and exit non-zero — never silently skip.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Linux/macOS | amd64 or arm64 | `upp self-update` | Full update flow available |
| Windows | Running on Windows | `upp self-update` | "not supported yet" error, exit non-zero |
