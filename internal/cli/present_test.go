package cli

import (
	"context"
	"errors"
	"testing"

	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/adapters/official"
	"github.com/JhnFrankz/upp/internal/doctor"
	"github.com/JhnFrankz/upp/internal/engine"
	"github.com/JhnFrankz/upp/internal/output"
	"github.com/JhnFrankz/upp/internal/uninstall"
)

func TestPresentGroups(t *testing.T) {
	brew := official.AdapterByName("brew")
	gh := official.AdapterByName("gh")
	npm := official.AdapterByName("npm")
	bun := official.AdapterByName("bun")

	toolGroups := []engine.ToolGroup{
		{
			Header:   "Homebrew",
			Manager:  brew,
			Adapters: []adapters.Adapter{brew, gh},
		},
		{
			Header:   "",
			Manager:  nil,
			Adapters: []adapters.Adapter{npm, bun},
		},
	}

	outcomes := []engine.CheckOutcome{
		{
			ToolID:         "brew",
			ToolName:       "brew",
			Status:         engine.StatusCurrent,
			CurrentVersion: "4.2.0",
		},
		{
			ToolID:          "gh",
			ToolName:        "gh",
			Status:          engine.StatusAvailable,
			CurrentVersion:  "2.40.0",
			LatestVersion:   "2.45.0",
			UpdateAvailable: true,
		},
		{
			ToolID:   "npm",
			ToolName: "npm",
			Status:   engine.StatusFailed,
			Err:      errors.New("connection refused"),
		},
		// bun is intentionally missing from outcomes to test fallback to StatusSkipped
	}

	groups := presentGroups(toolGroups, outcomes)

	if len(groups) != 2 {
		t.Fatalf("presentGroups returned %d groups, want 2", len(groups))
	}

	// First group: Homebrew
	if groups[0].Header != "Homebrew" {
		t.Errorf("group[0].Header = %q, want %q", groups[0].Header, "Homebrew")
	}
	if len(groups[0].Items) != 2 {
		t.Fatalf("group[0].Items len = %d, want 2", len(groups[0].Items))
	}
	// brew
	if item := groups[0].Items[0]; item.ID != "brew" || item.Status != output.StatusCurrent || item.Version != "4.2.0" {
		t.Errorf("brew item = %+v, want ID=brew, Status=StatusCurrent, Version=4.2.0", item)
	}
	// gh
	if item := groups[0].Items[1]; item.ID != "gh" || item.Status != output.StatusAvailable || item.Version != "2.40.0" {
		t.Errorf("gh item = %+v, want ID=gh, Status=StatusAvailable, Version=2.40.0", item)
	}

	// Second group: Standalone
	if groups[1].Header != "" {
		t.Errorf("group[1].Header = %q, want empty", groups[1].Header)
	}
	if len(groups[1].Items) != 2 {
		t.Fatalf("group[1].Items len = %d, want 2", len(groups[1].Items))
	}
	// npm
	if item := groups[1].Items[0]; item.ID != "npm" || item.Status != output.StatusFailed || item.Version != "" {
		t.Errorf("npm item = %+v, want ID=npm, Status=StatusFailed, Version=\"\"", item)
	}
	// bun (missing outcome -> Skipped)
	if item := groups[1].Items[1]; item.ID != "bun" || item.Status != output.StatusSkipped || item.Version != "" {
		t.Errorf("bun item = %+v, want ID=bun, Status=StatusSkipped, Version=\"\"", item)
	}
}

func TestPresentGroups_StatusSkippedOutcome(t *testing.T) {
	npm := official.AdapterByName("npm")
	toolGroups := []engine.ToolGroup{
		{
			Header:   "",
			Manager:  nil,
			Adapters: []adapters.Adapter{npm},
		},
	}
	outcomes := []engine.CheckOutcome{
		{
			ToolID:   "npm",
			ToolName: "npm",
			Status:   engine.StatusSkipped,
		},
	}

	groups := presentGroups(toolGroups, outcomes)
	if len(groups) != 1 || len(groups[0].Items) != 1 {
		t.Fatalf("expected 1 group with 1 item, got %+v", groups)
	}
	if item := groups[0].Items[0]; item.Status != output.StatusSkipped {
		t.Errorf("item status = %v, want StatusSkipped", item.Status)
	}
}

func TestPresentGroups_StatusUnknownOutcome(t *testing.T) {
	npm := official.AdapterByName("npm")
	toolGroups := []engine.ToolGroup{
		{
			Header:   "",
			Manager:  nil,
			Adapters: []adapters.Adapter{npm},
		},
	}
	outcomes := []engine.CheckOutcome{
		{
			ToolID:   "npm",
			ToolName: "npm",
			Status:   engine.StatusUnknown,
		},
	}

	groups := presentGroups(toolGroups, outcomes)
	if len(groups) != 1 || len(groups[0].Items) != 1 {
		t.Fatalf("expected 1 group with 1 item, got %+v", groups)
	}
	if item := groups[0].Items[0]; item.Status != output.StatusSkipped {
		t.Errorf("item status = %v, want StatusSkipped (never StatusAvailable)", item.Status)
	}
}

type stubPresentAdapter struct {
	name string
	info adapters.ToolInfo
}

func (s *stubPresentAdapter) Name() string { return s.name }
func (s *stubPresentAdapter) Detect() bool { return true }
func (s *stubPresentAdapter) Check(ctx context.Context) (adapters.UpdateInfo, error) {
	return adapters.UpdateInfo{}, nil
}
func (s *stubPresentAdapter) Update(ctx context.Context, dryRun bool) (adapters.Result, error) {
	return adapters.Result{}, nil
}
func (s *stubPresentAdapter) Info() adapters.ToolInfo { return s.info }

