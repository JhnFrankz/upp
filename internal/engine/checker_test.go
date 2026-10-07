package engine

import (
	"context"
	"errors"
	"testing"

	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/platform"
)

type mockAdapterForChecker struct {
	info       adapters.ToolInfo
	checkCalls int
	checkInfo  adapters.UpdateInfo
	checkErr   error
}

func (m *mockAdapterForChecker) Name() string { return m.info.Name }
func (m *mockAdapterForChecker) Detect() bool { return true }
func (m *mockAdapterForChecker) Check(ctx context.Context) (adapters.UpdateInfo, error) {
	m.checkCalls++
	return m.checkInfo, m.checkErr
}
func (m *mockAdapterForChecker) Update(ctx context.Context, dryRun bool) (adapters.Result, error) {
	return adapters.Result{Success: true}, nil
}
func (m *mockAdapterForChecker) Info() adapters.ToolInfo { return m.info }

type mockManagerChecker struct {
	mockAdapterForChecker
	checkPkgCalls int
	lastPkg       string
	pkgInfo       adapters.UpdateInfo
	pkgErr        error
}

func (m *mockManagerChecker) CheckPackage(ctx context.Context, pkg string) (adapters.UpdateInfo, error) {
	m.checkPkgCalls++
	m.lastPkg = pkg
	return m.pkgInfo, m.pkgErr
}

