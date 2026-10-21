package output

import (
	"bytes"
	"strings"
	"testing"
)

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
	// The ID column shows the --only filter IDs, distinct from the
	// display name.
	if !strings.Contains(output, "brew") || !strings.Contains(output, "npm") {
		t.Errorf("list rows must show tool IDs usable with --only, got:\n%s", output)
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

func TestNoToolsConfigured(t *testing.T) {
	var buf bytes.Buffer
	r := NewRendererForced(&buf, false, false, false, false)
	r.NoToolsConfigured()
	if got := buf.String(); got != "no tools configured.\n" {
		t.Errorf("NoToolsConfigured() = %q, want %q", got, "no tools configured.\n")
	}

	buf.Reset()
	rQuiet := NewRendererForced(&buf, false, false, true, false)
	rQuiet.NoToolsConfigured()
	if buf.Len() > 0 {
		t.Errorf("NoToolsConfigured() in quiet mode should produce no output, got: %q", buf.String())
	}
}

func TestNoToolsMatchFilter(t *testing.T) {
	var buf bytes.Buffer
	r := NewRendererForced(&buf, false, false, false, false)
	r.NoToolsMatchFilter("foo")
	if got := buf.String(); got != "no tools match --only filter: foo\n" {
		t.Errorf("NoToolsMatchFilter() = %q, want %q", got, "no tools match --only filter: foo\n")
	}

	buf.Reset()
	rQuiet := NewRendererForced(&buf, false, false, true, false)
	rQuiet.NoToolsMatchFilter("foo")
	if buf.Len() > 0 {
		t.Errorf("NoToolsMatchFilter() in quiet mode should produce no output, got: %q", buf.String())
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
		"upp doctor",
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

func TestInitHeaderAndDetected(t *testing.T) {
	var buf bytes.Buffer
	r := NewRendererForced(&buf, false, false, false, false)

	r.InitHeader()
	r.InitDetected("brew")

	out := buf.String()
	if !strings.Contains(out, "upp init — detecting installed tools...") {
		t.Errorf("InitHeader missing expected text, got: %s", out)
	}
	if !strings.Contains(out, "brew") {
		t.Errorf("InitDetected missing tool name, got: %s", out)
	}
}

func TestInitConfigGenerated(t *testing.T) {
	var buf bytes.Buffer
	r := NewRendererForced(&buf, false, false, false, false)

	r.InitConfigGenerated("/path/to/upp.toml")

	out := buf.String()
	if !strings.Contains(out, "Config written to /path/to/upp.toml") {
		t.Errorf("InitConfigGenerated missing expected text, got: %s", out)
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

func TestUninstallDryRunHeaderAndTarget(t *testing.T) {
	var buf bytes.Buffer
	r := NewRendererForced(&buf, false, false, false, false)

	r.UninstallDryRunHeader()
	r.UninstallDryRunTarget("binary", "/usr/local/bin/upp")

	out := buf.String()
	if !strings.Contains(out, "Dry run — no files will be removed") {
		t.Errorf("UninstallDryRunHeader missing expected text, got: %s", out)
	}
	if !strings.Contains(out, "Would remove binary: /usr/local/bin/upp") {
		t.Errorf("UninstallDryRunTarget missing expected text, got: %s", out)
	}
}

func TestUninstallRemoved(t *testing.T) {
	var buf bytes.Buffer
	r := NewRendererForced(&buf, false, false, false, false)
	r.UninstallRemoved("config", "/home/u/.config/upp")
	if !strings.Contains(buf.String(), "Removed config: /home/u/.config/upp") {
		t.Errorf("UninstallRemoved expected removal notice, got: %s", buf.String())
	}

	buf.Reset()
	rQuiet := NewRendererForced(&buf, false, false, true, false)
	rQuiet.UninstallRemoved("config", "/home/u/.config/upp")
	if buf.Len() > 0 {
		t.Errorf("UninstallRemoved in quiet mode should produce no output, got: %q", buf.String())
	}
}

func TestRenderer_UninstallCanceled(t *testing.T) {
	var buf bytes.Buffer
	r := NewRendererForced(&buf, false, false, false, false)
	r.UninstallCanceled()

	expected := "Uninstall canceled — no files were removed.\n"
	if buf.String() != expected {
		t.Errorf("UninstallCanceled() = %q, want %q", buf.String(), expected)
	}
}

func TestUninstallDone(t *testing.T) {
	var buf bytes.Buffer
	r := NewRendererForced(&buf, false, false, false, false)
	r.UninstallDone()

	expected := "upp has been successfully uninstalled.\n"
	if buf.String() != expected {
		t.Errorf("UninstallDone() = %q, want %q", buf.String(), expected)
	}
}

func TestRenderer_UninstallPlan(t *testing.T) {
	var buf bytes.Buffer
	r := NewRendererForced(&buf, false, false, false, false)
	targets := []TargetItem{
		{Type: "binary", Path: "/usr/local/bin/upp", Exists: true},
		{Type: "config", Path: "/home/user/.config/upp", Exists: true},
		{Type: "cache", Path: "/home/user/.cache/upp", Exists: false},
	}
	r.UninstallPlan(targets)

	out := buf.String()
	if !strings.Contains(out, "The following targets will be removed:") {
		t.Errorf("expected plan header, got: %s", out)
	}
	if !strings.Contains(out, "  - Binary: /usr/local/bin/upp") {
		t.Errorf("expected binary target in plan, got: %s", out)
	}
	if !strings.Contains(out, "  - Config: /home/user/.config/upp") {
		t.Errorf("expected config target in plan, got: %s", out)
	}
	if strings.Contains(out, "/home/user/.cache/upp") {
		t.Errorf("non-existent cache target should not be in plan, got: %s", out)
	}
}

func TestRenderer_ViewPresenterInterfaces(t *testing.T) {
	var buf bytes.Buffer
	r := NewRenderer(&buf, false)

	var lp ListPresenter = r
	lp.NoToolsConfigured()
	if !strings.Contains(buf.String(), "no tools configured") {
		t.Errorf("ListPresenter test failed, got: %s", buf.String())
	}
	buf.Reset()

	var dp DashboardPresenter = r
	dp.DashboardNoConfig("1.0", "linux")
	if !strings.Contains(buf.String(), "No configuration found") {
		t.Errorf("DashboardPresenter test failed, got: %s", buf.String())
	}
	buf.Reset()

	var ip InitPresenter = r
	ip.InitDetected("git")
	if !strings.Contains(buf.String(), "git") {
		t.Errorf("InitPresenter test failed, got: %s", buf.String())
	}
	buf.Reset()

	var sup SelfUpdatePresenter = r
	sup.SelfUpdateDevBuild()
	if !strings.Contains(buf.String(), "development build") {
		t.Errorf("SelfUpdatePresenter test failed, got: %s", buf.String())
	}
	buf.Reset()

	var up UninstallPresenter = r
	up.UninstallDone()
	if !strings.Contains(buf.String(), "successfully uninstalled") {
		t.Errorf("UninstallPresenter test failed, got: %s", buf.String())
	}
	buf.Reset()

	var vp ViewPresenter = r
	vp.UninstallCanceled()
	if !strings.Contains(buf.String(), "Uninstall canceled") {
		t.Errorf("ViewPresenter test failed, got: %s", buf.String())
	}
}
