package engine

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/adapters/official"
	"github.com/JhnFrankz/upp/internal/config"
	"github.com/JhnFrankz/upp/internal/platform"
)

type mockPlanAdapter struct {
	info adapters.ToolInfo
}

func (m *mockPlanAdapter) Name() string { return m.info.Name }
func (m *mockPlanAdapter) Detect() bool { return true }
func (m *mockPlanAdapter) Check(ctx context.Context) (adapters.UpdateInfo, error) {
	return adapters.UpdateInfo{}, nil
}
func (m *mockPlanAdapter) Update(ctx context.Context, dryRun bool) (adapters.Result, error) {
	return adapters.Result{}, nil
}
func (m *mockPlanAdapter) Info() adapters.ToolInfo { return m.info }

func TestPlan_OutcomeSegregation(t *testing.T) {
	cfg := &config.Config{}
	fakes := []adapters.Adapter{
		&mockPlanAdapter{info: adapters.ToolInfo{
			ID:           "gated-avail",
			Name:         "gated-avail",
			UpdatePolicy: adapters.PolicyGated,
		}},
		&mockPlanAdapter{info: adapters.ToolInfo{
			ID:           "gated-curr",
			Name:         "gated-curr",
			UpdatePolicy: adapters.PolicyGated,
		}},
		&mockPlanAdapter{info: adapters.ToolInfo{
			ID:           "skipped-tool",
			Name:         "skipped-tool",
			UpdatePolicy: adapters.PolicyGated,
		}},
		&mockPlanAdapter{info: adapters.ToolInfo{
			ID:           "failed-tool",
			Name:         "failed-tool",
			UpdatePolicy: adapters.PolicyGated,
		}},
	}
	eng := New(cfg, platform.OSLinux, WithAdapters(fakes))

	outcomes := []CheckOutcome{
		{
			ToolID:          "failed-tool",
			ToolName:        "failed-tool",
			Status:          StatusFailed,
			Err:             errors.New("subprocess crashed"),
			Stderr:          "fatal error",
			UpdateAvailable: false,
		},
		{
			ToolID:          "skipped-tool",
			ToolName:        "skipped-tool",
			Status:          StatusSkipped,
			UpdateAvailable: false,
		},
		{
			ToolID:          "gated-avail",
			ToolName:        "gated-avail",
			Status:          StatusAvailable,
			CurrentVersion:  "1.0.0",
			LatestVersion:   "1.1.0",
			UpdateAvailable: true,
		},
		{
			ToolID:          "gated-curr",
			ToolName:        "gated-curr",
			Status:          StatusCurrent,
			CurrentVersion:  "2.0.0",
			LatestVersion:   "2.0.0",
			UpdateAvailable: false,
		},
	}

	plan, err := eng.Plan(outcomes, Filter{})
	if err != nil {
		t.Fatalf("eng.Plan error: %v", err)
	}

	// 1. StatusFailed placed in Failed, never in Updates
	if len(plan.Failed) != 1 || plan.Failed[0].ToolID != "failed-tool" {
		t.Errorf("plan.Failed = %v, want 1 item (failed-tool)", plan.Failed)
	}
	for _, u := range plan.Updates {
		if u.ToolID == "failed-tool" {
			t.Errorf("failed-tool unexpectedly placed in plan.Updates")
		}
	}

	// 2. StatusSkipped placed in Skipped, never in Updates
	if len(plan.Skipped) != 1 || plan.Skipped[0].ToolID != "skipped-tool" {
		t.Errorf("plan.Skipped = %v, want 1 item (skipped-tool)", plan.Skipped)
	}
	for _, u := range plan.Updates {
		if u.ToolID == "skipped-tool" {
			t.Errorf("skipped-tool unexpectedly placed in plan.Updates")
		}
	}

	// 3. StatusAvailable placed in Updates
	if len(plan.Updates) != 1 || plan.Updates[0].ToolID != "gated-avail" {
		t.Errorf("plan.Updates = %v, want 1 item (gated-avail)", plan.Updates)
	}

	// 4. StatusCurrent placed in Current, never in Updates
	if len(plan.Current) != 1 || plan.Current[0].ToolID != "gated-curr" {
		t.Errorf("plan.Current = %v, want 1 item (gated-curr)", plan.Current)
	}
	for _, u := range plan.Updates {
		if u.ToolID == "gated-curr" {
			t.Errorf("gated-curr unexpectedly placed in plan.Updates")
		}
	}

	// 5. Empty outcomes produces empty plan without error
	emptyPlan, err := eng.Plan(nil, Filter{})
	if err != nil {
		t.Fatalf("Plan(nil) error: %v", err)
	}
	if len(emptyPlan.Updates) != 0 || len(emptyPlan.Current) != 0 || len(emptyPlan.Skipped) != 0 || len(emptyPlan.Failed) != 0 {
		t.Errorf("emptyPlan non-empty: %+v", emptyPlan)
	}
}

func TestPlan_PolicyGated(t *testing.T) {
	cfg := &config.Config{}
	fakes := []adapters.Adapter{
		&mockPlanAdapter{info: adapters.ToolInfo{
			ID:           "gated-with-update",
			Name:         "Gated With Update",
			UpdatePolicy: adapters.PolicyGated,
		}},
		&mockPlanAdapter{info: adapters.ToolInfo{
			ID:           "gated-no-update",
			Name:         "Gated No Update",
			UpdatePolicy: adapters.PolicyGated,
		}},
	}
	eng := New(cfg, platform.OSLinux, WithAdapters(fakes))

	t.Run("gated tool with UpdateAvailable == true placed in Updates", func(t *testing.T) {
		outcomes := []CheckOutcome{
			{
				ToolID:          "gated-with-update",
				ToolName:        "Gated With Update",
				Status:          StatusAvailable,
				CurrentVersion:  "1.0.0",
				LatestVersion:   "1.1.0",
				UpdateAvailable: true,
			},
		}

		plan, err := eng.Plan(outcomes, Filter{})
		if err != nil {
			t.Fatalf("Plan error: %v", err)
		}
		if len(plan.Updates) != 1 {
			t.Fatalf("len(plan.Updates) = %d, want 1", len(plan.Updates))
		}
		if plan.Updates[0].ToolID != "gated-with-update" {
			t.Errorf("plan.Updates[0].ToolID = %q, want gated-with-update", plan.Updates[0].ToolID)
		}
		if len(plan.Current) != 0 {
			t.Errorf("plan.Current not empty: %v", plan.Current)
		}
	})

	t.Run("gated tool with UpdateAvailable == false placed in Current", func(t *testing.T) {
		outcomes := []CheckOutcome{
			{
				ToolID:          "gated-no-update",
				ToolName:        "Gated No Update",
				Status:          StatusCurrent,
				CurrentVersion:  "1.0.0",
				LatestVersion:   "1.0.0",
				UpdateAvailable: false,
			},
		}

		plan, err := eng.Plan(outcomes, Filter{})
		if err != nil {
			t.Fatalf("Plan error: %v", err)
		}
		if len(plan.Updates) != 0 {
			t.Fatalf("len(plan.Updates) = %d, want 0", len(plan.Updates))
		}
		if len(plan.Current) != 1 || plan.Current[0].ToolID != "gated-no-update" {
			t.Errorf("plan.Current = %v, want 1 item (gated-no-update)", plan.Current)
		}
	})
}

