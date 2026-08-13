```yaml
schema: gentle-ai.verify-result/v1
evidence_revision: sha256:56bdc0ea3286781ea53bc111ab7467600300fe616b536f9747bc3c5de70cb6d3
verdict: pass_with_warnings
blockers: 0
critical_findings: 0
requirements: 8/8
scenarios: 25/25
test_command: go test ./... -count=1
test_exit_code: 0
test_output_hash: sha256:8b31c89c4b6fa0d552637473388691f3fcba7108f5c8a69ca2cc062e7ad86645
build_command: go vet ./...
build_exit_code: 0
build_output_hash: sha256:d23c0f05910f10fee0717ccf63308e3d6232d1f7f5d24209ee85af30e16a5a4b
```

# Verification Report

**Change**: upp-publish-automation
**Version**: N/A (spec has no version field)
**Mode**: Strict TDD (shell variant — per design D6, RED/GREEN evidence is shell assertion matrices; no Go code in scope, no fallback to Standard Mode)
**Branch**: feat/publish-automation (PR #24, OPEN, base main)
**Date**: 2026-08-12

## Completeness

| Metric | Value |
|--------|-------|
| Tasks total | 17 (1.1–1.4, 2.1–2.4, 3.1–3.4, 4.1–4.4, 5.1) |
| Tasks complete | 16 |
| Tasks incomplete | 1 (4.4 fork rehearsal — deferred by orchestrator, recorded UNEXECUTED below) |

Note: apply-progress.md states "15/16 tasks complete" — a miscount; tasks.md contains 17 tasks of which 16 are checked and only 4.4 is unchecked.

## Build & Tests Execution

**Build**: ✅ Passed
```text
$ go vet ./...
exit 0 (no findings)
```

**Tests**: ✅ all green
```text
$ go test ./... -count=1
?   	github.com/JhnFrankz/upp/cmd/upp	[no test files]
ok  	github.com/JhnFrankz/upp/internal/adapters	0.067s
ok  	github.com/JhnFrankz/upp/internal/adapters/official	0.054s
ok  	github.com/JhnFrankz/upp/internal/cli	34.752s
ok  	github.com/JhnFrankz/upp/internal/config	0.012s
ok  	github.com/JhnFrankz/upp/internal/output	0.007s
ok  	github.com/JhnFrankz/upp/internal/platform	0.005s
ok  	github.com/JhnFrankz/upp/internal/security	0.008s
ok  	github.com/JhnFrankz/upp/internal/selfupdate	0.408s
exit 0 — Go safety net green; zero Go files changed by this change
```

**Shell verification (this change's runtime evidence, re-executed fresh)**:

| Evidence | Command | Result |
|----------|---------|--------|
| Guard matrix (U1) | `bash /tmp/opencode/upp-guard-matrix.sh` (reused apply harness) | 11 PASS / 0 FAIL, exit 0 |
| Dry-run sequence | `make -n publish VERSION=v9.9.9` | exact D1 order; `git tag -a "v9.9.9"` → `git push origin "refs/tags/v9.9.9"`; no `--tags`; exit 0; zero side effects (`git tag -l` still only v0.1.0, v0.1.1) |
| Notes harness (U2) | `bash /tmp/opencode/upp-notes-matrix.sh` (reused apply harness) | 22 PASS / 0 FAIL, exit 0 |
| Syntax + lint (U2) | `bash -n scripts/publish-release.sh`; `/tmp/opencode/bin/shellcheck -S warning scripts/publish-release.sh` | exit 0 / clean |
| CI matrix (U3) | `python3 /tmp/opencode/upp-ci-matrix.py` (reused apply harness) | 10 PASS / 0 FAIL, exit 0 |
| CI lint + parse | `/tmp/opencode/bin/actionlint .github/workflows/ci.yml`; `yaml.safe_load` | clean; parses OK |
| README assertions | `grep -nE "attach|Drag|Upload|gh release upload|Edit release" README.md` | no matches (manual-attach removed); `make publish` flow present; `make release` docs kept |
| CI tag-creation ban | `grep -nE 'git (tag|push)|gh release delete' .github/workflows/ci.yml scripts/publish-release.sh` | only read-only `git tag -l` present; no tag creation/push/release deletion anywhere in the CI path |

Harness disposition: all three apply harnesses found in /tmp/opencode (`upp-guard-matrix.sh`, `upp-notes-matrix.sh`, `upp-ci-matrix.py`) and REUSED — re-executed fresh for this verify, outputs captured to /tmp/opencode/verify-ev/*.out.

**Coverage**: ➖ Not available — no shell coverage tool; zero Go files changed (Go coverage N/A for this change).

## Spec Compliance Matrix

| # | Requirement | Scenario | Covering test | Result |
|---|-------------|----------|---------------|--------|
| 1 | Publish Guards | Dirty tree | guard-matrix S1, S2, R1 (runtime) | ✅ COMPLIANT |
| 2 | Publish Guards | Wrong branch | guard-matrix S3, R3 (runtime) | ✅ COMPLIANT |
| 3 | Publish Guards | Invalid version | guard-matrix S4, S5 (runtime) | ✅ COMPLIANT |
| 4 | Publish Guards | Tag exists locally | guard-matrix S6 (runtime) | ✅ COMPLIANT |
| 5 | Publish Guards | Tag exists remotely | guard-matrix S7 (runtime) | ✅ COMPLIANT |
| 6 | Publish Guards | Happy path | guard-matrix S9 dry-run (runtime) | ⚠️ PARTIAL — exact tag-a + refspec sequence proven with zero side effects; real tag creation + push to origin never executed (task 4.4) |
| 7 | Tag Semantics | Annotated tag | make -n sequence + notes harness annotated tags (runtime) | ⚠️ PARTIAL — `git tag -a` + explicit refspec proven; real `make publish` tag creation/push unexecuted (4.4) |
| 8 | Tag Semantics | Prerelease | guard-matrix S5 (runtime) | ✅ COMPLIANT |
| 9 | Tag Semantics | CI tag creation | grep assertion (runtime static) — only `git tag -l` (read) in CI path | ✅ COMPLIANT |
| 10 | CI Release Gate | Tag push, CI green | ci-matrix C2/C6/C7/C8 (runtime static) | ⚠️ PARTIAL — gate + publish step structurally verified; live `gh release create` on a tag push unexecuted (4.4) |
| 11 | CI Release Gate | Tag push, CI red | ci-matrix C2 `needs: [test, lint]` (runtime static) | ✅ COMPLIANT — skip is GitHub platform semantics of the verified `needs` |
| 12 | CI Release Gate | Dispatch | ci-matrix C4 + C7 + C10 (runtime static) | ✅ COMPLIANT — dispatch satisfies job `if`, publish step gated `refs/tags/v` only, upload ungated → build+upload, no release |
| 13 | CI Release Gate | PR push | ci-matrix C4 (runtime static) | ✅ COMPLIANT — release job `if` excludes PR refs |
| 14 | Idempotent Release Creation | First run | source inspection + shellcheck (static only) | ⚠️ PARTIAL — create path matches design D3 exactly; live execution unobserved (4.4) |
| 15 | Idempotent Release Creation | Re-run | source inspection (static only) | ⚠️ PARTIAL — skip-if-present logic (`grep -qxF` on existing asset names) matches D3; live re-run unobserved (4.4) |
| 16 | Idempotent Release Creation | Partial upload | source inspection (static only) | ⚠️ PARTIAL — per-asset missing-check implemented; live unobserved (4.4) |
| 17 | Idempotent Release Creation | Failed run recovery | source inspection + grep (static only) — no delete/tag mutation in script | ⚠️ PARTIAL — nothing to retract implemented; live re-run unobserved (4.4) |
| 18 | Asset and Checksums Contract | Complete release | notes harness N6–N12 + unchanged `make release` | ⚠️ PARTIAL — 6 asset names + checksums generator verified; live release inspection + `sha256sum -c` unexecuted (4.4) |
| 19 | Asset and Checksums Contract | Sole generator | diff review (static) | ✅ COMPLIANT — `make release` unchanged and remains sole producer; script only consumes `dist/` |
| 20 | Asset and Checksums Contract | Missing checksums | `go test ./...` internal/selfupdate green + notes N13 warning (runtime) | ✅ COMPLIANT — fail-closed covered by existing Self-Update tests |
| 21 | Curated Release Notes | Publish | notes harness N1–N13 (runtime) | ✅ COMPLIANT |
| 22 | Curated Release Notes | Curated summary | notes harness N1, T1–T3 (runtime) | ✅ COMPLIANT — human tag message used; no auto-generation |
| 23 | Release Job Permissions | Release job | ci-matrix C3 (runtime static) | ✅ COMPLIANT |
| 24 | Release Job Permissions | Other jobs | ci-matrix C1 (runtime static) | ✅ COMPLIANT — top-level `contents: read`; test/lint inherit |
| 25 | Release Documentation | New release | README grep assertions (runtime) | ✅ COMPLIANT |

**Compliance summary**: 17/25 scenarios fully COMPLIANT, 8 PARTIAL (all with passing covering tests — see matrix), 0 UNTESTED, 0 FAILING.
Envelope counts (complete = has a passing covering test): requirements 8/8, scenarios 25/25.
All 8 PARTIAL scenarios share one root cause: task 4.4 (fork rehearsal) unexecuted — see below.

## TDD Compliance (Strict TDD shell variant)

| Check | Result | Details |
|-------|--------|---------|
| TDD Evidence reported | ✅ | apply-progress.md contains the TDD Cycle Evidence table (shell variant, design D6) |
| All tasks have tests | ✅ | Every implemented work unit has a harness; 4.4 is a verification task with no RED/GREEN by design |
| RED confirmed (tests exist) | ✅ | All three harnesses exist in /tmp/opencode and were re-executed this verify; apply recorded RED failures before implementation (guard 0/12, `bash -n` 127, ci 4/8 FAIL) |
| GREEN confirmed (tests pass) | ✅ | Re-ran fresh: guard 11/11, notes 22/22, ci 10/10 — all PASS at runtime |
| Triangulation adequate | ✅ | 11 guard scenarios (dirty ×2, non-main ×2, version ×2, local, remote, outside, dry-run, real-repo), 22 notes assertions, 10 CI assertions |
| Safety Net for modified files | ✅ | 0 Go files modified; `go test ./... -count=1` green in apply and this verify |

**TDD Compliance**: 6/6 checks passed

## Test Layer Distribution

| Layer | Tests | Files | Tools |
|-------|-------|-------|-------|
| Shell runtime harness (make publish guards) | 11 assertions | /tmp/opencode/upp-guard-matrix.sh | bash, git, make |
| Shell runtime harness (notes) | 22 assertions | /tmp/opencode/upp-notes-matrix.sh | bash, git |
| Static structure matrix (ci.yml) | 10 assertions | /tmp/opencode/upp-ci-matrix.py | Python 3 + PyYAML |
| Go safety net (pre-existing suite) | all packages ok | internal/* — unchanged | go test |
| **Total shell assertions** | **43** | **3 harnesses** | |

## Changed File Coverage

Coverage analysis skipped — no shell coverage tool exists, and the change touches no Go files. Static linters (shellcheck, actionlint, go vet) all clean on the changed surfaces.

## Assertion Quality

Audited all three harnesses against the banned-pattern list: no tautologies, no orphan empty checks (S9's empty-tag assertion has companion non-empty tag scenarios S6/S7), no type-only assertions, no ghost loops (the dist-fixture loop is setup, not assertion), no smoke-only tests, no mocks at all, and every assertion executes production code (`make publish`, the script's `notes` path, or a parse of the real ci.yml). Error-message and YAML-contract assertions verify behavior, not implementation trivia.

**Assertion quality**: ✅ All assertions verify real behavior (0 CRITICAL, 0 WARNING)

## Quality Metrics

**Linter (shellcheck -S warning)**: ✅ No findings on scripts/publish-release.sh
**Workflow lint (actionlint)**: ✅ Clean on .github/workflows/ci.yml
**Go vet**: ✅ No findings
**gofmt**: ➖ N/A — no Go files in scope

## Correctness (Static Evidence)

| Requirement | Status | Notes |
|------------|--------|-------|
| Publish Guards | ✅ Implemented | Guards 1–5 in exact D1 order, `ERROR:` messages, `exit 1`, positive checks `|| true` (Makefile:127-135) |
| Tag Semantics | ✅ Implemented | `git tag -a "$(VERSION)"`, explicit `refs/tags/$(VERSION)` refspec, never `--tags`; CI contains no tag mutation |
| CI Release Gate | ✅ Implemented | `needs: [test, lint]`, job `if` tags-v OR dispatch, publish step gated `refs/tags/v` with `GH_TOKEN` |
| Idempotent Release Creation | ✅ Implemented | create-or-upload with per-asset skip-if-present; never deletes releases or mutates tags |
| Asset and Checksums Contract | ✅ Implemented | `make release` untouched (sole generator, sha256sum format); script uploads exactly `dist/upp-*.tar.gz`, `dist/upp-*.zip`, `dist/checksums.txt` |
| Curated Release Notes | ✅ Implemented | Title + `## What's new` + `## Assets` + checksums warning from annotated tag message |
| Release Job Permissions | ✅ Implemented | Job-scoped `contents: write` on release only; top-level `contents: read`; auth via automatic `github.token` |
| Release Documentation | ✅ Implemented | README: `make publish` flow with guards + retraction guidance; manual attach removed; `make release` kept |

## Coherence (Design)

| Decision | Followed? | Notes |
|----------|-----------|-------|
| D1 single guarded publish target | ✅ Yes | Guards 1–5, order, messages, `|| true`, tag -a, explicit refspec, optional `gh run watch` — all exact |
| D2 CI release job (gate step, not job) | ✅ Yes | ci.yml matches the design YAML block, including `fetch-depth: 0` |
| D3 idempotent create-or-upload | ✅ Yes | Script `publish()` implements D3 verbatim; notes passed via `--notes-file` |
| D4 curated notes from tag message | ✅ Yes | `notes()` matches D4; two apply-time refinements (title line in output; `upp vX.Y.Z` prefix normalization) are within design intent, noted in apply-progress deviations |
| D5 job-scoped permissions | ✅ Yes | `contents: write` on release job only; no secrets/PATs |
| D6 shell-only verification | ✅ Yes | D6.1–D6.4 executed and passing; D6.5 fork rehearsal UNEXECUTED (see below) |
| D7 README | ✅ Yes | Manual attach removed; make publish flow + make release kept; no install.sh changes |

## Task 4.4 Fork Rehearsal — UNEXECUTED (honest record)

Read-only check performed (no fork or external resource created, per session constraints):

- `gh api repos/JhnFrankz/upp/forks` → empty list (exit 0)
- `gh repo view JhnFrankz/upp --json forkCount` → 0

No fork exists, so the rehearsal cannot run without creating one (out of scope). **This evidence path is unexecuted and does not count as a pass** for the 8 scenarios that depend on live GitHub execution. Exact procedure for when a fork is available:

1. Fork `JhnFrankz/upp` (`gh repo fork`), then in the fork: `make publish VERSION=v0.0.99` on a clean main tree → annotated tag created + pushed.
2. Watch fork CI on the tag push: test + lint green → release job runs `make release`, uploads artifacts, and `scripts/publish-release.sh publish` creates the GitHub Release `upp v0.0.99 — <summary>` with 6 assets.
3. Verify assets: run `make release` locally on the same commit and `sha256sum -c` the release's `checksums.txt` against local outputs.
4. Idempotency: `gh run rerun` the release job → completes with no asset re-uploaded and no release modification.
5. Dispatch check: trigger `workflow_dispatch` → artifacts built and uploaded, no release created.

## Issues Found

**CRITICAL**: None.

**WARNING**:
1. Task 4.4 (fork rehearsal) is UNEXECUTED: forkCount is 0 and creating a fork is out of scope for this session. 8/25 scenarios are therefore PARTIAL (fully implemented, structurally verified, live GitHub execution unproven). The first real `make publish` (v0.2.0) will be the de facto live rehearsal; until then the idempotency and release-creation paths carry residual integration risk.

**SUGGESTION**:
1. Run the 4.4 procedure on a fork before the first real publish, or explicitly accept the first real publish as the live rehearsal.
2. The orchestrator's launch prompt stated 23 scenarios; the retrieved spec.md contains 25 scenario rows. Envelope uses the authoritative spec count (25).

## Verdict

**PASS WITH WARNINGS** — every executable check passed at runtime (43/43 shell assertions across 3 harnesses, make -n exact sequence, shellcheck/actionlint/go vet clean, Go suite green); the single gap is the deferred live GitHub fork rehearsal (task 4.4), recorded honestly as UNEXECUTED and reflected as 8 PARTIAL scenarios.
