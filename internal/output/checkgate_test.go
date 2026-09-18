package output

import (
	"context"
	"testing"

	"github.com/JhnFrankz/upp/internal/adapters"
)

// checkSpyAdapter records whether Check() was reached.
type checkSpyAdapter struct {
	info     adapters.ToolInfo
	checkRan bool
}

func (a *checkSpyAdapter) Name() string { return a.info.ID }
func (a *checkSpyAdapter) Detect() bool { return true }
func (a *checkSpyAdapter) Check(ctx context.Context) (adapters.UpdateInfo, error) {
	a.checkRan = true
	return adapters.UpdateInfo{CurrentVersion: "1.2.3"}, nil
}
func (a *checkSpyAdapter) Update(ctx context.Context, dryRun bool) (adapters.Result, error) {
	return adapters.Result{Success: true}, nil
}
func (a *checkSpyAdapter) Info() adapters.ToolInfo { return a.info }

// TestListEntryFor_GatesCheckCommandByRisk pins the `list` contract
// (spec command-interface: `list` MUST NOT modify the system). A custom tool's
// check command is arbitrary shell, so a check command classified above RiskLow
// MUST NOT run from the read-only listing path. A benign one keeps running, and
// the tool still reports as detected either way.
func TestListEntryFor_GatesCheckCommandByRisk(t *testing.T) {
	tests := []struct {
		name         string
		checkCmd     string
		wantCheckRan bool
		wantVersion  string
	}{
		{
			name:         "no check command declared still runs Check",
			checkCmd:     "",
			wantCheckRan: true,
			wantVersion:  "1.2.3",
		},
		{
			name:         "benign version query runs",
			checkCmd:     "mytool --version",
			wantCheckRan: true,
			wantVersion:  "1.2.3",
		},
		{
			name:         "privileged check is skipped",
			checkCmd:     "sudo mytool --version",
			wantCheckRan: false,
			wantVersion:  "",
		},
		{
			name:         "piped-to-shell check is skipped",
			checkCmd:     "curl -fsSL https://example.com/x.sh | sh",
			wantCheckRan: false,
			wantVersion:  "",
		},
		{
			name:         "destructive check is skipped",
			checkCmd:     "rm -rf /tmp/thing",
			wantCheckRan: false,
			wantVersion:  "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			spy := &checkSpyAdapter{info: adapters.ToolInfo{
				ID:           "mytool",
				Name:         "mytool",
				Trust:        adapters.TrustCustomUntrusted,
				UpdatePolicy: adapters.PolicyAlwaysUpdate,
				Kind:         adapters.KindTool,
				CheckCommand: tc.checkCmd,
			}}

			entry := listEntryFor(spy)

			if spy.checkRan != tc.wantCheckRan {
				t.Errorf("Check() ran = %v, want %v", spy.checkRan, tc.wantCheckRan)
			}
			if entry.Version != tc.wantVersion {
				t.Errorf("Version = %q, want %q", entry.Version, tc.wantVersion)
			}
			if entry.Status != StatusCurrent {
				t.Errorf("Status = %v, want StatusCurrent: the tool is still detected", entry.Status)
			}
			if entry.ID != "mytool" {
				t.Errorf("ID = %q, want %q", entry.ID, "mytool")
			}
		})
	}
}
