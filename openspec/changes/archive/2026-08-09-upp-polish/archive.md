# Archive Report: upp-polish

## Final Status

**Status**: SUCCESS
**Archived**: 2026-08-09
**Change**: upp-polish — Post-rewrite polish (docs, CI, release pipeline, format, adapter tests)
**Issue tracker**: #1 closed automatically by merge of PR #6 ("Closes #1") — verified via `gh issue view 1` → state CLOSED, no manual close needed.

## Summary

Non-functional change (Capabilities None/None) that closed the post-rewrite hygiene gaps: README rewritten for the Go CLI (no more "Migration from upp.sh" references), GitHub Actions CI (test/lint/release jobs, golangci-lint blocking), `make release`/`make install` targets (assets + checksums, no tag/publish), skill registry refreshed with the 10 sdd-* skills, `openspec/config.yaml` committed, gofmt -s applied to 9 files, and `internal/adapters/official` coverage raised from 60.7% to 95.4% via hermetic table-driven tests behind an exec seam in `helper.go`.

## Final-State Evidence (at close, HEAD main = 12902cf, CI green)

- **Tasks**: 13/13 complete, 5/5 phases (tasks.md in this archive; apply-progress Engram #670).
- **Verification**: PASS per verify-report Engram #676 (0 CRITICAL, 0 blockers). Runtime gates re-confirmed: `go test ./... -count=1 -race` exit 0 (7/7 packages), `go build ./...` exit 0, `gofmt -s -l .` empty, `go vet ./...` clean, coverage 95.4% ≥ 80% threshold.
- **CI**: workflow success on main at HEAD 12902cf (merge PR #6), run 31346426706.
- **Known warnings carried from verify-report #676 (accepted, not defects)**: W1 docker.Update non-current-GOOS branches uncovered (GOOS not mockable, open design question); W2 was "change folder untracked — archive-time commit" and is resolved BY THIS ARCHIVE COMMIT. Suggestions S1/S2 (pnpm recovery-success branch, AdaptersForCurrentPlatform 75%) accepted under package gate.

## Specs Synced

| Domain | Action | Details |
|--------|--------|---------|
| (none) | No delta specs | proposal.md declares Capabilities New: None / Modified: None — no requirements or scenarios changed in the 6 domains. No `specs/` directory exists in the change folder. Main specs (`openspec/specs/*/spec.md`) untouched. |

**Zero delta specs to sync** — change is non-functional by design (docs, CI, tests, tooling).

## Artifacts Created

| Artifact | Location | Status |
|----------|----------|--------|
| proposal.md | `openspec/changes/archive/2026-08-09-upp-polish/proposal.md` | ✅ |
| design.md | `openspec/changes/archive/2026-08-09-upp-polish/design.md` | ✅ |
| tasks.md | `openspec/changes/archive/2026-08-09-upp-polish/tasks.md` | ✅ (13/13 tasks complete, 0 unchecked) |
| verify-report.md | `openspec/changes/archive/2026-08-09-upp-polish/verify-report.md` | ✅ |
| archive.md | this file | ✅ |

### Engram Observations Read (traceability)

- #670 — `sdd/upp-polish/apply-progress` (intermediate snapshot: phases 1-5, 13/13, coverage 95.4%)
- #676 — `sdd/upp-polish/verify-report` (verification PASS, final-state authority for quality gates)

### Commits Created

- Archive commit (this change): `docs: archive upp-polish change (SDD complete)` — commits the previously untracked `openspec/changes/upp-polish/` folder into `openspec/changes/archive/2026-08-09-upp-polish/` (pattern W2 from verify-report #676, resolved here). NOT pushed — orchestrator decides push.

## Archive Verification

- [x] Main specs updated correctly — N/A: zero delta specs (Capabilities None/None), documented above
- [x] Change folder moved to archive (`openspec/changes/archive/2026-08-09-upp-polish/`)
- [x] Archive contains all artifacts (proposal, design, tasks, verify-report)
- [x] Archived tasks.md has no unchecked implementation tasks (0 unchecked)
- [x] Active changes directory no longer has this change
- [x] Verbatim `diff -r` readback output included in result and is empty (no differences) — executed against a pre-move recursive snapshot; `diff -r` exit 0, no output
