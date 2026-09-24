package security

import (
	"strings"
	"testing"
)

// --- Risk Classification Edge Cases ---

func TestClassifyCommand_EmptyString(t *testing.T) {
	got := ClassifyCommand("")
	if got != RiskLow {
		t.Errorf("ClassifyCommand(\"\") = %v, want RiskLow", got)
	}
}

func TestClassifyCommand_WhitespaceOnly(t *testing.T) {
	got := ClassifyCommand("   ")
	if got != RiskLow {
		t.Errorf("ClassifyCommand(\"   \") = %v, want RiskLow", got)
	}
}

func TestClassifyCommand_CaseInsensitive(t *testing.T) {
	tests := []struct {
		name string
		cmd  string
		want RiskLevel
	}{
		{"SUDO uppercase", "SUDO apt upgrade", RiskHigh},
		{"Sudo mixed case", "Sudo apt upgrade", RiskHigh},
		{"RM -RF uppercase", "RM -RF /tmp", RiskHigh},
		{"CURL PIPE SH uppercase", "CURL https://x.com | SH", RiskHigh},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyCommand(tt.cmd)
			if got != tt.want {
				t.Errorf("ClassifyCommand(%q) = %v, want %v", tt.cmd, got, tt.want)
			}
		})
	}
}

func TestClassifyCommand_HighRiskEdgeCases(t *testing.T) {
	tests := []struct {
		name string
		cmd  string
	}{
		{"sudo with args", "sudo -E npm install -g"},
		{"rm -rf nested", "rm -rf /tmp/node_modules"},
		{"rm -r / dangerous", "rm -r /etc/important"},
		{"curl pipe sh bare", "curl https://install.sh | sh"},
		{"curl pipe bash", "curl https://install.sh | bash"},
		{"wget pipe sh", "wget -qO- https://x.com | sh"},
		{"curl pipe sh compact", "curl https://install.sh|sh"},
		{"curl pipe bash compact", "curl https://install.sh|bash"},
		{"wget pipe sh compact", "wget -qO- https://x.com|sh"},
		{"eval command", "eval $(docker-machine env)"},
		{"rm -rf root", "sudo rm -rf /"},
		{"curl -fsSL pipe", "curl -fsSL https://get.docker.com | sh"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyCommand(tt.cmd)
			if got != RiskHigh {
				t.Errorf("ClassifyCommand(%q) = %v, want RiskHigh", tt.cmd, got)
			}
		})
	}
}

func TestClassifyCommand_MediumRiskEdgeCases(t *testing.T) {
	tests := []struct {
		name string
		cmd  string
	}{
		{"apt remove package", "apt remove nginx"},
		{"brew uninstall package", "brew uninstall node"},
		{"npm uninstall global", "npm uninstall -g typescript"},
		{"pip uninstall package", "pip uninstall requests"},
		{"apt purge package", "apt purge old-package"},
		{"command chaining &&", "apt update && apt upgrade"},
		{"command chaining ||", "cmd1 || cmd2"},
		{"command chaining ;", "cmd1; cmd2"},
		{"multiple chaining", "cmd1 && cmd2; cmd3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyCommand(tt.cmd)
			if got != RiskMedium {
				t.Errorf("ClassifyCommand(%q) = %v, want RiskMedium", tt.cmd, got)
			}
		})
	}
}

func TestClassifyCommand_LowRiskEdgeCases(t *testing.T) {
	tests := []struct {
		name string
		cmd  string
	}{
		{"brew upgrade", "brew upgrade"},
		{"npm update global", "npm update -g"},
		{"pnpm update global", "pnpm update -g"},
		{"bun upgrade", "bun upgrade"},
		{"simple echo", "echo hello"},
		{"go version", "go version"},
		{"docker ps", "docker ps"},
		{"git status", "git status"},
		{"nvm install", "nvm install stable"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyCommand(tt.cmd)
			if got != RiskLow {
				t.Errorf("ClassifyCommand(%q) = %v, want RiskLow", tt.cmd, got)
			}
		})
	}
}

// --- Pipe to Shell Detection Edge Cases ---

