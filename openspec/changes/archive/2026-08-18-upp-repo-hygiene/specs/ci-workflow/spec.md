# Delta for ci-workflow

## ADDED Requirements

### Requirement: Go Linter Configuration and Local Parity

The repository MUST maintain a canonical `.golangci.yml` configuration file at the repository root. All Go linting in CI and local developer environments (`make lint` or direct `golangci-lint run`) MUST execute against this configuration to guarantee deterministic, reproducible results.

Configuration invariants:
- **Execution bounds**: `run.timeout` MUST be set to `5m`. `run.modules-download-mode` MUST be set to `readonly`. `run.tests` MUST be set to `true` (ensuring test code is checked).
- **Linter set**: `linters.disable-all` MUST be set to `true` with an explicit list of enabled linters:
  - `govet` (standard Go vet analyzers)
  - `errcheck` (unchecked error returns and type assertions)
  - `staticcheck` (advanced Go static analysis)
  - `unused` (unused constants, variables, functions, and types)
  - `gosimple` (simplification suggestions)
  - `ineffassign` (ineffectual variable assignments)
  - `gofmt` (formatting and simplification checks)
  - `unconvert` (unnecessary type conversions)
  - `misspell` (typo and spelling detection in comments and strings)
- **Linter settings**:
  - `errcheck.check-type-assertions`: MUST be `true`.
  - `govet.disable`: MUST include `fieldalignment` to avoid non-critical struct layout churn.
  - `misspell.locale`: MUST be `US`.
  - `gofmt.simplify`: MUST be `true`.
- **Issue reporting**: `issues.max-issues-per-linter` and `issues.max-same-issues` MUST be `0` (unlimited reporting).

Any linter violation MUST fail the CI `lint` job and local `make lint`.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| CI and local parity | `.golangci.yml` exists at root | `golangci-lint run ./...` executed in CI or locally | Runner uses `.golangci.yml`, enforces 5m timeout, scans production and test code, and checks 9 enabled analyzers |
| Unchecked type assertion | Go code contains unchecked type assertion `val := x.(string)` | Linter executes | `errcheck` flags unchecked assertion; linter exits non-zero |
| Unnecessary type conversion | Go code contains redundant type conversion `int(x)` where `x` is `int` | Linter executes | `unconvert` flags redundant conversion; linter exits non-zero |
| Spelling typo in comments | Go source or comment contains spelling mistake (e.g. `recieve`) | Linter executes | `misspell` flags typo under US locale; linter exits non-zero |
| Ineffectual assignment | Go code assigns a value to a variable that is immediately overwritten | Linter executes | `ineffassign` flags ineffectual assignment; linter exits non-zero |
| Clean codebase | All Go source files adhere to formatting, typing, and safety standards | Linter executes | Linter exits code 0 with 0 findings reported |

### Requirement: Repository Hygiene and Line Ending Normalization

The repository MUST maintain canonical `.gitignore`, `.gitattributes`, and `.editorconfig` files at the repository root to enforce repository cleanliness, prevent cross-platform CRLF corruption, and provide consistent editor formatting.

Hygiene invariants:
- **Git ignore (`.gitignore`)**:
  - Build & binaries: MUST ignore compiled outputs (`upp`, `upp.exe`, `dist/`, `bin/`, `*.exe`, `*.exe~`, `*.dll`, `*.so`, `*.dylib`).
  - Test & profiling: MUST ignore test binaries (`*.test`), test outputs (`*.out`, `coverage.out`, `coverage.html`, `coverage.txt`), and profiler dumps (`*.prof`, `*.pprof`, `cpu.out`, `mem.out`, `trace.out`).
  - IDE & editor: MUST ignore JetBrains (`.idea/`, `*.iml`, `*.iws`), VS Code (`.vscode/`, `*.code-workspace`), and swap/backup files (`*.swp`, `*.swo`, `*~`, `#*#`, `.#*`).
  - OS metadata: MUST ignore macOS (`.DS_Store`, `.DS_Store?`, `._*`, `.Spotlight-V100`, `.Traces`) and Windows (`Thumbs.db`, `ehthumbs.db`, `Desktop.ini`).
  - Tool/Agent caches: MUST ignore local caches (`.atl/`, `.codegraph/`, `.gemini/`).
- **Line ending normalization (`.gitattributes`)**:
  - Default text: MUST enforce `* text=auto eol=lf`.
  - Shell scripts: MUST explicitly enforce `*.sh text eol=lf` to ensure POSIX execution on checkouts across all operating systems.
  - Source & configs: MUST enforce `*.go text eol=lf`, `*.yml text eol=lf`, `*.yaml text eol=lf`, `*.toml text eol=lf`, `*.json text eol=lf`, `*.md text eol=lf`.
  - Binaries: MUST explicitly mark binary file patterns (`*.png`, `*.jpg`, `*.jpeg`, `*.gif`, `*.ico`, `*.tar.gz`, `*.zip`, `*.exe`).
- **Editor configuration (`.editorconfig`)**:
  - Global defaults: MUST declare `root = true`, `charset = utf-8`, `end_of_line = lf`, `insert_final_newline = true`, `trim_trailing_whitespace = true`.
  - Go & Makefiles: MUST set `indent_style = tab` and `indent_size = 4` for `[*.go]` and `[Makefile]`.
  - Configs & Shell scripts: MUST set `indent_style = space` and `indent_size = 2` for `[*.{yml,yaml,toml,json}]` and `[*.sh]`.
  - Markdown: MUST set `indent_style = space`, `indent_size = 2`, and `trim_trailing_whitespace = false` (preserving intentional two-space trailing line breaks).

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Cross-platform script checkout | Git repository cloned on Windows system with `core.autocrlf=true` | `scripts/install.sh` or `*.sh` checked out | `.gitattributes` mandates LF; script maintains LF line endings, preventing `\r: command not found` runtime failure |
| Binary and test artifact ignore | Developer runs `make build` and `go test -coverprofile=coverage.out` | `git status` inspected | `upp`, `upp.exe`, `coverage.out`, and profiling outputs are ignored; working tree remains clean |
| OS and IDE metadata ignore | macOS Finder creates `.DS_Store` or editor creates `.vscode/` / `.idea/` | `git status` inspected | Files match `.gitignore` and are not tracked by Git |
| Go source indentation | Contributor edits a Go source file or Makefile in an EditorConfig-enabled IDE | Editor saves file | Tab indentation (size 4), LF line endings, and trimmed trailing whitespace are enforced |
| Config file indentation | Contributor edits `.golangci.yml` or `config.toml` | Editor saves file | 2-space indentation, UTF-8, LF line endings, and final newline are enforced |
| Markdown line break preservation | Contributor edits `README.md` with intentional 2-space line breaks | Editor saves file | `trim_trailing_whitespace = false` preserves trailing whitespace for line breaks |
