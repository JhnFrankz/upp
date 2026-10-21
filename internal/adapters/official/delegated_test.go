package official

import (
	"context"
	"strings"
	"testing"

	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/security"
)

type dummyNonCheckerManager struct {
	adapters.Adapter
}

func (d *dummyNonCheckerManager) Name() string { return "dummy-mgr" }
func (d *dummyNonCheckerManager) Detect() bool { return true }
func (d *dummyNonCheckerManager) Check(ctx context.Context) (adapters.UpdateInfo, error) {
	return adapters.UpdateInfo{}, nil
}
func (d *dummyNonCheckerManager) Update(ctx context.Context, dryRun bool) (adapters.Result, error) {
	return adapters.Result{Success: true}, nil
}
func (d *dummyNonCheckerManager) Info() adapters.ToolInfo {
	return adapters.ToolInfo{ID: "dummy-mgr", Name: "Dummy Manager"}
}

type dummyToolNoPkg struct{}

func (d *dummyToolNoPkg) Name() string { return "gh" }
func (d *dummyToolNoPkg) Detect() bool { return true }
func (d *dummyToolNoPkg) Check(ctx context.Context) (adapters.UpdateInfo, error) {
	return adapters.UpdateInfo{}, nil
}
func (d *dummyToolNoPkg) Update(ctx context.Context, dryRun bool) (adapters.Result, error) {
	return adapters.Result{Success: true}, nil
}
func (d *dummyToolNoPkg) Info() adapters.ToolInfo {
	return adapters.ToolInfo{
		ID:             "gh",
		Name:           "GitHub CLI",
		Platforms:      platformsLinuxMacOSWindows,
		Trust:          security.TrustOfficial,
		Kind:           adapters.KindTool,
		Manager:        map[string]string{"linux": "apt"},
		ManagerPackage: map[string]string{"linux": ""}, // empty package!
	}
}

func TestDelegatedCheck_OwnerNotDetected(t *testing.T) {
	setExecFakes(t, execFakes{
		goos:     "linux",
		lookPath: map[string]bool{"gh": true, "apt": false},
	})

	info, err := DelegatedCheck(context.Background(), &GhAdapter{}, "2.45.0")
	if err != nil {
		t.Fatalf("DelegatedCheck() error = %v, want nil", err)
	}
	want := adapters.UpdateInfo{
		CurrentVersion:  "2.45.0",
		LatestVersion:   "2.45.0",
		UpdateAvailable: false,
	}
	if info != want {
		t.Errorf("DelegatedCheck() = %+v, want %+v", info, want)
	}
}

func TestDelegatedCheck_Success(t *testing.T) {
	setExecFakes(t, execFakes{
		goos:     "linux",
		lookPath: map[string]bool{"gh": true, "apt": true},
		shell: map[string]fakeResult{
			aptPolicyCmd("gh", "Installed"): {stdout: "2.45.0"},
			aptPolicyCmd("gh", "Candidate"): {stdout: "2.46.0"},
		},
	})

	info, err := DelegatedCheck(context.Background(), &GhAdapter{}, "2.45.0")
	if err != nil {
		t.Fatalf("DelegatedCheck() unexpected error: %v", err)
	}
	want := adapters.UpdateInfo{
		CurrentVersion:  "2.45.0",
		LatestVersion:   "2.46.0",
		UpdateAvailable: true,
	}
	if info != want {
		t.Errorf("DelegatedCheck() = %+v, want %+v", info, want)
	}
}

func TestDelegatedCheck_NoResolvingOwner(t *testing.T) {
	setExecFakes(t, execFakes{
		goos:         "linux",
		goBinaryPath: "/usr/local/go/bin/go",
		lookPath:     map[string]bool{"go": true},
	})

	_, err := DelegatedCheck(context.Background(), &GoAdapter{}, "1.22.0")
	if err == nil {
		t.Fatal("DelegatedCheck() expected error for unmanaged tool, got nil")
	}
	if !strings.Contains(err.Error(), "go has no resolving owner on linux") {
		t.Errorf("DelegatedCheck() error = %q, want contains %q", err.Error(), "go has no resolving owner on linux")
	}
}

func TestDelegatedCheck_ManagerNotPackageChecker(t *testing.T) {
	setExecFakes(t, execFakes{
		goos:     "linux",
		lookPath: map[string]bool{"gh": true, "apt": true},
	})

	origResolveOwner := resolveOwnerFn
	resolveOwnerFn = func(tool, os string) adapters.Adapter {
		return &dummyNonCheckerManager{}
	}
	t.Cleanup(func() {
		resolveOwnerFn = origResolveOwner
	})

	_, err := DelegatedCheck(context.Background(), &GhAdapter{}, "2.45.0")
	if err == nil {
		t.Fatal("DelegatedCheck() expected error for manager lacking PackageChecker, got nil")
	}
	if !strings.Contains(err.Error(), "gh's manager dummy-mgr does not support per-package checks") {
		t.Errorf("DelegatedCheck() error = %q, want contains expected message", err.Error())
	}
}

