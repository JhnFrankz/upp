# CI Workflow Specification

## Purpose

Defines the CI/CD contract for upp's GitHub Actions workflow (`.github/workflows/ci.yml`): run concurrency cancellation, per-job timeout bounds, static checks over scripts and workflow files, and the main branch protection delivery requirement. This domain does NOT amend the Release Process Specification: `needs: [test, lint]`, job-scoped `contents: write`, and the "no additional permissions MAY be granted" rule remain exactly as specified there.

## Requirements

### Requirement: Run Concurrency Cancellation

CI MUST declare a concurrency group `ci-${{ github.ref }}`. CI MUST set `cancel-in-progress` to `${{ !startsWith(github.ref, 'refs/tags/') }}`: a newer run on the same non-tag ref MUST cancel the superseded run, and runs on `refs/tags/*` MUST NOT be cancelled.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Superseded branch run | In-progress run on `refs/heads/feature-x` | Newer push to same ref | In-progress run cancelled, newer run proceeds |
| Tag run never cancelled | In-progress run for `v0.2.0` tag | Another run starts | No cancellation; tag run completes |
| Dispatch | Manual `workflow_dispatch` run | Run starts | Shares the main group (benign, accepted) |

### Requirement: Job Timeout Bounds

Every CI job MUST declare a `timeout-minutes` limit: `test` 15, `lint` 10, `release` 20. A job exceeding its limit MUST be terminated by GitHub and its run reported as failed.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Hung job | `test` hangs past 15 min | Limit reached | Job terminated, run fails |
| Healthy run | All jobs finish under limits | Jobs complete | No job hits its timeout |

### Requirement: Multiplatform Test Matrix

The `test` job MUST execute across a multiplatform matrix covering Linux (`ubuntu-latest`), macOS (`macos-latest`), and Windows (`windows-latest`) with `fail-fast: false` to ensure complete visibility of test outcomes on each platform.

Execution invariants:
- **Universal verification**: Each platform runner MUST execute `go vet ./...`, `go test ./... -count=1`, and `go test ./... -count=1 -race`.
- **Native compilation**: Each platform runner MUST compile the binary natively via `go build -trimpath ./cmd/upp` without dependency on external build tools (such as `make` on Windows).
- **Format gate**: The `gofmt -s -l` gate MUST execute on Linux (`runner.os == 'Linux'`) to enforce formatting standards without platform-specific shell syntax conflicts on Windows runners.
- **End-to-end smoke tests**: The `scripts/smoke-test.sh` gate MUST execute on POSIX platforms (Linux and macOS via `runner.os != 'Windows'`).
- **Release dependency**: The `release` job MUST depend on `test` and `lint` (`needs: [test, lint]`), requiring all matrix platforms to pass before packaging or publishing assets.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Multiplatform test run | PR or push to main | CI triggers `test` job | Separate runners spawn for `ubuntu-latest`, `macos-latest`, and `windows-latest` |
| Native build on Windows | `windows-latest` runner executes | Build step runs | `go build -trimpath ./cmd/upp` succeeds, producing `upp.exe` |
| Smoke test on POSIX | Linux or macOS runner executes | Smoke test step runs | `scripts/smoke-test.sh --skip-build` executes and validates CLI contract |
| Smoke test skipped on Windows | `windows-latest` runner executes | Smoke test step evaluated | Step skipped via `runner.os != 'Windows'` |
| Test failure on one OS | Failure on macOS runner with passing Linux/Windows | Matrix executes | `fail-fast: false` allows other runners to finish; overall run fails |

### Requirement: Script and Workflow Static Checks

The `lint` job MUST run `shellcheck -S warning` over `scripts/install.sh`, `scripts/smoke-test.sh`, and `scripts/publish-release.sh`, and MUST run actionlint pinned to `v1.7.7` over the workflow files. Any warning-level shellcheck finding or actionlint error MUST fail the `lint` job.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Shellcheck warning | A script has an unquoted-variable warning | `lint` job runs | Job fails; PR blocked |
| Actionlint error | Workflow YAML contains an invalid key | `lint` job runs | Job fails |
| Clean scripts and workflow | No findings in scripts or YAML | `lint` job runs | Job passes |

### Requirement: Main Branch Protection (Delivery Requirement)

Applied out-of-band at apply time via `gh api` — NOT workflow behavior. `main` MUST require a pull request before merging, MUST require checks `test` and `lint` to pass, MUST require linear history, and MUST auto-delete merged branches. Post-apply verification MUST confirm the protection is active.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Protection active | Apply completed | Verify via `gh api` | PR required, checks test+lint, linear history, auto-delete confirmed |
| Direct push | Protection active | Push to `main` | Rejected |

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
