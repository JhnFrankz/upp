# Proposal: Repository Hygiene, Tooling Configuration & Cross-Platform Parity

**Change ID**: `2026-08-18-upp-repo-hygiene`  
**Date**: 2026-08-18  
**Domain**: Repository Hygiene, Cross-Platform Stability, Developer Experience, CI/CD Parity  
**Status**: Proposed  

---

## 1. Intent

### 1.1 Problem Statement
`upp` is a compiled cross-platform CLI tool targeting Linux, macOS, and Windows. While the Go source code is robust and test coverage is comprehensive, the repository lacks essential repository-level hygiene configurations and cross-platform guardrails:
1. **Incomplete `.gitignore`**: Only contains 6 lines, omitting Windows binaries (`upp.exe`, `*.exe`), Go test and profiling artifacts (`*.test`, `*.prof`, `*.pprof`, `cpu.out`, `mem.out`, `trace.out`, `coverage.html`), IDE configurations (`.idea/`, `.vscode/`, `*.code-workspace`), and OS clutter (`.DS_Store`, `Thumbs.db`).
2. **Missing `.gitattributes`**: No line-ending normalization exists. On Windows systems where `core.autocrlf` or editor defaults apply CRLF, shell scripts (`scripts/*.sh`) and Go source files can be checked out or committed with CRLF line endings, causing instant script execution failure (`\r: command not found`) in POSIX/WSL/CI environments and triggering shellcheck warnings.
3. **Missing `.editorconfig`**: Contributor environments lack a shared formatting and whitespace baseline across editors (Go requires tab indentation via `gofmt`; YAML, TOML, JSON, and Shell scripts require 2 spaces).
4. **Missing `.golangci.yml` (Linter Parity Drift)**: CI executes `golangci-lint` without a committed repository configuration file, relying on defaults that may drift between local development environments, CI action runs, and future linter releases.
5. **Specification Gap**: Existing OpenSpec specifications (`openspec/specs/ci-workflow/spec.md`) specify shellcheck and actionlint but do not formalize repository hygiene or Go linter configuration contracts.

### 1.2 Technical Necessity & Value
- **Cross-Platform Execution Stability**: Enforces LF line endings on all text assets and shell scripts via Git attributes, eliminating CRLF-related runtime failures on POSIX systems.
- **Hermetic Git Working Trees**: Comprehensive `.gitignore` prevents inadvertent commits of binaries, test coverage files, profiler dumps, OS artifacts, and editor metadata.
- **Uniform Contributor DX**: `.editorconfig` provides out-of-the-box editor alignment across IDEs (VS Code, GoLand, Vim, Cursor, etc.).
- **Strict Local-to-CI Parity**: A committed `.golangci.yml` locks down active linters, timeout bounds, and settings, guaranteeing that `make lint` locally and `golangci-lint` in GitHub Actions produce deterministic, reproducible results.

---

## 2. Scope

### 2.1 In-Scope
- **`.gitignore` Expansion**:
  - Binaries and compiled outputs (`upp`, `upp.exe`, `dist/`, `bin/`, `*.exe`, `*.exe~`, `*.dll`, `*.so`, `*.dylib`).
  - Go test and profiling outputs (`*.test`, `*.out`, `*.prof`, `*.pprof`, `coverage.out`, `coverage.html`, `coverage.txt`, `cpu.out`, `mem.out`, `trace.out`).
  - IDE, editor, and swap files (`.idea/`, `*.iml`, `*.iws`, `.vscode/`, `*.code-workspace`, `*.swp`, `*.swo`, `*~`, `#*#`, `.#*`).
  - OS metadata (`.DS_Store`, `.DS_Store?`, `._*`, `.Spotlight-V100`, `.Traces`, `Thumbs.db`, `ehthumbs.db`, `Desktop.ini`).
  - Local AI & tool runtime caches (`.atl/`, `.codegraph/`, `.gemini/`).
- **`.gitattributes` Creation**:
  - Global default text normalization (`* text=auto eol=lf`).
  - Strict LF enforcement for shell scripts (`*.sh text eol=lf`).
  - Text normalization for Go, config, and doc formats (`*.go`, `*.yml`, `*.yaml`, `*.toml`, `*.json`, `*.md`).
  - Explicit binary designations (`*.png`, `*.jpg`, `*.tar.gz`, `*.zip`, `*.exe`).
