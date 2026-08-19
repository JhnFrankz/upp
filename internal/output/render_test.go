package output

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
	"testing"
)

func TestStatusIcons_ColorMode(t *testing.T) {
	r := NewRendererForced(&bytes.Buffer{}, true, true, false, false)

	tests := []struct {
		status   Status
		expected string
	}{
		{StatusUpdated, "✅"},
		{StatusSkipped, "⏭️ "},
		{StatusFailed, "❌"},
		{StatusAvailable, "⬆️ "},
		{StatusCurrent, "✔️ "},
	}

	for _, tt := range tests {
		icon := r.statusIcon(tt.status)
		if icon != tt.expected {
			t.Errorf("statusIcon(%d) = %q, want %q", tt.status, icon, tt.expected)
		}
	}
}

func TestStatusIcons_PlainMode(t *testing.T) {
	r := NewRendererForced(&bytes.Buffer{}, false, false, false, false)

	tests := []struct {
		status   Status
		expected string
	}{
		{StatusUpdated, "[updated]"},
		{StatusSkipped, "[skipped]"},
		{StatusFailed, "[failed]"},
		{StatusAvailable, "[available]"},
		{StatusCurrent, "[current]"},
	}

	for _, tt := range tests {
		icon := r.statusIcon(tt.status)
		if icon != tt.expected {
			t.Errorf("statusIcon(%d) = %q, want %q", tt.status, icon, tt.expected)
		}
	}
}

func TestColorize_NoColor(t *testing.T) {
	r := NewRendererForced(&bytes.Buffer{}, false, false, false, false)
	result := r.colorize("31", "hello")
	if result != "hello" {
		t.Errorf("colorize with no color should return plain text, got %q", result)
	}
}

func TestColorize_WithColor(t *testing.T) {
	r := NewRendererForced(&bytes.Buffer{}, true, false, false, false)
	result := r.colorize("31", "hello")
	if result != "\033[31mhello\033[0m" {
		t.Errorf("colorize with color should wrap in ANSI codes, got %q", result)
	}
}

func TestToolLine_Verbose(t *testing.T) {
	var buf bytes.Buffer
	r := NewRendererForced(&buf, false, true, false, false)

	r.ToolLine(ToolResult{
		Name:    "brew",
		Status:  StatusUpdated,
		Version: "4.1.0",
	})

	output := buf.String()
	if !strings.Contains(output, "brew") {
		t.Error("output should contain tool name")
	}
	if !strings.Contains(output, "4.1.0") {
		t.Error("output should contain version")
	}
	if !strings.Contains(output, "✅") {
		t.Error("output should contain status icon")
	}
}

func TestToolLine_Quiet(t *testing.T) {
	var buf bytes.Buffer
	r := NewRendererForced(&buf, false, true, true, false)

	r.ToolLine(ToolResult{
		Name:   "brew",
		Status: StatusSkipped,
	})

	output := buf.String()
	if !strings.Contains(output, "brew") {
		t.Error("output should contain tool name")
	}
	// Quiet mode should not contain verbose text like "(not installed)"
	if strings.Contains(output, "(not installed)") {
		t.Error("quiet mode should not contain verbose text")
	}
}

func TestToolLine_FailedWithError(t *testing.T) {
	var buf bytes.Buffer
	r := NewRendererForced(&buf, false, true, false, false)

	r.ToolLine(ToolResult{
		Name:   "npm",
		Status: StatusFailed,
		Error:  fmt.Errorf("network error"),
	})

	output := buf.String()
	if !strings.Contains(output, "npm") {
		t.Error("output should contain tool name")
	}
	if !strings.Contains(output, "❌") {
		t.Error("output should contain failed icon")
	}
}

func TestUpdateSummary_AllUpdated(t *testing.T) {
	var buf bytes.Buffer
	r := NewRendererForced(&buf, false, true, false, false)

	summary := Summary{
		Results: []ToolResult{
			{Name: "brew", Status: StatusUpdated, Version: "4.1.0"},
			{Name: "npm", Status: StatusUpdated, Version: "10.0.0"},
		},
		DryRun: false,
	}

	r.UpdateSummary(summary)

	output := buf.String()
	if !strings.Contains(output, "All clean!") {
		t.Errorf("summary should contain 'All clean!', got:\n%s", output)
	}
}

