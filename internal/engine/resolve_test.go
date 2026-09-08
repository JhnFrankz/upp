package engine

import (
	"reflect"
	"testing"

	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/adapters/official"
	"github.com/JhnFrankz/upp/internal/config"
	"github.com/JhnFrankz/upp/internal/platform"
)

func TestResolve_InjectedAdaptersOverride(t *testing.T) {
	cfg := &config.Config{}
	fakes := []adapters.Adapter{
		&dummyAdapter{name: "fake1"},
		&dummyAdapter{name: "fake2"},
	}

	eng := New(cfg, "linux", WithAdapters(fakes))

	t.Run("empty filter returns injected adapters directly", func(t *testing.T) {
		resolved, err := eng.Resolve(Filter{})
		if err != nil {
			t.Fatalf("Resolve(Filter{}) error: %v", err)
		}
		if len(resolved) != len(fakes) {
			t.Fatalf("len(resolved) = %d, want %d", len(resolved), len(fakes))
		}
		for i, a := range resolved {
			if a.Name() != fakes[i].Name() {
				t.Errorf("resolved[%d].Name() = %q, want %q", i, a.Name(), fakes[i].Name())
			}
		}
	})

	t.Run("filter applied to injected adapters", func(t *testing.T) {
		resolved, err := eng.Resolve(Filter{Only: []string{"fake1"}})
		if err != nil {
			t.Fatalf("Resolve(Filter{Only}) error: %v", err)
		}
		if len(resolved) != 1 || resolved[0].Name() != "fake1" {
			t.Fatalf("resolved = %v, want only fake1", resolved)
		}
	})
}

func TestResolve_PlatformOfficialAdapters(t *testing.T) {
	cfg := &config.Config{}

	t.Run("linux official adapters", func(t *testing.T) {
		eng := New(cfg, "linux")
		resolved, err := eng.Resolve(Filter{})
		if err != nil {
			t.Fatalf("Resolve(Filter{}) error: %v", err)
		}

		expected := official.AdaptersForPlatform(platform.OSLinux)
		if len(resolved) != len(expected) {
			t.Fatalf("len(resolved) = %d, want %d", len(resolved), len(expected))
		}
		for i, a := range resolved {
			if a.Name() != expected[i].Name() {
				t.Errorf("resolved[%d].Name() = %q, want %q", i, a.Name(), expected[i].Name())
			}
		}

		// Ensure key Linux managers and tools are present
		found := make(map[string]bool)
		for _, a := range resolved {
			found[a.Name()] = true
		}
		if !found["apt"] {
			t.Errorf("expected apt in linux adapters")
		}
		if !found["pacman"] {
			t.Errorf("expected pacman in linux adapters")
		}
		if !found["gh"] {
			t.Errorf("expected gh in linux adapters")
		}
		if found["winget"] {
			t.Errorf("winget should not be in linux adapters")
		}
		if found["scoop"] {
			t.Errorf("scoop should not be in linux adapters")
		}
	})

	t.Run("darwin official adapters", func(t *testing.T) {
		eng := New(cfg, "darwin")
		resolved, err := eng.Resolve(Filter{})
		if err != nil {
			t.Fatalf("Resolve(Filter{}) error: %v", err)
		}

		expected := official.AdaptersForPlatform(platform.OSMacOS)
		if len(resolved) != len(expected) {
			t.Fatalf("len(resolved) = %d, want %d", len(resolved), len(expected))
		}
		for i, a := range resolved {
			if a.Name() != expected[i].Name() {
				t.Errorf("resolved[%d].Name() = %q, want %q", i, a.Name(), expected[i].Name())
			}
		}

		found := make(map[string]bool)
		for _, a := range resolved {
			found[a.Name()] = true
		}
		if !found["brew"] {
			t.Errorf("expected brew in darwin adapters")
		}
		if found["apt"] {
			t.Errorf("apt should not be in darwin adapters")
		}
		if found["winget"] {
			t.Errorf("winget should not be in darwin adapters")
		}
	})

	t.Run("windows official adapters", func(t *testing.T) {
		eng := New(cfg, "windows")
		resolved, err := eng.Resolve(Filter{})
		if err != nil {
			t.Fatalf("Resolve(Filter{}) error: %v", err)
		}

		expected := official.AdaptersForPlatform(platform.OSWindows)
		if len(resolved) != len(expected) {
			t.Fatalf("len(resolved) = %d, want %d", len(resolved), len(expected))
		}
		for i, a := range resolved {
			if a.Name() != expected[i].Name() {
				t.Errorf("resolved[%d].Name() = %q, want %q", i, a.Name(), expected[i].Name())
			}
		}

		found := make(map[string]bool)
		for _, a := range resolved {
			found[a.Name()] = true
		}
		if !found["winget"] {
			t.Errorf("expected winget in windows adapters")
		}
		if !found["scoop"] {
			t.Errorf("expected scoop in windows adapters")
		}
		if found["apt"] {
			t.Errorf("apt should not be in windows adapters")
		}
		if found["brew"] {
			t.Errorf("brew should not be in windows adapters")
		}
	})
}

