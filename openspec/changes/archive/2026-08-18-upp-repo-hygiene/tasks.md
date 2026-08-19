# Tasks: upp-repo-hygiene — Repository Hygiene, Tooling Configuration & Cross-Platform Parity

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~190 total (4 created: `.gitattributes` ~30, `.editorconfig` ~25, `.golangci.yml` ~35, `spec.md` ~65; 2 modified: `.gitignore` ~30, `Makefile` ~5) |
| 400-line budget risk | Low |
| Chained PRs recommended | No |
| Suggested split | Single PR (atomic repo hygiene & tooling configuration standard) |
| Delivery strategy | single-pr |
| Chain strategy | N/A |
| Decision needed before apply | No |

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|----------------------|-----------------|-------------------|
| 1 | Git Attributes, Git Ignore & EditorConfig | PR 1 | `git check-attr -a scripts/install.sh && git check-ignore -v upp.exe .DS_Store` | N/A — Git & editor declarative config | revert `.gitattributes`, `.gitignore`, `.editorconfig` |
| 2 | Go Linter Configuration & Makefile Parity | PR 1 | `golangci-lint run ./... && make lint` | N/A — static analysis tooling | revert `.golangci.yml`, `Makefile` |
| 3 | Living Spec Synchronization & Quality Gates | PR 1 | `go test ./... -count=1 -race && make smoke && actionlint .github/workflows/ci.yml` | `bash scripts/smoke-test.sh --skip-build` | revert `openspec/specs/ci-workflow/spec.md` |

---

## Phase 1: Git Attributes, Git Ignore & EditorConfig (Work Unit 1)

- [x] 1.1 Git Attributes & Line Ending Normalization (`.gitattributes`)
  - **PRE-CHECK / RED**: Run `git check-attr -a scripts/install.sh cmd/upp/main.go` and verify `eol` and `text` are `unspecified`.
  - **IMPLEMENTATION / GREEN**: Create `.gitattributes` at repository root defining:
    - Global default text normalization: `* text=auto eol=lf`
    - Shell scripts LF enforcement: `*.sh text eol=lf`
    - Source and configuration files: `*.go text eol=lf`, `*.yml text eol=lf`, `*.yaml text eol=lf`, `*.toml text eol=lf`, `*.json text eol=lf`, `*.md text eol=lf`
    - Explicit binary designations: `*.png binary`, `*.jpg binary`, `*.jpeg binary`, `*.gif binary`, `*.ico binary`, `*.tar.gz binary`, `*.zip binary`, `*.exe binary`
  - **VERIFY**: Run `git check-attr -a scripts/install.sh scripts/smoke-test.sh cmd/upp/main.go dist/upp.exe` to verify `eol: lf` on text/scripts and `binary: set` on binaries. Run `file scripts/*.sh` to confirm LF line terminators.

- [x] 1.2 Git Ignore Rules Expansion (`.gitignore`)
  - **PRE-CHECK / RED**: Run `git check-ignore -v upp.exe coverage.html .idea/workspace.xml .DS_Store .gemini/session` and verify exit code 1 (unignored).
  - **IMPLEMENTATION / GREEN**: Expand root `.gitignore` into 5 logical categories:
    - Binaries & Build Outputs (`upp`, `upp.exe`, `dist/`, `bin/`, `*.exe`, `*.exe~`, `*.dll`, `*.so`, `*.dylib`)
    - Go Test, Coverage & Profiling artifacts (`*.test`, `*.out`, `*.prof`, `*.pprof`, `coverage.out`, `coverage.html`, `coverage.txt`, `cpu.out`, `mem.out`, `trace.out`)
    - IDE & Editor files (`.idea/`, `*.iml`, `*.iws`, `.vscode/`, `*.code-workspace`, `*.swp`, `*.swo`, `*~`, `#*#`, `.#*`)
    - OS metadata (`.DS_Store`, `.DS_Store?`, `._*`, `.Spotlight-V100`, `.Traces`, `Thumbs.db`, `ehthumbs.db`, `Desktop.ini`)
    - Local AI & Tool runtime state (`.atl/`, `.codegraph/`, `.gemini/`)
  - **VERIFY**: Run `git check-ignore -v upp upp.exe dist/test bin/app.exe app.dll pkg.test coverage.html cpu.out .idea/ .vscode/ .DS_Store Thumbs.db .atl/ .codegraph/ .gemini/` and confirm all match intended ignore lines.

