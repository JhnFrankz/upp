package output

import (
	"bytes"
	"errors"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// --- Minimal terminal simulator (VT100 subset) ---

// termScreen replays the ANSI subset emitted by CheckBoard ("\r", "\n",
// "\x1b[<n>A" cursor up, "\x1b[<n>B" cursor down, "\x1b[K" clear to end of
// line) onto a character grid. Tests assert the settled visible frame, so a
// broken cursor move fails the test instead of silently corrupting output.
// All bookkeeping is in byte space, which is self-consistent because col
// always mirrors the byte length of the row content written so far.
type termScreen struct {
	rows []strings.Builder
	row  int
	col  int
}

func newTermScreen() *termScreen {
	return &termScreen{rows: []strings.Builder{{}}}
}

func (t *termScreen) ensure(row int) {
	for len(t.rows) <= row {
		t.rows = append(t.rows, strings.Builder{})
	}
}

func (t *termScreen) Write(p []byte) (int, error) {
	i := 0
	for i < len(p) {
		c := p[i]
		switch {
		case c == '\r':
			t.col = 0
			i++
		case c == '\n':
			t.row++
			t.ensure(t.row)
			i++
		case c == 0x1b && i+1 < len(p) && p[i+1] == '[':
			j := i + 2
			for j < len(p) && !isFinalByte(p[j]) {
				j++
			}
			if j >= len(p) {
				return len(p), nil
			}
			n := 1
			if j > i+2 {
				if v, err := strconv.Atoi(string(p[i+2 : j])); err == nil {
					n = v
				}
			}
			switch p[j] {
			case 'A':
				t.row -= n
				if t.row < 0 {
					t.row = 0
				}
			case 'B':
				t.ensure(t.row + n)
				t.row += n
			case 'K':
				t.ensure(t.row)
				s := t.rows[t.row].String()
				if t.col < len(s) {
					t.rows[t.row].Reset()
					t.rows[t.row].WriteString(s[:t.col])
				}
			}
			i = j + 1
		default:
			t.ensure(t.row)
			t.rows[t.row].WriteByte(c)
			t.col++
			i++
		}
	}
	return len(p), nil
}

func isFinalByte(b byte) bool {
	return (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z')
}

// frame returns the visible rows with trailing blank rows trimmed.
func (t *termScreen) frame() []string {
	out := make([]string, 0, len(t.rows))
	for _, r := range t.rows {
		out = append(out, r.String())
	}
	for len(out) > 0 && out[len(out)-1] == "" {
		out = out[:len(out)-1]
	}
	return out
}

// replay renders everything written to buf through the simulator and returns
// the settled visible frame.
func replay(buf *bytes.Buffer) []string {
	screen := newTermScreen()
	_, _ = screen.Write(buf.Bytes())
	return screen.frame()
}

func assertFrame(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("board frame has %d rows, want %d:\ngot:  %q\nwant: %q", len(got), len(want), got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("row %d = %q, want %q", i, got[i], want[i])
		}
	}
}

// --- Start: canonical-order pending lines ---

func TestCheckBoard_Start_PaintsPendingLinesInCanonicalOrder(t *testing.T) {
	var buf bytes.Buffer
	b := NewCheckBoard(&buf, true, []string{"brew", "npm", "apt"})
	b.Start()

	assertFrame(t, replay(&buf), []string{
		"  ⟳ brew",
		"  ⟳ npm",
		"  ⟳ apt",
	})

	// Live-board mode must engage ANSI cursor control.
	if !strings.Contains(buf.String(), "\x1b[") {
		t.Error("color board output contains no ANSI escape sequences")
	}
}

// --- Complete: per-status in-place flips ---

func TestCheckBoard_Complete_AvailableFlipsOnlyTargetLine(t *testing.T) {
	var buf bytes.Buffer
	b := NewCheckBoard(&buf, true, []string{"brew", "npm", "apt"})
	b.Start()
	b.Complete(1, ToolResult{Name: "npm", Status: StatusAvailable, Version: "1.2 → 1.3"})

	assertFrame(t, replay(&buf), []string{
		"  ⟳ brew",
		"  ✓ npm 1.2 → 1.3",
		"  ⟳ apt",
	})
}

func TestCheckBoard_Complete_CurrentShowsUpToDate(t *testing.T) {
	var buf bytes.Buffer
	b := NewCheckBoard(&buf, true, []string{"brew", "npm"})
	b.Start()
	b.Complete(0, ToolResult{Name: "brew", Status: StatusCurrent, Version: "4.1.0"})

	assertFrame(t, replay(&buf), []string{
		"  ✓ brew up-to-date",
		"  ⟳ npm",
	})
}

func TestCheckBoard_Complete_SkippedShowsNotInstalled(t *testing.T) {
	var buf bytes.Buffer
	b := NewCheckBoard(&buf, true, []string{"brew", "gh"})
	b.Start()
	b.Complete(1, ToolResult{Name: "gh", Status: StatusSkipped})

	assertFrame(t, replay(&buf), []string{
		"  ⟳ brew",
		"  ✓ gh not installed",
	})
}

func TestCheckBoard_Complete_FailedShowsInlineError(t *testing.T) {
	var buf bytes.Buffer
	b := NewCheckBoard(&buf, true, []string{"brew", "apt"})
	b.Start()
	b.Complete(1, ToolResult{
		Name:   "apt",
		Status: StatusFailed,
		Error:  errors.New("network unreachable"),
	})

	assertFrame(t, replay(&buf), []string{
		"  ⟳ brew",
		"  ✗ apt: network unreachable",
	})
}

// --- Completion order never reorders lines ---

func TestCheckBoard_CompletionOrderDoesNotReorderLines(t *testing.T) {
	var buf bytes.Buffer
	b := NewCheckBoard(&buf, true, []string{"brew", "npm", "apt"})
	b.Start()

	// Out-of-order completions: last tool first, first tool second.
	b.Complete(2, ToolResult{Name: "apt", Status: StatusAvailable, Version: "2.0 → 2.1"})
	b.Complete(0, ToolResult{Name: "brew", Status: StatusCurrent, Version: "4.1.0"})

	assertFrame(t, replay(&buf), []string{
		"  ✓ brew up-to-date",
		"  ⟳ npm",
		"  ✓ apt 2.0 → 2.1",
	})
}

// --- Finish: settle final frame, idempotent ---

func TestCheckBoard_Finish_SettlesFrameAndIsIdempotent(t *testing.T) {
	var buf bytes.Buffer
	b := NewCheckBoard(&buf, true, []string{"brew", "npm"})
	b.Start()
	b.Complete(0, ToolResult{Name: "brew", Status: StatusAvailable, Version: "1.0 → 1.1"})

	b.Finish()
	sizeAfterFirstFinish := buf.Len()

	frame := replay(&buf)
	assertFrame(t, frame, []string{
		"  ✓ brew 1.0 → 1.1",
		"  ⟳ npm",
	})

	// Second Finish must be a no-op.
	b.Finish()
	if buf.Len() != sizeAfterFirstFinish {
		t.Errorf("second Finish wrote %d extra bytes; Finish must be idempotent", buf.Len()-sizeAfterFirstFinish)
	}
}

// --- Non-color fallback: one plain line per completion, no ANSI ---

func TestCheckBoard_NonColorFallback_OnePlainLinePerCompletion(t *testing.T) {
	var buf bytes.Buffer
	b := NewCheckBoard(&buf, false, []string{"brew", "npm", "apt"})

	b.Start()
	if buf.Len() != 0 {
		t.Errorf("fallback Start wrote %d bytes; fallback must not paint a board", buf.Len())
	}

	b.Complete(0, ToolResult{Name: "brew", Status: StatusAvailable, Version: "1.2 → 1.3"})
	b.Complete(1, ToolResult{Name: "npm", Status: StatusCurrent, Version: "10.2.0"})
	b.Complete(2, ToolResult{Name: "apt", Status: StatusFailed, Error: errors.New("lock held")})
	b.Finish()

	out := buf.String()
	if strings.Contains(out, "\x1b") {
		t.Errorf("fallback output contains ANSI escape sequences:\n%q", out)
	}

	got := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	want := []string{
		"  ✓ brew 1.2 → 1.3",
		"  ✓ npm up-to-date",
		"  ✗ apt: lock held",
	}
	if len(got) != len(want) {
		t.Fatalf("fallback emitted %d lines, want %d:\n%q", len(got), len(want), out)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("fallback line %d = %q, want %q", i, got[i], want[i])
		}
	}
}

