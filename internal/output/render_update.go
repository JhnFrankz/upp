package output

import (
	"fmt"
	"io"
	"strconv"
	"strings"
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
	// StatusDenied marks a tool whose update was planned but denied at the
	// confirmation prompt. It is distinct from StatusSkipped (not installed),
	// ensuring an installed tool whose update is denied is not falsely reported
	// as uninstalled.
	StatusDenied
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

// ProgressPresenter defines the contract for rendering progress indicators during multi-tool operations.
type ProgressPresenter interface {
	Progress(op string, current, total int, name string)
}

// UpdatePresenter defines the contract for rendering tool update execution and summaries.
type UpdatePresenter interface {
	ProgressPresenter
	ToolLine(result ToolResult)
	UpdateSummary(summary Summary)
	UpdateCancelled()
	DryRunHeader()
	DryRunPlanned(name string)
}

var (
	_ ProgressPresenter = (*Renderer)(nil)
	_ UpdatePresenter   = (*Renderer)(nil)
)

// --- Tool result rendering ---

// ToolLine renders a single tool result line.
func (r *Renderer) ToolLine(result ToolResult) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.quiet {
		r.quietToolLine(result)
		return
	}
	r.verboseToolLine(result)
}

func (r *Renderer) verboseToolLine(result ToolResult) {
	r.scratch.Reset()
	icon := r.statusIcon(result.Status)

	switch result.Status {
	case StatusUpdated, StatusAvailable, StatusCurrent:
		r.scratch.WriteString("  ")
		r.scratch.WriteString(icon)
		r.scratch.WriteByte(' ')
		r.writeColor(&r.scratch, "36", result.Name)
		if result.Version != "" {
			r.scratch.WriteByte(' ')
			r.writeColor(&r.scratch, "2", result.Version)
		}
		r.scratch.WriteByte('\n')
	case StatusSkipped:
		r.scratch.WriteString("  ")
		r.scratch.WriteString(icon)
		r.scratch.WriteByte(' ')
		r.writeColor(&r.scratch, "36", result.Name)
		r.scratch.WriteString(" (not installed)\n")
	case StatusFailed:
		r.scratch.WriteString("  ")
		r.scratch.WriteString(icon)
		r.scratch.WriteByte(' ')
		r.writeColor(&r.scratch, "31", result.Name)
		if result.Error != nil {
			r.scratch.WriteString(" (")
			r.scratch.WriteString(result.Error.Error())
			r.scratch.WriteByte(')')
		}
		r.scratch.WriteByte('\n')
		if r.verbose && !r.quiet && result.Stderr != "" {
			rem := strings.TrimSpace(result.Stderr)
			for len(rem) > 0 {
				var line string
				line, rem, _ = strings.Cut(rem, "\n")
				r.scratch.WriteString("    ")
				r.writeColor(&r.scratch, "2", "│")
				r.scratch.WriteByte(' ')
				r.writeColor(&r.scratch, "2", line)
				r.scratch.WriteByte('\n')
			}
		}
	case StatusDeselected:
		r.scratch.WriteString("  ")
		r.scratch.WriteString(icon)
		r.scratch.WriteByte(' ')
		r.writeColor(&r.scratch, "36", result.Name)
		r.scratch.WriteString(" (deselected)\n")
	case StatusDenied:
		r.scratch.WriteString("  ")
		r.scratch.WriteString(icon)
		r.scratch.WriteByte(' ')
		r.writeColor(&r.scratch, "36", result.Name)
		r.scratch.WriteString(" (denied)\n")
	default:
		return
	}
	_, _ = io.WriteString(r.w, r.scratch.String())
}

