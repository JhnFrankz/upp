# Delta for config-system

## REMOVED Requirements

### Requirement: Self-Update Detection Setting

**Reason**: This change deletes the self-update hint feature entirely (`maybeShowSelfUpdateHint`, `SelfUpdateHint`, and all hint tests); no code path renders the hint, so `settings.check_self_update` has no remaining consumer. Its config-system mandate — the opt-in boolean, the zero-network default guarantee, and the `{config-dir}/self-update-cache.json` cache created on hint checks — is removed with it. Release-cache behavior (24h TTL) remains governed solely by the self-update capability's GitHub Release Detection requirement.

**Migration**: None functional — self-update stays manual via `upp self-update`; no replacement setting exists. Existing config files containing `check_self_update = true` MUST be tolerated as an unknown settings key (ignored on load, never written back) per the Config Format forward-compatibility rule. A leftover `self-update-cache.json` in the config directory MAY remain on disk and MUST be ignored.
