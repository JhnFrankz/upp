# Apply Progress — upp-publish-automation (cumulative)

**Last updated**: 2026-08-12 (single batch — all apply tasks complete)
**Change**: upp-publish-automation
**Mode**: Strict TDD (shell variant — per design D6, strict TDD covers Go and there is NO Go code in scope; RED/GREEN evidence is the shell assertion matrix, `make -n` sequence, `bash -n`/shellcheck/actionlint). No Go tests invented, no fallback to Standard Mode.
**Delivery**: `auto-chain`, `stacked-to-main` → forecast resolved to **single PR, no chaining** (LOW risk, ~136 changed lines).
**Branch**: `feat/publish-automation` → PR **#24** (https://github.com/JhnFrankz/upp/pull/24), issue **#23**.

## Commit / Work Unit Evidence

| Unit | Commit | Focused test command & exact result | Runtime harness & exact result | Rollback boundary |
|------|--------|-------------------------------------|-------------------------------|-------------------|
| U1 | `2251205` feat(build): guarded make publish target | `bash /tmp/opencode/upp-guard-matrix.sh` → **11 PASS / 0 FAIL** (exit 0) | `make -n publish VERSION=v9.9.9` → exact sequence: guards 1–5 (D1 order) → `git tag -a "v9.9.9"` → `git push origin "refs/tags/v9.9.9"` → optional `gh run watch`; zero side effects (no tag created) | Delete `publish` target lines + remove `publish` from `.PHONY` only |
| U2 | `8424b17` feat(release): idempotent publish-release script | `bash -n scripts/publish-release.sh` exit 0; `shellcheck -S warning` clean | Scratch repo (annotated tag `v9.9.9` + 6 dist files): notes harness **22 PASS / 0 FAIL** — title `upp v9.9.9 — <summary>`, `## What's new` bullets, `## Assets` ×6, checksums warning; repo untouched (tags + tree unchanged) | Delete `scripts/publish-release.sh`; Makefile/CI untouched |
| U3 | `39b3f21` ci(release): tag-gated publish with job-scoped write perms | `/tmp/opencode/upp-ci-matrix.py` → **10 PASS / 0 FAIL** (PyYAML structural) | `actionlint .github/workflows/ci.yml` exit 0; `yaml.safe_load` parses clean; fork tag-push rehearsal deferred to sdd-verify (task 4.4) | Revert ci.yml hunk; script inert |
| Docs | `17ce7e1` docs(readme): document make publish flow | grep: no manual-attach text remains in README | `make help` lists `publish` target correctly | Revert README hunk only |

## TDD Cycle Evidence (strict-tdd shell variant)

| Task | Harness (evidence file) | RED | GREEN | TRIANGULATE | REFACTOR |
|------|------------------------|-----|-------|-------------|----------|
| 1.1 guard matrix | `/tmp/opencode/upp-guard-matrix.sh` | ✅ 0/12 — every scenario failed with `make: *** No rule to make target 'publish'. Stop.` (target absent) | ✅ 11/11 PASS after implementation | ✅ 11 scenarios: dirty ×2, non-main ×2, version ×2, local tag, remote tag, outside repo, dry-run, real repo | ➖ guards follow D1 order; no refactor needed |
| 1.2–1.3 guards + tag/push | same harness (S1–S9, R1, R3) | ✅ covered above | ✅ exact `ERROR:` messages + non-zero exits + no tag created | ✅ staged vs untracked variants; local vs remote tag | ➖ none |
| 1.4 `make -n` sequence | direct run + S9 in harness | ✅ S9 RED: no `git tag -a` line | ✅ exact sequence captured (see U1 row) | ✅ asserts no `push --tags`, tag before push, zero side effects | ➖ none |
| 2.1–2.3 script | `/tmp/opencode/upp-notes-matrix.sh` | ✅ `bash -n scripts/publish-release.sh` → exit 127 (file absent) | ✅ `bash -n` exit 0 + shellcheck clean + harness 22/22 PASS | ✅ N1–N13 + T1–T3 + E1–E4: prefixed/raw summaries, bullet normalization, single-line message, missing/malformed TAG, repo-untouched assertions | ➖ none |
| 2.4 notes verify | same harness | ✅ (above) | ✅ title, What's new, Assets ×6, checksums warning | ✅ 22 assertions | ➖ none |
| 3.1–3.4 ci.yml | `/tmp/opencode/upp-ci-matrix.py` | ✅ 4/8 FAIL: `needs`, job perms, `fetch-depth`, publish step absent | ✅ 10/10 PASS + actionlint + YAML parse | ✅ 10 structural assertions incl. negative gates (upload-artifact ungated, top-level read-only) | ➖ none |
| 5.1 README | grep assertions | ✅ manual-attach text present (RED) | ✅ manual attach removed; `make publish` flow + `make release` kept | ✅ CI paragraph + release section both verified | ➖ none |

## Verification Evidence (Phase 4.1–4.3, complete)

### 4.1 Guard matrix (exact outputs, 11/11)

