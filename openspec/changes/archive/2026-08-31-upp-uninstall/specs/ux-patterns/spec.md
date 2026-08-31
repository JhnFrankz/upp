# UX Patterns Specification (Delta)

## Purpose
Extends UX patterns to standardize confirmation prompts, dry-run output, and completion summaries for `upp uninstall`.

## Requirements

### Requirement: Uninstall Confirmation Prompt
In interactive mode, `upp uninstall` MUST display the exact list of targets to be removed followed by an explicit `[y/N]` confirmation prompt.

```text
The following files will be removed:
  - /usr/local/bin/upp
  - /usr/local/bin/upp.backup.20260831.100000

Configuration directory (~/.config/upp) will NOT be removed. Use --purge to remove it.

Are you sure you want to uninstall upp? [y/N]: 
```

When `--purge` is passed:
```text
The following files and directories will be removed:
  - /usr/local/bin/upp
  - /usr/local/bin/upp.backup.20260831.100000
  - /home/user/.config/upp (configuration directory)

Are you sure you want to completely uninstall upp and purge all configuration? [y/N]: 
```

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Confirmation prompt standard | TTY, `upp uninstall` | Execution | Lists binary + backups, shows prompt |
| Confirmation prompt purge | TTY, `upp uninstall --purge` | Execution | Lists binary + backups + config dir, shows prompt |
| Success summary | Confirmed and executed | Removal completes | Outputs "upp has been successfully uninstalled." |
