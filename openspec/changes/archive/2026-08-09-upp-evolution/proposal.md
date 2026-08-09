# Proposal: upp Evolution — Cross-Platform Dev Environment Updater

## Intent

upp is currently a single-file Bash CLI (~742 lines) that manages 10 tools on Linux only. The goal is to evolve it into a **cross-platform, modular, secure dev environment updater** that works identically on Linux, macOS, and Windows — so users never worry about which package manager or OS they're using.

The current script works but has hard limits: Bash-only (no Windows), no modularity (adding a tool means editing one giant file), no configuration persistence, no security model for custom tools, and no testability.

## Scope

### MVP v1.0 — Cross-Platform Core

**Platform support**: Linux, macOS, and Windows as first-class citizens — same features, same UX on all three.

**Modular architecture**:
- Tool adapters as separate, discoverable units
- Each adapter implements: `detect()`, `check()`, `update()`, `list()`
- Platform detection and per-platform tool catalog

**Official tool catalog by platform**:
- **Linux**: apt, brew, nvm, npm, pnpm, bun, gh, docker, go, opencode
- **macOS**: brew, nvm, npm, pnpm, bun, gh, docker, go, opencode
- **Windows**: winget, scoop, nvm, npm, pnpm, bun, gh, docker, go, opencode

**Configuration**:
- Portable config file (`~/.config/upp/config.toml` or equivalent — final format decided in design)
- `upp init` — interactive first-run wizard (detect installed tools, let user choose what to manage)
- `upp export` / `upp import` — share config between machines, Git-compatible
- Human-readable, versionable config

**Commands**:
- `upp init` — first-run wizard
- `upp update` — update selected tools
- `upp check` — check for available updates
- `upp list` — list installed/detected tools
- `upp dry-run` — preview what would be updated
- `upp export` — export config to stdout or file
- `upp import` — import config from file
- `--quiet` — control output verbosity (less output, not silent)
- `--ci` — non-interactive mode, no prompts, suitable for automation
- Interactive by default (explicit `--ci` to disable)

**Custom tools (minimum scope in MVP)**:
- Users can define custom tool commands in their config file (command name + update command)
- Custom tools are marked as untrusted by default
- Destructive or privileged operations on custom tools require explicit confirmation
- Show: action, origin, and required privileges before execution
- No plugin system — custom tools are simple command definitions in config only
- Architecture prepared for future: signatures, trusted repositories, sandboxing, plugin system

**Security**:
- Official providers: implemented and maintained by the project
- Custom tools: considered untrusted
- Destructive/privileged operations: require explicit confirmation
- Show action, origin, and required privileges
- Verify integrity when a reliable mechanism exists

**UX**:
- Interactive by default — asks before each update
- `--quiet` — reduces output verbosity (not silent, just less detail)
- `--ci` — non-interactive mode, no prompts, suitable for automation/CI
- English as default output language (configurable)
- Summary at the end with: updates, failures, and actions taken

**Distribution**:
- Official multiplatform binaries (Linux x86_64/aarch64, macOS Intel/Apple Silicon, Windows x86_64)
- Package managers when available (brew, apt, winget, etc.)
- Manual installation as alternative

**Rollback plan**:
- Keep `upp.sh` in repo as `upp-legacy.sh` during transition
- If new binary fails, users can symlink back to Bash script
- Configuration file is additive — no existing state is destroyed

### Future (deferred)
- Plugin system for community-contributed tools (beyond config-defined commands)
- Signatures and trusted repositories
- Sandbox execution for untrusted tools
- Remote config sync (beyond Git)

## Approach

1. **Language**: To be decided in `sdd-design` based on requirements (performance, portability, distribution, security, maintainability). Candidates: Go, Rust, or other compiled language with cross-platform support.
2. **Architecture**: Plugin-style tool adapters — each tool is a self-contained module with `detect()`, `check()`, `update()`, `list()` interfaces
3. **Config**: Portable, human-readable configuration file (TOML is a candidate; final format decided in design based on portability and readability requirements)
4. **Security**: Official providers ship with the binary; custom tools marked as untrusted, require explicit confirmation before destructive operations
5. **Migration**: Run both implementations in parallel during transition; Bash script remains as fallback

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `upp.sh` | Replaced | Becomes fallback reference (`upp-legacy.sh`); new binary takes over |
| `openspec/specs/` | New | Spec files for each capability |
| Config (`~/.config/upp/`) | New | Config file, first-run state |
| CI/CD | Modified | Build pipeline for cross-platform binaries |
| Distribution | New | GitHub releases, package manager formulas |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Cross-platform tool detection complexity (especially Windows) | High | Start with well-known tools per platform; extensible adapter model |
| Custom tools scope creep | Medium | Define minimal scope in MVP; full trust model deferred |
| User adoption regression (output language, UX changes) | Medium | Preserve familiar UX patterns; make English configurable |
| Language choice affects timeline | Medium | Design phase evaluates tradeoffs; defer decision until then |
| Windows parity with Linux/macOS | Medium | Windows adapters may have different underlying commands; UX stays identical |

## Dependencies

- Compiled language with cross-platform support (Go, Rust, or equivalent — decided in design)
- GitHub Actions for multi-platform CI/CD
- Configuration parsing library (format decided in design)

## Success Criteria

- [ ] Single binary runs on Linux (x86_64, aarch64), macOS (Intel, Apple Silicon), Windows (x86_64)
- [ ] `upp init` detects installed tools and generates config on all three platforms
- [ ] `upp update --ci` runs non-interactively without prompts
- [ ] `upp update --quiet` reduces output verbosity
- [ ] All official tools supported via modular adapters (per platform catalog)
- [ ] `upp export` / `upp import` works for config portability
- [ ] Custom tools can be defined in config with untrusted marking
- [ ] Config file is human-readable and Git-compatible
- [ ] Destructive operations require explicit confirmation
- [ ] Update summary shows successes, failures, and actions taken
- [ ] Bash script (`upp.sh`) remains functional as fallback
