# Security Model Specification (Delta)

## Purpose
Extends security classification and privilege policies to govern uninstallation.

## Requirements

### Requirement: Zero-Sudo Uninstallation Policy
`upp uninstall` MUST NEVER invoke `sudo` or attempt automatic privilege escalation. If the binary directory, executable, or configuration directory cannot be removed due to insufficient filesystem permissions, the command MUST abort before unlinking any files and output an actionable error message advising the user to adjust permissions or run with elevated privileges manually.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Unwritable directory | `/usr/local/bin` owned by root, user is non-root | `upp uninstall` | Fails closed with: "binary directory /usr/local/bin is not writable (upp never uses sudo; run with appropriate permissions or remove manually)" |
| Permission check preflight | Target directory unwritable | Execution | No files are unlinked or modified; preflight aborts immediately |
