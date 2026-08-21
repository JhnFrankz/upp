// Package output handles terminal rendering with color, emoji, and
// graceful degradation for pipes and dumb terminals.
package output

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"text/tabwriter"

	"github.com/JhnFrankz/upp/internal/adapters"
)

// Status represents the outcome of a tool operation.
type Status int

const (
	StatusUpdated Status = iota
	StatusSkipped
	StatusFailed
	StatusAvailable
	StatusCurrent
)

// ToolResult holds the result of processing a single tool.
type ToolResult struct {
	Name    string
	Status  Status
	Version string // current version after update, or current version
	Error   error  // non-nil only for StatusFailed
	Stderr  string // captured adapter stderr on failure (verbose diagnostics)
}

// Summary holds the complete results of an update or check run.
type Summary struct {
	Results []ToolResult
	DryRun  bool
}

// Renderer handles formatted terminal output.
type Renderer struct {
	w       io.Writer
	color   bool
	emoji   bool
	quiet   bool
	verbose bool
	mu      sync.Mutex
}

// NewRenderer creates a Renderer that detects color/emoji support.
func NewRenderer(w io.Writer, quiet bool) *Renderer {
	return NewRendererVerbose(w, quiet, false)
}

// NewRendererVerbose creates a Renderer with explicit verbose setting.
func NewRendererVerbose(w io.Writer, quiet, verbose bool) *Renderer {
	color := isTerminal(w)
	return &Renderer{
		w:       w,
		color:   color,
		emoji:   color, // emoji follows color support
		quiet:   quiet,
		verbose: verbose,
	}
}

// NewRendererForced creates a Renderer with explicit color/emoji/quiet/verbose settings.
func NewRendererForced(w io.Writer, color, emoji, quiet, verbose bool) *Renderer {
	return &Renderer{
		w:       w,
		color:   color,
		emoji:   emoji,
		quiet:   quiet,
		verbose: verbose,
	}
}

// Color reports whether this renderer emits ANSI color sequences. Callers
// building auxiliary writers (e.g. CheckBoard) reuse the renderer's single
// TTY-detection source instead of re-detecting (design D5).
func (r *Renderer) Color() bool {
	return r.color
}

// isTerminal checks if the writer is a terminal that supports ANSI codes.
func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// --- Status indicators ---

func (r *Renderer) statusIcon(s Status) string {
	if !r.emoji {
		return r.statusIconPlain(s)
	}
	switch s {
	case StatusUpdated:
		return "✅"
	case StatusSkipped:
		return "⏭️ "
	case StatusFailed:
		return "❌"
	case StatusAvailable:
		return "⬆️ "
	case StatusCurrent:
		return "✔️ "
	default:
		return "?"
	}
}

func (r *Renderer) statusIconPlain(s Status) string {
	switch s {
	case StatusUpdated:
		return "[updated]"
	case StatusSkipped:
		return "[skipped]"
	case StatusFailed:
		return "[failed]"
	case StatusAvailable:
		return "[available]"
	case StatusCurrent:
		return "[current]"
	default:
		return "[?]"
	}
}

func (r *Renderer) statusLabel(s Status) string {
	switch s {
	case StatusUpdated:
		return "updated"
	case StatusSkipped:
		return "skipped"
	case StatusFailed:
		return "failed"
	case StatusAvailable:
		return "available"
	case StatusCurrent:
		return "current"
	default:
		return "unknown"
	}
}

// --- Color helpers ---

func (r *Renderer) colorize(code, text string) string {
	if !r.color {
		return text
	}
	return "\033[" + code + "m" + text + "\033[0m"
}

func (r *Renderer) red(text string) string    { return r.colorize("31", text) }
func (r *Renderer) green(text string) string  { return r.colorize("32", text) }
func (r *Renderer) yellow(text string) string { return r.colorize("33", text) }
func (r *Renderer) cyan(text string) string   { return r.colorize("36", text) }
func (r *Renderer) dim(text string) string    { return r.colorize("2", text) }

// --- Tool result rendering ---

// ToolLine renders a single tool result line.
func (r *Renderer) ToolLine(result ToolResult) {
	if r.quiet {
		r.quietToolLine(result)
		return
	}
	r.verboseToolLine(result)
}

