package security

import "strings"

// DetectPrivileges inspects a command string for privilege escalation tokens.
// It detects Unix privilege escalators (sudo, doas, pkexec, su) and Windows
// elevation tokens (runas, admin).
func DetectPrivileges(cmd string) []string {
	lower := strings.ToLower(cmd)
	var privs []string
	seen := make(map[string]bool)

	add := func(p string) {
		if !seen[p] {
			seen[p] = true
			privs = append(privs, p)
		}
	}

	if strings.Contains(lower, "sudo") {
		add("sudo")
	}
	if strings.Contains(lower, "doas") {
		add("doas")
	}
	if strings.Contains(lower, "pkexec") {
		add("pkexec")
	}
	if strings.Contains(lower, " su ") || strings.HasPrefix(lower, "su ") || strings.HasSuffix(lower, " su") || lower == "su" {
		add("su")
	}
	if strings.Contains(lower, "runas") {
		add("runas")
	}
	if strings.Contains(lower, "admin") {
		add("admin")
	}
	return privs
}
