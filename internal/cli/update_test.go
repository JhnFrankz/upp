package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/adapters/official"
	"github.com/JhnFrankz/upp/internal/config"
	"github.com/JhnFrankz/upp/internal/output"
	"github.com/JhnFrankz/upp/internal/platform"
)

// fakeUpdateAdapter is a test double for the update gating matrix. It records
// whether Update was invoked — the behavioral signal the gating requirement
// is about — and lets each test control policy, trust, command, privileges,
// check result, and update outcome. The command/privileges fields drive the
// security-risk classification path (update.go) exactly like a real custom
// adapter's ToolInfo.
type fakeUpdateAdapter struct {
	name       string
	policy     adapters.UpdatePolicy
	trust      adapters.TrustLevel
	command    string
	privileges []string
	info       adapters.UpdateInfo
	checkErr   error
	updateErr  error
	result     adapters.Result
	updated    bool
	checkCount int  // Check() invocations — proves the no-double-check contract
	noDetect   bool // Detect() reports false (not installed)

	// Optional manager/ownership fields (WU3 group path). Setting kind=KindTool
	// + manager map lets the group enumeration find this adapter as an owned
	// tool; setting kind=KindManager lets it act as the manager in the batch.
	// The manager-behavior seams (checkPackage/updatePackage) are only invoked
	// when this adapter is the resolved manager in a group batch; when no
	// manager seam is wired, the default behaviors below keep the fake inert
	// for the standard per-tool tests.
	kind            adapters.Kind
	manager         map[string]string
	managerPackage  map[string]string
	checkPackage    func(pkg string) (adapters.UpdateInfo, error)
	updatePackage   func(pkg string) (adapters.Result, error)
	checkPkgCount   int
	updatePkgCount  int
	lastCheckPkg    string
	lastUpdatePkg   string
	updatePackageOn bool // UpdatePackage() ran (records the named package batch)
}

func (f *fakeUpdateAdapter) Name() string { return f.name }

func (f *fakeUpdateAdapter) Detect() bool { return !f.noDetect }

func (f *fakeUpdateAdapter) Check() (adapters.UpdateInfo, error) {
	f.checkCount++
	return f.info, f.checkErr
}

func (f *fakeUpdateAdapter) Update(dryRun bool) (adapters.Result, error) {
	f.updated = true
	return f.result, f.updateErr
}

func (f *fakeUpdateAdapter) Info() adapters.ToolInfo {
	return adapters.ToolInfo{
		ID:             f.name,
		Name:           f.name,
		Trust:          f.trust,
		UpdatePolicy:   f.policy,
		Command:        f.command,
		Privileges:     f.privileges,
		Kind:           f.kind,
		Manager:        f.manager,
		ManagerPackage: f.managerPackage,
	}
}

// CheckPackage runs the wired per-package availability seam, or defaults to
// current (no update) when unset — so a fake manager in a group batch reports
// no availability unless the test declares it.
func (f *fakeUpdateAdapter) CheckPackage(pkg string) (adapters.UpdateInfo, error) {
	f.checkPkgCount++
	f.lastCheckPkg = pkg
	if f.checkPackage != nil {
		return f.checkPackage(pkg)
	}
	return adapters.UpdateInfo{}, nil
}

// UpdatePackage runs the wired per-package updater seam, or defaults to
// success with no version change when unset.
func (f *fakeUpdateAdapter) UpdatePackage(pkg string) (adapters.Result, error) {
	f.updatePkgCount++
	f.lastUpdatePkg = pkg
	f.updatePackageOn = true
	if f.updatePackage != nil {
		return f.updatePackage(pkg)
	}
	return adapters.Result{Success: true, Before: "", After: ""}, nil
}

// fakeAdapterList returns a buildAdapterList seam yielding only the given
// fake adapters, hermetic and deterministic.
func fakeAdapterList(fakes ...*fakeUpdateAdapter) func(*config.Config, string) []adapters.Adapter {
	return func(*config.Config, string) []adapters.Adapter {
		result := make([]adapters.Adapter, 0, len(fakes))
		for _, f := range fakes {
			result = append(result, f)
		}
		return result
	}
}

// runUpdateWithFlags runs runUpdate with the given global/update flags
// against a single fake adapter in a hermetic HOME, returning the captured
// stdout and the runUpdate error. stdinIsTTY is pinned false: these legacy
// tests exercise the sequential (non-interactive) path, and the gate must
// not depend on the ambient test-runner stdin (design D2).
func runUpdateWithFlags(t *testing.T, fake *fakeUpdateAdapter, gf *GlobalFlags, uf *UpdateFlags) (string, error) {
	t.Helper()
	probeHome(t)
	deps := updateDeps{
		buildAdapterList: fakeAdapterList(fake),
		stdinIsTTY:       func() bool { return false },
	}
	var runErr error
	out := withCapturedStdout(func() {
		runErr = runUpdate(gf, uf, deps)
	})
	return out, runErr
}

// runUpdateWith runs runUpdate against a single fake adapter in a hermetic
// HOME, returning the captured stdout.
func runUpdateWith(t *testing.T, fake *fakeUpdateAdapter) string {
	t.Helper()
	out, err := runUpdateWithFlags(t, fake, &GlobalFlags{}, &UpdateFlags{})
	if err != nil {
		t.Errorf("runUpdate returned error: %v", err)
	}
	return out
}

// TestRunUpdate_GatingMatrix proves the Update Gating requirement (spec
// tool-adapter): update() runs for an adapter declaring PolicyGated (real
// update detection: apt, npm, nvm, pnpm) only when check() reported
// update_available=true; adapters declaring PolicyAlwaysUpdate — stubs like
// brew, winget/scoop, and custom adapters — always update, regardless of the
// check() result; a failed gated check is reported as failed, never current
// (design D2, matrix re-keyed from adapter ID to declared policy).
func TestRunUpdate_GatingMatrix(t *testing.T) {
	tests := []struct {
		name            string
		id              string
		policy          adapters.UpdatePolicy
		trust           adapters.TrustLevel
		updateAvailable bool
		checkErr        error
		wantUpdated     bool
		wantStatus      output.Status
	}{
		{
			name:            "gated apt with update available runs update",
			id:              "apt",
			policy:          adapters.PolicyGated,
			trust:           adapters.TrustOfficial,
			updateAvailable: true,
			wantUpdated:     true,
			wantStatus:      output.StatusUpdated,
		},
		{
			name:            "gated apt without update is reported current and update is skipped",
			id:              "apt",
			policy:          adapters.PolicyGated,
			trust:           adapters.TrustOfficial,
			updateAvailable: false,
			wantUpdated:     false,
			wantStatus:      output.StatusCurrent,
		},
		{
			name:            "always-update brew exempt: update still runs without update available",
			id:              "brew",
			policy:          adapters.PolicyAlwaysUpdate,
			trust:           adapters.TrustOfficial,
			updateAvailable: false,
			wantUpdated:     true,
			wantStatus:      output.StatusUpdated,
		},
		{
			name:            "always-update winget exempt: update still runs without update available",
			id:              "winget",
			policy:          adapters.PolicyAlwaysUpdate,
			trust:           adapters.TrustOfficial,
			updateAvailable: false,
			wantUpdated:     true,
			wantStatus:      output.StatusUpdated,
		},
		{
			name:            "custom trusted exempt: update still runs without update available",
			id:              "custom",
			policy:          adapters.PolicyAlwaysUpdate,
			trust:           adapters.TrustCustomTrusted,
			updateAvailable: false,
			wantUpdated:     true,
			wantStatus:      output.StatusUpdated,
		},
		{
			name:            "custom untrusted exempt: update still runs without update available",
			id:              "custom",
			policy:          adapters.PolicyAlwaysUpdate,
			trust:           adapters.TrustCustomUntrusted,
			updateAvailable: false,
			wantUpdated:     true,
			wantStatus:      output.StatusUpdated,
		},
		{
			name:        "gated check fails: reported failed, never current, update skipped",
			id:          "apt",
			policy:      adapters.PolicyGated,
			trust:       adapters.TrustOfficial,
			checkErr:    errors.New("check failed"),
			wantUpdated: false,
			wantStatus:  output.StatusFailed,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeUpdateAdapter{
				name:     tt.id,
				policy:   tt.policy,
				trust:    tt.trust,
				checkErr: tt.checkErr,
				info: adapters.UpdateInfo{
					CurrentVersion:  "1.0.0",
					LatestVersion:   "2.0.0",
					UpdateAvailable: tt.updateAvailable,
				},
				result: adapters.Result{Success: true, Before: "1.0.0", After: "2.0.0"},
			}
			out := runUpdateWith(t, fake)
			if fake.updated != tt.wantUpdated {
				t.Errorf("Update called = %v, want %v", fake.updated, tt.wantUpdated)
			}
			switch tt.wantStatus {
			case output.StatusUpdated:
				if !strings.Contains(out, "Updated: "+tt.id) {
					t.Errorf("output does not report tool as updated; got: %q", out)
				}
			case output.StatusCurrent:
				if strings.Contains(out, "Updated:") || strings.Contains(out, "Failed:") {
					t.Errorf("output reports tool as updated or failed, want current; got: %q", out)
				}
			case output.StatusFailed:
				if !strings.Contains(out, "Failed: "+tt.id) {
					t.Errorf("output does not report tool as failed; got: %q", out)
				}
			}
		})
	}
}

