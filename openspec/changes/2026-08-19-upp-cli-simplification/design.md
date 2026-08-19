# Technical Design: upp CLI Simplification & Subcommand/Flag Pruning

## 1. Technical Approach & Architecture Strategy

### 1.1 Intent & Overview
The goal of this change is to streamline the `upp` CLI user experience by removing redundant subcommands (`export`, `import`), providing an informative, read-only dashboard on bare `upp` invocation, introducing standard POSIX flag shorthands (`-q`, `-n`, `-v`), enabling verbose error diagnostics on adapter failure, and consolidating the help menu into two logical command groups (`Commands` and `Maintenance`).

```
                    ┌────────────────────────────────────────────────────────┐
                    │                      upp CLI Root                      │
                    └───────────────────────────┬────────────────────────────┘
                                                │
                 ┌──────────────────────────────┴──────────────────────────────┐
                 │                                                             │
         [ Bare Invocation ]                                         [ Subcommand Invocation ]
                 │                                                             │
    ┌────────────▼────────────┐                               ┌────────────────┴────────────────┐
    │      runDashboard       │                               │                                 │
    │  (internal/cli/root.go) │                       ┌───────▼────────┐                ┌───────▼────────┐
    └────────────┬────────────┘                       │    Commands    │                │  Maintenance   │
                 │                                    │ check, update, │                │ init,          │
     ┌───────────┴───────────┐                        │ list           │                │ self-update    │
     │                       │                        └───────┬────────┘                └────────────────┘
[ No Config ]         [ Config Exists ]                       │
     │                       │                        ┌───────▼─────────────────────────┐
Prompt to run         Render version,                 │ Global Flags:                   │
  `upp init`          platform, tool counts,          │   -q / --quiet                  │
                      & quickstart guide              │   -v / --verbose (on failures)  │
                                                      │   -n / --dry-run (update only)  │
                                                      └─────────────────────────────────┘
```

### 1.2 Key Subsystems & Interactions

1. **Subcommand Pruning**:
   - Physically remove `internal/cli/export.go` and `internal/cli/import.go`.
   - Remove `internal/config/export.go` and unused config export/import helper functions (`Export`, `ExportToFile`, `ImportFromFile`).
   - Remove `ExportFlags` and command registrations in `internal/cli/parser.go`.
   - Reorganize Cobra command groups from 3 (`tool`, `config`, `maintenance`) into 2 (`commands`, `maintenance`).
   - Attempting to run `upp export` or `upp import` will cleanly fail as an unknown command with exit code 1.

2. **Bare `upp` Dashboard Welcome Screen**:
   - Bare `upp` invocation executes `runDashboard(gf, version, cliDeps.dashboard)` in `internal/cli/root.go`.
   - Read-only and non-destructive: performs zero package manager queries and zero network operations.
   - If `!config.Exists()`: renders header banner and prompts the user to run `upp init` to configure their tools.
   - If `config.Exists()`: loads config, queries detected platform via `platform.Detect()`, counts enabled tools vs. platform catalog tools, and prints a structured overview and quick-start command reference (`upp check`, `upp update`, `upp list`, `upp --help`).
   - Respects `--quiet` / `-q` (suppresses decorative banner and guidance) and pipe/non-TTY modes (no ANSI escape codes).

3. **Standard UNIX Shorthands & Global Flags**:
   - `--quiet`: add single-letter shorthand `-q`.
   - `--verbose`: add persistent global flag `-v` (`GlobalFlags.Verbose`), enabling inline diagnostic output (subprocess stderr) when adapters fail.
   - `upp update --dry-run`: add single-letter shorthand `-n`.

4. **Verbose Error Diagnostics Rendering**:
   - Extend `internal/output/Renderer` to be verbose-aware.
   - When an adapter fails during `check` or `update` with `-v` / `--verbose` set:
     - Render concise failure status inline as usual.
     - Emit indented, formatted subprocess stderr directly beneath the failed tool entry.
   - In default (non-verbose) mode, raw subprocess stderr is suppressed to maintain terminal cleanliness.
   - In quiet mode (`-q`), `--quiet` takes precedence and suppresses verbose error details.