func (r *Renderer) verboseToolLine(result ToolResult) {
	icon := r.statusIcon(result.Status)
	name := r.cyan(result.Name)

	switch result.Status {
	case StatusUpdated:
		if result.Version != "" {
			_, _ = fmt.Fprintf(r.w, "  %s %s %s\n", icon, name, r.dim(result.Version))
		} else {
			_, _ = fmt.Fprintf(r.w, "  %s %s\n", icon, name)
		}
	case StatusSkipped:
		_, _ = fmt.Fprintf(r.w, "  %s %s (not installed)\n", icon, name)
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
	case StatusAvailable:
		if result.Version != "" {
			_, _ = fmt.Fprintf(r.w, "  %s %s %s\n", icon, name, r.dim(result.Version))
		} else {
			_, _ = fmt.Fprintf(r.w, "  %s %s\n", icon, name)
		}
	case StatusCurrent:
		if result.Version != "" {
			_, _ = fmt.Fprintf(r.w, "  %s %s %s\n", icon, name, r.dim(result.Version))
		} else {
			_, _ = fmt.Fprintf(r.w, "  %s %s\n", icon, name)
		}
	}
}

func (r *Renderer) quietToolLine(result ToolResult) {
	icon := r.statusIcon(result.Status)
	name := result.Name

	switch result.Status {
	case StatusFailed:
		errMsg := ""
		if result.Error != nil {
			errMsg = ": " + result.Error.Error()
		}
		_, _ = fmt.Fprintf(r.w, "%s %s%s\n", icon, name, errMsg)
	case StatusSkipped:
		_, _ = fmt.Fprintf(r.w, "%s %s\n", icon, name)
	default:
		_, _ = fmt.Fprintf(r.w, "%s %s\n", icon, name)
	}
}

// --- Progress ---

// Progress shows a progress indicator for multi-tool operations. The
// operation label comes from the caller ("Checking" for read-only check,
// "Updating" for update) so read-only runs never claim to update (D2).
func (r *Renderer) Progress(op string, current, total int, name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.quiet || total <= 1 {
		return
	}
	_, _ = fmt.Fprintf(r.w, "  %s %s %d/%d: %s\n",
		r.dim("⟳"), op, current, total, r.cyan(name))
}

// ProgressInPlace shows a single-line progress indicator. In interactive
// (color/TTY) mode, it updates in-place using carriage return \r without a
// trailing newline. In non-TTY/CI mode, it falls back to line-buffered output
// with newlines and without carriage returns.
func (r *Renderer) ProgressInPlace(op string, current, total int, name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.quiet || total <= 1 {
		return
	}
	if r.color {
		_, _ = fmt.Fprintf(r.w, "\r  %s %s %d/%d: %s",
			r.dim("⟳"), op, current, total, r.cyan(name))
	} else {
		_, _ = fmt.Fprintf(r.w, "  %s %s %d/%d: %s\n",
			r.dim("⟳"), op, current, total, r.cyan(name))
	}
}

// --- Summary ---

