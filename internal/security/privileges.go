package security

import "regexp"

var (
	reSudo   = regexp.MustCompile(`(?i)\bsudo\b`)
	reDoas   = regexp.MustCompile(`(?i)\bdoas\b`)
	rePkexec = regexp.MustCompile(`(?i)\bpkexec\b`)
	reSu     = regexp.MustCompile(`(?i)\bsu\b`)
	reRunas  = regexp.MustCompile(`(?i)\brunas(\b|admin)`)
	reAdmin  = regexp.MustCompile(`(?i)(\b|runas)admin(istrator)?\b`)
)

// DetectPrivileges inspects a command string for privilege escalation tokens.
// It detects Unix privilege escalators (sudo, doas, pkexec, su) and Windows
// elevation tokens (runas, admin).
func DetectPrivileges(cmd string) []string {
	var privs []string
	seen := make(map[string]bool)

	add := func(p string) {
		if !seen[p] {
			seen[p] = true
			privs = append(privs, p)
		}
	}

	if reSudo.MatchString(cmd) {
		add("sudo")
	}
	if reDoas.MatchString(cmd) {
		add("doas")
	}
	if rePkexec.MatchString(cmd) {
		add("pkexec")
	}
	if reSu.MatchString(cmd) {
		add("su")
	}
	if reRunas.MatchString(cmd) {
		add("runas")
	}
	if reAdmin.MatchString(cmd) {
		add("admin")
	}
	return privs
}
