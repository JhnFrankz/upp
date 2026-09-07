package cli

import (
	"strings"
	"testing"

	"github.com/JhnFrankz/upp/internal/adapters"
)

// runListWith runs runList with the given global flags against the given
// fake adapters in a hermetic HOME, returning the captured stdout.
func runListWith(t *testing.T, gf *GlobalFlags, fakes ...*fakeUpdateAdapter) string {
	t.Helper()
	probeHome(t)
	deps := listDeps{buildAdapterList: fakeAdapterList(fakes...)}
	out := withCapturedStdout(func() {
		if err := runList(gf, deps); err != nil {
			t.Errorf("runList returned error: %v", err)
		}
	})
	return out
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