- **`.editorconfig` Creation**:
  - Global UTF-8, LF line endings, trailing newline, whitespace trimming.
  - Go and Makefile indentation using tabs (indent_size 4).
  - YAML, TOML, JSON, and Shell script indentation using 2 spaces.
  - Markdown whitespace exception (`trim_trailing_whitespace = false`) to preserve intentional line breaks.
- **`.golangci.yml` Creation**:
  - Execution configuration: 5m timeout, readonly modules, test files included.
  - Enabled linters: `govet`, `errcheck`, `staticcheck`, `unused`, `gosimple`, `ineffassign`, `gofmt`, `unconvert`, `misspell`.
  - Specific settings: `errcheck.check-type-assertions: true`, `govet.disable: [fieldalignment]`, `misspell.locale: US`.
- **OpenSpec Specification Delta**:
  - Update `openspec/specs/ci-workflow/spec.md` with:
    - `Requirement: Go Linter Configuration and Local Parity`
    - `Requirement: Repository Hygiene and Line Ending Normalization`

### 2.2 Out-of-Scope / Non-Goals
- Adding external multi-language hook frameworks (e.g., Python-based `pre-commit`).
- Enabling aggressive linters that require pervasive refactoring of idiomatic CLI code (e.g., strict `revive` flag rules or `gosec` rule sets flagging test files).
- Modifying core application Go logic or CLI runtime code.
- Modifying the release asset build script logic or GitHub Release publishing flow.

---

## 3. Capabilities & Specification Changes

### 3.1 New Capabilities
None.

### 3.2 Modified Capabilities

#### `ci-workflow` (`openspec/specs/ci-workflow/spec.md`)
- **ADD Requirement: Go Linter Configuration and Local Parity**:
  - Repository MUST maintain a root `.golangci.yml` defining the active linter set (`govet`, `errcheck`, `staticcheck`, `unused`, `gosimple`, `ineffassign`, `gofmt`, `unconvert`, `misspell`), timeout limit (5m), and configuration rules.
  - CI `lint` job and local `make lint` MUST both execute against this configuration file to ensure deterministic results.
- **ADD Requirement: Repository Hygiene and Line Ending Normalization**:
  - Repository MUST maintain `.gitignore`, `.gitattributes`, and `.editorconfig` at the root.
  - `.gitattributes` MUST enforce `eol=lf` on all shell scripts (`*.sh`) and text files to prevent cross-platform CRLF corruption.

---

## 4. Technical Approach

### 4.1 Implementation Strategy (Approach 2: Comprehensive Hygiene & Cross-Platform Parity Standard)
Adopt the comprehensive hygiene architecture evaluated in `exploration.md`:

```
repo-root/
├── .editorconfig        # Universal editor formatting & indentation rules
├── .gitattributes       # Git EOL normalization (eol=lf) and binary definitions
├── .gitignore           # Comprehensive ignores (binaries, tests, prof, IDE, OS)
└── .golangci.yml        # Deterministic linter configuration (9 clean linters)
```

### 4.2 File Changes Overview

| File | Change Type | Purpose |
|---|---|---|
| `.gitignore` | Modified | Expand ignore rules to cover OS clutter, IDE dirs, Go test/prof binaries, and Windows `.exe` files. |
| `.gitattributes` | Created | Set `* text=auto eol=lf`, `*.sh text eol=lf`, and binary file types. |
| `.editorconfig` | Created | Configure root editor formatting rules (tabs for Go/Makefiles, 2 spaces for configs/shell). |
| `.golangci.yml` | Created | Configure `golangci-lint` with 9 verified clean linters and specific analyzer settings. |
| `openspec/specs/ci-workflow/spec.md` | Modified | Add formal requirements for `.golangci.yml` parity and repository hygiene invariants. |

### 4.3 Technical Conventions & Configurations

#### `.gitattributes` Configuration
```gitattributes
# Default text handling with LF line endings
* text=auto eol=lf

# Force LF on shell scripts (critical for Linux/CI execution on Windows checkouts)
*.sh text eol=lf

# Go source code
*.go text eol=lf

# Configuration and documentation files
*.yml text eol=lf
*.yaml text eol=lf
*.toml text eol=lf
*.json text eol=lf
*.md text eol=lf

# Binary files
*.png binary
*.jpg binary
*.jpeg binary
*.gif binary
*.ico binary
*.tar.gz binary
*.zip binary
*.exe binary
```

