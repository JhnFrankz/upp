package cli

import (
	"fmt"

	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/config"
	"github.com/JhnFrankz/upp/internal/engine"
	"github.com/JhnFrankz/upp/internal/output"
)

// outcomeToToolResult converts an engine.CheckOutcome to an output.ToolResult.
func outcomeToToolResult(oc engine.CheckOutcome) output.ToolResult {
	name := oc.ToolName
	if name == "" {
		name = oc.ToolID
	}
	res := output.ToolResult{
		Name: name,
	}
	switch oc.Status {
	case engine.StatusAvailable:
		res.Status = output.StatusAvailable
		res.Version = fmt.Sprintf("%s → %s", oc.CurrentVersion, oc.LatestVersion)
	case engine.StatusCurrent:
		res.Status = output.StatusCurrent
		res.Version = oc.CurrentVersion
	case engine.StatusSkipped, engine.StatusUnknown:
		res.Status = output.StatusSkipped
	case engine.StatusFailed:
		res.Status = output.StatusFailed
		res.Error = oc.Err
		res.Stderr = oc.Stderr
	default:
		res.Status = output.StatusSkipped
	}
	return res
}

func adapterIDs(adapterList []adapters.Adapter) []string {
	var ids []string
	for _, a := range adapterList {
		ids = append(ids, a.Name())
	}
	return ids
}

// toolSelectionID returns the canonical stable identity of an adapter used as
// the selector option ID and the adapter-lookup key (design D1). It is
// adapters.ToolInfo.ID, falling back to Adapter.Name() only when the ID is
// empty. It matches engine.CheckOutcome.ToolID, so selection, planning, and
// execution all key on the same identity — the display label is never a key.
func toolSelectionID(a adapters.Adapter) string {
	if id := a.Info().ID; id != "" {
		return id
	}
	return a.Name()
}

// adapterByID indexes adapters by their canonical selection identity so the
// interactive selector's option IDs resolve back to exactly one adapter
// (spec Canonical Tool Selection Identity). adapterIDs above keeps using
// Adapter.Name() for the FilterTools warning contract.
func adapterByID(adapterList []adapters.Adapter) map[string]adapters.Adapter {
	m := make(map[string]adapters.Adapter, len(adapterList))
	for _, a := range adapterList {
		m[toolSelectionID(a)] = a
	}
	return m
}

// buildAdapterList creates adapters for enabled tools from the config.
//
// NOTE: this is the CLI-layer entry point used by the adapter-list integration
// tests (TestBuildAdapterList_*). The update and list commands currently build
// their adapter list inline (update.go, list.go) with the same engine calls, so
// no production path reaches this function yet — see the open question in
// odd/tasks/delete-dead-cli-shim.md.
func buildAdapterList(cfg *config.Config, osName string) []adapters.Adapter {
	eng := engine.New(cfg, osName)
	res, _ := eng.Resolve(engine.Filter{})
	return res
}