func (r *Renderer) quietToolLine(result ToolResult) {
	icon := r.statusIcon(result.Status)
	r.scratch.Reset()
	r.scratch.WriteString(icon)
	r.scratch.WriteByte(' ')
	r.scratch.WriteString(result.Name)

	switch result.Status {
	case StatusFailed:
		if result.Error != nil {
			r.scratch.WriteString(": ")
			r.scratch.WriteString(result.Error.Error())
		}
	case StatusSkipped, StatusDenied:
	default:
		if result.Version != "" {
			r.scratch.WriteByte(' ')
			r.scratch.WriteString(result.Version)
		}
	}
	r.scratch.WriteByte('\n')
	_, _ = io.WriteString(r.w, r.scratch.String())
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
	r.scratch.Reset()
	r.scratch.WriteString("  ")
	r.writeColor(&r.scratch, "2", "⟳")
	r.scratch.WriteByte(' ')
	r.scratch.WriteString(op)
	r.scratch.WriteByte(' ')
	r.scratch.WriteString(strconv.Itoa(current))
	r.scratch.WriteByte('/')
	r.scratch.WriteString(strconv.Itoa(total))
	r.scratch.WriteString(": ")
	r.writeColor(&r.scratch, "36", name)
	r.scratch.WriteByte('\n')
	_, _ = io.WriteString(r.w, r.scratch.String())
}

// --- Summary ---

type summaryIndices struct {
	updated    []int
	current    []int
	skipped    []int
	deselected []int
	denied     []int
	failed     []int

	uBuf  [16]int
	cBuf  [16]int
	sBuf  [16]int
	dBuf  [8]int
	dnBuf [8]int
	fBuf  [16]int
}

// UpdateSummary renders the final summary of an update or check run.
func (r *Renderer) UpdateSummary(summary Summary) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var idx summaryIndices
	idx.updated = idx.uBuf[:0]
	idx.current = idx.cBuf[:0]
	idx.skipped = idx.sBuf[:0]
	idx.deselected = idx.dBuf[:0]
	idx.denied = idx.dnBuf[:0]
	idx.failed = idx.fBuf[:0]

	available := 0
	for i, res := range summary.Results {
		switch res.Status {
		case StatusUpdated:
			idx.updated = append(idx.updated, i)
		case StatusAvailable:
			if summary.DryRun {
				available++
			}
		case StatusCurrent:
			idx.current = append(idx.current, i)
		case StatusSkipped:
			idx.skipped = append(idx.skipped, i)
		case StatusDeselected:
			idx.deselected = append(idx.deselected, i)
		case StatusDenied:
			idx.denied = append(idx.denied, i)
		case StatusFailed:
			idx.failed = append(idx.failed, i)
		}
	}

	updated := len(idx.updated)
	current := len(idx.current)
	skipped := len(idx.skipped)
	deselected := len(idx.deselected)
	denied := len(idx.denied)
	failed := len(idx.failed)

	var sb strings.Builder
	sb.Grow(len(summary.Results)*48 + 128)
	sb.WriteByte('\n')

	// All skipped (or empty) → special message. Current, deselected, and denied tools
	// ARE installed/pending, so they keep this branch from firing (D6, PC2).
	if updated == 0 && available == 0 && failed == 0 && current == 0 && deselected == 0 && denied == 0 {
		sb.WriteString(r.statusIcon(StatusSkipped))
		sb.WriteString(" All tools not installed. Nothing to do.\n")
		_, _ = io.WriteString(r.w, sb.String())
		return
	}

	// A run is only "clean" when it really updated something, nothing is
	// pending, nothing failed, and nothing was skipped, deselected, or denied. A
	// --dry-run with pending updates reports "N would update" and never claims
	// "All clean!" (D3); a deselected or denied tool is outstanding work too.
	allClean := !summary.DryRun && updated > 0 && available == 0 && failed == 0 && skipped == 0 && deselected == 0 && denied == 0

	var icon string
	var suffix string
	if failed > 0 {
		icon = r.statusIcon(StatusFailed)
		suffix = ". Review errors above.\n"
	} else if allClean {
		// Spec ux-patterns Summary Report "All succeed": the clean line
		// counts failures explicitly even when zero ("N updated, 0 failed").
		icon = r.statusIcon(StatusUpdated)
		suffix = ", 0 failed. All clean!\n"
	} else if updated > 0 || available > 0 {
		icon = r.statusIcon(StatusUpdated)
		suffix = "\n"
	} else if deselected > 0 {
		// All pending work was deselected: report it under the deselected icon,
		// never as current.
		icon = r.statusIcon(StatusDeselected)
		suffix = "\n"
	} else if denied > 0 {
		icon = r.statusIcon(StatusDenied)
		suffix = "\n"
	} else {
		icon = r.statusIcon(StatusCurrent)
		suffix = "\n"
	}

	sb.WriteString(icon)
	sb.WriteByte(' ')

	first := true
	writeCountPart := func(count int, label, colorCode string) {
		if !first {
			sb.WriteString(", ")
		}
		first = false
		if colorCode != "" && r.color {
			sb.WriteString("\033[")
			sb.WriteString(colorCode)
			sb.WriteString("m")
			sb.WriteString(strconv.Itoa(count))
			sb.WriteByte(' ')
			sb.WriteString(label)
			sb.WriteString("\033[0m")
		} else {
			sb.WriteString(strconv.Itoa(count))
			sb.WriteByte(' ')
			sb.WriteString(label)
		}
	}

	if updated > 0 || available > 0 {
		label := "updated"
		if summary.DryRun {
			label = "would update"
		}
		writeCountPart(updated+available, label, "32")
	}
	if current > 0 {
		writeCountPart(current, "up to date", "")
	}
	if skipped > 0 {
		writeCountPart(skipped, "skipped", "")
	}
	if deselected > 0 {
		writeCountPart(deselected, "deselected", "")
	}
	if denied > 0 {
		writeCountPart(denied, "denied", "")
	}
	if failed > 0 {
		writeCountPart(failed, "failed", "31")
	}

	sb.WriteString(suffix)

	// List tools per category in non-quiet mode
	if !r.quiet {
		r.detailSummary(&sb, summary, &idx)
	}
	_, _ = io.WriteString(r.w, sb.String())
}