// --- WU2: gate resolution from owner (spec Update Gating — owned tool INERT) ---
//
// resolveEffectiveUpdatePolicy returns the UpdatePolicy that governs whether an
// adapter's Update() runs. For an owned tool (an official tool with a resolving
// manager, or a custom tool carrying an injected manager) the MANAGER's policy
// governs — the owned tool's own declared policy is INERT on the delegated
// path. The owner is resolved by OS platform key (the WU1 gotcha: platform
// constants, not runtime.GOOS).

// TestResolvingOwner proves resolvingOwner returns the manager adapter that
// owns a given adapter on a given OS, or nil when standalone (verify WARNING:
// cli.resolvingOwner 50% coverage). It exercises both branches directly: the
// CustomAdapter branch (a custom tool exposing its injected manager via
// ManagerAdapter) and the official branch (official.ResolveOwner keyed by
// platform constant). Since resolvingOwner is unexported, the test lives in
// package cli and drives it through the real adapters via the custom tool's
// injected manager.
func TestResolvingOwner(t *testing.T) {
	t.Run("custom adapter with injected manager returns the manager", func(t *testing.T) {
		brew := official.AdapterByName("brew")
		custom, err := adapters.NewCustomAdapter("mytool", "mytool", "", false, brew)
		if err != nil {
			t.Fatalf("NewCustomAdapter() error: %v", err)
		}
		if got := resolvingOwner(custom, platform.OSMacOS); got == nil || got.Name() != "brew" {
			t.Errorf("resolvingOwner(custom with brew manager) = %v, want brew adapter", got)
		}
	})

	t.Run("custom adapter without manager is standalone nil", func(t *testing.T) {
		custom, err := adapters.NewCustomAdapter("solo", "solo", "", false)
		if err != nil {
			t.Fatalf("NewCustomAdapter() error: %v", err)
		}
		if got := resolvingOwner(custom, platform.OSLinux); got != nil {
			t.Errorf("resolvingOwner(standalone custom) = %v, want nil", got)
		}
	})

	t.Run("official tool resolves via ResolveOwner", func(t *testing.T) {
		gh := official.AdapterByName("gh")
		if got := resolvingOwner(gh, platform.OSLinux); got == nil || got.Name() != "apt" {
			t.Errorf("resolvingOwner(gh, linux) = %v, want apt", got)
		}
		if got := resolvingOwner(gh, platform.OSMacOS); got == nil || got.Name() != "brew" {
			t.Errorf("resolvingOwner(gh, macos) = %v, want brew", got)
		}
	})

	t.Run("official standalone tool returns nil", func(t *testing.T) {
		npm := official.AdapterByName("npm")
		if got := resolvingOwner(npm, platform.OSLinux); got != nil {
			t.Errorf("resolvingOwner(npm, linux) = %v, want nil (standalone)", got)
		}
	})
}

// TestResolveEffectiveUpdatePolicy pins the INERT ownership semantics across
// every platform on any host: docker declares AlwaysUpdate but is owned by apt
// (Gated) on Linux and brew (Always) on macOS; gh is owned by apt (Gated) on
// Linux and brew (Always) on macOS; go owns nothing on Linux (standalone →
// own Always policy) but is owned by brew on macOS. Each case drives the
// effective policy directly, so the darwin branch is proven without running on
// a Mac.
func TestResolveEffectiveUpdatePolicy(t *testing.T) {
	tests := []struct {
		name string
		id   string
		os   string
		self adapters.UpdatePolicy
		want adapters.UpdatePolicy
	}{
		{"docker-linux-inherits-apt-gated", "docker", "linux", adapters.PolicyAlwaysUpdate, adapters.PolicyGated},
		{"docker-macos-inherits-brew-always", "docker", "macos", adapters.PolicyAlwaysUpdate, adapters.PolicyAlwaysUpdate},
		{"gh-linux-inherits-apt-gated", "gh", "linux", adapters.PolicyAlwaysUpdate, adapters.PolicyGated},
		{"gh-macos-inherits-brew-always", "gh", "macos", adapters.PolicyAlwaysUpdate, adapters.PolicyAlwaysUpdate},
		{"go-linux-standalone-own-always", "go", "linux", adapters.PolicyAlwaysUpdate, adapters.PolicyAlwaysUpdate},
		{"go-macos-inherits-brew-always", "go", "macos", adapters.PolicyAlwaysUpdate, adapters.PolicyAlwaysUpdate},
		{"npm-standalone-own-gated", "npm", "linux", adapters.PolicyGated, adapters.PolicyGated},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &fakeUpdateAdapter{name: tt.id, policy: tt.self}
			if got := resolveEffectiveUpdatePolicy(a, tt.os); got != tt.want {
				t.Errorf("resolveEffectiveUpdatePolicy(%q, %q) = %v, want %v", tt.id, tt.os, got, tt.want)
			}
		})
	}
}

// TestRunUpdate_OwnedToolInheritsGatedGate proves the INERT gate at the
// sequential flow level on a Linux host: an owned tool (docker) declaring
// PolicyAlwaysUpdate must be SKIPPED as current (never updated) when its
// resolving manager (apt, Gated) reports no update available. This is the
// "docker owned by apt (Gated) on Linux, apt check reports no update →
// delegated apt.Update() skipped; docker reported current" scenario.
func TestRunUpdate_OwnedToolInheritsGatedGate(t *testing.T) {
	probeHome(t)
	// docker's own Check reports no update (its design behavior); the manager
	// apt is Gated, so the gate skips docker even though docker's own declared
	// policy is AlwaysUpdate.
	docker := &fakeUpdateAdapter{
		name:   "docker",
		policy: adapters.PolicyAlwaysUpdate, // docker's own policy — INERT
		trust:  adapters.TrustOfficial,
		info: adapters.UpdateInfo{
			CurrentVersion:  "26.1.4",
			LatestVersion:   "26.1.4",
			UpdateAvailable: false, // docker reports no self-update availability
		},
		result: adapters.Result{Success: true, Before: "26.1.4", After: "26.1.4"},
	}
	deps := updateDeps{
		buildAdapterList: fakeAdapterList(docker),
		stdinIsTTY:       func() bool { return false },
	}
	out := withCapturedStdout(func() {
		if err := runUpdate(&GlobalFlags{}, &UpdateFlags{}, deps); err != nil {
			t.Errorf("runUpdate returned error: %v", err)
		}
	})
	if docker.updated {
		t.Error("owned docker must inherit apt (Gated) policy and NOT update when apt reports no update")
	}
	if !strings.Contains(out, "Up to date: docker") {
		t.Errorf("owned docker must be reported current; got: %q", out)
	}
}

// TestRunUpdate_CheckTimeoutStructuredError proves the check-timeout scenario
// (spec tool-adapter Subprocess Timeouts): a check() deadline marks the tool
// failed and the run continues updating other tools. The structured message
// itself is proven by TestTimeoutErr (the update flow's summary lists only
// tool names, not error text).
func TestRunUpdate_CheckTimeoutStructuredError(t *testing.T) {
	probeHome(t)
	hanging := &fakeUpdateAdapter{
		name:     "brew",
		policy:   adapters.PolicyAlwaysUpdate,
		trust:    adapters.TrustOfficial,
		checkErr: context.DeadlineExceeded,
	}
	ok := &fakeUpdateAdapter{
		name:   "npm",
		policy: adapters.PolicyGated,
		trust:  adapters.TrustOfficial,
		info: adapters.UpdateInfo{
			CurrentVersion:  "1.0.0",
			LatestVersion:   "2.0.0",
			UpdateAvailable: true,
		},
		result: adapters.Result{Success: true, Before: "1.0.0", After: "2.0.0"},
	}
	deps := updateDeps{
		buildAdapterList: func(*config.Config, string) []adapters.Adapter {
			return []adapters.Adapter{hanging, ok}
		},
		stdinIsTTY: func() bool { return false },
	}
	out := withCapturedStdout(func() {
		if err := runUpdate(&GlobalFlags{}, &UpdateFlags{}, deps); err != nil {
			t.Errorf("runUpdate returned error: %v", err)
		}
	})
	if !strings.Contains(out, "Failed: brew") {
		t.Errorf("output lacks failed result for brew; got: %q", out)
	}
	if !ok.updated {
		t.Error("other tools must still update after a check timeout")
	}
}

// TestTimeoutErr proves the structured timeout mapping (design D3, spec
// Subprocess Timeouts): a context deadline produces a tool/op/limit message
// naming the right limit per operation, errors.Is detection is preserved, and
// non-timeout errors pass through unchanged.
func TestTimeoutErr(t *testing.T) {
	tests := []struct {
		name string
		op   string
		want string
	}{
		{"check", "check", "brew check timed out after " + adapters.CheckTimeout.String()},
		{"update", "update", "brew update timed out after " + adapters.UpdateTimeout.String()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := timeoutErr("brew", tt.op, context.DeadlineExceeded)
			if !errors.Is(err, context.DeadlineExceeded) {
				t.Errorf("timeout error must preserve errors.Is(err, context.DeadlineExceeded); got: %v", err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error %q does not contain %q", err.Error(), tt.want)
			}
		})
	}
	t.Run("non-timeout error passes through unchanged", func(t *testing.T) {
		orig := fmt.Errorf("boom")
		if got := timeoutErr("brew", "update", orig); got != orig {
			t.Errorf("non-timeout error changed: got %v, want original %v", got, orig)
		}
	})
}

