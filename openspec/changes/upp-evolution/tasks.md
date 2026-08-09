# Tasks: upp Evolution — Cross-Platform Dev Environment Updater

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 2800–3500 |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | PR 1 → PR 2 → PR 3 → PR 4 → PR 5 → PR 6 → PR 7 → PR 8 |
| Delivery strategy | ask-on-risk |
| Chain strategy | pending |

Decision needed before apply: Yes
Chained PRs recommended: Yes
Chain strategy: pending
400-line budget risk: High

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|----------------------|-----------------|-------------------|
| 1 | Go module + core types + platform detection | PR 1 | `go build ./...` | `go vet ./...` | `cmd/`, `internal/platform/` |
| 2 | Config system | PR 2 | `go test ./internal/config/...` | `upp init` on fresh system | `internal/config/` |
| 3 | Official adapters (batch 1: package managers) | PR 3 | `go test ./internal/adapters/...` | `upp check` on Linux | `internal/adapters/official/` (apt, brew, npm, pnpm) |
| 4 | Official adapters (batch 2: runtimes + tools) | PR 4 | `go test ./internal/adapters/...` | `upp check` on Linux | `internal/adapters/official/` (nvm, bun, gh, docker, go, opencode) |
| 5 | Security + custom adapter | PR 5 | `go test ./internal/security/...` | `upp update --ci` with custom tool | `internal/security/`, `internal/adapters/custom.go` |
| 6 | CLI commands + output rendering | PR 6 | `go build ./... && ./upp --help` | `upp init && upp update` end-to-end | `internal/cli/`, `internal/output/` |
| 7 | Polish + distribution | PR 7 | `make build-all` | `upp list` on each platform | `Makefile`, `upp-legacy.sh`, docs |
| 8 | Tests + final verification | PR 8 | `go test ./...` | Full `upp update` flow | `*_test.go` files |

## Phase 1: Foundation / Infrastructure

- [x] 1.1 Create `go.mod` with module `github.com/jhan/upp`, add dependencies: `cobra`, `BurntSushi/toml`
- [x] 1.2 Create `internal/adapters/interface.go` — define `Adapter`, `UpdateInfo`, `Result`, `ToolInfo` types per design
- [x] 1.3 Create `internal/platform/detect.go` — `runtime.GOOS`/`runtime.GOARCH` → `Platform` struct with OS/Arch fields
- [x] 1.4 Create `internal/platform/catalog.go` — `CatalogFor(os string) []Adapter` with Linux/macOS/Windows registry
- [x] 1.5 Create `cmd/upp/main.go` — entry point: version string, cobra root command, call `cli.Execute()`

## Phase 2: Configuration System

- [ ] 2.1 Create `internal/config/config.go` — `Config` struct with `Version`, `Settings`, `Tools`, `Custom` fields; `Load()`, `Save()`, `Validate()`
- [ ] 2.2 Create `internal/config/defaults.go` — default settings (`language: "en"`, `interactive: true`), platform catalog merge
- [ ] 2.3 Create `internal/config/export.go` — `Export(w io.Writer)`, `Import(r io.Reader)` with TOML round-trip

## Phase 3: Official Tool Adapters

- [ ] 3.1 Create `internal/adapters/official/apt.go` — Linux-only, `apt upgrade` update
- [ ] 3.2 Create `internal/adapters/official/brew.go` — Linux+macOS, `brew update && brew upgrade`
- [ ] 3.3 Create `internal/adapters/official/npm.go` — All platforms, `npm update -g`
- [ ] 3.4 Create `internal/adapters/official/pnpm.go` — All platforms, `pnpm update -g` with corruption recovery
- [ ] 3.5 Create `internal/adapters/official/nvm.go` — All platforms, `nvm install stable`
- [ ] 3.6 Create `internal/adapters/official/bun.go` — All platforms, `bun upgrade`
- [ ] 3.7 Create `internal/adapters/official/gh.go` — Platform dispatch: apt/brew/winget
- [ ] 3.8 Create `internal/adapters/official/docker.go` — Platform dispatch: apt/brew/winget
- [ ] 3.9 Create `internal/adapters/official/go.go` — Platform dispatch: brew/winget/manual
- [ ] 3.10 Create `internal/adapters/official/opencode.go` — All platforms, curl installer

## Phase 4: Security & Custom Adapter

- [ ] 4.1 Create `internal/security/trust.go` — `TrustLevel` enum, `RiskLevel` enum, `ClassifyCommand()` hybrid heuristic
- [ ] 4.2 Create `internal/security/confirm.go` — interactive confirmation prompt, `--ci` error behavior
- [ ] 4.3 Create `internal/adapters/custom.go` — config-defined adapter: exec user command, trust/risk integration

## Phase 5: CLI Commands & Output

- [ ] 5.1 Create `internal/cli/parser.go` — cobra root command, global flag binding (`--quiet`, `--ci`, `--only`, `--skip`)
- [ ] 5.2 Create `internal/cli/flags.go` — `Flags` struct with parsed values
- [ ] 5.3 Implement `init` command — detect tools, interactive selection, config generation
- [ ] 5.4 Implement `update` command — iterate adapters, detect→check→confirm→update flow, `--dry-run` support
- [ ] 5.5 Implement `check` command — query adapters for updates, display summary
- [ ] 5.6 Implement `list` command — show installed/detected tools with status
- [ ] 5.7 Implement `export`/`import` commands — TOML round-trip, `-o` flag for export
- [ ] 5.8 Implement default (no args) — show status + available updates (read-only)
- [ ] 5.9 Create `internal/output/render.go` — ANSI color, emoji status, pipe detection
- [ ] 5.10 Create `internal/output/language.go` — i18n string lookup (English default, architecture for future)

## Phase 6: Testing & Verification

- [ ] 6.1 Write unit tests for `internal/config/` — load, save, validate, export/import, defaults
- [ ] 6.2 Write unit tests for `internal/platform/` — detection, catalog filtering
- [ ] 6.3 Write unit tests for `internal/security/` — risk classification, confirmation logic
- [ ] 6.4 Write unit tests for adapter interface compliance — shared test harness
- [ ] 6.5 Write integration test — full update flow with mock adapters

## Phase 7: Polish & Distribution

- [ ] 7.1 Rename `upp.sh` → `upp-legacy.sh`
- [ ] 7.2 Create `Makefile` — build targets for all 5 platform/arch combinations
- [ ] 7.3 Verify `go build ./...` succeeds on Linux
- [ ] 7.4 Manual smoke test: `go run ./cmd/upp init`, `go run ./cmd/upp list`, `go run ./cmd/upp check`