// --- Concurrency: mutex-serialized updates under -race ---

func TestCheckBoard_ConcurrentComplete_SerializesUpdates(t *testing.T) {
	tools := []string{"t0", "t1", "t2", "t3", "t4", "t5", "t6", "t7"}
	var buf bytes.Buffer
	b := NewCheckBoard(&buf, true, tools)
	b.Start()

	results := make([]ToolResult, len(tools))
	for i, name := range tools {
		status := StatusCurrent
		version := ""
		switch i % 4 {
		case 0:
			status = StatusAvailable
			version = "1.0 → 1.1"
		case 1:
			status = StatusCurrent
		case 2:
			status = StatusSkipped
		case 3:
			status = StatusFailed
		}
		results[i] = ToolResult{Name: name, Status: status, Version: version}
		if status == StatusFailed {
			results[i].Error = errors.New("boom " + name)
		}
	}

	var wg sync.WaitGroup
	for i := range tools {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			b.Complete(i, results[i])
		}(i)
	}
	wg.Wait()
	b.Finish()

	got := replay(&buf)
	if len(got) != len(tools) {
		t.Fatalf("settled board has %d rows, want %d:\n%q", len(got), len(tools), got)
	}
	for i, name := range tools {
		var want string
		switch results[i].Status {
		case StatusAvailable:
			want = "  ✓ " + name + " 1.0 → 1.1"
		case StatusCurrent:
			want = "  ✓ " + name + " up-to-date"
		case StatusSkipped:
			want = "  ✓ " + name + " not installed"
		case StatusFailed:
			want = "  ✗ " + name + ": boom " + name
		}
		if got[i] != want {
			t.Errorf("row %d = %q, want %q", i, got[i], want)
		}
	}
}

// --- Defensive: out-of-range indices are ignored, never panic ---

func TestCheckBoard_Complete_OutOfRangeIndexIgnored(t *testing.T) {
	var buf bytes.Buffer
	b := NewCheckBoard(&buf, true, []string{"brew", "npm"})
	b.Start()

	b.Complete(-1, ToolResult{Name: "ghost", Status: StatusCurrent})
	b.Complete(99, ToolResult{Name: "ghost", Status: StatusCurrent})

	assertFrame(t, replay(&buf), []string{
		"  ⟳ brew",
		"  ⟳ npm",
	})
}
