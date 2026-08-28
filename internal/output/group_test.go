package output

import (
	"slices"
	"testing"

	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/adapters/official"
	"github.com/JhnFrankz/upp/internal/platform"
)

// adapterNames is a tiny helper that extracts the Name() of each adapter in
// order, so grouped-order assertions compare against the identifier that
// --only/--skip round-trip on rather than the display label.
func adapterNames(adapters []adapters.Adapter) []string {
	names := make([]string, len(adapters))
	for i, a := range adapters {
		names[i] = a.Name()
	}
	return names
}

// --- GroupOrder (WU3 interactive grouping) ---

// TestGroupOrder_OwnedToolGroupedUnderManager proves GroupOrder reorders the
// input so a manager row leads its group, the owned tool follows it, and a
// standalone tool lands last. On macOS gh is owned by brew, npm is standalone.
func TestGroupOrder_OwnedToolGroupedUnderManager(t *testing.T) {
	tools := []adapters.Adapter{
		official.AdapterByName("npm"),
		official.AdapterByName("gh"),
		official.AdapterByName("brew"),
	}
	ordered := GroupOrder(tools, platform.OSMacOS)
	if got, want := adapterNames(ordered), []string{"brew", "gh", "npm"}; !slices.Equal(got, want) {
		t.Errorf("GroupOrder = %v, want %v (manager first, owned tool, then standalone)", got, want)
	}
}

// TestGroupOrder_ManagersFollowCanonicalAllAdaptersOrder proves manager groups
// are emitted in official.AllAdapters order (apt, brew, winget, scoop), not the
// input order. On linux both apt and brew are present and gh+docker are owned by
// apt, so apt's group leads even though brew appears after them in the input.
func TestGroupOrder_ManagersFollowCanonicalAllAdaptersOrder(t *testing.T) {
	tools := []adapters.Adapter{
		official.AdapterByName("brew"),
		official.AdapterByName("gh"),
		official.AdapterByName("docker"),
		official.AdapterByName("apt"),
	}
	ordered := GroupOrder(tools, platform.OSLinux)
	if got, want := adapterNames(ordered), []string{"apt", "gh", "docker", "brew"}; !slices.Equal(got, want) {
		t.Errorf("GroupOrder = %v, want %v (apt group leads per AllAdapters order)", got, want)
	}
}

// TestGroupOrder_PerPlatformResolution proves the owner is resolved from the
// request's platform: gh belongs to brew on macOS, apt on Linux, winget on
// Windows. The same input reorders differently per platform.
func TestGroupOrder_PerPlatformResolution(t *testing.T) {
	tests := []struct {
		name    string
		os      string
		manager string
	}{
		{name: "macos", os: platform.OSMacOS, manager: "brew"},
		{name: "linux", os: platform.OSLinux, manager: "apt"},
		{name: "windows", os: platform.OSWindows, manager: "winget"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tools := []adapters.Adapter{
				official.AdapterByName(tt.manager),
				official.AdapterByName("gh"),
			}
			ordered := GroupOrder(tools, tt.os)
			if got, want := adapterNames(ordered), []string{tt.manager, "gh"}; !slices.Equal(got, want) {
				t.Errorf("GroupOrder(%s) = %v, want %v", tt.os, got, want)
			}
		})
	}
}

// TestGroupOrder_FilteredManagerFallsToStandalone proves a tool whose manager
// was filtered out (apt removed from the set) does NOT produce a phantom group:
// it falls to the standalone tail, preserving the flat --only/--skip round-trip.
func TestGroupOrder_FilteredManagerFallsToStandalone(t *testing.T) {
	tools := []adapters.Adapter{official.AdapterByName("docker")} // linux owner apt absent
	ordered := GroupOrder(tools, platform.OSLinux)
	if got, want := adapterNames(ordered), []string{"docker"}; !slices.Equal(got, want) {
		t.Errorf("GroupOrder = %v, want %v (filtered manager must not reorder or drop docker)", got, want)
	}
}