func TestUpdateSummary_PartialFailure(t *testing.T) {
	var buf bytes.Buffer
	r := NewRendererForced(&buf, false, true, false, false)

	summary := Summary{
		Results: []ToolResult{
			{Name: "brew", Status: StatusUpdated, Version: "4.1.0"},
			{Name: "npm", Status: StatusFailed, Error: fmt.Errorf("timeout")},
		},
		DryRun: false,
	}

	r.UpdateSummary(summary)

	output := buf.String()
	if !strings.Contains(output, "failed") {
		t.Errorf("summary should contain 'failed', got:\n%s", output)
	}
	if !strings.Contains(output, "Review errors above") {
		t.Errorf("summary should contain 'Review errors above', got:\n%s", output)
	}
}

func TestUpdateSummary_AllSkipped(t *testing.T) {
	var buf bytes.Buffer
	r := NewRendererForced(&buf, false, true, false, false)

	summary := Summary{
		Results: []ToolResult{
			{Name: "brew", Status: StatusSkipped},
			{Name: "npm", Status: StatusSkipped},
		},
		DryRun: false,
	}

	r.UpdateSummary(summary)

	output := buf.String()
	if !strings.Contains(output, "All tools not installed") {
		t.Errorf("summary should contain 'All tools not installed', got:\n%s", output)
	}
}

func TestUpdateSummary_DryRun(t *testing.T) {
	var buf bytes.Buffer
	r := NewRendererForced(&buf, false, true, false, false)

	summary := Summary{
		Results: []ToolResult{
			{Name: "brew", Status: StatusAvailable, Version: "4.0.0 → 4.1.0"},
		},
		DryRun: true,
	}

	r.UpdateSummary(summary)

	output := buf.String()
	if !strings.Contains(output, "would update") {
		t.Errorf("dry-run summary should say 'would update', got:\n%s", output)
	}
	if strings.Contains(output, "All clean!") {
		t.Errorf("dry-run summary must never claim 'All clean!' while updates are pending, got:\n%s", output)
	}
}

func TestUpdateSummary_NotCleanWithSkips(t *testing.T) {
	var buf bytes.Buffer
	r := NewRendererForced(&buf, false, true, false, false)

	summary := Summary{
		Results: []ToolResult{
			{Name: "brew", Status: StatusUpdated, Version: "4.1.0"},
			{Name: "npm", Status: StatusSkipped},
		},
		DryRun: false,
	}

	r.UpdateSummary(summary)

	output := buf.String()
	if !strings.Contains(output, "1 updated") {
		t.Errorf("summary should count 1 updated, got:\n%s", output)
	}
	if strings.Contains(output, "All clean!") {
		t.Errorf("summary must not claim 'All clean!' when tools were skipped, got:\n%s", output)
	}
}

func TestCheckSummary_UpdatesAvailable(t *testing.T) {
	var buf bytes.Buffer
	r := NewRendererForced(&buf, false, true, false, false)

	results := []ToolResult{
		{Name: "brew", Status: StatusAvailable, Version: "4.0.0 → 4.1.0"},
		{Name: "npm", Status: StatusCurrent, Version: "10.0.0"},
	}

	r.CheckSummary(results)

	output := buf.String()
	if !strings.Contains(output, "1 available") {
		t.Errorf("check summary should show '1 available', got:\n%s", output)
	}
	// Current tools are "up to date", never "current" (D4 wording).
	if !strings.Contains(output, "1 up to date") {
		t.Errorf("check summary should show '1 up to date', got:\n%s", output)
	}
	if strings.Contains(output, "1 current") {
		t.Errorf("check summary must say 'up to date' not 'current', got:\n%s", output)
	}
}

func TestCheckSummary_AllCurrent(t *testing.T) {
	var buf bytes.Buffer
	r := NewRendererForced(&buf, false, true, false, false)

	results := []ToolResult{
		{Name: "brew", Status: StatusCurrent, Version: "4.1.0"},
		{Name: "npm", Status: StatusCurrent, Version: "10.0.0"},
	}

	r.CheckSummary(results)

	output := buf.String()
	if !strings.Contains(output, "All tools up to date") {
		t.Errorf("check summary should say 'All tools up to date', got:\n%s", output)
	}
}