func TestHasPipeToShell_EdgeCases(t *testing.T) {
	tests := []struct {
		name string
		cmd  string
		want bool
	}{
		{"pipe to sh", "cat file | sh", true},
		{"pipe to bash", "echo test | bash", true},
		{"pipe to zsh", "source | zsh", true},
		{"pipe to zsh with space", "source | zsh", true},
		{"compact pipe to sh", "curl https://x.com|sh", true},
		{"compact pipe to bash", "curl https://x.com|bash", true},
		{"compact wget pipe sh", "wget -qO- https://x.com|sh", true},
		{"pipe not shell", "cat file | grep foo", false},
		{"no pipe", "echo hello", false},
		{"double pipe", "cmd1 || cmd2", false},
		{"pipe sh in url", "curl https://x.com/sh", false},
		{"pipe at end", "echo test |", false},
		{"pipe at start", "| sh", true},
		{"multiple pipes", "cat file | grep sh | head", false},
		{"pipe to fish", "echo test | fish", true},
		{"pipe to zsh with or-or", "echo test || zsh", true}, // || zsh contains | zsh
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasPipeToShell(tt.cmd); got != tt.want {
				t.Errorf("hasPipeToShell(%q) = %v, want %v", tt.cmd, got, tt.want)
			}
		})
	}
}

// --- Command Chaining Detection Edge Cases ---

func TestHasCommandChaining_EdgeCases(t *testing.T) {
	tests := []struct {
		name string
		cmd  string
		want bool
	}{
		{"double amp", "cmd1 && cmd2", true},
		{"double pipe", "cmd1 || cmd2", true},
		{"semicolon", "cmd1; cmd2", true},
		{"no chaining", "cmd1", false},
		{"double amp in string", "echo '&&'", true},
		{"semicolon in URL", "curl https://x.com; echo done", true},
		{"multiple &&", "cmd1 && cmd2 && cmd3", true},
		{"mixed chaining", "cmd1 && cmd2; cmd3 || cmd4", true},
		{"empty string", "", false},
		{"spaces only", "   ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasCommandChaining(tt.cmd); got != tt.want {
				t.Errorf("hasCommandChaining(%q) = %v, want %v", tt.cmd, got, tt.want)
			}
		})
	}
}

// --- Risk Level String Edge Cases ---

func TestRiskLevelString_EdgeCases(t *testing.T) {
	tests := []struct {
		level RiskLevel
		want  string
	}{
		{RiskLow, "LOW"},
		{RiskMedium, "MEDIUM"},
		{RiskHigh, "HIGH"},
		{RiskLevel(-1), "UNKNOWN"},
		{RiskLevel(99), "UNKNOWN"},
		{RiskLevel(3), "UNKNOWN"},
	}

	for _, tt := range tests {
		if got := tt.level.String(); got != tt.want {
			t.Errorf("RiskLevel(%d).String() = %q, want %q", tt.level, got, tt.want)
		}
	}
}

// --- ConfirmAction Edge Cases ---

func TestConfirmAction_EmptyToolName(t *testing.T) {
	cfg := ConfirmConfig{
		ToolName:   "",
		TrustLevel: TrustOfficial,
		RiskLevel:  RiskLow,
		Command:    "echo test",
		CI:         false,
	}
	got := ConfirmAction(cfg)
	if got != ConfirmAuto {
		t.Errorf("ConfirmAction with empty tool name = %v, want ConfirmAuto", got)
	}
}

func TestConfirmAction_EmptyCommand(t *testing.T) {
	cfg := ConfirmConfig{
		ToolName:   "mytool",
		TrustLevel: TrustCustomUntrusted,
		RiskLevel:  RiskLow,
		Command:    "",
		CI:         false,
	}
	got := ConfirmAction(cfg)
	if got != ConfirmProceed {
		t.Errorf("ConfirmAction with empty command = %v, want ConfirmProceed", got)
	}
}

