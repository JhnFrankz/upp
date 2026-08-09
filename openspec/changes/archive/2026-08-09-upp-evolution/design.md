# Design: upp Evolution — Cross-Platform Dev Environment Updater

## Technical Approach

Rewrite the Bash-only `upp.sh` (~742 lines) as a compiled, cross-platform binary. The architecture separates concerns into CLI parsing, platform detection, tool adapters (official + custom), config management, and a security/confirmation layer. Official adapters ship with the binary; custom adapters are config-defined commands treated as untrusted.

---

## Language Evaluation

| Criterion | Go | Rust | Zig |
|-----------|-----|------|-----|
| Cross-compilation | Trivial (`GOOS=linux GOARCH=arm64 go build`) | Requires cross toolchain or CI images | Cross-compilation improving, less mature |
| Binary size | ~8-15 MB (static, no runtime) | ~2-5 MB (static, no runtime) | ~1-3 MB |
| Startup time | ~5ms | ~2ms | ~2ms |
| Compilation speed | Fast (~3s full rebuild) | Slow (~30-60s full rebuild) | Fast |
| Ecosystem maturity | Excellent — stdlib covers HTTP, TOML, CLI, testing | Excellent — crates.io rich, but selection burden | Growing, fewer crates |
| Concurrency | Goroutines — simple, built-in | tokio/async — powerful but complex | Threads, manual |
| Memory safety | GC (safe, no data races) | Ownership (safe, compile-time) | Manual, unsafe |
| Maintainability | Very high — simple language, easy to read | High if team knows Rust; steep learning curve | Medium — smaller community |
| Community & docs | Massive — huge ecosystem, tutorials, answers | Large — strong in systems/CLI space | Small — niche, fewer resources |
| Cross-platform process exec | `os/exec` — battle-tested | `std::process` — solid | `std.ChildProcess` — solid |

**Recommendation: Go**

Rationale:
1. **Cross-compilation is trivial.** Go produces static binaries for all 3 platforms + 3 architectures with a single build command. Rust and Zig require cross toolchains or CI images — more friction for a project that needs multi-platform binaries.
2. **Compilation speed matters for iteration.** This project has 12+ tool adapters that will be iterated on frequently. Fast rebuilds keep the development loop tight.
3. **Maintainability over micro-optimization.** The binary is invoked once per update cycle — startup time and binary size are negligible concerns. Readability and contributor-friendliness win.
4. **Simpler concurrency model.** Goroutines handle parallel tool checks/updates without async complexity. A CLI tool doesn't need Rust's concurrency guarantees.
5. **Ecosystem fit.** Go's stdlib + `cobra` (CLI), `toml` (config), and `os/exec` (process) cover every need with minimal dependencies.

**Risk**: Go's GC adds ~5ms startup. Acceptable for a CLI invoked manually.

---

## Architecture Decisions

### Decision: Module Structure

**Choice**: Flat package hierarchy with clear domain boundaries.

```
cmd/upp/main.go          — Entry point, version info
internal/
  cli/                   — Argument parsing, flag handling, output formatting
  config/                — TOML load/save/validate/export/import
  platform/              — OS detection, tool catalog registry
  adapters/              — Tool adapter implementations
    official/            — apt, brew, nvm, npm, pnpm, bun, gh, docker, go, opencode
    custom.go            — Config-defined custom tool adapter
    interface.go         — Adapter trait definition
  security/              — Trust levels, risk classifier, confirmation prompts
  output/                — Color/emoji/progress/summary rendering
```

**Alternatives considered**: `pkg/` for public API (rejected — no external consumers yet), plugin system with Go plugins (rejected — scope creep, distribution complexity).

**Rationale**: `internal/` enforces encapsulation. Flat packages avoid over-engineering for a single-binary CLI. The adapter interface is the only contract boundary.

### Decision: CLI Framework

**Choice**: `cobra` (de facto Go CLI standard).

**Alternatives considered**: `flag` stdlib (too manual for subcommands), `urfave/cli` (less popular, fewer examples), custom parser (unnecessary).

**Rationale**: `cobra` handles subcommands (`init`, `update`, `check`, etc.), flag inheritance (`--quiet`, `--ci`), help generation, and shell completions. It's the standard for Go CLIs.

---

## Data Flow

```
main.go
  │
  ▼
cli.Parse() ──→ Flags + Args
  │
  ▼
config.Load() ──→ Config (settings, tools, custom)
  │
  ▼
platform.Detect() ──→ OS + Arch
  │
  ▼
platform.Catalog() ──→ []Adapter (filtered by platform + config)
  │
  ▼
security.Evaluate() ──→ Trust levels, risk classification
  │
  ▼
Command Handler (init|update|check|list|export|import)
  │
  ├─→ For each adapter:
  │     detect() → check() → [confirm?] → update()
  │     result collected into Summary
  │
  ▼
output.Render(summary)
```

