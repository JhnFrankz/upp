package output

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

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
	makeRawFn     = term.MakeRaw
	restoreTerm   = term.Restore
	termGetSizeFn = term.GetSize
)

const selectorKeyHints = "Space toggle · a all · n none · Enter confirm · q/Esc cancel"

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

// Run is RunContext with a background context.
func (s *CheckboxSelector) Run() (SelectResult, error) {
	return s.RunContext(context.Background())
}

const escSequenceWait = 25 * time.Millisecond

type readByteMsg struct {
	b   byte
	err error
}

// bytePump reads bytes from r on one persistent goroutine so RunContext can
// wait on ctx.Done() and short escape-sequence timeouts without spawning a
// goroutine per keystroke. A Read blocked when ctx cancels is abandoned until
// the process exits or r unblocks.
type bytePump struct {
	ch   chan readByteMsg
	stop chan struct{}
}

func newBytePump(r io.Reader) *bytePump {
	p := &bytePump{
		ch:   make(chan readByteMsg, 64),
		stop: make(chan struct{}),
	}
	go func() {
		var buf [64]byte
		for {
			n, err := r.Read(buf[:])
			for i := 0; i < n; i++ {
				select {
				case p.ch <- readByteMsg{b: buf[i]}:
				case <-p.stop:
					return
				}
			}
			if err != nil {
				select {
				case p.ch <- readByteMsg{err: err}:
				case <-p.stop:
				}
				return
			}
		}
	}()
	return p
}

func (p *bytePump) close() { close(p.stop) }

func (p *bytePump) next(ctx context.Context) (byte, error) {
	select {
	case msg := <-p.ch:
		return msg.b, msg.err
	case <-ctx.Done():
		return 0, ctx.Err()
	}
}

func (p *bytePump) nextTimeout(ctx context.Context, d time.Duration) (byte, bool, error) {
	select {
	case msg := <-p.ch:
		return msg.b, true, msg.err
	default:
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case msg := <-p.ch:
		return msg.b, true, msg.err
	case <-timer.C:
		return 0, false, nil
	case <-ctx.Done():
		return 0, false, ctx.Err()
	}
}

