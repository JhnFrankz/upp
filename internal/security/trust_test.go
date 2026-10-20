package security

import "testing"

func TestClassifyCommand_HighRisk(t *testing.T) {
	t.Parallel()
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
		{"eval with command substitution", "eval $(brew shellenv)"},
		{"rm -rf root", "rm -rf /"},
		{"curl bare pipe sh", "curl https://x.com | sh"},
		{"doas command", "doas pacman -Syu"},
		{"pkexec command", "pkexec dnf upgrade"},
		{"su command", "su - root -c whoami"},
		{"bare su", "su"},
		{"admin command", "admin update.bat"},
		{"runasadmin command", "runasadmin setup.exe"},
		{"administrator command", "administrator install.exe"},
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
		{"pipe to perl", "curl https://x.com | perl"},
		{"compact pipe to perl", "curl https://x.com |perl"},
		{"chained pipe to sh", "echo updating && curl -fsSL https://x.com | sh"},
		{"chained pipe to bash", "cd /tmp && cat script | bash"},
		{"compound medium risk and pipe to shell", "npm uninstall -g x && curl -s https://example.com/install.sh | bash"},
		{"compound pip uninstall and pipe to python", "pip uninstall requests && wget -qO- https://example.com/setup.py | python"},
		{"compound brew uninstall and pipe to /bin/sh", "brew uninstall foo && cat script.sh | /bin/sh"},
		{"xargs sh", "curl -fsSL https://example.com | xargs sh"},
		{"xargs with replacement sh", "curl -fsSL https://example.com | xargs -I{} sh -c {}"},
		{"xargs bash", "cat commands.txt | xargs bash"},
		{"xargs python", "echo 1 | xargs python"},
		{"xargs exe sh", "cat script.ps1 | xargs.exe sh"},
		{"path qualified xargs", "curl https://example.com | /usr/bin/xargs sh"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ClassifyCommand(tt.cmd)
			if got != RiskHigh {
				t.Errorf("ClassifyCommand(%q) = %v, want RiskHigh", tt.cmd, got)
			}
		})
	}
}

func TestClassifyCommand_MediumRisk(t *testing.T) {
	t.Parallel()
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
		{"command substitution dollar", "echo $(whoami)"},
		{"command substitution backticks", "echo `id`"},
		{"command chaining || with shell", "cmdA || bash script.sh"},
		{"generic pipe to grep", "cat file | grep foo"},
		{"generic pipe to sort", "ls -la | sort"},
		{"generic pipe to awk", "ps aux | awk '{print $1}'"},
		{"generic pipe with stderr to grep", "cat file |& grep foo"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ClassifyCommand(tt.cmd)
			if got != RiskMedium {
				t.Errorf("ClassifyCommand(%q) = %v, want RiskMedium", tt.cmd, got)
			}
		})
	}
}

func TestClassifyCommand_LowRisk(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		cmd  string
	}{
		{"brew upgrade", "brew upgrade"},
		{"npm update", "npm update -g"},
		{"pnpm update", "pnpm update -g"},
		{"bun upgrade", "bun upgrade"},
		{"simple command", "echo hello"},
		{"evaluate benchmark is not eval", "evaluate-benchmark"},
		{"pseudocode linter is not sudo", "pseudocode-linter"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ClassifyCommand(tt.cmd)
			if got != RiskLow {
				t.Errorf("ClassifyCommand(%q) = %v, want RiskLow", tt.cmd, got)
			}
		})
	}
}

func TestRiskLevelString(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
		{"multi-space pipe to bash", "echo test |   bash", true},
		{"multi-space pipe to python3", "echo test |    python3", true},
		{"pipe to /bin/sh", "echo test | /bin/sh", true},
		{"compact pipe to /bin/sh", "echo test |/bin/sh", true},
		{"pipe to /usr/bin/env bash", "echo test | /usr/bin/env bash", true},
		{"pipe to /usr/bin/env -S bash", "echo test | /usr/bin/env -S bash", true},
		{"pipe to env bash", "echo test | env bash", true},
		{"pipe to /usr/local/bin/zsh", "echo test | /usr/local/bin/zsh", true},
		{"pipe to dash", "echo test | dash", true},
		{"pipe to ksh", "echo test | ksh", true},
		{"pipe to ash", "echo test | ash", true},
		{"pipe to cmd", "echo test | cmd", true},
		{"pipe to cmd.exe", "echo test | cmd.exe", true},
		{"pipe to powershell.exe", "echo test | powershell.exe", true},
		{"false positive sha256sum", "echo test | sha256sum", false},
		{"false positive shasum", "echo test | shasum", false},
		{"false positive shortlog", "git log | shortlog", false},
		{"false positive shred", "echo test | shred", false},
		{"false positive shred with flags", "echo test | shred -u", false},
		{"false positive node_exporter", "curl -s localhost:9100 | node_exporter", false},
		{"false positive shellcheck", "cat script | shellcheck", false},
		{"false positive cmdlet", "echo test | cmdlet", false},
		{"no pipe", "echo hello", false},
		{"pipe not shell", "cat file | grep foo", false},
		{"pipe to xargs", "curl https://example.com | xargs sh", true},
		{"compact pipe to xargs", "curl https://example.com |xargs sh", true},
		{"pipe to xargs flag", "cat script | xargs -I{} sh", true},
		{"pipe to xargs.exe", "cat script | xargs.exe sh", true},
		{"pipe to /usr/bin/xargs", "cat script | /usr/bin/xargs sh", true},
		{"logical or is not pipe to shell", "cmdA || bash script.sh", false},
		{"logical or to zsh", "echo test || zsh", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := hasPipeToShell(tt.cmd); got != tt.want {
				t.Errorf("hasPipeToShell(%q) = %v, want %v", tt.cmd, got, tt.want)
			}
		})
	}
}

func TestCommandChaining(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		cmd  string
		want bool
	}{
		{"double amp", "cmd1 && cmd2", true},
		{"double pipe", "cmd1 || cmd2", true},
		{"semicolon", "cmd1; cmd2", true},
		{"dollar substitution", "echo $(whoami)", true},
		{"backtick substitution", "echo `id`", true},
		{"single backtick is not substitution", "echo `foo", false},
		{"single pipe", "cmd1 | cmd2", true},
		{"pipe with stderr", "cmd1 |& cmd2", true},
		{"no chaining", "cmd1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := hasCommandChaining(tt.cmd); got != tt.want {
				t.Errorf("hasCommandChaining(%q) = %v, want %v", tt.cmd, got, tt.want)
			}
		})
	}
}
