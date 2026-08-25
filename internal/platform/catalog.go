package platform

import (
	"github.com/JhnFrankz/upp/internal/adapters"
)

// ToolEntry describes a tool in the platform catalog.
// Kind and Manager carry a display copy of the adapter's declared ownership
// model (design: ToolInfo is canonical, the catalog mirrors it for display;
// parity_test pins both). Manager is a platform -> owning manager ID map,
// nil for standalone tools.
type ToolEntry struct {
	ID        string
	Name      string
	Platforms []string
	Kind      adapters.Kind
	Manager   map[string]string
}

// OfficialTools is the complete registry of built-in tools.
// Each tool lists the platforms it supports and its ownership declaration.
var OfficialTools = []ToolEntry{
	{ID: "apt", Name: "APT Package Manager", Platforms: []string{OSLinux}, Kind: adapters.KindManager},
	{ID: "brew", Name: "Homebrew", Platforms: []string{OSLinux, OSMacOS}, Kind: adapters.KindManager},
	{ID: "winget", Name: "Windows Package Manager", Platforms: []string{OSWindows}, Kind: adapters.KindManager},
	{ID: "scoop", Name: "Scoop", Platforms: []string{OSWindows}, Kind: adapters.KindManager},
	{ID: "nvm", Name: "Node Version Manager", Platforms: []string{OSLinux, OSMacOS, OSWindows}, Kind: adapters.KindTool},
	{ID: "npm", Name: "npm", Platforms: []string{OSLinux, OSMacOS, OSWindows}, Kind: adapters.KindTool},
	{ID: "pnpm", Name: "pnpm", Platforms: []string{OSLinux, OSMacOS, OSWindows}, Kind: adapters.KindTool},
	{ID: "bun", Name: "Bun", Platforms: []string{OSLinux, OSMacOS, OSWindows}, Kind: adapters.KindTool},
	{ID: "gh", Name: "GitHub CLI", Platforms: []string{OSLinux, OSMacOS, OSWindows}, Kind: adapters.KindTool, Manager: map[string]string{OSLinux: "apt", OSMacOS: "brew", OSWindows: "winget"}},
	{ID: "docker", Name: "Docker", Platforms: []string{OSLinux, OSMacOS, OSWindows}, Kind: adapters.KindTool, Manager: map[string]string{OSLinux: "apt", OSMacOS: "brew", OSWindows: "winget"}},
	{ID: "go", Name: "Go", Platforms: []string{OSLinux, OSMacOS, OSWindows}, Kind: adapters.KindTool, Manager: map[string]string{OSMacOS: "brew", OSWindows: "winget"}},
	{ID: "opencode", Name: "OpenCode", Platforms: []string{OSLinux, OSMacOS, OSWindows}, Kind: adapters.KindTool},
}

// CatalogFor returns the list of official tools available on the given OS.
func CatalogFor(os string) []ToolEntry {
	var catalog []ToolEntry
	for _, tool := range OfficialTools {
		for _, p := range tool.Platforms {
			if p == os {
				catalog = append(catalog, tool)
				break
			}
		}
	}
	return catalog
}

// IsOfficialTool returns true if the given tool ID is in the official catalog.
func IsOfficialTool(id string) bool {
	for _, tool := range OfficialTools {
		if tool.ID == id {
			return true
		}
	}
	return false
}

// FilterByPlatform returns adapter tool IDs that match the given platform.
func FilterByPlatform(os string) []string {
	var ids []string
	for _, tool := range CatalogFor(os) {
		ids = append(ids, tool.ID)
	}
	return ids
}
