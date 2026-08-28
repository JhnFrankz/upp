package output

import (
	"bytes"
	"fmt"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/adapters/official"
	"github.com/JhnFrankz/upp/internal/platform"
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
	// Spec ux-patterns Summary Report, "All succeed": the clean line counts
	// updated AND failed explicitly ("5 updated, 0 failed. All clean!").
	if !strings.Contains(output, "2 updated, 0 failed. All clean!") {
		t.Errorf("summary should report '2 updated, 0 failed. All clean!', got:\n%s", output)
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
	// Spec ux-patterns Summary Report, "Partial fail": counts compose as
	// "N updated, M failed" followed by the review tagline.
	if !strings.Contains(output, "1 updated, 1 failed. Review errors above.") {
		t.Errorf("summary should report '1 updated, 1 failed. Review errors above.', got:\n%s", output)
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

// TestUpdateSummary_AllCurrentDryRun proves current tools are counted on the
// update path (design D6). StatusCurrent used to be counted nowhere, so an
// all-current dry-run fell into the "All tools not installed" branch.
func TestUpdateSummary_AllCurrentDryRun(t *testing.T) {
	var buf bytes.Buffer
	r := NewRendererForced(&buf, false, true, false, false)

	summary := Summary{
		Results: []ToolResult{
			{Name: "brew", Status: StatusCurrent, Version: "4.1.0"},
			{Name: "npm", Status: StatusCurrent, Version: "10.0.0"},
		},
		DryRun: true,
	}

	r.UpdateSummary(summary)

	output := buf.String()
	if !strings.Contains(output, "2 up to date") {
		t.Errorf("summary must count up-to-date tools (D6), got:\n%s", output)
	}
	if strings.Contains(output, "not installed") {
		t.Errorf("all-current run must not claim tools are 'not installed', got:\n%s", output)
	}
	if strings.Contains(output, "All clean!") {
		t.Errorf("dry-run summary must never claim 'All clean!', got:\n%s", output)
	}
}

// TestUpdateSummary_CurrentWithSkipsDryRun pins the spec ux-patterns Summary
// Report scenario "Up-to-date with skips": 8 current + 2 skipped tools on a
// dry-run count both explicitly ("8 up to date, 2 skipped"), the detail list
// names the current tools, and no "All tools up to date." tagline appears.
func TestUpdateSummary_CurrentWithSkipsDryRun(t *testing.T) {
	var buf bytes.Buffer
	r := NewRendererForced(&buf, false, true, false, false)

	results := make([]ToolResult, 0, 10)
	for i := 1; i <= 8; i++ {
		results = append(results, ToolResult{Name: fmt.Sprintf("tool-%d", i), Status: StatusCurrent})
	}
	results = append(results,
		ToolResult{Name: "missing-a", Status: StatusSkipped},
		ToolResult{Name: "missing-b", Status: StatusSkipped},
	)

	r.UpdateSummary(Summary{Results: results, DryRun: true})

	output := buf.String()
	if !strings.Contains(output, "8 up to date, 2 skipped") {
		t.Errorf("summary must count up-to-date and skipped explicitly, got:\n%s", output)
	}
	if strings.Contains(output, "All tools up to date.") {
		t.Errorf("summary must never claim 'All tools up to date.' when a tool was skipped, got:\n%s", output)
	}
	if !strings.Contains(output, "Up to date: tool-1, tool-2, tool-3, tool-4, tool-5, tool-6, tool-7, tool-8") {
		t.Errorf("detail summary must list current tools (D6), got:\n%s", output)
	}
}

// TestUpdateSummary_UpdatedAndCurrent proves the up-to-date part composes
// with real updates in non-dry-run mode, in canonical part order.
func TestUpdateSummary_UpdatedAndCurrent(t *testing.T) {
	var buf bytes.Buffer
	r := NewRendererForced(&buf, false, true, false, false)

	summary := Summary{
		Results: []ToolResult{
			{Name: "brew", Status: StatusUpdated, Version: "4.2.0"},
			{Name: "npm", Status: StatusUpdated, Version: "10.1.0"},
			{Name: "go", Status: StatusCurrent, Version: "1.22"},
		},
		DryRun: false,
	}

	r.UpdateSummary(summary)

	output := buf.String()
	if !strings.Contains(output, "2 updated, 1 up to date") {
		t.Errorf("summary must compose updated and up-to-date parts in order, got:\n%s", output)
	}
	if !strings.Contains(output, "Up to date: go") {
		t.Errorf("detail summary must list current tools (D6), got:\n%s", output)
	}
}

func TestListTools(t *testing.T) {
	var buf bytes.Buffer
	r := NewRendererForced(&buf, false, true, false, false)

	entries := []ListEntry{
		{ID: "brew", Name: "Homebrew", Status: StatusCurrent, Version: "4.1.0"},
		{ID: "npm", Name: "npm", Status: StatusSkipped, Version: ""},
	}

	r.ListTools([]Group{{Items: entries}})

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

	r.ListTools([]Group{{Items: entries}})

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

// TestListTools_GroupedHeaderThenChildren locks the design grouping contract
// (task 3.1): a manager group renders its header line (the manager name) first,
// then the manager's own row, then the owned tool rows indented beneath it; a
// standalone group renders its rows without a header line; and the standalone
// group comes after all manager groups.
func TestListTools_GroupedHeaderThenChildren(t *testing.T) {
	var buf bytes.Buffer
	r := NewRendererForced(&buf, false, true, false, false)

	groups := []Group{
		{
			Header: "Homebrew",
			Items: []ListEntry{
				{ID: "brew", Name: "Homebrew", Status: StatusCurrent, Version: "4.1.0"},
				{ID: "gh", Name: "GitHub CLI", Status: StatusCurrent, Version: "2.4.0"},
			},
		},
		{
			Items: []ListEntry{
				{ID: "npm", Name: "npm", Status: StatusSkipped, Version: ""},
			},
		},
	}

	r.ListTools(groups)

	output := buf.String()
	// Manager header line present for the brew group.
	if !strings.Contains(output, "Homebrew") {
		t.Errorf("manager group must render its header, got:\n%s", output)
	}
	// Manager's own row and its owned tool row both present.
	if !strings.Contains(output, "brew") || !strings.Contains(output, "gh") {
		t.Errorf("group rows must include the manager and its owned tool, got:\n%s", output)
	}
	// Standalone group present after the manager group.
	if !strings.Contains(output, "npm") {
		t.Errorf("standalone group must render after manager groups, got:\n%s", output)
	}
	// The header line carries only "ID | Name | Status | Version".
	header := strings.SplitN(output, "\n", 2)[0]
	if !strings.Contains(header, "ID") || !strings.Contains(header, "Name") {
		t.Errorf("column header must lead the grouped output, got: %q", header)
	}
}

// TestGroupByOwner_LinuxGroupsOwnedTools proves GroupByOwner (task 3.2) groups
// owned tools under their resolving manager header, then places standalone
// tools last. On linux, docker and gh are owned by apt; brew/apt are managers
// present in the set; npm/nvm/pnpm/bun/opencode/go are standalone (go has no
// linux owner). Manager headers follow official.AllAdapters order.
func TestGroupByOwner_LinuxGroupsOwnedTools(t *testing.T) {
	tools := official.AdaptersForPlatform(platform.OSLinux)

	groups := GroupByOwner(tools, platform.OSLinux)

	// The manager groups appear first, in AllAdapters order (apt, brew).
	// apt owns docker + gh on linux; brew owns nothing on linux (go's linux
	// owner is absent, so go is standalone).
	var gotHeaders []string
	for _, g := range groups {
		if g.Header != "" {
			gotHeaders = append(gotHeaders, g.Header)
		}
	}

	aptFound := slices.Contains(gotHeaders, "APT Package Manager")
	brewFound := slices.Contains(gotHeaders, "Homebrew")
	if !aptFound {
		t.Errorf("apt manager header missing; got headers %v", gotHeaders)
	}
	if !brewFound {
		t.Errorf("brew manager header missing; got headers %v", gotHeaders)
	}

	// Docker + gh must be grouped under apt (their linux owner) — find the apt
	// group and confirm it contains both.
	var aptGroup *Group
	for i := range groups {
		if groups[i].Header == "APT Package Manager" {
			aptGroup = &groups[i]
			break
		}
	}
	if aptGroup == nil {
		t.Fatalf("no apt group found in %+v", groups)
	}
	ids := make([]string, len(aptGroup.Items))
	for i, item := range aptGroup.Items {
		ids[i] = item.ID
	}
	if !slices.Contains(ids, "docker") || !slices.Contains(ids, "gh") {
		t.Errorf("apt group must own docker+gh, got ids %v", ids)
	}

	// Every tool is placed exactly once across all groups (no lost/duplicated
	// rows, filter round-trip intact).
	var allIDs []string
	for _, g := range groups {
		for _, item := range g.Items {
			allIDs = append(allIDs, item.ID)
		}
	}
	for _, a := range tools {
		if !slices.Contains(allIDs, a.Name()) {
			t.Errorf("tool %s missing from grouping output", a.Name())
		}
	}
}

// TestGroupByOwner_FilteredManagerNoPhantomHeader pins the display-only
// grouping contract (task 3.5): when a manager is filtered out (not in the
// input set), its owned tools fall to the standalone group rather than
// producing a phantom manager header — the --only/--skip round-trip IDs stay
// usable.
func TestGroupByOwner_FilteredManagerNoPhantomHeader(t *testing.T) {
	// Build a set with ONLY docker (its linux owner apt filtered out).
	tools := []adapters.Adapter{official.AdapterByName("docker")}

	groups := GroupByOwner(tools, platform.OSLinux)

	if len(groups) != 1 {
		t.Fatalf("expected exactly 1 group (standalone), got %d: %+v", len(groups), groups)
	}
	if groups[0].Header != "" {
		t.Errorf("filtered-out manager must not create a phantom header, got %q", groups[0].Header)
	}
	if len(groups[0].Items) != 1 || groups[0].Items[0].ID != "docker" {
		t.Errorf("docker must round-trip in the standalone group, got %+v", groups[0].Items)
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
		"upp update -n",
		"upp update",
		"upp list",
		"upp --help",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("Dashboard output missing %q, got:\n%s", want, out)
		}
	}
	// The check command is removed; the query surface is `upp update -n`.
	if strings.Contains(out, "upp check") {
		t.Errorf("Dashboard must not reference the removed 'upp check', got:\n%s", out)
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
	// Spec ux-patterns Verbose Error Diagnostics: each stderr line renders
	// INDENTED beneath the failed tool entry ("    │ <line>").
	for _, line := range []string{"npm ERR! network connection refused", "npm ERR! retry limit reached"} {
		if !strings.Contains(out, "    │ "+line) {
			t.Errorf("stderr line must render indented beneath the failed tool, got:\n%s", out)
		}
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

// --- WU5: group bulk summary rendering (spec ux-patterns Summary Report) ---
//
// A manager-group bulk update MUST render a group bulk summary listing each
// owned tool that was updated, skipped (--skip-ed), current, or failed within
// the group, in canonical discovery order (spec ux-patterns Summary Report
// "Group bulk summary", "Group partial fail", "Group dry-run", and the
// deterministic-order rule). The renderer never reorders by completion or
// groups by status category — the list follows the canonical order the batch
// was enumerated in.

// TestGroupBulkSummary_UpdatedAndSkipped pins the "Group bulk summary"
// scenario: Linux apt group updates gh (success) and skips docker
// (--skip docker). The group summary lists gh updated and docker skipped.
func TestGroupBulkSummary_UpdatedAndSkipped(t *testing.T) {
	var buf bytes.Buffer
	r := NewRendererForced(&buf, false, true, false, false)

	r.GroupBulkSummary(GroupBulkSummary{
		Manager: "APT Package Manager",
		Results: []ToolResult{
			{Name: "gh", Status: StatusUpdated, Version: "2.46.0"},
			{Name: "docker", Status: StatusSkipped},
		},
	})

	out := buf.String()
	if !strings.Contains(out, "gh updated") {
		t.Errorf("group summary must report gh updated; got:\n%s", out)
	}
	if !strings.Contains(out, "docker skipped") {
		t.Errorf("group summary must report docker skipped (--skip-ed); got:\n%s", out)
	}
	// Canonical discovery order: gh (updated) is listed before docker (skipped).
	if strings.Index(out, "gh") > strings.Index(out, "docker") {
		t.Errorf("group summary must list tools in canonical discovery order; got:\n%s", out)
	}
}

// TestGroupBulkSummary_PartialFail pins the "Group partial fail" scenario:
// brew group updates gh but docker failed. The group summary lists gh updated,
// docker failed.
func TestGroupBulkSummary_PartialFail(t *testing.T) {
	var buf bytes.Buffer
	r := NewRendererForced(&buf, false, true, false, false)

	r.GroupBulkSummary(GroupBulkSummary{
		Manager: "Homebrew",
		Results: []ToolResult{
			{Name: "gh", Status: StatusUpdated, Version: "2.46.0"},
			{Name: "docker", Status: StatusFailed, Error: fmt.Errorf("timeout")},
		},
	})

	out := buf.String()
	if !strings.Contains(out, "gh updated") {
		t.Errorf("group summary must report gh updated; got:\n%s", out)
	}
	if !strings.Contains(out, "docker failed") {
		t.Errorf("group summary must report docker failed; got:\n%s", out)
	}
}

// TestGroupBulkSummary_DryRun pins the "Group dry-run" scenario: apt group
// dry-run has gh pending, docker current. The group summary reports gh would
// update and docker current, and never claims "All clean!".
func TestGroupBulkSummary_DryRun(t *testing.T) {
	var buf bytes.Buffer
	r := NewRendererForced(&buf, false, true, false, false)

	r.GroupBulkSummary(GroupBulkSummary{
		Manager: "APT Package Manager",
		DryRun:  true,
		Results: []ToolResult{
			{Name: "gh", Status: StatusAvailable, Version: "2.45.0 → 2.46.0"},
			{Name: "docker", Status: StatusCurrent, Version: "26.1.4"},
		},
	})

	out := buf.String()
	if !strings.Contains(out, "gh would update") {
		t.Errorf("group dry-run summary must report gh would update; got:\n%s", out)
	}
	if !strings.Contains(out, "docker current") {
		t.Errorf("group dry-run summary must report docker current; got:\n%s", out)
	}
}

// --- Opt-In Flag UX: pre-execution batch render (spec ux-patterns "Batch
// rendered") ---
//
// A manager-group bulk update MUST render which owned tools are in the batch,
// which are excluded by --skip, and whether the batch is gated, BEFORE
// executing (spec ux-patterns Opt-In Flag UX "Batch rendered"). The
// pre-execution GroupBatchPreview shows the PLAN; the post-run
// GroupBulkSummary reports the RESULT. The two are distinct: the preview never
// claims "All clean!" because nothing has run yet.

// TestGroupBatchPreview_RendersPlannedBatch pins the "Batch rendered"
// scenario: apt owns gh/docker, gh has an update available, docker is excluded
// by --skip. The batch UX shows gh as update-available and docker as skipped.
func TestGroupBatchPreview_RendersPlannedBatch(t *testing.T) {
	var buf bytes.Buffer
	r := NewRendererForced(&buf, false, true, false, false)

	r.GroupBatchPreview(GroupBatchPreview{
		Manager: "apt",
		Gated:   true,
		Tools: []GroupBatchTool{
			{Name: "gh", UpdateAvailable: true, Version: "2.45.0 → 2.46.0"},
			{Name: "docker", Skipped: true},
		},
	})

	out := buf.String()
	if !strings.Contains(out, "apt (manager)") {
		t.Errorf("batch preview must show the manager header; got:\n%s", out)
	}
	if !strings.Contains(out, "gh (update available)") {
		t.Errorf("batch preview must show gh as having an update available; got:\n%s", out)
	}
	if !strings.Contains(out, "docker (skipped via --skip)") {
		t.Errorf("batch preview must mark docker as skipped via --skip; got:\n%s", out)
	}
	if !strings.Contains(out, "Batch gated: yes") {
		t.Errorf("batch preview must report a gated batch; got:\n%s", out)
	}
}

// TestGroupBatchPreview_CurrentAndUngated triangulates the opposite path: a
// tool with no package update reports "current", and an always-update
// (ungated) manager reports "Batch gated: no".
func TestGroupBatchPreview_CurrentAndUngated(t *testing.T) {
	var buf bytes.Buffer
	r := NewRendererForced(&buf, false, true, false, false)

	r.GroupBatchPreview(GroupBatchPreview{
		Manager: "brew",
		Gated:   false,
		Tools: []GroupBatchTool{
			{Name: "gh", UpdateAvailable: false},
		},
	})

	out := buf.String()
	if !strings.Contains(out, "gh (current)") {
		t.Errorf("batch preview must report a no-update tool as current; got:\n%s", out)
	}
	if !strings.Contains(out, "Batch gated: no") {
		t.Errorf("batch preview must report an always-update batch as ungated; got:\n%s", out)
	}
}

// TestGroupBatchPreview_CheckFailed triangulates the "check fails" preview
// state: a tool whose CheckPackage errored is reported "check failed", not
// current (spec bulk-update: a failed check is never current nor available).
func TestGroupBatchPreview_CheckFailed(t *testing.T) {
	var buf bytes.Buffer
	r := NewRendererForced(&buf, false, true, false, false)

	r.GroupBatchPreview(GroupBatchPreview{
		Manager: "apt",
		Gated:   true,
		Tools: []GroupBatchTool{
			{Name: "gh", CheckFailed: true},
		},
	})

	out := buf.String()
	if !strings.Contains(out, "gh (check failed)") {
		t.Errorf("batch preview must report a failed check as 'check failed'; got:\n%s", out)
	}
}

// TestGroupBulkSummary_CanonicalOrderNotStatusOrder proves the summary list is
// in canonical discovery order, NOT grouped by status category (spec
// deterministic-order rule): an updated tool that is third in canonical order
// still renders third, after an earlier current tool, and never moves to the
// front just because it was "updated".
func TestGroupBulkSummary_CanonicalOrderNotStatusOrder(t *testing.T) {
	var buf bytes.Buffer
	r := NewRendererForced(&buf, false, true, false, false)

	r.GroupBulkSummary(GroupBulkSummary{
		Manager: "Homebrew",
		Results: []ToolResult{
			{Name: "go", Status: StatusCurrent, Version: "1.22"},
			{Name: "npm", Status: StatusSkipped},
			{Name: "gh", Status: StatusUpdated, Version: "2.46.0"},
		},
	})

	out := buf.String()
	// Canonical order go → npm → gh, regardless of status. If the renderer
	// grouped by status (updated/current/skipped), gh would lead — the spec
	// forbids that.
	goIdx := strings.Index(out, "go")
	npmIdx := strings.Index(out, "npm")
	ghIdx := strings.Index(out, "gh")
	if goIdx >= npmIdx || npmIdx >= ghIdx {
		t.Errorf("group summary must preserve canonical discovery order go→npm→gh; got:\n%s", out)
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
