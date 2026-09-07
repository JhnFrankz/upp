# Delta for platform-detection

## MODIFIED Requirements

### Requirement: Tool Catalog

The system MUST maintain a catalog of official tools per platform. Each tool entry: id, display name, adapter type, platforms, and owning manager per platform. Manager entries (apt, brew, pacman, winget, scoop) mark themselves as managers; owned tools declare their per-platform manager.

**Linux catalog**: apt, brew, pacman, nvm, npm, pnpm, bun, uv, gh(→apt), docker(→apt), go, opencode
**macOS catalog**: brew, nvm, npm, pnpm, bun, uv, gh(→brew), docker(→brew), go(→brew), opencode
**Windows catalog**: winget, scoop, nvm, npm, pnpm, bun, uv, gh(→winget), docker(→winget), go(→winget), opencode

A tool with no resolving owner on a platform (nvm, npm, pnpm, bun, uv, opencode, go-on-Linux) MUST NOT carry a manager for that platform and remains standalone.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Linux tool lookup | Platform is Linux | Catalog queried for `apt` | Returns valid adapter |
| Linux pacman lookup | Platform is Linux | Catalog queried for `pacman` | Returns valid adapter marked as `KindManager` |
| macOS tool exclusion | Platform is macOS | Catalog queried for `apt` | Tool not in catalog |
| macOS pacman exclusion | Platform is macOS | Catalog queried for `pacman` | Tool not in catalog |
| Windows pacman exclusion | Platform is Windows | Catalog queried for `pacman` | Tool not in catalog |
| Windows tool lookup | Platform is Windows | Catalog queried for `winget` | Returns valid adapter |
| gh owner on macOS | Platform is macOS | Catalog queried for `gh` | Entry has `owner=brew` |
| docker owner on Linux | Platform is Linux | Catalog queried for `docker` | Entry has `owner=apt` |
| go no owner on Linux | Platform is Linux | Catalog queried for `go` | Entry has no `owner` (stands alone) |
| uv standalone on Linux | Platform is Linux | Catalog queried for `uv` | Returns valid adapter entry with `Kind=KindTool` and no owning manager |
| uv standalone on macOS | Platform is macOS | Catalog queried for `uv` | Returns valid adapter entry with `Kind=KindTool` and no owning manager |
| uv standalone on Windows | Platform is Windows | Catalog queried for `uv` | Returns valid adapter entry with `Kind=KindTool` and no owning manager |

(Previously: the Linux, macOS, and Windows catalogs included 11, 9, and 10 official tools respectively; `uv` was not included in any platform catalog.)
