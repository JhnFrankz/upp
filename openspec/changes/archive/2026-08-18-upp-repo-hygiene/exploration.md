# Exploration: Repository Hygiene, Tooling Configuration & Cross-Platform Standards

**Change ID**: `2026-08-18-upp-repo-hygiene`  
**Date**: 2026-08-18  
**Domain**: Repository Hygiene, Tooling Configuration, CI/CD Parity, Cross-Platform Standards  
**Status**: Completed  

---

## 1. Context & Problem Statement

`upp` is a compiled cross-platform CLI tool written in Go 1.22+ targeting Linux (amd64, arm64), macOS (amd64, arm64), and Windows (amd64). While the codebase has strong test coverage (table-driven tests across adapters, CLI commands, and self-update) and an active GitHub Actions CI pipeline (`.github/workflows/ci.yml`), several repository-level hygiene and tooling configuration gaps exist:

1. **Incomplete `.gitignore`**:
   - The current `.gitignore` contains only 6 lines (`dist/`, `upp`, `coverage.out`, `.atl/`, `.codegraph/`).
   - It omits Windows executable artifacts (`upp.exe`, `*.exe`, `*.exe~`), Go profiling/test binary artifacts (`*.test`, `*.prof`, `*.pprof`, `cpu.out`, `mem.out`, `trace.out`, `coverage.html`), IDE files (`.idea/`, `.vscode/`, `*.code-workspace`), and OS clutter (`.DS_Store`, `Thumbs.db`).
2. **Missing `.gitattributes`**:
   - No `.gitattributes` exists in the repository root.
   - On Windows environments where `core.autocrlf` may be enabled by default, shell scripts (`scripts/install.sh`, `scripts/smoke-test.sh`, `scripts/publish-release.sh`) or Go files can be checked out or committed with CRLF line endings.
   - CRLF in shell scripts causes immediate execution failures in Linux/WSL/CI (`\r: command not found`) and triggers linter/shellcheck warnings.
3. **Missing `.editorconfig`**:
   - Contributors using various IDEs (VS Code, GoLand, IntelliJ, Vim, Emacs, Cursor) lack an automated baseline for indentation styles, line endings, character encoding, and trailing whitespace.
   - In Go, `gofmt` mandates tab indentation, whereas YAML, TOML, JSON, and Shell scripts require space indentation (2 spaces).
4. **Missing `.golangci.yml` (Local-to-CI Parity Drift)**:
   - CI runs `golangci/golangci-lint-action@v7` (pinned to `version: v2.12.2`), but without a repository `.golangci.yml` configuration file, both CI and local developers run against default linters or arbitrary local environment configs.
   - There is no committed contract defining which linters are active, linter settings, timeout bounds, and issue exclusions.
5. **OpenSpec Alignment**:
   - Current specs in `openspec/specs/` (e.g., `ci-workflow/spec.md`) specify shellcheck and actionlint, but do not formalize repository hygiene, cross-platform EOL normalization, or `.golangci.yml` local-to-CI parity.

---

## 2. Technical Investigation & Current State

### 2.1 File State Analysis

| File | Current Status | Deficiencies Identified |
|---|---|---|
| `.gitignore` | Present (6 lines) | Missing Windows binaries (`upp.exe`, `*.exe`), Go test/prof binaries (`*.test`, `*.prof`), OS metadata (`.DS_Store`, `Thumbs.db`), IDE dirs (`.idea/`, `.vscode/`). |
| `.gitattributes` | Missing | No EOL normalization; risk of CRLF corruption for `scripts/*.sh` on Windows. |
| `.editorconfig` | Missing | No cross-editor formatting baseline for Go (tabs), YAML/TOML/JSON/Shell (2 spaces), UTF-8, and LF. |
| `.golangci.yml` | Missing | CI and local runs rely on default linters without explicit configuration, risking version/linter drift. |
| `openspec/specs/ci-workflow/spec.md` | Present | Lacks explicit specification of `.golangci.yml` contract and repository hygiene requirements. |

### 2.2 Linter & Tooling Verification

