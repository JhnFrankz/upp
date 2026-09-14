# Repository Rules

This file is the source of repo conventions for AI code reviews (gga) and AI agents working in this repository.

## Project

- upp — cross-platform dev environment updater (Go binary, cobra CLI + TOML config).
- Single Go module at repo root. Entry point: `cmd/upp/main.go`.

## Go

- Format with `gofmt -s` (CI enforces a clean `gofmt -s -l`).
- Lint with golangci-lint v2.13.2 (`.golangci.yml`); `go vet` is the Makefile fallback.
- Keep dependencies in `go.mod` minimal.

## Testing (strict TDD)

- Test runner: `go test ./... -count=1` (workspace-level, covers the module).
- Write table-driven tests with the stdlib `testing` package. Any new behavior needs tests.
- CI also runs the race detector (`go test ./... -count=1 -race`).
- E2E smoke check: `bash scripts/smoke-test.sh --skip-build`.

## Architecture

- Package-per-layer layout under `internal/`: adapters (with `official/`), cli, config, engine, output, platform, security, selfupdate, uninstall.
- One adapter per supported tool; register new tools in the adapters registry. Follow the existing adapter pattern.
- Respect the security/trust model (spec domain: `security-model`) when touching command execution or updates.

## CLI contract (must keep working)

- Commands: `list`, `update`, `init`, `self-update`, `uninstall`.
- Flags: `--ci`, `--dry-run` (`-n`), `--quiet` (`-q`), `--verbose` (`-v`), `--only`.

## Commits

- Conventional commits: `feat|fix|docs|chore|ci(scope): summary`.
- Specs and canonical docs live in `openspec/`; keep factual claims there in sync with reality.