---

## 2. Architectural Decisions & Alternatives

### Decision 1: Complete Subcommand Pruning vs. Deprecation Flags
- **Choice**: Complete physical removal of `upp export`, `upp import`, and `internal/config/export.go`.
- **Alternatives Considered**:
  - *Alternative A: Deprecation Warnings*: Retain commands with hidden visibility and log a warning directing users to manual file operations.
  - *Alternative B: Shell Passthrough*: Alias `upp export` to `cat ~/.config/upp/config.toml`.
- **Rationale**: The config file is standard TOML located at standard paths (`~/.config/upp/config.toml` on Unix, `%APPDATA%/upp/config.toml` on Windows). Standard file utilities (`cp`, `cat`, git) already provide superior dotfiles management and backup capabilities. Physical removal immediately eliminates dead code, simplifies the help menu, and avoids carrying legacy baggage forward.

### Decision 2: Bare `upp` Dashboard Render Seam
- **Choice**: Dedicated `runDashboard` function in `internal/cli/root.go` backed by an injectable dependency seam (`dashboardDeps`) in `internal/cli/deps.go` and presentation methods in `internal/output/render.go`.
- **Alternatives Considered**:
  - *Alternative A: Interactive TUI*: Launch a full-screen BubbleTea / ncurses menu prompting for operations.
  - *Alternative B: Retain `upp check` Default*: Keep running `upp check` when no arguments are passed.
- **Rationale**: An interactive TUI adds substantial external dependencies and breaks headless / CI automation. Defaulting to `upp check` causes unexpected network latency and execution time on bare invocation. An instant (<5ms), informative dashboard provides immediate discovery for new users while remaining fully script-safe and non-destructive.

### Decision 3: Flag Shorthands and Global Verbose Logging
- **Choice**: Persistent `-v` / `--verbose` flag on root command stored in `GlobalFlags.Verbose`, plumbed to `output.Renderer` and execution loops.
- **Alternatives Considered**:
  - *Alternative A: Per-command `--verbose` flags*: Add local `--verbose` only to `check` and `update`.
  - *Alternative B: Environment Variable Only (`UPP_DEBUG=1`)*: Use env vars instead of CLI flags.
  - *Alternative C: Unconditional stderr emission*: Always print adapter stderr on failure.
- **Rationale**: POSIX conventions expect `-v` and `-q` as global persistent flags. Unconditional stderr printing creates noisy terminal output when failures are routine (e.g. missing tools or offline checks). Persistent `-v` provides uniform diagnostic control across all subcommands, with explicit precedence rules (`--quiet` overrides `--verbose`).

### Decision 4: Help Template Customization vs. Cobra Default Groups
- **Choice**: Native Cobra `GroupID` assignment to two clean groups (`commands` and `maintenance`) defined in `internal/cli/parser.go` / `internal/cli/root.go`.
- **Alternatives Considered**:
  - *Alternative A: Custom Go Template*: Override `root.SetUsageTemplate(...)` with custom regex parsing.
  - *Alternative B: Flat Command List*: Remove groups entirely and list commands alphabetically.
- **Rationale**: Custom usage templates are fragile and frequently break across Cobra minor version upgrades. Native Cobra groups (`root.AddGroup(...)`) provide clean, standard help formatting across both `upp --help` and `upp help` subcommands without template manipulation.

---

## 3. Concrete File Specifications & Code Modifications

### 3.1 Deleted Files
- `internal/cli/export.go`
- `internal/cli/import.go`
- `internal/config/export.go`

### 3.2 `internal/cli/parser.go` Modifications
- **GlobalFlags**:
  ```go
  type GlobalFlags struct {
      Quiet   bool
      Verbose bool
      CI      bool
      Only    string
      Skip    string
  }
  ```