func TestUpdateCommand_DryRunShorthand(t *testing.T) {
	gf := &GlobalFlags{}
	cmd := NewUpdateCommand(gf)

	dryRunFlag := cmd.Flags().Lookup("dry-run")
	if dryRunFlag == nil || dryRunFlag.Shorthand != "n" {
		t.Fatalf("expected --dry-run to have shorthand -n, got %v", dryRunFlag)
	}

	err := cmd.ParseFlags([]string{"-n"})
	if err != nil {
		t.Fatalf("ParseFlags error: %v", err)
	}
	val, err := cmd.Flags().GetBool("dry-run")
	if err != nil || !val {
		t.Errorf("expected -n to set dry-run=true, got val=%v, err=%v", val, err)
	}
}

func TestRunUpdate_VerboseFailureDiagnostics(t *testing.T) {
	probeHome(t)
	failing := &fakeUpdateAdapter{
		name:      "broken-tool",
		policy:    adapters.PolicyAlwaysUpdate,
		trust:     adapters.TrustOfficial,
		updateErr: fmt.Errorf("permission denied reading /var/cache/apt"),
	}
	deps := updateDeps{
		buildAdapterList: func(*config.Config, string) []adapters.Adapter {
			return []adapters.Adapter{failing}
		},
		stdinIsTTY: func() bool { return false },
	}

	// With -v
	outVerbose := withCapturedStdout(func() {
		gf := &GlobalFlags{Verbose: true}
		_ = runUpdate(gf, &UpdateFlags{}, deps)
	})
	if !strings.Contains(outVerbose, "permission denied reading /var/cache/apt") {
		t.Errorf("expected verbose update to contain stderr diagnostic, got:\n%s", outVerbose)
	}

	// Without -v
	outDefault := withCapturedStdout(func() {
		gf := &GlobalFlags{Verbose: false}
		_ = runUpdate(gf, &UpdateFlags{}, deps)
	})
	if strings.Contains(outDefault, "permission denied reading /var/cache/apt") {
		t.Errorf("expected default update to suppress stderr diagnostic, got:\n%s", outDefault)
	}

	// With -q and -v
	outQuiet := withCapturedStdout(func() {
		gf := &GlobalFlags{Verbose: true, Quiet: true}
		_ = runUpdate(gf, &UpdateFlags{}, deps)
	})
	if strings.Contains(outQuiet, "permission denied reading /var/cache/apt") {
		t.Errorf("expected quiet mode to suppress stderr diagnostic, got:\n%s", outQuiet)
	}
}

// --- Phase 3: interactive selection (design D2/D4/D5/D7/D8) ---

// interactiveUpdateDeps builds an updateDeps with the Phase 3 seams set:
// TTY stdin, the injected selector (recording its options), and the given
// fake adapters. Zero selector return values exercise the production gate
// logic paths (design D2: the seam's return is authoritative).
func interactiveUpdateDeps(fakes []*fakeUpdateAdapter, selector func(opts []output.SelectOption) ([]string, bool)) updateDeps {
	return updateDeps{
		buildAdapterList: fakeAdapterList(fakes...),
		stdinIsTTY:       func() bool { return true },
		selector:         selector,
	}
}

// fakeSelector returns a selector seam that records the options it received
// and returns the given selection. The recorded options prove the pending
// set is rendered with ID/Label/Version from the pre-check outcomes (design
// D9: safeCheck's "Current → Latest" string is the per-row version).
func fakeSelector(selected []string, canceled bool) (func(opts []output.SelectOption) ([]string, bool), *[]output.SelectOption) {
	var got []output.SelectOption
	return func(opts []output.SelectOption) ([]string, bool) {
		got = opts
		return selected, canceled
	}, &got
}

// withStdin swaps os.Stdin for the duration of fn so security.ConfirmAction
// prompts (which default to os.Stdin) can be answered deterministically.
// Sequential-only, like withCapturedStdout: no t.Parallel exists in this
// package.
func withStdin(t *testing.T, input string, fn func()) {
	t.Helper()
	origStdin := os.Stdin
	r, w, _ := os.Pipe()
	_, _ = w.WriteString(input)
	_ = w.Close()
	os.Stdin = r
	defer func() { os.Stdin = origStdin }()
	fn()
}

// TestRunUpdate_SelectorGateMatrix proves the interactive gate (design D2,
// spec command-interface + ux-patterns): the selector runs ONLY when stdin
// is a TTY AND --ci, --quiet, and --dry-run are all unset. Every other
// combination keeps today's sequential non-interactive behavior — the
// selector seam must never be called.
func TestRunUpdate_SelectorGateMatrix(t *testing.T) {
	available := func() adapters.UpdateInfo {
		return adapters.UpdateInfo{
			CurrentVersion:  "1.0.0",
			LatestVersion:   "2.0.0",
			UpdateAvailable: true,
		}
	}
	pending := func() *fakeUpdateAdapter {
		return &fakeUpdateAdapter{
			name:   "npm",
			policy: adapters.PolicyGated,
			trust:  adapters.TrustOfficial,
			info:   available(),
			result: adapters.Result{Success: true, Before: "1.0.0", After: "2.0.0"},
		}
	}

	tests := []struct {
		name       string
		gf         *GlobalFlags
		uf         *UpdateFlags
		stdinIsTTY func() bool
		wantSelect bool // selector seam called?
	}{
		{"TTY plain update shows selector", &GlobalFlags{}, &UpdateFlags{}, func() bool { return true }, true},
		{"non-TTY skips selector", &GlobalFlags{}, &UpdateFlags{}, func() bool { return false }, false},
		{"--ci skips selector", &GlobalFlags{CI: true}, &UpdateFlags{}, func() bool { return true }, false},
		{"--quiet skips selector", &GlobalFlags{Quiet: true}, &UpdateFlags{}, func() bool { return true }, false},
		{"--dry-run skips selector", &GlobalFlags{}, &UpdateFlags{DryRun: true}, func() bool { return true }, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			probeHome(t)
			fake := pending()
			called := false
			deps := updateDeps{
				buildAdapterList: fakeAdapterList(fake),
				stdinIsTTY:       tt.stdinIsTTY,
				selector: func(opts []output.SelectOption) ([]string, bool) {
					called = true
					return nil, false
				},
			}
			withCapturedStdout(func() {
				if err := runUpdate(tt.gf, tt.uf, deps); err != nil {
					t.Errorf("runUpdate returned error: %v", err)
				}
			})
			if called != tt.wantSelect {
				t.Errorf("selector called = %v, want %v", called, tt.wantSelect)
			}
		})
	}
}

// TestRunUpdate_NoPendingSkipsSelector proves the no-pending-updates
// scenario (spec ux-patterns "No pending updates"): with only current or
// skipped tools, the selector is skipped and the normal summary is shown —
// the selector must not render an empty list.
func TestRunUpdate_NoPendingSkipsSelector(t *testing.T) {
	probeHome(t)
	current := &fakeUpdateAdapter{
		name:   "npm",
		policy: adapters.PolicyGated,
		trust:  adapters.TrustOfficial,
		info: adapters.UpdateInfo{
			CurrentVersion:  "1.0.0",
			LatestVersion:   "1.0.0",
			UpdateAvailable: false,
		},
	}
	skipped := &fakeUpdateAdapter{
		name:     "brew",
		policy:   adapters.PolicyAlwaysUpdate,
		trust:    adapters.TrustOfficial,
		noDetect: true,
	}

	called := false
	deps := updateDeps{
		buildAdapterList: fakeAdapterList(current, skipped),
		stdinIsTTY:       func() bool { return true },
		selector: func(opts []output.SelectOption) ([]string, bool) {
			called = true
			return nil, false
		},
	}
	out := withCapturedStdout(func() {
		if err := runUpdate(&GlobalFlags{}, &UpdateFlags{}, deps); err != nil {
			t.Errorf("runUpdate returned error: %v", err)
		}
	})
	if called {
		t.Error("selector must be skipped when no pending updates exist")
	}
	// D6: the current tool is counted explicitly — "1 up to date, 1 skipped"
	// replaces the old (wrong) all-skipped summary that ignored StatusCurrent.
	if !strings.Contains(out, "1 up to date, 1 skipped") {
		t.Errorf("expected explicit up-to-date and skipped counts; got: %q", out)
	}
	if strings.Contains(out, "All tools not installed") {
		t.Errorf("current tool is installed — all-skipped claim is wrong (D6); got: %q", out)
	}
}