// TestCheckSummary_CurrentAndSkipped locks the D4 honesty contract: when an
// enabled tool is skipped (not installed), the check summary must count it
// explicitly and MUST NOT print the "All tools up to date." tagline.
func TestCheckSummary_CurrentAndSkipped(t *testing.T) {
	var buf bytes.Buffer
	r := NewRendererForced(&buf, false, true, false, false)

	results := []ToolResult{
		{Name: "apt", Status: StatusCurrent, Version: "2.4.0"},
		{Name: "npm", Status: StatusSkipped},
	}

	r.CheckSummary(results)

	output := buf.String()
	if !strings.Contains(output, "1 up to date, 1 skipped") {
		t.Errorf("check summary must count skipped tools explicitly, got:\n%s", output)
	}
	if strings.Contains(output, "All tools up to date.") {
		t.Errorf("check must never claim 'All tools up to date.' when a tool was skipped, got:\n%s", output)
	}
	// Non-quiet detail lists the skipped tools too (D4).
	if !strings.Contains(output, "npm") {
		t.Errorf("non-quiet check detail should list skipped tools, got:\n%s", output)
	}
}

// TestCheckSummary_AllSkipped locks the all-skipped branch: with nothing
// current, available, or failed, check must say "Nothing to do." — never the
// up-to-date tagline.
func TestCheckSummary_AllSkipped(t *testing.T) {
	var buf bytes.Buffer
	r := NewRendererForced(&buf, false, true, false, false)

	results := []ToolResult{
		{Name: "brew", Status: StatusSkipped},
		{Name: "docker", Status: StatusSkipped},
	}

	r.CheckSummary(results)

	output := buf.String()
	if !strings.Contains(output, "Nothing to do") {
		t.Errorf("all-skipped check summary should say 'Nothing to do.', got:\n%s", output)
	}
	if strings.Contains(output, "All tools up to date.") {
		t.Errorf("all-skipped check must never claim 'All tools up to date.', got:\n%s", output)
	}
}

// TestCheckSummary_EmptyResults preserves the empty enabled-tool list case:
// with zero results the tagline stays (parser_test.go bare-upp and the
// all-tools-disabled integration test depend on it).
func TestCheckSummary_EmptyResults(t *testing.T) {
	var buf bytes.Buffer
	r := NewRendererForced(&buf, false, true, false, false)

	r.CheckSummary(nil)

	output := buf.String()
	if !strings.Contains(output, "All tools up to date.") {
		t.Errorf("empty check must keep the 'All tools up to date.' tagline, got:\n%s", output)
	}
}

// TestCheckSummary_AvailableAndSkipped triangulates the parts path: pending
// updates plus skipped tools are both counted, never the tagline.
func TestCheckSummary_AvailableAndSkipped(t *testing.T) {
	var buf bytes.Buffer
	r := NewRendererForced(&buf, false, true, false, false)

	results := []ToolResult{
		{Name: "apt", Status: StatusAvailable, Version: "2.4.0 → 2.4.5"},
		{Name: "npm", Status: StatusSkipped},
	}

	r.CheckSummary(results)

	output := buf.String()
	if !strings.Contains(output, "1 available, 1 skipped") {
		t.Errorf("check summary must count available and skipped tools, got:\n%s", output)
	}
	if strings.Contains(output, "All tools up to date.") {
		t.Errorf("check must never claim 'All tools up to date.' when a tool was skipped, got:\n%s", output)
	}
}

// TestCheckSummary_CurrentAndFailed triangulates the parts path with a
// failure: current tools are still counted as up to date alongside failed.
func TestCheckSummary_CurrentAndFailed(t *testing.T) {
	var buf bytes.Buffer
	r := NewRendererForced(&buf, false, true, false, false)

	results := []ToolResult{
		{Name: "apt", Status: StatusCurrent, Version: "2.4.0"},
		{Name: "npm", Status: StatusFailed, Error: fmt.Errorf("timeout")},
	}

	r.CheckSummary(results)

	output := buf.String()
	if !strings.Contains(output, "1 up to date, 1 failed") {
		t.Errorf("check summary must count current and failed tools, got:\n%s", output)
	}
	if strings.Contains(output, "All tools up to date.") {
		t.Errorf("check must never claim 'All tools up to date.' when a tool failed, got:\n%s", output)
	}
}

