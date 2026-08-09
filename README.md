# upp

Cross-platform dev environment updater. One binary to update all your development tools on Linux, macOS, and Windows.

upp detects installed tools, checks for updates, and applies them safely with interactive confirmation. Supports official package managers (apt, brew, npm, pnpm, winget, scoop), runtime managers (nvm, bun), and custom tools defined in a TOML config.

## Features

- **Cross-platform**: Linux (amd64, arm64), macOS (Intel, Apple Silicon), Windows (amd64)
- **Official adapters**: apt, brew, npm, pnpm, nvm, bun, gh, docker, go, opencode, winget, scoop
- **Custom tools**: define your own update commands in `config.toml`
- **Security**: trust levels, risk classification, interactive confirmation for destructive ops
- **CI mode**: non-interactive, exits on failure (`--ci`)
- **Dry run**: preview updates without applying (`--dry-run`)
- **Filtering**: `--only` and `--skip` to target specific tools
- **Export/import**: share your tool configuration as TOML

## Installation

### Binary download

Download the latest binary from [GitHub Releases](https://github.com/JhnFrankz/upp/releases):

```bash
# Linux (amd64)
curl -fsSL https://github.com/JhnFrankz/upp/releases/latest/download/upp-linux-amd64 -o upp
chmod +x upp
sudo mv upp /usr/local/bin/

# macOS (Apple Silicon)
curl -fsSL https://github.com/JhnFrankz/upp/releases/latest/download/upp-darwin-arm64 -o upp
chmod +x upp
sudo mv upp /usr/local/bin/
```

### Install script

```bash
curl -fsSL https://raw.githubusercontent.com/JhnFrankz/upp/main/scripts/install.sh | bash
```

### Homebrew (coming soon)

```bash
# brew install JhnFrankz/tap/upp
```

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

| Command | Description |
|---------|-------------|
| `upp` | Show status (same as `check`) |
| `upp init` | Detect tools, generate `~/.config/upp/config.toml` |
| `upp update` | Apply updates for all enabled tools |
| `upp check` | Check for available updates |
| `upp list` | List detected tools and their status |
| `upp export` | Export config to TOML (stdout or `-o file.toml`) |
| `upp import` | Import config from TOML file |

## Flags

### Global flags

| Flag | Description |
|------|-------------|
| `--quiet` | Reduce output to essential status only |
| `--ci` | Non-interactive mode (exit non-zero on failure) |
| `--only <tools>` | Process only these tools (comma-separated) |
| `--skip <tools>` | Skip these tools (comma-separated) |

### Command-specific flags

| Command | Flag | Description |
|---------|------|-------------|
| `update` | `--dry-run` | Preview updates without applying |
| `export` | `-o <file>` | Write output to file (default: stdout) |

## Configuration

Config file: `~/.config/upp/config.toml` (Linux/macOS) or `%APPDATA%/upp/config.toml` (Windows).

```toml
version = 1

[settings]
language = "en"
interactive = true

[tools.apt]
enabled = true

[tools.brew]
enabled = true

[custom.mytool]
command = "mytool --update"
check_cmd = "mytool --version"
trusted = false
```

### Tool configuration

Each official tool can be enabled/disabled:

```toml
[tools.npm]
enabled = true
```

### Custom tools

Define your own tools:

```toml
[custom.my-cli]
command = "my-cli self-update"
check_cmd = "my-cli --version"
trusted = false  # requires confirmation
```

Custom tools are treated as untrusted by default. Set `trusted = true` to skip confirmation (still confirms for high-risk commands).

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

### Lint

```bash
make lint
```

### Smoke test

```bash
make smoke
```

## Migration from upp.sh

The legacy Bash script is preserved as `upp-legacy.sh`. To use it as a fallback:

```bash
alias upp=upp-legacy.sh
# or
ln -sf upp-legacy.sh /usr/local/bin/upp
```

## License

MIT License. See [LICENSE](LICENSE) for details.