// Package output handles terminal rendering with color, emoji, and
// graceful degradation for pipes and dumb terminals.
package output

import (
	"fmt"
	"io"
	"os"
	"strings"
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
}

// Summary holds the complete results of an update or check run.
type Summary struct {
	Results []ToolResult
	DryRun  bool
}

// Renderer handles formatted terminal output.
type Renderer struct {
	w     io.Writer
	color bool
	emoji bool
	quiet bool
}

// NewRenderer creates a Renderer that detects color/emoji support.
func NewRenderer(w io.Writer, quiet bool) *Renderer {
	color := isTerminal(w)
	return &Renderer{
		w:     w,
		color: color,
		emoji: color, // emoji follows color support
		quiet: quiet,
	}
}

// NewRendererForced creates a Renderer with explicit color/emoji settings.
func NewRendererForced(w io.Writer, color, emoji, quiet bool) *Renderer {
	return &Renderer{
		w:     w,
		color: color,
		emoji: emoji,
		quiet: quiet,
	}
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

// Progress shows a progress indicator for multi-tool operations.
func (r *Renderer) Progress(current, total int, name string) {
	if r.quiet || total <= 1 {
		return
	}
	_, _ = fmt.Fprintf(r.w, "  %s Updating %d/%d: %s\n",
		r.dim("⟳"), current, total, r.cyan(name))
}

// --- Summary ---

// UpdateSummary renders the final summary of an update or check run.
func (r *Renderer) UpdateSummary(summary Summary) {
	updated, skipped, failed := countByStatus(summary.Results)

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
	if skipped > 0 {
		parts = append(parts, fmt.Sprintf("%d skipped", skipped))
	}
	if failed > 0 {
		parts = append(parts, r.red(fmt.Sprintf("%d failed", failed)))
	}

	_, _ = fmt.Fprintln(r.w)

	// All skipped (or empty) → special message
	if updated == 0 && available == 0 && failed == 0 {
		_, _ = fmt.Fprintf(r.w, "%s All tools not installed. Nothing to do.\n", r.statusIcon(StatusSkipped))
		return
	}

	summaryLine := strings.Join(parts, ", ")

	if failed > 0 {
		_, _ = fmt.Fprintf(r.w, "%s %s. Review errors above.\n", r.statusIcon(StatusFailed), summaryLine)
	} else if updated > 0 || available > 0 {
		_, _ = fmt.Fprintf(r.w, "%s %s. All clean!\n", r.statusIcon(StatusUpdated), summaryLine)
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
	skipped := filterByStatus(summary.Results, StatusSkipped)
	failed := filterByStatus(summary.Results, StatusFailed)

	if len(updated) > 0 {
		ids := toolNames(updated)
		_, _ = fmt.Fprintf(r.w, "  %s %s\n", r.green("Updated:"), strings.Join(ids, ", "))
	}
	if len(skipped) > 0 {
		ids := toolNames(skipped)
		_, _ = fmt.Fprintf(r.w, "  Skipped: %s\n", strings.Join(ids, ", "))
	}
	if len(failed) > 0 {
		ids := toolNames(failed)
		_, _ = fmt.Fprintf(r.w, "  %s %s\n", r.red("Failed:"), strings.Join(ids, ", "))
	}
}

// CheckSummary renders the summary for a check (read-only) operation.
func (r *Renderer) CheckSummary(results []ToolResult) {
	var available, current, failed int
	for _, res := range results {
		switch res.Status {
		case StatusAvailable:
			available++
		case StatusCurrent:
			current++
		case StatusFailed:
			failed++
		}
	}

	_, _ = fmt.Fprintln(r.w)

	if available == 0 && failed == 0 {
		_, _ = fmt.Fprintf(r.w, "%s All tools up to date.\n", r.statusIcon(StatusCurrent))
		return
	}

	var parts []string
	if available > 0 {
		parts = append(parts, r.yellow(fmt.Sprintf("%d available", available)))
	}
	if current > 0 {
		parts = append(parts, fmt.Sprintf("%d current", current))
	}
	if failed > 0 {
		parts = append(parts, r.red(fmt.Sprintf("%d failed", failed)))
	}

	_, _ = fmt.Fprintf(r.w, "%s %s\n", r.statusIcon(StatusAvailable), strings.Join(parts, ", "))

	if !r.quiet {
		for _, res := range results {
			if res.Status == StatusAvailable {
				_, _ = fmt.Fprintf(r.w, "  %s %s %s\n",
					r.statusIcon(StatusAvailable),
					r.cyan(res.Name),
					r.dim(res.Version))
			}
		}
	}
}

// --- List output ---

// ListTools renders a table of detected tools.
func (r *Renderer) ListTools(tools []ListEntry) {
	w := tabwriter.NewWriter(r.w, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
		r.cyan("Tool"), "Name", "Status", "Version")

	for _, t := range tools {
		icon := r.statusIcon(t.Status)
		status := r.statusLabel(t.Status)
		version := t.Version
		if version == "" {
			version = "-"
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			icon, t.Name, status, version)
	}
	_ = w.Flush()
}

// ListEntry holds data for a single tool in the list output.
type ListEntry struct {
	Name    string
	Status  Status
	Version string
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
	_, _ = fmt.Fprintf(r.w, "  %s\n", fmt.Sprintf("Update upp from %s to %s?", current, latest))
	_, _ = fmt.Fprintf(r.w, "    %s\n", fmt.Sprintf("Target: %s", target))
	_, _ = fmt.Fprintf(r.w, "  %s", "Proceed? [y/N] ")
}

// SelfUpdateDevBuild prints the development-build message (spec R1:
// exit 0, no update claim). Always shown, including quiet mode.
func (r *Renderer) SelfUpdateDevBuild() {
	_, _ = fmt.Fprintln(r.w, "development build; self-update is only available for release builds")
}

// SelfUpdateUpToDate prints the already-up-to-date message with the
// current release tag (spec R1). Always shown.
func (r *Renderer) SelfUpdateUpToDate(tag string) {
	_, _ = fmt.Fprintln(r.w, fmt.Sprintf("already up to date (%s)", tag))
}

// SelfUpdateDone prints the replacement success line: current → latest.
// Always shown.
func (r *Renderer) SelfUpdateDone(current, latest string) {
	_, _ = fmt.Fprintln(r.w, fmt.Sprintf("upp updated: %s → %s", current, latest))
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
	_, _ = fmt.Fprintln(r.w, fmt.Sprintf("⬆️ upp %s available (current %s) — run \"upp self-update\"", latest, current))
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