func (r *Renderer) writeIndexSection(sb *strings.Builder, results []ToolResult, indices []int, label string, colorCode string) bool {
	if len(indices) == 0 {
		return false
	}
	sb.WriteString("  ")
	if colorCode != "" && r.color {
		sb.WriteString("\033[")
		sb.WriteString(colorCode)
		sb.WriteString("m")
		sb.WriteString(label)
		sb.WriteString("\033[0m ")
	} else {
		sb.WriteString(label)
		sb.WriteByte(' ')
	}
	for j, idx := range indices {
		if j > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(results[idx].Name)
	}
	sb.WriteByte('\n')
	return true
}

func (r *Renderer) detailSummary(sb *strings.Builder, summary Summary, idx *summaryIndices) {
	r.writeIndexSection(sb, summary.Results, idx.updated, "Updated:", "32")
	r.writeIndexSection(sb, summary.Results, idx.current, "Up to date:", "32")
	r.writeIndexSection(sb, summary.Results, idx.skipped, "Skipped:", "")
	r.writeIndexSection(sb, summary.Results, idx.deselected, "Deselected:", "")
	r.writeIndexSection(sb, summary.Results, idx.denied, "Denied:", "")
	if r.writeIndexSection(sb, summary.Results, idx.failed, "Failed:", "31") {
		if r.verbose && !r.quiet {
			for _, i := range idx.failed {
				if summary.Results[i].Stderr != "" {
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

// UpdateCancelled prints the fixed message when the user cancels an
// interactive update run (design D8): nothing was updated, exit 0.
// US spelling is intentional: the repo lint (golangci-lint misspell,
// locale US) rejects the British double-L spelling.
func (r *Renderer) UpdateCancelled() {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, _ = fmt.Fprintln(r.w, "Update canceled — no changes made.")
}

// --- Dry run ---

// DryRunHeader prints the dry-run header.
func (r *Renderer) DryRunHeader() {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, _ = fmt.Fprintf(r.w, "%s Dry run — no changes will be made\n\n", r.statusIcon(StatusAvailable))
}

// DryRunPlanned prints a planned action for dry-run mode.
func (r *Renderer) DryRunPlanned(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, _ = fmt.Fprintf(r.w, "  %s %s\n", r.statusIcon(StatusAvailable), r.cyan(name))
}
