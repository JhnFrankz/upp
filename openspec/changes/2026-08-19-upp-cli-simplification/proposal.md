# Proposal: upp CLI Simplification & Subcommand/Flag Pruning

## Intent

The `upp` CLI currently carries unnecessary complexity and minor ergonomic friction points that detract from an optimal developer experience:
1. **Redundant Subcommands**: `upp export` and `upp import` duplicate standard file operations (`cp`, `cat`) and dotfiles workflows on `~/.config/upp/config.toml`, adding cognitive load and a dedicated "Config Commands" help group that clutters `upp --help`.
2. **Ambiguous Bare `upp` Invocation**: Running bare `upp` (without subcommands) currently acts as an exact duplicate of `upp check`. For a modern CLI, bare invocation should provide a clean, educational, non-destructive dashboard and welcome screen that displays version information, configured tools status, and primary command guidance.
3. **Missing Standard UNIX Shorthands**: Common flags lack standard single-letter shorthands (e.g., `-q` for `--quiet`, `-n` for `--dry-run`), deviating from standard POSIX/Unix CLI expectations.
4. **Opaque Diagnostic Capability**: When package manager adapters fail during check or update, users have no standard `--verbose` / `-v` flag to view detailed adapter subprocess stderr for troubleshooting.
5. **Help Menu Fragmentation**: Grouping is fragmented across "Tool Commands", "Config Commands", and "Maintenance". Consolidating into "Commands" and "Maintenance" clarifies the mental model.

Streamlining the CLI interface eliminates dead code, aligns flags with standard conventions, improves discoverability, and delivers an intuitive, zero-friction developer experience.

---

## Scope

### In Scope
- **Subcommand Pruning**: Completely remove `upp export` and `upp import` subcommands from the CLI (`internal/cli/export.go`, `internal/cli/import.go`) and remove config serialization helpers from `internal/config/export.go`.
- **Help Restructuring**: Eliminate the "Config Commands" group and reorganize root `--help` into two clean groups:
  - `Commands`: `check`, `list`, `update`
  - `Maintenance`: `init`, `self-update`
- **Bare `upp` Dashboard / Welcome Screen**: Transform bare `upp` invocation (no arguments) to render an educational, non-destructive dashboard displaying:
  - `upp` version and platform information
  - Summary of configured and enabled tools
  - Clear next-step guidance for primary workflows (`upp check`, `upp update`, `upp list`)
- **Standard UNIX Shorthands**:
  - Add `-q` shorthand for `--quiet`
  - Add `-n` shorthand for `--dry-run` on `upp update`
  - Add `-v` / `--verbose` global persistent flag to output adapter stderr and diagnostics upon failure
- **Spec Updates**: Update `command-interface`, `config-system`, and `ux-patterns` canonical specs.
- **Documentation & Tests**: Update `README.md`, unit tests, and integration test suites.

### Out of Scope / Non-Goals
- **Interactive Check-and-Prompt Execution in Bare `upp`**: Bare `upp` remains strictly informative and non-destructive (dashboard), directing users to explicit action commands (`upp update`, `upp check`).
- **Interactive TUI / Terminal Menus**: No ncurses/bubbletea TUI widgets or multi-select menus.
- **Config Schema Changes**: No alterations to the underlying TOML configuration schema (`[tools]`, `[custom]`, `[settings]`).
- **`--json` Output Format**: Deferred to a dedicated machine-readable interface change.
- **Tool Adapter Architecture Changes**: Core adapter interfaces (`detect`, `check`, `update`, `list`) remain unchanged.

---

## Capabilities

### New Capabilities
None.