func TestPlan_PolicyAlwaysUpdate(t *testing.T) {
	cfg := &config.Config{}
	fakes := []adapters.Adapter{
		&mockPlanAdapter{info: adapters.ToolInfo{
			ID:           "always-tool",
			Name:         "Always Tool",
			UpdatePolicy: adapters.PolicyAlwaysUpdate,
		}},
	}
	eng := New(cfg, platform.OSLinux, WithAdapters(fakes))

	t.Run("always-update tool with UpdateAvailable == false placed unconditionally in Updates", func(t *testing.T) {
		outcomes := []CheckOutcome{
			{
				ToolID:          "always-tool",
				ToolName:        "Always Tool",
				Status:          StatusCurrent,
				CurrentVersion:  "3.0.0",
				LatestVersion:   "3.0.0",
				UpdateAvailable: false,
			},
		}

		plan, err := eng.Plan(outcomes, Filter{})
		if err != nil {
			t.Fatalf("Plan error: %v", err)
		}
		if len(plan.Updates) != 1 {
			t.Fatalf("len(plan.Updates) = %d, want 1", len(plan.Updates))
		}
		if plan.Updates[0].ToolID != "always-tool" {
			t.Errorf("plan.Updates[0].ToolID = %q, want always-tool", plan.Updates[0].ToolID)
		}
		if len(plan.Current) != 0 {
			t.Errorf("plan.Current should be empty, got: %v", plan.Current)
		}
	})

	t.Run("always-update tool with UpdateAvailable == true placed in Updates", func(t *testing.T) {
		outcomes := []CheckOutcome{
			{
				ToolID:          "always-tool",
				ToolName:        "Always Tool",
				Status:          StatusAvailable,
				CurrentVersion:  "3.0.0",
				LatestVersion:   "3.1.0",
				UpdateAvailable: true,
			},
		}

		plan, err := eng.Plan(outcomes, Filter{})
		if err != nil {
			t.Fatalf("Plan error: %v", err)
		}
		if len(plan.Updates) != 1 || plan.Updates[0].ToolID != "always-tool" {
			t.Fatalf("plan.Updates = %v, want 1 item (always-tool)", plan.Updates)
		}
		if len(plan.Current) != 0 {
			t.Errorf("plan.Current should be empty, got: %v", plan.Current)
		}
	})
}

func TestPlan_OwnedToolPolicyInheritance(t *testing.T) {
	cfg := &config.Config{}

	t.Run("gh owned by apt (PolicyGated) on Linux with UpdateAvailable == false placed in Current", func(t *testing.T) {
		eng := New(cfg, platform.OSLinux)
		// gh declares PolicyAlwaysUpdate in its own info, but apt declares PolicyGated.
		outcomes := []CheckOutcome{
			{
				ToolID:          "gh",
				ToolName:        "GitHub CLI",
				Status:          StatusCurrent,
				CurrentVersion:  "2.40.0",
				LatestVersion:   "2.40.0",
				UpdateAvailable: false,
			},
		}

		plan, err := eng.Plan(outcomes, Filter{})
		if err != nil {
			t.Fatalf("Plan error: %v", err)
		}
		if len(plan.Updates) != 0 {
			t.Errorf("gh unexpectedly placed in Updates, should inherit apt's PolicyGated and be Current: %v", plan.Updates)
		}
		if len(plan.Current) != 1 || plan.Current[0].ToolID != "gh" {
			t.Errorf("plan.Current = %v, want gh", plan.Current)
		}
	})

	t.Run("gh owned by apt on Linux with UpdateAvailable == true placed in Updates", func(t *testing.T) {
		eng := New(cfg, platform.OSLinux)
		outcomes := []CheckOutcome{
			{
				ToolID:          "gh",
				ToolName:        "GitHub CLI",
				Status:          StatusAvailable,
				CurrentVersion:  "2.40.0",
				LatestVersion:   "2.41.0",
				UpdateAvailable: true,
			},
		}

		plan, err := eng.Plan(outcomes, Filter{})
		if err != nil {
			t.Fatalf("Plan error: %v", err)
		}
		if len(plan.Updates) != 1 || plan.Updates[0].ToolID != "gh" {
			t.Errorf("plan.Updates = %v, want gh", plan.Updates)
		}
		if len(plan.Current) != 0 {
			t.Errorf("plan.Current = %v, want empty", plan.Current)
		}
	})

	t.Run("gh owned by brew (PolicyAlwaysUpdate) on macOS with UpdateAvailable == false placed in Updates", func(t *testing.T) {
		eng := New(cfg, platform.OSMacOS)
		outcomes := []CheckOutcome{
			{
				ToolID:          "gh",
				ToolName:        "GitHub CLI",
				Status:          StatusCurrent,
				CurrentVersion:  "2.40.0",
				LatestVersion:   "2.40.0",
				UpdateAvailable: false,
			},
		}

		plan, err := eng.Plan(outcomes, Filter{})
		if err != nil {
			t.Fatalf("Plan error: %v", err)
		}
		if len(plan.Updates) != 1 || plan.Updates[0].ToolID != "gh" {
			t.Errorf("plan.Updates = %v, want gh (inherited PolicyAlwaysUpdate from brew)", plan.Updates)
		}
		if len(plan.Current) != 0 {
			t.Errorf("plan.Current = %v, want empty", plan.Current)
		}
	})

	t.Run("custom tool with manager = apt inherits PolicyGated", func(t *testing.T) {
		aptMgr := official.AdapterByName("apt")
		customApt, err := adapters.NewCustomAdapter("mytool", "echo 1", "echo 1", true, aptMgr)
		if err != nil {
			t.Fatalf("NewCustomAdapter error: %v", err)
		}

		eng := New(cfg, platform.OSLinux, WithAdapters([]adapters.Adapter{aptMgr, customApt}))
		outcomes := []CheckOutcome{
			{
				ToolID:          "mytool",
				ToolName:        "mytool",
				Status:          StatusCurrent,
				CurrentVersion:  "1.0.0",
				LatestVersion:   "1.0.0",
				UpdateAvailable: false,
			},
		}

		plan, err := eng.Plan(outcomes, Filter{})
		if err != nil {
			t.Fatalf("Plan error: %v", err)
		}
		if len(plan.Updates) != 0 {
			t.Errorf("mytool unexpectedly placed in Updates: %v", plan.Updates)
		}
		if len(plan.Current) != 1 || plan.Current[0].ToolID != "mytool" {
			t.Errorf("plan.Current = %v, want mytool", plan.Current)
		}
	})

	t.Run("custom tool with manager = brew inherits PolicyAlwaysUpdate", func(t *testing.T) {
		brewMgr := official.AdapterByName("brew")
		customBrew, err := adapters.NewCustomAdapter("mybrewtool", "echo 1", "echo 1", true, brewMgr)
		if err != nil {
			t.Fatalf("NewCustomAdapter error: %v", err)
		}

		eng := New(cfg, platform.OSMacOS, WithAdapters([]adapters.Adapter{brewMgr, customBrew}))
		outcomes := []CheckOutcome{
			{
				ToolID:          "mybrewtool",
				ToolName:        "mybrewtool",
				Status:          StatusCurrent,
				CurrentVersion:  "1.0.0",
				LatestVersion:   "1.0.0",
				UpdateAvailable: false,
			},
		}

		plan, err := eng.Plan(outcomes, Filter{})
		if err != nil {
			t.Fatalf("Plan error: %v", err)
		}
		if len(plan.Updates) != 1 || plan.Updates[0].ToolID != "mybrewtool" {
			t.Errorf("plan.Updates = %v, want mybrewtool", plan.Updates)
		}
		if len(plan.Current) != 0 {
			t.Errorf("plan.Current = %v, want empty", plan.Current)
		}
	})
}

