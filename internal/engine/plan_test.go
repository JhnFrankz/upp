package engine

import (
	"errors"
	"testing"

	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/adapters/official"
	"github.com/JhnFrankz/upp/internal/config"
	"github.com/JhnFrankz/upp/internal/platform"
)

type mockPlanAdapter struct {
	info adapters.ToolInfo
}

func (m *mockPlanAdapter) Name() string                                { return m.info.Name }
func (m *mockPlanAdapter) Detect() bool                                { return true }
func (m *mockPlanAdapter) Check() (adapters.UpdateInfo, error)         { return adapters.UpdateInfo{}, nil }
func (m *mockPlanAdapter) Update(dryRun bool) (adapters.Result, error) { return adapters.Result{}, nil }
func (m *mockPlanAdapter) Info() adapters.ToolInfo                     { return m.info }

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
		if pu.ManagerID != "apt" {
			t.Errorf("ManagerID = %q, want apt", pu.ManagerID)
		}
		if pu.PackageName != "gh" {
			t.Errorf("PackageName = %q, want gh", pu.PackageName)
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
		if pu.RiskCommand != "sudo apt install --only-upgrade gh" {
			t.Errorf("RiskCommand = %q, want sudo apt install --only-upgrade gh", pu.RiskCommand)
		}
		if len(pu.Privileges) != 1 || pu.Privileges[0] != "sudo" {
			t.Errorf("Privileges = %v, want [sudo] from apt owner", pu.Privileges)
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