- [x] 1.3 Universal Editor Configuration (`.editorconfig`)
  - **PRE-CHECK / RED**: Verify `.editorconfig` does not exist at root (`test ! -f .editorconfig`).
  - **IMPLEMENTATION / GREEN**: Create root `.editorconfig` defining:
    - Global defaults: `root = true`, `charset = utf-8`, `end_of_line = lf`, `insert_final_newline = true`, `trim_trailing_whitespace = true`
    - Go source & Makefiles: `[*.go]` and `[Makefile]` with `indent_style = tab` and `indent_size = 4`
    - Configs & Shell scripts: `[*.{yml,yaml,toml,json}]` and `[*.sh]` with `indent_style = space` and `indent_size = 2`
    - Markdown docs: `[*.md]` with `indent_style = space`, `indent_size = 2`, and `trim_trailing_whitespace = false` (preserving two-space trailing line breaks)
  - **VERIFY**: Validate `.editorconfig` formatting syntax and rules against all specified filetypes.

---

## Phase 2: Go Linter Configuration & Local Parity (Work Unit 2)

- [x] 2.1 Canonical Go Linter Configuration (`.golangci.yml`)
  - **PRE-CHECK / RED**: Verify `.golangci.yml` does not exist at root (`test ! -f .golangci.yml`).
  - **IMPLEMENTATION / GREEN**: Create canonical `.golangci.yml` at repository root:
    - `run`: `timeout: 5m`, `modules-download-mode: readonly`, `tests: true`
    - `linters`: `disable-all: true`, enabling 9 verified clean linters (`govet`, `errcheck`, `staticcheck`, `unused`, `gosimple`, `ineffassign`, `gofmt`, `unconvert`, `misspell`)
    - `linters-settings`: `errcheck.check-type-assertions: true`, `errcheck.check-blank: false`, `govet.disable: [fieldalignment]`, `misspell.locale: US`, `gofmt.simplify: true`
    - `issues`: `max-issues-per-linter: 0`, `max-same-issues: 0`
  - **VERIFY**: Run `golangci-lint run ./...` (or `go run github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.5 run ./...`). Must complete in <5m and report 0 issues.

- [x] 2.2 Makefile Linter Target Parity (`Makefile`)
  - **PRE-CHECK**: Inspect `Makefile` `lint` target to confirm alignment with `.golangci.yml` discovery.
  - **IMPLEMENTATION / GREEN**: Ensure `Makefile` `lint` target runs `golangci-lint run ./...` with fallback to `$(MAKE) vet` when `golangci-lint` binary is missing.
  - **VERIFY**: Run `make lint` locally; confirm it executes cleanly without error.

---

## Phase 3: OpenSpec Synchronization & Quality Gates (Work Unit 3)

- [x] 3.1 OpenSpec Living Specification Update (`openspec/specs/ci-workflow/spec.md`)
  - **PRE-CHECK / RED**: Verify `openspec/specs/ci-workflow/spec.md` lacks requirements for `Go Linter Configuration and Local Parity` and `Repository Hygiene and Line Ending Normalization`.
  - **IMPLEMENTATION / GREEN**: Update `openspec/specs/ci-workflow/spec.md` with the specification delta from `openspec/changes/2026-08-18-upp-repo-hygiene/specs/ci-workflow/spec.md`:
    - `Requirement: Go Linter Configuration and Local Parity` (5m timeout, readonly modules, test inclusion, 9 enabled linters, type assertion checking, unlimited issue reporting)
    - `Requirement: Repository Hygiene and Line Ending Normalization` (`.gitignore` 5 categories, `.gitattributes` global LF/script LF/binaries, `.editorconfig` tab vs. 2-space indentation and Markdown whitespace preservation)
  - **VERIFY**: Confirm all requirement definitions, invariants, and scenario tables match between delta and living spec.

- [x] 3.2 Full Gate Verification Suite
  - **VERIFY**: Execute complete CI & quality gate verification:
    1. **Go Linter Gate**: `golangci-lint run ./...` and `make lint` pass with 0 findings.
    2. **Go Quality & Formatting Gate**: `go vet ./...` passes; `test -z "$(gofmt -s -l .)"` passes.
    3. **Go Test & Race Gate**: `go test ./... -count=1 -race` passes all tests cleanly.
    4. **Build & Smoke Test Gate**: `make build` and `make smoke` pass cleanly.
    5. **Shell & Workflow Linting Gate**: `shellcheck -S warning scripts/*.sh` and `actionlint .github/workflows/ci.yml` pass.
    6. **Git Hygiene & EOL Gate**: `git check-attr -a scripts/*.sh cmd/upp/main.go dist/upp.exe` and `git check-ignore -v upp.exe coverage.html .DS_Store .idea/` pass cleanly.
