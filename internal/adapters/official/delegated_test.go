package official

import (
	"context"
	"errors"
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

func (d *dummyToolNoPkg) Name() string { return "custom-pkgless" }
func (d *dummyToolNoPkg) Detect() bool { return true }
func (d *dummyToolNoPkg) Check(ctx context.Context) (adapters.UpdateInfo, error) {
	return adapters.UpdateInfo{}, nil
}
func (d *dummyToolNoPkg) Update(ctx context.Context, dryRun bool) (adapters.Result, error) {
	return adapters.Result{Success: true}, nil
}
func (d *dummyToolNoPkg) Info() adapters.ToolInfo {
	return adapters.ToolInfo{
		ID:             "custom-pkgless",
		Name:           "Custom Tool",
		Platforms:      platformsLinuxMacOSWindows,
		Trust:          security.TrustOfficial,
		Kind:           adapters.KindTool,
		Manager:        map[string]string{"linux": "apt"},
		ManagerPackage: map[string]string{"linux": ""}, // empty package!
	}
}

type dummyCustomTool struct{}

func (d *dummyCustomTool) Name() string { return "custom-tool" }
func (d *dummyCustomTool) Detect() bool { return true }
func (d *dummyCustomTool) Check(ctx context.Context) (adapters.UpdateInfo, error) {
	return adapters.UpdateInfo{}, nil
}
func (d *dummyCustomTool) Update(ctx context.Context, dryRun bool) (adapters.Result, error) {
	return adapters.Result{Success: true}, nil
}
func (d *dummyCustomTool) Info() adapters.ToolInfo {
	return adapters.ToolInfo{
		ID:             "custom-tool",
		Name:           "Custom Tool",
		Platforms:      platformsLinuxMacOSWindows,
		Trust:          security.TrustOfficial,
		Kind:           adapters.KindTool,
		Manager:        map[string]string{"linux": "apt"},
		ManagerPackage: map[string]string{"linux": "custom-package"},
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
	if !errors.Is(err, adapters.ErrNoResolvingOwner) {
		t.Errorf("DelegatedCheck() error = %v, want errors.Is(err, adapters.ErrNoResolvingOwner)", err)
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
	if !errors.Is(err, adapters.ErrUnsupportedPackageCheck) {
		t.Errorf("DelegatedCheck() error = %v, want errors.Is(err, adapters.ErrUnsupportedPackageCheck)", err)
	}
}

func TestDelegatedCheck_NoManagerPackage(t *testing.T) {
	setExecFakes(t, execFakes{
		goos:     "linux",
		lookPath: map[string]bool{"apt": true},
	})

	_, err := DelegatedCheck(context.Background(), &dummyToolNoPkg{}, "2.45.0")
	if err == nil {
		t.Fatal("DelegatedCheck() expected error when ManagerPackage is empty, got nil")
	}
	if !strings.Contains(err.Error(), "custom-pkgless has no manager package on linux") {
		t.Errorf("DelegatedCheck() error = %q, want contains %q", err.Error(), "custom-pkgless has no manager package on linux")
	}
	if !errors.Is(err, adapters.ErrNoManagerPackage) {
		t.Errorf("DelegatedCheck() error = %v, want errors.Is(err, adapters.ErrNoManagerPackage)", err)
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
	if !errors.Is(res.Error, adapters.ErrNoResolvingOwner) {
		t.Errorf("DelegatedUpdate() res.Error = %v, want errors.Is(res.Error, adapters.ErrNoResolvingOwner)", res.Error)
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
	if !errors.Is(err, adapters.ErrUnsupportedPackageUpdate) {
		t.Errorf("DelegatedUpdate() error = %v, want errors.Is(err, adapters.ErrUnsupportedPackageUpdate)", err)
	}
}

func TestDelegatedUpdate_NoManagerPackage(t *testing.T) {
	setExecFakes(t, execFakes{
		goos:     "linux",
		lookPath: map[string]bool{"apt": true},
	})

	res, err := DelegatedUpdate(context.Background(), &dummyToolNoPkg{}, false)
	if err == nil {
		t.Fatal("DelegatedUpdate() expected error when ManagerPackage is empty, got nil")
	}
	if res.Success {
		t.Errorf("DelegatedUpdate() Success = true, want false")
	}
	if !strings.Contains(err.Error(), "custom-pkgless has no manager package on linux") {
		t.Errorf("DelegatedUpdate() error = %q, want contains %q", err.Error(), "custom-pkgless has no manager package on linux")
	}
	if !errors.Is(err, adapters.ErrNoManagerPackage) {
		t.Errorf("DelegatedUpdate() error = %v, want errors.Is(err, adapters.ErrNoManagerPackage)", err)
	}
}

func TestDelegated_NilAdapter(t *testing.T) {
	_, err := DelegatedCheck(context.Background(), nil, "1.0.0")
	if err == nil || !strings.Contains(err.Error(), "adapter is nil") {
		t.Fatalf("DelegatedCheck(nil) error = %v, want contains 'adapter is nil'", err)
	}
	if !errors.Is(err, adapters.ErrNilAdapter) {
		t.Fatalf("DelegatedCheck(nil) error = %v, want errors.Is(err, adapters.ErrNilAdapter)", err)
	}

	res, err := DelegatedUpdate(context.Background(), nil, false)
	if err != nil {
		t.Fatalf("DelegatedUpdate(nil) unexpected err: %v", err)
	}
	if res.Success || res.Error == nil || !strings.Contains(res.Error.Error(), "adapter is nil") {
		t.Fatalf("DelegatedUpdate(nil) res = %+v, want Failure with 'adapter is nil'", res)
	}
	if !errors.Is(res.Error, adapters.ErrNilAdapter) {
		t.Fatalf("DelegatedUpdate(nil) res.Error = %v, want errors.Is(res.Error, adapters.ErrNilAdapter)", res.Error)
	}
}

func TestDelegated_NilContext(t *testing.T) {
	setExecFakes(t, execFakes{
		goos:     "linux",
		lookPath: map[string]bool{"gh": true, "apt": true},
		shell: map[string]fakeResult{
			aptPolicyCmd("gh", "Installed"): {stdout: "2.45.0"},
			aptPolicyCmd("gh", "Candidate"): {stdout: "2.46.0"},
		},
	})

	var nilCtx context.Context
	info, err := DelegatedCheck(nilCtx, &GhAdapter{}, "2.45.0")
	if err != nil {
		t.Fatalf("DelegatedCheck(nil ctx) error = %v, want nil", err)
	}
	if !info.UpdateAvailable {
		t.Errorf("DelegatedCheck(nil ctx) UpdateAvailable = false, want true")
	}

	res, err := DelegatedUpdate(nilCtx, &GhAdapter{}, true)
	if err != nil || !res.Success {
		t.Fatalf("DelegatedUpdate(nil ctx) = %+v, %v, want Success=true", res, err)
	}
}

func TestDelegatedCheck_UnregisteredAdapterWithDeclaredManager(t *testing.T) {
	setExecFakes(t, execFakes{
		goos:     "linux",
		lookPath: map[string]bool{"apt": true},
		shell: map[string]fakeResult{
			aptPolicyCmd("custom-package", "Installed"): {stdout: "1.0.0"},
			aptPolicyCmd("custom-package", "Candidate"): {stdout: "1.1.0"},
		},
	})

	info, err := DelegatedCheck(context.Background(), &dummyCustomTool{}, "1.0.0")
	if err != nil {
		t.Fatalf("DelegatedCheck() unexpected error: %v", err)
	}
	want := adapters.UpdateInfo{
		CurrentVersion:  "1.0.0",
		LatestVersion:   "1.1.0",
		UpdateAvailable: true,
	}
	if info != want {
		t.Errorf("DelegatedCheck() = %+v, want %+v", info, want)
	}
}

func TestDelegatedUpdate_UnregisteredAdapterWithDeclaredManager(t *testing.T) {
	setExecFakes(t, execFakes{
		goos:     "linux",
		lookPath: map[string]bool{"apt": true},
		shell:    map[string]fakeResult{"sudo apt install --only-upgrade custom-package": {}},
	})

	res, err := DelegatedUpdate(context.Background(), &dummyCustomTool{}, false)
	if err != nil {
		t.Fatalf("DelegatedUpdate() unexpected error: %v", err)
	}
	if !res.Success {
		t.Errorf("DelegatedUpdate() Success = false, want true")
	}
}

func TestGoAdapter_HonorsResolveOwnerSeam(t *testing.T) {
	setExecFakes(t, execFakes{
		goos:         "linux",
		goBinaryPath: "/usr/local/go/bin/go",
		lookPath:     map[string]bool{"go": true},
		cmdArgs:      map[string]fakeResult{"go": {stdout: "go version go1.22.0 linux/amd64"}},
	})

	origResolveOwner := resolveOwnerFn
	resolveOwnerFn = func(tool, os string) adapters.Adapter {
		return &dummyNonCheckerManager{}
	}
	t.Cleanup(func() {
		resolveOwnerFn = origResolveOwner
	})

	_, err := (&GoAdapter{}).Check(context.Background())
	if err == nil {
		t.Fatal("GoAdapter.Check() expected error when seam returns non-checker manager, got nil")
	}
	if !strings.Contains(err.Error(), "go's manager dummy-mgr does not support per-package checks") {
		t.Errorf("GoAdapter.Check() error = %q, want contains expected message", err.Error())
	}
	if !errors.Is(err, adapters.ErrUnsupportedPackageCheck) {
		t.Errorf("GoAdapter.Check() error = %v, want errors.Is(err, adapters.ErrUnsupportedPackageCheck)", err)
	}

	res, err := (&GoAdapter{}).Update(context.Background(), false)
	if err == nil {
		t.Fatal("GoAdapter.Update() expected error when seam returns non-updater manager, got nil")
	}
	if !strings.Contains(err.Error(), "go's manager dummy-mgr does not support per-package updates") {
		t.Errorf("GoAdapter.Update() error = %q, want contains expected message", err.Error())
	}
	if !errors.Is(err, adapters.ErrUnsupportedPackageUpdate) {
		t.Errorf("GoAdapter.Update() error = %v, want errors.Is(err, adapters.ErrUnsupportedPackageUpdate)", err)
	}
	if res.Success {
		t.Errorf("GoAdapter.Update() Success = true, want false")
	}
}
