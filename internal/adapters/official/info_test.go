package official

import (
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/JhnFrankz/upp/internal/adapters"
)

// infoCase is one table row for Info(): golden static metadata per adapter.
type infoCase struct {
	name    string
	newAdpt func() adapters.Adapter
	want    adapters.ToolInfo
}

// TestInfo verifies Info() for all 12 official adapters: golden
// ID/Name/Platforms/Trust/UpdatePolicy/Kind/Manager, plus the manager
// command declarations (SelfUpdateCommand/PackageUpdateCommand, pacman
// Privileges) and Name() consistency
// with the adapter ID. UpdatePolicy is declared explicitly at every Info()
// site (design D6, spec Update Gating) — the golden value pins the declared
// policy per adapter. Kind distinguishes manager adapters (apt/brew/winget/
// scoop) from owned/standalone tools; Manager is a platform→owner map that
// is nil for standalone tools (spec Tool Ownership Declaration). No exec
// seam involved — pure static metadata.
func TestInfo(t *testing.T) {
	tests := []infoCase{
		{"apt", func() adapters.Adapter { return &AptAdapter{} }, adapters.ToolInfo{ID: "apt", Name: "APT Package Manager", Platforms: []string{"linux"}, Trust: adapters.TrustOfficial, UpdatePolicy: adapters.PolicyGated, Kind: adapters.KindManager, SelfUpdateCommand: aptSelfUpdateCmd, PackageUpdateCommand: aptPackageUpdateTemplate}},
		{"brew", func() adapters.Adapter { return &BrewAdapter{} }, adapters.ToolInfo{ID: "brew", Name: "Homebrew", Platforms: []string{"linux", "macos"}, Trust: adapters.TrustOfficial, UpdatePolicy: adapters.PolicyAlwaysUpdate, Kind: adapters.KindManager, SelfUpdateCommand: brewSelfUpdateCmd, PackageUpdateCommand: brewPackageUpdateTemplate}},
		{"pacman", func() adapters.Adapter { return &PacmanAdapter{} }, adapters.ToolInfo{ID: "pacman", Name: "Pacman Package Manager", Platforms: []string{"linux"}, Trust: adapters.TrustOfficial, UpdatePolicy: adapters.PolicyGated, Kind: adapters.KindManager, Privileges: []string{"sudo"}, SelfUpdateCommand: pacmanSelfUpdateCmd, PackageUpdateCommand: pacmanPackageUpdateTemplate}},
		{"npm", func() adapters.Adapter { return &NpmAdapter{} }, adapters.ToolInfo{ID: "npm", Name: "npm", Platforms: []string{"linux", "macos", "windows"}, Trust: adapters.TrustOfficial, UpdatePolicy: adapters.PolicyGated, Kind: adapters.KindTool, Command: "npm update -g"}},
		{"pnpm", func() adapters.Adapter { return &PnpmAdapter{} }, adapters.ToolInfo{ID: "pnpm", Name: "pnpm", Platforms: []string{"linux", "macos", "windows"}, Trust: adapters.TrustOfficial, UpdatePolicy: adapters.PolicyGated, Kind: adapters.KindTool, Command: "pnpm update -g"}},
		{"bun", func() adapters.Adapter { return &BunAdapter{} }, adapters.ToolInfo{ID: "bun", Name: "Bun", Platforms: []string{"linux", "macos", "windows"}, Trust: adapters.TrustOfficial, UpdatePolicy: adapters.PolicyAlwaysUpdate, Kind: adapters.KindTool, Command: "bun upgrade"}},
		{"gh", func() adapters.Adapter { return &GhAdapter{} }, adapters.ToolInfo{ID: "gh", Name: "GitHub CLI", Platforms: []string{"linux", "macos", "windows"}, Trust: adapters.TrustOfficial, UpdatePolicy: adapters.PolicyAlwaysUpdate, Kind: adapters.KindTool, Manager: map[string]string{"linux": "apt", "macos": "brew", "windows": "winget"}, ManagerPackage: map[string]string{"linux": "gh", "macos": "gh", "windows": "gh"}}},
		{"docker", func() adapters.Adapter { return &DockerAdapter{} }, adapters.ToolInfo{ID: "docker", Name: "Docker", Platforms: []string{"linux", "macos", "windows"}, Trust: adapters.TrustOfficial, UpdatePolicy: adapters.PolicyAlwaysUpdate, Kind: adapters.KindTool, Manager: map[string]string{"linux": "apt", "macos": "brew", "windows": "winget"}, ManagerPackage: map[string]string{"linux": "docker-ce", "macos": "docker", "windows": "Docker.Docker"}}},
		{"go", func() adapters.Adapter { return &GoAdapter{} }, adapters.ToolInfo{ID: "go", Name: "Go", Platforms: []string{"linux", "macos", "windows"}, Trust: adapters.TrustOfficial, UpdatePolicy: adapters.PolicyAlwaysUpdate, Kind: adapters.KindTool, Manager: map[string]string{"macos": "brew", "windows": "winget"}, ManagerPackage: map[string]string{"macos": "golang", "windows": "GoLang.Go"}, Command: "curl -fsSL " + goTarballURL(runtime.GOARCH) + " | sudo tar -C /usr/local -xzf -"}},
		{"opencode", func() adapters.Adapter { return &OpenCodeAdapter{} }, adapters.ToolInfo{ID: "opencode", Name: "OpenCode", Platforms: []string{"linux", "macos", "windows"}, Trust: adapters.TrustOfficial, UpdatePolicy: adapters.PolicyAlwaysUpdate, Kind: adapters.KindTool, Command: "curl -fsSL https://opencode.ai/install | bash"}},
		{"winget", func() adapters.Adapter { return &WingetAdapter{} }, adapters.ToolInfo{ID: "winget", Name: "Windows Package Manager", Platforms: []string{"windows"}, Trust: adapters.TrustOfficial, UpdatePolicy: adapters.PolicyAlwaysUpdate, Kind: adapters.KindManager, SelfUpdateCommand: wingetSelfUpdateCmd, PackageUpdateCommand: wingetPackageUpdateTemplate}},
		{"scoop", func() adapters.Adapter { return &ScoopAdapter{} }, adapters.ToolInfo{ID: "scoop", Name: "Scoop", Platforms: []string{"windows"}, Trust: adapters.TrustOfficial, UpdatePolicy: adapters.PolicyAlwaysUpdate, Kind: adapters.KindManager, SelfUpdateCommand: scoopSelfUpdateCmd}},
		{"nvm", func() adapters.Adapter { return &NVMAdapter{} }, adapters.ToolInfo{ID: "nvm", Name: "Node Version Manager", Platforms: []string{"linux", "macos", "windows"}, Trust: adapters.TrustOfficial, UpdatePolicy: adapters.PolicyGated, Kind: adapters.KindTool, Command: "bash -c 'source \"${NVM_DIR:-$HOME/.nvm}/nvm.sh\" >/dev/null 2>&1 && nvm install stable'"}},
		{"uv", func() adapters.Adapter { return &UvAdapter{} }, adapters.ToolInfo{ID: "uv", Name: "uv", Platforms: []string{"linux", "macos", "windows"}, Trust: adapters.TrustOfficial, UpdatePolicy: adapters.PolicyGated, Kind: adapters.KindTool, Command: "uv self update"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := tt.newAdpt()
			info := a.Info()
			if !reflect.DeepEqual(info, tt.want) {
				t.Errorf("Info() = %+v, want %+v", info, tt.want)
			}
			if name := a.Name(); name != tt.want.ID {
				t.Errorf("Name() = %q, want %q", name, tt.want.ID)
			}
		})
	}
}