func TestConfirmAction_OfficialAlwaysProceeds(t *testing.T) {
	// Official tools should always auto-proceed regardless of risk
	riskLevels := []RiskLevel{RiskLow, RiskMedium, RiskHigh}

	for _, risk := range riskLevels {
		cfg := ConfirmConfig{
			ToolName:   "brew",
			TrustLevel: TrustOfficial,
			RiskLevel:  risk,
			Command:    "brew upgrade",
			CI:         false,
		}
		got := ConfirmAction(cfg)
		if got != ConfirmAuto {
			t.Errorf("official tool with risk %v should auto-proceed, got %v", risk, got)
		}
	}
}

func TestConfirmAction_CustomUntrusted_CI_AllRisks(t *testing.T) {
	// D4: untrusted custom in CI — Low auto-proceeds, Medium and High error.
	riskLevels := []struct {
		risk RiskLevel
		want ConfirmDecision
	}{
		{RiskLow, ConfirmAuto},
		{RiskMedium, ConfirmError},
		{RiskHigh, ConfirmError},
	}

	for _, tt := range riskLevels {
		cfg := ConfirmConfig{
			ToolName:   "mytool",
			TrustLevel: TrustCustomUntrusted,
			RiskLevel:  tt.risk,
			Command:    "mytool --update",
			CI:         true,
		}
		got := ConfirmAction(cfg)
		if got != tt.want {
			t.Errorf("untrusted custom tool in CI with risk %v = %v, want %v", tt.risk, got, tt.want)
		}
	}
}

func TestConfirmAction_CustomTrusted_CI_RiskBelowHigh(t *testing.T) {
	// Trusted custom tools should auto-proceed in CI if risk < High
	riskLevels := []RiskLevel{RiskLow, RiskMedium}

	for _, risk := range riskLevels {
		cfg := ConfirmConfig{
			ToolName:   "mytool",
			TrustLevel: TrustCustomTrusted,
			RiskLevel:  risk,
			Command:    "mytool --update",
			CI:         true,
		}
		got := ConfirmAction(cfg)
		if got != ConfirmAuto {
			t.Errorf("trusted custom tool in CI with risk %v should auto-proceed, got %v", risk, got)
		}
	}
}

func TestConfirmAction_CustomTrusted_CI_HighRisk(t *testing.T) {
	// Trusted custom tools should error in CI if risk = High
	cfg := ConfirmConfig{
		ToolName:   "mytool",
		TrustLevel: TrustCustomTrusted,
		RiskLevel:  RiskHigh,
		Command:    "sudo mytool --update",
		CI:         true,
	}
	got := ConfirmAction(cfg)
	if got != ConfirmError {
		t.Errorf("trusted custom tool in CI with high risk should error, got %v", got)
	}
}