// TestGroupOrder_CustomToolWithInjectedManager proves a custom adapter declaring
// a resolving owner is grouped under that manager, via ownerIDOf's CustomAdapter
// branch rather than the ToolInfo.Manager map (which a custom tool never sets).
func TestGroupOrder_CustomToolWithInjectedManager(t *testing.T) {
	brew := official.AdapterByName("brew")
	custom, err := adapters.NewCustomAdapter("mytool", "mytool", "", false, brew)
	if err != nil {
		t.Fatalf("NewCustomAdapter() error: %v", err)
	}
	tools := []adapters.Adapter{custom, brew}
	ordered := GroupOrder(tools, platform.OSMacOS)
	// brew (manager) leads, then the custom tool it owns; nothing is standalone.
	if got, want := adapterNames(ordered), []string{"brew", "mytool"}; !slices.Equal(got, want) {
		t.Errorf("GroupOrder = %v, want %v (custom tool grouped under its injected manager)", got, want)
	}
}

// --- OwnerGroupLabel (selector group header wiring) ---

// TestOwnerGroupLabel_OwnedToolReturnsManagerLabel proves a tool whose owner is
// present in the run set resolves to the manager display name, so the selector
// renders a group header for the owned tool.
func TestOwnerGroupLabel_OwnedToolReturnsManagerLabel(t *testing.T) {
	brew := official.AdapterByName("brew")
	gh := official.AdapterByName("gh")
	tools := []adapters.Adapter{brew, gh}
	if got := OwnerGroupLabel(gh, platform.OSMacOS, tools); got != "Homebrew" {
		t.Errorf("OwnerGroupLabel(gh, macos) = %q, want %q", got, "Homebrew")
	}
}

// TestOwnerGroupLabel_StandaloneToolReturnsEmpty proves a tool with no resolving
// owner on this platform gets no group header.
func TestOwnerGroupLabel_StandaloneToolReturnsEmpty(t *testing.T) {
	npm := official.AdapterByName("npm")
	tools := []adapters.Adapter{npm}
	if got := OwnerGroupLabel(npm, platform.OSLinux, tools); got != "" {
		t.Errorf("OwnerGroupLabel(standalone) = %q, want empty", got)
	}
}

// TestOwnerGroupLabel_FilteredManagerReturnsEmpty proves a tool whose owner is
// NOT in the run set produces no phantom group header (the selector never shows
// a header for a manager that was filtered out).
func TestOwnerGroupLabel_FilteredManagerReturnsEmpty(t *testing.T) {
	gh := official.AdapterByName("gh")
	// gh is owned by apt on linux, but apt is not among the tools.
	if got := OwnerGroupLabel(gh, platform.OSLinux, []adapters.Adapter{gh}); got != "" {
		t.Errorf("OwnerGroupLabel(gh, filtered apt) = %q, want empty", got)
	}
}

// TestOwnerGroupLabel_CustomToolWithInjectedManager proves a custom adapter's
// injected manager resolves to the manager display name via the CustomAdapter
// branch of ownerIDOf, exactly like the interactive selector wiring.
func TestOwnerGroupLabel_CustomToolWithInjectedManager(t *testing.T) {
	brew := official.AdapterByName("brew")
	custom, err := adapters.NewCustomAdapter("mytool", "mytool", "", false, brew)
	if err != nil {
		t.Fatalf("NewCustomAdapter() error: %v", err)
	}
	tools := []adapters.Adapter{brew, custom}
	if got := OwnerGroupLabel(custom, platform.OSMacOS, tools); got != "Homebrew" {
		t.Errorf("OwnerGroupLabel(custom tool) = %q, want %q", got, "Homebrew")
	}
}