func TestResolve_DisabledToolsExcluded(t *testing.T) {
	cfg := &config.Config{
		Tools: map[string]config.ToolConfig{
			"gh": {Enabled: false},
		},
	}
	eng := New(cfg, "linux")
	resolved, err := eng.Resolve(Filter{})
	if err != nil {
		t.Fatalf("Resolve(Filter{}) error: %v", err)
	}

	for _, a := range resolved {
		if a.Name() == "gh" {
			t.Errorf("tool gh was disabled in config but still present in resolved list")
		}
	}
}

func TestResolve_CustomToolManagerBinding(t *testing.T) {
	cfg := &config.Config{
		Custom: map[string]config.CustomTool{
			"custom-apt": {
				Command:  "echo update",
				CheckCmd: "echo 1.0",
				Manager:  "apt",
				Trusted:  true,
			},
			"custom-invalid": {
				Command: "echo update",
				Manager: "invalid",
			},
			"custom-gh": {
				Command: "echo update",
				Manager: "gh", // gh is KindTool, not KindManager
			},
			"custom-empty-cmd": {
				Command: "", // invalid custom command, should be skipped safely
				Manager: "apt",
			},
		},
	}

	eng := New(cfg, "linux")
	resolved, err := eng.Resolve(Filter{})
	if err != nil {
		t.Fatalf("Resolve(Filter{}) error: %v", err)
	}

	byName := make(map[string]adapters.Adapter)
	for _, a := range resolved {
		byName[a.Name()] = a
	}

	if _, exists := byName["custom-empty-cmd"]; exists {
		t.Errorf("custom-empty-cmd with empty command should have been skipped safely")
	}

	t.Run("custom tool with valid KindManager binds manager adapter", func(t *testing.T) {
		a, exists := byName["custom-apt"]
		if !exists {
			t.Fatalf("custom-apt not found in resolved adapters")
		}
		ca, ok := a.(*adapters.CustomAdapter)
		if !ok {
			t.Fatalf("custom-apt is not *adapters.CustomAdapter: %T", a)
		}
		mgr := ca.ManagerAdapter()
		if mgr == nil {
			t.Fatalf("custom-apt ManagerAdapter() is nil, want apt manager")
		}
		if mgr.Name() != "apt" {
			t.Errorf("custom-apt ManagerAdapter().Name() = %q, want apt", mgr.Name())
		}
	})

	t.Run("custom tool with invalid manager falls back to standalone", func(t *testing.T) {
		a, exists := byName["custom-invalid"]
		if !exists {
			t.Fatalf("custom-invalid not found in resolved adapters")
		}
		ca, ok := a.(*adapters.CustomAdapter)
		if !ok {
			t.Fatalf("custom-invalid is not *adapters.CustomAdapter: %T", a)
		}
		if ca.ManagerAdapter() != nil {
			t.Errorf("custom-invalid ManagerAdapter() = %v, want nil", ca.ManagerAdapter())
		}
	})

	t.Run("custom tool with KindTool manager falls back to standalone", func(t *testing.T) {
		a, exists := byName["custom-gh"]
		if !exists {
			t.Fatalf("custom-gh not found in resolved adapters")
		}
		ca, ok := a.(*adapters.CustomAdapter)
		if !ok {
			t.Fatalf("custom-gh is not *adapters.CustomAdapter: %T", a)
		}
		if ca.ManagerAdapter() != nil {
			t.Errorf("custom-gh ManagerAdapter() = %v, want nil (gh is KindTool)", ca.ManagerAdapter())
		}
	})
}

