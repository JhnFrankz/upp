package official

import (
	"testing"

	"github.com/JhnFrankz/upp/internal/platform"
)

// The adapter registry (AllAdapters) and the platform catalog
// (platform.OfficialTools) are two independent registries of official tools.
// They must stay in lockstep: a tool added to one but not the other is a
// silent drift that surfaces as missing tools in `upp list`, wrong
// availability hints, or catalog entries that no adapter can satisfy.

func catalogIndex(t *testing.T) map[string]platform.ToolEntry {
	t.Helper()
	index := make(map[string]platform.ToolEntry, len(platform.OfficialTools))
	for _, entry := range platform.OfficialTools {
		if _, dup := index[entry.ID]; dup {
			t.Fatalf("platform catalog has duplicate tool ID: %s", entry.ID)
		}
		index[entry.ID] = entry
	}
	return index
}

func TestEveryAdapterIsInCatalog(t *testing.T) {
	catalog := catalogIndex(t)
	for _, a := range AllAdapters() {
		id := a.Name()
		if _, ok := catalog[id]; !ok {
			t.Errorf(
				"adapter %q has no catalog entry in platform.OfficialTools; "+
					"add a ToolEntry with ID %q to keep the catalog in sync",
				id, id,
			)
		}
	}
}

func TestEveryCatalogEntryHasAdapter(t *testing.T) {
	adaptersByName := make(map[string]bool, len(AllAdapters()))
	for _, a := range AllAdapters() {
		adaptersByName[a.Name()] = true
	}
	for _, entry := range platform.OfficialTools {
		if !adaptersByName[entry.ID] {
			t.Errorf(
				"catalog tool %q has no adapter registered in AllAdapters(); "+
					"remove the ToolEntry or add the adapter",
				entry.ID,
			)
		}
	}
}

func TestCatalogPlatformsMatchAdapterPlatforms(t *testing.T) {
	catalog := catalogIndex(t)
	for _, a := range AllAdapters() {
		id := a.Name()
		entry, ok := catalog[id]
		if !ok {
			continue // already reported by TestEveryAdapterIsInCatalog
		}
		got := setOf(a.Info().Platforms)
		want := setOf(entry.Platforms)
		if !sameSet(got, want) {
			t.Errorf(
				"platform drift for %q: adapter claims %v, catalog claims %v",
				id, sorted(a.Info().Platforms), sorted(entry.Platforms),
			)
		}
	}
}

// TestCatalogOwnershipMatchesAdapter pins the catalog's display copy of the
// ownership model to the adapter's declared Kind/Manager, so the two
// registries (catalog for display, adapter for canonical behavior) cannot
// silently drift (design: catalog carries a display copy; parity pins both).
func TestCatalogOwnershipMatchesAdapter(t *testing.T) {
	catalog := catalogIndex(t)
	for _, a := range AllAdapters() {
		id := a.Name()
		entry, ok := catalog[id]
		if !ok {
			continue // already reported by TestEveryAdapterIsInCatalog
		}
		info := a.Info()
		if entry.Kind != info.Kind {
			t.Errorf("ownership kind drift for %q: catalog %v, adapter %v", id, entry.Kind, info.Kind)
		}
		if !sameStringMap(entry.Manager, info.Manager) {
			t.Errorf("ownership manager drift for %q: catalog %v, adapter %v", id, entry.Manager, info.Manager)
		}
		if !sameStringMap(entry.ManagerPackage, info.ManagerPackage) {
			t.Errorf(
				"ownership package drift for %q: catalog %v, adapter %v",
				id, entry.ManagerPackage, info.ManagerPackage,
			)
		}
	}
}

// TestCatalogPackageMapping pins the sudo-critical package mapping against the
// catalog mirror (task 1.5 golden table). The mapping is the source of truth
// for what command a manager-group bulk update runs, so the exact package
// names per platform must be pinned here AND in the adapter Info(), and both
// must agree (TestCatalogOwnershipMatchesAdapter). The docker->docker-ce row
// on apt is the highest-risk entry: a typo would run a wrong sudo command.
func TestCatalogPackageMapping(t *testing.T) {
	catalog := catalogIndex(t)
	for _, a := range AllAdapters() {
		id := a.Name()
		entry, ok := catalog[id]
		if !ok {
			continue // already reported by TestEveryAdapterIsInCatalog
		}
		if !sameStringMap(entry.ManagerPackage, a.Info().ManagerPackage) {
			t.Errorf(
				"package map drift for %q: catalog %v, adapter %v",
				id, entry.ManagerPackage, a.Info().ManagerPackage,
			)
		}
	}
}

// TestEveryManagerHasManagerPackage pins the WU1 invariant: every owned
// tool's Manager[p] MUST have a corresponding ManagerPackage[p]. An owned
// tool whose manager is known but whose package is not would make a
// manager-group bulk update run a wrong command (or a sudo command with the
// wrong package name), so the package map must not silently lag the manager
// map. This test references adapters.ToolInfo.ManagerPackage, which does not
// exist until WU1.GREEN adds the field — so it is the RED test for task 1.1.
func TestEveryManagerHasManagerPackage(t *testing.T) {
	for _, a := range AllAdapters() {
		info := a.Info()
		for platform, owner := range info.Manager {
			if pkg := info.ManagerPackage[platform]; pkg == "" {
				t.Errorf(
					"tool %q is owned by %q on %q but declares no ManagerPackage[%q]; "+
						"add the per-platform package name so an owned tool can be bulk-updated",
					info.ID, owner, platform, platform,
				)
			}
		}
		for platform := range info.ManagerPackage {
			if info.Manager[platform] == "" {
				t.Errorf(
					"tool %q declares ManagerPackage[%q]=%q but has no owner on %q; "+
						"a package must never exist without its owner",
					info.ID, platform, info.ManagerPackage[platform], platform,
				)
			}
		}
	}
}

