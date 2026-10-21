// Package output handles terminal rendering with color, emoji, and
// graceful degradation for pipes and dumb terminals.
package output

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
)

// LogPresenter defines the interface for emitting error and warning messages.
type LogPresenter interface {
	Error(msg string)
	Warning(msg string)
}

// Renderer handles formatted terminal output.
type Renderer struct {
	w       io.Writer
	color   bool
	emoji   bool
	quiet   bool
	verbose bool
	mu      sync.Mutex
	scratch strings.Builder
}

var _ LogPresenter = (*Renderer)(nil)

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
	case StatusDenied:
		return "🚫"
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
	case StatusDenied:
		return "[denied]"
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
	case StatusDenied:
		return "denied"
	default:
		return "unknown"
	}
}

// --- Color helpers ---

func (r *Renderer) writeColor(sb *strings.Builder, code, text string) {
	if !r.color {
		sb.WriteString(text)
		return
	}
	sb.WriteString("\033[")
	sb.WriteString(code)
	sb.WriteByte('m')
	sb.WriteString(text)
	sb.WriteString("\033[0m")
}

func (r *Renderer) colorize(code, text string) string {
	if !r.color {
		return text
	}
	return "\033[" + code + "m" + text + "\033[0m"
}

func (r *Renderer) red(text string) string    { return r.colorize("31", text) }
func (r *Renderer) yellow(text string) string { return r.colorize("33", text) }
func (r *Renderer) cyan(text string) string   { return r.colorize("36", text) }
func (r *Renderer) dim(text string) string    { return r.colorize("2", text) }

// --- Error output ---

// Error prints a prefixed error message.
func (r *Renderer) Error(msg string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, _ = fmt.Fprintf(r.w, "%s %s\n", r.statusIcon(StatusFailed), r.red(msg))
}

// Warning prints a warning message.
func (r *Renderer) Warning(msg string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, _ = fmt.Fprintf(r.w, "⚠️  %s\n", r.yellow(msg))
}