func TestResolve_FilterOnly(t *testing.T) {
	cfg := &config.Config{}
	eng := New(cfg, "darwin")

	t.Run("filter exact subset", func(t *testing.T) {
		resolved, err := eng.Resolve(Filter{Only: []string{"brew", "npm"}})
		if err != nil {
			t.Fatalf("Resolve error: %v", err)
		}
		if len(resolved) != 2 {
			t.Fatalf("len(resolved) = %d, want 2", len(resolved))
		}
		names := []string{resolved[0].Name(), resolved[1].Name()}
		if !reflect.DeepEqual(names, []string{"brew", "npm"}) {
			t.Errorf("resolved names = %v, want [brew, npm]", names)
		}
	})

	t.Run("filter case-insensitive match", func(t *testing.T) {
		resolved, err := eng.Resolve(Filter{Only: []string{"BREW"}})
		if err != nil {
			t.Fatalf("Resolve error: %v", err)
		}
		if len(resolved) != 1 || resolved[0].Name() != "brew" {
			t.Fatalf("resolved = %v, want only brew", resolved)
		}
	})

	t.Run("filter unknown returns empty slice without error", func(t *testing.T) {
		resolved, err := eng.Resolve(Filter{Only: []string{"unknown"}})
		if err != nil {
			t.Fatalf("Resolve error: %v", err)
		}
		if len(resolved) != 0 {
			t.Fatalf("len(resolved) = %d, want 0", len(resolved))
		}
	})
}

func TestResolvingOwner_Helpers(t *testing.T) {
	aptAdapter := official.AdapterByName("apt")
	brewAdapter := official.AdapterByName("brew")
	wingetAdapter := official.AdapterByName("winget")
	ghAdapter := official.AdapterByName("gh")
	dockerAdapter := official.AdapterByName("docker")
	goAdapter := official.AdapterByName("go")
	npmAdapter := official.AdapterByName("npm")

	t.Run("ResolvingOwner official adapters across platforms", func(t *testing.T) {
		// gh on linux -> apt
		if owner := ResolvingOwner(ghAdapter, "linux"); owner == nil || owner.Name() != "apt" {
			t.Errorf("ResolvingOwner(gh, linux) = %v, want apt", owner)
		}
		// gh on darwin / macos -> brew
		if owner := ResolvingOwner(ghAdapter, "darwin"); owner == nil || owner.Name() != "brew" {
			t.Errorf("ResolvingOwner(gh, darwin) = %v, want brew", owner)
		}
		if owner := ResolvingOwner(ghAdapter, "macos"); owner == nil || owner.Name() != "brew" {
			t.Errorf("ResolvingOwner(gh, macos) = %v, want brew", owner)
		}
		// gh on windows -> winget
		if owner := ResolvingOwner(ghAdapter, "windows"); owner == nil || owner.Name() != "winget" {
			t.Errorf("ResolvingOwner(gh, windows) = %v, want winget", owner)
		}
		// go on linux -> nil (standalone)
		if owner := ResolvingOwner(goAdapter, "linux"); owner != nil {
			t.Errorf("ResolvingOwner(go, linux) = %v, want nil", owner)
		}
		// npm on linux -> nil (standalone)
		if owner := ResolvingOwner(npmAdapter, "linux"); owner != nil {
			t.Errorf("ResolvingOwner(npm, linux) = %v, want nil", owner)
		}
		// nil adapter -> nil
		if owner := ResolvingOwner(nil, "linux"); owner != nil {
			t.Errorf("ResolvingOwner(nil, linux) = %v, want nil", owner)
		}
	})

	t.Run("ResolvingOwner custom adapter with and without manager", func(t *testing.T) {
		customWithManager, err := adapters.NewCustomAdapter("mytool", "echo 1", "", false, brewAdapter)
		if err != nil {
			t.Fatalf("NewCustomAdapter error: %v", err)
		}
		if owner := ResolvingOwner(customWithManager, "macos"); owner == nil || owner.Name() != "brew" {
			t.Errorf("ResolvingOwner(customWithManager, macos) = %v, want brew", owner)
		}

		customStandalone, err := adapters.NewCustomAdapter("standalone", "echo 1", "", false)
		if err != nil {
			t.Fatalf("NewCustomAdapter error: %v", err)
		}
		if owner := ResolvingOwner(customStandalone, "linux"); owner != nil {
			t.Errorf("ResolvingOwner(customStandalone, linux) = %v, want nil", owner)
		}
	})

	t.Run("ResolvingOwner allAdapters override", func(t *testing.T) {
		mockApt := &dummyAdapter{name: "apt"}
		allAdapters := []adapters.Adapter{mockApt}
		if owner := ResolvingOwner(ghAdapter, "linux", allAdapters); owner != mockApt {
			t.Errorf("ResolvingOwner with allAdapters override = %v, want mockApt", owner)
		}
	})

	t.Run("OwnedPackage across platforms", func(t *testing.T) {
		// gh package names
		if pkg := OwnedPackage(ghAdapter, "linux"); pkg != "gh" {
			t.Errorf("OwnedPackage(gh, linux) = %q, want gh", pkg)
		}
		if pkg := OwnedPackage(ghAdapter, "darwin"); pkg != "gh" {
			t.Errorf("OwnedPackage(gh, darwin) = %q, want gh", pkg)
		}
		if pkg := OwnedPackage(ghAdapter, "windows"); pkg != "gh" {
			t.Errorf("OwnedPackage(gh, windows) = %q, want gh", pkg)
		}

		// docker package names
		if pkg := OwnedPackage(dockerAdapter, "linux"); pkg != "docker-ce" {
			t.Errorf("OwnedPackage(docker, linux) = %q, want docker-ce", pkg)
		}
		if pkg := OwnedPackage(dockerAdapter, "darwin"); pkg != "docker" {
			t.Errorf("OwnedPackage(docker, darwin) = %q, want docker", pkg)
		}
		if pkg := OwnedPackage(dockerAdapter, "windows"); pkg != "Docker.Docker" {
			t.Errorf("OwnedPackage(docker, windows) = %q, want Docker.Docker", pkg)
		}

		// go package names
		if pkg := OwnedPackage(goAdapter, "linux"); pkg != "" {
			t.Errorf("OwnedPackage(go, linux) = %q, want empty string", pkg)
		}
		if pkg := OwnedPackage(goAdapter, "darwin"); pkg != "golang" {
			t.Errorf("OwnedPackage(go, darwin) = %q, want golang", pkg)
		}
		if pkg := OwnedPackage(goAdapter, "windows"); pkg != "GoLang.Go" {
			t.Errorf("OwnedPackage(go, windows) = %q, want GoLang.Go", pkg)
		}

		// nil adapter and standalone tool
		if pkg := OwnedPackage(nil, "linux"); pkg != "" {
			t.Errorf("OwnedPackage(nil, linux) = %q, want empty string", pkg)
		}
		if pkg := OwnedPackage(npmAdapter, "linux"); pkg != "" {
			t.Errorf("OwnedPackage(npm, linux) = %q, want empty string", pkg)
		}
	})

	t.Run("UpdateCmdName mappings", func(t *testing.T) {
		tests := []struct {
			manager string
			want    string
		}{
			{"apt", "sudo apt install --only-upgrade"},
			{"brew", "brew upgrade"},
			{"winget", "winget upgrade"},
			{"scoop", "scoop upgrade"},
			{"pacman", "pacman upgrade"},
			{"custom-mgr", "custom-mgr upgrade"},
		}

		for _, tt := range tests {
			if got := UpdateCmdName(tt.manager); got != tt.want {
				t.Errorf("UpdateCmdName(%q) = %q, want %q", tt.manager, got, tt.want)
			}
		}
	})

	// Suppress unused variable warnings if any
	_ = aptAdapter
	_ = brewAdapter
	_ = wingetAdapter
}

