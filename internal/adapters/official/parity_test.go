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
