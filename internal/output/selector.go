package output

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"golang.org/x/term"
)

// SelectOption describes one selectable tool in the checkbox selector.
type SelectOption struct {
	ID      string // tool ID (--only key)
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
	w     io.Writer
	r     io.Reader
	opts  []SelectOption
	color *bool
}

// NewCheckboxSelector creates a checkbox selector rendering to w and reading
// keys from r. All options start pre-checked; the cursor starts on the first.
func NewCheckboxSelector(w io.Writer, r io.Reader, opts []SelectOption) *CheckboxSelector {
	return &CheckboxSelector{w: w, r: r, opts: opts}
}

// WithColor explicitly sets or overrides whether ANSI color and in-place
// cursor repositioning are enabled. By default it is auto-detected from w.
func (s *CheckboxSelector) WithColor(color bool) *CheckboxSelector {
	s.color = &color
	return s
}

// Run is RunContext with a background context.
func (s *CheckboxSelector) Run() (SelectResult, error) {
	return s.RunContext(context.Background())
}

// ctxReader makes a blocking Read abortable by ctx. A Read pending when ctx
// is cancelled is abandoned (its goroutine ends with the process or when the
// underlying reader unblocks); callers must not reuse the reader afterwards.
type ctxReader struct {
	ctx context.Context
	r   io.Reader
}

func (c ctxReader) Read(p []byte) (int, error) {
	if err := c.ctx.Err(); err != nil {
		return 0, err
	}
	type result struct {
		n   int
		err error
	}
	ch := make(chan result, 1)
	buf := make([]byte, len(p))
	go func() {
		n, err := c.r.Read(buf)
		ch <- result{n, err}
	}()
	select {
	case res := <-ch:
		copy(p, buf[:res.n])
		return res.n, res.err
	case <-c.ctx.Done():
		return 0, c.ctx.Err()
	}
}

// RunContext renders the selector and processes keys until Enter (confirm),
// Esc/q (cancel), ctx is cancelled, or input fails. It fails closed: end of
// input or a read error before an explicit Enter cancels the selection
// instead of confirming the pre-checked tools. For a *os.File reader the terminal is
// put into raw mode first and restored on every exit path (confirm, cancel,
// error, panic) via defer — the threat-matrix contract for terminal state.
// Non-File readers (bytes.Buffer in tests, pipes) skip raw mode entirely.
func (s *CheckboxSelector) RunContext(ctx context.Context) (SelectResult, error) {
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
	useColor := r.Color()
	if s.color != nil {
		useColor = *s.color
	}

	if useColor {
		_, _ = io.WriteString(s.w, "\x1b[?25l")
		defer func() { _, _ = io.WriteString(s.w, "\x1b[?25h") }()
	}

	linesRendered := 0
	render := func() {
		var sb strings.Builder
		sb.Grow(len(s.opts)*80 + 64)
		if useColor && linesRendered > 0 {
			sb.WriteString("\x1b[")
			sb.WriteString(strconv.Itoa(linesRendered))
			sb.WriteString("A\r")
		}

		currentLines := 0
		lastGroup := ""
		for i, opt := range s.opts {
			// A group header line renders once, before the first option of
			// that group (design: selector group headers). Options without a
			// Group field (or repeated within a contiguous run) are unaffected.
			if opt.Group != "" && opt.Group != lastGroup {
				if useColor {
					sb.WriteString("\r\x1b[K")
				}
				sb.WriteString(opt.Group)
				sb.WriteByte('\n')
				currentLines++
				lastGroup = opt.Group
			}
			marker := "[ ]"
			if selected[i] {
				marker = "[x]"
			}
			if useColor {
				sb.WriteString("\r\x1b[K")
			}
			if i == cursor {
				if useColor {
					sb.WriteString("\x1b[36m▶\x1b[0m")
				} else {
					sb.WriteString("▶")
				}
				sb.WriteString("  ")
			} else {
				sb.WriteString("   ")
			}
			sb.WriteString(marker)
			sb.WriteByte(' ')
			if useColor {
				sb.WriteString("\x1b[36m")
				sb.WriteString(opt.Label)
				sb.WriteString("\x1b[0m")
			} else {
				sb.WriteString(opt.Label)
			}
			if opt.Version != "" {
				sb.WriteByte(' ')
				if useColor {
					sb.WriteString("\x1b[2m")
					sb.WriteString(opt.Version)
					sb.WriteString("\x1b[0m")
				} else {
					sb.WriteString(opt.Version)
				}
			}
			sb.WriteByte('\n')
			currentLines++
		}
		linesRendered = currentLines
		_, _ = io.WriteString(s.w, sb.String())
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

	br := bufio.NewReader(ctxReader{ctx: ctx, r: s.r})
	render()
	for {
		b, err := br.ReadByte()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return SelectResult{Canceled: true}, nil
			}
			return SelectResult{Canceled: true}, err
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
		case 0x03, 0x04, 'q':
			return SelectResult{Canceled: true}, nil
		case 0x1b:
			// Escape either starts an arrow sequence (\x1b[A / \x1b[B) or
			// cancels on its own (raw-mode terminals send bare \x1b for Esc).
			// If no subsequent bytes were received in the same burst (br.Buffered() == 0),
			// cancel immediately without blocking on a synchronous Peek/Read.
			if br.Buffered() == 0 {
				return SelectResult{Canceled: true}, nil
			}
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