func TestPresentGroups_SharedDisplayNameDoesNotCollide(t *testing.T) {
	toolA := &stubPresentAdapter{
		name: "tool-a",
		info: adapters.ToolInfo{
			ID:   "tool-a-id",
			Name: "Shared Display Name",
		},
	}
	toolB := &stubPresentAdapter{
		name: "tool-b",
		info: adapters.ToolInfo{
			ID:   "tool-b-id",
			Name: "Shared Display Name",
		},
	}
	toolGroups := []engine.ToolGroup{
		{
			Header:   "Group",
			Adapters: []adapters.Adapter{toolA, toolB},
		},
	}
	outcomes := []engine.CheckOutcome{
		{
			ToolID:          "tool-a-id",
			ToolName:        "Shared Display Name",
			Status:          engine.StatusAvailable,
			CurrentVersion:  "1.0.0",
			LatestVersion:   "2.0.0",
			UpdateAvailable: true,
		},
		{
			ToolID:         "tool-b-id",
			ToolName:       "Shared Display Name",
			Status:         engine.StatusCurrent,
			CurrentVersion: "1.0.0",
		},
	}

	groups := presentGroups(toolGroups, outcomes)
	if len(groups) != 1 || len(groups[0].Items) != 2 {
		t.Fatalf("expected 1 group with 2 items, got %+v", groups)
	}
	if groups[0].Items[0].Status != output.StatusAvailable {
		t.Errorf("item 0 status = %v, want StatusAvailable", groups[0].Items[0].Status)
	}
	if groups[0].Items[1].Status != output.StatusCurrent {
		t.Errorf("item 1 status = %v, want StatusCurrent", groups[0].Items[1].Status)
	}
}

type fakeDiagnosticPresenter struct {
	results []output.DiagnosticResult
	quiet   bool
	verbose bool
	called  bool
}

func (f *fakeDiagnosticPresenter) DoctorResults(results []output.DiagnosticResult, quiet, verbose bool) {
	f.results = results
	f.quiet = quiet
	f.verbose = verbose
	f.called = true
}

type fakeUninstallPresenter struct {
	targets []output.TargetItem
	called  bool
}

func (f *fakeUninstallPresenter) UninstallDryRunHeader()                        {}
func (f *fakeUninstallPresenter) UninstallDryRunTarget(targetType, path string) {}
func (f *fakeUninstallPresenter) UninstallRemoved(targetType, path string)      {}
func (f *fakeUninstallPresenter) UninstallPlan(targets []output.TargetItem) {
	f.targets = targets
	f.called = true
}
func (f *fakeUninstallPresenter) UninstallCanceled() {}
func (f *fakeUninstallPresenter) UninstallDone()     {}

func TestPresentDoctorResults_Mapping(t *testing.T) {
	fake := &fakeDiagnosticPresenter{}
	results := []doctor.CheckResult{
		{
			Category: "System",
			Name:     "OS",
			Status:   doctor.SeverityOK,
			Message:  "Linux supported",
		},
		{
			Category: "Config",
			Name:     "Lock",
			Status:   doctor.SeverityWarn,
			Message:  "Stale lock",
			FixHint:  "rm lock",
		},
		{
			Category: "Tools",
			Name:     "Brew",
			Status:   doctor.SeverityError,
			Message:  "Corrupt formula",
			Detail:   "syntax error at line 5",
		},
		{
			Category: "Network",
			Name:     "DNS",
			Status:   doctor.Severity(99), // unmapped -> SeverityWarn
			Message:  "Unknown severity",
		},
	}

	presentDoctorResults(fake, results, true, false)

	if !fake.called {
		t.Fatal("DoctorResults was not called on DiagnosticPresenter")
	}
	if !fake.quiet || fake.verbose {
		t.Errorf("quiet/verbose flags not passed correctly: quiet=%v, verbose=%v", fake.quiet, fake.verbose)
	}
	if len(fake.results) != 4 {
		t.Fatalf("expected 4 results, got %d", len(fake.results))
	}

	if fake.results[0].Status != output.DiagnosticSeverityOK {
		t.Errorf("expected OK severity for item 0, got %v", fake.results[0].Status)
	}
	if fake.results[1].Status != output.DiagnosticSeverityWarn || fake.results[1].FixHint != "rm lock" {
		t.Errorf("expected Warn severity and FixHint for item 1, got %+v", fake.results[1])
	}
	if fake.results[2].Status != output.DiagnosticSeverityError || fake.results[2].Detail != "syntax error at line 5" {
		t.Errorf("expected Error severity and Detail for item 2, got %+v", fake.results[2])
	}
	if fake.results[3].Status != output.DiagnosticSeverityWarn {
		t.Errorf("expected fallback Warn severity for item 3, got %v", fake.results[3].Status)
	}
}

func TestPresentUninstallPlan_Mapping(t *testing.T) {
	fake := &fakeUninstallPresenter{}
	targets := []uninstall.Target{
		{Type: uninstall.TargetBinary, Path: "/usr/local/bin/upp", Exists: true},
		{Type: uninstall.TargetConfig, Path: "/home/user/.config/upp", Exists: false},
	}

	presentUninstallPlan(fake, targets)

	if !fake.called {
		t.Fatal("UninstallPlan was not called on UninstallPresenter")
	}
	if len(fake.targets) != 2 {
		t.Fatalf("expected 2 targets, got %d", len(fake.targets))
	}
	if fake.targets[0].Type != string(uninstall.TargetBinary) || !fake.targets[0].Exists {
		t.Errorf("target 0 unexpected: %+v", fake.targets[0])
	}
	if fake.targets[1].Type != string(uninstall.TargetConfig) || fake.targets[1].Exists {
		t.Errorf("target 1 unexpected: %+v", fake.targets[1])
	}
}