// RunContext renders the selector and processes keys until Enter (confirm),
// Esc/q (cancel), ctx is canceled, or input fails. It fails closed: end of
// input or a read error before an explicit Enter cancels the selection
// instead of confirming the pre-checked tools. Ctrl-C (0x03) returns
// context.Canceled so callers exit with 130. For a *os.File reader the terminal
// is put into raw mode first and restored on every exit path (confirm, cancel,
// error, panic) via defer — the threat-matrix contract for terminal state.
// Non-File readers (bytes.Buffer in tests, pipes) skip raw mode entirely.
func (s *CheckboxSelector) RunContext(ctx context.Context) (SelectResult, error) {
	rawMode := false
	if f, ok := s.r.(*os.File); ok {
		state, err := makeRawFn(int(f.Fd()))
		if err != nil {
			return SelectResult{}, fmt.Errorf("raw mode: %w", err)
		}
		rawMode = true
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

	termHeight := 0
	if f, ok := s.w.(*os.File); ok {
		if _, h, err := termGetSizeFn(int(f.Fd())); err == nil && h > 0 {
			termHeight = h
		}
	}
	bodyBudget := 0
	if termHeight > 0 {
		// Reserve 1 row for the trailing newline after the footer so cursor-up
		// redraws never scroll past row 1, plus 1 row for the keybinding footer.
		maxLines := max(1, termHeight-1)
		if maxLines > 1 {
			bodyBudget = maxLines - 1
		} else {
			bodyBudget = 1
		}
	}

	if useColor {
		_, _ = io.WriteString(s.w, "\x1b[?25l")
		defer func() { _, _ = io.WriteString(s.w, "\x1b[?25h") }()
	}

	scrollTop := 0
	linesRendered := 0
	render := func() {
		var sb strings.Builder
		sb.Grow(len(s.opts)*80 + 128)
		if useColor {
			sb.WriteString(noWrapOn)
		}
		if useColor && linesRendered > 0 {
			sb.WriteString("\x1b[")
			sb.WriteString(strconv.Itoa(linesRendered))
			sb.WriteString("A\r")
		}

		newline := "\n"
		if rawMode {
			newline = "\r\n"
		}

		var start, end int
		scrollTop, start, end = selectorWindow(s.opts, cursor, scrollTop, bodyBudget)

		currentLines := 0
		writeMore := func(arrow string, count int) {
			if useColor {
				sb.WriteString("\r\x1b[K\x1b[2m   ")
				sb.WriteString(arrow)
				sb.WriteByte(' ')
				sb.WriteString(strconv.Itoa(count))
				sb.WriteString(" more\x1b[0m")
			} else {
				sb.WriteString("   ")
				sb.WriteString(arrow)
				sb.WriteByte(' ')
				sb.WriteString(strconv.Itoa(count))
				sb.WriteString(" more")
			}
			sb.WriteString(newline)
			currentLines++
		}

		if start > 0 {
			writeMore("↑", start)
		}

		lastGroup := ""
		for i := start; i < end; i++ {
			opt := s.opts[i]
			// A group header line renders once, before the first option of
			// that group (design: selector group headers). Options without a
			// Group field (or repeated within a contiguous run) are unaffected.
			if opt.Group != "" && opt.Group != lastGroup {
				if useColor {
					sb.WriteString("\r\x1b[K")
				}
				sb.WriteString(opt.Group)
				sb.WriteString(newline)
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
			sb.WriteString(newline)
			currentLines++
		}

		if end < len(s.opts) {
			writeMore("↓", len(s.opts)-end)
		}

		if termHeight == 0 || termHeight > 2 {
			if useColor {
				sb.WriteString("\r\x1b[K\x1b[2m")
				sb.WriteString(selectorKeyHints)
				sb.WriteString("\x1b[0m")
			} else {
				sb.WriteString(selectorKeyHints)
			}
			sb.WriteString(newline)
			currentLines++
		}

		if useColor && currentLines < linesRendered {
			sb.WriteString("\x1b[J")
		}
		linesRendered = currentLines
		if useColor {
			sb.WriteString(noWrapOff)
		}
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

	render()
	pump := newBytePump(s.r)
	defer pump.close()
	for {
		b, err := pump.next(ctx)
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
			if len(selected) > 0 {
				selected[cursor] = !selected[cursor]
			}
		case 'a':
			for i := range selected {
				selected[i] = true
			}
		case 'n':
			for i := range selected {
				selected[i] = false
			}
		case 'k':
			if cursor > 0 {
				cursor--
			}
		case 'j':
			if cursor < len(s.opts)-1 {
				cursor++
			}
		case 0x03:
			return SelectResult{Canceled: true}, context.Canceled
		case 0x04, 'q':
			return SelectResult{Canceled: true}, nil
		case 0x1b:
			// Escape either starts an arrow sequence (\x1b[A/\x1b[B or
			// application-mode \x1bOA/\x1bOB) or cancels on its own. Wait
			// briefly so split SSH/tmux bursts still register as arrows
			// without hanging bare Esc.
			next, ok, err := pump.nextTimeout(ctx, escSequenceWait)
			if err != nil {
				if errors.Is(err, io.EOF) {
					return SelectResult{Canceled: true}, nil
				}
				return SelectResult{Canceled: true}, err
			}
			if !ok || (next != '[' && next != 'O') {
				return SelectResult{Canceled: true}, nil
			}
			arrow, ok, err := pump.nextTimeout(ctx, escSequenceWait)
			if err != nil {
				if errors.Is(err, io.EOF) {
					return SelectResult{Canceled: true}, nil
				}
				return SelectResult{Canceled: true}, err
			}
			if !ok {
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

func selectorWindow(opts []SelectOption, cursor, scrollTop, bodyBudget int) (int, int, int) {
	if len(opts) == 0 {
		return 0, 0, 0
	}
	if bodyBudget <= 0 {
		return 0, 0, len(opts)
	}
	if cursor < scrollTop {
		scrollTop = cursor
	}
	end := computeWindowEnd(opts, scrollTop, bodyBudget)
	for cursor >= end && scrollTop < len(opts)-1 {
		scrollTop++
		end = computeWindowEnd(opts, scrollTop, bodyBudget)
	}
	return scrollTop, scrollTop, end
}

func computeWindowEnd(opts []SelectOption, start, bodyBudget int) int {
	topInd := 0
	if start > 0 {
		topInd = 1
	}

	totalLines := topInd
	lastGroup := ""
	for i := start; i < len(opts); i++ {
		if opts[i].Group != "" && opts[i].Group != lastGroup {
			totalLines++
			lastGroup = opts[i].Group
		}
		totalLines++
	}
	if totalLines <= bodyBudget {
		return len(opts)
	}

	avail := bodyBudget - topInd - 1 // reserve 1 line for bottom ↓ indicator
	used := 0
	lastGroup = ""
	end := start
	for i := start; i < len(opts); i++ {
		cost := 1
		nextGroup := lastGroup
		if opts[i].Group != "" && opts[i].Group != lastGroup {
			cost++
			nextGroup = opts[i].Group
		}
		if i > start && used+cost > avail {
			break
		}
		used += cost
		lastGroup = nextGroup
		end = i + 1
	}
	return end
}
