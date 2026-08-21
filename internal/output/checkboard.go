package output

import (
	"fmt"
	"io"
	"strings"
	"sync"
)

// CheckBoard paints a live pre-check board: exactly one stable line per
// filtered tool, laid out in canonical tool discovery order, flipping each
// line in place as its check completes (design D4).
//
// In color mode the board owns multi-line ANSI cursor control (design D1):
// Complete moves the cursor up to the target row, clears to end of line,
// rewrites that single line, and returns to the bottom row — never a
// full-board redraw. A private mutex serializes every write so concurrent
// worker completions can never interleave or corrupt output. Completion order
// cannot reorder lines because each completion is slotted by index.
//
// Without color the board falls back to one plain line per completion with no
// ANSI cursor control (design D5); Start and Finish print nothing.
type CheckBoard struct {
	w        io.Writer
	color    bool
	lines    []string // rendered text of each board row, canonical order
	mu       sync.Mutex
	finished bool
}

// NewCheckBoard creates a board with one pending row per tool name, in the
// given canonical order. color enables the in-place ANSI board; callers pass
// Renderer.Color() so TTY detection keeps a single source of truth (D5).
func NewCheckBoard(w io.Writer, color bool, tools []string) *CheckBoard {
	lines := make([]string, len(tools))
	for i, name := range tools {
		lines[i] = boardPendingLine(name, color)
	}
	return &CheckBoard{w: w, color: color, lines: lines}
}

// Start paints the initial frame: one pending line per tool in canonical
// order. Call it once, before the first check completes.
func (b *CheckBoard) Start() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.color || len(b.lines) == 0 {
		return
	}
	var sb strings.Builder
	for i, line := range b.lines {
		sb.WriteString(line)
		if i < len(b.lines)-1 {
			sb.WriteByte('\n')
		}
	}
	_, _ = io.WriteString(b.w, sb.String())
}

// Complete flips exactly one line in place: the row at index is rewritten to
// reflect res while every other row stays untouched. Out-of-range indices are
// ignored. Safe for concurrent use; the mutex serializes all output.
func (b *CheckBoard) Complete(index int, res ToolResult) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if index < 0 || index >= len(b.lines) {
		return
	}
	line, ok := boardResultLine(res, b.color)
	if !ok {
		return // unknown status leaves the row untouched
	}
	b.lines[index] = line

	if !b.color {
		// Fallback: one plain line per completion, no ANSI cursor control.
		_, _ = fmt.Fprintf(b.w, "%s\n", line)
		return
	}
	b.rewriteRow(index)
}

// Finish settles the final frame so subsequent output (the selector) starts
// on a fresh line. Idempotent: repeated calls write nothing.
func (b *CheckBoard) Finish() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.finished {
		return
	}
	b.finished = true
	if !b.color {
		return
	}
	_, _ = io.WriteString(b.w, "\n")
}

// rewriteRow moves the cursor up to row index, clears that line, rewrites it,
// and returns the cursor to the bottom row. Caller holds b.mu; color only.
func (b *CheckBoard) rewriteRow(index int) {
	up := len(b.lines) - 1 - index
	var sb strings.Builder
	sb.WriteByte('\r')
	if up > 0 {
		fmt.Fprintf(&sb, "\x1b[%dA", up)
	}
	sb.WriteString("\x1b[K")
	sb.WriteString(b.lines[index])
	if up > 0 {
		fmt.Fprintf(&sb, "\x1b[%dB", up)
	}
	sb.WriteByte('\r')
	_, _ = io.WriteString(b.w, sb.String())
}

// boardPendingLine renders a not-yet-checked row.
func boardPendingLine(name string, color bool) string {
	return boardMarkerLine("⟳", "2", name, "", "", color)
}

// boardResultLine renders a completed row for the given outcome. The second
// return is false for statuses the board never receives during a pre-check.
func boardResultLine(res ToolResult, color bool) (string, bool) {
	switch res.Status {
	case StatusAvailable:
		return boardMarkerLine("✓", "32", res.Name, res.Version, "", color), true
	case StatusCurrent:
		return boardMarkerLine("✓", "32", res.Name, "up-to-date", "", color), true
	case StatusSkipped:
		return boardMarkerLine("✓", "2", res.Name, "not installed", "", color), true
	case StatusFailed:
		detail := ""
		if res.Error != nil {
			detail = res.Error.Error()
		}
		return boardMarkerLine("✗", "31", res.Name, "", detail, color), true
	default:
		return "", false
	}
}

// boardMarkerLine builds "  <marker> <name>[ <detail>|: <errDetail>]" with the
// marker wrapped in the given SGR code when color is on.
func boardMarkerLine(marker, code, name, detail, errDetail string, color bool) string {
	if color {
		marker = "\x1b[" + code + "m" + marker + "\x1b[0m"
	}
	line := "  " + marker + " " + name
	switch {
	case errDetail != "":
		line += ": " + errDetail
	case detail != "":
		line += " " + detail
	}
	return line
}