#### `.editorconfig` Configuration
```ini
root = true

[*]
charset = utf-8
end_of_line = lf
insert_final_newline = true
trim_trailing_whitespace = true

[*.go]
indent_style = tab
indent_size = 4

[Makefile]
indent_style = tab
indent_size = 4

[*.{yml,yaml,toml,json}]
indent_style = space
indent_size = 2

[*.sh]
indent_style = space
indent_size = 2

[*.md]
indent_style = space
indent_size = 2
trim_trailing_whitespace = false
```

#### `.golangci.yml` Configuration
```yaml
run:
  timeout: 5m
  modules-download-mode: readonly
  tests: true

linters:
  disable-all: true
  enable:
    - govet
    - errcheck
    - staticcheck
    - unused
    - gosimple
    - ineffassign
    - gofmt
    - unconvert
    - misspell

linters-settings:
  errcheck:
    check-type-assertions: true
    check-blank: false
  govet:
    disable:
      - fieldalignment
  misspell:
    locale: US
  gofmt:
    simplify: true

issues:
  max-issues-per-linter: 0
  max-same-issues: 0
```

---

## 5. Verification Strategy

### 5.1 Verification Commands
All changes will be validated using local gates and toolchains:
1. **Linter Gate**:
   - `golangci-lint run ./...` — must exit 0 with 0 findings across all packages.
   - `make lint` — must execute without errors.
2. **Go Code Quality & Tests**:
   - `go vet ./...` — must pass.
   - `test -z "$(gofmt -s -l .)"` — must pass (all files properly formatted).
   - `go test ./... -count=1 -race` — all unit and race tests must pass.
   - `make build` and `make smoke` — build and smoke tests must pass.
3. **Shell & Workflow Linting**:
   - `shellcheck -S warning scripts/*.sh` — must exit 0.
   - `actionlint .github/workflows/ci.yml` — must exit 0.
4. **Git Attributes & EOL Validation**:
   - `git check-attr -a scripts/install.sh` — must confirm `text: set`, `eol: lf`.
   - `git check-attr -a cmd/upp/main.go` — must confirm `text: set`, `eol: lf`.
   - `file scripts/*.sh` — must report ASCII/UTF-8 text with standard LF line terminators.
5. **Git Ignore Validation**:
   - Verify ignored patterns (e.g. `upp.exe`, `coverage.html`, `.DS_Store`, `.idea/`) via `git check-ignore -v <file>`.

---

## 6. Risks & Tradeoffs

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| **Linter version drift between developer machine and CI action** | Low | Low | `.golangci.yml` uses well-established linter rules supported across `v1.x` and `v2.x` releases. |
| **Markdown trailing whitespace trimming conflict** | Low | Low | Explicitly set `trim_trailing_whitespace = false` for `*.md` in `.editorconfig` to preserve Markdown 2-space line breaks. |
| **Accidental git CRLF conversion on Windows checkouts** | Low | Med | `.gitattributes` explicitly mandates `eol=lf` on `*.sh` and text files, overriding local `core.autocrlf=true`. |

---

## 7. Rollback Plan

If any configuration causes unforeseen developer workflow friction or CI incompatibility:
1. Revert `.gitignore`, `.gitattributes`, `.editorconfig`, `.golangci.yml`, and `openspec/specs/ci-workflow/spec.md`.
2. Git working tree returns to baseline without affecting compiled binary functionality or runtime semantics.

---

## 8. Success Criteria

- [ ] `.gitignore` updated with binaries, profiling artifacts, OS metadata, and IDE directories.
- [ ] `.gitattributes` created with global `eol=lf`, script LF enforcement, and binary declarations.
- [ ] `.editorconfig` created with tab indentation for Go/Makefiles and 2-space indentation for configs/scripts.
- [ ] `.golangci.yml` created with the 9 verified clean linters and zero false positives.
- [ ] `openspec/specs/ci-workflow/spec.md` updated with linter parity and repository hygiene requirements.
- [ ] `golangci-lint run ./...`, `go test ./... -count=1 -race`, `make smoke`, `shellcheck`, and `actionlint` pass cleanly.