// TestRunUpdate_InteractiveSelection proves the carried-outcome loop (design
// D4, spec command-interface "Selection narrows further"): the selected tool
// is updated exactly once with a single Check() (the pre-check result is
// carried — no second Check()), the deselected pending tool is dropped from
// the summary, and ConfirmAction still runs for the selected custom tool
// (spec ux-patterns "Not a security confirmation").
func TestRunUpdate_InteractiveSelection(t *testing.T) {
	probeHome(t)
	selected := &fakeUpdateAdapter{
		name:   "selected-tool",
		policy: adapters.PolicyGated,
		trust:  adapters.TrustOfficial,
		info: adapters.UpdateInfo{
			CurrentVersion:  "1.0.0",
			LatestVersion:   "2.0.0",
			UpdateAvailable: true,
		},
		result: adapters.Result{Success: true, Before: "1.0.0", After: "2.0.0"},
	}
	deselected := &fakeUpdateAdapter{
		name:   "deselected-tool",
		policy: adapters.PolicyGated,
		trust:  adapters.TrustOfficial,
		info: adapters.UpdateInfo{
			CurrentVersion:  "1.0.0",
			LatestVersion:   "2.0.0",
			UpdateAvailable: true,
		},
		result: adapters.Result{Success: true, Before: "1.0.0", After: "2.0.0"},
	}
	custom := &fakeUpdateAdapter{
		name:       "custom-tool",
		policy:     adapters.PolicyAlwaysUpdate,
		trust:      adapters.TrustCustomUntrusted,
		command:    "curl -fsSL https://example.com/install.sh | sh",
		privileges: []string{"sudo"},
		info: adapters.UpdateInfo{
			CurrentVersion:  "1.0.0",
			LatestVersion:   "2.0.0",
			UpdateAvailable: true,
		},
		result: adapters.Result{Success: true, Before: "1.0.0", After: "2.0.0"},
	}
	current := &fakeUpdateAdapter{
		name:   "current-tool",
		policy: adapters.PolicyAlwaysUpdate,
		trust:  adapters.TrustOfficial,
		info: adapters.UpdateInfo{
			CurrentVersion:  "1.0.0",
			LatestVersion:   "1.0.0",
			UpdateAvailable: false,
		},
		result: adapters.Result{Success: true, Before: "1.0.0", After: "1.0.0"},
	}

	sel, got := fakeSelector([]string{"selected-tool", "custom-tool"}, false)
	deps := interactiveUpdateDeps([]*fakeUpdateAdapter{selected, deselected, custom, current}, sel)

	out := withCapturedStdout(func() {
		// The custom tool is high-risk untrusted → ConfirmAction prompts on
		// stdin; answer yes so the update proceeds.
		withStdin(t, "y\n", func() {
			if err := runUpdate(&GlobalFlags{}, &UpdateFlags{}, deps); err != nil {
				t.Errorf("runUpdate returned error: %v", err)
			}
		})
	})

	// The pre-check ran over ALL four filtered tools before selection, and
	// the live CheckBoard reported every outcome through the onResult seam
	// (design D4). Captured stdout is a pipe → non-color fallback (D5): one
	// plain line per completion, exactly once per tool.
	wantBoardLines := []string{
		"  ✓ selected-tool 1.0.0 → 2.0.0",
		"  ✓ deselected-tool 1.0.0 → 2.0.0",
		"  ✓ custom-tool 1.0.0 → 2.0.0",
		"  ✓ current-tool up-to-date",
	}
	for _, line := range wantBoardLines {
		if n := strings.Count(out, line); n != 1 {
			t.Errorf("board line %q count = %d, want 1; got:\n%s", line, n, out)
		}
	}
	// Spec ux-patterns Progress Indication: the board replaces the old
	// "Checking X/Y" counter in the TTY pre-check — it must never return.
	if strings.Contains(out, "Checking") {
		t.Errorf("board must replace the old Checking X/Y counter; got: %q", out)
	}

	// Deselected tool: Update never called.
	if deselected.updated {
		t.Error("deselected tool must never be updated")
	}
	// Selected tools: updated exactly once.
	if !selected.updated {
		t.Error("selected tool must be updated")
	}
	if !custom.updated {
		t.Error("selected custom tool must be updated")
	}
	// D7: current always-update tools are NOT force-updated in interactive
	// TTY runs — only the pending selection is processed.
	if current.updated {
		t.Error("current always-update tool must not be force-updated in TTY mode (D7)")
	}
	// No-double-check contract: exactly one Check() per tool, carried forward.
	for _, f := range []*fakeUpdateAdapter{selected, deselected, custom, current} {
		if f.checkCount != 1 {
			t.Errorf("%s Check() count = %d, want 1 (carried outcome, no second check)", f.name, f.checkCount)
		}
	}
	// ConfirmAction still ran for the selected custom tool (selector is NOT
	// a security confirmation — the high-risk untrusted prompt appeared).
	if !strings.Contains(out, "Proceed? [y/N]") {
		t.Errorf("expected ConfirmAction prompt for selected custom tool; got: %q", out)
	}

	// Selector options carry ID/Label/Version from the pending pre-check
	// outcomes, in input order — only the pending (available) tools are
	// selectable (design D9: the "Current → Latest" string; D7: no
	// always-update tools in the list).
	wantOpts := []output.SelectOption{
		{ID: "selected-tool", Label: "selected-tool", Version: "1.0.0 → 2.0.0"},
		{ID: "deselected-tool", Label: "deselected-tool", Version: "1.0.0 → 2.0.0"},
		{ID: "custom-tool", Label: "custom-tool", Version: "1.0.0 → 2.0.0"},
	}
	if len(*got) != len(wantOpts) {
		t.Fatalf("selector options = %d, want %d: %+v", len(*got), len(wantOpts), *got)
	}
	for i, want := range wantOpts {
		if (*got)[i] != want {
			t.Errorf("selector option[%d] = %+v, want %+v", i, (*got)[i], want)
		}
	}

	// Summary reflects executed selection: deselected pending tool dropped,
	// both selected tools updated — counts reflect executed selection, not
	// the pending set.
	if !strings.Contains(out, "2 updated") {
		t.Errorf("expected summary '2 updated'; got: %q", out)
	}
	if !strings.Contains(out, "Updated: selected-tool, custom-tool") {
		t.Errorf("expected detail line listing only the selected tools; got: %q", out)
	}
	if strings.Contains(out, "Updated: current-tool") {
		t.Errorf("current always-update tool must not appear as updated; got: %q", out)
	}
}

// TestRunUpdate_SelectorCancel proves the cancel path (spec ux-patterns "Esc
// cancels run"/"q cancels run", design D8): nothing is updated, the fixed
// cancel message is shown, and the run exits 0.
func TestRunUpdate_SelectorCancel(t *testing.T) {
	probeHome(t)
	pending := &fakeUpdateAdapter{
		name:   "npm",
		policy: adapters.PolicyGated,
		trust:  adapters.TrustOfficial,
		info: adapters.UpdateInfo{
			CurrentVersion:  "1.0.0",
			LatestVersion:   "2.0.0",
			UpdateAvailable: true,
		},
		result: adapters.Result{Success: true, Before: "1.0.0", After: "2.0.0"},
	}
	always := &fakeUpdateAdapter{
		name:   "brew",
		policy: adapters.PolicyAlwaysUpdate,
		trust:  adapters.TrustOfficial,
		info: adapters.UpdateInfo{
			CurrentVersion:  "1.0.0",
			LatestVersion:   "1.0.0",
			UpdateAvailable: false,
		},
		result: adapters.Result{Success: true, Before: "1.0.0", After: "1.0.0"},
	}

	sel, _ := fakeSelector(nil, true)
	deps := interactiveUpdateDeps([]*fakeUpdateAdapter{pending, always}, sel)

	out := withCapturedStdout(func() {
		if err := runUpdate(&GlobalFlags{}, &UpdateFlags{}, deps); err != nil {
			t.Errorf("cancel must exit 0; runUpdate returned error: %v", err)
		}
	})

	// Pre-check completed before the cancel, reported through the live board
	// (design D4; non-color fallback in captured stdout, D5).
	for _, line := range []string{
		"  ✓ npm 1.0.0 → 2.0.0",
		"  ✓ brew up-to-date",
	} {
		if n := strings.Count(out, line); n != 1 {
			t.Errorf("board line %q count = %d, want 1; got:\n%s", line, n, out)
		}
	}
	if strings.Contains(out, "Checking") {
		t.Errorf("board must replace the old Checking X/Y counter; got: %q", out)
	}
	if !strings.Contains(out, "Update canceled — no changes made.") {
		t.Errorf("expected the fixed cancel message; got: %q", out)
	}
	// Nothing updated — not even always-update tools (design D7: the
	// interactive run acts on the pending selection only).
	if pending.updated {
		t.Error("cancel must not update pending tools")
	}
	if always.updated {
		t.Error("cancel must not update always-update tools (D7: no force-update in TTY)")
	}
}

// dryRunFakes builds n current tools plus named not-installed (skipped) tools
// for the dry-run summary scenarios.
func dryRunFakes(currentCount int, skippedNames ...string) []*fakeUpdateAdapter {
	mk := func(name string, installed bool) *fakeUpdateAdapter {
		f := &fakeUpdateAdapter{
			name:   name,
			policy: adapters.PolicyGated,
			trust:  adapters.TrustOfficial,
			info: adapters.UpdateInfo{
				CurrentVersion:  "1.0.0",
				LatestVersion:   "1.0.0",
				UpdateAvailable: false,
			},
			result: adapters.Result{Success: true, Before: "1.0.0", After: "1.0.0"},
		}
		f.noDetect = !installed
		return f
	}
	fakes := make([]*fakeUpdateAdapter, 0, currentCount+len(skippedNames))
	for i := 1; i <= currentCount; i++ {
		fakes = append(fakes, mk(fmt.Sprintf("tool-%d", i), true))
	}
	for _, name := range skippedNames {
		fakes = append(fakes, mk(name, false))
	}
	return fakes
}

