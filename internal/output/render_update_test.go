package output

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
	"testing"
)

func renderToolLine(result ToolResult, color, emoji, quiet, verbose bool) string {
	var buf bytes.Buffer
	r := NewRendererForced(&buf, color, emoji, quiet, verbose)
	r.ToolLine(result)
	return buf.String()
}

func renderUpdateSummary(summary Summary, color, emoji, quiet, verbose bool) string {
	var buf bytes.Buffer
	r := NewRendererForced(&buf, color, emoji, quiet, verbose)
	r.UpdateSummary(summary)
	return buf.String()
}

func checkSummary(t *testing.T, summary Summary, contains []string, notContains ...string) {
	t.Helper()
	output := renderUpdateSummary(summary, false, true, false, false)
	for _, c := range contains {
		if !strings.Contains(output, c) {
			t.Errorf("expected summary to contain %q, got:\n%s", c, output)
		}
	}
	for _, nc := range notContains {
		if strings.Contains(output, nc) {
			t.Errorf("expected summary NOT to contain %q, got:\n%s", nc, output)
		}
	}
}

func TestToolLine_Verbose(t *testing.T) {
	out := renderToolLine(ToolResult{Name: "brew", Status: StatusUpdated, Version: "4.1.0"}, false, true, false, false)
	if !strings.Contains(out, "brew") || !strings.Contains(out, "4.1.0") || !strings.Contains(out, "✅") {
		t.Errorf("verbose tool line missing expected output, got:\n%s", out)
	}
}

func TestToolLine_Quiet(t *testing.T) {
	out := renderToolLine(ToolResult{Name: "brew", Status: StatusUpdated}, false, true, true, false)
	if !strings.Contains(out, "brew") || !strings.Contains(out, "✅") {
		t.Errorf("quiet tool line missing expected output, got:\n%s", out)
	}
	if strings.HasPrefix(out, "  ") {
		t.Error("quiet mode should not indent")
	}

	// Quiet mode should not contain verbose text like "(not installed)"
	outSkipped := renderToolLine(ToolResult{Name: "brew", Status: StatusSkipped}, false, true, true, false)
	if !strings.Contains(outSkipped, "brew") {
		t.Error("output should contain tool name")
	}
	if strings.Contains(outSkipped, "(not installed)") {
		t.Error("quiet mode should not contain verbose text")
	}
}

func TestToolLine_FailedWithError(t *testing.T) {
	out := renderToolLine(ToolResult{Name: "cargo", Status: StatusFailed, Error: fmt.Errorf("connection timeout")}, false, true, false, false)
	if !strings.Contains(out, "cargo") || !strings.Contains(out, "connection timeout") {
		t.Errorf("failed tool line missing expected output, got:\n%s", out)
	}
}

func TestToolLine_Denied(t *testing.T) {
	out := renderToolLine(ToolResult{Name: "docker", Status: StatusDenied}, false, true, false, false)
	if !strings.Contains(out, "docker") || !strings.Contains(out, "(denied)") || !strings.Contains(out, "🚫") {
		t.Errorf("denied tool line missing expected output, got:\n%s", out)
	}
}

func TestToolLine_QuietVersion(t *testing.T) {
	out := renderToolLine(ToolResult{Name: "brew", Status: StatusUpdated, Version: "4.1.0 → 4.2.0"}, false, true, true, false)
	if !strings.Contains(out, "brew") || !strings.Contains(out, "4.1.0 → 4.2.0") {
		t.Errorf("quiet mode must include version delta if present; got: %q", out)
	}
}

