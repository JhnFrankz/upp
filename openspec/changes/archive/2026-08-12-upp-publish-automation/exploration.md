# Exploration: upp-publish-automation

> SDD phase: sdd-explore | Change: `upp-publish-automation` | Date: 2026-08-12
> Persistence: hybrid (OpenSpec + Engram) | Read-only exploration (no files modified beyond this artifact)

## Exploration: `make publish` automation for GitHub releases

### Current State

**Release surface today (Makefile):**

- `VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")` injected via `-X main.version` (Makefile:5-6, `cmd/upp/main.go`). A clean checkout of a `vX.Y.Z` tag yields exactly `vX.Y.Z`; a dirty tree appends `-dirty`.
- `release` target (Makefile:96-124): depends on `build-all`, cross-compiles 5 platforms (linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64), stages each binary as `dist/.stage/upp-{os}-{arch}/upp`, packages `.tar.gz` (unix) / `.zip` (windows), then generates `dist/checksums.txt` with `sha256sum` (fallback `shasum -a 256`) in the format `"<hex>  <name>"` (two spaces). Comment: "no tag, no publish".
- `install` target (Makefile:126-131): installs to `PREFIX` (default `/usr/local/bin`).
- **No `publish` target, no `tag` target, no release-notes generator.** No gh/API usage anywhere in the repo (verified by grep; `gh` is NOT referenced in Makefile, scripts/, or workflows).

**CI (`.github/workflows/ci.yml`):**

- Triggers: push to main, push of `v*` tags, PRs, `workflow_dispatch`. Top-level `permissions: contents: read`.
- `test` job: Go 1.22.x, vet, gofmt gate, unit tests, race tests, build, smoke test. `lint` job: golangci-lint pinned v1.60.3.
- `release` job (`if: startsWith(github.ref, 'refs/tags/v') || github.event_name == 'workflow_dispatch'`): runs `make release`, uploads `dist/**` as artifact `upp-dist`. It does NOT create a GitHub Release. File header comment: "publishing a GitHub Release (tag + assets) is a manual, user-triggered step (see README)". The job is independent (no `needs:` on test/lint).

**How v0.1.1 was actually published (evidence: GitHub API for tag v0.1.1):**

- Release id 369443551, published 2026-08-12 18:20 UTC by JhnFrankz, `target_commitish: main`, name `"upp v0.1.1 — self-update + opt-in update detection"`, not a prerelease.
- 6 assets: 4 `upp-{os}-{arch}.tar.gz`, `upp-windows-amd64.zip`, and `checksums.txt` (446 bytes, uploaded last at 18:20:40 — uploads staggered 18:20:21→18:20:40, consistent with a sequential manual upload, e.g. `gh release upload` or browser).
- **checksums.txt was produced by `make release`** (its format is exactly the one `verifyChecksum` parses; the asset set matches the `release` target output 1:1) — but attached to the release by hand. There is no script, workflow step, or commit in the repo that performs the publish. Release notes follow a repo-local convention: `## What's new` bullets, `## Assets` section, and a warning that releases must keep shipping `checksums.txt`.
- Follow-up recorded in the archived change `2026-08-12-upp-versioning-auto-update` (archive-report.md, design.md): "`make publish` automation (releases still manual; self-update fails closed without checksums.txt)".

**Self-update consumption contract (internal/selfupdate):**

- `client.go`: resolves latest via `https://api.github.com/repos/JhnFrankz/upp/releases/latest` (only `tag_name` is read), downloads `upp-{os}-{arch}.tar.gz` **and `checksums.txt` from the SAME release** via `https://github.com/JhnFrankz/upp/releases/download/{tag}/` with an HTTPS-only redirect policy (fail-closed off-HTTPS). Owner/repo are hard-coded constants.
- `assets.go`: maps platform → `upp-{os}-{arch}.tar.gz` (darwin/linux + amd64/arm64 only; windows fails closed).
- `update.go`: sha256 verified against the `checksums.txt` entry — a missing, malformed, or mismatched entry → `ErrChecksumMismatch`, binary untouched. Extraction reads ONLY the `upp-{os}-{arch}/upp` entry (matches the Makefile stage layout); absolute paths, `..` traversal, symlinks/hardlinks abort.
- `version.go`: strict grammar — `dev`, `vX.Y.Z`, `vX.Y.Z-N-gHASH[-dirty]`. Dev/dirty builds refuse to update (`ErrDevelopmentBuild`, no network). A release tag that is not a clean `vX.Y.Z` would fail to parse.

**install.sh:**

- `VERSION=latest` → api.github.com latest; downloads from `github.com/.../releases/download/{tag}`, installs to `INSTALL_DIR` (default `/usr/local/bin`; note: `PREFIX` is the Makefile `install` var, `INSTALL_DIR` is install.sh's). Checksum verification is warn-and-skip (weaker than self-update's fail-closed).

**Environment:** `gh` CLI 2.97.0 available locally; remote origin `https://github.com/JhnFrankz/upp.git`; repo has only one workflow, no issue templates, no label conventions.

### Affected Areas