// UpdateSummary renders the final summary of an update or check run.
func (r *Renderer) UpdateSummary(summary Summary) {
	updated, skipped, failed := countByStatus(summary.Results)
	current := countByStatusType(summary.Results, StatusCurrent)

	// In dry-run mode, StatusAvailable counts as "would update"
	available := 0
	if summary.DryRun {
		available = countByStatusType(summary.Results, StatusAvailable)
	}

	var parts []string

	if updated > 0 || available > 0 {
		label := "updated"
		if summary.DryRun {
			label = "would update"
		}
		count := updated + available
		parts = append(parts, r.green(fmt.Sprintf("%d %s", count, label)))
	}
	if current > 0 {
		parts = append(parts, fmt.Sprintf("%d up to date", current))
	}
	if skipped > 0 {
		parts = append(parts, fmt.Sprintf("%d skipped", skipped))
	}
	if failed > 0 {
		parts = append(parts, r.red(fmt.Sprintf("%d failed", failed)))
	}

	_, _ = fmt.Fprintln(r.w)

	// All skipped (or empty) → special message. Current tools ARE installed,
	// so they keep this branch from firing (D6).
	if updated == 0 && available == 0 && failed == 0 && current == 0 {
		_, _ = fmt.Fprintf(r.w, "%s All tools not installed. Nothing to do.\n", r.statusIcon(StatusSkipped))
		return
	}

	summaryLine := strings.Join(parts, ", ")

	// A run is only "clean" when it really updated something, nothing is
	// pending, nothing failed, and nothing was skipped. A --dry-run with
	// pending updates reports "N would update" and never claims "All clean!"
	// (D3).
	allClean := !summary.DryRun && updated > 0 && available == 0 && failed == 0 && skipped == 0

	if failed > 0 {
		_, _ = fmt.Fprintf(r.w, "%s %s. Review errors above.\n", r.statusIcon(StatusFailed), summaryLine)
	} else if allClean {
		_, _ = fmt.Fprintf(r.w, "%s %s. All clean!\n", r.statusIcon(StatusUpdated), summaryLine)
	} else if updated > 0 || available > 0 {
		_, _ = fmt.Fprintf(r.w, "%s %s\n", r.statusIcon(StatusUpdated), summaryLine)
	} else {
		_, _ = fmt.Fprintf(r.w, "%s %s\n", r.statusIcon(StatusCurrent), summaryLine)
	}

	// List tools per category in non-quiet mode
	if !r.quiet {
		r.detailSummary(summary)
	}
}

func (r *Renderer) detailSummary(summary Summary) {
	updated := filterByStatus(summary.Results, StatusUpdated)
	current := filterByStatus(summary.Results, StatusCurrent)
	skipped := filterByStatus(summary.Results, StatusSkipped)
	failed := filterByStatus(summary.Results, StatusFailed)

	if len(updated) > 0 {
		ids := toolNames(updated)
		_, _ = fmt.Fprintf(r.w, "  %s %s\n", r.green("Updated:"), strings.Join(ids, ", "))
	}
	if len(current) > 0 {
		ids := toolNames(current)
		_, _ = fmt.Fprintf(r.w, "  %s %s\n", r.green("Up to date:"), strings.Join(ids, ", "))
	}
	if len(skipped) > 0 {
		ids := toolNames(skipped)
		_, _ = fmt.Fprintf(r.w, "  Skipped: %s\n", strings.Join(ids, ", "))
	}
	if len(failed) > 0 {
		ids := toolNames(failed)
		_, _ = fmt.Fprintf(r.w, "  %s %s\n", r.red("Failed:"), strings.Join(ids, ", "))
		if r.verbose && !r.quiet {
			for _, f := range failed {
				if f.Stderr != "" {
					for _, line := range strings.Split(strings.TrimSpace(f.Stderr), "\n") {
						_, _ = fmt.Fprintf(r.w, "    %s %s\n", r.dim("│"), r.dim(line))
					}
				}
			}
		}
	}
}