Direct evaluation of the codebase against `golangci-lint` linters revealed:
- **Clean Linters (0 findings across entire repo)**:
  - `govet`: Clean (passes `go vet ./...` as required by CI).
  - `errcheck`: Clean (all error returns properly handled or explicitly checked).
  - `staticcheck`: Clean (advanced static analysis passes).
  - `unused`: Clean (no dead constants, variables, or unexported functions).
  - `gosimple`: Clean (idiomatic Go constructs).
  - `ineffassign`: Clean (no redundant variable assignments).
  - `gofmt`: Clean (matches `gofmt -s`).
  - `unconvert`: Clean (no redundant type conversions).
  - `misspell`: Clean (no typo findings in source code or comments).
- **Linters with False Positives / Requiring Specific Exclusions**:
  - `revive` (default): Flags unused parameters in standard Cobra command callbacks `RunE: func(cmd *cobra.Command, args []string) error` and standard HTTP handlers.
  - `gosec` (default): Flags test helper file writes (`0644` vs `0600`) and test subshell commands.

---

## 3. Approaches Evaluated

### Approach 1: Minimalist Additions (Additive Only)
- **Description**: Add bare-bones `.gitignore` additions, a single-line `.gitattributes` (`* text=auto`), a minimal `.editorconfig`, and a basic `.golangci.yml` enabling default linters.
- **Pros**:
  - Smallest possible diff.
  - Low initial effort.
- **Cons**:
  - Incomplete OS/IDE/profiling ignore patterns leave repository vulnerable to accidental commits of IDE folders or test binaries.
  - Single-line `.gitattributes` does not guarantee explicit LF enforcement on shell scripts or binary handling.
  - Does not update OpenSpec documentation to lock down the repository hygiene contract.
- **Verdict**: Rejected.

---

### Approach 2: Comprehensive Repository Hygiene & Cross-Platform Parity Standard (Recommended)
- **Description**:
  1. **`.gitignore`**: Grouped, clean configuration covering:
     - Binaries & build outputs: `upp`, `upp.exe`, `dist/`, `bin/`, `*.exe`, `*.exe~`, `*.dll`, `*.so`, `*.dylib`.
     - Go test & profiling artifacts: `*.test`, `*.out`, `*.prof`, `*.pprof`, `coverage.out`, `coverage.html`, `coverage.txt`, `cpu.out`, `mem.out`, `trace.out`.
     - IDE & editor files: `.idea/`, `*.iml`, `*.iws`, `.vscode/`, `*.code-workspace`, `*.swp`, `*.swo`, `*~`, `#*#`, `.#*`.
     - OS metadata: `.DS_Store`, `.DS_Store?`, `._*`, `.Spotlight-V100`, `.Traces`, `Thumbs.db`, `ehthumbs.db`, `Desktop.ini`.
     - Local runtime state: `.atl/`, `.codegraph/`, `.gemini/`.
  2. **`.gitattributes`**:
     - Global default: `* text=auto eol=lf`
     - Shell scripts: `*.sh text eol=lf` (prevents Windows CRLF execution failures)
     - Go source code: `*.go text eol=lf`
     - Configurations & documentation: `*.yml text eol=lf`, `*.yaml text eol=lf`, `*.toml text eol=lf`, `*.json text eol=lf`, `*.md text eol=lf`
     - Binary assets: `*.png binary`, `*.jpg binary`, `*.tar.gz binary`, `*.zip binary`, `*.exe binary`
  3. **`.editorconfig`**:
     - Global baseline: `root = true`, `charset = utf-8`, `end_of_line = lf`, `insert_final_newline = true`, `trim_trailing_whitespace = true`.
     - Go & Makefiles: `indent_style = tab`, `indent_size = 4`.
     - YAML, TOML, JSON, Shell: `indent_style = space`, `indent_size = 2`.
     - Markdown: `indent_style = space`, `indent_size = 2`, `trim_trailing_whitespace = false` (preserves Markdown 2-space line breaks).
  4. **`.golangci.yml`**:
     - Explicit configuration with `run` (5m timeout, readonly modules, test files included).
     - Enabled linters: `govet`, `errcheck`, `staticcheck`, `unused`, `gosimple`, `ineffassign`, `gofmt`, `unconvert`, `misspell`.
     - Tuned settings: `errcheck.check-type-assertions: true`, `govet.disable: [fieldalignment]`, `misspell.locale: US`.
     - Local-to-CI parity: Guarantees `make lint` and CI `golangci-lint` produce identical outcomes.
  5. **OpenSpec Updates**:
     - Update `openspec/specs/ci-workflow/spec.md` to formalize repository hygiene and linter configuration requirements.
