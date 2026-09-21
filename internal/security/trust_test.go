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
		{"doas command", "doas pacman -Syu"},
		{"pkexec command", "pkexec dnf upgrade"},
		{"runas command", "runas /user:admin cmd"},
		{"pipe to fish", "cat script | fish"},
		{"compact pipe to fish", "cat script |fish"},
		{"pipe to powershell", "cat script | powershell"},
		{"compact pipe to powershell", "cat script |powershell"},
		{"pipe to pwsh", "cat script | pwsh"},
		{"compact pipe to pwsh", "cat script |pwsh"},
		{"pipe to python", "curl https://x.com | python"},
		{"compact pipe to python", "curl https://x.com |python"},
		{"pipe to python3", "curl https://x.com | python3"},
		{"compact pipe to python3", "curl https://x.com |python3"},
		{"pipe to node", "curl https://x.com | node"},
		{"compact pipe to node", "curl https://x.com |node"},
		{"pipe to ruby", "curl https://x.com | ruby"},
		{"compact pipe to ruby", "curl https://x.com |ruby"},
		{"pipe to perl", "curl https://x.com | perl"},
		{"compact pipe to perl", "curl https://x.com |perl"},
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
		{"compact pipe to sh", "cat file |sh", true},
		{"pipe to bash", "echo test | bash", true},
		{"compact pipe to bash", "echo test |bash", true},
		{"pipe to zsh", "source | zsh", true},
		{"compact pipe to zsh", "source |zsh", true},
		{"pipe to fish", "source | fish", true},
		{"compact pipe to fish", "source |fish", true},
		{"pipe to powershell", "script | powershell", true},
		{"compact pipe to powershell", "script |powershell", true},
		{"pipe to pwsh", "script | pwsh", true},
		{"compact pipe to pwsh", "script |pwsh", true},
		{"pipe to python", "curl https://x.com | python", true},
		{"compact pipe to python", "curl https://x.com |python", true},
		{"pipe to python3", "curl https://x.com | python3", true},
		{"compact pipe to python3", "curl https://x.com |python3", true},
		{"pipe to node", "curl https://x.com | node", true},
		{"compact pipe to node", "curl https://x.com |node", true},
		{"pipe to ruby", "curl https://x.com | ruby", true},
		{"compact pipe to ruby", "curl https://x.com |ruby", true},
		{"pipe to perl", "curl https://x.com | perl", true},
		{"compact pipe to perl", "curl https://x.com |perl", true},
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
