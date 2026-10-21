package adapters

import (
	"context"
	"sync"
	"testing"

	"github.com/JhnFrankz/upp/internal/config"
	"github.com/JhnFrankz/upp/internal/platform"
)

type mockAdapter struct {
	name      string
	kind      Kind
	platforms []string
}

func (m *mockAdapter) Name() string { return m.name }
func (m *mockAdapter) Detect() bool { return true }
func (m *mockAdapter) Check(ctx context.Context) (UpdateInfo, error) {
	return UpdateInfo{}, nil
}
func (m *mockAdapter) Update(ctx context.Context, dryRun bool) (Result, error) {
	return Result{}, nil
}
func (m *mockAdapter) Info() ToolInfo {
	return ToolInfo{
		ID:        m.name,
		Name:      m.name,
		Kind:      m.kind,
		Platforms: m.platforms,
	}
}

type mockProvider struct {
	adapters []Adapter
	byName   map[string]Adapter
	owners   map[string]Adapter
}

func (m *mockProvider) AdaptersForPlatform(os string) []Adapter {
	var res []Adapter
	for _, a := range m.adapters {
		if a == nil {
			res = append(res, nil)
			continue
		}
		info := a.Info()
		if len(info.Platforms) == 0 || info.SupportsPlatform(os) {
			res = append(res, a)
		}
	}
	return res
}

func (m *mockProvider) AdapterByName(name string) Adapter {
	if m.byName != nil {
		return m.byName[name]
	}
	return nil
}

func (m *mockProvider) ResolveOwner(tool, os string) Adapter {
	if m.owners != nil {
		return m.owners[tool+":"+os]
	}
	return nil
}

func TestProviderRegistrationAndHelpers(t *testing.T) {
	orig := GlobalProvider()
	defer RegisterProvider(orig)

	p := &mockProvider{
		adapters: []Adapter{
			&mockAdapter{name: "brew", kind: KindManager, platforms: []string{platform.OSMacOS, platform.OSLinux}},
		},
		byName: map[string]Adapter{
			"brew": &mockAdapter{name: "brew", kind: KindManager, platforms: []string{platform.OSMacOS, platform.OSLinux}},
		},
		owners: map[string]Adapter{
			"gh:macos": &mockAdapter{name: "brew", kind: KindManager},
		},
	}

	RegisterProvider(p)
	if got := GlobalProvider(); got != p {
		t.Fatalf("GlobalProvider() = %v, want %v", got, p)
	}

	darwinAdapters := AdaptersForPlatform("darwin")
	if len(darwinAdapters) != 1 || darwinAdapters[0].Name() != "brew" {
		t.Errorf("AdaptersForPlatform(darwin) = %v, want [brew]", darwinAdapters)
	}

	upperDarwinAdapters := AdaptersForPlatform("DARWIN")
	if len(upperDarwinAdapters) != 1 || upperDarwinAdapters[0].Name() != "brew" {
		t.Errorf("AdaptersForPlatform(DARWIN) = %v, want [brew]", upperDarwinAdapters)
	}

	if freebsdAdapters := AdaptersForPlatform("freebsd"); freebsdAdapters != nil {
		t.Errorf("AdaptersForPlatform(freebsd) = %v, want nil", freebsdAdapters)
	}

	a := AdapterByName("brew")
	if a == nil || a.Name() != "brew" {
		t.Errorf("AdapterByName(brew) = %v, want brew", a)
	}

	owner := ResolveOwner("gh", "macos")
	if owner == nil || owner.Name() != "brew" {
		t.Errorf("ResolveOwner(gh, macos) = %v, want brew", owner)
	}

	// Verify un-normalized OS ("darwin" and uppercase "DARWIN") normalizes to "macos"
	ownerDarwin := ResolveOwner("gh", "darwin")
	if ownerDarwin == nil || ownerDarwin.Name() != "brew" {
		t.Errorf("ResolveOwner(gh, darwin) = %v, want brew", ownerDarwin)
	}

	ownerDarwinUpper := ResolveOwner("gh", "DARWIN")
	if ownerDarwinUpper == nil || ownerDarwinUpper.Name() != "brew" {
		t.Errorf("ResolveOwner(gh, DARWIN) = %v, want brew", ownerDarwinUpper)
	}

	if ownerFreebsd := ResolveOwner("gh", "freebsd"); ownerFreebsd != nil {
		t.Errorf("ResolveOwner(gh, freebsd) = %v, want nil", ownerFreebsd)
	}
}

func TestResolveAdapters_NilProviderReturnsEmpty(t *testing.T) {
	orig := GlobalProvider()
	RegisterProvider(nil)
	defer RegisterProvider(orig)

	resolved, err := ResolveAdapters(nil, "linux")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved == nil || len(resolved) != 0 {
		t.Errorf("expected empty non-nil slice, got %v", resolved)
	}
}