func TestPlan_PlannedUpdateFields(t *testing.T) {
	cfg := &config.Config{}

	t.Run("standalone tool planned update fields", func(t *testing.T) {
		standaloneAdapter := &mockPlanAdapter{
			info: adapters.ToolInfo{
				ID:           "my-standalone",
				Name:         "My Standalone Tool",
				UpdatePolicy: adapters.PolicyGated,
				Command:      "custom-standalone-upgrade --all",
				Privileges:   []string{"sudo"},
				Trust:        adapters.TrustCustomTrusted,
			},
		}

		eng := New(cfg, platform.OSLinux, WithAdapters([]adapters.Adapter{standaloneAdapter}))
		outcomes := []CheckOutcome{
			{
				ToolID:          "my-standalone",
				ToolName:        "My Standalone Tool",
				Status:          StatusAvailable,
				CurrentVersion:  "1.0.0",
				LatestVersion:   "1.2.0",
				UpdateAvailable: true,
			},
		}

		plan, err := eng.Plan(outcomes, Filter{})
		if err != nil {
			t.Fatalf("Plan error: %v", err)
		}
		if len(plan.Updates) != 1 {
			t.Fatalf("len(plan.Updates) = %d, want 1", len(plan.Updates))
		}

		pu := plan.Updates[0]
		if pu.ToolID != "my-standalone" {
			t.Errorf("ToolID = %q, want my-standalone", pu.ToolID)
		}
		if pu.ToolName != "My Standalone Tool" {
			t.Errorf("ToolName = %q, want My Standalone Tool", pu.ToolName)
		}
		if pu.ManagerID != "" {
			t.Errorf("ManagerID = %q, want empty", pu.ManagerID)
		}
		if pu.PackageName != "" {
			t.Errorf("PackageName = %q, want empty", pu.PackageName)
		}
		if pu.UpdatePolicy != adapters.PolicyGated {
			t.Errorf("UpdatePolicy = %v, want PolicyGated", pu.UpdatePolicy)
		}
		if pu.CurrentVersion != "1.0.0" {
			t.Errorf("CurrentVersion = %q, want 1.0.0", pu.CurrentVersion)
		}
		if pu.LatestVersion != "1.2.0" {
			t.Errorf("LatestVersion = %q, want 1.2.0", pu.LatestVersion)
		}
		if pu.RiskCommand != "custom-standalone-upgrade --all" {
			t.Errorf("RiskCommand = %q, want custom-standalone-upgrade --all", pu.RiskCommand)
		}
		if len(pu.Privileges) != 1 || pu.Privileges[0] != "sudo" {
			t.Errorf("Privileges = %v, want [sudo]", pu.Privileges)
		}
		if pu.Trust != adapters.TrustCustomTrusted {
			t.Errorf("Trust = %v, want TrustCustomTrusted", pu.Trust)
		}
	})

	t.Run("owned tool planned update fields on Linux (gh owned by apt)", func(t *testing.T) {
		eng := New(cfg, platform.OSLinux)
		outcomes := []CheckOutcome{
			{
				ToolID:          "gh",
				ToolName:        "GitHub CLI",
				Status:          StatusAvailable,
				CurrentVersion:  "2.40.0",
				LatestVersion:   "2.41.0",
				UpdateAvailable: true,
			},
		}

		plan, err := eng.Plan(outcomes, Filter{})
		if err != nil {
			t.Fatalf("Plan error: %v", err)
		}
		if len(plan.Updates) != 1 {
			t.Fatalf("len(plan.Updates) = %d, want 1", len(plan.Updates))
		}

		pu := plan.Updates[0]
		if pu.ToolID != "gh" {
			t.Errorf("ToolID = %q, want gh", pu.ToolID)
		}
		if pu.ToolName != "GitHub CLI" {
			t.Errorf("ToolName = %q, want GitHub CLI", pu.ToolName)
		}
		wantManagerID := official.AdapterByName("gh").Info().Manager[platform.OSLinux]
		wantPkgName := official.AdapterByName("gh").Info().ManagerPackage[platform.OSLinux]
		wantRiskCmd := "sudo apt install --only-upgrade gh"
		if wantManagerID == "pacman" {
			wantRiskCmd = "sudo pacman -S --noconfirm github-cli"
		}
		if pu.ManagerID != wantManagerID {
			t.Errorf("ManagerID = %q, want %s", pu.ManagerID, wantManagerID)
		}
		if pu.PackageName != wantPkgName {
			t.Errorf("PackageName = %q, want %s", pu.PackageName, wantPkgName)
		}
		if pu.UpdatePolicy != adapters.PolicyGated {
			t.Errorf("UpdatePolicy = %v, want PolicyGated", pu.UpdatePolicy)
		}
		if pu.CurrentVersion != "2.40.0" {
			t.Errorf("CurrentVersion = %q, want 2.40.0", pu.CurrentVersion)
		}
		if pu.LatestVersion != "2.41.0" {
			t.Errorf("LatestVersion = %q, want 2.41.0", pu.LatestVersion)
		}
		if pu.RiskCommand != wantRiskCmd {
			t.Errorf("RiskCommand = %q, want %s", pu.RiskCommand, wantRiskCmd)
		}
		if len(pu.Privileges) != 1 || pu.Privileges[0] != "sudo" {
			t.Errorf("Privileges = %v, want [sudo] from owner", pu.Privileges)
		}
		if pu.Trust != adapters.TrustOfficial {
			t.Errorf("Trust = %v, want TrustOfficial", pu.Trust)
		}
	})

	t.Run("owned tool planned update fields on macOS (gh owned by brew)", func(t *testing.T) {
		eng := New(cfg, platform.OSMacOS)
		outcomes := []CheckOutcome{
			{
				ToolID:          "gh",
				ToolName:        "GitHub CLI",
				Status:          StatusAvailable,
				CurrentVersion:  "2.40.0",
				LatestVersion:   "2.41.0",
				UpdateAvailable: true,
			},
		}

		plan, err := eng.Plan(outcomes, Filter{})
		if err != nil {
			t.Fatalf("Plan error: %v", err)
		}
		if len(plan.Updates) != 1 {
			t.Fatalf("len(plan.Updates) = %d, want 1", len(plan.Updates))
		}

		pu := plan.Updates[0]
		if pu.ManagerID != "brew" {
			t.Errorf("ManagerID = %q, want brew", pu.ManagerID)
		}
		if pu.PackageName != "gh" {
			t.Errorf("PackageName = %q, want gh", pu.PackageName)
		}
		if pu.RiskCommand != "brew upgrade gh" {
			t.Errorf("RiskCommand = %q, want brew upgrade gh", pu.RiskCommand)
		}
	})

	t.Run("standalone tool with empty Command falls back to toolName + update", func(t *testing.T) {
		fallbackAdapter := &mockPlanAdapter{
			info: adapters.ToolInfo{
				ID:           "generic-tool",
				Name:         "Generic Tool",
				UpdatePolicy: adapters.PolicyAlwaysUpdate,
				Command:      "",
			},
		}

		eng := New(cfg, platform.OSLinux, WithAdapters([]adapters.Adapter{fallbackAdapter}))
		outcomes := []CheckOutcome{
			{
				ToolID:          "generic-tool",
				ToolName:        "Generic Tool",
				Status:          StatusCurrent,
				CurrentVersion:  "1.0.0",
				LatestVersion:   "1.0.0",
				UpdateAvailable: false,
			},
		}

		plan, err := eng.Plan(outcomes, Filter{})
		if err != nil {
			t.Fatalf("Plan error: %v", err)
		}
		if len(plan.Updates) != 1 {
			t.Fatalf("len(plan.Updates) = %d, want 1", len(plan.Updates))
		}
		if plan.Updates[0].RiskCommand != "Generic Tool update" {
			t.Errorf("RiskCommand = %q, want Generic Tool update", plan.Updates[0].RiskCommand)
		}
	})
}

