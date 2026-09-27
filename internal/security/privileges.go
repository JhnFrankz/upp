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

	if reSudo.MatchString(cmd) {
		privs = append(privs, "sudo")
	}
	if reDoas.MatchString(cmd) {
		privs = append(privs, "doas")
	}
	if rePkexec.MatchString(cmd) {
		privs = append(privs, "pkexec")
	}
	if reSu.MatchString(cmd) {
		privs = append(privs, "su")
	}
	if reRunas.MatchString(cmd) {
		privs = append(privs, "runas")
	}
	if reAdmin.MatchString(cmd) {
		privs = append(privs, "admin")
	}
	return privs
}
