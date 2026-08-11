# Delta for config-system

## MODIFIED Requirements

### Requirement: Config Defaults

The system MUST provide sensible defaults for all settings. Defaults MUST be applied when fields are absent from an existing config file. First-run state MUST be determined by explicit config file existence, not inferred from applied defaults. Partial configs MUST default tool sections to the platform catalog.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Missing file | No config file exists | `upp init` | First-run wizard runs and creates config |
| Empty file | Config file exists but is empty | Config loaded | All defaults applied; NOT first-run |
| Partial config | Only `[settings]` section present | Config loaded | Tool sections default to platform catalog; NOT first-run |
| Full config | All fields present | Config loaded | Loaded as-is; defaults not applied |

(Previously: missing-file state was inferred from defaults, so first run appeared to have an existing config and the init wizard never ran; partial configs did not default tools to the catalog.)
