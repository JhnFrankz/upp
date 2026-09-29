// Package output handles terminal rendering with color, emoji, and
// graceful degradation for pipes and dumb terminals.
package output

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
	"text/tabwriter"

	"github.com/JhnFrankz/upp/internal/doctor"
	"github.com/JhnFrankz/upp/internal/uninstall"
)

// Status represents the outcome of a tool operation.
type Status int

const (
	StatusUpdated Status = iota
	StatusSkipped
	StatusFailed
	StatusAvailable
	StatusCurrent
	// StatusDeselected marks a tool that was pending (planned for update) but
	// left unselected by the user. It is distinct from StatusSkipped (not
	// installed / confirm-deny) and from StatusCurrent, so a deselection is
	// reported rather than silently dropped (design D4, PC2).
	StatusDeselected
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

var isTerminalFn = isTerminal

// NewRenderer creates a Renderer that detects color/emoji support.
func NewRenderer(w io.Writer, quiet bool) *Renderer {
	return NewRendererVerbose(w, quiet, false)
}

// NewRendererVerbose creates a Renderer with explicit verbose setting.
func NewRendererVerbose(w io.Writer, quiet, verbose bool) *Renderer {
	color := isTerminalFn(w)
	emoji := color // emoji follows color support
	noColor := os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb"
	if noColor {
		color = false
		emoji = false
	}
	return &Renderer{
		w:       w,
		color:   color,
		emoji:   emoji,
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
	case StatusDeselected:
		return "☐ "
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
	case StatusDeselected:
		return "[deselected]"
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
	case StatusDeselected:
		return "deselected"
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
			rem := strings.TrimSpace(result.Stderr)
			for len(rem) > 0 {
				var line string
				line, rem, _ = strings.Cut(rem, "\n")
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
	case StatusDeselected:
		_, _ = fmt.Fprintf(r.w, "  %s %s (deselected)\n", icon, name)
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

// Summary renders the final summary of an update or check run.
func (r *Renderer) Summary(summary Summary) {
	r.UpdateSummary(summary)
}

// UpdateSummary renders the final summary of an update or check run.
func (r *Renderer) UpdateSummary(summary Summary) {
	updated, skipped, failed := countByStatus(summary.Results)
	current := countByStatusType(summary.Results, StatusCurrent)
	deselected := countByStatusType(summary.Results, StatusDeselected)

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
	if deselected > 0 {
		parts = append(parts, fmt.Sprintf("%d deselected", deselected))
	}
	if failed > 0 {
		parts = append(parts, r.red(fmt.Sprintf("%d failed", failed)))
	}

	var sb strings.Builder
	sb.Grow(len(summary.Results)*48 + 128)
	sb.WriteByte('\n')

	// All skipped (or empty) → special message. Current and deselected tools
	// ARE installed/pending, so they keep this branch from firing (D6, PC2).
	if updated == 0 && available == 0 && failed == 0 && current == 0 && deselected == 0 {
		sb.WriteString(r.statusIcon(StatusSkipped))
		sb.WriteString(" All tools not installed. Nothing to do.\n")
		_, _ = io.WriteString(r.w, sb.String())
		return
	}

	summaryLine := strings.Join(parts, ", ")

	// A run is only "clean" when it really updated something, nothing is
	// pending, nothing failed, and nothing was skipped or deselected. A
	// --dry-run with pending updates reports "N would update" and never claims
	// "All clean!" (D3); a deselected pending tool is outstanding work too.
	allClean := !summary.DryRun && updated > 0 && available == 0 && failed == 0 && skipped == 0 && deselected == 0

	if failed > 0 {
		sb.WriteString(r.statusIcon(StatusFailed))
		sb.WriteByte(' ')
		sb.WriteString(summaryLine)
		sb.WriteString(". Review errors above.\n")
	} else if allClean {
		// Spec ux-patterns Summary Report "All succeed": the clean line
		// counts failures explicitly even when zero ("N updated, 0 failed").
		sb.WriteString(r.statusIcon(StatusUpdated))
		sb.WriteByte(' ')
		sb.WriteString(summaryLine)
		sb.WriteString(", 0 failed. All clean!\n")
	} else if updated > 0 || available > 0 {
		sb.WriteString(r.statusIcon(StatusUpdated))
		sb.WriteByte(' ')
		sb.WriteString(summaryLine)
		sb.WriteByte('\n')
	} else if deselected > 0 {
		// All pending work was deselected: report it under the deselected icon,
		// never as current.
		sb.WriteString(r.statusIcon(StatusDeselected))
		sb.WriteByte(' ')
		sb.WriteString(summaryLine)
		sb.WriteByte('\n')
	} else {
		sb.WriteString(r.statusIcon(StatusCurrent))
		sb.WriteByte(' ')
		sb.WriteString(summaryLine)
		sb.WriteByte('\n')
	}

	// List tools per category in non-quiet mode
	if !r.quiet {
		r.detailSummary(&sb, summary)
	}
	_, _ = io.WriteString(r.w, sb.String())
}

func writeJoinedNames(sb *strings.Builder, results []ToolResult) {
	for i, r := range results {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(r.Name)
	}
}

func hasStatus(results []ToolResult, status Status) bool {
	for i := range results {
		if results[i].Status == status {
			return true
		}
	}
	return false
}

func writeJoinedNamesForStatus(sb *strings.Builder, results []ToolResult, status Status) bool {
	first := true
	for i := range results {
		if results[i].Status == status {
			if !first {
				sb.WriteString(", ")
			}
			sb.WriteString(results[i].Name)
			first = false
		}
	}
	return !first
}

func (r *Renderer) detailSummary(sb *strings.Builder, summary Summary) {
	if hasStatus(summary.Results, StatusUpdated) {
		sb.WriteString("  ")
		sb.WriteString(r.green("Updated:"))
		sb.WriteByte(' ')
		writeJoinedNamesForStatus(sb, summary.Results, StatusUpdated)
		sb.WriteByte('\n')
	}
	if hasStatus(summary.Results, StatusCurrent) {
		sb.WriteString("  ")
		sb.WriteString(r.green("Up to date:"))
		sb.WriteByte(' ')
		writeJoinedNamesForStatus(sb, summary.Results, StatusCurrent)
		sb.WriteByte('\n')
	}
	if hasStatus(summary.Results, StatusSkipped) {
		sb.WriteString("  Skipped: ")
		writeJoinedNamesForStatus(sb, summary.Results, StatusSkipped)
		sb.WriteByte('\n')
	}
	if hasStatus(summary.Results, StatusDeselected) {
		sb.WriteString("  Deselected: ")
		writeJoinedNamesForStatus(sb, summary.Results, StatusDeselected)
		sb.WriteByte('\n')
	}
	if hasStatus(summary.Results, StatusFailed) {
		sb.WriteString("  ")
		sb.WriteString(r.red("Failed:"))
		sb.WriteByte(' ')
		writeJoinedNamesForStatus(sb, summary.Results, StatusFailed)
		sb.WriteByte('\n')
		if r.verbose && !r.quiet {
			for i := range summary.Results {
				if summary.Results[i].Status == StatusFailed && summary.Results[i].Stderr != "" {
					rem := strings.TrimSpace(summary.Results[i].Stderr)
					for len(rem) > 0 {
						var line string
						line, rem, _ = strings.Cut(rem, "\n")
						sb.WriteString("    ")
						sb.WriteString(r.dim("│"))
						sb.WriteByte(' ')
						sb.WriteString(r.dim(line))
						sb.WriteByte('\n')
					}
				}
			}
		}
	}
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

// NoToolsConfigured prints the empty-list message for `upp list` when no
// tools are configured. Suppressed under --quiet like other list status
// output.
func (r *Renderer) NoToolsConfigured() {
	if r.quiet {
		return
	}
	_, _ = fmt.Fprintln(r.w, "no tools configured.")
}

// NoToolsMatchFilter prints the --only filter mismatch message for `upp
// list`: the filter matched none of the configured tools. Suppressed under
// --quiet like other list status output.
func (r *Renderer) NoToolsMatchFilter(filter string) {
	if r.quiet {
		return
	}
	_, _ = fmt.Fprintf(r.w, "no tools match --only filter: %s\n", filter)
}

// ListEntry holds data for a single tool in the list output.
type ListEntry struct {
	ID      string // --only filter ID (e.g. "apt", "brew")
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
	_, _ = fmt.Fprintf(r.w, "    %-14s %s\n", "upp doctor", "Run environment and health diagnostics")
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

// UninstallPlan renders the list of targets to be removed before interactive confirmation.
func (r *Renderer) UninstallPlan(targets []uninstall.Target) {
	_, _ = fmt.Fprintln(r.w, "The following targets will be removed:")
	hasExisting := false
	for _, t := range targets {
		if t.Exists {
			hasExisting = true
			break
		}
	}
	for _, t := range targets {
		if hasExisting && !t.Exists {
			continue
		}
		typeName := string(t.Type)
		if len(typeName) > 0 {
			typeName = strings.ToUpper(typeName[:1]) + typeName[1:]
		}
		_, _ = fmt.Fprintf(r.w, "  - %s: %s\n", typeName, t.Path)
	}
	_, _ = fmt.Fprintln(r.w)
}

// UninstallCanceled prints the cancellation message when the user declines uninstallation.
func (r *Renderer) UninstallCanceled() {
	_, _ = fmt.Fprintln(r.w, "Uninstall canceled — no files were removed.")
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

func (r *Renderer) doctorIcon(s doctor.Severity) string {
	switch s {
	case doctor.SeverityOK:
		return r.green("[✓]")
	case doctor.SeverityWarn:
		return r.yellow("[!]")
	case doctor.SeverityError:
		return r.red("[✗]")
	default:
		return "[?]"
	}
}

// DoctorResults renders diagnostics results grouped by category.
func (r *Renderer) DoctorResults(results []doctor.CheckResult, quiet, verbose bool) {
	hasErrors := doctor.HasErrors(results)
	hasWarnings := doctor.HasWarnings(results)

	if quiet && !hasErrors && !hasWarnings {
		_, _ = io.WriteString(r.w, "upp doctor: all checks passed.\n")
		return
	}

	displayResults := results
	if quiet {
		var filtered []doctor.CheckResult
		for _, res := range results {
			if res.Status != doctor.SeverityOK {
				filtered = append(filtered, res)
			}
		}
		displayResults = filtered
	}

	var categoryOrder []string
	categoryMap := make(map[string][]doctor.CheckResult)
	for _, res := range displayResults {
		cat := res.Category
		if cat == "" {
			cat = "General"
		}
		if _, exists := categoryMap[cat]; !exists {
			categoryOrder = append(categoryOrder, cat)
		}
		categoryMap[cat] = append(categoryMap[cat], res)
	}

	var sb strings.Builder
	sb.Grow(len(displayResults)*96 + 256)

	firstCategory := true
	for _, cat := range categoryOrder {
		if !firstCategory {
			sb.WriteByte('\n')
		}
		firstCategory = false
		sb.WriteString(cat)
		sb.WriteByte('\n')

		items := categoryMap[cat]
		for _, item := range items {
			icon := r.doctorIcon(item.Status)
			sb.WriteString("  ")
			sb.WriteString(icon)
			sb.WriteByte(' ')
			if item.Name != "" {
				sb.WriteString(item.Name)
				sb.WriteString(": ")
				sb.WriteString(item.Message)
				sb.WriteByte('\n')
			} else {
				sb.WriteString(item.Message)
				sb.WriteByte('\n')
			}

			if verbose && item.Detail != "" {
				rem := item.Detail
				for len(rem) > 0 {
					var line string
					line, rem, _ = strings.Cut(rem, "\n")
					if line != "" {
						sb.WriteString("    ")
						sb.WriteString(line)
						sb.WriteByte('\n')
					}
				}
			}

			if item.FixHint != "" {
				sb.WriteString("    Hint: ")
				sb.WriteString(item.FixHint)
				sb.WriteByte('\n')
			}
		}
	}

	passedCount := 0
	warnCount := 0
	errCount := 0
	for _, res := range results {
		switch res.Status {
		case doctor.SeverityOK:
			passedCount++
		case doctor.SeverityWarn:
			warnCount++
		case doctor.SeverityError:
			errCount++
		}
	}

	sb.WriteString("\nResults: ")
	sb.WriteString(strconv.Itoa(passedCount))
	sb.WriteString(" passed, ")
	sb.WriteString(strconv.Itoa(warnCount))
	sb.WriteString(" warnings, ")
	sb.WriteString(strconv.Itoa(errCount))
	sb.WriteString(" errors.\n")
	_, _ = io.WriteString(r.w, sb.String())
}
