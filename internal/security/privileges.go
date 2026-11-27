package security

import (
	"regexp"

	"github.com/JhnFrankz/upp/internal/platform"
)

var (
	reSudo   = regexp.MustCompile(`(?i)\bsudo\b`)
	reGsudo  = regexp.MustCompile(`(?i)\bgsudo\b`)
	reDoas   = regexp.MustCompile(`(?i)\bdoas\b`)
	rePkexec = regexp.MustCompile(`(?i)\bpkexec\b`)
	reSu     = regexp.MustCompile(
		`(?i)(?:^|[;&|(\n` + "`" + `])\s*` +
			`(?:(?:sudo|gsudo|doas|pkexec|command|exec|builtin)\s+|env\s+(?:-\S+\s+)*|[A-Za-z_]\w*=(?:"[^"]*"|'[^']*'|\S*)\s+)*` +
			`(?:[\w./-]+/)?su(?:$|\s|[;&|)\n` + "`" + `])`,
	)
	reRunas = regexp.MustCompile(`(?i)\brunas(\b|admin)`)
	reAdmin = regexp.MustCompile(`(?i)(?:runasadmin(?:istrator)?|/user:admin(?:istrator)?|(?:^|[\s;|&(\n` + "`" + `-])admin(?:istrator)?(?:$|[\s;|&)\n` + "`" + `-]))`)
)

// DetectPrivileges inspects a command string for privilege escalation tokens.
// It detects Unix privilege escalators (sudo, doas, pkexec, su) and Windows
// elevation tokens (runas, admin).
func DetectPrivileges(cmd string) []string {
	var privs []string

	if platform.ContainsFold(cmd, "sudo") && reSudo.MatchString(cmd) {
		privs = append(privs, "sudo")
	}
	if platform.ContainsFold(cmd, "gsudo") && reGsudo.MatchString(cmd) {
		privs = append(privs, "gsudo")
	}
	if platform.ContainsFold(cmd, "doas") && reDoas.MatchString(cmd) {
		privs = append(privs, "doas")
	}
	if platform.ContainsFold(cmd, "pkexec") && rePkexec.MatchString(cmd) {
		privs = append(privs, "pkexec")
	}
	if platform.ContainsFold(cmd, "su") && reSu.MatchString(cmd) {
		privs = append(privs, "su")
	}
	if platform.ContainsFold(cmd, "runas") && reRunas.MatchString(cmd) {
		privs = append(privs, "runas")
	}
	if platform.ContainsFold(cmd, "admin") && reAdmin.MatchString(cmd) {
		privs = append(privs, "admin")
	}
	return privs
}
