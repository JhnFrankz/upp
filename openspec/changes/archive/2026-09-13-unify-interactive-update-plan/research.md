```yaml
schema: gentle-ai.sdd-research/v1
revision: 1
outcome: done
change: unify-interactive-update-plan
project: upp
accessed_at: "2026-09-12"
admission:
  schema_name: gentle-ai.sdd-research-capability
  schema_version: 1
  declared_grants: [documentation, open-web]
  declared_providers:
    documentation: context7 MCP (resolve-library-id + query-docs)
    open-web: webfetch
  observed_providers:
    documentation: reachable (context7 MCP)
    open-web: unavailable (no webfetch tool exposed; MCP resource inventory empty)
  decision: admitted
questions:
  - id: Q1
    text: "In established multi-package-manager updaters (topgrade and native managers), when a tool is current but its command is an always-update/self-refresh action, is the command executed anyway or skipped as current?"
    outcome: supported
  - id: Q2
    text: "What is the documented behavior/cost of `brew update` (self-refresh) when nothing is outdated — cheap no-op vs active work?"
    outcome: supported
  - id: Q3
    text: "Any documented convention for mapping a multi-select CLI/selector's chosen items back to stable machine identifiers rather than display labels?"
    outcome: supported
evidence:
  openspec: openspec/changes/unify-interactive-update-plan/research.md
  engram: sdd/unify-interactive-update-plan/research
```

# Research: External evidence for the interactive update pending-set policy

**Change**: `unify-interactive-update-plan`
**Date**: 2026-09-12
**Outcome**: `done`
**Revision**: `1`
**Scope**: External evidence only. The product decision (whether `plan.Updates` should include `PolicyAlwaysUpdate` tools when `UpdateAvailable == false`) remains pending and is owned by the orchestrator; this artifact does not decide it.

---

## 1. Admission

| Field | Value |
|---|---|
| Declared capability schema | `gentle-ai.sdd-research-capability` (schemaVersion 1) |
| Declared grants | `documentation`, `open-web` |
| Declared providers | `documentation` = context7 MCP (`resolve-library-id` + `query-docs`); `open-web` = webfetch |
| Observed providers | `documentation` reachable and used; `open-web` not reachable in this runtime (no `webfetch` tool is exposed, and the MCP resource inventory is empty) |
| Decision | Admitted |

Admission note: the declaration carries the exact grant names required by this phase. The `open-web` provider could not be reached, so no unvalidated web-only claim is made; every claim below was obtained through the admitted `documentation` class and is mapped to a source ID. `open-web` unavailability is recorded as a deficiency, not as a claim.

---

## 2. Questions

- **Q1**: In established multi-package-manager updaters (e.g. topgrade, and native managers), when a tool is already at the latest version but its update command is an "always update / self-refresh" action (e.g. `brew update`, `bun upgrade`), is that command executed anyway, or skipped as current?
- **Q2**: What is the documented behavior/cost of `brew update` (self-refresh) when nothing is outdated — cheap no-op vs. active work? (Official docs/man pages.)
- **Q3**: Any documented convention/guidance for mapping a multi-select CLI/selector's chosen items back to stable machine identifiers rather than display labels?

---

## 3. Sources

| ID | Class | Title | Publisher | URL | Accessed | Excerpt |
|---|---|---|---|---|---|---|
| S1 | documentation | `brew update` / "update, up" — Homebrew Manpage | Homebrew (`docs.brew.sh`) | https://docs.brew.sh/Manpage | 2026-09-12 | "The update command fetches the latest version of Homebrew and all formulae from GitHub. It performs necessary migrations and can be configured with options like auto-update for faster execution or verbose mode for detailed output." |
| S2 | documentation | Homebrew > EnvConfig > Auto-Update (`HOMEBREW_AUTO_UPDATE_SECS`) | Homebrew (`docs.brew.sh`) | https://docs.brew.sh/rubydoc/Homebrew/EnvConfig.html | 2026-09-12 | "The auto-update behavior can be tuned using HOMEBREW_AUTO_UPDATE_SECS, which determines the frequency of background updates before commands like install or upgrade. The default interval is 24 hours, but this reduces to 1 hour for developer commands or 5 minutes if the API installation is disabled." |
| S3 | documentation | `HOMEBREW_NO_AUTO_UPDATE` — Homebrew FAQ | Homebrew (`docs.brew.sh`) | https://docs.brew.sh/FAQ | 2026-09-12 | "Disable automatic updates for the Homebrew environment: `export HOMEBREW_NO_AUTO_UPDATE=1`" |
| S4 | documentation | Locking installed formulae at specific versions (`$HOMEBREW_NO_AUTO_UPDATE`) | Homebrew (`docs.brew.sh`) | https://docs.brew.sh/Versions | 2026-09-12 | "Setting HOMEBREW_NO_AUTO_UPDATE prevents Homebrew from refreshing metadata automatically. This ensures Homebrew only learns about new versions when the user manually runs brew update, though it does not prevent brew upgrade from modifying installed packages." |
| S5 | documentation | CONTRIBUTING — "Implement Update Function for New Step" (topgrade) | topgrade-rs (GitHub) | https://github.com/topgrade-rs/topgrade/blob/main/CONTRIBUTING.md | 2026-09-12 | "Implement the update function for a new step, typically named `run_xxx()`. This function should check for installation, print a separator, and execute the update command." / "Check if this step is installed, if not, then this update will be skipped." |
| S6 | documentation | API Reference — Step / Runner / "Implement a Custom Update Step" (topgrade) | topgrade-rs (GitHub) | https://github.com/topgrade-rs/topgrade/blob/main/_autodocs/api-reference-index.md | 2026-09-12 | Custom step example: "Skip if already updated: `if is_current()? { return Err(SkipStep(\"Already up to date\".into()).into()); }`", and `BrewFormula => { runner.execute(*self, \"Brew\", || unix::run_brew_formula(ctx, BrewVariant::Path))?; ... }` |
| S7 | documentation | `<option>` element — Attributes (MDN Web Docs) | Mozilla (MDN) | https://developer.mozilla.org/en-US/docs/Web/HTML/Reference/Elements/option | 2026-09-12 | "The value attribute specifies the data submitted with a form when the option is selected. If omitted, the element's text content is submitted instead." |
| S8 | documentation | `HTMLOptionElement.value` / `HTMLOptionElement.label` (MDN Web Docs) | Mozilla (MDN) | https://developer.mozilla.org/en-US/docs/Web/API/HTMLOptionElement | 2026-09-12 | "A string that reflects the value of the `value` HTML attribute, if it exists; otherwise reflects value of the `Node.textContent` property." / `label` "represents the text displayed for an option". |

