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
