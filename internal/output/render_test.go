package output

import (
	"bytes"
	"io"
	"strings"
	"sync"
	"testing"
)

func TestStatusIcons_ColorMode(t *testing.T) {
	var buf bytes.Buffer
	r := NewRendererForced(&buf, true, true, false, false)

	tests := []struct {
		status Status
		want   string
	}{
		{StatusUpdated, "✅"},
		{StatusSkipped, "⏭️ "},
		{StatusFailed, "❌"},
		{StatusAvailable, "⬆️ "},
		{StatusCurrent, "✔️ "},
		{StatusDeselected, "☐ "},
		{StatusDenied, "🚫"},
		{Status(99), "?"},
	}

	for _, tt := range tests {
		got := r.statusIcon(tt.status)
		if got != tt.want {
			t.Errorf("statusIcon(%d) = %q, want %q", tt.status, got, tt.want)
		}
	}
}

func TestStatusIcons_PlainMode(t *testing.T) {
	var buf bytes.Buffer
	r := NewRendererForced(&buf, false, false, false, false)

	tests := []struct {
		status Status
		want   string
	}{
		{StatusUpdated, "[updated]"},
		{StatusSkipped, "[skipped]"},
		{StatusFailed, "[failed]"},
		{StatusAvailable, "[available]"},
		{StatusCurrent, "[current]"},
		{StatusDeselected, "[deselected]"},
		{StatusDenied, "[denied]"},
		{Status(99), "[?]"},
	}

	for _, tt := range tests {
		got := r.statusIcon(tt.status)
		if got != tt.want {
			t.Errorf("statusIcon(%d) = %q, want %q", tt.status, got, tt.want)
		}
	}
}

func TestColorize_NoColor(t *testing.T) {
	var buf bytes.Buffer
	r := NewRendererForced(&buf, false, false, false, false)

	got := r.colorize("31", "hello")
	if got != "hello" {
		t.Errorf("colorize without color = %q, want %q", got, "hello")
	}
}

func TestColorize_WithColor(t *testing.T) {
	var buf bytes.Buffer
	r := NewRendererForced(&buf, true, false, false, false)

	got := r.colorize("31", "hello")
	want := "\033[31mhello\033[0m"
	if got != want {
		t.Errorf("colorize with color = %q, want %q", got, want)
	}
}

func TestStatusConstants(t *testing.T) {
	// We test the status enum values by checking the constants.
	if StatusUpdated != 0 {
		t.Error("StatusUpdated should be 0")
	}
	if StatusFailed != 2 {
		t.Errorf("StatusFailed should be 2, got %d", StatusFailed)
	}
}

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

func TestCountByStatus(t *testing.T) {
	results := []ToolResult{
		{Status: StatusUpdated},
		{Status: StatusUpdated},
		{Status: StatusSkipped},
		{Status: StatusFailed},
		{Status: StatusFailed},
		{Status: StatusFailed},
	}

	updated, skipped, failed := countByStatus(results)
	if updated != 2 {
		t.Errorf("expected 2 updated, got %d", updated)
	}
	if skipped != 1 {
		t.Errorf("expected 1 skipped, got %d", skipped)
	}
	if failed != 3 {
		t.Errorf("expected 3 failed, got %d", failed)
	}
}

func TestFilterByStatus(t *testing.T) {
	results := []ToolResult{
		{Name: "a", Status: StatusUpdated},
		{Name: "b", Status: StatusFailed},
		{Name: "c", Status: StatusUpdated},
	}

	filtered := filterByStatus(results, StatusUpdated)
	if len(filtered) != 2 {
		t.Errorf("expected 2 filtered results, got %d", len(filtered))
	}
	for _, r := range filtered {
		if r.Status != StatusUpdated {
			t.Errorf("expected StatusUpdated, got %d", r.Status)
		}
	}
}

func TestToolNames(t *testing.T) {
	results := []ToolResult{
		{Name: "brew"},
		{Name: "npm"},
		{Name: "docker"},
	}

	names := toolNames(results)
	if len(names) != 3 {
		t.Errorf("expected 3 names, got %d", len(names))
	}
	if names[0] != "brew" || names[1] != "npm" || names[2] != "docker" {
		t.Errorf("unexpected names: %v", names)
	}
}