func TestPlan_FilterOnly(t *testing.T) {
	cfg := &config.Config{}
	fakes := []adapters.Adapter{
		&mockPlanAdapter{info: adapters.ToolInfo{
			ID:           "npm",
			Name:         "npm",
			UpdatePolicy: adapters.PolicyGated,
		}},
		&mockPlanAdapter{info: adapters.ToolInfo{
			ID:           "brew",
			Name:         "brew",
			UpdatePolicy: adapters.PolicyAlwaysUpdate,
		}},
		&mockPlanAdapter{info: adapters.ToolInfo{
			ID:           "bun",
			Name:         "bun",
			UpdatePolicy: adapters.PolicyAlwaysUpdate,
		}},
	}
	eng := New(cfg, platform.OSLinux, WithAdapters(fakes))

	outcomes := []CheckOutcome{
		{
			ToolID:          "npm",
			ToolName:        "npm",
			Status:          StatusAvailable,
			CurrentVersion:  "9.0.0",
			LatestVersion:   "10.0.0",
			UpdateAvailable: true,
		},
		{
			ToolID:          "brew",
			ToolName:        "brew",
			Status:          StatusCurrent,
			CurrentVersion:  "4.0.0",
			LatestVersion:   "4.0.0",
			UpdateAvailable: false,
		},
		{
			ToolID:          "bun",
			ToolName:        "bun",
			Status:          StatusCurrent,
			CurrentVersion:  "1.0.0",
			LatestVersion:   "1.0.0",
			UpdateAvailable: false,
		},
	}

	t.Run("filter only targets specific tool", func(t *testing.T) {
		plan, err := eng.Plan(outcomes, Filter{Only: []string{"npm"}})
		if err != nil {
			t.Fatalf("Plan error: %v", err)
		}
		if len(plan.Updates) != 1 || plan.Updates[0].ToolID != "npm" {
			t.Errorf("plan.Updates = %v, want only npm", plan.Updates)
		}
	})

	t.Run("filter case-insensitive matching", func(t *testing.T) {
		plan, err := eng.Plan(outcomes, Filter{Only: []string{"BREW"}})
		if err != nil {
			t.Fatalf("Plan error: %v", err)
		}
		if len(plan.Updates) != 1 || plan.Updates[0].ToolID != "brew" {
			t.Errorf("plan.Updates = %v, want only brew", plan.Updates)
		}
	})

	t.Run("filter multiple tools", func(t *testing.T) {
		plan, err := eng.Plan(outcomes, Filter{Only: []string{"npm", "bun"}})
		if err != nil {
			t.Fatalf("Plan error: %v", err)
		}
		if len(plan.Updates) != 2 {
			t.Fatalf("len(plan.Updates) = %d, want 2", len(plan.Updates))
		}
	})

	t.Run("filter unknown tool returns empty plan", func(t *testing.T) {
		plan, err := eng.Plan(outcomes, Filter{Only: []string{"unknown"}})
		if err != nil {
			t.Fatalf("Plan error: %v", err)
		}
		if len(plan.Updates) != 0 || len(plan.Current) != 0 || len(plan.Skipped) != 0 || len(plan.Failed) != 0 {
			t.Errorf("plan non-empty for unknown filter: %+v", plan)
		}
	})
}

