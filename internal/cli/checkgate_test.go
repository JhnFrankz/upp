package cli

import (
	"context"
	"io"
	"testing"

	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/output"
	"github.com/JhnFrankz/upp/internal/security"
)

// checkGateAdapter is a minimal adapter that only declares a check command, so
// the check gate can be exercised without any real tool. checkRan records
// whether the underlying check actually executed.
type checkGateAdapter struct {
	info     adapters.ToolInfo
	checkRan bool
}

func (a *checkGateAdapter) Name() string { return a.info.ID }
func (a *checkGateAdapter) Detect() bool { return true }
func (a *checkGateAdapter) Check(ctx context.Context) (adapters.UpdateInfo, error) {
	a.checkRan = true
	return adapters.UpdateInfo{CurrentVersion: "1.0.0"}, nil
}
func (a *checkGateAdapter) Update(ctx context.Context, dryRun bool) (adapters.Result, error) {
	return adapters.Result{Success: true}, nil
}
func (a *checkGateAdapter) Info() adapters.ToolInfo { return a.info }

func checkGateAdapters(checks ...string) []*checkGateAdapter {
	list := make([]*checkGateAdapter, 0, len(checks))
	for i, c := range checks {
		list = append(list, &checkGateAdapter{info: adapters.ToolInfo{
			ID:           "tool" + string(rune('a'+i)),
			Name:         "tool" + string(rune('a'+i)),
			Trust:        adapters.TrustCustomUntrusted,
			UpdatePolicy: adapters.PolicyAlwaysUpdate,
			Kind:         adapters.KindTool,
			CheckCommand: c,
		}})
	}
	return list
}

// TestEnforceRiskFor pins when a planned row's REAL command risk must decide the
// confirmation decision instead of the TrustOfficial auto-proceed shortcut
// (spec security-model: custom check-command gate; official/custom `--ci` rule).
func TestEnforceRiskFor(t *testing.T) {
	tests := []struct {
		name  string
		trust adapters.TrustLevel
		risk  security.RiskLevel
		ci    bool
		want  bool
	}{
		{
			name:  "official low-risk keeps auto-proceed",
			trust: adapters.TrustOfficial,
			risk:  security.RiskLow,
			want:  false,
		},
		{
			name:  "official low-risk under --ci keeps auto-proceed",
			trust: adapters.TrustOfficial,
			risk:  security.RiskLow,
			ci:    true,
			want:  false,
		},
		{
			name:  "official high-risk prompts interactively",
			trust: adapters.TrustOfficial,
			risk:  security.RiskHigh,
			want:  true,
		},
		{
			name:  "official high-risk proceeds under --ci because upp ships the command",
			trust: adapters.TrustOfficial,
			risk:  security.RiskHigh,
			ci:    true,
			want:  false,
		},
		{
			name:  "custom untrusted high-risk prompts interactively",
			trust: adapters.TrustCustomUntrusted,
			risk:  security.RiskHigh,
			want:  true,
		},
		{
			name:  "custom untrusted high-risk fails under --ci",
			trust: adapters.TrustCustomUntrusted,
			risk:  security.RiskHigh,
			ci:    true,
			want:  true,
		},
		{
			name:  "custom trusted high-risk still fails under --ci",
			trust: adapters.TrustCustomTrusted,
			risk:  security.RiskHigh,
			ci:    true,
			want:  true,
		},
		{
			name:  "custom trusted medium-risk proceeds under --ci",
			trust: adapters.TrustCustomTrusted,
			risk:  security.RiskMedium,
			ci:    true,
			want:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := enforceRiskFor(tc.trust, tc.risk, tc.ci); got != tc.want {
				t.Errorf("enforceRiskFor(%v, %v, ci=%v) = %v, want %v",
					tc.trust, tc.risk, tc.ci, got, tc.want)
			}
		})
	}
}

// TestAuthorizeChecks_GatesByRisk pins the update-side half of the check gate
// (spec security-model: custom check-command gate). A check command is
// arbitrary shell, so it is gated by real risk exactly like an update command:
// --ci denies with a non-zero error, and an interactive denial neutralizes that
// check while leaving the rest of the run intact.
//
// Length and order MUST be preserved: the sequential update path indexes
// outcomes by filtered-adapter position, so a shorter list panics there.
func TestAuthorizeChecks_GatesByRisk(t *testing.T) {
	const (
		benign    = "mytool --version"
		dangerous = "sudo mytool --version"
	)

	tests := []struct {
		name    string
		checks  []string
		ci      bool
		stdin   string
		wantRan int
		wantErr bool
	}{
		{
			name:    "no check command needs no consent",
			checks:  []string{""},
			wantRan: 1,
		},
		{
			name:    "benign checks need no consent",
			checks:  []string{benign, benign},
			wantRan: 2,
		},
		{
			name:    "--ci denies a dangerous check",
			checks:  []string{benign, dangerous},
			ci:      true,
			wantErr: true,
		},
		{
			name:    "interactive denial neutralizes only that check",
			checks:  []string{benign, dangerous},
			stdin:   "n\n",
			wantRan: 1,
		},
		{
			name:    "interactive approval runs it",
			checks:  []string{benign, dangerous},
			stdin:   "y\n",
			wantRan: 2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gf := &GlobalFlags{CI: tc.ci}
			spies := checkGateAdapters(tc.checks...)
			list := make([]adapters.Adapter, len(spies))
			for i, s := range spies {
				list[i] = s
			}
			r := output.NewRenderer(io.Discard, false)

			var kept []adapters.Adapter
			var err error
			withStdin(t, tc.stdin, func() {
				kept, err = authorizeChecks(gf, list, r)
			})

			if tc.wantErr {
				if err == nil {
					t.Fatal("expected a non-zero error for a dangerous check under --ci")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(kept) != len(list) {
				t.Fatalf("authorizeChecks returned %d adapters, want %d: length must be preserved or the update path panics on index", len(kept), len(list))
			}
			for i := range kept {
				if kept[i].Name() != list[i].Name() {
					t.Errorf("position %d = %q, want %q: order must be preserved", i, kept[i].Name(), list[i].Name())
				}
			}

			for i, a := range kept {
				if _, checkErr := a.Check(context.Background()); checkErr != nil {
					t.Fatalf("Check() on position %d: %v", i, checkErr)
				}
			}

			ran := 0
			for _, s := range spies {
				if s.checkRan {
					ran++
				}
			}
			if ran != tc.wantRan {
				t.Errorf("%d underlying checks ran, want %d", ran, tc.wantRan)
			}
		})
	}
}
