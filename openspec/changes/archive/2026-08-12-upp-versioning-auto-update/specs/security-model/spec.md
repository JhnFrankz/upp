# Delta for security-model

## MODIFIED Requirements

### Requirement: Official Tool Integrity

Official tool adapters MUST only invoke platform-native package managers or known official installers (brew, apt, winget, scoop, nvm, npm, pnpm, official curl installers).

Official adapters MUST NOT execute arbitrary user-provided commands.

Self-update integrity MUST fail closed: the replacement archive's sha256 MUST match `checksums.txt` from the SAME release, both fetched over HTTPS with ~10s timeouts. Mismatch or missing entry MUST abort — original binary untouched, non-zero exit (stricter than install.sh's warn-and-skip). Downloaded bytes MUST be extracted, never executed.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Official brew | Platform macOS | `brew.update()` | Runs `brew upgrade` only |
| Official docker | Platform Linux | `docker.update()` | Runs `apt upgrade docker-ce` only |
| Self-update mismatch | Archive sha256 ≠ checksums.txt | Verify | Abort, binary untouched, exit non-zero |
| Self-update missing entry | checksums.txt has no asset line | Verify | Abort, binary untouched, exit non-zero |
| Self-update HTTPS-only | Asset URL over plain HTTP | Download | Refused, exit non-zero |

(Previously: integrity covered official tool adapters only; self-update verification was undefined.)