- `Makefile` — add `publish` (and likely `publish-tag`) targets; keep `release` unchanged (CI reuses it).
- `.github/workflows/ci.yml` — extend `release` job to create the GitHub Release; needs `permissions: contents: write` (job-scoped), `needs: [test, lint]`, and publish gated strictly on tag push (NOT `workflow_dispatch`, which has no clean tag).
- `README.md` — "Release" section (lines ~254-263) documents manual publishing and must be updated.
- `scripts/` — optional new script(s) for release-notes generation / publish helpers, following the existing bash-scripts convention (`scripts/install.sh`, `scripts/smoke-test.sh`).
- `internal/selfupdate` — NOT affected. The asset naming, archive layout (`upp-{os}-{arch}/upp`), and checksums format contract are already satisfied by `make release`; automation must preserve that output, not change the consumer.
- OpenSpec specs — publishing is currently undocumented as a requirement domain; a new spec domain (e.g. `release-process`) is likely for the spec phase.

### Approaches

1. **Local `make publish` shelling to `gh`** — one target: clean-tree + version-format guards, `git tag -a`, `git push --tags`, wait for/derive CI artifacts (or build locally), `gh release create --title "upp $V — ..." dist/* dist/checksums.txt`.
   - Pros: no CI changes; gh CLI already present; quick to build; human in the loop for notes; reuses existing `release` output.
   - Cons: local builds risk `-dirty` binaries (self-update refuses them) unless strictly guarded; binary reproducibility depends on the local machine (v0.1.1 assets are documented as CI-built); two ways to publish (drift risk); downloading the CI artifact adds a wait-loop.
   - Effort: Low.

2. **GitHub Actions release workflow triggered by tag push** — extend the existing `release` job: `needs: [test, lint]`, `permissions: contents: write` (job-scoped), `make release`, then `gh release create "$TAG" dist/* --generate-notes` (or `--notes-file`), idempotent create-or-upload.
   - Pros: hermetic, CI-built (non-dirty) assets; single source of truth; matches the existing "release job builds assets on tags" design (only the manual attach step disappears); no local tooling beyond `git tag && git push`.
   - Cons: tag creation remains manual (or via a small local target); needs workflow edit + token-permission escalation (mitigated by job scoping); notes are auto-generated unless curated; a tag pushed before CI passes still triggers a run — `needs:` gates publishing but not the tag's existence.
   - Effort: Medium.

3. **Hybrid: `make publish` = tag + push; CI does build + checksums + release** — Makefile target performs guards (clean tree, `vX.Y.Z` format, tag absent, on main), creates/pushes the tag; CI approach (2) does everything else.
   - Pros: one-command UX with zero local binary builds (no dirty-build risk at all); CI-built assets; minimal local deps (git only); consistent with the Makefile-as-developer-interface convention.
   - Cons: two components to maintain (Makefile + workflow); feedback loop — user must wait for CI to publish (mitigable with `gh run watch`); guards duplicated between Makefile and workflow.
   - Effort: Medium.

**GoReleaser vs hand-rolled (cross-cutting):** GoReleaser would replace `make release` end-to-end (build, archive, checksums, changelog, release creation). Pros: battle-tested, changelog/SBOM extras. Cons: new external dependency; default archive layout puts the binary at archive root (no `upp-{os}-{arch}/` directory), which would BREAK `selfupdate.extract` (`binarySuffix = "/upp"`) unless `wrap_in_directory`/`name_template` are configured to reproduce the current layout — otherwise the self-update contract must be re-specified; strict-TDD project would need config fixtures/tests for a third-party tool's behavior. For a 5-platform matrix already handled by 25 Makefile lines, hand-rolled is the lower-risk fit; GoReleaser is a defensible later refactor.

### Recommendation

Approach 3 (hybrid), implemented as: (1) `make publish` (guards: clean tree, current branch main, VERSION parses as clean `vX.Y.Z`, tag does not exist; then `git tag -a` + `git push` origin tag; optionally `gh run watch`); (2) CI `release` job hardened: `needs: [test, lint]`, job-scoped `permissions: contents: write`, publish ONLY on `startsWith(github.ref, 'refs/tags/v')` (keep `workflow_dispatch` build-only), `make release`, then `gh release create "$TAG" dist/* --notes-file ...` with create-or-upload idempotency (protect against re-runs), keeping the repo's release-note convention (name `upp vX.Y.Z — <summary>`, `## What's new` / `## Assets` / checksums warning). This preserves the self-update contract exactly (same `make release` output, checksums.txt included) while removing the manual attach step — the single failure mode that breaks self-update today.

### Risks

- `contents: write` permission escalation in the workflow — mitigated by scoping it to the `release` job only (top-level stays `contents: read`).
- A tag can be pushed before CI validates main — publishing must be gated by `needs: [test, lint]`; tag deletion/retraction remains a manual recovery path.
- `workflow_dispatch` has no clean tag: publishing on dispatch would produce releases with `git describe` versions (`v0.1.1-N-gHASH`) that fail the self-update version grammar — dispatch must stay build-only.
- Local `-dirty` builds break self-update (`ErrDevelopmentBuild`) — eliminated by building only in CI (hybrid); the Makefile guard is the second line of defense.
- Non-idempotent `gh release create` (fails if the tag's release already exists) — needs create-or-upload semantics and a documented retry path.
- `checksums.txt` must list every uploaded asset in sha256sum `"<hex>  <name>"` format — a missed asset = fail-closed update for that platform; keep `make release` as the single asset+checksum generator (no parallel generators).
- No release automation tests exist; strict TDD applies to any new Go code, but the Makefile/CI surface can only be tested via shell checks and manual verification — verification strategy must be explicit in the spec phase.

### Ready for Proposal

Yes — proceed to `sdd-propose`. The orchestrator should tell the user: publishing today is a manual `gh release` step; the recommended fix is a hybrid `make publish` (tag+push) with CI completing build, checksums, and release creation on tag push, reusing the existing `make release` output verbatim so the self-update contract stays unchanged.