- **Flag Registration in `BuildRoot()`**:
  ```go
  root.PersistentFlags().BoolVarP(&gf.Quiet, "quiet", "q", false, "reduce output to essential status only")
  root.PersistentFlags().BoolVarP(&gf.Verbose, "verbose", "v", false, "enable verbose diagnostic output on failure")
  root.PersistentFlags().BoolVar(&gf.CI, "ci", false, "non-interactive mode (exit non-zero on failure)")
  root.PersistentFlags().StringVar(&gf.Only, "only", "", "process only these tools (comma-separated)")
  root.PersistentFlags().StringVar(&gf.Skip, "skip", "", "skip these tools (comma-separated)")
  ```
- **Command Grouping in `AddCommands()`**:
  ```go
  root.AddGroup(
      &cobra.Group{ID: "commands", Title: "Commands"},
      &cobra.Group{ID: "maintenance", Title: "Maintenance"},
  )

  check := NewCheckCommand(gf)
  check.GroupID = "commands"
  update := NewUpdateCommand(gf)
  update.GroupID = "commands"
  list := NewListCommand(gf)
  list.GroupID = "commands"
  init := NewInitCommand(gf)
  init.GroupID = "maintenance"
  selfUpdate := NewSelfUpdateCommand(gf)
  selfUpdate.GroupID = "maintenance"

  root.AddCommand(check, update, list, init, selfUpdate)
  ```
- **Remove `ExportFlags` struct**.

### 3.3 `internal/cli/root.go` Implementation
Create `internal/cli/root.go` to house `runDashboard` and bare invocation logic:
```go
package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/JhnFrankz/upp/internal/config"
	"github.com/JhnFrankz/upp/internal/output"
	"github.com/JhnFrankz/upp/internal/platform"
)

type dashboardDeps struct {
	configExists func() bool
	loadConfig   func() (*config.Config, error)
	detectPlatform func() (platform.Platform, error)
}

func runDashboard(gf *GlobalFlags, version string, w io.Writer, deps dashboardDeps) error {
	if deps.configExists == nil {
		deps.configExists = config.Exists
	}
	if deps.loadConfig == nil {
		deps.loadConfig = config.Load
	}
	if deps.detectPlatform == nil {
		deps.detectPlatform = platform.Detect
	}

	p, err := deps.detectPlatform()
	if err != nil {
		p = platform.Platform{OS: "unknown", Arch: "unknown"}
	}

	r := output.NewRenderer(w, gf.Quiet)

	if !deps.configExists() {
		r.DashboardNoConfig(version, fmt.Sprintf("%s/%s", p.OS, p.Arch))
		return nil
	}

	cfg, err := deps.loadConfig()
	if err != nil {
		return fmt.Errorf("cannot load config: %w", err)
	}

	// Count enabled tools vs available platform catalog tools + custom tools
	platformCatalog := platform.CatalogFor(p.OS)
	totalAvailable := len(platformCatalog) + len(cfg.Custom)
	
	enabledCount := 0
	for _, tool := range platformCatalog {
		if tCfg, ok := cfg.Tools[tool.ID]; !ok || tCfg.Enabled {
			enabledCount++
		}
	}
	for _, custom := range cfg.Custom {
		_ = custom
		enabledCount++
	}

	r.Dashboard(output.DashboardData{
		Version:        version,
		Platform:       fmt.Sprintf("%s/%s", p.OS, p.Arch),
		EnabledTools:   enabledCount,
		AvailableTools: totalAvailable,
	})

	return nil
}
```
Update `BuildRoot().RunE` in `parser.go` / `root.go`:
```go
RunE: func(cmd *cobra.Command, args []string) error {
    return runDashboard(gf, cmd.Root().Version, os.Stdout, cliDeps.dashboard)
},
```

### 3.4 `internal/cli/deps.go` Update
Add `dashboard dashboardDeps` to `cliDeps`:
```go
var cliDeps struct {
	dashboard  dashboardDeps
	check      checkDeps
	update     updateDeps
	list       listDeps
	selfUpdate selfUpdateDeps
}
```

### 3.5 `internal/cli/update.go` Update
- Add `-n` shorthand for `--dry-run`:
  ```go
  cmd.Flags().BoolVarP(&uf.DryRun, "dry-run", "n", false, "show planned actions without executing")
  ```
