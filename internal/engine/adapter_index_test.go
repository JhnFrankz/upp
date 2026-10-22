package engine

import (
	"testing"

	"github.com/JhnFrankz/upp/internal/adapters"
)

func TestBuildAdapterIndex(t *testing.T) {
	t.Run("empty slice", func(t *testing.T) {
		idx := buildAdapterIndex(nil)
		if len(idx) != 0 {
			t.Errorf("expected empty index, got len=%d", len(idx))
		}
	})

	t.Run("nil adapter ignored", func(t *testing.T) {
		idx := buildAdapterIndex([]adapters.Adapter{nil})
		if len(idx) != 0 {
			t.Errorf("expected empty index, got len=%d", len(idx))
		}
	})

	t.Run("indexes id, name, and their lowercase variants", func(t *testing.T) {
		a := &mockPlanAdapter{
			info: adapters.ToolInfo{
				ID:   "ToolID",
				Name: "ToolName",
			},
		}
		idx := buildAdapterIndex([]adapters.Adapter{a})

		for _, key := range []string{"ToolID", "toolid", "ToolName", "toolname"} {
			if got := idx.find(key, ""); got != a {
				t.Errorf("idx.find(%q, \"\") = %v, want %v", key, got, a)
			}
		}
	})

	t.Run("identical id and name handled without duplicate work", func(t *testing.T) {
		a := &mockPlanAdapter{
			info: adapters.ToolInfo{
				ID:   "tool",
				Name: "tool",
			},
		}
		idx := buildAdapterIndex([]adapters.Adapter{a})

		if got := idx.find("tool", ""); got != a {
			t.Errorf("idx.find(\"tool\", \"\") = %v, want %v", got, a)
		}
		if len(idx) != 1 {
			t.Errorf("expected 1 entry in index, got %d", len(idx))
		}
	})

	t.Run("empty id and empty name", func(t *testing.T) {
		a := &mockPlanAdapter{
			info: adapters.ToolInfo{
				ID:   "",
				Name: "",
			},
		}
		idx := buildAdapterIndex([]adapters.Adapter{a})
		if len(idx) != 0 {
			t.Errorf("expected 0 entries in index, got %d", len(idx))
		}
	})

	t.Run("overwrite false preserves existing entry", func(t *testing.T) {
		a1 := &mockPlanAdapter{info: adapters.ToolInfo{ID: "tool", Name: "first"}}
		a2 := &mockPlanAdapter{info: adapters.ToolInfo{ID: "tool", Name: "second"}}

		idx := make(adapterIndex)
		indexAdapter(idx, a1, false)
		indexAdapter(idx, a2, false)

		if got := idx["tool"]; got != a1 {
			t.Errorf("idx[\"tool\"] = %v, want a1 %v", got, a1)
		}
	})

	t.Run("overwrite true overwrites existing entry", func(t *testing.T) {
		a1 := &mockPlanAdapter{info: adapters.ToolInfo{ID: "tool", Name: "first"}}
		a2 := &mockPlanAdapter{info: adapters.ToolInfo{ID: "tool", Name: "second"}}

		idx := make(adapterIndex)
		indexAdapter(idx, a1, true)
		indexAdapter(idx, a2, true)

		if got := idx["tool"]; got != a2 {
			t.Errorf("idx[\"tool\"] = %v, want a2 %v", got, a2)
		}
	})
}