func TestPlan_FilterAll(t *testing.T) {
	cfg := &config.Config{}
	fakes := []adapters.Adapter{
		&mockPlanAdapter{info: adapters.ToolInfo{
			ID:           "gated-tool",
			Name:         "gated-tool",
			UpdatePolicy: adapters.PolicyGated,
		}},
		&mockPlanAdapter{info: adapters.ToolInfo{
			ID:           "skipped-tool",
			Name:         "skipped-tool",
			UpdatePolicy: adapters.PolicyGated,
		}},
		&mockPlanAdapter{info: adapters.ToolInfo{
			ID:           "failed-tool",
			Name:         "failed-tool",
			UpdatePolicy: adapters.PolicyGated,
		}},
	}
	eng := New(cfg, platform.OSLinux, WithAdapters(fakes))

	outcomes := []CheckOutcome{
		{
			ToolID:          "gated-tool",
			ToolName:        "gated-tool",
			Status:          StatusCurrent,
			CurrentVersion:  "1.0.0",
			LatestVersion:   "1.0.0",
			UpdateAvailable: false,
		},
		{
			ToolID:          "skipped-tool",
			ToolName:        "skipped-tool",
			Status:          StatusSkipped,
			UpdateAvailable: false,
		},
		{
			ToolID:          "failed-tool",
			ToolName:        "failed-tool",
			Status:          StatusFailed,
			Err:             errors.New("fail"),
			UpdateAvailable: false,
		},
	}

	t.Run("without All=true gated tool with UpdateAvailable=false is Current", func(t *testing.T) {
		plan, err := eng.Plan(outcomes, Filter{All: false})
		if err != nil {
			t.Fatalf("Plan error: %v", err)
		}
		if len(plan.Updates) != 0 {
			t.Errorf("plan.Updates = %v, want 0", plan.Updates)
		}
		if len(plan.Current) != 1 || plan.Current[0].ToolID != "gated-tool" {
			t.Errorf("plan.Current = %v, want 1 item (gated-tool)", plan.Current)
		}
		if len(plan.Skipped) != 1 {
			t.Errorf("plan.Skipped = %v, want 1", plan.Skipped)
		}
		if len(plan.Failed) != 1 {
			t.Errorf("plan.Failed = %v, want 1", plan.Failed)
		}
	})

	t.Run("with All=true gated tool is treated as pending update", func(t *testing.T) {
		plan, err := eng.Plan(outcomes, Filter{All: true})
		if err != nil {
			t.Fatalf("Plan error: %v", err)
		}
		if len(plan.Updates) != 1 || plan.Updates[0].ToolID != "gated-tool" {
			t.Errorf("plan.Updates = %v, want 1 item (gated-tool)", plan.Updates)
		}
		if len(plan.Current) != 0 {
			t.Errorf("plan.Current = %v, want 0", plan.Current)
		}
		// Failed and skipped tools must still be segregated and never in Updates
		if len(plan.Skipped) != 1 || plan.Skipped[0].ToolID != "skipped-tool" {
			t.Errorf("plan.Skipped = %v, want 1 item (skipped-tool)", plan.Skipped)
		}
		if len(plan.Failed) != 1 || plan.Failed[0].ToolID != "failed-tool" {
			t.Errorf("plan.Failed = %v, want 1 item (failed-tool)", plan.Failed)
		}
	})
}

