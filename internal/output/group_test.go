package output

import (
	"errors"
	"slices"
	"testing"

	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/adapters/official"
	"github.com/JhnFrankz/upp/internal/engine"
	"github.com/JhnFrankz/upp/internal/platform"
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

	groups := PresentGroups(toolGroups, outcomes)

	if len(groups) != 2 {
		t.Fatalf("PresentGroups returned %d groups, want 2", len(groups))
	}

	// First group: Homebrew
	if groups[0].Header != "Homebrew" {
		t.Errorf("group[0].Header = %q, want %q", groups[0].Header, "Homebrew")
	}
	if len(groups[0].Items) != 2 {
		t.Fatalf("group[0].Items len = %d, want 2", len(groups[0].Items))
	}
	// brew
	if item := groups[0].Items[0]; item.ID != "brew" || item.Status != StatusCurrent || item.Version != "4.2.0" {
		t.Errorf("brew item = %+v, want ID=brew, Status=StatusCurrent, Version=4.2.0", item)
	}
	// gh
	if item := groups[0].Items[1]; item.ID != "gh" || item.Status != StatusAvailable || item.Version != "2.40.0" {
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
	if item := groups[1].Items[0]; item.ID != "npm" || item.Status != StatusFailed || item.Version != "" {
		t.Errorf("npm item = %+v, want ID=npm, Status=StatusFailed, Version=\"\"", item)
	}
	// bun (missing outcome -> Skipped)
	if item := groups[1].Items[1]; item.ID != "bun" || item.Status != StatusSkipped || item.Version != "" {
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

	groups := PresentGroups(toolGroups, outcomes)
	if len(groups) != 1 || len(groups[0].Items) != 1 {
		t.Fatalf("expected 1 group with 1 item, got %+v", groups)
	}
	if item := groups[0].Items[0]; item.Status != StatusSkipped {
		t.Errorf("item status = %v, want StatusSkipped", item.Status)
	}
}

func TestBackwardCompatibilityHelpers(t *testing.T) {
	tools := []adapters.Adapter{
		official.AdapterByName("npm"),
		official.AdapterByName("gh"),
		official.AdapterByName("brew"),
	}

	// GroupOrder
	ordered := GroupOrder(tools, platform.OSMacOS)
	names := make([]string, len(ordered))
	for i, a := range ordered {
		names[i] = a.Name()
	}
	if !slices.Equal(names, []string{"brew", "gh", "npm"}) {
		t.Errorf("GroupOrder = %v, want [brew gh npm]", names)
	}

	// OwnerGroupLabel
	label := OwnerGroupLabel(official.AdapterByName("gh"), platform.OSMacOS, tools)
	if label != "Homebrew" {
		t.Errorf("OwnerGroupLabel = %q, want Homebrew", label)
	}

	// GroupByOwner
	groups := GroupByOwner(tools, platform.OSMacOS)
	if len(groups) < 1 {
		t.Fatalf("GroupByOwner returned empty groups")
	}
	if groups[0].Header != "Homebrew" {
		t.Errorf("GroupByOwner header = %q, want Homebrew", groups[0].Header)
	}
}