func TestOwnedCheckerAdapter_CheckPackageSuccess(t *testing.T) {
	base := &mockAdapterForChecker{
		info: adapters.ToolInfo{ID: "gh", Name: "gh"},
	}
	mgr := &mockManagerChecker{
		pkgInfo: adapters.UpdateInfo{
			CurrentVersion:  "1.0.0",
			LatestVersion:   "1.1.0",
			UpdateAvailable: true,
		},
	}
	wrapper := &ownedCheckerAdapter{
		Adapter: base,
		checker: mgr,
		pkg:     "gh",
	}

	info, err := wrapper.Check(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mgr.checkPkgCalls != 1 {
		t.Errorf("expected 1 call to CheckPackage, got %d", mgr.checkPkgCalls)
	}
	if mgr.lastPkg != "gh" {
		t.Errorf("expected package 'gh', got %q", mgr.lastPkg)
	}
	if base.checkCalls != 0 {
		t.Errorf("base adapter Check should not have been called, got %d", base.checkCalls)
	}
	if info.CurrentVersion != "1.0.0" || info.LatestVersion != "1.1.0" || !info.UpdateAvailable {
		t.Errorf("unexpected UpdateInfo returned: %+v", info)
	}
}

func TestOwnedCheckerAdapter_CheckPackageError(t *testing.T) {
	base := &mockAdapterForChecker{
		info: adapters.ToolInfo{ID: "gh", Name: "gh"},
	}
	expectedErr := errors.New("manager check failed")
	mgr := &mockManagerChecker{
		pkgErr: expectedErr,
	}
	wrapper := &ownedCheckerAdapter{
		Adapter: base,
		checker: mgr,
		pkg:     "gh",
	}

	info, err := wrapper.Check(context.Background())
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
	if base.checkCalls != 0 {
		t.Errorf("base adapter Check should not have been called on manager error, got %d", base.checkCalls)
	}
	if info != (adapters.UpdateInfo{}) {
		t.Errorf("expected empty UpdateInfo, got %+v", info)
	}
}

func TestOwnedCheckerAdapter_FallbackToBaseCheck(t *testing.T) {
	base := &mockAdapterForChecker{
		info: adapters.ToolInfo{ID: "gh", Name: "gh"},
		checkInfo: adapters.UpdateInfo{
			CurrentVersion:  "2.0.0",
			LatestVersion:   "2.0.0",
			UpdateAvailable: false,
		},
	}
	// Package checker returns empty UpdateInfo and nil error
	mgr := &mockManagerChecker{
		pkgInfo: adapters.UpdateInfo{},
		pkgErr:  nil,
	}
	wrapper := &ownedCheckerAdapter{
		Adapter: base,
		checker: mgr,
		pkg:     "gh",
	}

	info, err := wrapper.Check(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mgr.checkPkgCalls != 1 {
		t.Errorf("expected 1 call to CheckPackage, got %d", mgr.checkPkgCalls)
	}
	if base.checkCalls != 1 {
		t.Errorf("base adapter Check should have been called as fallback, got %d", base.checkCalls)
	}
	if info.CurrentVersion != "2.0.0" {
		t.Errorf("expected fallback version 2.0.0, got %q", info.CurrentVersion)
	}
}

func TestPrepareCheckAdapters_TableDriven(t *testing.T) {
	mgr := &mockManagerChecker{
		mockAdapterForChecker: mockAdapterForChecker{
			info: adapters.ToolInfo{
				ID:   "apt",
				Name: "apt",
				Kind: adapters.KindManager,
			},
		},
	}
	nonCheckerMgr := &mockAdapterForChecker{
		info: adapters.ToolInfo{
			ID:   "custommgr",
			Name: "custommgr",
			Kind: adapters.KindManager,
		},
	}

	ownedTool := &mockAdapterForChecker{
		info: adapters.ToolInfo{
			ID:   "gh",
			Name: "gh",
			Manager: map[string]string{
				platform.OSLinux: "apt",
			},
			ManagerPackage: map[string]string{
				platform.OSLinux: "gh",
			},
		},
	}

	ownedToolNoPkg := &mockAdapterForChecker{
		info: adapters.ToolInfo{
			ID:   "tool-nopkg",
			Name: "tool-nopkg",
			Manager: map[string]string{
				platform.OSLinux: "apt",
			},
		},
	}

	ownedToolNonChecker := &mockAdapterForChecker{
		info: adapters.ToolInfo{
			ID:   "tool-nonchecker",
			Name: "tool-nonchecker",
			Manager: map[string]string{
				platform.OSLinux: "custommgr",
			},
			ManagerPackage: map[string]string{
				platform.OSLinux: "pkg",
			},
		},
	}

	standaloneTool := &mockAdapterForChecker{
		info: adapters.ToolInfo{
			ID:   "npm",
			Name: "npm",
		},
	}

	allList := []adapters.Adapter{mgr, nonCheckerMgr, ownedTool, ownedToolNoPkg, ownedToolNonChecker, standaloneTool}

	tests := []struct {
		name        string
		input       []adapters.Adapter
		osName      string
		allAdapters []adapters.Adapter
		wantWrapped map[string]bool
	}{
		{
			name:        "empty adapter list",
			input:       nil,
			osName:      platform.OSLinux,
			allAdapters: allList,
			wantWrapped: map[string]bool{},
		},
		{
			name:        "wraps owned tool with checker manager",
			input:       []adapters.Adapter{ownedTool, standaloneTool},
			osName:      platform.OSLinux,
			allAdapters: allList,
			wantWrapped: map[string]bool{
				"gh":  true,
				"npm": false,
			},
		},
		{
			name:        "leaves owned tool without package unwrapped",
			input:       []adapters.Adapter{ownedToolNoPkg},
			osName:      platform.OSLinux,
			allAdapters: allList,
			wantWrapped: map[string]bool{
				"tool-nopkg": false,
			},
		},
		{
			name:        "leaves owned tool whose manager does not implement PackageChecker unwrapped",
			input:       []adapters.Adapter{ownedToolNonChecker},
			osName:      platform.OSLinux,
			allAdapters: allList,
			wantWrapped: map[string]bool{
				"tool-nonchecker": false,
			},
		},
		{
			name:        "mismatched OS leaves tool unwrapped",
			input:       []adapters.Adapter{ownedTool},
			osName:      platform.OSMacOS,
			allAdapters: allList,
			wantWrapped: map[string]bool{
				"gh": false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PrepareCheckAdapters(tt.input, tt.osName, tt.allAdapters)
			if len(got) != len(tt.input) {
				t.Fatalf("got length %d, want %d", len(got), len(tt.input))
			}
			for i, a := range got {
				orig := tt.input[i]
				id := orig.Info().ID
				shouldWrap := tt.wantWrapped[id]
				_, isWrapped := a.(*ownedCheckerAdapter)
				if isWrapped != shouldWrap {
					t.Errorf("tool %q wrapped = %v, want %v", id, isWrapped, shouldWrap)
				}
				if !shouldWrap && a != orig {
					t.Errorf("unwrapped tool %q returned different pointer", id)
				}
			}
		})
	}
}

func TestEngine_PrepareCheckAdaptersMethod(t *testing.T) {
	mgr := &mockManagerChecker{
		mockAdapterForChecker: mockAdapterForChecker{
			info: adapters.ToolInfo{
				ID:   "apt",
				Name: "apt",
				Kind: adapters.KindManager,
			},
		},
	}
	tool := &mockAdapterForChecker{
		info: adapters.ToolInfo{
			ID:   "gh",
			Name: "gh",
			Manager: map[string]string{
				platform.OSLinux: "apt",
			},
			ManagerPackage: map[string]string{
				platform.OSLinux: "gh",
			},
		},
	}

	all := []adapters.Adapter{mgr, tool}
	eng := New(nil, platform.OSLinux, WithAdapters(all))

	wrapped := eng.PrepareCheckAdapters([]adapters.Adapter{tool})
	if len(wrapped) != 1 {
		t.Fatalf("expected 1 adapter, got %d", len(wrapped))
	}
	oca, ok := wrapped[0].(*ownedCheckerAdapter)
	if !ok {
		t.Fatalf("expected *ownedCheckerAdapter, got %T", wrapped[0])
	}
	if oca.pkg != "gh" {
		t.Errorf("expected package 'gh', got %q", oca.pkg)
	}

	// Test hermeticity: when manager is NOT in WithAdapters, tool must NOT be wrapped
	engHermetic := New(nil, platform.OSLinux, WithAdapters([]adapters.Adapter{tool}))
	unwrappedHermetic := engHermetic.PrepareCheckAdapters([]adapters.Adapter{tool})
	if len(unwrappedHermetic) != 1 {
		t.Fatalf("expected 1 adapter, got %d", len(unwrappedHermetic))
	}
	if _, isWrapped := unwrappedHermetic[0].(*ownedCheckerAdapter); isWrapped {
		t.Errorf("expected tool not to be wrapped when manager is missing from WithAdapters")
	}

	// Test default engine without WithAdapters: resolves tools and wraps
	engDefault := New(nil, platform.OSLinux)
	ghOfficial := officialAdapterByName(t, "gh")
	if ghOfficial != nil {
		wrappedDefault := engDefault.PrepareCheckAdapters([]adapters.Adapter{ghOfficial})
		if len(wrappedDefault) == 1 {
			if _, isWrapped := wrappedDefault[0].(*ownedCheckerAdapter); !isWrapped {
				t.Errorf("expected official gh to be wrapped by default engine on Linux")
			}
		}
	}
}

func officialAdapterByName(t *testing.T, name string) adapters.Adapter {
	t.Helper()
	for _, a := range []adapters.Adapter{toolGhForTest()} {
		if a.Name() == name || a.Info().ID == name {
			return a
		}
	}
	return nil
}

func toolGhForTest() adapters.Adapter {
	return &mockAdapterForChecker{
		info: adapters.ToolInfo{
			ID:   "gh",
			Name: "gh",
			Manager: map[string]string{
				platform.OSLinux: "apt",
			},
			ManagerPackage: map[string]string{
				platform.OSLinux: "gh",
			},
		},
	}
}

func TestPrepareCheckAdapters_NoDoubleWrap(t *testing.T) {
	mgr := &mockManagerChecker{
		mockAdapterForChecker: mockAdapterForChecker{
			info: adapters.ToolInfo{
				ID:   "apt",
				Name: "apt",
				Kind: adapters.KindManager,
			},
		},
	}
	tool := &mockAdapterForChecker{
		info: adapters.ToolInfo{
			ID:   "gh",
			Name: "gh",
			Manager: map[string]string{
				platform.OSLinux: "apt",
			},
			ManagerPackage: map[string]string{
				platform.OSLinux: "gh",
			},
		},
	}

	firstPass := PrepareCheckAdapters([]adapters.Adapter{tool}, platform.OSLinux, []adapters.Adapter{mgr, tool})
	if len(firstPass) != 1 {
		t.Fatalf("expected 1 adapter, got %d", len(firstPass))
	}
	oca, ok := firstPass[0].(*ownedCheckerAdapter)
	if !ok {
		t.Fatalf("expected *ownedCheckerAdapter on first pass, got %T", firstPass[0])
	}

	// Second pass over already-wrapped adapter
	secondPass := PrepareCheckAdapters(firstPass, platform.OSLinux, []adapters.Adapter{mgr, tool})
	if len(secondPass) != 1 {
		t.Fatalf("expected 1 adapter, got %d", len(secondPass))
	}
	if secondPass[0] != oca {
		t.Errorf("expected second pass to return same *ownedCheckerAdapter without re-wrapping")
	}
}