// TestRunUpdate_DryRunCurrentWithSkips pins the spec ux-patterns Summary
// Report scenario "Up-to-date with skips": upp update --dry-run over 8
// current and 2 not-installed tools reports "8 up to date, 2 skipped" and
// never claims "All tools up to date." or "All clean!".
func TestRunUpdate_DryRunCurrentWithSkips(t *testing.T) {
	probeHome(t)
	deps := interactiveUpdateDeps(dryRunFakes(8, "missing-a", "missing-b"), nil)

	out := withCapturedStdout(func() {
		if err := runUpdate(&GlobalFlags{}, &UpdateFlags{DryRun: true}, deps); err != nil {
			t.Errorf("runUpdate returned error: %v", err)
		}
	})

	if !strings.Contains(out, "8 up to date, 2 skipped") {
		t.Errorf("expected explicit '8 up to date, 2 skipped' summary; got:\n%s", out)
	}
	if strings.Contains(out, "All tools up to date.") {
		t.Errorf("summary must never claim 'All tools up to date.' when a tool was skipped; got:\n%s", out)
	}
	if strings.Contains(out, "All clean!") {
		t.Errorf("dry-run summary must never claim 'All clean!'; got:\n%s", out)
	}
	if strings.Contains(out, "not installed. Nothing to do") {
		t.Errorf("current tools are installed — 'not installed' claim is wrong (D6); got:\n%s", out)
	}
}

// TestRunUpdate_DryRunPendingNeverClean proves the pending-only dry-run path
// reports "N would update" explicitly and never prints "All clean!" while
// updates are pending (spec ux-patterns Summary Report, D3).
func TestRunUpdate_DryRunPendingNeverClean(t *testing.T) {
	probeHome(t)
	pending := &fakeUpdateAdapter{
		name:   "npm",
		policy: adapters.PolicyGated,
		trust:  adapters.TrustOfficial,
		info: adapters.UpdateInfo{
			CurrentVersion:  "1.0.0",
			LatestVersion:   "2.0.0",
			UpdateAvailable: true,
		},
		result: adapters.Result{Success: true, Before: "1.0.0", After: "2.0.0"},
	}
	deps := interactiveUpdateDeps([]*fakeUpdateAdapter{pending}, nil)

	out := withCapturedStdout(func() {
		if err := runUpdate(&GlobalFlags{}, &UpdateFlags{DryRun: true}, deps); err != nil {
			t.Errorf("runUpdate returned error: %v", err)
		}
	})

	if !strings.Contains(out, "would update") {
		t.Errorf("pending-only dry-run must report pending updates explicitly; got:\n%s", out)
	}
	if strings.Contains(out, "All clean!") {
		t.Errorf("pending-only path must never print 'All clean!'; got:\n%s", out)
	}
}

// TestRunUpdate_ManagerSelfUpdateDryRun pins the manager self-update row
// rendering contract (spec ux-patterns "Manager Self-Update Row Rendering"):
// brew — which reports no self-update availability signal by design
// (UpdateAvailable=false) — renders as CURRENT in `upp update -n` (there is
// no `-n` signal to plan); apt and winget — which report real availability —
// render as PLANNED ("would update") in `-n`. The dry-run summary must never
// pair "All clean!" with a pending manager update.
func TestRunUpdate_ManagerSelfUpdateDryRun(t *testing.T) {
	probeHome(t)
	brewCurrent := &fakeUpdateAdapter{
		name:   "brew",
		policy: adapters.PolicyAlwaysUpdate,
		trust:  adapters.TrustOfficial,
		info: adapters.UpdateInfo{
			CurrentVersion:  "4.1.0",
			LatestVersion:   "4.1.0",
			UpdateAvailable: false, // brew current-only by design
		},
		result: adapters.Result{Success: true, Before: "4.1.0", After: "4.1.0"},
	}
	aptPending := &fakeUpdateAdapter{
		name:   "apt",
		policy: adapters.PolicyGated,
		trust:  adapters.TrustOfficial,
		info: adapters.UpdateInfo{
			CurrentVersion:  "2.4.0",
			LatestVersion:   "2.4.5",
			UpdateAvailable: true,
		},
		result: adapters.Result{Success: true, Before: "2.4.0", After: "2.4.5"},
	}
	wingetPending := &fakeUpdateAdapter{
		name:   "winget",
		policy: adapters.PolicyAlwaysUpdate,
		trust:  adapters.TrustOfficial,
		info: adapters.UpdateInfo{
			CurrentVersion:  "v1.8.2301",
			LatestVersion:   "v1.8.2311",
			UpdateAvailable: true,
		},
		result: adapters.Result{Success: true, Before: "v1.8.2301", After: "v1.8.2311"},
	}

	deps := interactiveUpdateDeps([]*fakeUpdateAdapter{brewCurrent, aptPending, wingetPending}, nil)
	out := withCapturedStdout(func() {
		if err := runUpdate(&GlobalFlags{}, &UpdateFlags{DryRun: true}, deps); err != nil {
			t.Errorf("runUpdate returned error: %v", err)
		}
	})

	// brew: current, never "would update" (no -n signal exists for it).
	if !strings.Contains(out, "up to date") {
		t.Errorf("brew must render current in -n; got:\n%s", out)
	}
	if !strings.Contains(out, "Up to date: brew") {
		t.Errorf("brew must be listed as up to date in -n; got:\n%s", out)
	}
	// brew must never render as a planned (available) self-update action.
	if strings.Contains(out, "[available] brew") || strings.Contains(out, "Dry run — no changes") && strings.Contains(out, "brew (") {
		t.Errorf("brew must never be 'would update' in -n (current-only by design); got:\n%s", out)
	}
	// apt + winget: planned self-update actions reported explicitly.
	if !strings.Contains(out, "2 would update") {
		t.Errorf("pending apt+winget must report '2 would update'; got:\n%s", out)
	}
	// Dry-run with a pending manager update must never claim "All clean!".
	if strings.Contains(out, "All clean!") {
		t.Errorf("'All clean!' may not be paired with a pending manager update in -n; got:\n%s", out)
	}
}

// TestRunUpdate_ManagerSelfUpdateBrewNeverSelector pins the interactive TTY
// selector contract (spec ux-patterns "Manager Self-Update Row Rendering"):
// brew — which reports no self-update availability by design — MUST NOT appear
// in the pending CheckboxSelector, because self-updates run only via the
// sequential/`--ci` PolicyAlwaysUpdate path. apt and winget, which report real
// availability, DO appear. Brew must never force-update in a TTY run (design
// D7).
func TestRunUpdate_ManagerSelfUpdateBrewNeverSelector(t *testing.T) {
	probeHome(t)
	brewCurrent := &fakeUpdateAdapter{
		name:   "brew",
		policy: adapters.PolicyAlwaysUpdate,
		trust:  adapters.TrustOfficial,
		info: adapters.UpdateInfo{
			CurrentVersion:  "4.1.0",
			LatestVersion:   "4.1.0",
			UpdateAvailable: false,
		},
		result: adapters.Result{Success: true, Before: "4.1.0", After: "4.1.0"},
	}
	aptPending := &fakeUpdateAdapter{
		name:   "apt",
		policy: adapters.PolicyGated,
		trust:  adapters.TrustOfficial,
		info: adapters.UpdateInfo{
			CurrentVersion:  "2.4.0",
			LatestVersion:   "2.4.5",
			UpdateAvailable: true,
		},
		result: adapters.Result{Success: true, Before: "2.4.0", After: "2.4.5"},
	}
	wingetPending := &fakeUpdateAdapter{
		name:   "winget",
		policy: adapters.PolicyAlwaysUpdate,
		trust:  adapters.TrustOfficial,
		info: adapters.UpdateInfo{
			CurrentVersion:  "v1.8.2301",
			LatestVersion:   "v1.8.2311",
			UpdateAvailable: true,
		},
		result: adapters.Result{Success: true, Before: "v1.8.2301", After: "v1.8.2311"},
	}

	sel, got := fakeSelector([]string{"apt", "winget"}, false)
	deps := interactiveUpdateDeps([]*fakeUpdateAdapter{brewCurrent, aptPending, wingetPending}, sel)
	out := withCapturedStdout(func() {
		if err := runUpdate(&GlobalFlags{}, &UpdateFlags{}, deps); err != nil {
			t.Errorf("runUpdate returned error: %v", err)
		}
	})

	// Selector options: only the pending managers (apt + winget); brew absent.
	wantOpts := []output.SelectOption{
		{ID: "apt", Label: "apt", Version: "2.4.0 → 2.4.5"},
		{ID: "winget", Label: "winget", Version: "v1.8.2301 → v1.8.2311"},
	}
	if len(*got) != len(wantOpts) {
		t.Fatalf("selector options = %d, want %d: %+v", len(*got), len(wantOpts), *got)
	}
	for i, want := range wantOpts {
		if (*got)[i] != want {
			t.Errorf("selector option[%d] = %+v, want %+v", i, (*got)[i], want)
		}
	}
	// brew must never be force-updated in a TTY interactive run (design D7).
	if brewCurrent.updated {
		t.Error("brew must never be force-updated in TTY mode (D7): only the pending selection is processed")
	}
	// The board should show brew current and apt/winget available.
	if !strings.Contains(out, "brew up-to-date") {
		t.Errorf("board must show brew current; got:\n%s", out)
	}
}

