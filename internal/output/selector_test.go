package output

import (
	"bytes"
	"os"
	"slices"
	"strings"
	"testing"

	"golang.org/x/term"
)

// TestSelector_KeyHandling is the behavioral contract for the checkbox
// selector (spec ux-patterns "Interactive Update Tool Selection"): all
// pending tools pre-checked, ↑/↓ move, Space toggles, a/n select all/none,
// Enter (\r or \n) confirms, Esc/q cancel, unknown keys ignored, and the
// returned selection preserves display order regardless of interaction order.
func TestSelector_KeyHandling(t *testing.T) {
	opts := []SelectOption{
		{ID: "brew", Label: "brew", Version: "4.0.0 → 4.1.0"},
		{ID: "npm", Label: "npm", Version: "10.0.0 → 10.1.0"},
		{ID: "go", Label: "go", Version: "1.22.0 → 1.23.0"},
		{ID: "docker", Label: "docker", Version: "26.0.0 → 27.0.0"},
	}

	tests := []struct {
		name       string
		input      string
		wantSel    []string
		wantCancel bool
	}{
		{
			name:    "enter CR confirms all pre-checked",
			input:   "\r",
			wantSel: []string{"brew", "npm", "go", "docker"},
		},
		{
			name:    "enter LF confirms",
			input:   "\n",
			wantSel: []string{"brew", "npm", "go", "docker"},
		},
		{
			name:    "space toggles the cursor row off",
			input:   " \r",
			wantSel: []string{"npm", "go", "docker"},
		},
		{
			name:    "down then space deselects the second row",
			input:   "\x1b[B \r",
			wantSel: []string{"brew", "go", "docker"},
		},
		{
			name:    "up from the first row stays and toggles it",
			input:   "\x1b[A \r",
			wantSel: []string{"npm", "go", "docker"},
		},
		{
			name:    "down past the last row stays and toggles it",
			input:   "\x1b[B\x1b[B\x1b[B\x1b[B \r",
			wantSel: []string{"brew", "npm", "go"},
		},
		{
			name:    "n deselects all",
			input:   "n\r",
			wantSel: []string{},
		},
		{
			name:    "a reselects all after n",
			input:   "na\r",
			wantSel: []string{"brew", "npm", "go", "docker"},
		},
		{
			name:       "esc cancels",
			input:      "\x1b",
			wantCancel: true,
		},
		{
			name:       "q cancels",
			input:      "q",
			wantCancel: true,
		},
		{
			name:    "unknown keys are ignored",
			input:   "xyz\r",
			wantSel: []string{"brew", "npm", "go", "docker"},
		},
		{
			// Deselect all rows in order brew→npm→go→docker, then reselect
			// docker BEFORE go. The result must still list [go, docker] in
			// display order — appending by interaction order would yield
			// [docker, go].
			name:    "selection preserves display order",
			input:   "\x1b[B\x1b[B\x1b[B \x1b[A\x1b[A\x1b[A \x1b[B \x1b[B \x1b[B \x1b[A \r",
			wantSel: []string{"go", "docker"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			sel := NewCheckboxSelector(&buf, strings.NewReader(tt.input), opts)
			res, err := sel.Run()
			if err != nil {
				t.Fatalf("Run() error: %v", err)
			}
			if res.Canceled != tt.wantCancel {
				t.Errorf("Canceled = %v, want %v", res.Canceled, tt.wantCancel)
			}
			if !slices.Equal(res.Selected, tt.wantSel) {
				t.Errorf("Selected = %v, want %v", res.Selected, tt.wantSel)
			}
			// The selector must actually render the option list — a run that
			// returns without rendering proves nothing.
			out := buf.String()
			if !strings.Contains(out, "brew") {
				t.Errorf("output must render the option list, got:\n%s", out)
			}
		})
	}
}

// TestSelector_RenderShape locks the visible checkbox-list shape: checked
// boxes, the cursor marker, inline versions, and the toggle being re-rendered.
func TestSelector_RenderShape(t *testing.T) {
	var buf bytes.Buffer
	opts := []SelectOption{
		{ID: "brew", Label: "brew", Version: "4.0.0 → 4.1.0"},
		{ID: "npm", Label: "npm"},
	}

	sel := NewCheckboxSelector(&buf, strings.NewReader(" \r"), opts)
	res, err := sel.Run()
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	if res.Canceled {
		t.Error("Run must not report canceled on Space + Enter")
	}

	out := buf.String()
	if !strings.Contains(out, "[x] brew") {
		t.Errorf("first render must show brew checked, got:\n%s", out)
	}
	if !strings.Contains(out, "[ ] brew") {
		t.Errorf("after Space the toggled row must render unchecked, got:\n%s", out)
	}
	if !strings.Contains(out, "4.0.0 → 4.1.0") {
		t.Errorf("version must render inline, got:\n%s", out)
	}
	if !strings.Contains(out, "▶") {
		t.Errorf("cursor marker must render, got:\n%s", out)
	}
}

