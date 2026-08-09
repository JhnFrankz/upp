package platform

// ToolEntry describes a tool in the platform catalog.
type ToolEntry struct {
	ID        string
	Name      string
	Platforms []string
}

// OfficialTools is the complete registry of built-in tools.
// Each tool lists the platforms it supports.
var OfficialTools = []ToolEntry{
	{ID: "apt", Name: "APT Package Manager", Platforms: []string{OSLinux}},
	{ID: "brew", Name: "Homebrew", Platforms: []string{OSLinux, OSMacOS}},
	{ID: "winget", Name: "Windows Package Manager", Platforms: []string{OSWindows}},
	{ID: "scoop", Name: "Scoop", Platforms: []string{OSWindows}},
	{ID: "nvm", Name: "Node Version Manager", Platforms: []string{OSLinux, OSMacOS, OSWindows}},
	{ID: "npm", Name: "npm", Platforms: []string{OSLinux, OSMacOS, OSWindows}},
	{ID: "pnpm", Name: "pnpm", Platforms: []string{OSLinux, OSMacOS, OSWindows}},
	{ID: "bun", Name: "Bun", Platforms: []string{OSLinux, OSMacOS, OSWindows}},
	{ID: "gh", Name: "GitHub CLI", Platforms: []string{OSLinux, OSMacOS, OSWindows}},
	{ID: "docker", Name: "Docker", Platforms: []string{OSLinux, OSMacOS, OSWindows}},
	{ID: "go", Name: "Go", Platforms: []string{OSLinux, OSMacOS, OSWindows}},
	{ID: "opencode", Name: "OpenCode", Platforms: []string{OSLinux, OSMacOS, OSWindows}},
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