```
PASS  S1 dirty tree (untracked)       -> ERROR: working tree not clean
PASS  S2 dirty tree (staged)          -> ERROR: working tree not clean
PASS  S3 non-main branch              -> ERROR: must run on main
PASS  S4 VERSION=1.2.0 (no v)         -> ERROR: version must match vX.Y.Z
PASS  S5 VERSION=v0.2.0-rc1           -> ERROR: version must match vX.Y.Z
PASS  S6 local tag exists             -> ERROR: tag already exists
PASS  S7 remote tag exists            -> ERROR: tag already exists on origin
PASS  S8 outside a repo               -> non-zero, ERROR:
PASS  S9 happy path (dry-run)         -> exact sequence, zero side effects
PASS  R1 real repo dirty tree         -> ERROR: working tree not clean
PASS  R3 real-repo clone non-main     -> ERROR: must run on main
```
All abort runs: exit non-zero, no tag created, pre-existing tags untouched.

### 4.2 `make -n publish VERSION=v9.9.9` exact sequence

```
test -z "$(git status --porcelain 2>/dev/null)" || { echo "ERROR: working tree not clean"; exit 1; }
test "$(git branch --show-current)" = "main" || { echo "ERROR: must run on main"; exit 1; }
echo "v9.9.9" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+$' || { echo "ERROR: version must match vX.Y.Z"; exit 1; }
git rev-parse --verify "refs/tags/v9.9.9" >/dev/null 2>&1 && { echo "ERROR: tag already exists"; exit 1; } || true
git ls-remote --exit-code origin "refs/tags/v9.9.9" >/dev/null 2>&1 && { echo "ERROR: tag already exists on origin"; exit 1; } || true
git tag -a "v9.9.9"
git push origin "refs/tags/v9.9.9"
if command -v gh >/dev/null 2>&1; then gh run watch; fi
```
Exit 0; after run `git tag -l` unchanged (v0.1.0, v0.1.1 only) — zero side effects; no `--tags` anywhere.

### 4.3 Static checks

- `bash -n scripts/publish-release.sh` → exit 0
- `shellcheck -S warning scripts/publish-release.sh` (v0.10.0, /tmp/opencode/bin) → clean
- `actionlint .github/workflows/ci.yml` (v1.7.7, /tmp/opencode/bin) → clean
- `python3 yaml.safe_load(.github/workflows/ci.yml)` → parses; ci matrix 10/10 PASS
- Safety net: `go test ./... -count=1` → all packages ok (no Go changes; suite green)

### 4.4 Fork rehearsal

**DEFERRED to sdd-verify** per session parameters (do not create a tag or publish anything for real in apply).

## Files Changed

| File | Action | What |
|------|--------|------|
| `Makefile` | Modified | `publish` in `.PHONY`; guarded publish target (D1 guards 1–5, tag -a, explicit-refspec push, optional gh run watch); help comment |
| `scripts/publish-release.sh` | Created | `notes`/`publish` subcommands, set -euo pipefail, TAG from `$GITHUB_REF_NAME`, vX.Y.Z validation, summary normalization (`upp vX.Y.Z:` prefix), create-or-upload |
| `.github/workflows/ci.yml` | Modified | release: needs [test, lint], job-scoped contents: write, checkout fetch-depth 0, publish step gated refs/tags/v with GH_TOKEN; top-level contents: read unchanged; header comment updated |
| `README.md` | Modified | Release section: make publish flow, tag-retraction guidance, make release kept; CI paragraph updated; manual attach removed |

Not touched (as required): `internal/selfupdate/*`, `scripts/install.sh`.

## Deviations from Design

None structurally. Two implementation notes, both within design intent:
1. `notes` output includes the title line `upp vTAG — <summary>` as its first line (design D4 lists summary/What's new/Assets/warning; task 2.4 explicitly requires "notes contain title"). `publish` passes the title separately via `--title` (D3) and the body leads with the summary, matching the repo's v0.1.1 release convention.
2. Summary normalization: strips an optional leading `upp vX.Y.Z:`/`upp vX.Y.Z —` prefix from the tag-message first line (repo habit, e.g. v0.1.1 tag message) so the title doesn't duplicate `upp vX.Y.Z`.

## Issues Found

- Real-repo dirty tree masks later guards in a matrix run — resolved by running reachability scenarios in scratch repos/clones (R3 uses a clean-tree clone).
- shellcheck/actionlint not installed on the machine — obtained as static binaries in `/tmp/opencode/bin` (no system install).

## Remaining Tasks

- [ ] 4.4 Fork rehearsal (deferred to sdd-verify per orchestrator).

## Status

15/16 tasks complete (4.4 deferred by design). Ready for sdd-verify.

## Workload / PR Boundary

- Mode: single PR (forecast LOW risk, ~136 lines actual — matches ~120–150 estimate)
- Work units: U1 → U2 → U3 (+docs) as 4 commits on one branch
- PR: #24, base main ← head feat/publish-automation; label type:feature; Closes #23
- Issue #23 created; `status:approved` label pending maintainer approval (not self-added).