// TestRunUpdate_AllSucceedSummary pins the ux-patterns Summary Report
// "All succeed" scenario end to end: a sequential (non-TTY) update run that
// updates every tool ends with the explicit-counts clean line
// ("N updated, 0 failed. All clean!").
func TestRunUpdate_AllSucceedSummary(t *testing.T) {
	probeHome(t)
	updated := &fakeUpdateAdapter{
		name:   "brew",
		policy: adapters.PolicyAlwaysUpdate,
		trust:  adapters.TrustOfficial,
		info: adapters.UpdateInfo{
			CurrentVersion:  "1.0.0",
			LatestVersion:   "2.0.0",
			UpdateAvailable: true,
		},
		result: adapters.Result{Success: true, Before: "1.0.0", After: "2.0.0"},
	}
	deps := updateDeps{
		buildAdapterList: fakeAdapterList(updated),
		stdinIsTTY:       func() bool { return false },
	}

	out := withCapturedStdout(func() {
		if err := runUpdate(&GlobalFlags{}, &UpdateFlags{}, deps); err != nil {
			t.Errorf("runUpdate returned error: %v", err)
		}
	})

	if !strings.Contains(out, "1 updated, 0 failed. All clean!") {
		t.Errorf("all-succeed run must report '1 updated, 0 failed. All clean!', got:\n%s", out)
	}
}

// TestProcessSelectedOutcome_Coverage directly exercises
// processSelectedOutcome across its decision branches (verify SUGGESTION:
// interactive-path coverage). Each case drives the security ConfirmAction
// decision via trust/risk/CI configuration, the policy gate via UpdateInfo,
// and the update outcome via the fake's Result. The interactive prompt case
// injects stdin via withStdin so the Deny decision is deterministic.
func TestProcessSelectedOutcome_Coverage(t *testing.T) {
	newInfo := func(available bool) adapters.UpdateInfo {
		return adapters.UpdateInfo{
			CurrentVersion:  "1.0.0",
			LatestVersion:   "2.0.0",
			UpdateAvailable: available,
		}
	}

	tests := []struct {
		name        string
		fake        *fakeUpdateAdapter
		gf          *GlobalFlags
		updateInfo  adapters.UpdateInfo
		stdin       string // non-empty → wrap the case in withStdin (prompt answer)
		wantStatus  output.Status
		wantFailed  bool
		wantUpdated bool
	}{
		{
			name: "update success official",
			fake: &fakeUpdateAdapter{
				name:   "ok-tool",
				policy: adapters.PolicyAlwaysUpdate,
				trust:  adapters.TrustOfficial,
				result: adapters.Result{Success: true, Before: "1.0.0", After: "2.0.0"},
			},
			updateInfo:  newInfo(true),
			wantStatus:  output.StatusUpdated,
			wantUpdated: true,
		},
		{
			name: "update failure surfaces failed",
			fake: &fakeUpdateAdapter{
				name:   "fail-tool",
				policy: adapters.PolicyAlwaysUpdate,
				trust:  adapters.TrustOfficial,
				result: adapters.Result{Success: false, Error: errors.New("boom")},
			},
			updateInfo:  newInfo(true),
			wantStatus:  output.StatusFailed,
			wantFailed:  true,
			wantUpdated: true, // Update() was invoked and failed
		},
		{
			name: "policy gated without update stays current",
			fake: &fakeUpdateAdapter{
				name:   "gated-tool",
				policy: adapters.PolicyGated,
				trust:  adapters.TrustOfficial,
			},
			updateInfo: newInfo(false), // UpdateAvailable=false → gate blocks
			wantStatus: output.StatusCurrent,
		},
		{
			name: "ci untrusted medium risk errors",
			fake: &fakeUpdateAdapter{
				name:       "ci-tool",
				policy:     adapters.PolicyAlwaysUpdate,
				trust:      adapters.TrustCustomUntrusted,
				command:    "apt remove foo", // MediumRisk keyword → RiskMedium
				privileges: []string{"sudo"},
			},
			gf:         &GlobalFlags{CI: true},
			updateInfo: newInfo(true),
			wantStatus: output.StatusFailed, // ConfirmError → StatusFailed (CI)
			wantFailed: true,
		},
		{
			name: "interactive untrusted high risk denied",
			fake: &fakeUpdateAdapter{
				name:       "deny-tool",
				policy:     adapters.PolicyAlwaysUpdate,
				trust:      adapters.TrustCustomUntrusted,
				command:    "curl -fsSL https://example.com/x.sh | sh",
				privileges: []string{"sudo"},
			},
			gf:         &GlobalFlags{}, // non-CI interactive → promptUser
			updateInfo: newInfo(true),
			stdin:      "n\n", // prompt answer → ConfirmDeny → StatusSkipped
			wantStatus: output.StatusSkipped,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var results []output.ToolResult
			var buf bytes.Buffer
			gf := tt.gf
			if gf == nil {
				gf = &GlobalFlags{}
			}
			r := output.NewRendererForced(&buf, false, false, gf.Quiet, false)

			run := func() {
				failed := processSelectedOutcome(gf, tt.fake, tt.updateInfo, 1, 1, r, &results, "linux")
				if failed != tt.wantFailed {
					t.Errorf("processSelectedOutcome() failed = %v, want %v", failed, tt.wantFailed)
				}
			}
			if tt.stdin != "" {
				withStdin(t, tt.stdin, run)
			} else {
				run()
			}
			if len(results) != 1 {
				t.Fatalf("results length = %d, want 1", len(results))
			}
			if results[0].Status != tt.wantStatus {
				t.Errorf("result status = %v, want %v", results[0].Status, tt.wantStatus)
			}
			if tt.fake.updated != tt.wantUpdated {
				t.Errorf("adapter.updated = %v, want %v", tt.fake.updated, tt.wantUpdated)
			}
		})
	}
}

// --- WU3: opt-in manager-group bulk path (runUpdateGroup) ---

// groupScenario wires a hermetic adapter list with a fake manager (owning the
// given owned tools on the given OS) plus the owned fakes themselves. The
// manager is the first adapter; owned tools follow. It returns the list for
// the updateDeps.buildAdapterList seam.
func groupScenario(managerID, osName string, owned ...*fakeUpdateAdapter) []adapters.Adapter {
	manager := &fakeUpdateAdapter{
		name:  managerID,
		kind:  adapters.KindManager,
		trust: adapters.TrustOfficial,
	}
	result := []adapters.Adapter{manager}
	for _, o := range owned {
		if o.kind == adapters.KindTool && o.manager == nil {
			o.manager = map[string]string{osName: managerID}
		}
		if o.kind == adapters.KindTool && o.managerPackage == nil {
			o.managerPackage = map[string]string{osName: o.name}
		}
		result = append(result, o)
	}
	return result
}

// runUpdateGroupWith runs runUpdateGroup directly over a hermetic adapter
// list on the given osName, returning captured stdout and the error. Calling
// runUpdateGroup (not runUpdate) lets each test pass its own osName, so a
// brew/macos scenario is proven on any host (the ownership registry is keyed
// by platform constant, not runtime.GOOS). stdin, when non-empty, is swapped
// into os.Stdin for the duration so a sudo group prompt can be answered.
func runUpdateGroupWith(t *testing.T, gf *GlobalFlags, manager, osName string, stdin string, adapterList []adapters.Adapter) (string, error) {
	t.Helper()
	probeHome(t)
	var runErr error
	out := withCapturedStdout(func() {
		runGroup := func() {
			runErr = runUpdateGroup(gf, &UpdateFlags{Manager: manager}, adapterList, osName)
		}
		if stdin != "" {
			withStdin(t, stdin, runGroup)
		} else {
			runGroup()
		}
	})
	return out, runErr
}

// runUpdateManagerFlag runs runUpdate with the --manager flag against a
// hermetic adapter list, returning captured stdout and the error. It proves
// the flag ROUTING (task 3.3): uf.Manager != "" dispatches to runUpdateGroup.
// It uses the host platform OS, so it is only used for host-native scenarios
// (apt on linux). stdin, when non-empty, is swapped into os.Stdin so a sudo
// group prompt can be answered.
func runUpdateManagerFlag(t *testing.T, gf *GlobalFlags, manager, stdin string, adapterList []adapters.Adapter) (string, error) {
	t.Helper()
	probeHome(t)
	deps := updateDeps{
		buildAdapterList: func(*config.Config, string) []adapters.Adapter {
			return adapterList
		},
		stdinIsTTY: func() bool { return false },
	}
	var runErr error
	out := withCapturedStdout(func() {
		run := func() {
			runErr = runUpdate(gf, &UpdateFlags{Manager: manager}, deps)
		}
		if stdin != "" {
			withStdin(t, stdin, run)
		} else {
			run()
		}
	})
	return out, runErr
}

