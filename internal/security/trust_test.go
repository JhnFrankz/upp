package security

import "testing"

func TestClassifyCommand_HighRisk(t *testing.T) {
	tests := []struct {
		name string
		cmd  string
	}{
		{"sudo command", "sudo apt upgrade"},
		{"rm -rf", "rm -rf /tmp/foo"},
		{"rm -r /", "rm -r /tmp"},
		{"curl pipe sh", "curl -fsSL https://example.com | sh"},
		{"wget pipe sh", "wget -qO- https://example.com | sh"},
		{"eval", "eval $(something)"},
		{"rm -rf root", "rm -rf /"},
		{"curl bare pipe sh", "curl https://x.com | sh"},
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

func TestClassifyCommand_MediumRisk(t *testing.T) {
	tests := []struct {
		name string
		cmd  string
	}{
		{"apt remove", "apt remove nginx"},
		{"brew uninstall", "brew uninstall node"},
		{"npm uninstall -g", "npm uninstall -g typescript"},
		{"command chaining &&", "apt update && apt upgrade"},
		{"command chaining ||", "cmd1 || cmd2"},
		{"command chaining ;", "cmd1; cmd2"},
		{"pip uninstall", "pip uninstall requests"},
		{"apt purge", "apt purge old-package"},
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

func TestClassifyCommand_LowRisk(t *testing.T) {
	tests := []struct {
		name string
		cmd  string
	}{
		{"brew upgrade", "brew upgrade"},
		{"npm update", "npm update -g"},
		{"pnpm update", "pnpm update -g"},
		{"bun upgrade", "bun upgrade"},
		{"simple command", "echo hello"},
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

func TestRiskLevelString(t *testing.T) {
	tests := []struct {
		level RiskLevel
		want  string
	}{
		{RiskLow, "LOW"},
		{RiskMedium, "MEDIUM"},
		{RiskHigh, "HIGH"},
		{RiskLevel(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		if got := tt.level.String(); got != tt.want {
			t.Errorf("RiskLevel(%d).String() = %q, want %q", tt.level, got, tt.want)
		}
	}
}

func TestPipeToShell(t *testing.T) {
	tests := []struct {
		name string
		cmd  string
		want bool
	}{
		{"pipe to sh", "cat file | sh", true},
		{"pipe to bash", "echo test | bash", true},
		{"pipe to zsh", "source | zsh", true},
		{"no pipe", "echo hello", false},
		{"pipe not shell", "cat file | grep foo", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasPipeToShell(tt.cmd); got != tt.want {
				t.Errorf("hasPipeToShell(%q) = %v, want %v", tt.cmd, got, tt.want)
			}
		})
	}
}

func TestCommandChaining(t *testing.T) {
	tests := []struct {
		name string
		cmd  string
		want bool
	}{
		{"double amp", "cmd1 && cmd2", true},
		{"double pipe", "cmd1 || cmd2", true},
		{"semicolon", "cmd1; cmd2", true},
		{"no chaining", "cmd1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasCommandChaining(tt.cmd); got != tt.want {
				t.Errorf("hasCommandChaining(%q) = %v, want %v", tt.cmd, got, tt.want)
			}
		})
	}
}
