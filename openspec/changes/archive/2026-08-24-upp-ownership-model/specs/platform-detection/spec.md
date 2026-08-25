# Delta for platform-detection

## MODIFIED Requirements

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