// TestCheckSummary_UnknownStatusFailsClosed guards the branch comments in
// CheckSummary: a status outside the known enum must never fall through to
// the "All tools up to date." tagline (the branches assume the switch counts
// every status; a future Status value must fail closed, not claim clean).
func TestCheckSummary_UnknownStatusFailsClosed(t *testing.T) {
	var buf bytes.Buffer
	r := NewRendererForced(&buf, false, true, false, false)

	results := []ToolResult{
		{Name: "mystery", Status: Status(99)},
	}

	r.CheckSummary(results)

	output := buf.String()
	if strings.Contains(output, "All tools up to date.") {
		t.Errorf("check must never claim 'All tools up to date.' for an unknown status, got:\n%s", output)
	}
	if !strings.Contains(output, "1 failed") {
		t.Errorf("unknown status must fail closed as failed, got:\n%s", output)
	}
}

func TestListTools(t *testing.T) {
	var buf bytes.Buffer
	r := NewRendererForced(&buf, false, true, false, false)

	entries := []ListEntry{
		{ID: "brew", Name: "Homebrew", Status: StatusCurrent, Version: "4.1.0"},
		{ID: "npm", Name: "npm", Status: StatusSkipped, Version: ""},
	}

	r.ListTools(entries)

	output := buf.String()
	if !strings.Contains(output, "Homebrew") {
		t.Error("list should contain tool name")
	}
	if !strings.Contains(output, "4.1.0") {
		t.Error("list should contain version")
	}
}

func TestListTools_IDColumn(t *testing.T) {
	var buf bytes.Buffer
	r := NewRendererForced(&buf, false, true, false, false)

	entries := []ListEntry{
		{ID: "brew", Name: "Homebrew", Status: StatusCurrent, Version: "4.1.0"},
		{ID: "npm", Name: "npm", Status: StatusSkipped, Version: ""},
	}

	r.ListTools(entries)

	output := buf.String()
	header := strings.SplitN(output, "\n", 2)[0]
	if !strings.Contains(header, "ID") || !strings.Contains(header, "Name") ||
		!strings.Contains(header, "Status") || !strings.Contains(header, "Version") {
		t.Errorf("list header must be 'ID | Name | Status | Version', got: %q", header)
	}
	if strings.Contains(header, "Tool") {
		t.Errorf("list header must not mislabel the ID column as 'Tool', got: %q", header)
	}
	// The ID column shows the --only/--skip filter IDs, distinct from the
	// display name.
	if !strings.Contains(output, "brew") || !strings.Contains(output, "npm") {
		t.Errorf("list rows must show tool IDs usable with --only/--skip, got:\n%s", output)
	}
}

func TestDryRunHeader(t *testing.T) {
	var buf bytes.Buffer
	r := NewRendererForced(&buf, false, true, false, false)

	r.DryRunHeader()

	output := buf.String()
	if !strings.Contains(output, "Dry run") {
		t.Errorf("dry run header should contain 'Dry run', got:\n%s", output)
	}
}

func TestProgress_SingleTool(t *testing.T) {
	var buf bytes.Buffer
	r := NewRendererForced(&buf, false, true, false, false)

	// Single tool should not show progress
	r.Progress("Checking", 1, 1, "brew")

	if buf.Len() > 0 {
		t.Error("progress should not show for single tool")
	}
}

func TestProgress_CheckVerb(t *testing.T) {
	var buf bytes.Buffer
	r := NewRendererForced(&buf, false, true, false, false)

	r.Progress("Checking", 2, 5, "brew")

	output := buf.String()
	if !strings.Contains(output, "Checking 2/5") {
		t.Errorf("progress should say 'Checking 2/5', got:\n%s", output)
	}
	if !strings.Contains(output, "brew") {
		t.Errorf("progress should contain tool name, got:\n%s", output)
	}
	if strings.Contains(output, "Updating") {
		t.Errorf("read-only progress must not say 'Updating', got:\n%s", output)
	}
}