// TestManagerCommandDeclarations is the declaration sweep (spec tool-adapter
// "Adapter Interface", design D1/D2): every KindManager adapter in the
// registry MUST declare a non-empty SelfUpdateCommand — the exact command its
// Update() executes; package managers (apt/brew/winget/pacman) MUST also
// declare the per-package PackageUpdateCommand template carrying
// PackagePlaceholder; scoop is self-only and MUST NOT declare one. Non-manager
// adapters MUST NOT declare the manager-only surface. pacman MUST declare
// Privileges ["sudo"] (spec security-model "Pacman privileged update prompts").
func TestManagerCommandDeclarations(t *testing.T) {
	// Package managers and their pinned per-package command templates.
	packageManagers := map[string]string{
		"apt":    "sudo apt install --only-upgrade <pkg>",
		"brew":   "brew upgrade <pkg>",
		"winget": "winget upgrade <pkg>",
		"pacman": "sudo pacman -S --noconfirm <pkg>",
	}

	for _, a := range AllAdapters() {
		info := a.Info()
		if info.Kind != adapters.KindManager {
			if info.SelfUpdateCommand != "" || info.PackageUpdateCommand != "" {
				t.Errorf("%s: KindTool adapter declares manager-only command fields (self=%q pkg=%q)",
					info.ID, info.SelfUpdateCommand, info.PackageUpdateCommand)
			}
			continue
		}

		if info.SelfUpdateCommand == "" {
			t.Errorf("%s: KindManager adapter declares empty SelfUpdateCommand", info.ID)
		}

		wantPkg, isPackageManager := packageManagers[info.ID]
		if isPackageManager {
			if !strings.Contains(info.PackageUpdateCommand, adapters.PackagePlaceholder) {
				t.Errorf("%s: PackageUpdateCommand = %q, want it to contain PackagePlaceholder %q",
					info.ID, info.PackageUpdateCommand, adapters.PackagePlaceholder)
			}
			if info.PackageUpdateCommand != wantPkg {
				t.Errorf("%s: PackageUpdateCommand = %q, want %q", info.ID, info.PackageUpdateCommand, wantPkg)
			}
		} else if info.PackageUpdateCommand != "" {
			t.Errorf("%s: self-only manager declares PackageUpdateCommand = %q, want empty",
				info.ID, info.PackageUpdateCommand)
		}
	}

	pacmanInfo := (&PacmanAdapter{}).Info()
	if !equalPrivileges(pacmanInfo.Privileges, []string{"sudo"}) {
		t.Errorf("pacman Privileges = %v, want [sudo]", pacmanInfo.Privileges)
	}
}
