package output

import (
	"bufio"
	"fmt"
	"io"
	"os"

	"golang.org/x/term"
)

// SelectOption describes one selectable tool in the checkbox selector.
type SelectOption struct {
	ID      string // tool ID (--only/--skip key)
	Label   string // display name
	Version string // inline "Current → Latest"
	Group   string // owning manager label; non-empty options render a group header line first
}

// SelectResult holds the outcome of a selector run.
type SelectResult struct {
	Selected []string // selected IDs in display order
	Canceled bool     // Esc/q
}

// Test seams (mirrors shellExecWithTimeoutFn in adapters/exec.go): package
// vars swapped by tests so raw mode can be exercised hermetically without
// touching a real terminal. Production behavior is preserved — they
// initialize to the real term implementations.
var (
	makeRawFn   = term.MakeRaw
	restoreTerm = term.Restore
)

// CheckboxSelector renders an interactive multi-select list and reads keys
// from the injected reader.
type CheckboxSelector struct {
	w    io.Writer
	r    io.Reader
	opts []SelectOption
}

// NewCheckboxSelector creates a checkbox selector rendering to w and reading
// keys from r. All options start pre-checked; the cursor starts on the first.
func NewCheckboxSelector(w io.Writer, r io.Reader, opts []SelectOption) *CheckboxSelector {
	return &CheckboxSelector{w: w, r: r, opts: opts}
}

// Run renders the selector and processes keys until Enter (confirm), Esc/q
// (cancel), or the reader is exhausted. For a *os.File reader the terminal is
// put into raw mode first and restored on every exit path (confirm, cancel,
// error, panic) via defer — the threat-matrix contract for terminal state.
// Non-File readers (bytes.Buffer in tests, pipes) skip raw mode entirely.
func (s *CheckboxSelector) Run() (SelectResult, error) {
	if f, ok := s.r.(*os.File); ok {
		state, err := makeRawFn(int(f.Fd()))
		if err != nil {
			return SelectResult{}, fmt.Errorf("raw mode: %w", err)
		}
		defer func() { _ = restoreTerm(int(f.Fd()), state) }()
	}

	selected := make([]bool, len(s.opts))
	for i := range selected {
		selected[i] = true // all pending tools pre-checked
	}
	cursor := 0

	// The renderer detects color/emoji from w: TTY renders with ANSI colors,
	// buffer/pipe writers stay plain (existing Renderer conventions).
	r := NewRenderer(s.w, false)
	render := func() {
		lastGroup := ""
		for i, opt := range s.opts {
			// A group header line renders once, before the first option of
			// that group (design: selector group headers). Options without a
			// Group field (or repeated within a contiguous run) are unaffected.
			if opt.Group != "" && opt.Group != lastGroup {
				_, _ = fmt.Fprintln(s.w, opt.Group)
				lastGroup = opt.Group
			}
			marker := "[ ]"
			if selected[i] {
				marker = "[x]"
			}
			prefix := "  "
			if i == cursor {
				prefix = r.cyan("▶") + " "
			}
			line := fmt.Sprintf("%s %s %s", prefix, marker, r.cyan(opt.Label))
			if opt.Version != "" {
				line += " " + r.dim(opt.Version)
			}
			_, _ = fmt.Fprintln(s.w, line)
		}
	}

	collect := func() []string {
		ids := make([]string, 0, len(s.opts))
		for i, opt := range s.opts {
			if selected[i] {
				ids = append(ids, opt.ID)
			}
		}
		return ids
	}

	br := bufio.NewReader(s.r)
	render()
	for {
		b, err := br.ReadByte()
		if err != nil {
			return SelectResult{Selected: collect()}, nil
		}
		switch b {
		case '\r', '\n':
			return SelectResult{Selected: collect()}, nil
		case ' ':
			selected[cursor] = !selected[cursor]
		case 'a':
			for i := range selected {
				selected[i] = true
			}
		case 'n':
			for i := range selected {
				selected[i] = false
			}
		case 'q':
			return SelectResult{Canceled: true}, nil
		case 0x1b:
			// Escape either starts an arrow sequence (\x1b[A / \x1b[B) or
			// cancels on its own; any other \x1b-prefixed sequence also
			// cancels (raw-mode terminals send bare \x1b for Esc).
			next, err := br.Peek(1)
			if err != nil || next[0] != '[' {
				return SelectResult{Canceled: true}, nil
			}
			_, _ = br.ReadByte() // consume '['
			arrow, err := br.ReadByte()
			if err != nil {
				return SelectResult{Canceled: true}, nil
			}
			switch arrow {
			case 'A':
				if cursor > 0 {
					cursor--
				}
			case 'B':
				if cursor < len(s.opts)-1 {
					cursor++
				}
			}
		default:
			// Unknown keys are ignored.
		}
		render()
	}
}