// TestSelector_RenderGroupHeader locks the grouped selector contract (task
// 3.3): a SelectOption carrying a non-empty Group renders that group header
// line once, before its options; options with no Group (or the same Group)
// render without repeating the header; and selection still returns the IDs in
// display order.
func TestSelector_RenderGroupHeader(t *testing.T) {
	var buf bytes.Buffer
	opts := []SelectOption{
		{ID: "brew", Label: "brew", Version: "4.0.0 → 4.1.0", Group: "Homebrew"},
		{ID: "gh", Label: "GitHub CLI", Version: "2.3.0 → 2.4.0", Group: "Homebrew"},
		{ID: "npm", Label: "npm"},
	}

	sel := NewCheckboxSelector(&buf, strings.NewReader("\r"), opts)
	res, err := sel.Run()
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	if res.Canceled {
		t.Error("Run must not report canceled on Enter")
	}
	if !slices.Equal(res.Selected, []string{"brew", "gh", "npm"}) {
		t.Errorf("Selected = %v, want [brew gh npm]", res.Selected)
	}

	out := buf.String()
	// Group header renders exactly once, before its two options.
	if n := strings.Count(out, "Homebrew"); n != 1 {
		t.Errorf("group header 'Homebrew' must render once, got %d\n%s", n, out)
	}
	// Options present; npm (no Group) renders without a header.
	if !strings.Contains(out, "[x] brew") || !strings.Contains(out, "[x] GitHub CLI") || !strings.Contains(out, "[x] npm") {
		t.Errorf("grouped selector must render all options, got:\n%s", out)
	}
	// Group header precedes its options.
	if strings.Index(out, "Homebrew") > strings.Index(out, "[x] brew") {
		t.Errorf("group header must render before its options, got:\n%s", out)
	}
}

// TestSelector_GroupSelectionPreservesOrder pins the display-order selection
// contract with groups (task 3.3): toggling an option across two groups still
// returns selected IDs in display order (brew group then standalone), never
// interaction order.
func TestSelector_GroupSelectionPreservesOrder(t *testing.T) {
	var buf bytes.Buffer
	opts := []SelectOption{
		{ID: "brew", Label: "brew", Group: "Homebrew"},
		{ID: "gh", Label: "GitHub CLI", Group: "Homebrew"},
		{ID: "npm", Label: "npm"},
	}
	// Deselect brew (cursor stays on brew, one Space): result must be the
	// display-order remaining set [gh, npm] (brew group item then standalone).
	sel := NewCheckboxSelector(&buf, strings.NewReader(" \r"), opts)
	res, err := sel.Run()
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	if res.Canceled {
		t.Error("Run must not report canceled")
	}
	if !slices.Equal(res.Selected, []string{"gh", "npm"}) {
		t.Errorf("Selected = %v, want [gh npm]", res.Selected)
	}
}

// --- Raw mode (threat matrix: terminal raw mode) ---
//
// The reader is a real *os.File (os.Pipe read end) so Run enters the raw-mode
// path; makeRawFn/restoreTerm are swapped for recording fakes so the terminal
// is never actually touched.