func TestConfirmAction_CustomHighRisk_Interactive_Prompts(t *testing.T) {
	// High risk should always prompt regardless of trust
	tests := []struct {
		name  string
		trust TrustLevel
		input string
		want  ConfirmDecision
	}{
		{"untrusted yes", TrustCustomUntrusted, "y\n", ConfirmProceed},
		{"untrusted no", TrustCustomUntrusted, "n\n", ConfirmDeny},
		{"trusted yes", TrustCustomTrusted, "y\n", ConfirmProceed},
		{"trusted no", TrustCustomTrusted, "n\n", ConfirmDeny},
		{"yes full word", TrustCustomUntrusted, "yes\n", ConfirmProceed},
		{"no full word", TrustCustomUntrusted, "no\n", ConfirmDeny},
		{"empty input defaults deny", TrustCustomUntrusted, "\n", ConfirmDeny},
		{"uppercase YES", TrustCustomUntrusted, "YES\n", ConfirmProceed},
		{"uppercase NO", TrustCustomUntrusted, "NO\n", ConfirmDeny},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			cfg := ConfirmConfig{
				ToolName:   "mytool",
				TrustLevel: tt.trust,
				RiskLevel:  RiskHigh,
				Command:    "sudo mytool --update",
				CI:         false,
				Privileges: []string{"sudo"},
				Reader:     reader,
			}
			got := ConfirmAction(cfg)
			if got != tt.want {
				t.Errorf("ConfirmAction() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfirmAction_CustomMediumRisk_Interactive_Prompts(t *testing.T) {
	// Untrusted medium risk should prompt
	tests := []struct {
		name  string
		input string
		want  ConfirmDecision
	}{
		{"yes", "y\n", ConfirmProceed},
		{"no", "n\n", ConfirmDeny},
		{"empty", "\n", ConfirmDeny},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			cfg := ConfirmConfig{
				ToolName:   "mytool",
				TrustLevel: TrustCustomUntrusted,
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

func TestConfirmAction_CustomTrustedMediumRisk_Proceeds(t *testing.T) {
	// Trusted custom tool with medium risk: shows info, no prompt
	cfg := ConfirmConfig{
		ToolName:   "mytool",
		TrustLevel: TrustCustomTrusted,
		RiskLevel:  RiskMedium,
		Command:    "mytool --update",
		CI:         false,
	}
	got := ConfirmAction(cfg)
	if got != ConfirmProceed {
		t.Errorf("trusted custom medium risk should proceed, got %v", got)
	}
}

func TestConfirmAction_CustomLowRisk_Proceeds(t *testing.T) {
	// Low risk always proceeds with info
	tests := []struct {
		name  string
		trust TrustLevel
	}{
		{"untrusted", TrustCustomUntrusted},
		{"trusted", TrustCustomTrusted},
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
				t.Errorf("low risk custom tool should proceed, got %v", got)
			}
		})
	}
}

// --- ConfirmConfig Privileges Display ---

func TestConfirmAction_PrivilegesDisplay(t *testing.T) {
	reader := strings.NewReader("n\n")
	cfg := ConfirmConfig{
		ToolName:   "mytool",
		TrustLevel: TrustCustomUntrusted,
		RiskLevel:  RiskHigh,
		Command:    "sudo mytool --update",
		CI:         false,
		Privileges: []string{"sudo", "admin"},
		Reader:     reader,
	}
	got := ConfirmAction(cfg)
	if got != ConfirmDeny {
		t.Errorf("expected ConfirmDeny, got %v", got)
	}
}

func TestConfirmAction_NoPrivileges(t *testing.T) {
	reader := strings.NewReader("n\n")
	cfg := ConfirmConfig{
		ToolName:   "mytool",
		TrustLevel: TrustCustomUntrusted,
		RiskLevel:  RiskHigh,
		Command:    "mytool --update",
		CI:         false,
		Privileges: nil,
		Reader:     reader,
	}
	got := ConfirmAction(cfg)
	if got != ConfirmDeny {
		t.Errorf("expected ConfirmDeny, got %v", got)
	}
}

// --- HighRiskKeywords Coverage ---

func TestHighRiskKeywords_AllCovered(t *testing.T) {
	// Verify all high risk keywords are detected case-insensitively.
	for _, kw := range HighRiskKeywords {
		t.Run(kw, func(t *testing.T) {
			cmd := "test " + kw + " extra"
			got := ClassifyCommand(cmd)
			if got != RiskHigh {
				t.Errorf("keyword %q should classify as RiskHigh, got %v", kw, got)
			}
		})
	}
}

func TestHighRiskKeyword_CurlFsSL_ExactCase(t *testing.T) {
	tests := []struct {
		name string
		cmd  string
		want RiskLevel
	}{
		{"exact case with pipe", "curl -fsSL https://example.com | sh", RiskHigh},
		{"lowercase with pipe", "curl -fssl https://example.com | sh", RiskHigh},
		{"exact case no pipe", "test curl -fsSL extra", RiskHigh},
		{"lowercase no pipe", "test curl -fssl extra", RiskHigh},
		{"uppercase no pipe", "test CURL -FSSL extra", RiskHigh},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyCommand(tt.cmd)
			if got != tt.want {
				t.Errorf("ClassifyCommand(%q) = %v, want %v", tt.cmd, got, tt.want)
			}
		})
	}
}

// --- MediumRiskKeywords Coverage ---

func TestMediumRiskKeywords_AllCovered(t *testing.T) {
	// Verify all medium risk keywords are detected
	for _, kw := range MediumRiskKeywords {
		cmd := "test " + kw + " extra"
		got := ClassifyCommand(cmd)
		if got != RiskMedium {
			t.Errorf("keyword %q should classify as RiskMedium, got %v", kw, got)
		}
	}
}

// --- ConfirmAction Decision Matrix ---

func TestConfirmAction_DecisionMatrix(t *testing.T) {
	// Full decision matrix from design.md D4:
	// CI: Low→Auto; Medium→Auto(trusted)/Err(untrusted); High→Err (even trusted).
	// Interactive: High→prompt (any); Medium→prompt(untrusted)/info(trusted); Low→info.
	tests := []struct {
		name       string
		trustLevel TrustLevel
		risk       RiskLevel
		ci         bool
		input      string
		want       ConfirmDecision
	}{
		// Official tools: always auto-proceed
		{"official low", TrustOfficial, RiskLow, false, "", ConfirmAuto},
		{"official medium", TrustOfficial, RiskMedium, false, "", ConfirmAuto},
		{"official high", TrustOfficial, RiskHigh, false, "", ConfirmAuto},

		// Custom untrusted, CI: low auto (D4), medium/high error
		{"untrusted CI low", TrustCustomUntrusted, RiskLow, true, "", ConfirmAuto},
		{"untrusted CI medium", TrustCustomUntrusted, RiskMedium, true, "", ConfirmError},
		{"untrusted CI high", TrustCustomUntrusted, RiskHigh, true, "", ConfirmError},

		// Custom trusted, CI: auto if risk < high, error if high
		{"trusted CI low", TrustCustomTrusted, RiskLow, true, "", ConfirmAuto},
		{"trusted CI medium", TrustCustomTrusted, RiskMedium, true, "", ConfirmAuto},
		{"trusted CI high", TrustCustomTrusted, RiskHigh, true, "", ConfirmError},

		// Custom untrusted, interactive: prompt if risk >= medium
		{"untrusted interactive low", TrustCustomUntrusted, RiskLow, false, "", ConfirmProceed},
		{"untrusted interactive medium yes", TrustCustomUntrusted, RiskMedium, false, "y\n", ConfirmProceed},
		{"untrusted interactive medium no", TrustCustomUntrusted, RiskMedium, false, "n\n", ConfirmDeny},
		{"untrusted interactive high yes", TrustCustomUntrusted, RiskHigh, false, "y\n", ConfirmProceed},
		{"untrusted interactive high no", TrustCustomUntrusted, RiskHigh, false, "n\n", ConfirmDeny},

		// Custom trusted, interactive: prompt only if high
		{"trusted interactive low", TrustCustomTrusted, RiskLow, false, "", ConfirmProceed},
		{"trusted interactive medium", TrustCustomTrusted, RiskMedium, false, "", ConfirmProceed},
		{"trusted interactive high yes", TrustCustomTrusted, RiskHigh, false, "y\n", ConfirmProceed},
		{"trusted interactive high no", TrustCustomTrusted, RiskHigh, false, "n\n", ConfirmDeny},

		// Zero-value trust (unset): MUST resolve to least-privileged, never auto-proceed
		{"zero trust CI high", TrustLevel(0), RiskHigh, true, "", ConfirmError},
		{"zero trust CI medium", TrustLevel(0), RiskMedium, true, "", ConfirmError},
		{"zero trust interactive high yes", TrustLevel(0), RiskHigh, false, "y\n", ConfirmProceed},
		{"zero trust interactive high no", TrustLevel(0), RiskHigh, false, "n\n", ConfirmDeny},

		// Unknown trust value: must behave as untrusted, never bypass the matrix
		{"unknown trust CI medium", TrustLevel(99), RiskMedium, true, "", ConfirmError},
		{"unknown trust interactive high yes", TrustLevel(99), RiskHigh, false, "y\n", ConfirmProceed},
		{"unknown trust interactive high no", TrustLevel(99), RiskHigh, false, "n\n", ConfirmDeny},

		// Zero-value risk (unset): MUST resolve to High (fail-closed), never RiskLow auto-proceed
		{"zero risk CI untrusted", TrustCustomUntrusted, RiskLevel(0), true, "", ConfirmError},

		// Unknown risk value: default branch treats it as High (fail-closed)
		{"unknown risk CI", TrustCustomUntrusted, RiskLevel(99), true, "", ConfirmError},
		{"unknown risk interactive no", TrustCustomUntrusted, RiskLevel(99), false, "n\n", ConfirmDeny},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var reader *strings.Reader
			if tt.input != "" {
				reader = strings.NewReader(tt.input)
			}

			cfg := ConfirmConfig{
				ToolName:   "mytool",
				TrustLevel: tt.trustLevel,
				RiskLevel:  tt.risk,
				Command:    "mytool --update",
				CI:         tt.ci,
				Reader:     reader,
			}

			got := ConfirmAction(cfg)
			if got != tt.want {
				t.Errorf("ConfirmAction() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestConfirmAction_EnforceRiskFalse_DefaultMatrix pins the DEFAULT path
// (EnforceRisk:false, the zero value) so a future D4 change that raises the
// default cannot silently reclassify existing decisions. This is the
// approval test for WU4 REFACTOR: EnforceRisk:false MUST keep every decision
// byte-identical to the pre-EnforceRisk matrix. Each row mirrors design D4.
func TestConfirmAction_EnforceRiskFalse_DefaultMatrix(t *testing.T) {
	tests := []struct {
		name       string
		trustLevel TrustLevel
		risk       RiskLevel
		ci         bool
		input      string
		want       ConfirmDecision
	}{
		// Official tools: TrustOfficial always auto-proceeds when EnforceRisk
		// is false (the default) — the real command risk is NOT consulted.
		{"official low", TrustOfficial, RiskLow, false, "", ConfirmAuto},
		{"official medium", TrustOfficial, RiskMedium, false, "", ConfirmAuto},
		{"official high", TrustOfficial, RiskHigh, false, "", ConfirmAuto},
		{"official high CI", TrustOfficial, RiskHigh, true, "", ConfirmAuto},

		// Custom untrusted, CI: low auto (D4), medium/high error.
		{"untrusted CI low", TrustCustomUntrusted, RiskLow, true, "", ConfirmAuto},
		{"untrusted CI medium", TrustCustomUntrusted, RiskMedium, true, "", ConfirmError},
		{"untrusted CI high", TrustCustomUntrusted, RiskHigh, true, "", ConfirmError},

		// Custom trusted, CI: auto if risk < high, error if high.
		{"trusted CI low", TrustCustomTrusted, RiskLow, true, "", ConfirmAuto},
		{"trusted CI medium", TrustCustomTrusted, RiskMedium, true, "", ConfirmAuto},
		{"trusted CI high", TrustCustomTrusted, RiskHigh, true, "", ConfirmError},

		// Custom untrusted, interactive: prompt if risk >= medium.
		{"untrusted interactive low", TrustCustomUntrusted, RiskLow, false, "", ConfirmProceed},
		{"untrusted interactive medium yes", TrustCustomUntrusted, RiskMedium, false, "y\n", ConfirmProceed},
		{"untrusted interactive medium no", TrustCustomUntrusted, RiskMedium, false, "n\n", ConfirmDeny},
		{"untrusted interactive high yes", TrustCustomUntrusted, RiskHigh, false, "y\n", ConfirmProceed},
		{"untrusted interactive high no", TrustCustomUntrusted, RiskHigh, false, "n\n", ConfirmDeny},

		// Custom trusted, interactive: prompt only if high.
		{"trusted interactive low", TrustCustomTrusted, RiskLow, false, "", ConfirmProceed},
		{"trusted interactive medium", TrustCustomTrusted, RiskMedium, false, "", ConfirmProceed},
		{"trusted interactive high yes", TrustCustomTrusted, RiskHigh, false, "y\n", ConfirmProceed},
		{"trusted interactive high no", TrustCustomTrusted, RiskHigh, false, "n\n", ConfirmDeny},

		// Zero-value trust (unset): MUST resolve to least-privileged.
		{"zero trust CI high", TrustLevel(0), RiskHigh, true, "", ConfirmError},
		{"zero trust CI medium", TrustLevel(0), RiskMedium, true, "", ConfirmError},
		{"zero trust interactive high yes", TrustLevel(0), RiskHigh, false, "y\n", ConfirmProceed},
		{"zero trust interactive high no", TrustLevel(0), RiskHigh, false, "n\n", ConfirmDeny},

		// Unknown trust value: behave as untrusted, never bypass the matrix.
		{"unknown trust CI medium", TrustLevel(99), RiskMedium, true, "", ConfirmError},
		{"unknown trust interactive high yes", TrustLevel(99), RiskHigh, false, "y\n", ConfirmProceed},

		// Zero-value risk: MUST resolve to High (fail-closed).
		{"zero risk CI untrusted", TrustCustomUntrusted, RiskLevel(0), true, "", ConfirmError},

		// Unknown risk: default branch treats it as High (fail-closed).
		{"unknown risk CI", TrustCustomUntrusted, RiskLevel(99), true, "", ConfirmError},
		{"unknown risk interactive no", TrustCustomUntrusted, RiskLevel(99), false, "n\n", ConfirmDeny},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var reader *strings.Reader
			if tt.input != "" {
				reader = strings.NewReader(tt.input)
			}

			cfg := ConfirmConfig{
				ToolName:   "mytool",
				TrustLevel: tt.trustLevel,
				RiskLevel:  tt.risk,
				Command:    "mytool --update",
				CI:         tt.ci,
				Reader:     reader,
				// EnforceRisk left at its ZERO value (false): the default path.
				// This row MUST reproduce the pre-EnforceRisk decision matrix.
			}

			got := ConfirmAction(cfg)
			if got != tt.want {
				t.Errorf("ConfirmAction(EnforceRisk:false) = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestConfirmAction_EnforceRiskFalse_OfficialNeverPrompts guards the
// TrustOfficial short-circuit under the default: an official tool with
// EnforceRisk:false MUST auto-proceed even for High risk and prompt-producing
// TrustLevel combinations — the official default never consults the reader.
func TestConfirmAction_EnforceRiskFalse_OfficialNeverPrompts(t *testing.T) {
	// reader is an EMPTY reader (no stdin): if ConfirmAction ever consulted
	// the prompt for a TrustOfficial default row, it would hit EOF and return
	// ConfirmDeny. We expect ConfirmAuto, proving no prompt path runs.
	cfg := ConfirmConfig{
		ToolName:    "gh",
		TrustLevel:  TrustOfficial,
		RiskLevel:   RiskHigh,
		Command:     "sudo apt install --only-upgrade gh",
		Privileges:  []string{"sudo"},
		CI:          false,
		EnforceRisk: false,                 // default — the D4 guard must keep ConfirmAuto
		Reader:      strings.NewReader(""), // EOF if a prompt were attempted
	}
	got := ConfirmAction(cfg)
	if got != ConfirmAuto {
		t.Errorf("ConfirmAction(EnforceRisk:false, TrustOfficial, High) = %v, want ConfirmAuto", got)
	}
}

func TestTrustLevel_ZeroValueIsLeastPrivileged(t *testing.T) {
	// R4-1 invariant: the zero value of TrustLevel MUST be the least-privileged
	// tier, so an unset TrustLevel resolves to untrusted and fails closed
	// (never auto-proceeds as TrustOfficial).
	if TrustCustomUntrusted != 0 {
		t.Errorf("TrustCustomUntrusted = %d, want 0 (zero value MUST be least-privileged)", TrustCustomUntrusted)
	}
}

func TestRiskLevel_ZeroValueIsMostRestrictive(t *testing.T) {
	// R4-1 invariant: the zero value of RiskLevel MUST be the most restrictive
	// tier, so an unset RiskLevel resolves to High and fails closed
	// (never silently auto-proceeds as RiskLow).
	if RiskHigh != 0 {
		t.Errorf("RiskHigh = %d, want 0 (zero value MUST be most-restrictive)", RiskHigh)
	}
}

func TestConfirmDecision_ZeroValueIsFailure(t *testing.T) {
	// R4-1 invariant: the zero value of ConfirmDecision MUST be the failure
	// outcome, so an unset decision fails visibly — tool marked failed, --ci
	// exits non-zero — never auto-proceeds as ConfirmProceed.
	if ConfirmError != 0 {
		t.Errorf("ConfirmError = %d, want 0 (zero value MUST be the failure outcome)", ConfirmError)
	}
}