- When adapter check or update fails, capture stderr diagnostics and populate `ToolResult.Stderr`:
  ```go
  r := output.NewRendererForced(os.Stdout, isTerminal(os.Stdout), isTerminal(os.Stdout), gf.Quiet, gf.Verbose)
  ```

### 3.6 `internal/output/render.go` Enhancements
- Extend `Renderer` struct:
  ```go
  type Renderer struct {
      w       io.Writer
      color   bool
      emoji   bool
      quiet   bool
      verbose bool
      mu      sync.Mutex
  }
  ```
- Update constructors:
  ```go
  func NewRenderer(w io.Writer, quiet bool) *Renderer {
      return NewRendererVerbose(w, quiet, false)
  }

  func NewRendererVerbose(w io.Writer, quiet, verbose bool) *Renderer {
      color := isTerminal(w)
      return &Renderer{
          w:       w,
          color:   color,
          emoji:   color,
          quiet:   quiet,
          verbose: verbose,
      }
  }

  func NewRendererForced(w io.Writer, color, emoji, quiet, verbose bool) *Renderer {
      return &Renderer{
          w:       w,
          color:   color,
          emoji:   emoji,
          quiet:   quiet,
          verbose: verbose,
      }
  }
  ```
- Add `DashboardData` struct and rendering methods:
  ```go
  type DashboardData struct {
      Version        string
      Platform       string
      EnabledTools   int
      AvailableTools int
  }

  func (r *Renderer) Dashboard(data DashboardData) {
      if r.quiet {
          return
      }
      _, _ = fmt.Fprintf(r.w, "%s upp %s (%s)\n\n", r.cyan("●"), data.Version, data.Platform)
      _, _ = fmt.Fprintf(r.w, "  Tools: %d enabled (%d configured for platform)\n\n", data.EnabledTools, data.AvailableTools)
      _, _ = fmt.Fprintln(r.w, "  Commands:")
      _, _ = fmt.Fprintf(r.w, "    %-14s %s\n", "upp check", "Check for tool updates (read-only)")
      _, _ = fmt.Fprintf(r.w, "    %-14s %s\n", "upp update", "Update all enabled tools (-n for dry-run)")
      _, _ = fmt.Fprintf(r.w, "    %-14s %s\n", "upp list", "List configured tools and versions")
      _, _ = fmt.Fprintf(r.w, "    %-14s %s\n", "upp --help", "Show help and options")
  }

  func (r *Renderer) DashboardNoConfig(version, platform string) {
      if r.quiet {
          return
      }
      _, _ = fmt.Fprintf(r.w, "%s upp %s (%s)\n\n", r.cyan("●"), version, platform)
      _, _ = fmt.Fprintln(r.w, "  No configuration found.")
      _, _ = fmt.Fprintln(r.w, "  Run \"upp init\" to detect installed tools and initialize your config.")
  }
  ```
- Add verbose error diagnostics rendering in `ToolLine` / `detailSummary` / `CheckSummary`:
  ```go
  // In verboseToolLine when StatusFailed:
  case StatusFailed:
      errMsg := ""
      if result.Error != nil {
          errMsg = " (" + result.Error.Error() + ")"
      }
      _, _ = fmt.Fprintf(r.w, "  %s %s%s\n", icon, r.red(result.Name), errMsg)
      if r.verbose && !r.quiet && result.Stderr != "" {
          for _, line := range strings.Split(strings.TrimSpace(result.Stderr), "\n") {
              _, _ = fmt.Fprintf(r.w, "    %s %s\n", r.dim("│"), r.dim(line))
          }
      }
  ```

---

## 4. Threat & Edge Case Matrix