// TestRunUpdate_GroupOptInFlagTriggersGroup proves task 3.3 (spec bulk-update
// "Opt-in triggers group" / command-interface "Manager triggers group"): when
// --manager is set, runUpdate routes to the group path and enumerates the
// manager's owned tools. A bare upp update (Manager empty) never does.
func TestRunUpdate_GroupOptInFlagTriggersGroup(t *testing.T) {
	t.Run("manager set triggers the apt group", func(t *testing.T) {
		gh := &fakeUpdateAdapter{name: "gh", kind: adapters.KindTool, policy: adapters.PolicyAlwaysUpdate, trust: adapters.TrustOfficial}
		// gh owned by apt on linux
		gh.manager = map[string]string{"linux": "apt"}
		gh.managerPackage = map[string]string{"linux": "gh"}
		list := groupScenario("apt", "linux", gh)
		// manager is index 0 (apt); wire its per-package availability + updater.
		apt := list[0].(*fakeUpdateAdapter)
		apt.checkPackage = func(pkg string) (adapters.UpdateInfo, error) {
			return adapters.UpdateInfo{CurrentVersion: "2.45.0", LatestVersion: "2.46.0", UpdateAvailable: true}, nil
		}
		apt.updatePackage = func(pkg string) (adapters.Result, error) {
			return adapters.Result{Success: true, Before: "2.45.0", After: "2.46.0"}, nil
		}
		// Routing test goes through runUpdate (host OS = linux, apt-native).
		// The apt group command is sudo → RiskHigh → prompt; answer yes so the
		// group proceeds (this is the flag-routing proof, not the security one).
		out, err := runUpdateManagerFlag(t, &GlobalFlags{}, "apt", "y\n", list)
		if err != nil {
			t.Fatalf("runUpdate group error: %v", err)
		}
		if !strings.Contains(out, "gh updated") {
			t.Errorf("group path must update gh via apt; got:\n%s", out)
		}
		if apt.lastUpdatePkg != "gh" {
			t.Errorf("apt UpdatePackage must run for gh, got %q", apt.lastUpdatePkg)
		}
	})

	t.Run("bare update never triggers a group", func(t *testing.T) {
		gh := &fakeUpdateAdapter{name: "gh", kind: adapters.KindTool, policy: adapters.PolicyAlwaysUpdate, trust: adapters.TrustOfficial}
		deps := updateDeps{
			buildAdapterList: fakeAdapterList(gh),
			stdinIsTTY:       func() bool { return false },
		}
		// uf.Manager is empty → runUpdate runs the STANDARD sequential path,
		// which never calls the group path (gh.detect/check/update, no group).
		_ = withCapturedStdout(func() {
			if err := runUpdate(&GlobalFlags{}, &UpdateFlags{}, deps); err != nil {
				t.Errorf("bare update error: %v", err)
			}
		})
		if gh.updatePackageOn {
			t.Error("bare update must never run the group UpdatePackage path")
		}
	})
}

// TestRunUpdate_GroupSkipExcludesOwnedTool proves spec bulk-update "Skip
// excludes owned tool": `upp update --manager apt --skip docker` batches only
// gh (docker excluded).
func TestRunUpdate_GroupSkipExcludesOwnedTool(t *testing.T) {
	gh := &fakeUpdateAdapter{name: "gh", kind: adapters.KindTool, policy: adapters.PolicyAlwaysUpdate, trust: adapters.TrustOfficial}
	docker := &fakeUpdateAdapter{name: "docker", kind: adapters.KindTool, policy: adapters.PolicyAlwaysUpdate, trust: adapters.TrustOfficial}
	list := groupScenario("apt", "linux", gh, docker)
	apt := list[0].(*fakeUpdateAdapter)
	apt.checkPackage = func(pkg string) (adapters.UpdateInfo, error) {
		return adapters.UpdateInfo{CurrentVersion: "2.45.0", LatestVersion: "2.46.0", UpdateAvailable: true}, nil
	}
	apt.updatePackage = func(pkg string) (adapters.Result, error) {
		return adapters.Result{Success: true, Before: "2.45.0", After: "2.46.0"}, nil
	}

	gf := &GlobalFlags{Skip: "docker"}
	out, err := runUpdateGroupWith(t, gf, "apt", "linux", "y\n", list)
	if err != nil {
		t.Fatalf("runUpdate group error: %v", err)
	}
	if !strings.Contains(out, "gh updated") {
		t.Errorf("gh must be updated; got:\n%s", out)
	}
	if strings.Contains(out, "docker") {
		t.Errorf("--skip docker must exclude docker from the group batch; got:\n%s", out)
	}
	if apt.lastUpdatePkg != "gh" {
		t.Errorf("apt must only update gh (skipped docker), got %q", apt.lastUpdatePkg)
	}
}

// TestRunUpdate_GroupGatedBlocksAndRuns proves design D5 / spec bulk-update
// "Group Gate Inheritance (Gated)": a PolicyGated manager (apt) gates the
// whole group on group availability (any owned package has an update) — blocked
// when none available, runs when at least one is available.
func TestRunUpdate_GroupGatedBlocksAndRuns(t *testing.T) {
	newApt := func(policy adapters.UpdatePolicy, avail bool) (*fakeUpdateAdapter, *fakeUpdateAdapter) {
		gh := &fakeUpdateAdapter{name: "gh", kind: adapters.KindTool, policy: adapters.PolicyAlwaysUpdate, trust: adapters.TrustOfficial}
		gh.manager = map[string]string{"linux": "apt"}
		gh.managerPackage = map[string]string{"linux": "gh"}
		apt := &fakeUpdateAdapter{name: "apt", kind: adapters.KindManager, policy: policy, trust: adapters.TrustOfficial}
		apt.checkPackage = func(pkg string) (adapters.UpdateInfo, error) {
			return adapters.UpdateInfo{CurrentVersion: "2.45.0", LatestVersion: "2.46.0", UpdateAvailable: avail}, nil
		}
		apt.updatePackage = func(pkg string) (adapters.Result, error) {
			return adapters.Result{Success: true, Before: "2.45.0", After: "2.46.0"}, nil
		}
		return apt, gh
	}

	t.Run("gated group blocks when no owned package has an update", func(t *testing.T) {
		apt, gh := newApt(adapters.PolicyGated, false)
		out, err := runUpdateGroupWith(t, &GlobalFlags{}, "apt", "linux", "", []adapters.Adapter{apt, gh})
		if err != nil {
			t.Fatalf("gated block error: %v", err)
		}
		if gh.updatePackageOn {
			t.Error("gated group with no availability must NOT run any package update")
		}
		if !strings.Contains(out, "gh current") {
			t.Errorf("gated block must report owned tool current; got:\n%s", out)
		}
	})

	t.Run("gated group runs when gh has an update", func(t *testing.T) {
		apt, gh := newApt(adapters.PolicyGated, true)
		// sudo apt group command is RiskHigh → prompt; answer yes so it proceeds.
		out, err := runUpdateGroupWith(t, &GlobalFlags{}, "apt", "linux", "y\n", []adapters.Adapter{apt, gh})
		if err != nil {
			t.Fatalf("gated run error: %v", err)
		}
		if apt.lastUpdatePkg != "gh" {
			t.Error("gated group with an available package must run the apt UpdatePackage for gh")
		}
		if !strings.Contains(out, "gh updated") {
			t.Errorf("gated run must update gh; got:\n%s", out)
		}
	})

	t.Run("always-update group runs regardless of check", func(t *testing.T) {
		brew := &fakeUpdateAdapter{name: "brew", kind: adapters.KindManager, policy: adapters.PolicyAlwaysUpdate, trust: adapters.TrustOfficial}
		gh := &fakeUpdateAdapter{name: "gh", kind: adapters.KindTool, policy: adapters.PolicyAlwaysUpdate, trust: adapters.TrustOfficial}
		gh.manager = map[string]string{"macos": "brew"}
		gh.managerPackage = map[string]string{"macos": "gh"}
		// brew AlwaysUpdate: group runs even though gh reports NO availability.
		brew.checkPackage = func(pkg string) (adapters.UpdateInfo, error) {
			return adapters.UpdateInfo{CurrentVersion: "2.45.0", LatestVersion: "2.45.0", UpdateAvailable: false}, nil
		}
		brew.updatePackage = func(pkg string) (adapters.Result, error) {
			return adapters.Result{Success: true, Before: "2.45.0", After: "2.45.0"}, nil
		}
		out, err := runUpdateGroupWith(t, &GlobalFlags{}, "brew", "macos", "", []adapters.Adapter{brew, gh})
		if err != nil {
			t.Fatalf("always group error: %v", err)
		}
		if brew.lastUpdatePkg != "gh" {
			t.Error("AlwaysUpdate group must run its package update regardless of check result")
		}
		if !strings.Contains(out, "gh updated") {
			t.Errorf("always group must update gh; got:\n%s", out)
		}
	})
}

