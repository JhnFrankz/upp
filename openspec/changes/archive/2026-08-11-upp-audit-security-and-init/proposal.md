# Proposal: upp audit — security confirmation & first-run init fixes

## Intent

Close two release-blocking defects (audit #40): (a) custom-tool confirmation bypass — `ClassifyCommand` never sees the real command and `trusted: true` promotes custom to official trust, so high-risk custom commands run without confirmation; (b) broken `upp init` wizard — `Load()` returns Version=1 defaults for missing files, so init thinks a config exists.

## Scope

### In Scope
- Real command into `ClassifyCommand`; `trusted: true` never promotes to official or short-circuits risk matrix; high-risk custom always confirms; `--ci` fails without explicit confirmation.
- `upp init`: explicit existence check (`os.Stat`); `Load()` calls `ApplyDefaults`.
- Regression probes for both fixes.

### Out of Scope
- i18n (inert; debt), hermetic CLI tests (t4), update-gating (t5)
- Hygiene (t6): unknown-tool validation, mapArch, dead code, custom-id collision, .atl drift
- Output-transparency extras (privileges, dry-run command)

## Capabilities

### New Capabilities
None.

### Modified Capabilities
- `security-model`: MODIFIED **Tool Trust Levels** (3 tiers: Official/CustomTrusted/CustomUntrusted; `trusted` never maps to official) and **Confirmation for Destructive Operations** (`--ci` fails high-risk custom even if `trusted = true`).
- `ux-patterns`: MODIFIED **Default Interactive Mode**: resolves contradiction — officials never prompt (security-model wins); every custom command passes the risk matrix.
- `config-system`: MODIFIED **Config Defaults**: first-run state from explicit file existence; partial configs default tools to catalog.

## Approach

- `internal/security/confirm.go`: add CustomTrusted/CustomUntrusted tiers; risk consulted before trust.
- `internal/cli/update.go`: real command into `ClassifyCommand`; `--ci` high-risk errors non-zero.
- `internal/adapters/custom.go`: `trusted` → CustomTrusted, never Official.
- `internal/cli/init.go`: `os.Stat` before `Load()`; restore overwrite prompt.
- `internal/config/config.go`: `Load()` calls `ApplyDefaults`.
- Probes in `internal/security`/`internal/cli` (from `/tmp/opencode/upp-audit/probe/repo`).

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| internal/cli/update.go | Modified | Real command into risk classification; `--ci` failure |
| internal/adapters/custom.go | Modified | `trusted` → CustomTrusted mapping |
| internal/security/confirm.go | Modified | Trust tiers; no official short-circuit for custom |
| internal/cli/init.go | Modified | `os.Stat` existence check; wizard restored |
| internal/config/config.go | Modified | `ApplyDefaults` in `Load()` |
| security/cli tests | New | Regression probes |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Behavior change: trusted custom users now prompted for high-risk | Med | Release note; probes pin new matrix |
| Empty-config behavior change (defaults to catalog) | Med | Update smoke test + "no config" assertions |
| TrustLevel refactor touches tests | Low | Same change; `go test ./... -count=1` gate |

## Rollback Plan

Revert the change branch: schema untouched, only `trusted` semantics change. No migration; deltas archived separately.

## Dependencies

None external. Audit probes (#40) reused as regression tests.

## Success Criteria

- [ ] `--ci` high-risk custom exits non-zero: "requires confirmation"
- [ ] Interactive high-risk custom prompts even with `trusted = true`
- [ ] Low-risk trusted custom proceeds without prompt
- [ ] `upp init` without config runs wizard and creates config
- [ ] `upp init` with config prompts (overwrite/merge/cancel)
- [ ] Probes committed; `go test ./... -count=1` + smoke green
- [ ] Spec deltas written and archived
