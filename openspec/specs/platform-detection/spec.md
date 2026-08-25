# Platform Detection & Tool Catalog Specification

## Purpose

Detect the runtime platform (OS + architecture) and provide the official tool catalog per platform. This is the foundation all other subsystems depend on.

## Requirements

### Requirement: Platform Detection

The system MUST detect the current OS (Linux, macOS, Windows) and CPU architecture (x86_64, aarch64, arm64) at startup.

The system MUST store detected platform as a structured value accessible by all adapters.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Linux x86_64 | Running on Linux x86_64 | Platform detected | OS=`linux`, arch=`x86_64` |
| macOS ARM | Running on Apple Silicon Mac | Platform detected | OS=`macos`, arch=`aarch64` |
| Windows | Running on Windows x86_64 | Platform detected | OS=`windows`, arch=`x86_64` |
| Unknown platform | Running on unsupported OS | Platform detected | Error with platform info, exit non-zero |

### Requirement: Tool Catalog

The system MUST maintain a catalog of official tools per platform. Each tool entry: id, display name, adapter type, platforms, and owning manager per platform. Manager entries (apt, brew, winget, scoop) mark themselves as managers; owned tools declare their per-platform manager.

**Linux catalog**: apt, brew, nvm, npm, pnpm, bun, gh(→apt), docker(→apt), go, opencode
**macOS catalog**: brew, nvm, npm, pnpm, bun, gh(→brew), docker(→brew), go(→brew), opencode
**Windows catalog**: winget, scoop, nvm, npm, pnpm, bun, gh(→winget), docker(→winget), go(→winget), opencode

A tool with no resolving owner on a platform (nvm, npm, pnpm, bun, opencode, go-on-Linux) MUST NOT carry a manager for that platform and remains standalone.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Linux tool lookup | Platform is Linux | Catalog queried for `apt` | Returns valid adapter |
| macOS tool exclusion | Platform is macOS | Catalog queried for `apt` | Tool not in catalog |
| Windows tool lookup | Platform is Windows | Catalog queried for `winget` | Returns valid adapter |
| gh owner on macOS | Platform is macOS | Catalog queried for `gh` | Entry has `owner=brew` |
| docker owner on Linux | Platform is Linux | Catalog queried for `docker` | Entry has `owner=apt` |
| go no owner on Linux | Platform is Linux | Catalog queried for `go` | Entry has no `owner` (stands alone) |

(Previously: the catalog entries had no owner field; each entry carried only id, display name, adapter type, and platforms.)

### Requirement: Catalog Extension

The system MUST allow the catalog to be extended via config file without recompilation. Config-defined tools are treated as custom (see security-model).

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Custom tool in config | Config has `[tools.custom.mytool]` | Catalog queried for `mytool` | Returns custom adapter entry |
| Duplicate id | Config defines tool with same id as official | Catalog merge | Official tool takes precedence, warning emitted |
