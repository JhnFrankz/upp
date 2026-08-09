package security

import (
	"strings"
	"testing"
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
				TrustLevel: "official",
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
	// Untrusted custom tools always error in CI — no prompts possible.
	tests := []struct {
		name      string
		riskLevel RiskLevel
		want      ConfirmDecision
	}{
		{"low risk CI error", RiskLow, ConfirmError},
		{"medium risk CI error", RiskMedium, ConfirmError},
		{"high risk CI error", RiskHigh, ConfirmError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := ConfirmConfig{
				ToolName:   "mytool",
				TrustLevel: "custom",
				RiskLevel:  tt.riskLevel,
				Command:    "mytool --update",
				CI:         true,
				Trusted:    false,
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
				TrustLevel: "custom",
				RiskLevel:  tt.riskLevel,
				Command:    "mytool --update",
				CI:         true,
				Trusted:    true,
			}
			got := ConfirmAction(cfg)
			if got != tt.want {
				t.Errorf("ConfirmAction() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfirmAction_CustomHighRisk_Interactive(t *testing.T) {
	// High risk always prompts — test with "yes" input.
	reader := strings.NewReader("y\n")
	cfg := ConfirmConfig{
		ToolName:   "mytool",
		TrustLevel: "custom",
		RiskLevel:  RiskHigh,
		Command:    "sudo mytool --update",
		CI:         false,
		Trusted:    false,
		Privileges: []string{"sudo"},
		Reader:     reader,
	}
	got := ConfirmAction(cfg)
	if got != ConfirmProceed {
		t.Errorf("ConfirmAction() = %v, want ConfirmProceed", got)
	}
}

func TestConfirmAction_CustomHighRisk_Deny(t *testing.T) {
	// High risk with "no" input.
	reader := strings.NewReader("n\n")
	cfg := ConfirmConfig{
		ToolName:   "mytool",
		TrustLevel: "custom",
		RiskLevel:  RiskHigh,
		Command:    "sudo mytool --update",
		CI:         false,
		Trusted:    true,
		Privileges: []string{"sudo"},
		Reader:     reader,
	}
	got := ConfirmAction(cfg)
	if got != ConfirmDeny {
		t.Errorf("ConfirmAction() = %v, want ConfirmDeny", got)
	}
}

func TestConfirmAction_CustomMediumRisk_Interactive(t *testing.T) {
	// Untrusted medium risk prompts.
	reader := strings.NewReader("y\n")
	cfg := ConfirmConfig{
		ToolName:   "mytool",
		TrustLevel: "custom",
		RiskLevel:  RiskMedium,
		Command:    "mytool --update",
		CI:         false,
		Trusted:    false,
		Reader:     reader,
	}
	got := ConfirmAction(cfg)
	if got != ConfirmProceed {
		t.Errorf("ConfirmAction() = %v, want ConfirmProceed", got)
	}
}

func TestConfirmAction_CustomTrustedMediumRisk_ShowsInfo(t *testing.T) {
	// Trusted custom tool with medium risk: shows info, no prompt.
	cfg := ConfirmConfig{
		ToolName:   "mytool",
		TrustLevel: "custom",
		RiskLevel:  RiskMedium,
		Command:    "mytool --update",
		CI:         false,
		Trusted:    true,
	}
	got := ConfirmAction(cfg)
	if got != ConfirmProceed {
		t.Errorf("ConfirmAction() = %v, want ConfirmProceed", got)
	}
}

func TestConfirmAction_CustomLowRisk_ShowsInfo(t *testing.T) {
	// Low risk always proceeds with info.
	cfg := ConfirmConfig{
		ToolName:   "mytool",
		TrustLevel: "custom",
		RiskLevel:  RiskLow,
		Command:    "mytool --version",
		CI:         false,
		Trusted:    false,
	}
	got := ConfirmAction(cfg)
	if got != ConfirmProceed {
		t.Errorf("ConfirmAction() = %v, want ConfirmProceed", got)
	}
}