// CheckSummary renders the summary for a check (read-only) operation.
// The "All tools up to date." tagline appears ONLY when every enabled tool
// was checked and current (current>0, nothing available/skipped/failed) or
// when the enabled-tool list is empty (parser_test bare-upp / integration
// all-disabled contract). Any skipped or failed tool suppresses the tagline
// and is counted explicitly (D4).
func (r *Renderer) CheckSummary(results []ToolResult) {
	var available, current, skipped, failed int
	for _, res := range results {
		switch res.Status {
		case StatusAvailable:
			available++
		case StatusCurrent:
			current++
		case StatusSkipped:
			skipped++
		case StatusFailed:
			failed++
		default:
			// Fail closed: a status outside the known enum must never fall
			// through to the "All tools up to date." tagline. Count it as a
			// failure so the summary reports it explicitly (guarded by
			// TestCheckSummary_UnknownStatusFailsClosed).
			failed++
		}
	}

	_, _ = fmt.Fprintln(r.w)

	if available == 0 && skipped == 0 && failed == 0 {
		// current > 0 (everything checked and current) or empty enabled-tool
		// list: keep the tagline (spec + test-enforced). Unknown statuses
		// cannot reach here — the counting switch fails them closed as failed.
		_, _ = fmt.Fprintf(r.w, "%s All tools up to date.\n", r.statusIcon(StatusCurrent))
		return
	}

	if available == 0 && failed == 0 {
		// Nothing pending or failed — only skipped tools (and current ones):
		// "Nothing to do." unless some tools are actually current.
		if current > 0 {
			_, _ = fmt.Fprintf(r.w, "%s %d up to date, %d skipped\n",
				r.statusIcon(StatusCurrent), current, skipped)
		} else {
			_, _ = fmt.Fprintf(r.w, "%s Nothing to do.\n", r.statusIcon(StatusSkipped))
		}
	} else {
		var parts []string
		if available > 0 {
			parts = append(parts, r.yellow(fmt.Sprintf("%d available", available)))
		}
		if current > 0 {
			parts = append(parts, fmt.Sprintf("%d up to date", current))
		}
		if skipped > 0 {
			parts = append(parts, fmt.Sprintf("%d skipped", skipped))
		}
		if failed > 0 {
			parts = append(parts, r.red(fmt.Sprintf("%d failed", failed)))
		}

		_, _ = fmt.Fprintf(r.w, "%s %s\n", r.statusIcon(StatusAvailable), strings.Join(parts, ", "))
	}

	// Non-quiet detail lists the pending and skipped tools (D4), and failed diagnostics in verbose mode.
	if !r.quiet {
		for _, res := range results {
			if res.Status == StatusAvailable || res.Status == StatusSkipped {
				_, _ = fmt.Fprintf(r.w, "  %s %s %s\n",
					r.statusIcon(res.Status),
					r.cyan(res.Name),
					r.dim(res.Version))
			} else if res.Status == StatusFailed && r.verbose && res.Stderr != "" {
				for _, line := range strings.Split(strings.TrimSpace(res.Stderr), "\n") {
					_, _ = fmt.Fprintf(r.w, "    %s %s\n", r.dim("│"), r.dim(line))
				}
			}
		}
	}
}

// --- List output ---

// ListTools renders a table of detected tools.
func (r *Renderer) ListTools(tools []ListEntry) {
	w := tabwriter.NewWriter(r.w, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
		r.cyan("ID"), "Name", "Status", "Version")

	for _, t := range tools {
		status := r.statusLabel(t.Status)
		version := t.Version
		if version == "" {
			version = "-"
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			t.ID, t.Name, status, version)
	}
	_ = w.Flush()
}

// ListEntry holds data for a single tool in the list output.
type ListEntry struct {
	ID      string // --only/--skip filter ID (e.g. "apt", "brew")
	Name    string
	Status  Status
	Version string
}

// --- Dashboard output ---

// DashboardData holds info for rendering the bare upp welcome dashboard.
type DashboardData struct {
	Version        string
	Platform       string
	EnabledTools   int
	AvailableTools int
}

// Dashboard renders the bare upp welcome dashboard.
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

// DashboardNoConfig renders the bare upp guidance when no config exists.
func (r *Renderer) DashboardNoConfig(version, platform string) {
	if r.quiet {
		return
	}
	_, _ = fmt.Fprintf(r.w, "%s upp %s (%s)\n\n", r.cyan("●"), version, platform)
	_, _ = fmt.Fprintln(r.w, "  No configuration found.")
	_, _ = fmt.Fprintln(r.w, "  Run \"upp init\" to detect installed tools and initialize your config.")
}

// UpdateCancelled prints the fixed message when the user cancels an
// interactive update run (design D8): nothing was updated, exit 0.
// US spelling is intentional: the repo lint (golangci-lint misspell,
// locale US) rejects the British double-L spelling.
func (r *Renderer) UpdateCancelled() {
	_, _ = fmt.Fprintln(r.w, "Update canceled — no changes made.")
}

// --- Dry run ---

// DryRunHeader prints the dry-run header.
func (r *Renderer) DryRunHeader() {
	_, _ = fmt.Fprintf(r.w, "%s Dry run — no changes will be made\n\n", r.statusIcon(StatusAvailable))
}

// DryRunPlanned prints a planned action for dry-run mode.
func (r *Renderer) DryRunPlanned(name string) {
	_, _ = fmt.Fprintf(r.w, "  %s %s\n", r.statusIcon(StatusAvailable), r.cyan(name))
}

// --- Init output ---