func TestProgress_UpdateVerb(t *testing.T) {
	var buf bytes.Buffer
	r := NewRendererForced(&buf, false, true, false, false)

	r.Progress("Updating", 3, 10, "npm")

	output := buf.String()
	if !strings.Contains(output, "Updating 3/10") {
		t.Errorf("progress should say 'Updating 3/10', got:\n%s", output)
	}
	if !strings.Contains(output, "npm") {
		t.Errorf("progress should contain tool name, got:\n%s", output)
	}
	if strings.Contains(output, "Checking") {
		t.Errorf("update progress must not say 'Checking', got:\n%s", output)
	}
}

func TestSelfUpdatePrompt(t *testing.T) {
	var buf bytes.Buffer
	r := NewRenderer(&buf, false)
	r.SelfUpdatePrompt("v0.1.0", "v0.1.1", "/home/u/.local/bin/upp")

	want := "  Update upp from v0.1.0 to v0.1.1?\n    Target: /home/u/.local/bin/upp\n  Proceed? [y/N] "
	if got := buf.String(); got != want {
		t.Errorf("prompt = %q, want %q", got, want)
	}
}

func TestSelfUpdatePrompt_QuietNeverSuppresses(t *testing.T) {
	var buf bytes.Buffer
	r := NewRenderer(&buf, true)
	r.SelfUpdatePrompt("v0.1.0", "v0.1.1", "/home/u/.local/bin/upp")

	if got := buf.String(); !strings.Contains(got, "Proceed?") {
		t.Errorf("prompt must never be suppressed by quiet mode, got %q", got)
	}
}

func TestSelfUpdateMessages(t *testing.T) {
	var buf bytes.Buffer
	r := NewRenderer(&buf, false)
	r.SelfUpdateDevBuild()
	r.SelfUpdateUpToDate("v0.1.1")
	r.SelfUpdateDone("v0.1.0", "v0.1.1")

	want := "development build; self-update is only available for release builds\n" +
		"already up to date (v0.1.1)\n" +
		"upp updated: v0.1.0 → v0.1.1\n"
	if got := buf.String(); got != want {
		t.Errorf("messages = %q, want %q", got, want)
	}
}

func TestSelfUpdateHint(t *testing.T) {
	var buf bytes.Buffer
	r := NewRenderer(&buf, false)
	r.SelfUpdateHint("v0.1.0", "v0.1.1")

	want := "⬆️ upp v0.1.1 available (current v0.1.0) — run \"upp self-update\"\n"
	if got := buf.String(); got != want {
		t.Errorf("hint = %q, want %q", got, want)
	}
}

// TestSelfUpdateHint_QuietSuppresses locks the ONE place quiet mode
// applies to self-update output: the hint is informational output
// (unlike the confirm prompt, which quiet never suppresses).
func TestSelfUpdateHint_QuietSuppresses(t *testing.T) {
	var buf bytes.Buffer
	r := NewRenderer(&buf, true)
	r.SelfUpdateHint("v0.1.0", "v0.1.1")

	if got := buf.String(); got != "" {
		t.Errorf("quiet mode must suppress the hint, got %q", got)
	}
}

// TestUpdateCancelled locks the fixed cancel message for interactive update
// runs (design D8, spec ux-patterns "Esc cancels run" / "q cancels run"):
// "Update canceled — no changes made." (US spelling per repo misspell locale)
func TestUpdateCancelled(t *testing.T) {
	var buf bytes.Buffer
	r := NewRendererForced(&buf, false, false, false, false)

	r.UpdateCancelled()

	want := "Update canceled — no changes made.\n"
	if got := buf.String(); got != want {
		t.Errorf("UpdateCancelled() = %q, want %q", got, want)
	}
}