func TestResolveEffectiveUpdatePolicy(t *testing.T) {
	t.Run("nil adapter returns PolicyGated", func(t *testing.T) {
		got := ResolveEffectiveUpdatePolicy(nil, platform.OSLinux)
		if got != adapters.PolicyGated {
			t.Errorf("ResolveEffectiveUpdatePolicy(nil) = %v, want PolicyGated", got)
		}
	})

	t.Run("standalone tools return declared policy", func(t *testing.T) {
		npm := official.AdapterByName("npm")
		if npm == nil {
			t.Fatal("official npm adapter not found")
		}
		if got := ResolveEffectiveUpdatePolicy(npm, platform.OSLinux); got != adapters.PolicyGated {
			t.Errorf("npm policy on Linux = %v, want PolicyGated", got)
		}

		brew := official.AdapterByName("brew")
		if brew == nil {
			t.Fatal("official brew adapter not found")
		}
		if got := ResolveEffectiveUpdatePolicy(brew, platform.OSMacOS); got != adapters.PolicyAlwaysUpdate {
			t.Errorf("brew policy on macOS = %v, want PolicyAlwaysUpdate", got)
		}

		bun := official.AdapterByName("bun")
		if bun == nil {
			t.Fatal("official bun adapter not found")
		}
		if got := ResolveEffectiveUpdatePolicy(bun, platform.OSLinux); got != adapters.PolicyAlwaysUpdate {
			t.Errorf("bun policy on Linux = %v, want PolicyAlwaysUpdate", got)
		}
	})

	t.Run("owned tools return manager declared policy across platforms", func(t *testing.T) {
		gh := official.AdapterByName("gh")
		if gh == nil {
			t.Fatal("official gh adapter not found")
		}

		// On Linux, gh is owned by apt (PolicyGated)
		if got := ResolveEffectiveUpdatePolicy(gh, platform.OSLinux); got != adapters.PolicyGated {
			t.Errorf("gh policy on Linux (owned by apt) = %v, want PolicyGated", got)
		}

		// On macOS, gh is owned by brew (PolicyAlwaysUpdate)
		if got := ResolveEffectiveUpdatePolicy(gh, platform.OSMacOS); got != adapters.PolicyAlwaysUpdate {
			t.Errorf("gh policy on macOS (owned by brew) = %v, want PolicyAlwaysUpdate", got)
		}

		docker := official.AdapterByName("docker")
		if docker == nil {
			t.Fatal("official docker adapter not found")
		}

		// On Linux, docker is owned by apt (PolicyGated)
		if got := ResolveEffectiveUpdatePolicy(docker, platform.OSLinux); got != adapters.PolicyGated {
			t.Errorf("docker policy on Linux (owned by apt) = %v, want PolicyGated", got)
		}

		// On macOS, docker is owned by brew (PolicyAlwaysUpdate)
		if got := ResolveEffectiveUpdatePolicy(docker, platform.OSMacOS); got != adapters.PolicyAlwaysUpdate {
			t.Errorf("docker policy on macOS (owned by brew) = %v, want PolicyAlwaysUpdate", got)
		}

		goTool := official.AdapterByName("go")
		if goTool == nil {
			t.Fatal("official go adapter not found")
		}

		// On Linux, go is standalone (no manager), returns its declared policy (PolicyAlwaysUpdate)
		if got := ResolveEffectiveUpdatePolicy(goTool, platform.OSLinux); got != adapters.PolicyAlwaysUpdate {
			t.Errorf("go policy on Linux (standalone) = %v, want PolicyAlwaysUpdate", got)
		}

		// On macOS, go is owned by brew (PolicyAlwaysUpdate)
		if got := ResolveEffectiveUpdatePolicy(goTool, platform.OSMacOS); got != adapters.PolicyAlwaysUpdate {
			t.Errorf("go policy on macOS (owned by brew) = %v, want PolicyAlwaysUpdate", got)
		}
	})

	t.Run("injected mock manager policy overrides tool declared policy", func(t *testing.T) {
		mockGatedMgr := &mockPlanAdapter{
			info: adapters.ToolInfo{
				ID:           "mock-mgr-gated",
				Name:         "mock-mgr-gated",
				Kind:         adapters.KindManager,
				UpdatePolicy: adapters.PolicyGated,
			},
		}
		mockAlwaysMgr := &mockPlanAdapter{
			info: adapters.ToolInfo{
				ID:           "mock-mgr-always",
				Name:         "mock-mgr-always",
				Kind:         adapters.KindManager,
				UpdatePolicy: adapters.PolicyAlwaysUpdate,
			},
		}
		mockOwnedTool := &mockPlanAdapter{
			info: adapters.ToolInfo{
				ID:           "mock-tool",
				Name:         "mock-tool",
				UpdatePolicy: adapters.PolicyAlwaysUpdate, // declared is AlwaysUpdate
				Manager: map[string]string{
					"linux": "mock-mgr-gated",
					"macos": "mock-mgr-always",
				},
			},
		}

		// Linux: mock-mgr-gated governs -> PolicyGated
		allLinux := []adapters.Adapter{mockGatedMgr, mockOwnedTool}
		if got := ResolveEffectiveUpdatePolicy(mockOwnedTool, "linux", allLinux); got != adapters.PolicyGated {
			t.Errorf("mockOwnedTool on Linux with mockGatedMgr = %v, want PolicyGated", got)
		}

		// macOS: mock-mgr-always governs -> PolicyAlwaysUpdate
		allMacOS := []adapters.Adapter{mockAlwaysMgr, mockOwnedTool}
		if got := ResolveEffectiveUpdatePolicy(mockOwnedTool, "macos", allMacOS); got != adapters.PolicyAlwaysUpdate {
			t.Errorf("mockOwnedTool on macOS with mockAlwaysMgr = %v, want PolicyAlwaysUpdate", got)
		}
	})

	t.Run("custom tool with manager inherits manager policy", func(t *testing.T) {
		aptMgr := official.AdapterByName("apt")
		brewMgr := official.AdapterByName("brew")

		customApt, err := adapters.NewCustomAdapter("custom-apt", "cmd", "check", true, aptMgr)
		if err != nil {
			t.Fatalf("NewCustomAdapter error: %v", err)
		}
		if got := ResolveEffectiveUpdatePolicy(customApt, platform.OSLinux); got != adapters.PolicyGated {
			t.Errorf("custom-apt policy = %v, want PolicyGated", got)
		}

		customBrew, err := adapters.NewCustomAdapter("custom-brew", "cmd", "check", true, brewMgr)
		if err != nil {
			t.Fatalf("NewCustomAdapter error: %v", err)
		}
		if got := ResolveEffectiveUpdatePolicy(customBrew, platform.OSMacOS); got != adapters.PolicyAlwaysUpdate {
			t.Errorf("custom-brew policy = %v, want PolicyAlwaysUpdate", got)
		}

		customStandalone, err := adapters.NewCustomAdapter("custom-standalone", "cmd", "check", true)
		if err != nil {
			t.Fatalf("NewCustomAdapter error: %v", err)
		}
		if got := ResolveEffectiveUpdatePolicy(customStandalone, platform.OSLinux); got != adapters.PolicyAlwaysUpdate {
			t.Errorf("custom-standalone policy = %v, want PolicyAlwaysUpdate", got)
		}
	})
}

// --- fix-risk-command-synthesis WU2: row-class plan derivation + gate truth ---
//
// D3: the plan derives RiskCommand from the adapter declaration of the row's
// class — manager self-row -> SelfUpdateCommand; owned-package row ->
// RenderPackageCommand(owner.PackageUpdateCommand, pkg); custom
// manager-delegated row -> the delegated manager's SelfUpdateCommand (because
// custom.Update() runs manager.Update()); standalone -> the declared command or
// the "<name> update" fallback. D4: a manager self-row is marked with ManagerID
// iff its Info() declares privileges, so only privileged managers (pacman)
// tighten the gate through EnforceRisk.

// planRowForAdapter derives the single PlannedUpdate the engine produces for one
// adapter on osName, so tests can assert the exact RiskCommand/ManagerID/
// Privileges the confirmation gate consumes.
func planRowForAdapter(t *testing.T, osName string, allAdapters []adapters.Adapter, id, name string) PlannedUpdate {
	t.Helper()
	eng := New(&config.Config{}, osName, WithAdapters(allAdapters))
	outcomes := []CheckOutcome{{
		ToolID:          id,
		ToolName:        name,
		Status:          StatusAvailable,
		CurrentVersion:  "1.0.0",
		LatestVersion:   "2.0.0",
		UpdateAvailable: true,
	}}
	plan, err := eng.Plan(outcomes, Filter{})
	if err != nil {
		t.Fatalf("Plan error: %v", err)
	}
	if len(plan.Updates) != 1 {
		t.Fatalf("len(plan.Updates) = %d, want 1 (current=%d skipped=%d failed=%d)",
			len(plan.Updates), len(plan.Current), len(plan.Skipped), len(plan.Failed))
	}
	return plan.Updates[0]
}