// InitHeader prints the init wizard header.
func (r *Renderer) InitHeader() {
	_, _ = fmt.Fprintln(r.w, "upp init — detecting installed tools...")
	_, _ = fmt.Fprintln(r.w)
}

// InitDetected prints a detected tool.
func (r *Renderer) InitDetected(name string) {
	_, _ = fmt.Fprintf(r.w, "  %s %s\n", r.statusIcon(StatusCurrent), r.cyan(name))
}

// InitConfigGenerated prints the config generated message.
func (r *Renderer) InitConfigGenerated(path string) {
	_, _ = fmt.Fprintln(r.w)
	_, _ = fmt.Fprintf(r.w, "%s Config written to %s\n", r.statusIcon(StatusUpdated), path)
}

// --- Self-update output ---

// SelfUpdatePrompt prints the pre-replace confirmation (design D8):
// current → latest versions and the resolved binary path, then the
// Proceed question. It is never suppressed by quiet mode (spec flag
// semantics: --quiet must not hide the confirm prompt).
func (r *Renderer) SelfUpdatePrompt(current, latest, target string) {
	_, _ = fmt.Fprintf(r.w, "  Update upp from %s to %s?\n", current, latest)
	_, _ = fmt.Fprintf(r.w, "    Target: %s\n", target)
	_, _ = fmt.Fprint(r.w, "  Proceed? [y/N] ")
}

// SelfUpdateDevBuild prints the development-build message (spec R1:
// exit 0, no update claim). Always shown, including quiet mode.
func (r *Renderer) SelfUpdateDevBuild() {
	_, _ = fmt.Fprintln(r.w, "development build; self-update is only available for release builds")
}

// SelfUpdateUpToDate prints the already-up-to-date message with the
// current release tag (spec R1). Always shown.
func (r *Renderer) SelfUpdateUpToDate(tag string) {
	_, _ = fmt.Fprintf(r.w, "already up to date (%s)\n", tag)
}

// SelfUpdateDone prints the replacement success line: current → latest.
// Always shown.
func (r *Renderer) SelfUpdateDone(current, latest string) {
	_, _ = fmt.Fprintf(r.w, "upp updated: %s → %s\n", current, latest)
}

// SelfUpdateHint appends the opt-in update-detection hint (design D9):
// exactly one line, after the check summary. It is the one piece of
// self-update output that quiet mode DOES suppress — the hint is
// informational output, unlike the confirm prompt.
func (r *Renderer) SelfUpdateHint(current, latest string) {
	if r.quiet {
		return
	}
	// The template leads with the latest version: "⬆️ upp v{latest}
	// available (current {current})" (spec ux-patterns).
	_, _ = fmt.Fprintf(r.w, "⬆️ upp %s available (current %s) — run \"upp self-update\"\n", latest, current)
}

// --- Error output ---

// Error prints a prefixed error message.
func (r *Renderer) Error(msg string) {
	_, _ = fmt.Fprintf(r.w, "%s %s\n", r.statusIcon(StatusFailed), r.red(msg))
}

// Warning prints a warning message.
func (r *Renderer) Warning(msg string) {
	_, _ = fmt.Fprintf(r.w, "⚠️  %s\n", r.yellow(msg))
}

// --- Helpers ---

func countByStatus(results []ToolResult) (updated, skipped, failed int) {
	for _, r := range results {
		switch r.Status {
		case StatusUpdated:
			updated++
		case StatusSkipped:
			skipped++
		case StatusFailed:
			failed++
		}
	}
	return
}

func countByStatusType(results []ToolResult, status Status) int {
	count := 0
	for _, r := range results {
		if r.Status == status {
			count++
		}
	}
	return count
}

func filterByStatus(results []ToolResult, status Status) []ToolResult {
	var filtered []ToolResult
	for _, r := range results {
		if r.Status == status {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

func toolNames(results []ToolResult) []string {
	var names []string
	for _, r := range results {
		names = append(names, r.Name)
	}
	return names
}

// --- Adapter-based helpers ---

// StatusFromResult maps an adapters.Result to a Status.
func StatusFromResult(result adapters.Result) Status {
	if result.Success {
		return StatusUpdated
	}
	return StatusFailed
}