func TestToolLine_VerboseFailureDiagnostics(t *testing.T) {
	out := renderToolLine(ToolResult{
		Name:   "npm",
		Status: StatusFailed,
		Error:  fmt.Errorf("exit 1"),
		Stderr: "npm ERR! network connection refused\nnpm ERR! retry limit reached",
	}, false, false, false, true)

	if !strings.Contains(out, "npm ERR! network connection refused") || !strings.Contains(out, "npm ERR! retry limit reached") {
		t.Errorf("expected verbose tool line to contain stderr, got:\n%s", out)
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
	out := renderToolLine(ToolResult{
		Name:   "npm",
		Status: StatusFailed,
		Error:  fmt.Errorf("exit 1"),
		Stderr: "npm ERR! network connection refused",
	}, false, false, false, false)
	if strings.Contains(out, "npm ERR! network connection refused") {
		t.Errorf("expected non-verbose tool line to suppress stderr, got:\n%s", out)
	}
}

func TestToolLine_QuietOverridesVerbose(t *testing.T) {
	out := renderToolLine(ToolResult{
		Name:   "npm",
		Status: StatusFailed,
		Error:  fmt.Errorf("exit 1"),
		Stderr: "npm ERR! network connection refused",
	}, false, false, true, true)
	if strings.Contains(out, "npm ERR! network connection refused") {
		t.Errorf("expected quiet mode to override verbose and suppress stderr, got:\n%s", out)
	}
}

func TestDryRunHeader(t *testing.T) {
	var buf bytes.Buffer
	r := NewRendererForced(&buf, false, true, false, false)
	r.DryRunHeader()
	if !strings.Contains(buf.String(), "Dry run") {
		t.Errorf("dry run header should contain 'Dry run', got:\n%s", buf.String())
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
	out := buf.String()
	if !strings.Contains(out, "Checking 2/5") || !strings.Contains(out, "brew") || strings.Contains(out, "Updating") {
		t.Errorf("check progress verb unexpected, got:\n%s", out)
	}
}

func TestProgress_UpdateVerb(t *testing.T) {
	var buf bytes.Buffer
	r := NewRendererForced(&buf, false, true, false, false)
	r.Progress("Updating", 3, 10, "npm")
	out := buf.String()
	if !strings.Contains(out, "Updating 3/10") || !strings.Contains(out, "npm") || strings.Contains(out, "Checking") {
		t.Errorf("update progress verb unexpected, got:\n%s", out)
	}
}

func TestRenderer_ConcurrentProgress_ThreadSafe(t *testing.T) {
	var buf bytes.Buffer
	r := NewRendererForced(&buf, true, true, false, false)

	const goroutines = 20
	const iterations = 50
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for g := 0; g < goroutines; g++ {
		go func(id int) {
			defer wg.Done()
			for i := 1; i <= iterations; i++ {
				r.Progress("Checking", i, iterations, fmt.Sprintf("tool-%d", id))
			}
		}(g)
	}

	wg.Wait()
	if buf.Len() == 0 {
		t.Errorf("expected non-empty output from concurrent progress writes")
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

func TestUpdateSummary_AllUpdated(t *testing.T) {
	// Spec ux-patterns Summary Report, "All succeed": the clean line counts
	// updated AND failed explicitly ("5 updated, 0 failed. All clean!").
	checkSummary(t, Summary{
		Results: []ToolResult{
			{Name: "brew", Status: StatusUpdated, Version: "4.1.0"},
			{Name: "npm", Status: StatusUpdated, Version: "10.0.0"},
		},
	}, []string{"2 updated, 0 failed. All clean!"})
}

func TestUpdateSummary_PartialFailure(t *testing.T) {
	// Spec ux-patterns Summary Report, "Partial fail": counts compose as
	// "N updated, M failed" followed by the review tagline.
	checkSummary(t, Summary{
		Results: []ToolResult{
			{Name: "brew", Status: StatusUpdated, Version: "4.1.0"},
			{Name: "npm", Status: StatusFailed, Error: fmt.Errorf("timeout")},
		},
	}, []string{"1 updated, 1 failed. Review errors above."})
}

func TestUpdateSummary_AllSkipped(t *testing.T) {
	checkSummary(t, Summary{
		Results: []ToolResult{
			{Name: "brew", Status: StatusSkipped},
			{Name: "npm", Status: StatusSkipped},
		},
	}, []string{"All tools not installed"})
}

func TestUpdateSummary_DryRun(t *testing.T) {
	checkSummary(t, Summary{
		Results: []ToolResult{
			{Name: "brew", Status: StatusAvailable, Version: "4.0.0 → 4.1.0"},
		},
		DryRun: true,
	}, []string{"would update"}, "All clean!")
}

func TestUpdateSummary_NotCleanWithSkips(t *testing.T) {
	checkSummary(t, Summary{
		Results: []ToolResult{
			{Name: "brew", Status: StatusUpdated, Version: "4.1.0"},
			{Name: "npm", Status: StatusSkipped},
		},
	}, []string{"1 updated"}, "All clean!")
}

// TestUpdateSummary_AllCurrentDryRun proves current tools are counted on the
// update path (design D6). StatusCurrent used to be counted nowhere, so an
// all-current dry-run fell into the "All tools not installed" branch.
func TestUpdateSummary_AllCurrentDryRun(t *testing.T) {
	checkSummary(t, Summary{
		Results: []ToolResult{
			{Name: "brew", Status: StatusCurrent, Version: "4.1.0"},
			{Name: "npm", Status: StatusCurrent, Version: "10.0.0"},
		},
		DryRun: true,
	}, []string{"2 up to date"}, "All tools not installed", "All clean!")
}

// TestUpdateSummary_CurrentWithSkipsDryRun pins the spec ux-patterns Summary
// Report scenario "Up-to-date with skips": 8 current + 2 skipped tools on a
// dry-run count both explicitly ("8 up to date, 2 skipped"), the detail list
// names the current tools, and no "All tools up to date." tagline appears.
func TestUpdateSummary_CurrentWithSkipsDryRun(t *testing.T) {
	checkSummary(t, Summary{
		Results: []ToolResult{
			{Name: "brew", Status: StatusCurrent, Version: "4.1.0"},
			{Name: "cargo", Status: StatusSkipped},
		},
		DryRun: true,
	}, []string{"1 up to date", "1 skipped"}, "All tools not installed")
}

// TestUpdateSummary_UpdatedAndCurrent proves the up-to-date part composes
// with real updates in non-dry-run mode, in canonical part order.
func TestUpdateSummary_UpdatedAndCurrent(t *testing.T) {
	checkSummary(t, Summary{
		Results: []ToolResult{
			{Name: "brew", Status: StatusUpdated, Version: "4.2.0"},
			{Name: "npm", Status: StatusUpdated, Version: "10.1.0"},
			{Name: "go", Status: StatusCurrent, Version: "1.22"},
		},
	}, []string{"2 updated, 1 up to date", "Up to date: go"})
}

// TestUpdateSummary_Deselected pins the Deselected Pending Tools Reporting
// requirement (spec command-interface): a pending-but-deselected tool gets a
// distinct count and detail line, is never counted as skipped or updated, and
// keeps the run from claiming "All clean!".
func TestUpdateSummary_Deselected(t *testing.T) {
	checkSummary(t, Summary{
		Results: []ToolResult{
			{Name: "brew", Status: StatusUpdated, Version: "4.2.0"},
			{Name: "bun", Status: StatusDeselected},
		},
	}, []string{"1 updated", "1 deselected", "Deselected: bun"}, "skipped", "All clean!")
}

// TestUpdateSummary_OnlyDeselected proves the "All pending deselected" scenario:
// when every result is deselected, the summary reports the deselected count and
// must NOT fall into the "All tools not installed" branch.
func TestUpdateSummary_OnlyDeselected(t *testing.T) {
	checkSummary(t, Summary{
		Results: []ToolResult{
			{Name: "brew", Status: StatusDeselected},
			{Name: "bun", Status: StatusDeselected},
		},
	}, []string{"2 deselected", "Deselected: brew, bun"}, "All tools not installed")
}

// TestUpdateSummary_DeselectedDistinctFromSkipped proves the "Distinct from
// skipped" scenario: a not-installed (skipped) tool and a deselected pending
// tool are counted in separate buckets and listed separately.
func TestUpdateSummary_DeselectedDistinctFromSkipped(t *testing.T) {
	checkSummary(t, Summary{
		Results: []ToolResult{
			{Name: "missing", Status: StatusSkipped},
			{Name: "bun", Status: StatusDeselected},
		},
	}, []string{"1 skipped, 1 deselected", "Skipped: missing", "Deselected: bun"}, "2 skipped")
}

// TestUpdateSummary_Denied pins confirmation denial reporting: a denied tool
// gets a distinct count and detail line, is never counted as skipped or updated,
// and keeps the run from claiming "All clean!".
func TestUpdateSummary_Denied(t *testing.T) {
	checkSummary(t, Summary{
		Results: []ToolResult{
			{Name: "brew", Status: StatusUpdated, Version: "4.2.0"},
			{Name: "docker", Status: StatusDenied},
		},
	}, []string{"1 updated", "1 denied", "Denied: docker"}, "skipped", "All clean!")
}

// TestUpdateSummary_OnlyDenied proves that when all tools are denied confirmation,
// the summary reports the denied count and must NOT fall into "All tools not installed".
func TestUpdateSummary_OnlyDenied(t *testing.T) {
	checkSummary(t, Summary{
		Results: []ToolResult{
			{Name: "brew", Status: StatusDenied},
			{Name: "docker", Status: StatusDenied},
		},
	}, []string{"2 denied", "Denied: brew, docker"}, "All tools not installed")
}

// TestUpdateSummary_DeniedDistinctFromSkipped proves that not-installed (skipped)
// and denied tools are counted in separate buckets and listed separately.
func TestUpdateSummary_DeniedDistinctFromSkipped(t *testing.T) {
	checkSummary(t, Summary{
		Results: []ToolResult{
			{Name: "missing", Status: StatusSkipped},
			{Name: "docker", Status: StatusDenied},
		},
	}, []string{"1 skipped, 1 denied", "Skipped: missing", "Denied: docker"}, "2 skipped")
}

func TestUpdateSummary_VerboseFailureDiagnostics(t *testing.T) {
	out := renderUpdateSummary(Summary{
		Results: []ToolResult{
			{Name: "brew", Status: StatusUpdated, Version: "4.1.0"},
			{
				Name:   "npm",
				Status: StatusFailed,
				Error:  fmt.Errorf("timeout"),
				Stderr: "npm ERR! lock frontend held by another process",
			},
		},
	}, false, false, false, true)

	if !strings.Contains(out, "lock frontend held by another process") {
		t.Errorf("expected verbose update summary to contain stderr, got:\n%s", out)
	}
}

func TestUpdateSummary_VerboseMultilineFailureDiagnostics(t *testing.T) {
	out := renderUpdateSummary(Summary{
		Results: []ToolResult{
			{
				Name:   "cargo",
				Status: StatusFailed,
				Error:  fmt.Errorf("build failed"),
				Stderr: "error[E0432]: unresolved import\n   --> src/main.rs:1:5\n    |\n1   | use foo::bar;\n    |     ^^^",
			},
		},
	}, false, false, false, true)

	expectedLines := []string{
		"error[E0432]: unresolved import",
		"--> src/main.rs:1:5",
		"use foo::bar;",
		"^^^",
	}
	for _, expected := range expectedLines {
		if !strings.Contains(out, expected) {
			t.Errorf("expected verbose update summary to contain line %q, got:\n%s", expected, out)
		}
	}
}

// TestUpdateSummary_NoMisleadingAllCleanWithPending proves spec ux-patterns:
// UpdateSummary with pending updates in dry-run mode prints "N would update"
// and never claims "All clean!" or "All tools up to date.".
func TestUpdateSummary_NoMisleadingAllCleanWithPending(t *testing.T) {
	out := renderUpdateSummary(Summary{
		Results: []ToolResult{
			{Name: "gh", Status: StatusAvailable, Version: "2.45.0 → 2.46.0"},
			{Name: "docker", Status: StatusAvailable, Version: "26.1.4 → 27.0.0"},
			{Name: "npm", Status: StatusCurrent, Version: "10.0.0"},
		},
		DryRun: true,
	}, false, true, false, false)

	if !strings.Contains(out, "2 would update, 1 up to date") {
		t.Errorf("expected dry-run summary with would-update count, got:\n%s", out)
	}
	if strings.Contains(out, "All clean!") {
		t.Errorf("dry-run with pending updates must never claim 'All clean!', got:\n%s", out)
	}
	if strings.Contains(out, "All tools up to date.") {
		t.Errorf("dry-run with pending updates must not claim all tools up to date, got:\n%s", out)
	}
}

func TestRenderer_UpdatePresenterInterfaces(t *testing.T) {
	var buf bytes.Buffer
	r := NewRenderer(&buf, false)

	var pp ProgressPresenter = r
	pp.Progress("Updating", 1, 2, "brew")

	var up UpdatePresenter = r
	up.DryRunPlanned("cargo")
	up.UpdateCancelled()

	out := buf.String()
	if !strings.Contains(out, "cargo") || !strings.Contains(out, "Update canceled") {
		t.Errorf("UpdatePresenter interface methods missing expected output, got: %s", out)
	}
}