// pacmanOwnedFixture is an owned tool that resolves to the REAL pacman manager
// on Linux. The registry has no pacman-owned tool today, so this synthetic
// fixture is the only way to reach the owned-render row class for pacman.
func pacmanOwnedFixture() adapters.Adapter {
	return &mockPlanAdapter{info: adapters.ToolInfo{
		ID:             "ripgrep",
		Name:           "ripgrep",
		Kind:           adapters.KindTool,
		UpdatePolicy:   adapters.PolicyAlwaysUpdate,
		Trust:          adapters.TrustOfficial,
		Manager:        map[string]string{"linux": "pacman"},
		ManagerPackage: map[string]string{"linux": "ripgrep"},
	}}
}

func TestPlan_RowClassRiskCommands(t *testing.T) {
	pacman := official.AdapterByName("pacman")
	apt := official.AdapterByName("apt")
	brew := official.AdapterByName("brew")
	winget := official.AdapterByName("winget")
	scoop := official.AdapterByName("scoop")
	npm := official.AdapterByName("npm")

	customDelegated, err := adapters.NewCustomAdapter("mytool", "echo update", "", false, pacman)
	if err != nil {
		t.Fatalf("NewCustomAdapter error: %v", err)
	}

	customStandalone := &mockPlanAdapter{info: adapters.ToolInfo{
		ID:           "standalone-custom",
		Name:         "standalone-custom",
		Kind:         adapters.KindTool,
		UpdatePolicy: adapters.PolicyAlwaysUpdate,
		Trust:        adapters.TrustCustomTrusted,
		Command:      "custom-standalone-upgrade --all",
	}}

	tests := []struct {
		name      string
		osName    string
		all       []adapters.Adapter
		id        string
		toolName  string
		wantRisk  string
		wantMgrID string
		wantPrivs []string
	}{
		{
			name:      "pacman self row uses the real sudo command and marks ManagerID",
			osName:    platform.OSLinux,
			all:       []adapters.Adapter{pacman},
			id:        "pacman",
			toolName:  "Pacman Package Manager",
			wantRisk:  "sudo pacman -S --noconfirm pacman",
			wantMgrID: "pacman",
			wantPrivs: []string{"sudo"},
		},
		{
			name:      "apt self row uses the real command but stays auto-proceed (no declared privileges)",
			osName:    platform.OSLinux,
			all:       []adapters.Adapter{apt},
			id:        "apt",
			toolName:  "APT Package Manager",
			wantRisk:  "sudo apt install --only-upgrade apt",
			wantMgrID: "",
			wantPrivs: []string{"sudo"},
		},
		{
			name:      "brew self row uses the real command",
			osName:    platform.OSMacOS,
			all:       []adapters.Adapter{brew},
			id:        "brew",
			toolName:  "Homebrew",
			wantRisk:  "brew update",
			wantMgrID: "",
			wantPrivs: nil,
		},
		{
			name:      "winget self row uses the real command",
			osName:    platform.OSWindows,
			all:       []adapters.Adapter{winget},
			id:        "winget",
			toolName:  "Windows Package Manager",
			wantRisk:  "winget upgrade winget",
			wantMgrID: "",
			wantPrivs: nil,
		},
		{
			name:      "scoop self row uses the real command",
			osName:    platform.OSWindows,
			all:       []adapters.Adapter{scoop},
			id:        "scoop",
			toolName:  "Scoop",
			wantRisk:  "scoop update scoop",
			wantMgrID: "",
			wantPrivs: nil,
		},
		{
			name:      "pacman owned package row renders the real privileged package command",
			osName:    platform.OSLinux,
			all:       []adapters.Adapter{pacman, pacmanOwnedFixture()},
			id:        "ripgrep",
			toolName:  "ripgrep",
			wantRisk:  "sudo pacman -S --noconfirm ripgrep",
			wantMgrID: "pacman",
			wantPrivs: []string{"sudo"},
		},
		{
			name:      "custom manager-delegated row inherits the manager's real self command",
			osName:    platform.OSLinux,
			all:       []adapters.Adapter{pacman, customDelegated},
			id:        "mytool",
			toolName:  "mytool",
			wantRisk:  "sudo pacman -S --noconfirm pacman",
			wantMgrID: "pacman",
			wantPrivs: []string{"sudo"},
		},
		{
			name:      "custom standalone keeps its declared command",
			osName:    platform.OSLinux,
			all:       []adapters.Adapter{customStandalone},
			id:        "standalone-custom",
			toolName:  "standalone-custom",
			wantRisk:  "custom-standalone-upgrade --all",
			wantMgrID: "",
			wantPrivs: nil,
		},
		{
			name:      "official standalone npm declares its real update command",
			osName:    platform.OSLinux,
			all:       []adapters.Adapter{npm},
			id:        "npm",
			toolName:  "npm",
			wantRisk:  "npm update -g",
			wantMgrID: "",
			wantPrivs: nil,
		},
		{
			name:      "official standalone pnpm declares its real update command",
			osName:    platform.OSLinux,
			all:       []adapters.Adapter{official.AdapterByName("pnpm")},
			id:        "pnpm",
			toolName:  "pnpm",
			wantRisk:  "pnpm update -g",
			wantMgrID: "",
			wantPrivs: nil,
		},
		{
			name:      "official standalone bun declares its real update command",
			osName:    platform.OSLinux,
			all:       []adapters.Adapter{official.AdapterByName("bun")},
			id:        "bun",
			toolName:  "bun",
			wantRisk:  "bun upgrade",
			wantMgrID: "",
			wantPrivs: nil,
		},
		{
			name:      "official standalone uv declares its real update command",
			osName:    platform.OSLinux,
			all:       []adapters.Adapter{official.AdapterByName("uv")},
			id:        "uv",
			toolName:  "uv",
			wantRisk:  "uv self update",
			wantMgrID: "",
			wantPrivs: nil,
		},
		{
			name:      "official standalone opencode declares its real installer command",
			osName:    platform.OSLinux,
			all:       []adapters.Adapter{official.AdapterByName("opencode")},
			id:        "opencode",
			toolName:  "opencode",
			wantRisk:  "opencode update",
			wantMgrID: "",
			wantPrivs: nil,
		},
		{
			// nvm's declared command chains with `&&`, so ClassifyCommand
			// returns RiskMedium and the gate now prompts for it interactively
			// (see enforceRiskFor in internal/cli). Pinned here so the declared
			// command and its consequence stay visible together.
			name:      "official standalone nvm declares its real update command",
			osName:    platform.OSLinux,
			all:       []adapters.Adapter{official.AdapterByName("nvm")},
			id:        "nvm",
			toolName:  "Node Version Manager",
			wantRisk:  "bash -c 'source \"${NVM_DIR:-$HOME/.nvm}/nvm.sh\" >/dev/null 2>&1 && nvm install --lts'",
			wantMgrID: "",
			wantPrivs: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pu := planRowForAdapter(t, tt.osName, tt.all, tt.id, tt.toolName)
			if pu.RiskCommand != tt.wantRisk {
				t.Errorf("RiskCommand = %q, want %q", pu.RiskCommand, tt.wantRisk)
			}
			if pu.ManagerID != tt.wantMgrID {
				t.Errorf("ManagerID = %q, want %q", pu.ManagerID, tt.wantMgrID)
			}
			if !reflect.DeepEqual(pu.Privileges, tt.wantPrivs) {
				t.Errorf("Privileges = %v, want %v", pu.Privileges, tt.wantPrivs)
			}
		})
	}
}

