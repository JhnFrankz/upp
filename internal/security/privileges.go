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

// containsFold reports whether substr is within s, using ASCII case-insensitivity without heap allocations.
func containsFold(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	limit := len(s) - len(substr)
	for i := 0; i <= limit; i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			c1 := s[i+j]
			c2 := substr[j]
			if c1 == c2 {
				continue
			}
			if c1 >= 'A' && c1 <= 'Z' {
				c1 += 'a' - 'A'
			}
			if c2 >= 'A' && c2 <= 'Z' {
				c2 += 'a' - 'A'
			}
			if c1 != c2 {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// DetectPrivileges inspects a command string for privilege escalation tokens.
// It detects Unix privilege escalators (sudo, doas, pkexec, su) and Windows
// elevation tokens (runas, admin).
func DetectPrivileges(cmd string) []string {
	var privs []string

	if containsFold(cmd, "sudo") && reSudo.MatchString(cmd) {
		privs = append(privs, "sudo")
	}
	if containsFold(cmd, "doas") && reDoas.MatchString(cmd) {
		privs = append(privs, "doas")
	}
	if containsFold(cmd, "pkexec") && rePkexec.MatchString(cmd) {
		privs = append(privs, "pkexec")
	}
	if containsFold(cmd, "su") && reSu.MatchString(cmd) {
		privs = append(privs, "su")
	}
	if containsFold(cmd, "runas") && reRunas.MatchString(cmd) {
		privs = append(privs, "runas")
	}
	if containsFold(cmd, "admin") && reAdmin.MatchString(cmd) {
		privs = append(privs, "admin")
	}
	return privs
}