// TestOwnerIDOf covers ownerIDOf directly (verify WARNING: output.ownerIDOf
// ~80% coverage). It proves both branches: the CustomAdapter branch (a custom
// tool returns its injected manager's Name()) and the official branch (a tool
// reads its canonical Info().Manager[os] — present when owned on the platform,
// absent when standalone).
func TestOwnerIDOf(t *testing.T) {
	t.Run("custom adapter with injected manager returns the manager ID", func(t *testing.T) {
		brew := official.AdapterByName("brew")
		custom, err := adapters.NewCustomAdapter("mytool", "mytool", "", false, brew)
		if err != nil {
			t.Fatalf("NewCustomAdapter() error: %v", err)
		}
		if got := ownerIDOf(custom, platform.OSMacOS); got != "brew" {
			t.Errorf("ownerIDOf(custom with brew manager) = %q, want %q", got, "brew")
		}
	})

	t.Run("custom adapter without manager is standalone empty", func(t *testing.T) {
		custom, err := adapters.NewCustomAdapter("solo", "solo", "", false)
		if err != nil {
			t.Fatalf("NewCustomAdapter() error: %v", err)
		}
		if got := ownerIDOf(custom, platform.OSLinux); got != "" {
			t.Errorf("ownerIDOf(standalone custom) = %q, want empty", got)
		}
	})

	t.Run("official tool reads Manager[os] present", func(t *testing.T) {
		gh := official.AdapterByName("gh")
		if got := ownerIDOf(gh, platform.OSLinux); got != "apt" {
			t.Errorf("ownerIDOf(gh, linux) = %q, want %q", got, "apt")
		}
		if got := ownerIDOf(gh, platform.OSWindows); got != "winget" {
			t.Errorf("ownerIDOf(gh, windows) = %q, want %q", got, "winget")
		}
	})

	t.Run("official tool reads Manager[os] absent (standalone)", func(t *testing.T) {
		goTool := official.AdapterByName("go")
		// go owns nothing on Linux: Manager has no "linux" key → "".
		if got := ownerIDOf(goTool, platform.OSLinux); got != "" {
			t.Errorf("ownerIDOf(go, linux) = %q, want empty (standalone)", got)
		}
		npm := official.AdapterByName("npm")
		if got := ownerIDOf(npm, platform.OSLinux); got != "" {
			t.Errorf("ownerIDOf(npm, linux) = %q, want empty (standalone)", got)
		}
	})
}

// --- GroupByOwner (grouped list/selector rows, supplementary) ---

// TestGroupByOwner_PerPlatformBuckets proves GroupByOwner buckets gh under its
// resolving manager per platform, using the official adapters for that platform
// so the owner manager is actually present in the set.
func TestGroupByOwner_PerPlatformBuckets(t *testing.T) {
	tests := []struct {
		name       string
		os         string
		managerID  string
		wantHeader string
	}{
		{name: "macos", os: platform.OSMacOS, managerID: "brew", wantHeader: "Homebrew"},
		{name: "linux", os: platform.OSLinux, managerID: "apt", wantHeader: "APT Package Manager"},
		{name: "windows", os: platform.OSWindows, managerID: "winget", wantHeader: "Windows Package Manager"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := official.AdapterByName(tt.managerID)
			gh := official.AdapterByName("gh")
			tools := []adapters.Adapter{manager, gh}

			groups := GroupByOwner(tools, tt.os)
			if len(groups) != 1 {
				t.Fatalf("GroupByOwner(%s) produced %d groups, want 1: %+v", tt.os, len(groups), groups)
			}
			if groups[0].Header != tt.wantHeader {
				t.Errorf("group header = %q, want %q", groups[0].Header, tt.wantHeader)
			}
			ids := make([]string, len(groups[0].Items))
			for i, item := range groups[0].Items {
				ids[i] = item.ID
			}
			if !slices.Equal(ids, []string{tt.managerID, "gh"}) {
				t.Errorf("group items = %v, want [%s gh] (manager row leads, owned tool follows)", ids, tt.managerID)
			}
		})
	}
}

// --- WU5 refactor: deterministic canonical discovery order (task 5.3) ---
//
// The group bulk summary must follow the same canonical discovery order the
// board/selector use (manager rows first in AllAdapters order, then their owned
// tools, then standalone tools), regardless of input order. These tests pin
// that both grouping helpers (GroupByOwner for list/summary grouping and
// GroupOrder for the board/selector feed) agree on the deterministic canonical
// order and preserve the selector/board feed round-trip.

// TestGroupByOwner_DeterministicCanonicalOrder proves GroupByOwner emits groups
// in the deterministic canonical discovery order when fed the canonical adapter
// feed (buildAdapterList → AdaptersForPlatform): manager groups lead in
// official.AllAdapters order (each manager's own row first, then its owned
// tools in the feed order), and standalone tools follow. Every tool appears
// exactly once. This is the order the group bulk summary mirrors so concurrent
// completion never reorders it.
func TestGroupByOwner_DeterministicCanonicalOrder(t *testing.T) {
	// The canonical platform feed (mirrors buildAdapterList → AdaptersForPlatform).
	tools := official.AdaptersForPlatform(platform.OSLinux)

	groups := GroupByOwner(tools, platform.OSLinux)

	// Manager group headers follow AllAdapters order: apt leads brew on Linux.
	var gotHeaders []string
	for _, g := range groups {
		if g.Header != "" {
			gotHeaders = append(gotHeaders, g.Header)
		}
	}
	if !slices.Equal(gotHeaders, []string{"APT Package Manager", "Homebrew"}) {
		t.Errorf("group headers = %v, want [APT Package Manager Homebrew]", gotHeaders)
	}
	// apt's group: the manager row leads, then its owned tools gh + docker.
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
	if !slices.Equal(ids, []string{"apt", "gh", "docker"}) {
		t.Errorf("apt group items = %v, want [apt gh docker] (manager row then owned tools)", ids)
	}
	// Every tool is placed exactly once, no lost/duplicated rows.
	var flat []string
	for _, g := range groups {
		for _, item := range g.Items {
			flat = append(flat, item.ID)
		}
	}
	if !slices.Equal(flat, groupOrderForFeed(t)) {
		t.Errorf("GroupByOwner flat = %v, want %v (deterministic canonical group order)", flat, groupOrderForFeed(t))
	}
}