// TestPlan_RiskCommandEqualsExecutedDeclaration is the D2 byte-equality pin at
// the plan boundary (spec security-model "Gate input is the executed command",
// spec tool-adapter "Declared equals executed"). For every managed row,
// plan.RiskCommand MUST byte-equal the adapter declaration that the update path
// actually executes:
//
//   - self/delegated rows -> ToolInfo.SelfUpdateCommand
//   - owned rows          -> RenderPackageCommand(owner.PackageUpdateCommand, pkg)
//
// The execution half of the equality is proven in package official by
// TestUpdateRunsDeclaredCommand, which replaces the runCmdFn seam and captures
// the exact command Update(false)/UpdatePackage(pkg) runs. That seam is
// package-private (engine imports official, so official cannot import engine),
// so the two equalities are proven in their own packages and composed here:
// plan == declaration == executed command. The apt/brew/winget owned literals
// are additionally pinned byte-for-byte to prove no prompt churn.
func TestPlan_RiskCommandEqualsExecutedDeclaration(t *testing.T) {
	gh := official.AdapterByName("gh")
	ghLinuxMgr := gh.Info().Manager[platform.OSLinux]
	ghLinuxPkg := gh.Info().ManagerPackage[platform.OSLinux]
	ghLinuxLiteral := "sudo apt install --only-upgrade gh"
	if ghLinuxMgr == "pacman" {
		ghLinuxLiteral = "sudo pacman -S --noconfirm github-cli"
	}

	tests := []struct {
		name        string
		osName      string
		all         []adapters.Adapter
		id          string
		toolName    string
		mgrID       string
		pkg         string // "" -> self row
		wantLiteral string // optional byte-identical cross-version pin
	}{
		{
			name: "apt self", osName: platform.OSLinux,
			all: []adapters.Adapter{official.AdapterByName("apt")},
			id:  "apt", toolName: "APT Package Manager", mgrID: "apt",
		},
		{
			name: "brew self", osName: platform.OSMacOS,
			all: []adapters.Adapter{official.AdapterByName("brew")},
			id:  "brew", toolName: "Homebrew", mgrID: "brew",
		},
		{
			name: "winget self", osName: platform.OSWindows,
			all: []adapters.Adapter{official.AdapterByName("winget")},
			id:  "winget", toolName: "Windows Package Manager", mgrID: "winget",
		},
		{
			name: "scoop self", osName: platform.OSWindows,
			all: []adapters.Adapter{official.AdapterByName("scoop")},
			id:  "scoop", toolName: "Scoop", mgrID: "scoop",
		},
		{
			name: "pacman self", osName: platform.OSLinux,
			all: []adapters.Adapter{official.AdapterByName("pacman")},
			id:  "pacman", toolName: "Pacman Package Manager", mgrID: "pacman",
		},
		{
			name: "linux owned gh", osName: platform.OSLinux,
			all: []adapters.Adapter{gh, official.AdapterByName(ghLinuxMgr)},
			id:  "gh", toolName: "GitHub CLI", mgrID: ghLinuxMgr, pkg: ghLinuxPkg,
			wantLiteral: ghLinuxLiteral,
		},
		{
			name: "brew owned gh", osName: platform.OSMacOS,
			all: []adapters.Adapter{gh, official.AdapterByName("brew")},
			id:  "gh", toolName: "GitHub CLI", mgrID: "brew", pkg: "gh",
			wantLiteral: "brew upgrade gh",
		},
		{
			name: "winget owned gh", osName: platform.OSWindows,
			all: []adapters.Adapter{gh, official.AdapterByName("winget")},
			id:  "gh", toolName: "GitHub CLI", mgrID: "winget", pkg: "gh",
			wantLiteral: "winget upgrade gh",
		},
		{
			name: "pacman owned ripgrep (synthetic owned fixture)", osName: platform.OSLinux,
			all: []adapters.Adapter{official.AdapterByName("pacman"), pacmanOwnedFixture()},
			id:  "ripgrep", toolName: "ripgrep", mgrID: "pacman", pkg: "ripgrep",
			wantLiteral: "sudo pacman -S --noconfirm ripgrep",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := official.AdapterByName(tt.mgrID).Info()
			want := info.SelfUpdateCommand
			if tt.pkg != "" {
				want = adapters.RenderPackageCommand(info.PackageUpdateCommand, tt.pkg)
			}

			pu := planRowForAdapter(t, tt.osName, tt.all, tt.id, tt.toolName)
			if pu.RiskCommand != want {
				t.Errorf("RiskCommand = %q, want the executed declaration %q", pu.RiskCommand, want)
			}
			if tt.wantLiteral != "" && pu.RiskCommand != tt.wantLiteral {
				t.Errorf("RiskCommand = %q, want byte-identical literal %q", pu.RiskCommand, tt.wantLiteral)
			}
		})
	}
}
