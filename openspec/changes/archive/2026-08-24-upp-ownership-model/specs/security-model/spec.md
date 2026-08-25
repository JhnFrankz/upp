# Delta for security-model

## MODIFIED Requirements

### Requirement: Official Tool Integrity

Official tool adapters MUST only invoke platform-native package managers or known official installers (brew, apt, winget, scoop, nvm, npm, pnpm, official curl installers).

An owned tool (gh, docker, go) MUST NOT invoke a manager command itself; its update MUST delegate to its owning manager, so the command executed and the privileges incurred are those of the manager, and the owned tool's risk derives from its manager's operation, not its own hardcoded command. A tool with no resolving owner uses its own official installer.

Official adapters MUST NOT execute arbitrary user-provided commands.

Self-update integrity MUST fail closed: the replacement archive's sha256 MUST match `checksums.txt` from the SAME release, both fetched over HTTPS with ~10s timeouts. Mismatch or missing entry MUST abort — original binary untouched, non-zero exit (stricter than install.sh's warn-and-skip). Downloaded bytes MUST be extracted, never executed.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Official brew | Platform macOS | `brew.update()` | Runs `brew update` only |
| Linux docker delegates | Platform Linux, docker owned by apt | `docker.update()` | The owning manager (apt) updates docker; no hardcoded `apt upgrade docker-ce` |
| macOS gh delegates | Platform macOS, gh owned by brew | `gh.update()` | Delegates to brew; no hardcoded `brew upgrade gh` |
| Self-update mismatch | Archive sha256 ≠ checksums.txt | Verify | Abort, binary untouched, exit non-zero |
| Self-update missing entry | checksums.txt has no asset line | Verify | Abort, binary untouched, exit non-zero |
| Self-update HTTPS-only | Asset URL over plain HTTP | Download | Refused, exit non-zero |

(Previously: `docker.update()` on Linux ran `apt upgrade docker-ce` and `gh.update()` ran its own hardcoded manager command; an owned tool's integrity and risk were independent of any manager.)