### Modified Capabilities
- **[`command-interface`](file:///home/jhan/projects/upp/openspec/specs/command-interface/spec.md)**:
  - Remove `export` and `import` subcommands from the command catalog.
  - Redefine bare `upp` root invocation to render the welcome/dashboard screen.
  - Add `-q` (`--quiet`), `-n` (`--dry-run`), and `-v` / `--verbose` flags.
  - Update help grouping to two sections: `Commands` (`check`, `list`, `update`) and `Maintenance` (`init`, `self-update`).
- **[`config-system`](file:///home/jhan/projects/upp/openspec/specs/config-system/spec.md)**:
  - Remove `Export/Import` requirement and serialization utilities.
  - Specify standard filesystem operations for config backup and portability.
- **[`ux-patterns`](file:///home/jhan/projects/upp/openspec/specs/ux-patterns/spec.md)**:
  - Add requirement for the Bare `upp` Dashboard / Welcome Screen (version, tool overview, next-step guidance).
  - Add requirement for Verbose Diagnostics (`-v` / `--verbose` stderr emission on adapter failure).
  - Update help output formatting specification.

---

## Approach

### 1. Delivery Strategy & Slicing
To ensure rigorous review, maintain continuous test greenness, and stay well within the 400-line change budget per slice, implementation is divided into 2 sequential slices:

#### Slice 1: Subcommand Pruning, Help Restructuring & Flag Shorthands (~220 lines)
- Delete [`internal/cli/export.go`](file:///home/jhan/projects/upp/internal/cli/export.go), [`internal/cli/import.go`](file:///home/jhan/projects/upp/internal/cli/import.go), and [`internal/config/export.go`](file:///home/jhan/projects/upp/internal/config/export.go).
- Update [`internal/cli/parser.go`](file:///home/jhan/projects/upp/internal/cli/parser.go):
  - Remove `export` and `import` command registrations.
  - Replace 3 command groups with 2 groups: `Commands` (ID: `commands`) and `Maintenance` (ID: `maintenance`).
  - Assign `check`, `list`, `update` to `Commands`; assign `init`, `self-update` to `Maintenance`.
  - Add `-q` shorthand for `--quiet`.
  - Add `-v` / `--verbose` global persistent flag to [`GlobalFlags`](file:///home/jhan/projects/upp/internal/cli/parser.go#L15-L20).
- Update [`internal/cli/update.go`](file:///home/jhan/projects/upp/internal/cli/update.go):
  - Add `-n` shorthand for `--dry-run`.
- Update parser and command tests in `internal/cli/parser_test.go` and `internal/cli/help_test.go`.
- Spec deltas for `command-interface` and `config-system`.

#### Slice 2: Bare `upp` Dashboard & Verbose Diagnostics (~280 lines)
- Implement bare `upp` dashboard runner in `internal/cli/root.go` / `internal/cli/parser.go`:
  - Print banner/version, platform, configured tool counts (enabled / total available).
  - Print quick-reference guide for primary commands: `upp check`, `upp update`, `upp list`.
  - Respect `--quiet` and non-TTY environments cleanly.
- Implement verbose diagnostics in execution pipeline:
  - When `-v` / `--verbose` is set, capture and output subprocess stderr when adapter operations fail.
- Update `internal/cli/check.go` and `internal/cli/update.go` tests.
- Spec delta for `ux-patterns`.
- Update [`README.md`](file:///home/jhan/projects/upp/README.md) documentation and command references.

---

## Affected Areas

| Area | Impact | Description |
|---|---|---|
| [`internal/cli/export.go`](file:///home/jhan/projects/upp/internal/cli/export.go) | Deleted | Remove `upp export` subcommand |
| [`internal/cli/import.go`](file:///home/jhan/projects/upp/internal/cli/import.go) | Deleted | Remove `upp import` subcommand |
| [`internal/config/export.go`](file:///home/jhan/projects/upp/internal/config/export.go) | Deleted | Remove export/import helper functions |
| [`internal/cli/parser.go`](file:///home/jhan/projects/upp/internal/cli/parser.go) | Modified | Update flags (`-q`, `-v`), help groups (`Commands`, `Maintenance`), remove pruned commands |
| [`internal/cli/update.go`](file:///home/jhan/projects/upp/internal/cli/update.go) | Modified | Add `-n` shorthand to `--dry-run` |
| [`internal/cli/check.go`](file:///home/jhan/projects/upp/internal/cli/check.go) | Modified | Wire verbose diagnostic output on failure |
| [`internal/cli/help_test.go`](file:///home/jhan/projects/upp/internal/cli/help_test.go) | Modified | Assert 2 help groups and absence of `export`/`import` |
| [`internal/cli/parser_test.go`](file:///home/jhan/projects/upp/internal/cli/parser_test.go) | Modified | Update tests for flag shorthands and command tree |
| [`openspec/specs/command-interface/spec.md`](file:///home/jhan/projects/upp/openspec/specs/command-interface/spec.md) | Modified | Update command catalog, global flags, help groups |
| [`openspec/specs/config-system/spec.md`](file:///home/jhan/projects/upp/openspec/specs/config-system/spec.md) | Modified | Prune Export/Import requirement |
| [`openspec/specs/ux-patterns/spec.md`](file:///home/jhan/projects/upp/openspec/specs/ux-patterns/spec.md) | Modified | Add Bare `upp` Dashboard and Verbose Output requirements |
| [`README.md`](file:///home/jhan/projects/upp/README.md) | Modified | Update CLI command reference, flag tables, and quickstart examples |

---

## Verification Strategy

### 1. Automated Unit & Integration Tests
- **Flag Parsing**:
  - Test `-q` acts identically to `--quiet`.
  - Test `-n` acts identically to `--dry-run` on `upp update`.
  - Test `-v` / `--verbose` sets `GlobalFlags.Verbose = true`.
- **Command Tree & Help Output**:
  - Test `upp export` and `upp import` return unknown command error (exit 1).
  - Test `upp --help` shows exactly 2 groups: `Commands` (`check`, `list`, `update`) and `Maintenance` (`init`, `self-update`).
  - Test `completion` command remains hidden.
- **Bare `upp` Dashboard**:
  - Test bare `upp` execution renders version, tools summary, and command guidance.
  - Test bare `upp` performs zero destructive actions and makes no unauthorized network calls.
- **Verbose Diagnostics**:
  - Test adapter failure output contains detailed subprocess stderr when `-v` is active, and remains concise when `-v` is omitted.
- **Full Suite**:
  - Run `go test -v -count=1 ./...` across all packages.

### 2. Manual Verification
- Run `./bin/upp` without arguments: verify clean dashboard output.
- Run `./bin/upp --help`: verify simplified two-group structure.
- Run `./bin/upp update -n`: verify dry-run execution.
- Run `./bin/upp check -q`: verify quiet output summary.

---

## Risks & Tradeoffs

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| **Breaking scripts using `upp export` or `upp import`** | Low | Low | Config file is standard TOML located at standard paths (`~/.config/upp/config.toml`). Standard shell commands (`cp`, `cat`) provide direct replacements. Clear release notes will document the removal. |
| **Bare `upp` behavior change** | Low | Low | Bare `upp` previously duplicated `upp check`. Switching to an educational dashboard is non-destructive, safe, and explicitly instructs the user to run `upp check` or `upp update`. |
| **Verbose output noise** | Low | Low | Verbose logging is strictly opt-in via `-v` / `--verbose`; default output remains clean and concise inline status. |

---

## Rollback Plan

If regressions occur, changes can be rolled back via git revert on the feature branch. Because no data format or config schema changes are introduced, rollback is instantaneous and carries zero migration risk.

---

## Dependencies

- Standard Go toolchain (Go 1.22+).
- `github.com/spf13/cobra` for CLI command tree and flag parsing.
- Existing internal packages (`internal/config`, `internal/adapters`, `internal/output`).

---

## Success Criteria

- [ ] `upp export` and `upp import` subcommands are completely removed (`export.go`, `import.go` deleted).
- [ ] `upp --help` displays 2 command groups: `Commands` (`check`, `list`, `update`) and `Maintenance` (`init`, `self-update`).
- [ ] `-q` is supported as shorthand for `--quiet`.
- [ ] `-n` is supported as shorthand for `--dry-run` on `upp update`.
- [ ] `-v` / `--verbose` global flag outputs adapter stderr diagnostics on failure.
- [ ] Bare `upp` renders an informative, non-destructive dashboard showing version, configured tools, and command guidance.
- [ ] `openspec` specs (`command-interface`, `config-system`, `ux-patterns`) updated to reflect all changes.
- [ ] All tests pass: `go test -v -count=1 ./...` is 100% green.