func TestStatusFromResult(t *testing.T) {
	// We test the mapping logic by checking the constants.
	if StatusUpdated != 0 {
		t.Error("StatusUpdated should be 0")
	}
	if StatusFailed != 2 {
		t.Errorf("StatusFailed should be 2, got %d", StatusFailed)
	}
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

func TestRenderer_ConcurrentProgress_ThreadSafe(t *testing.T) {
	var buf bytes.Buffer
	r := NewRendererForced(&buf, true, true, false, false)

	const goroutines = 20
	const iterations = 50
	var wg sync.WaitGroup
	wg.Add(goroutines * 2)

	for g := 0; g < goroutines; g++ {
		go func(id int) {
			defer wg.Done()
			for i := 1; i <= iterations; i++ {
				r.Progress("Checking", i, iterations, fmt.Sprintf("tool-%d", id))
			}
		}(g)
		go func(id int) {
			defer wg.Done()
			for i := 1; i <= iterations; i++ {
				r.ProgressInPlace("Checking", i, iterations, fmt.Sprintf("tool-%d", id))
			}
		}(g)
	}

	wg.Wait()
	if buf.Len() == 0 {
		t.Errorf("expected non-empty output from concurrent progress writes")
	}
}

func TestProgressInPlace_TTY(t *testing.T) {
	var buf bytes.Buffer
	r := NewRendererForced(&buf, true, true, false, false)

	r.ProgressInPlace("Checking", 2, 5, "brew")

	output := buf.String()
	if !strings.HasPrefix(output, "\r") {
		t.Errorf("TTY ProgressInPlace must start with carriage return \\r, got: %q", output)
	}
	if strings.HasSuffix(output, "\n") {
		t.Errorf("TTY ProgressInPlace must not end with newline \\n, got: %q", output)
	}
	if !strings.Contains(output, "Checking 2/5") || !strings.Contains(output, "brew") {
		t.Errorf("TTY ProgressInPlace must contain operation and tool, got: %q", output)
	}
}

func TestProgressInPlace_NonTTY(t *testing.T) {
	var buf bytes.Buffer
	r := NewRendererForced(&buf, false, false, false, false)

	r.ProgressInPlace("Checking", 2, 5, "brew")

	output := buf.String()
	if strings.Contains(output, "\r") {
		t.Errorf("Non-TTY ProgressInPlace must not contain carriage return \\r, got: %q", output)
	}
	if !strings.HasSuffix(output, "\n") {
		t.Errorf("Non-TTY ProgressInPlace must end with newline \\n, got: %q", output)
	}
	if !strings.Contains(output, "Checking 2/5") || !strings.Contains(output, "brew") {
		t.Errorf("Non-TTY ProgressInPlace must contain operation and tool, got: %q", output)
	}
}

func TestProgressInPlace_SingleTool(t *testing.T) {
	var buf bytes.Buffer
	r := NewRendererForced(&buf, true, true, false, false)

	r.ProgressInPlace("Checking", 1, 1, "brew")

	if buf.Len() > 0 {
		t.Errorf("ProgressInPlace should not show for single tool, got: %q", buf.String())
	}
}

func TestProgressInPlace_Quiet(t *testing.T) {
	var buf bytes.Buffer
	r := NewRendererForced(&buf, true, true, true, false)

	r.ProgressInPlace("Checking", 2, 5, "brew")

	if buf.Len() > 0 {
		t.Errorf("ProgressInPlace should be suppressed in quiet mode, got: %q", buf.String())
	}
}

func TestDashboard_Formatting(t *testing.T) {
	var buf bytes.Buffer
	r := NewRendererForced(&buf, false, false, false, false)

	data := DashboardData{
		Version:        "v0.2.0",
		Platform:       "linux/amd64",
		EnabledTools:   5,
		AvailableTools: 10,
	}

	r.Dashboard(data)

	out := buf.String()
	for _, want := range []string{
		"upp v0.2.0 (linux/amd64)",
		"Tools: 5 enabled (10 configured for platform)",
		"Commands:",
		"upp check",
		"upp update",
		"upp list",
		"upp --help",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("Dashboard output missing %q, got:\n%s", want, out)
		}
	}
}

func TestDashboard_QuietSuppresses(t *testing.T) {
	var buf bytes.Buffer
	r := NewRendererForced(&buf, true, true, true, false)

	data := DashboardData{
		Version:        "v0.2.0",
		Platform:       "linux/amd64",
		EnabledTools:   5,
		AvailableTools: 10,
	}

	r.Dashboard(data)

	if buf.Len() > 0 {
		t.Errorf("Dashboard in quiet mode must produce no output, got: %q", buf.String())
	}
}