// TestGroupByOwner_CanonicalOrderRoundTrip proves GroupByOwner's canonical
// order matches GroupOrder's board/selector feed: every tool appears exactly
// once, in the same canonical relative order, so the summary list and the
// interactive board/selector never disagree on ordering (spec deterministic
// order rule).
func TestGroupByOwner_CanonicalOrderRoundTrip(t *testing.T) {
	tools := official.AdaptersForPlatform(platform.OSLinux)

	groupOrder := adapterNames(GroupOrder(tools, platform.OSLinux))

	// Flatten GroupByOwner groups into adapter ID order.
	var flat []string
	for _, g := range GroupByOwner(tools, platform.OSLinux) {
		for _, item := range g.Items {
			flat = append(flat, item.ID)
		}
	}
	// GroupByOwner includes each manager's own row at the head of its group;
	// GroupOrder includes manager rows too. They must agree on canonical order.
	if !slices.Equal(groupOrder, flat) {
		t.Errorf("GroupOrder = %v, GroupByOwner flat = %v (must match canonical order)", groupOrder, flat)
	}
}

// TestGroupOrder_DeterministicCanonicalOrder pins GroupOrder (the board and
// selector feed) to the deterministic canonical discovery order produced from
// the canonical adapter feed: manager groups lead in AllAdapters order (each
// manager's own row, then its owned tools in the feed order), then standalone
// tools last. Feeding the canonical platform set makes the order fully
// deterministic regardless of out-of-order concurrent completion.
func TestGroupOrder_DeterministicCanonicalOrder(t *testing.T) {
	tools := official.AdaptersForPlatform(platform.OSLinux)
	ordered := GroupOrder(tools, platform.OSLinux)
	want := groupOrderForFeed(t)
	if got := adapterNames(ordered); !slices.Equal(got, want) {
		t.Errorf("GroupOrder = %v, want %v (deterministic canonical group order)", got, want)
	}
}

// groupOrderForFeed returns the canonical group order the group helpers must
// produce for the given feed on Linux: apt group (apt, gh, docker) leads
// because apt is first in AllAdapters, then brew group (brew), then standalone
// tools in feed order (nvm, npm, pnpm, bun, go, opencode).
func groupOrderForFeed(t *testing.T) []string {
	t.Helper()
	return []string{"apt", "gh", "docker", "brew", "nvm", "npm", "pnpm", "bun", "go", "opencode"}
}

// TestGroupByOwner_CustomToolBucketedUnderManager proves GroupByOwner places a
// custom tool with an injected manager under that manager's group, not into the
// standalone tail (via ownerIDOf's CustomAdapter branch).
func TestGroupByOwner_CustomToolBucketedUnderManager(t *testing.T) {
	brew := official.AdapterByName("brew")
	custom, err := adapters.NewCustomAdapter("mytool", "mytool", "", false, brew)
	if err != nil {
		t.Fatalf("NewCustomAdapter() error: %v", err)
	}
	groups := GroupByOwner([]adapters.Adapter{brew, custom}, platform.OSMacOS)
	if len(groups) != 1 || groups[0].Header != "Homebrew" {
		t.Fatalf("group = %+v, want a single Homebrew group", groups)
	}
	ids := make([]string, len(groups[0].Items))
	for i, item := range groups[0].Items {
		ids[i] = item.ID
	}
	if !slices.Equal(ids, []string{"brew", "mytool"}) {
		t.Errorf("items = %v, want [brew mytool]", ids)
	}
}
