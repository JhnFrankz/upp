package cli

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/config"
	"github.com/JhnFrankz/upp/internal/output"
)

// fakeUpdateAdapter is a test double for the update gating matrix. It records
// whether Update was invoked — the behavioral signal the gating requirement
// is about — and lets each test control trust, command, privileges, check
// result, and update outcome. The command/privileges fields drive the
// security-risk classification path (update.go) exactly like a real custom
// adapter's ToolInfo.
type fakeUpdateAdapter struct {
	name       string
	trust      adapters.TrustLevel
	command    string
	privileges []string
	info       adapters.UpdateInfo
	checkErr   error
	updateErr  error
	result     adapters.Result
	updated    bool
}

func (f *fakeUpdateAdapter) Name() string { return f.name }

func (f *fakeUpdateAdapter) Detect() bool { return true }

func (f *fakeUpdateAdapter) Check() (adapters.UpdateInfo, error) {
	return f.info, f.checkErr
}

func (f *fakeUpdateAdapter) Update(dryRun bool) (adapters.Result, error) {
	f.updated = true
	return f.result, f.updateErr
}

func (f *fakeUpdateAdapter) Info() adapters.ToolInfo {
	return adapters.ToolInfo{
		ID:         f.name,
		Name:       f.name,
		Trust:      f.trust,
		Command:    f.command,
		Privileges: f.privileges,
	}
}

// fakeAdapterList returns a buildAdapterList seam yielding only the given
// fake adapter, hermetic and deterministic.
func fakeAdapterList(fake *fakeUpdateAdapter) func(*config.Config, string) []adapters.Adapter {
	return func(*config.Config, string) []adapters.Adapter {
		return []adapters.Adapter{fake}
	}
}

// runUpdateWithFlags runs runUpdate with the given global/update flags
// against a single fake adapter in a hermetic HOME, returning the captured
// stdout and the runUpdate error.
func runUpdateWithFlags(t *testing.T, fake *fakeUpdateAdapter, gf *GlobalFlags, uf *UpdateFlags) (string, error) {
	t.Helper()
	probeHome(t)
	deps := updateDeps{buildAdapterList: fakeAdapterList(fake)}
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
// tool-adapter): update() runs for a gated official adapter (real update
// detection: apt, npm, nvm, pnpm) only when check() reported
// update_available=true; official adapters WITHOUT update detection (stubs
// like brew) are exempt and always update; winget/scoop are exempt; custom
// adapters are exempt (they report false by design and still update).
func TestRunUpdate_GatingMatrix(t *testing.T) {
	tests := []struct {
		name            string
		id              string
		trust           adapters.TrustLevel
		updateAvailable bool
		wantUpdated     bool
		wantStatus      output.Status
	}{
		{
			name:            "gated dynamic apt with update available runs update",
			id:              "apt",
			trust:           adapters.TrustOfficial,
			updateAvailable: true,
			wantUpdated:     true,
			wantStatus:      output.StatusUpdated,
		},
		{
			name:            "gated dynamic apt without update is reported current and update is skipped",
			id:              "apt",
			trust:           adapters.TrustOfficial,
			updateAvailable: false,
			wantUpdated:     false,
			wantStatus:      output.StatusCurrent,
		},
		{
			name:            "official stub brew exempt: update still runs without update available",
			id:              "brew",
			trust:           adapters.TrustOfficial,
			updateAvailable: false,
			wantUpdated:     true,
			wantStatus:      output.StatusUpdated,
		},
		{
			name:            "winget exempt: update still runs without update available",
			id:              "winget",
			trust:           adapters.TrustOfficial,
			updateAvailable: false,
			wantUpdated:     true,
			wantStatus:      output.StatusUpdated,
		},
		{
			name:            "custom trusted exempt: update still runs without update available",
			id:              "custom",
			trust:           adapters.TrustCustomTrusted,
			updateAvailable: false,
			wantUpdated:     true,
			wantStatus:      output.StatusUpdated,
		},
		{
			name:            "custom untrusted exempt: update still runs without update available",
			id:              "custom",
			trust:           adapters.TrustCustomUntrusted,
			updateAvailable: false,
			wantUpdated:     true,
			wantStatus:      output.StatusUpdated,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeUpdateAdapter{
				name:  tt.id,
				trust: tt.trust,
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
			if tt.wantStatus == output.StatusUpdated {
				if !strings.Contains(out, "Updated: "+tt.id) {
					t.Errorf("output does not report tool as updated; got: %q", out)
				}
			} else if strings.Contains(out, "Updated:") || strings.Contains(out, "Failed:") {
				t.Errorf("output reports tool as updated or failed, want current; got: %q", out)
			}
		})
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
		trust:    adapters.TrustOfficial,
		checkErr: context.DeadlineExceeded,
	}
	ok := &fakeUpdateAdapter{
		name:  "npm",
		trust: adapters.TrustOfficial,
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