---

## 4. Validated claims

Each claim maps to one or more source IDs. Claims describe documented external behavior; they do not decide the product policy.

| ID | Question | Claim | Sources | Confidence | Notes / uncertainty |
|---|---|---|---|---|---|
| C1 | Q1, Q2 | `brew update` is an active action, not a currency check: it "fetches the latest version of Homebrew and all formulae from GitHub" and performs migrations. Nothing in the documented description conditions it on whether any package is outdated. | S1 | high | Official man page. Describes what the command does when invoked; does not enumerate a fast-path for "nothing outdated". |
| C2 | Q2 | Homebrew deliberately throttles its *automatic* invocation of the same self-refresh (before install/upgrade) — default at most once per 24 h, 1 h for developer commands, 5 min when API installation is disabled — and allows disabling it entirely via `HOMEBREW_NO_AUTO_UPDATE`. A self-refresh that must be rate-limited is documented as costly work, not a free no-op. | S2, S3, S4 | high | The throttle describes the implicit auto-update path. It is strong evidence about the cost model; it is not a literal statement that a manual `brew update` is slow. |
| C3 | Q1 | Topgrade's documented step contract selects work by tool **installation/presence** (`which`/`require`), not by whether an update is available; the step then executes its update command. | S5 | high | Maintainer-authored project documentation. |
| C4 | Q1 | Topgrade treats an "already up to date" check as an explicit **opt-in** custom-step pattern (`SkipStep`), not as the default: the documented `BrewFormula` step simply invokes the brew update routine for a detected Homebrew. | S6 | medium-high | The default-no-currency-check conclusion is inferred from the documented step contract plus the opt-in skip example; no doc was found that states verbatim "topgrade runs `brew update` even when nothing is outdated". |
| C5 | Q3 | The canonical documented convention for a selector separates the submitted machine value (`value` attribute) from the displayed label; when `value` is omitted, the displayed text content is submitted instead. This is the same separation of stable identifier vs. display label that the change needs when mapping selected items back to tool IDs. | S7, S8 | medium | Web-platform (HTML) convention retrieved via documentation; no CLI-specific standard on this point was located because the `open-web` provider was unavailable. Treat as analogous guidance, not a CLI mandate. |

---

## 5. Contradictions, uncertainty, freshness

- **Contradictions**: none observed across sources. S4 restates S3's mechanism (manual `brew update` still refreshes metadata even when automatic updates are disabled) and does not conflict with C1.
- **Uncertainty**:
  - C4 is indirect: it rests on the documented topgrade step contract and the explicit opt-in skip example, not on a sentence that names `brew update` and states it runs unconditionally.
  - C2 reads the cost model from the auto-update throttle; it does not present a benchmark of a manual `brew update`.
  - C5 is a web-platform (HTML) convention used as an analogy; the requested CLI-specific documentation could not be searched because the declared `open-web` provider was unreachable.
- **Freshness**: all sources were accessed 2026-09-12. S1–S4 are the live Homebrew documentation site; S5–S6 are topgrade project docs on the `main` branch (moving target); S7–S8 are MDN web platform docs.
- **Provider deficiency**: `open-web` (webfetch) was declared but not reachable in this runtime. No claim in this artifact depends on it.

---

## 6. Product choices (separate, non-authoritative)

These are decisions for the orchestrator/user — not evidence, and not decided here.

| ID | Decision | Status | Evidence relevance |
|---|---|---|---|
| PC1 | Should `plan.Updates` include `PolicyAlwaysUpdate` tools when `UpdateAvailable == false` (i.e. should current self-refresh tools such as `brew`/`bun` appear in the interactive selector's pending set)? | pending | C1–C4 show that real multi-manager updaters (topgrade) run self-refresh commands for detected tools without a currency gate, and that `brew update` carries non-trivial documented cost. Both "include" (match topgrade-style self-refresh) and "exclude" (avoid unnecessary costly refreshes) are defensible; the orchestrator owns the call. |
| PC2 | How should deselected pending tools be represented in the report (`StatusSkipped` vs. a new deselected status)? | pending | Not addressed by external evidence; carried over from the exploration. |

---

## 7. Outcome

`done` — all three questions are supported by mapped, admitted sources. The pending product choice (PC1) keeps the pre-proposal state not ready for proposal; the orchestrator must confirm decisions before `sdd-propose`.