func TestResolveAdapters_PlatformAndCustomTools(t *testing.T) {
	mgr := &mockAdapter{name: "brew", kind: KindManager, platforms: []string{platform.OSMacOS}}
	tool := &mockAdapter{name: "gh", kind: KindTool, platforms: []string{platform.OSMacOS}}

	p := &mockProvider{
		adapters: []Adapter{mgr, tool},
		byName: map[string]Adapter{
			"brew": mgr,
		},
	}

	cfg := &config.Config{
		Tools: map[string]config.ToolConfig{
			"gh": {Enabled: false},
		},
		Custom: map[string]config.CustomTool{
			"custom-managed": {
				Command:  "custom-managed --version",
				CheckCmd: "custom-managed check",
				Manager:  "brew",
				Package:  "pkg-managed",
				Trusted:  true,
			},
			"custom-standalone": {
				Command: "custom-standalone --version",
			},
			"custom-disabled": {
				Command: "custom-disabled --version",
			},
		},
	}
	cfg.Tools["custom-disabled"] = config.ToolConfig{Enabled: false}

	resolved, err := ResolveAdapters(cfg, "darwin", WithProvider(p))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	names := make(map[string]bool)
	for _, a := range resolved {
		names[a.Name()] = true
	}

	if !names["brew"] {
		t.Errorf("expected brew to be present in resolved adapters")
	}
	if names["gh"] {
		t.Errorf("expected gh to be filtered out (disabled in Tools)")
	}
	if !names["custom-managed"] {
		t.Errorf("expected custom-managed to be present")
	}
	if !names["custom-standalone"] {
		t.Errorf("expected custom-standalone to be present")
	}
	if names["custom-disabled"] {
		t.Errorf("expected custom-disabled to be filtered out")
	}

	// Verify manager binding on custom-managed
	for _, a := range resolved {
		if a.Name() == "custom-managed" {
			ca, ok := a.(*CustomAdapter)
			if !ok {
				t.Fatalf("expected *CustomAdapter, got %T", a)
			}
			if ca.ManagerAdapter() != mgr {
				t.Errorf("custom-managed ManagerAdapter() = %v, want %v", ca.ManagerAdapter(), mgr)
			}
			if ca.Package() != "pkg-managed" {
				t.Errorf("custom-managed Package() = %q, want pkg-managed", ca.Package())
			}
			if !ca.IsTrusted() {
				t.Errorf("custom-managed IsTrusted() = false, want true")
			}
		}
	}
}

func TestResolveAdapters_CustomToolDisabledManager(t *testing.T) {
	mgr := &mockAdapter{name: "brew", kind: KindManager, platforms: []string{platform.OSMacOS}}
	p := &mockProvider{
		adapters: []Adapter{mgr},
		byName: map[string]Adapter{
			"brew": mgr,
		},
	}

	cfg := &config.Config{
		Tools: map[string]config.ToolConfig{
			"brew": {Enabled: false},
		},
		Custom: map[string]config.CustomTool{
			"my-tool": {
				Command: "my-tool --update",
				Manager: "brew",
			},
		},
	}

	resolved, err := ResolveAdapters(cfg, "macos", WithProvider(p))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resolved) != 1 || resolved[0].Name() != "my-tool" {
		t.Fatalf("expected only my-tool, got %v", resolved)
	}

	ca, ok := resolved[0].(*CustomAdapter)
	if !ok {
		t.Fatalf("expected *CustomAdapter, got %T", resolved[0])
	}
	if ca.ManagerAdapter() != nil {
		t.Errorf("expected nil ManagerAdapter when manager is disabled in config, got %v", ca.ManagerAdapter())
	}
}

func TestResolveAdapters_NilAdapterInProviderList(t *testing.T) {
	mgr := &mockAdapter{name: "brew", kind: KindManager, platforms: []string{platform.OSMacOS}}
	p := &mockProvider{
		adapters: []Adapter{nil, mgr, nil},
		byName: map[string]Adapter{
			"brew": mgr,
		},
	}

	resolved, err := ResolveAdapters(nil, "macos", WithProvider(p))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resolved) != 1 || resolved[0].Name() != "brew" {
		t.Errorf("expected 1 adapter [brew], got %v", resolved)
	}
}

func TestResolveAdapters_CustomToolManagerDisabledByID(t *testing.T) {
	mgr := &mockAdapter{name: "Homebrew", kind: KindManager, platforms: []string{platform.OSMacOS}}
	p := &mockProvider{
		adapters: []Adapter{mgr},
		byName: map[string]Adapter{
			"brew": mgr,
		},
	}

	cfg := &config.Config{
		Tools: map[string]config.ToolConfig{
			"Homebrew": {Enabled: false},
		},
		Custom: map[string]config.CustomTool{
			"tool": {
				Command: "tool --update",
				Manager: "brew",
			},
		},
	}

	resolved, err := ResolveAdapters(cfg, "macos", WithProvider(p))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resolved) != 1 {
		t.Fatalf("expected 1 custom tool, got %d", len(resolved))
	}
	ca, ok := resolved[0].(*CustomAdapter)
	if !ok {
		t.Fatalf("expected *CustomAdapter, got %T", resolved[0])
	}
	if ca.ManagerAdapter() != nil {
		t.Errorf("expected nil ManagerAdapter when manager is disabled by ID/name in config, got %v", ca.ManagerAdapter())
	}
}

func TestProviderConcurrency(t *testing.T) {
	orig := GlobalProvider()
	defer RegisterProvider(orig)

	mgr := &mockAdapter{name: "brew", kind: KindManager, platforms: []string{platform.OSMacOS}}
	p := &mockProvider{
		adapters: []Adapter{mgr},
		byName: map[string]Adapter{
			"brew": mgr,
		},
		owners: map[string]Adapter{
			"gh:macos": mgr,
		},
	}
	RegisterProvider(p)

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			switch idx % 4 {
			case 0:
				RegisterProvider(p)
			case 1:
				_ = GlobalProvider()
			case 2:
				_ = AdaptersForPlatform("darwin")
				_ = ResolveOwner("gh", "darwin")
			default:
				_, _ = ResolveAdapters(nil, "darwin")
			}
		}(i)
	}
	wg.Wait()
}
