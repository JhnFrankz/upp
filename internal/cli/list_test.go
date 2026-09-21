package cli

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/platform"
)

// runListWith runs runList with the given global flags against the given
// fake adapters in a hermetic HOME, returning the captured stdout.
func runListWith(t *testing.T, gf *GlobalFlags, fakes ...*fakeUpdateAdapter) string {
	t.Helper()
	probeHome(t)
	deps := listDeps{buildAdapterList: fakeAdapterList(fakes...)}
	out := withCapturedStdout(func() {
		if err := runList(context.Background(), gf, deps); err != nil {
			t.Errorf("runList returned error: %v", err)
		}
	})
	return out
}

func TestRunList_ContextCanceled(t *testing.T) {
	probeHome(t)
	tool := &fakeUpdateAdapter{
		name:   "apt",
		policy: adapters.PolicyAlwaysUpdate,
		trust:  adapters.TrustOfficial,
		info:   adapters.UpdateInfo{CurrentVersion: "1.0.0"},
	}
	deps := listDeps{buildAdapterList: fakeAdapterList(tool)}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := runList(ctx, &GlobalFlags{}, deps)
	if err == nil {
		t.Fatal("runList(canceled ctx): want error, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("runList(canceled ctx): error = %v, want errors.Is context.Canceled", err)
	}
}

// TestRunList_EmptyVsFilterMismatch pins the two "nothing to list" exit
// paths: a genuinely empty configured set must say so, while an --only
// filter that matched nothing must report the filter mismatch — never the
// misleading "no tools configured". Both messages go through the Renderer,
// so --quiet suppresses them like every other list status output.
func TestRunList_EmptyVsFilterMismatch(t *testing.T) {
	tool := &fakeUpdateAdapter{
		name:   "apt",
		policy: adapters.PolicyAlwaysUpdate,
		trust:  adapters.TrustOfficial,
		info:   adapters.UpdateInfo{CurrentVersion: "1.0.0"},
	}

	tests := []struct {
		name        string
		gf          *GlobalFlags
		fakes       []*fakeUpdateAdapter
		wantContain string
		wantAbsent  string
		wantEmpty   bool
	}{
		{
			name:        "only-unknown-with-tools/filter-mismatch-message",
			gf:          &GlobalFlags{Only: "nonexistent"},
			fakes:       []*fakeUpdateAdapter{tool},
			wantContain: "no tools match --only filter: nonexistent",
			wantAbsent:  "No tools configured.",
		},
		{
			name:      "only-unknown-with-tools/quiet-suppresses",
			gf:        &GlobalFlags{Only: "nonexistent", Quiet: true},
			fakes:     []*fakeUpdateAdapter{tool},
			wantEmpty: true,
		},
		{
			name:        "only-unknown-empty-list/filter-mismatch-message",
			gf:          &GlobalFlags{Only: "nonexistent"},
			fakes:       nil,
			wantContain: "no tools match --only filter: nonexistent",
			wantAbsent:  "No tools configured.",
		},
		{
			name:      "only-unknown-empty-list/quiet-suppresses",
			gf:        &GlobalFlags{Only: "nonexistent", Quiet: true},
			fakes:     nil,
			wantEmpty: true,
		},
		{
			name:        "empty-config/no-tools-message",
			gf:          &GlobalFlags{},
			fakes:       nil,
			wantContain: "no tools configured",
			wantAbsent:  "",
		},
		{
			name:      "empty-config/quiet-suppresses",
			gf:        &GlobalFlags{Quiet: true},
			fakes:     nil,
			wantEmpty: true,
		},
		{
			name:        "tools-listed/no-empty-message",
			gf:          &GlobalFlags{},
			fakes:       []*fakeUpdateAdapter{tool},
			wantContain: "apt",
			wantAbsent:  "no tools",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := runListWith(t, tt.gf, tt.fakes...)

			if tt.wantEmpty {
				if strings.TrimSpace(out) != "" {
					t.Errorf("--quiet must suppress the message, got:\n%s", out)
				}
				return
			}
			if !strings.Contains(out, tt.wantContain) {
				t.Errorf("output must contain %q, got:\n%s", tt.wantContain, out)
			}
			if tt.wantAbsent != "" && strings.Contains(out, tt.wantAbsent) {
				t.Errorf("output must NOT contain %q, got:\n%s", tt.wantAbsent, out)
			}
		})
	}
}

// TestRunList_GatesCheckCommandByRisk pins that `upp list` never executes check
// commands requiring consent (RiskAboveLow), neutralizing them while still
// listing the tool as detected with an empty version.
func TestRunList_GatesCheckCommandByRisk(t *testing.T) {
	benignTool := &fakeUpdateAdapter{
		name:         "benign",
		checkCommand: "benign --version",
		trust:        adapters.TrustCustomUntrusted,
		info:         adapters.UpdateInfo{CurrentVersion: "1.0.0"},
	}
	riskyTool := &fakeUpdateAdapter{
		name:         "risky",
		checkCommand: "sudo risky --version",
		trust:        adapters.TrustCustomUntrusted,
		info:         adapters.UpdateInfo{CurrentVersion: "2.0.0"},
	}

	out := runListWith(t, &GlobalFlags{}, benignTool, riskyTool)

	if benignTool.checkCount != 1 {
		t.Errorf("benign tool checkCount = %d, want 1", benignTool.checkCount)
	}
	if riskyTool.checkCount != 0 {
		t.Errorf("risky tool checkCount = %d, want 0 (must be neutralized)", riskyTool.checkCount)
	}
	if !strings.Contains(out, "benign") || !strings.Contains(out, "1.0.0") {
		t.Errorf("output should list benign tool with version 1.0.0, got:\n%s", out)
	}
	if !strings.Contains(out, "risky") {
		t.Errorf("output should list risky tool, got:\n%s", out)
	}
	if strings.Contains(out, "2.0.0") {
		t.Errorf("risky tool version 2.0.0 must NOT be in output, got:\n%s", out)
	}
}

// TestRunList_OwnedToolGroupedUnderManager proves that tools owned by a manager
// are grouped under that manager's header in list output.
func TestRunList_OwnedToolGroupedUnderManager(t *testing.T) {
	p, err := platform.Detect()
	if err != nil {
		t.Fatalf("platform.Detect error: %v", err)
	}

	mgr := &fakeUpdateAdapter{
		name:     "apt",
		infoName: "APT Package Manager",
		kind:     adapters.KindManager,
		info:     adapters.UpdateInfo{CurrentVersion: "1.0.0"},
	}
	tool := &fakeUpdateAdapter{
		name:           "gh",
		kind:           adapters.KindTool,
		manager:        map[string]string{p.OS: "apt"},
		managerPackage: map[string]string{p.OS: "gh"},
		info:           adapters.UpdateInfo{CurrentVersion: "2.4.0"},
	}

	out := runListWith(t, &GlobalFlags{}, mgr, tool)

	if !strings.Contains(out, "APT Package Manager") {
		t.Errorf("output missing manager header 'APT Package Manager', got:\n%s", out)
	}
	if !strings.Contains(out, "gh") || !strings.Contains(out, "2.4.0") {
		t.Errorf("output missing owned tool 'gh' 2.4.0, got:\n%s", out)
	}
}
