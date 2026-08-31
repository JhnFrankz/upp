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
		// Spec ux-patterns Summary Report "All succeed": the clean line
		// counts failures explicitly even when zero ("N updated, 0 failed").
		_, _ = fmt.Fprintf(r.w, "%s %s, 0 failed. All clean!\n", r.statusIcon(StatusUpdated), summaryLine)
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

// --- Group bulk summary (WU5) ---

// GroupBulkSummary holds the results of a manager-group bulk update (spec
// ux-patterns Summary Report: each owned tool updated, skipped, current, or
// failed within the group). Results MUST be in canonical discovery order —
// the renderer never reorders by completion or groups by status category, so
// out-of-order concurrent completion cannot reorder the report (deterministic
// order rule).
type GroupBulkSummary struct {
	// Manager is the owning manager's display label for the group header.
	Manager string
	// DryRun indicates a --dry-run group: pending tools report "would update".
	DryRun bool
	// Results are the group batch outcomes in canonical discovery order.
	Results []ToolResult
}

// GroupBulkSummary renders the group bulk summary: a manager header, then one
// line per owned tool in canonical order describing its outcome (updated /
// skipped / current / failed / would update). It does NOT print "All tools up
// to date." when any tool in the batch was skipped or unchecked, and it never
// pairs "All clean!" with a pending update (spec ux-patterns Summary Report).
func (r *Renderer) GroupBulkSummary(summary GroupBulkSummary) {
	_, _ = fmt.Fprintln(r.w)

	if summary.Manager != "" {
		_, _ = fmt.Fprintf(r.w, "  %s\n", r.cyan(summary.Manager))
	}

	for _, res := range summary.Results {
		label := r.statusLabel(res.Status)
		if res.Status == StatusAvailable && summary.DryRun {
			label = "would update"
		}
		if res.Status == StatusFailed {
			errMsg := ""
			if res.Error != nil {
				errMsg = " (" + res.Error.Error() + ")"
			}
			_, _ = fmt.Fprintf(r.w, "    %s %s %s%s\n", r.statusIcon(StatusFailed), r.red(res.Name), "failed", errMsg)
			continue
		}
		version := ""
		if res.Version != "" {
			version = " " + r.dim(res.Version)
		}
		_, _ = fmt.Fprintf(r.w, "    %s %s %s%s\n", r.statusIcon(res.Status), r.cyan(res.Name), label, version)
	}

	// Deterministic aggregate line (spec ux-patterns Summary Report): counts
	// every category explicitly; never claims "All clean!" on a dry-run.
	updated, skipped, failed := countByStatus(summary.Results)
	current := countByStatusType(summary.Results, StatusCurrent)
	available := 0
	if summary.DryRun {
		available = countByStatusType(summary.Results, StatusAvailable)
	}

	var parts []string
	if updated > 0 || (summary.DryRun && available > 0) {
		label := "updated"
		if summary.DryRun {
			label = "would update"
		}
		count := updated + available
		if summary.DryRun {
			count = available // only the pending tools "would update" on a dry-run
		}
		parts = append(parts, r.green(fmt.Sprintf("%d %s", count, label)))
	}
	if current > 0 {
		parts = append(parts, fmt.Sprintf("%d current", current))
	}
	if skipped > 0 {
		parts = append(parts, fmt.Sprintf("%d skipped", skipped))
	}
	if failed > 0 {
		parts = append(parts, r.red(fmt.Sprintf("%d failed", failed)))
	}

	if len(parts) == 0 {
		_, _ = fmt.Fprintf(r.w, "  %s No owned tools in the group. Nothing to do.\n", r.statusIcon(StatusSkipped))
		return
	}
	summaryLine := strings.Join(parts, ", ")
	allClean := !summary.DryRun && updated == 0 && failed == 0 && skipped == 0 && current > 0
	if failed > 0 {
		_, _ = fmt.Fprintf(r.w, "  %s %s. Review errors above.\n", r.statusIcon(StatusFailed), summaryLine)
	} else if allClean {
		_, _ = fmt.Fprintf(r.w, "  %s %s, 0 failed. All clean!\n", r.statusIcon(StatusUpdated), summaryLine)
	} else {
		_, _ = fmt.Fprintf(r.w, "  %s %s\n", r.statusIcon(StatusUpdated), summaryLine)
	}
}

// GroupBatchTool is a single owned tool in the planned manager-group batch,
// rendered BEFORE any update executes (spec ux-patterns Opt-In Flag UX).
type GroupBatchTool struct {
	// Name is the owned tool's display name.
	Name string
	// UpdateAvailable indicates the tool's package has an update pending (a
	// Gated manager will update it; an AlwaysUpdate manager runs regardless).
	UpdateAvailable bool
	// Version is a "current → latest" display when UpdateAvailable is true.
	Version string
	// Skipped indicates the tool was excluded from the batch via --skip.
	Skipped bool
	// CheckFailed indicates the per-package availability check errored; the
	// tool is reported "check failed", never current nor update available.
	CheckFailed bool
}

