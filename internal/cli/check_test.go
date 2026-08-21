package cli

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/config"
)

func TestRunCheck_Concurrent_OrderingAndIsolation(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Adapter 0: fast, up to date
	// Adapter 1: panics in Check
	// Adapter 2: slow with delay, available update
	// Adapter 3: timeout error
	// Adapter 4: up to date
	a0 := &fakeUpdateAdapter{name: "tool-0", policy: adapters.PolicyGated, trust: adapters.TrustOfficial, info: adapters.UpdateInfo{CurrentVersion: "1.0.0"}}
	a1 := &fakePanickingAdapter{name: "tool-1", panicCheck: true}
	a2 := &fakeDelayedAdapter{name: "tool-2", delay: 20 * time.Millisecond, info: adapters.UpdateInfo{UpdateAvailable: true, CurrentVersion: "1.0.0", LatestVersion: "1.2.0"}}
	a3 := &fakeDelayedAdapter{name: "tool-3", checkErr: context.DeadlineExceeded}
	a4 := &fakeUpdateAdapter{name: "tool-4", policy: adapters.PolicyGated, trust: adapters.TrustOfficial, info: adapters.UpdateInfo{CurrentVersion: "1.0.0"}}

	adapterListFunc := func(*config.Config, string) []adapters.Adapter {
		return []adapters.Adapter{a0, a1, a2, a3, a4}
	}

	setCLIDeps(t, checkDeps{buildAdapterList: adapterListFunc}, updateDeps{}, listDeps{}, selfUpdateDeps{})

	out := withCapturedStdout(func() {
		gf := &GlobalFlags{}
		err := runCheck(gf, "v0.1.0", cliDeps.check)
		if err != nil {
			t.Errorf("runCheck returned error: %v", err)
		}
	})

	// Verify all tools are represented in the output summary / details
	if !strings.Contains(out, "1 available") {
		t.Errorf("expected '1 available' in summary, got:\n%s", out)
	}
	if !strings.Contains(out, "2 up to date") {
		t.Errorf("expected '2 up to date' in summary, got:\n%s", out)
	}
	if !strings.Contains(out, "2 failed") {
		t.Errorf("expected '2 failed' in summary, got:\n%s", out)
	}
}

func TestRunCheck_VerboseFailureDiagnostics(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	failingAdapter := &fakeDelayedAdapter{
		name:     "fail-tool",
		checkErr: fmt.Errorf("lock frontend held by another process"),
	}

	deps := checkDeps{
		buildAdapterList: func(*config.Config, string) []adapters.Adapter {
			return []adapters.Adapter{failingAdapter}
		},
	}

	// With -v, diagnostics should be emitted
	outVerbose := withCapturedStdout(func() {
		gf := &GlobalFlags{Verbose: true}
		_ = runCheck(gf, "v0.1.0", deps)
	})
	if !strings.Contains(outVerbose, "lock frontend held by another process") {
		t.Errorf("expected verbose output to contain stderr diagnostic, got:\n%s", outVerbose)
	}

	// Without -v, stderr is suppressed
	outDefault := withCapturedStdout(func() {
		gf := &GlobalFlags{Verbose: false}
		_ = runCheck(gf, "v0.1.0", deps)
	})
	if strings.Contains(outDefault, "lock frontend held by another process") {
		t.Errorf("expected default output to suppress stderr diagnostic, got:\n%s", outDefault)
	}

	// With -q and -v, quiet takes precedence and suppresses
	outQuiet := withCapturedStdout(func() {
		gf := &GlobalFlags{Verbose: true, Quiet: true}
		_ = runCheck(gf, "v0.1.0", deps)
	})
	if strings.Contains(outQuiet, "lock frontend held by another process") {
		t.Errorf("expected quiet mode to suppress stderr diagnostic, got:\n%s", outQuiet)
	}
}