func TestDashboardNoConfig(t *testing.T) {
	var buf bytes.Buffer
	r := NewRendererForced(&buf, false, false, false, false)

	r.DashboardNoConfig("v0.2.0", "linux/amd64")

	out := buf.String()
	for _, want := range []string{
		"upp v0.2.0 (linux/amd64)",
		"No configuration found.",
		"Run \"upp init\" to detect installed tools and initialize your config.",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("DashboardNoConfig output missing %q, got:\n%s", want, out)
		}
	}
}

func TestDashboardNoConfig_QuietSuppresses(t *testing.T) {
	var buf bytes.Buffer
	r := NewRendererForced(&buf, true, true, true, false)

	r.DashboardNoConfig("v0.2.0", "linux/amd64")

	if buf.Len() > 0 {
		t.Errorf("DashboardNoConfig in quiet mode must produce no output, got: %q", buf.String())
	}
}

func TestDashboard_PlainNonTTY(t *testing.T) {
	var buf bytes.Buffer
	r := NewRendererForced(&buf, false, false, false, false)

	r.Dashboard(DashboardData{
		Version:        "v0.2.0",
		Platform:       "linux/amd64",
		EnabledTools:   3,
		AvailableTools: 3,
	})

	out := buf.String()
	if strings.Contains(out, "\033[") {
		t.Errorf("Plain non-TTY dashboard must not contain ANSI escape codes, got: %q", out)
	}
}

func TestToolLine_VerboseFailureDiagnostics(t *testing.T) {
	var buf bytes.Buffer
	r := NewRendererForced(&buf, false, false, false, true)

	r.ToolLine(ToolResult{
		Name:   "npm",
		Status: StatusFailed,
		Error:  fmt.Errorf("exit 1"),
		Stderr: "npm ERR! network connection refused\nnpm ERR! retry limit reached",
	})

	out := buf.String()
	if !strings.Contains(out, "npm ERR! network connection refused") {
		t.Errorf("expected verbose tool line to contain stderr, got:\n%s", out)
	}
	if !strings.Contains(out, "npm ERR! retry limit reached") {
		t.Errorf("expected verbose tool line to contain stderr line 2, got:\n%s", out)
	}
}

func TestToolLine_NonVerboseFailureSuppressed(t *testing.T) {
	var buf bytes.Buffer
	r := NewRendererForced(&buf, false, false, false, false)

	r.ToolLine(ToolResult{
		Name:   "npm",
		Status: StatusFailed,
		Error:  fmt.Errorf("exit 1"),
		Stderr: "npm ERR! network connection refused",
	})

	out := buf.String()
	if strings.Contains(out, "npm ERR! network connection refused") {
		t.Errorf("expected non-verbose tool line to suppress stderr, got:\n%s", out)
	}
}

func TestToolLine_QuietOverridesVerbose(t *testing.T) {
	var buf bytes.Buffer
	r := NewRendererForced(&buf, false, false, true, true)

	r.ToolLine(ToolResult{
		Name:   "npm",
		Status: StatusFailed,
		Error:  fmt.Errorf("exit 1"),
		Stderr: "npm ERR! network connection refused",
	})

	out := buf.String()
	if strings.Contains(out, "npm ERR! network connection refused") {
		t.Errorf("expected quiet mode to override verbose and suppress stderr, got:\n%s", out)
	}
}

func TestUpdateSummary_VerboseFailureDiagnostics(t *testing.T) {
	var buf bytes.Buffer
	r := NewRendererForced(&buf, false, false, false, true)

	summary := Summary{
		Results: []ToolResult{
			{Name: "brew", Status: StatusUpdated, Version: "4.1.0"},
			{
				Name:   "npm",
				Status: StatusFailed,
				Error:  fmt.Errorf("timeout"),
				Stderr: "npm ERR! lock frontend held by another process",
			},
		},
		DryRun: false,
	}

	r.UpdateSummary(summary)

	out := buf.String()
	if !strings.Contains(out, "lock frontend held by another process") {
		t.Errorf("expected verbose update summary to contain stderr, got:\n%s", out)
	}
}