// GroupBatchPreview is the planned manager-group bulk batch shown before
// execution: the manager header, each owned tool and its planned state, and
// whether the batch is gated.
type GroupBatchPreview struct {
	// Manager is the owning manager's display label.
	Manager string
	// Gated indicates the batch inherits a PolicyGated manager (apt): owned
	// tools update only when their package has an update. An AlwaysUpdate
	// manager (brew/winget/scoop) is ungated and runs regardless.
	Gated bool
	// Tools is the planned batch in canonical discovery order.
	Tools []GroupBatchTool
}

// GroupBatchPreview renders the planned manager-group batch before execution:
// a manager header line, one line per owned tool describing its planned state
// (update available / current / skipped via --skip / check failed), and a
// "Batch gated: yes|no" line. It never claims "All clean!" — this is a
// PRE-execution plan, not a result.
func (r *Renderer) GroupBatchPreview(preview GroupBatchPreview) {
	_, _ = fmt.Fprintln(r.w)

	if preview.Manager != "" {
		_, _ = fmt.Fprintf(r.w, "  %s (manager)\n", r.cyan(preview.Manager))
	}

	for _, t := range preview.Tools {
		switch {
		case t.CheckFailed:
			_, _ = fmt.Fprintf(r.w, "    %s %s (check failed)\n", r.statusIcon(StatusFailed), r.cyan(t.Name))
		case t.Skipped:
			_, _ = fmt.Fprintf(r.w, "    %s %s (skipped via --skip)\n", r.statusIcon(StatusSkipped), r.cyan(t.Name))
		case t.UpdateAvailable:
			_, _ = fmt.Fprintf(r.w, "    %s %s (update available)\n", r.statusIcon(StatusAvailable), r.cyan(t.Name))
			if t.Version != "" {
				_, _ = fmt.Fprintf(r.w, "      %s\n", r.dim(t.Version))
			}
		default:
			_, _ = fmt.Fprintf(r.w, "    %s %s (current)\n", r.statusIcon(StatusCurrent), r.cyan(t.Name))
		}
	}

	gated := "no"
	if preview.Gated {
		gated = "yes"
	}
	_, _ = fmt.Fprintf(r.w, "  %s Batch gated: %s\n", r.statusIcon(StatusSkipped), gated)
}

// --- List output ---

// ListTools renders a table of detected tools, grouped by owning manager
// when groups carry a non-empty Header (design: group rendering/wiring).
// Each group prints its manager header line once, then its child rows indented
// beneath it (the manager's own row leads, followed by the tools it owns).
// A standalone group (empty Header) prints only its rows.
func (r *Renderer) ListTools(groups []Group) {
	w := tabwriter.NewWriter(r.w, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
		r.cyan("ID"), "Name", "Status", "Version")

	for _, g := range groups {
		if g.Header != "" {
			_, _ = fmt.Fprintf(w, "%s\n", g.Header)
		}
		for _, t := range g.Items {
			status := r.statusLabel(t.Status)
			version := t.Version
			if version == "" {
				version = "-"
			}
			_, _ = fmt.Fprintf(w, "  %s\t%s\t%s\t%s\n",
				t.ID, t.Name, status, version)
		}
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
	_, _ = fmt.Fprintf(r.w, "    %-14s %s\n", "upp update -n", "Preview pending updates (--dry-run)")
	_, _ = fmt.Fprintf(r.w, "    %-14s %s\n", "upp update", "Apply updates to all enabled tools")
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

// --- Uninstall output ---

// UninstallDryRunHeader prints the dry-run header for uninstallation.
func (r *Renderer) UninstallDryRunHeader() {
	_, _ = fmt.Fprintf(r.w, "%s Dry run — no files will be removed\n\n", r.statusIcon(StatusAvailable))
}

// UninstallDryRunTarget prints a planned target to be removed during dry-run.
func (r *Renderer) UninstallDryRunTarget(targetType, path string) {
	_, _ = fmt.Fprintf(r.w, "  %s Would remove %s: %s\n", r.statusIcon(StatusAvailable), targetType, r.cyan(path))
}

// UninstallRemoved prints a successfully removed target.
func (r *Renderer) UninstallRemoved(targetType, path string) {
	if r.quiet {
		return
	}
	_, _ = fmt.Fprintf(r.w, "  %s Removed %s: %s\n", r.statusIcon(StatusUpdated), targetType, path)
}

// UninstallDone prints the uninstallation completion message.
func (r *Renderer) UninstallDone() {
	_, _ = fmt.Fprintln(r.w, "upp has been successfully uninstalled.")
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