func TestResolve_EdgeCases(t *testing.T) {
	t.Run("nil config executes safely", func(t *testing.T) {
		eng := New(nil, "linux")
		resolved, err := eng.Resolve(Filter{})
		if err != nil {
			t.Fatalf("Resolve(Filter{}) with nil cfg error: %v", err)
		}
		if len(resolved) == 0 {
			t.Fatalf("len(resolved) = 0, want official adapters")
		}
	})

	t.Run("disabled custom tool excluded", func(t *testing.T) {
		cfg := &config.Config{
			Tools: map[string]config.ToolConfig{
				"custom1": {Enabled: false},
			},
			Custom: map[string]config.CustomTool{
				"custom1": {Command: "echo 1"},
				"custom2": {Command: "echo 2"},
			},
		}
		eng := New(cfg, "linux")
		resolved, err := eng.Resolve(Filter{})
		if err != nil {
			t.Fatalf("Resolve error: %v", err)
		}
		for _, a := range resolved {
			if a.Name() == "custom1" {
				t.Errorf("custom1 was disabled in cfg.Tools but was resolved")
			}
		}
	})

	t.Run("unknown OS passes through canonicalOS", func(t *testing.T) {
		got := canonicalOS("freebsd")
		if got != "freebsd" {
			t.Errorf("canonicalOS(freebsd) = %q, want freebsd", got)
		}
	})

	t.Run("filter matches tool ID differing from name", func(t *testing.T) {
		cfg := &config.Config{}
		eng := New(cfg, "linux")
		// "gh" adapter has ID "gh" and Name "gh"
		resolved, err := eng.Resolve(Filter{Only: []string{"GH"}})
		if err != nil {
			t.Fatalf("Resolve error: %v", err)
		}
		if len(resolved) != 1 || resolved[0].Name() != "gh" {
			t.Fatalf("resolved = %v, want gh", resolved)
		}
	})
}