| Threat / Edge Case | Likelihood | Impact | Mitigation Strategy |
|---|---|---|---|
| **Headless CI bare `upp` invocation** | Med | Low | Bare `upp` uses `isTerminal(w)` to detect non-TTY environments and disables ANSI codes. Bare `upp` is strictly read-only, makes no network calls, and exits with code 0 without blocking. |
| **Missing `config.toml` on bare invocation** | High | Low | `config.Exists()` is checked before invoking `config.Load()`. If missing, `DashboardNoConfig` renders a friendly prompt to run `upp init` and exits cleanly with code 0. |
| **Sensitive info or unbounded output in adapter stderr** | Med | Med | Subprocess stderr output is truncated to 500 characters and stripped of control sequences before being passed to `ToolResult.Stderr`. Verbose stderr is rendered only on adapter failure when `-v` is explicitly supplied. |
| **Conflicting flag combinations (`-q` and `-v`)** | Low | Low | Strict precedence is enforced: `--quiet` (`-q`) overrides `--verbose` (`-v`). In quiet mode, all verbose diagnostic stderr output is suppressed. |
| **Conflicting flag combinations (`--only` and `--skip`)** | Low | Low | Existing precedence holds: `--only` takes complete precedence over `--skip`. |
| **Pruned subcommand invocation in legacy scripts** | Med | Low | Invocations of `upp export` and `upp import` return an unknown command error with exit status 1. Release notes clearly document replacement with standard file tools (`cp ~/.config/upp/config.toml <dest>`). |

---

## 5. Delivery Strategy: 2 Review Slices

### Slice 1: Subcommand Pruning, Help Restructuring & Flag Shorthands (~220 lines)
- **Work Units**:
  - **WU-1.1**: Delete `internal/cli/export.go`, `internal/cli/import.go`, and `internal/config/export.go`. Clean up obsolete tests in `internal/config/config_test.go`, `internal/config/config_expanded_test.go`, and `internal/cli/integration_test.go`.
  - **WU-1.2**: Update `internal/cli/parser.go` and `internal/cli/update.go`:
    - Register `-q` shorthand for `--quiet`.
    - Register `-v` shorthand for `--verbose` on `GlobalFlags`.
    - Register `-n` shorthand for `--dry-run` on `UpdateFlags`.
    - Restructure help groups into `Commands` (`check`, `list`, `update`) and `Maintenance` (`init`, `self-update`).
    - Remove `export` and `import` command registrations and `ExportFlags`.
  - **WU-1.3**: Update `internal/cli/help_test.go` and `internal/cli/parser_test.go` to assert the 2 new groups, test flag shorthands, and verify unknown command rejection for `export`/`import`.
- **Target Line Budget**: ~220 lines (net negative due to file deletions).
- **Verification Commands**:
  ```bash
  go test -v -count=1 ./internal/config/...
  go test -v -count=1 -run "TestHelp_|TestBuildRoot_|TestAddCommands|TestParseFilter" ./internal/cli
  ```

### Slice 2: Bare `upp` Dashboard & Verbose Diagnostics (~280 lines)
- **Work Units**:
  - **WU-2.1**: Update `internal/output/render.go` and `internal/output/render_test.go`:
    - Add `Dashboard` and `DashboardNoConfig` methods to `Renderer`.
    - Add `verbose` field, updated constructors, and indented stderr diagnostic output on failed tool entries.
    - Add unit tests for dashboard formatting, TTY/non-TTY modes, quiet mode suppression, and verbose error rendering.
  - **WU-2.2**: Implement `runDashboard` and `dashboardDeps` in `internal/cli/root.go` and wire into `BuildRoot().RunE` via `cliDeps.dashboard`. Add hermetic tests in `internal/cli/root_test.go` / `internal/cli/parser_test.go`.
  - **WU-2.3**: Wire `GlobalFlags.Verbose` into `runCheck` (`internal/cli/check.go`) and `runUpdate` (`internal/cli/update.go`), capturing adapter stderr on failure and passing it to the renderer.
  - **WU-2.4**: Update `scripts/smoke-test.sh` and `README.md` to reflect the new dashboard behavior, updated flags, and pruned subcommands.
- **Target Line Budget**: ~280 lines.
- **Verification Commands**:
  ```bash
  go test -v -count=1 ./internal/output/...
  go test -v -count=1 ./internal/cli/...
  go test -v -count=1 ./...
  ./scripts/smoke-test.sh
  ```