// TestRunUpdate_GroupCheckFailed proves spec bulk-update "Check fails": when the
// manager's CheckPackage returns an error for an owned tool, the group reports
// that tool as "check failed" (never current nor update available), does NOT
// run its UpdatePackage, and the group continues (does not abort).
func TestRunUpdate_GroupCheckFailed(t *testing.T) {
	gh := &fakeUpdateAdapter{name: "gh", kind: adapters.KindTool, policy: adapters.PolicyAlwaysUpdate, trust: adapters.TrustOfficial}
	gh.manager = map[string]string{"linux": "apt"}
	gh.managerPackage = map[string]string{"linux": "gh"}
	apt := &fakeUpdateAdapter{name: "apt", kind: adapters.KindManager, policy: adapters.PolicyGated, trust: adapters.TrustOfficial}
	apt.checkPackage = func(pkg string) (adapters.UpdateInfo, error) {
		return adapters.UpdateInfo{}, errors.New("apt-cache policy failed")
	}
	apt.updatePackage = func(pkg string) (adapters.Result, error) {
		return adapters.Result{Success: true, Before: "2.45.0", After: "2.46.0"}, nil
	}

	out, err := runUpdateGroupWith(t, &GlobalFlags{}, "apt", "linux", "", []adapters.Adapter{apt, gh})
	if err != nil {
		t.Fatalf("group check-failed error: %v", err)
	}
	if gh.updatePackageOn {
		t.Error("a failed CheckPackage must NOT run the owned tool's UpdatePackage")
	}
	if !strings.Contains(out, "gh (check failed)") {
		t.Errorf("group must report a failed check as 'check failed'; got:\n%s", out)
	}
}

// TestRunUpdate_GroupCISudoFails proves spec security-model "--ci sudo group
// fails" / bulk-update: a sudo-heavy apt group package command is RiskHigh and,
// with EnforceRisk=true, --ci fails the group non-zero despite each owned tool
// being TrustOfficial.
func TestRunUpdate_GroupCISudoFails(t *testing.T) {
	gh := &fakeUpdateAdapter{name: "gh", kind: adapters.KindTool, policy: adapters.PolicyAlwaysUpdate, trust: adapters.TrustOfficial}
	gh.manager = map[string]string{"linux": "apt"}
	gh.managerPackage = map[string]string{"linux": "gh"}
	apt := &fakeUpdateAdapter{name: "apt", kind: adapters.KindManager, policy: adapters.PolicyGated, trust: adapters.TrustOfficial}
	apt.checkPackage = func(pkg string) (adapters.UpdateInfo, error) {
		return adapters.UpdateInfo{CurrentVersion: "2.45.0", LatestVersion: "2.46.0", UpdateAvailable: true}, nil
	}
	apt.updatePackage = func(pkg string) (adapters.Result, error) {
		return adapters.Result{Success: true, Before: "2.45.0", After: "2.46.0"}, nil
	}

	_, err := runUpdateGroupWith(t, &GlobalFlags{CI: true}, "apt", "linux", "", []adapters.Adapter{apt, gh})
	if err == nil {
		t.Fatal("--ci sudo apt group must fail non-zero (high risk needs confirmation)")
	}
	if gh.updatePackageOn {
		t.Error("--ci must NOT execute the sudo package command for the group")
	}
}

// TestRunUpdate_GroupNonSudoProceeds proves spec security-model "Non-sudo
// group proceeds": a brew group (brew upgrade gh, no sudo) risk is LOW, so it
// proceeds in non-CI mode without a prompt.
func TestRunUpdate_GroupNonSudoProceeds(t *testing.T) {
	brew := &fakeUpdateAdapter{name: "brew", kind: adapters.KindManager, policy: adapters.PolicyAlwaysUpdate, trust: adapters.TrustOfficial}
	gh := &fakeUpdateAdapter{name: "gh", kind: adapters.KindTool, policy: adapters.PolicyAlwaysUpdate, trust: adapters.TrustOfficial}
	gh.manager = map[string]string{"macos": "brew"}
	gh.managerPackage = map[string]string{"macos": "gh"}
	brew.checkPackage = func(pkg string) (adapters.UpdateInfo, error) {
		return adapters.UpdateInfo{CurrentVersion: "2.45.0", LatestVersion: "2.46.0", UpdateAvailable: true}, nil
	}
	brew.updatePackage = func(pkg string) (adapters.Result, error) {
		return adapters.Result{Success: true, Before: "2.45.0", After: "2.46.0"}, nil
	}

	out, err := runUpdateGroupWith(t, &GlobalFlags{}, "brew", "macos", "", []adapters.Adapter{brew, gh})
	if err != nil {
		t.Fatalf("non-sudo group error: %v", err)
	}
	if brew.lastUpdatePkg != "gh" {
		t.Error("non-sudo brew group must proceed and update gh")
	}
	if !strings.Contains(out, "gh updated") {
		t.Errorf("non-sudo group must update gh; got:\n%s", out)
	}
}

// TestRunUpdate_GroupDryRunPlansWithoutExecuting proves spec ux-patterns
// "Group dry-run": `upp update --manager apt --dry-run` reports a pending
// owned tool as "would update" WITHOUT running the mutating package command.
func TestRunUpdate_GroupDryRunPlansWithoutExecuting(t *testing.T) {
	gh := &fakeUpdateAdapter{name: "gh", kind: adapters.KindTool, policy: adapters.PolicyAlwaysUpdate, trust: adapters.TrustOfficial}
	gh.manager = map[string]string{"linux": "apt"}
	gh.managerPackage = map[string]string{"linux": "gh"}
	apt := &fakeUpdateAdapter{name: "apt", kind: adapters.KindManager, policy: adapters.PolicyGated, trust: adapters.TrustOfficial}
	apt.checkPackage = func(pkg string) (adapters.UpdateInfo, error) {
		return adapters.UpdateInfo{CurrentVersion: "2.45.0", LatestVersion: "2.46.0", UpdateAvailable: true}, nil
	}
	apt.updatePackage = func(pkg string) (adapters.Result, error) {
		return adapters.Result{Success: true, Before: "2.45.0", After: "2.46.0"}, nil
	}

	// Drive through runUpdate with DryRun set; it must plan, never execute.
	deps := updateDeps{
		buildAdapterList: func(*config.Config, string) []adapters.Adapter {
			return []adapters.Adapter{apt, gh}
		},
		stdinIsTTY: func() bool { return false },
	}
	out := withCapturedStdout(func() {
		if err := runUpdate(&GlobalFlags{}, &UpdateFlags{Manager: "apt", DryRun: true}, deps); err != nil {
			t.Errorf("group dry-run error: %v", err)
		}
	})
	if !strings.Contains(out, "would update") {
		t.Errorf("group dry-run must report the pending owned tool as 'would update'; got:\n%s", out)
	}
	if apt.updatePkgCount != 0 {
		t.Errorf("group dry-run must NEVER execute the package command; apt.UpdatePackage ran %d times", apt.updatePkgCount)
	}
}

// TestVerifyPins_StrictTTDScenarios pins the five CRITICAL scenarios from the
// verify report: bare --ci dashboard, update --ci failure exit, list --only
// round-trip, -v shorthand diagnostics, and clean all-success verbose output.
func TestVerifyPins_StrictTTDScenarios(t *testing.T) {
	writeCheckConfig(t, "")
	fail := &fakeUpdateAdapter{name: "broken", policy: adapters.PolicyAlwaysUpdate, trust: adapters.TrustOfficial, updateErr: fmt.Errorf("lock held")}
	okAd := &fakeUpdateAdapter{name: "apt", policy: adapters.PolicyGated, trust: adapters.TrustOfficial, info: adapters.UpdateInfo{CurrentVersion: "1.0.0", LatestVersion: "2.0.0", UpdateAvailable: true}, result: adapters.Result{Success: true, Before: "1.0.0", After: "2.0.0"}}
	run := func(args ...string) (string, error) {
		setCLIDeps(t, updateDeps{buildAdapterList: fakeAdapterList(fail, okAd), stdinIsTTY: func() bool { return false }}, listDeps{buildAdapterList: fakeAdapterList(fail, okAd)}, selfUpdateDeps{})
		root, gf := BuildRoot()
		AddCommands(root, gf)
		root.SetArgs(args)
		var err error
		out := withCapturedStdout(func() { err = root.Execute() })
		return out, err
	}
	for _, tt := range []struct {
		name, args, want, not string
		wantErr               bool
	}{
		{"bare --ci dashboard", "--ci", "upp update -n", "", false},
		{"update --ci non-zero on failure", "update --ci --only broken", "Failed: broken", "", true},
		{"list --only round-trip", "list --only apt", "apt", "brew", false},
		{"-v shorthand diagnostics", "update -v --only broken", "lock held", "", false},
		{"all-success verbose clean", "update -v --only apt", "1 updated", "│", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			out, err := run(strings.Fields(tt.args)...)
			if (err != nil) != tt.wantErr {
				t.Fatalf("pin %q: err=%v, wantErr=%v, output:\n%s", tt.name, err, tt.wantErr, out)
			}
			if !strings.Contains(out, tt.want) || (tt.not != "" && strings.Contains(out, tt.not)) {
				t.Errorf("pin %q assertions failed, output:\n%s", tt.name, out)
			}
		})
	}
}