---

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `cmd/upp/main.go` | Create | Entry point, version, cobra setup |
| `internal/cli/parser.go` | Create | Flag parsing, global options |
| `internal/cli/flags.go` | Create | Flag struct definitions |
| `internal/config/config.go` | Create | Config struct, load/save/validate |
| `internal/config/defaults.go` | Create | Default values, platform catalog merge |
| `internal/config/export.go` | Create | Export/import round-trip |
| `internal/platform/detect.go` | Create | OS/arch detection |
| `internal/platform/catalog.go` | Create | Official tool registry per platform |
| `internal/adapters/interface.go` | Create | Adapter trait: detect/check/update/list |
| `internal/adapters/official/*.go` | Create | One file per tool (apt.go, brew.go, etc.) |
| `internal/adapters/custom.go` | Create | Config-defined custom tool adapter |
| `internal/security/trust.go` | Create | Trust levels, risk classifier |
| `internal/security/confirm.go` | Create | Interactive confirmation prompts |
| `internal/output/render.go` | Create | Color/emoji/progress/summary output |
| `internal/output/language.go` | Create | i18n string lookup (en default) |
| `go.mod` | Create | Module definition, dependencies |
| `upp.sh` | Rename | → `upp-legacy.sh` (fallback reference) |
| `Makefile` | Create | Build targets for all platforms |

---

## Interfaces / Contracts

### Adapter Interface

```go
type Adapter interface {
    Name() string
    Detect() bool
    Check() (UpdateInfo, error)
    Update(dryRun bool) (Result, error)
    Info() ToolInfo
}

type UpdateInfo struct {
    CurrentVersion  string
    LatestVersion   string
    UpdateAvailable bool
}

type Result struct {
    Success    bool
    Before     string
    After      string
    Error      error
    Privileges []string // e.g. ["sudo"]
}

type ToolInfo struct {
    ID        string
    Name      string
    Platforms []string
    Trust     TrustLevel
}
```

### Config Schema (TOML)

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

### Risk Classification

```go
type RiskLevel int

const (
    RiskLow    RiskLevel = iota // no privileges, non-destructive
    RiskMedium                  // may modify system state
    RiskHigh                    // sudo, rm, untrusted network
)

func ClassifyCommand(cmd string) RiskLevel {
    // Hybrid approach:
    // 1. Keyword matching: sudo, rm -rf, curl|sh, wget|sh, admin patterns
    // 2. Pattern matching: command chaining (&&, ||, ;), pipe to shell
    // 3. Future: AST parsing for complex commands
}
```

---

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit | Config parsing, version comparison, risk classification | Go `testing` + table-driven tests |
| Unit | Platform detection, catalog filtering | Mock OS values, verify catalog output |
| Unit | Adapter interface compliance | Shared test harness, each adapter runs against contract |
| Integration | Full update flow with mock adapters | Stub adapters that simulate success/failure/partial |
| E2E | Real tool updates on CI runners | GitHub Actions matrix (Linux, macOS, Windows) with actual tool installs |
| Security | Confirmation prompts, trust boundaries | Mock stdin for interactive tests, verify `--ci` behavior |

Key test patterns:
- Table-driven tests for flag parsing and config validation
- Golden file tests for output formatting (color vs pipe)
- Subprocess tests for adapter command execution

---

## Threat Matrix

N/A — no routing, shell commands executed by the binary itself via `os/exec`, no subprocess sandboxing, no VCS/PR automation, no executable-file classification. Official adapters invoke known package managers; custom adapters execute user-defined commands (risk-classified, confirmed).

---

## Migration / Rollout

### Phase 1: Parallel Run
- Rename `upp.sh` → `upp-legacy.sh`
- New Go binary installs as `upp`
- Both coexist; users can symlink back to `upp-legacy.sh` if needed

### Phase 2: Config Bootstrap
- `upp init` detects installed tools, generates `~/.config/upp/config.toml`
- No existing state destroyed — config is additive
- Legacy script has no config dependency

### Phase 3: Distribution
- GitHub Actions builds: `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`, `windows/amd64`
- Homebrew formula, apt .deb, winget manifest
- `curl | bash` installer for manual setup

### Rollback
- Users keep `upp-legacy.sh` in repo
- If new binary fails, `alias upp=upp-legacy.sh` or `ln -sf upp-legacy.sh /usr/local/bin/upp`
- Config file is TOML — human-readable, manually editable if needed

---

## Config Format Evaluation

| Criterion | TOML | YAML | JSON |
|-----------|------|------|------|
| Readability | Excellent — key=value, clear structure | Good — indentation-based, but whitespace-sensitive | Poor — verbose, no comments |
| Comments | Yes (`#`) | Yes (`#`) | No |
| Portability | Native in Go, Rust, Python, Ruby | Native in Go, Rust, Python | Native everywhere |
| Validation | Strong typing with struct tags | Requires validation library | Schema validation possible |
| Human-editable | Excellent | Good (indentation pitfalls) | Poor (commas, braces) |
| Git diffability | Excellent — small diffs, stable ordering | Good — indentation diffs noisy | Good — but verbose diffs |
| Ecosystem | `BurntSushi/toml`, `pelletier/go-toml` | `gopkg.in/yaml.v3` | stdlib |