func TestSelector_RawMode_MakeRawOnceAndRestoreOnConfirm(t *testing.T) {
	var buf bytes.Buffer
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error: %v", err)
	}
	defer func() { _ = r.Close() }()
	_, _ = w.WriteString("\r")
	_ = w.Close()

	var rawCalls, restoreCalls int
	var rawFD, restoreFD uintptr
	oldMakeRaw, oldRestore := makeRawFn, restoreTerm
	makeRawFn = func(fd int) (*term.State, error) {
		rawCalls++
		rawFD = uintptr(fd)
		return &term.State{}, nil
	}
	restoreTerm = func(fd int, _ *term.State) error {
		restoreCalls++
		restoreFD = uintptr(fd)
		return nil
	}
	defer func() { makeRawFn, restoreTerm = oldMakeRaw, oldRestore }()

	sel := NewCheckboxSelector(&buf, r, []SelectOption{{ID: "brew", Label: "brew"}})
	res, err := sel.Run()
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	if rawCalls != 1 {
		t.Errorf("MakeRaw must be called exactly once, got %d calls", rawCalls)
	}
	if restoreCalls != 1 {
		t.Errorf("Restore must be called on confirm, got %d calls", restoreCalls)
	}
	if rawFD != r.Fd() || restoreFD != r.Fd() {
		t.Errorf("MakeRaw/Restore fd mismatch: raw=%d restore=%d reader=%d", rawFD, restoreFD, r.Fd())
	}
	if res.Canceled || len(res.Selected) != 1 {
		t.Errorf("unexpected result: %+v", res)
	}
}

func TestSelector_RawMode_RestoreOnCancel(t *testing.T) {
	for _, tt := range []struct {
		name  string
		input string
	}{
		{name: "esc", input: "\x1b"},
		{name: "q", input: "q"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			r, w, err := os.Pipe()
			if err != nil {
				t.Fatalf("os.Pipe() error: %v", err)
			}
			defer func() { _ = r.Close() }()
			_, _ = w.WriteString(tt.input)
			_ = w.Close()

			var restoreCalls int
			oldMakeRaw, oldRestore := makeRawFn, restoreTerm
			makeRawFn = func(fd int) (*term.State, error) { return &term.State{}, nil }
			restoreTerm = func(fd int, _ *term.State) error { restoreCalls++; return nil }
			defer func() { makeRawFn, restoreTerm = oldMakeRaw, oldRestore }()

			sel := NewCheckboxSelector(&buf, r, []SelectOption{{ID: "brew", Label: "brew"}})
			res, err := sel.Run()
			if err != nil {
				t.Fatalf("Run() error: %v", err)
			}
			if !res.Canceled {
				t.Errorf("Run must report canceled for input %q", tt.input)
			}
			if restoreCalls != 1 {
				t.Errorf("Restore must be called on cancel, got %d calls", restoreCalls)
			}
		})
	}
}

// panickingWriter injects a panic DURING Run, after raw mode is entered —
// proving the deferred Restore runs on the panic unwind path (threat matrix:
// terminal left in raw state).
type panickingWriter struct{}

func (panickingWriter) Write([]byte) (int, error) { panic("injected panic") }

func TestSelector_RawMode_RestoreOnPanic(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error: %v", err)
	}
	defer func() { _ = r.Close() }()
	_, _ = w.WriteString("x") // never read: the writer panics on first render
	_ = w.Close()

	var restoreCalls int
	oldMakeRaw, oldRestore := makeRawFn, restoreTerm
	makeRawFn = func(fd int) (*term.State, error) { return &term.State{}, nil }
	restoreTerm = func(fd int, _ *term.State) error { restoreCalls++; return nil }
	defer func() { makeRawFn, restoreTerm = oldMakeRaw, oldRestore }()

	sel := NewCheckboxSelector(panickingWriter{}, r, []SelectOption{{ID: "brew", Label: "brew"}})

	func() {
		defer func() {
			if rec := recover(); rec == nil {
				t.Error("Run must propagate the injected panic")
			}
		}()
		_, _ = sel.Run()
		t.Error("Run must panic when the writer panics")
	}()

	if restoreCalls != 1 {
		t.Errorf("Restore must run on panic, got %d calls", restoreCalls)
	}
}

func TestSelector_NonFileReaderSkipsRaw(t *testing.T) {
	var buf bytes.Buffer
	oldMakeRaw, oldRestore := makeRawFn, restoreTerm
	makeRawFn = func(fd int) (*term.State, error) {
		t.Error("MakeRaw must not be called for a non-File reader")
		return &term.State{}, nil
	}
	restoreTerm = func(fd int, _ *term.State) error {
		t.Error("Restore must not be called for a non-File reader")
		return nil
	}
	defer func() { makeRawFn, restoreTerm = oldMakeRaw, oldRestore }()

	sel := NewCheckboxSelector(&buf, strings.NewReader("\r"), []SelectOption{{ID: "brew", Label: "brew"}})
	res, err := sel.Run()
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	if res.Canceled || len(res.Selected) != 1 {
		t.Errorf("unexpected result: %+v", res)
	}
}