func TestNewRenderer_NonTTY(t *testing.T) {
	// A bytes.Buffer is not a TTY, so color should be false
	var buf bytes.Buffer
	r := NewRenderer(&buf, false)

	if r.color {
		t.Error("renderer for non-TTY should not use color")
	}
	if r.emoji {
		t.Error("renderer for non-TTY should not use emoji")
	}
}

func TestNewRenderer_NoColor(t *testing.T) {
	orig := isTerminalFn
	isTerminalFn = func(w io.Writer) bool { return true }
	defer func() { isTerminalFn = orig }()

	t.Setenv("NO_COLOR", "1")
	var buf bytes.Buffer
	r := NewRenderer(&buf, false)
	if r.color {
		t.Error("expected color to be false when NO_COLOR is set")
	}
	if r.emoji {
		t.Error("expected emoji to be false when NO_COLOR is set")
	}

	r.ToolLine(ToolResult{Name: "mytool", Status: StatusUpdated})
	out := buf.String()
	if strings.Contains(out, "\033[") {
		t.Errorf("unexpected ANSI color sequence in output: %q", out)
	}
	if strings.Contains(out, "✅") {
		t.Errorf("unexpected emoji in output: %q", out)
	}
}

func TestNewRenderer_TermDumb(t *testing.T) {
	orig := isTerminalFn
	isTerminalFn = func(w io.Writer) bool { return true }
	defer func() { isTerminalFn = orig }()

	t.Setenv("TERM", "dumb")
	var buf bytes.Buffer
	r := NewRenderer(&buf, false)
	if r.color {
		t.Error("expected color to be false when TERM=dumb")
	}
	if r.emoji {
		t.Error("expected emoji to be false when TERM=dumb")
	}

	r.ToolLine(ToolResult{Name: "mytool", Status: StatusUpdated})
	out := buf.String()
	if strings.Contains(out, "\033[") {
		t.Errorf("unexpected ANSI color sequence in output: %q", out)
	}
	if strings.Contains(out, "✅") {
		t.Errorf("unexpected emoji in output: %q", out)
	}
}

func TestRenderer_Color(t *testing.T) {
	var buf bytes.Buffer
	rNoColor := NewRendererForced(&buf, false, false, false, false)
	if rNoColor.Color() {
		t.Error("Color() should be false for forced non-color renderer")
	}

	rColor := NewRendererForced(&buf, true, false, false, false)
	if !rColor.Color() {
		t.Error("Color() should be true for forced color renderer")
	}
}

func TestRenderer_ErrorAndWarning(t *testing.T) {
	var buf bytes.Buffer
	r := NewRendererForced(&buf, false, false, false, false)

	r.Error("fatal failure")
	if !strings.Contains(buf.String(), "[failed] fatal failure") {
		t.Errorf("Error() output missing expected content, got: %s", buf.String())
	}

	buf.Reset()
	r.Warning("potential issue")
	if !strings.Contains(buf.String(), "⚠️  potential issue") {
		t.Errorf("Warning() output missing expected content, got: %s", buf.String())
	}
}

func TestRenderer_LogPresenterInterface(t *testing.T) {
	var buf bytes.Buffer
	var lp LogPresenter = NewRendererForced(&buf, false, false, false, false)

	lp.Error("test error")
	lp.Warning("test warning")

	out := buf.String()
	if !strings.Contains(out, "test error") || !strings.Contains(out, "test warning") {
		t.Errorf("LogPresenter interface test missing expected output, got: %s", out)
	}
}

func TestRenderer_ConcurrentThreadSafety(t *testing.T) {
	var buf bytes.Buffer
	r := NewRendererForced(&buf, true, true, false, false)

	var wg sync.WaitGroup
	const workers = 10
	const iterations = 20
	wg.Add(workers * 4)

	for w := 0; w < workers; w++ {
		go func(id int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				r.ToolLine(ToolResult{Name: "brew", Status: StatusUpdated})
			}
		}(w)
		go func(id int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				r.Error("something went wrong")
			}
		}(w)
		go func(id int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				r.Warning("potential issue")
			}
		}(w)
		go func(id int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				r.DryRunPlanned("cargo")
			}
		}(w)
	}

	wg.Wait()
	if buf.Len() == 0 {
		t.Error("expected non-empty output from concurrent operations")
	}
}