**Recommendation: TOML**

Rationale: The spec already requires TOML. It's the right choice — comments, human readability, Git compatibility, and strong typing with Go struct tags. YAML's whitespace sensitivity is a liability for user-edited config files. JSON lacks comments and is too verbose for hand-editing.

---

## Tool Adapter Model

### Official Adapters

Each official tool gets a Go file implementing the `Adapter` interface. Platform-specific logic lives in the adapter (not scattered across the codebase):

```
internal/adapters/
  interface.go         — Adapter, UpdateInfo, Result, ToolInfo types
  apt.go               — Linux-only: apt update/upgrade
  brew.go              — Linux + macOS: brew update/upgrade
  winget.go            — Windows-only: winget upgrade
  scoop.go             — Windows-only: scoop update
  nvm.go               — All platforms: nvm install stable
  npm.go               — All platforms: npm update -g
  pnpm.go              — All platforms: pnpm update -g (with corruption recovery)
  bun.go               — All platforms: bun upgrade
  gh.go                — Platform dispatches to apt/brew/winget
  docker.go            — Platform dispatches to apt/brew/winget
  go.go                — Platform dispatches to brew/winget/manual
  opencode.go          — All platforms: curl installer
  custom.go            — Config-defined: exec user command
```

### Registry Pattern

```go
// platform/catalog.go
func CatalogFor(os string) []Adapter {
    var adapters []Adapter
    switch os {
    case "linux":
        adapters = append(adapters, &AptAdapter{}, &BrewAdapter{}, ...)
    case "macos":
        adapters = append(adapters, &BrewAdapter{}, ...)
    case "windows":
        adapters = append(adapters, &WingetAdapter{}, &ScoopAdapter{}, ...)
    }
    return adapters
}
```

Custom adapters are instantiated from config and appended to the catalog.

---

## Security Architecture

### Trust Flow

```
Tool arrives
  │
  ├─ Official? → TrustLevel=Trusted → Risk classified → If High: confirm
  │
  └─ Custom? → TrustLevel=Untrusted
        │
        ├─ trusted=true in config → If High: still confirm
        └─ trusted=false → If Medium+: confirm
```

### Confirmation Display (before every custom tool action)

```
  mytool (custom, untrusted)
  Command: mytool --update
  Privileges: sudo required
  Risk: HIGH
  Proceed? [y/N]
```

### `--ci` Behavior

- Official tools: auto-proceed (no prompts)
- Custom untrusted tools: **error and exit non-zero** — cannot confirm in CI
- Custom trusted tools: auto-proceed if risk < High; error if risk = High

---

## Platform Abstraction

Platform-specific logic is **isolated to adapters and platform detection** — the rest of the codebase is platform-agnostic.

```
internal/platform/
  detect.go     — runtime.GOOS, runtime.GOARCH → Platform enum
  catalog.go    — returns platform-specific adapter list
  paths.go      — config dir: ~/.config/upp (Linux/macOS), %APPDATA%/upp (Windows)
```

Adapters handle platform differences internally (e.g., `gh` adapter dispatches to `apt` on Linux, `brew` on macOS, `winget` on Windows).

---

## Distribution Strategy

### Binary Builds (GitHub Actions matrix)

| Target | GOOS | GOARCH | Output |
|--------|------|--------|--------|
| Linux x86_64 | linux | amd64 | `upp-linux-amd64` |
| Linux ARM64 | linux | arm64 | `upp-linux-arm64` |
| macOS Intel | darwin | amd64 | `upp-darwin-amd64` |
| macOS Apple Silicon | darwin | arm64 | `upp-darwin-arm64` |
| Windows x86_64 | windows | amd64 | `upp-windows-amd64.exe` |

### Package Managers

- **Homebrew**: Formula in tap repo, builds from source or binary install
- **apt**: `.deb` package built in CI
- **winget**: Manifest in winget-pkgs or custom repo
- **Manual**: `curl -fsSL https://.../install.sh | bash` (downloads correct binary)

### Binary Naming

GitHub release assets: `upp-{os}-{arch}` (`.exe` suffix for Windows). Install script renames to `upp`.

---

## Open Questions — Resolved

- [x] **i18n scope**: English-only in MVP. Architecture prepared for future languages via config/i18n system. No additional translations in v1.0.
- [x] **Config versioning**: Include `version = 1` field in config for forward-compatible migration.
- [x] **Custom tool risk heuristics**: Hybrid approach — keyword matching (`sudo`, `rm`, `curl|sh`) + intermediate pattern matching. Sufficient for MVP, can evolve to AST parsing later.
- [x] **Auto-update for upp itself**: Separate command `upp self-update`. Not part of `upp update` flow.