func sameStringMap(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

func TestCatalogNamesMatchAdapterNames(t *testing.T) {
	catalog := catalogIndex(t)
	for _, a := range AllAdapters() {
		id := a.Name()
		entry, ok := catalog[id]
		if !ok {
			continue // already reported by TestEveryAdapterIsInCatalog
		}
		if entry.Name != a.Info().Name {
			t.Errorf(
				"display-name drift for %q: adapter says %q, catalog says %q",
				id, a.Info().Name, entry.Name,
			)
		}
	}
}

// setOf and sameSet keep the assertions order-independent so reordering
// platform lists is not reported as drift.

func setOf(items []string) map[string]bool {
	set := make(map[string]bool, len(items))
	for _, item := range items {
		set[item] = true
	}
	return set
}

func sameSet(a, b map[string]bool) bool {
	if len(a) != len(b) {
		return false
	}
	for key := range a {
		if !b[key] {
			return false
		}
	}
	return true
}

func sorted(items []string) []string {
	out := make([]string, len(items))
	copy(out, items)
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j] < out[j-1]; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

// TestParseWingetUpgradeOutput drives the pure, fail-closed helper that scans
// `winget upgrade` (no args) output for the winget self row. It records the
// self row Id as Microsoft.AppInstaller and returns (current, latest, found);
// unparseable or absent rows yield found=false (task 2.1).
func TestParseWingetUpgradeOutput(t *testing.T) {
	tests := []struct {
		name      string
		out       string
		wantCur   string
		wantLat   string
		wantFound bool
	}{
		{
			name:      "own-row-lead-v-4-part",
			out:       "Name  Id  Version  Available  Source\n------\nwinget  Microsoft.AppInstaller  v1.8.2301  v1.8.2311  winget\n",
			wantCur:   "v1.8.2301",
			wantLat:   "v1.8.2311",
			wantFound: true,
		},
		{
			name:      "own-row-plain-versions",
			out:       "winget  Microsoft.AppInstaller  1.8.2301  1.8.2311  winget\n",
			wantCur:   "1.8.2301",
			wantLat:   "1.8.2311",
			wantFound: true,
		},
		{
			name:      "no-own-row",
			out:       "Name  Id  Version  Available  Source\n------\nfoo  Baz.Corp.App  1.0.0  2.0.0  winget\n",
			wantCur:   "",
			wantLat:   "",
			wantFound: false,
		},
		{
			name:      "empty-output",
			out:       "",
			wantCur:   "",
			wantLat:   "",
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotCur, gotLat, gotFound := parseWingetUpgradeOutput(tt.out)
			if gotCur != tt.wantCur {
				t.Errorf("current = %q, want %q", gotCur, tt.wantCur)
			}
			if gotLat != tt.wantLat {
				t.Errorf("latest = %q, want %q", gotLat, tt.wantLat)
			}
			if gotFound != tt.wantFound {
				t.Errorf("found = %v, want %v", gotFound, tt.wantFound)
			}
		})
	}
}

// TestParseScoopStatusOutput drives the pure, fail-closed helper that scans
// `scoop status` output for the scoop self row (task 3.1). It returns
// (current, latest, found); an absent or unparseable scoop row yields
// found=false (current-only fallback when the output shape is unstable). A
// leading WARN line (scoop prints "Scoop is out of date" to stderr) is
// tolerated — the table is still parsed. The leading-v on a version is
// tolerated since scoop versions genuinely carry the leading v, matching the
// winget parity conventions.
func TestParseScoopStatusOutput(t *testing.T) {
	tests := []struct {
		name      string
		out       string
		wantCur   string
		wantLat   string
		wantFound bool
	}{
		{
			name:      "own-row",
			out:       "Name       Installed  Latest\n----       ---------  ------\nscoop      1.0.0      1.2.0\n",
			wantCur:   "1.0.0",
			wantLat:   "1.2.0",
			wantFound: true,
		},
		{
			name:      "own-row-lead-v",
			out:       "scoop  v0.22.0  v0.23.0\n",
			wantCur:   "v0.22.0",
			wantLat:   "v0.23.0",
			wantFound: true,
		},
		{
			name:      "warn-stderr-tolerated",
			out:       "WARN  Scoop is out of date. Run 'scoop update' to get the latest changes.\n\nName       Installed  Latest\n----       ---------  ------\nscoop      0.3.0      0.4.0\n",
			wantCur:   "0.3.0",
			wantLat:   "0.4.0",
			wantFound: true,
		},
		{
			name:      "no-own-row-fallback-current-only",
			out:       "Name       Installed  Latest\n----       ---------  ------\nfoo        1.0.0      1.2.0\n",
			wantCur:   "",
			wantLat:   "",
			wantFound: false,
		},
		{
			name:      "empty-output",
			out:       "",
			wantCur:   "",
			wantLat:   "",
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotCur, gotLat, gotFound := parseScoopStatusOutput(tt.out)
			if gotCur != tt.wantCur {
				t.Errorf("current = %q, want %q", gotCur, tt.wantCur)
			}
			if gotLat != tt.wantLat {
				t.Errorf("latest = %q, want %q", gotLat, tt.wantLat)
			}
			if gotFound != tt.wantFound {
				t.Errorf("found = %v, want %v", gotFound, tt.wantFound)
			}
		})
	}
}