- **Pros**:
  - Closes all cross-platform EOL, IDE clutter, and linter drift vulnerabilities.
  - Zero false positives on current clean codebase.
  - Fully transparent and maintainable.
- **Cons**:
  - Requires updating documentation and configuration across 4 config files.
- **Verdict**: **Recommended**.

---

### Approach 3: Strict Multi-Tool Matrix (pre-commit framework, heavy linter suite)
- **Description**: Introduce Python-based `pre-commit` framework, git hooks, and 30+ golangci-lint linters (including strict revive, gosec, cyclomatic complexity checkers).
- **Pros**:
  - Maximum theoretical static analysis checks.
- **Cons**:
  - Introduces non-Go external runtime dependencies (Python / pip).
  - High friction for contributors on Windows/macOS.
  - Requires numerous code refactors or inline ignore comments for idiomatic Cobra CLI patterns.
- **Verdict**: Rejected as unnecessarily complex and high-friction.

---

## 4. Tradeoff Matrix

| Criterion | Approach 1 (Minimalist) | Approach 2 (Comprehensive Hygiene) | Approach 3 (Strict Multi-Tool) |
|---|---|---|---|
| **Cross-Platform Protection (CRLF/Windows binaries)** | Partial | **Full & Explicit** | Full |
| **Local-to-CI Linter Parity** | Weak (defaults only) | **Strict (explicit .golangci.yml)** | Strict |
| **Developer Friction** | Low | **Zero (native tooling)** | High (external Python deps) |
| **Editor Integration** | Weak | **Complete (.editorconfig)** | Complete |
| **Maintenance Overhead** | Low | **Very Low** | High |
| **OpenSpec Traceability** | None | **Full (RFC 2119 + Scenarios)** | Full |

---

## 5. Specification Impact Analysis

### Affected Specs in `openspec/specs/`:

1. **`openspec/specs/ci-workflow/spec.md`**:
   - Current Section `Requirement: Script and Workflow Static Checks` mentions `shellcheck` and `actionlint`.
   - **Proposed Modification**: Add `Requirement: Go Linter Configuration and Local Parity`:
     - The repository MUST maintain a committed `.golangci.yml` defining active linters (`govet`, `errcheck`, `staticcheck`, `unused`, `gosimple`, `ineffassign`, `gofmt`, `unconvert`, `misspell`), execution timeout, and settings.
     - CI `lint` job MUST execute `golangci-lint` using this committed configuration without ad-hoc CLI flags.
     - `make lint` MUST execute `golangci-lint run ./...` respecting `.golangci.yml`.
   - **Proposed Requirement: Repository Hygiene & Cross-Platform Invariants**:
     - The repository MUST maintain `.gitignore`, `.gitattributes`, and `.editorconfig` at the root.
     - `.gitattributes` MUST enforce LF line endings (`eol=lf`) on all shell scripts (`*.sh`) and text files to guarantee cross-platform execution stability.

2. **Other Specs**:
   - `release-process/spec.md`: Verified — no breaking changes; release packaging benefits from clean `.gitignore` and LF normalization.
   - `command-interface/spec.md`, `config-system/spec.md`, `platform-detection/spec.md`, `security-model/spec.md`, `self-update/spec.md`, `tool-adapter/spec.md`, `ux-patterns/spec.md`: No functional impact.

---

## 6. Implementation Plan & Next Steps

1. **Phase 1: Configuration Definition**:
   - Create comprehensive `.gitignore`.
   - Create `.gitattributes` with `* text=auto eol=lf` and explicit rules for `*.sh`, `*.go`, `*.yml`, `*.toml`, and binary types.
   - Create `.editorconfig` with tabs for Go/Makefile and 2 spaces for YAML/TOML/JSON/Markdown/Shell.
   - Create `.golangci.yml` with validated clean linters.
2. **Phase 2: OpenSpec Delta**:
   - Prepare `proposal.md`, `design.md`, `tasks.md`, and spec delta in `openspec/changes/2026-08-18-upp-repo-hygiene/`.
3. **Phase 3: Verification**:
   - Validate with `go vet ./...`, `gofmt -s -l .`, `golangci-lint run ./...`, `shellcheck scripts/*.sh`, and `actionlint`.
   - Validate git status and EOL behavior across sample files.