func TestDelegatedCheck_NoManagerPackage(t *testing.T) {
	setExecFakes(t, execFakes{
		goos:     "linux",
		lookPath: map[string]bool{"gh": true, "apt": true},
	})

	_, err := DelegatedCheck(context.Background(), &dummyToolNoPkg{}, "2.45.0")
	if err == nil {
		t.Fatal("DelegatedCheck() expected error when ManagerPackage is empty, got nil")
	}
	if !strings.Contains(err.Error(), "gh has no manager package on linux") {
		t.Errorf("DelegatedCheck() error = %q, want contains %q", err.Error(), "gh has no manager package on linux")
	}
}

func TestDelegatedUpdate_DryRun(t *testing.T) {
	setExecFakes(t, execFakes{
		goos:     "linux",
		lookPath: map[string]bool{"gh": true, "apt": true},
	})

	res, err := DelegatedUpdate(context.Background(), &GhAdapter{}, true)
	if err != nil {
		t.Fatalf("DelegatedUpdate() dry-run error = %v, want nil", err)
	}
	if !res.Success {
		t.Errorf("DelegatedUpdate() dry-run Success = false, want true")
	}
}

func TestDelegatedUpdate_Success(t *testing.T) {
	setExecFakes(t, execFakes{
		goos:     "linux",
		lookPath: map[string]bool{"gh": true, "apt": true},
		shell:    map[string]fakeResult{"sudo apt install --only-upgrade gh": {}},
	})

	res, err := DelegatedUpdate(context.Background(), &GhAdapter{}, false)
	if err != nil {
		t.Fatalf("DelegatedUpdate() error = %v, want nil", err)
	}
	if !res.Success {
		t.Errorf("DelegatedUpdate() Success = false, want true")
	}
}

func TestDelegatedUpdate_NoResolvingOwner(t *testing.T) {
	setExecFakes(t, execFakes{
		goos:         "linux",
		goBinaryPath: "/usr/local/go/bin/go",
		lookPath:     map[string]bool{"go": true},
	})

	res, err := DelegatedUpdate(context.Background(), &GoAdapter{}, false)
	if err != nil {
		t.Fatalf("DelegatedUpdate() unexpected return error: %v", err)
	}
	if res.Success {
		t.Errorf("DelegatedUpdate() Success = true, want false")
	}
	if res.Error == nil || !strings.Contains(res.Error.Error(), "go has no resolving owner on linux") {
		t.Errorf("DelegatedUpdate() res.Error = %v, want contains %q", res.Error, "go has no resolving owner on linux")
	}
}

func TestDelegatedUpdate_ManagerNotPackageUpdater(t *testing.T) {
	setExecFakes(t, execFakes{
		goos:     "linux",
		lookPath: map[string]bool{"gh": true, "apt": true},
	})

	origResolveOwner := resolveOwnerFn
	resolveOwnerFn = func(tool, os string) adapters.Adapter {
		return &dummyNonCheckerManager{}
	}
	t.Cleanup(func() {
		resolveOwnerFn = origResolveOwner
	})

	res, err := DelegatedUpdate(context.Background(), &GhAdapter{}, false)
	if err == nil {
		t.Fatal("DelegatedUpdate() expected error when manager lacks PackageUpdater, got nil")
	}
	if res.Success {
		t.Errorf("DelegatedUpdate() Success = true, want false")
	}
	if !strings.Contains(err.Error(), "gh's manager dummy-mgr does not support per-package updates") {
		t.Errorf("DelegatedUpdate() error = %q, want contains expected message", err.Error())
	}
}

func TestDelegatedUpdate_NoManagerPackage(t *testing.T) {
	setExecFakes(t, execFakes{
		goos:     "linux",
		lookPath: map[string]bool{"gh": true, "apt": true},
	})

	res, err := DelegatedUpdate(context.Background(), &dummyToolNoPkg{}, false)
	if err == nil {
		t.Fatal("DelegatedUpdate() expected error when ManagerPackage is empty, got nil")
	}
	if res.Success {
		t.Errorf("DelegatedUpdate() Success = true, want false")
	}
	if !strings.Contains(err.Error(), "gh has no manager package on linux") {
		t.Errorf("DelegatedUpdate() error = %q, want contains %q", err.Error(), "gh has no manager package on linux")
	}
}
