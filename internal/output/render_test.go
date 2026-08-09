package output

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func TestStatusIcons_ColorMode(t *testing.T) {
	r := NewRendererForced(&bytes.Buffer{}, true, true, false)

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
	r := NewRendererForced(&bytes.Buffer{}, false, false, false)

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
	r := NewRendererForced(&bytes.Buffer{}, false, false, false)
	result := r.colorize("31", "hello")
	if result != "hello" {
		t.Errorf("colorize with no color should return plain text, got %q", result)
	}
}

func TestColorize_WithColor(t *testing.T) {
	r := NewRendererForced(&bytes.Buffer{}, true, false, false)
	result := r.colorize("31", "hello")
	if result != "\033[31mhello\033[0m" {
		t.Errorf("colorize with color should wrap in ANSI codes, got %q", result)
	}
}

func TestToolLine_Verbose(t *testing.T) {
	var buf bytes.Buffer
	r := NewRendererForced(&buf, false, true, false)

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
	r := NewRendererForced(&buf, false, true, true)

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
	r := NewRendererForced(&buf, false, true, false)

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
	r := NewRendererForced(&buf, false, true, false)

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
	r := NewRendererForced(&buf, false, true, false)

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
	r := NewRendererForced(&buf, false, true, false)

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
	r := NewRendererForced(&buf, false, true, false)

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
}

func TestCheckSummary_UpdatesAvailable(t *testing.T) {
	var buf bytes.Buffer
	r := NewRendererForced(&buf, false, true, false)

	results := []ToolResult{
		{Name: "brew", Status: StatusAvailable, Version: "4.0.0 → 4.1.0"},
		{Name: "npm", Status: StatusCurrent, Version: "10.0.0"},
	}

	r.CheckSummary(results)

	output := buf.String()
	if !strings.Contains(output, "1 available") {
		t.Errorf("check summary should show '1 available', got:\n%s", output)
	}
	if !strings.Contains(output, "1 current") {
		t.Errorf("check summary should show '1 current', got:\n%s", output)
	}
}

func TestCheckSummary_AllCurrent(t *testing.T) {
	var buf bytes.Buffer
	r := NewRendererForced(&buf, false, true, false)

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

func TestListTools(t *testing.T) {
	var buf bytes.Buffer
	r := NewRendererForced(&buf, false, true, false)

	entries := []ListEntry{
		{Name: "Homebrew", Status: StatusCurrent, Version: "4.1.0"},
		{Name: "npm", Status: StatusSkipped, Version: ""},
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

func TestDryRunHeader(t *testing.T) {
	var buf bytes.Buffer
	r := NewRendererForced(&buf, false, true, false)

	r.DryRunHeader()

	output := buf.String()
	if !strings.Contains(output, "Dry run") {
		t.Errorf("dry run header should contain 'Dry run', got:\n%s", output)
	}
}

func TestProgress_SingleTool(t *testing.T) {
	var buf bytes.Buffer
	r := NewRendererForced(&buf, false, true, false)

	// Single tool should not show progress
	r.Progress(1, 1, "brew")

	if buf.Len() > 0 {
		t.Error("progress should not show for single tool")
	}
}

func TestProgress_MultiTool(t *testing.T) {
	var buf bytes.Buffer
	r := NewRendererForced(&buf, false, true, false)

	r.Progress(2, 5, "brew")

	output := buf.String()
	if !strings.Contains(output, "2/5") {
		t.Errorf("progress should contain '2/5', got:\n%s", output)
	}
	if !strings.Contains(output, "brew") {
		t.Errorf("progress should contain tool name, got:\n%s", output)
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
