# upp

Cross-platform dev environment updater. One binary to update all your development tools on Linux, macOS, and Windows.

upp detects installed tools, checks for updates, and applies them safely with interactive confirmation. It ships official adapters for the most common package and runtime managers (apt, brew, npm, pnpm, nvm, bun, gh, docker, go, opencode, winget, scoop) and lets you define custom tools in a TOML config.

## Features

- **Cross-platform**: Linux (amd64, arm64), macOS (Intel, Apple Silicon), Windows (amd64)
- **Official adapters**: apt, brew, npm, pnpm, nvm, bun, gh, docker, go, opencode, winget, scoop
- **Custom tools**: define your own update commands in `config.toml`
- **Security**: trust levels, risk classification, and confirmation prompts for custom tools
- **CI mode**: non-interactive, exits non-zero on failure (`--ci`)
- **Dry run**: preview updates without applying (`--dry-run`)
- **Filtering**: `--only` and `--skip` to target specific tools
- **Export/import**: share your tool configuration as TOML

## Installation

### Binary download

Download the archive for your platform from [GitHub Releases](https://github.com/JhnFrankz/upp/releases) (each release ships `upp-<os>-<arch>.tar.gz` for Linux/macOS, `upp-windows-amd64.zip` for Windows, plus `checksums.txt`):

```bash
# Linux (amd64)
curl -fsSL -o upp.tar.gz https://github.com/JhnFrankz/upp/releases/latest/download/upp-linux-amd64.tar.gz
tar xzf upp.tar.gz upp-linux-amd64/upp
sudo mv upp-linux-amd64/upp /usr/local/bin/

# macOS (Apple Silicon)
curl -fsSL -o upp.tar.gz https://github.com/JhnFrankz/upp/releases/latest/download/upp-darwin-arm64.tar.gz
tar xzf upp.tar.gz upp-darwin-arm64/upp
sudo mv upp-darwin-arm64/upp /usr/local/bin/
```

Windows: download `upp-windows-amd64.zip` and extract `upp-windows-amd64/upp.exe`.

### Install script

```bash
curl -fsSL https://raw.githubusercontent.com/JhnFrankz/upp/main/scripts/install.sh | bash
```

The script detects your OS and architecture, downloads the matching release archive, verifies its SHA-256 checksum against `checksums.txt`, and installs the binary to `/usr/local/bin` (override with `INSTALL_DIR=/your/path` or pin a version with `VERSION=v0.1.0`).

### Build from source

```bash
git clone https://github.com/JhnFrankz/upp.git
cd upp
make build
# binary: ./upp
```

## Quick start

```bash
# Initialize config (detects installed tools)
upp init

# List detected tools
upp list

# Check for updates
upp check

# Apply updates
upp update

# Preview updates without applying
upp update --dry-run
```

## Commands

| Command | Description | Interactive | Modifies system |
|---------|-------------|-------------|-----------------|
| `upp` | Show status and available updates (read-only, like `check`) | No | No |
| `upp init` | First-run wizard: detect tools, generate `~/.config/upp/config.toml` | Yes | Yes (creates config) |
| `upp update` | Apply updates for all enabled tools | Yes | Yes |
| `upp update --dry-run` | Preview updates without executing | No | No |
| `upp check` | Check for available updates | No | No |
| `upp self-update` | Update the upp binary itself (checks, verifies, asks for confirmation) | Yes (confirm) | Yes (replaces binary) |
| `upp list` | List detected tools and their status | No | No |
| `upp export` | Export config to TOML (stdout or `-o file.toml`) | No | No |
| `upp import <file>` | Import config from a TOML file (confirms replace) | Yes | Yes (replaces config) |

## Self-update

`upp self-update` replaces the upp binary itself with the latest release: it checks the newest release over HTTPS, verifies the downloaded archive's SHA-256 against `checksums.txt`, and asks for confirmation before an atomic replace (with a timestamped `.backup.<ts>` of the previous binary). It never uses `sudo`; if the install directory is not writable, it tells you to make it writable or install under your home directory.

- **Opt-in hint**: with `settings.check_self_update = true` (default **false** — zero network calls by default), `upp check` and bare `upp` append one hint line when a newer release is known: `⬆️ upp v0.1.1 available (current v0.1.0) — run "upp self-update"`. The hint never changes the exit code, is omitted with `--quiet`, stays silent offline, and is cached for 24h in `{config-dir}/self-update-cache.json`.
- **Deny paths**: non-TTY stdin or `--ci` deny the update with a clear message and exit non-zero — never hang, auto-proceed, or silently skip. Decline at the prompt = no changes, exit 0.
- **Flags**: `self-update` accepts no flags in v1; `--only`/`--skip` are ignored. `--quiet` does not suppress the confirmation prompt.
- **Limits**: development/dirty builds never claim updates (release builds only), Windows is not supported yet, and releases must ship `checksums.txt` or the update fails closed.

## Flags

### Global flags

Available on every command:

| Flag | Description |
|------|-------------|
| `--quiet` | Reduce output to essential status only (summary still shown) |
| `--ci` | Non-interactive mode: no prompts, exit non-zero on failure |
| `--only <tools>` | Process only these tools (comma-separated, takes precedence over `--skip`) |
| `--skip <tools>` | Process all enabled tools except these (comma-separated) |
| `--version` | Print the upp version and exit |

### Command-specific flags

| Command | Flag | Description |
|---------|------|-------------|
| `update` | `--dry-run` | Preview updates without applying |
| `export` | `-o <file>` | Write output to file (default: stdout) |

## Configuration

Config file: `~/.config/upp/config.toml` on Linux/macOS, `%APPDATA%/upp/config.toml` on Windows. The directory is created on first run; if the file does not exist, upp runs with defaults.

```toml
version = 1

[settings]
interactive = true       # prompt before each update
check_self_update = false  # opt-in update hint after check/bare upp (default: off, zero network)

[tools.apt]
enabled = true
platforms = ["linux"]

[tools.brew]
enabled = true
platforms = ["linux", "macos"]

[custom.mytool]
command = "mytool --update"
check_cmd = "mytool --version"
trusted = false
```

### Tool configuration

Each official tool can be enabled or disabled, with optional platform restrictions:

```toml
[tools.npm]
enabled = true
```

### Custom tools

Define your own tools with an update `command` and an optional `check_cmd`:

```toml
[custom.my-cli]
command = "my-cli self-update"
check_cmd = "my-cli --version"
trusted = false  # requires confirmation
```

Custom tools are treated as untrusted by default. See [Security & trust](#security--trust) for how `trusted` affects confirmation.

## Security & trust

upp distinguishes two trust levels:

- **Official**: adapters implemented and maintained by the upp project, shipped with the binary. They only invoke platform-native package managers or known official installers. Official tools always proceed automatically — they never show a confirmation prompt, not even in `--ci` mode.
- **Custom**: user-defined commands from `[custom.*]` in the config. Untrusted by default.

Every update action is displayed before execution: tool name, trust level, the command, and required privileges.

The confirmation matrix below applies to **custom tools** (official tools are never prompted):

| Risk | Examples | `trusted = false` | `trusted = true` |
|------|----------|-------------------|------------------|
| Low | Non-destructive, no privileges | Proceeds with info | Proceeds silently |
| Medium | May modify system state (`apt remove`, `brew uninstall`, chained commands) | Confirmation required | Proceeds with info |
| High | `sudo`, `rm -rf`, pipe-to-shell, network to untrusted sources | Confirmation required | Confirmation required |

High-risk operations always require confirmation, regardless of trust. In `--ci` mode no prompts are shown: a custom tool that would require confirmation fails with an error telling you to run interactively or mark the tool as `trusted = true`.

## Platform support

| Platform | Architecture | Status |
|----------|--------------|--------|
| Linux | x86_64 (amd64) | Supported |
| Linux | ARM64 (aarch64) | Supported |
| macOS | Intel (amd64) | Supported |
| macOS | Apple Silicon (arm64) | Supported |
| Windows | x86_64 (amd64) | Supported |

## Development

### Prerequisites

- Go 1.22+
- Make (optional)

### Build

```bash
# Build for current platform
make build

# Cross-compile for all platforms
make build-all

# Install to /usr/local/bin (override with PREFIX=/custom/path)
make install
```

### Test

```bash
# Run all tests
make test

# Run tests with verbose output
make test-verbose

# Run tests with race detector
make test-race

# Run tests with coverage
make test-cover
```

### Lint and format

```bash
make lint    # golangci-lint (falls back to go vet)
make fmt     # gofmt -s -w
make vet     # go vet
make tidy    # go mod tidy
```

### Clean and help

```bash
make clean   # remove build artifacts (dist/, binary, coverage)
make help    # list all targets
```

### Smoke test

```bash
make smoke
```

### CI

`.github/workflows/ci.yml` runs on every push/PR: vet, a gofmt format gate, unit and race tests, a build, and the smoke test, plus golangci-lint (v1.60.3) in a separate job. On version tags (`v*`) the release job (after `test` and `lint` pass) builds the assets and publishes the GitHub Release; a manual dispatch builds and uploads the assets as artifacts without publishing.

### Release

```bash
make publish VERSION=v0.2.0
```

Publishing is one command. `make publish` checks the guards first — working tree clean, current branch `main`, `VERSION` matching `vX.Y.Z`, and the tag absent locally and on `origin` — aborting with `ERROR:` and creating nothing if any fails. It then creates the annotated tag `vX.Y.Z` and pushes it. The tag message becomes the release notes: the first line is the summary, the following lines become the `## What's new` bullets. CI then builds the assets and completes the release (title `upp vX.Y.Z — <summary>`, `## Assets`, checksums warning). If CI fails after the tag push, retract the unpublished tag (`git tag -d vX.Y.Z && git push origin :refs/tags/vX.Y.Z`), fix, and re-publish; a published tag is never retracted.

```bash
make release
```

Builds cross-platform assets into `dist/` and generates `checksums.txt` (sha256, one line per archive — the format `upp self-update` verifies). CI runs it automatically; it never tags or publishes.

## License

MIT License. See [LICENSE](LICENSE) for details.
