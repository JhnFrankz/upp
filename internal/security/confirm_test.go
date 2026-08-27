package security

import (
	"strings"
	"testing"

	"github.com/JhnFrankz/upp/internal/adapters"
)

func TestConfirmAction_OfficialTools(t *testing.T) {
	tests := []struct {
		name      string
		riskLevel RiskLevel
	}{
		{"official low risk", RiskLow},
		{"official medium risk", RiskMedium},
		{"official high risk", RiskHigh},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := ConfirmConfig{
				ToolName:   "brew",
				TrustLevel: adapters.TrustOfficial,
				RiskLevel:  tt.riskLevel,
				Command:    "brew upgrade",
				CI:         false,
			}
			got := ConfirmAction(cfg)
			if got != ConfirmAuto {
				t.Errorf("ConfirmAction() = %v, want ConfirmAuto", got)
			}
		})
	}
}

func TestConfirmAction_CustomUntrusted_CI(t *testing.T) {
	// D4: CI Low→Auto (even untrusted); Medium→Err; High→Err — no prompts possible.
	tests := []struct {
		name      string
		riskLevel RiskLevel
		want      ConfirmDecision
	}{
		{"low risk CI auto", RiskLow, ConfirmAuto},
		{"medium risk CI error", RiskMedium, ConfirmError},
		{"high risk CI error", RiskHigh, ConfirmError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := ConfirmConfig{
				ToolName:   "mytool",
				TrustLevel: adapters.TrustCustomUntrusted,
				RiskLevel:  tt.riskLevel,
				Command:    "mytool --update",
				CI:         true,
			}
			got := ConfirmAction(cfg)
			if got != tt.want {
				t.Errorf("ConfirmAction() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfirmAction_CustomTrusted_CI(t *testing.T) {
	tests := []struct {
		name      string
		riskLevel RiskLevel
		want      ConfirmDecision
	}{
		{"low risk CI auto", RiskLow, ConfirmAuto},
		{"medium risk CI auto", RiskMedium, ConfirmAuto},
		{"high risk CI error", RiskHigh, ConfirmError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := ConfirmConfig{
				ToolName:   "mytool",
				TrustLevel: adapters.TrustCustomTrusted,
				RiskLevel:  tt.riskLevel,
				Command:    "mytool --update",
				CI:         true,
			}
			got := ConfirmAction(cfg)
			if got != tt.want {
				t.Errorf("ConfirmAction() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfirmAction_CustomHighRisk_Interactive(t *testing.T) {
	// High risk always prompts — regardless of trust.
	tests := []struct {
		name  string
		trust adapters.TrustLevel
		input string
		want  ConfirmDecision
	}{
		{"untrusted yes", adapters.TrustCustomUntrusted, "y\n", ConfirmProceed},
		{"untrusted no", adapters.TrustCustomUntrusted, "n\n", ConfirmDeny},
		{"trusted yes", adapters.TrustCustomTrusted, "y\n", ConfirmProceed},
		{"trusted no", adapters.TrustCustomTrusted, "n\n", ConfirmDeny},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := ConfirmConfig{
				ToolName:   "mytool",
				TrustLevel: tt.trust,
				RiskLevel:  RiskHigh,
				Command:    "sudo mytool --update",
				CI:         false,
				Privileges: []string{"sudo"},
				Reader:     strings.NewReader(tt.input),
			}
			got := ConfirmAction(cfg)
			if got != tt.want {
				t.Errorf("ConfirmAction() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfirmAction_CustomMediumRisk_Interactive(t *testing.T) {
	// Untrusted medium risk prompts; trusted medium risk shows info and proceeds.
	tests := []struct {
		name  string
		trust adapters.TrustLevel
		input string
		want  ConfirmDecision
	}{
		{"untrusted yes", adapters.TrustCustomUntrusted, "y\n", ConfirmProceed},
		{"untrusted no", adapters.TrustCustomUntrusted, "n\n", ConfirmDeny},
		{"trusted no input needed", adapters.TrustCustomTrusted, "", ConfirmProceed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var reader *strings.Reader
			if tt.input != "" {
				reader = strings.NewReader(tt.input)
			}
			cfg := ConfirmConfig{
				ToolName:   "mytool",
				TrustLevel: tt.trust,
				RiskLevel:  RiskMedium,
				Command:    "mytool --update",
				CI:         false,
				Reader:     reader,
			}
			got := ConfirmAction(cfg)
			if got != tt.want {
				t.Errorf("ConfirmAction() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfirmAction_EnforceRiskOfficialHigh_Interactive(t *testing.T) {
	// D4 reclassification: an owned tool is TrustOfficial, but with
	// EnforceRisk:true the REAL command risk decides — so a sudo-heavy High
	// risk must PROMPT (any trust), never short-circuit to ConfirmAuto.
	tests := []struct {
		name  string
		input string
		want  ConfirmDecision
	}{
		{"yes proceeds", "y\n", ConfirmProceed},
		{"no denies", "n\n", ConfirmDeny},
		{"empty defaults deny", "\n", ConfirmDeny},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := ConfirmConfig{
				ToolName:    "gh",
				TrustLevel:  adapters.TrustOfficial,
				RiskLevel:   RiskHigh,
				Command:     "sudo apt install --only-upgrade gh",
				Privileges:  []string{"sudo"},
				CI:          false,
				EnforceRisk: true,
				Reader:      strings.NewReader(tt.input),
			}
			got := ConfirmAction(cfg)
			if got != tt.want {
				t.Errorf("ConfirmAction(EnforceRisk:true, TrustOfficial, High) = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfirmAction_EnforceRiskOfficialHigh_CI(t *testing.T) {
	// D4 reclassification in --ci: EnforceRisk:true + TrustOfficial + High
	// risk MUST error (non-zero die), not auto-proceed as the official
	// short-circuit would.
	cfg := ConfirmConfig{
		ToolName:    "gh",
		TrustLevel:  adapters.TrustOfficial,
		RiskLevel:   RiskHigh,
		Command:     "sudo apt install --only-upgrade gh",
		Privileges:  []string{"sudo"},
		CI:          true,
		EnforceRisk: true,
	}
	got := ConfirmAction(cfg)
	if got != ConfirmError {
		t.Errorf("ConfirmAction(EnforceRisk:true, TrustOfficial, High, CI) = %v, want ConfirmError", got)
	}
}

func TestConfirmAction_EnforceRiskOfficialLow(t *testing.T) {
	// D4 triangulation: EnforceRisk:true + TrustOfficial does NOT make every
	// official row prompt — the RISK decides. A Low (non-sudo) command stays
	// auto-proceed (CI) / info-proceed (interactive) because Low never
	// requires confirmation, even for an owned official tool.
	tests := []struct {
		name string
		ci   bool
		want ConfirmDecision
	}{
		{"interactive low info-proceeds", false, ConfirmProceed},
		{"CI low auto", true, ConfirmAuto},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := ConfirmConfig{
				ToolName:    "gh",
				TrustLevel:  adapters.TrustOfficial,
				RiskLevel:   RiskLow,
				Command:     "brew upgrade gh",
				CI:          tt.ci,
				EnforceRisk: true,
			}
			got := ConfirmAction(cfg)
			if got != tt.want {
				t.Errorf("ConfirmAction(EnforceRisk:true, TrustOfficial, Low, ci=%v) = %v, want %v", tt.ci, got, tt.want)
			}
		})
	}
}

func TestConfirmAction_CustomLowRisk_Interactive(t *testing.T) {
	// Low risk always shows info and proceeds — regardless of trust.
	tests := []struct {
		name  string
		trust adapters.TrustLevel
	}{
		{"untrusted", adapters.TrustCustomUntrusted},
		{"trusted", adapters.TrustCustomTrusted},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := ConfirmConfig{
				ToolName:   "mytool",
				TrustLevel: tt.trust,
				RiskLevel:  RiskLow,
				Command:    "mytool --version",
				CI:         false,
			}
			got := ConfirmAction(cfg)
			if got != ConfirmProceed {
				t.Errorf("ConfirmAction() = %v, want ConfirmProceed", got)
			}
		})
	}
}
