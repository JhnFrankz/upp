package platform

import (
	"strings"
)

// TokenizeShellCommand tokenizes a shell command string into arguments, respecting
// single and double quotes, backslash escapes, and statement separators (; && || | \n).
// It returns tokens for the first command statement encountered.
func TokenizeShellCommand(command string) []string {
	var tokens []string
	var cur strings.Builder
	inToken := false
	var quote byte

	for i := 0; i < len(command); i++ {
		ch := command[i]
		if quote != 0 {
			if ch == quote {
				quote = 0
				continue
			}
			if quote == '"' && ch == '\\' && i+1 < len(command) {
				next := command[i+1]
				if next == '\n' {
					i++
					continue
				}
				if next == '\r' && i+2 < len(command) && command[i+2] == '\n' {
					i += 2
					continue
				}
				if next == '"' || next == '\\' || next == '$' || next == '`' {
					cur.WriteByte(next)
					i++
					continue
				}
			}
			cur.WriteByte(ch)
			continue
		}

		if ch == '\'' || ch == '"' {
			quote = ch
			inToken = true
			continue
		}

		if ch == '\\' && i+1 < len(command) {
			next := command[i+1]
			if next == '\n' {
				i++
				continue
			}
			if next == '\r' && i+2 < len(command) && command[i+2] == '\n' {
				i += 2
				continue
			}
			if next == ' ' || next == '\t' || next == '"' || next == '\'' || next == ';' || next == '|' || next == '&' || next == '\\' {
				cur.WriteByte(next)
				inToken = true
				i++
				continue
			}
		}

		if ch == ' ' || ch == '\t' || ch == '\r' {
			if inToken {
				tokens = append(tokens, cur.String())
				cur.Reset()
				inToken = false
			}
			continue
		}

		if ch == ';' || ch == '|' || ch == '&' || ch == '\n' {
			if inToken {
				tokens = append(tokens, cur.String())
				cur.Reset()
				inToken = false
			}
			if len(tokens) > 0 {
				break
			}
			continue
		}

		cur.WriteByte(ch)
		inToken = true
	}

	if inToken {
		tokens = append(tokens, cur.String())
	}
	return tokens
}

// ExtractBaseCommand returns the base executable name for a command string.
// It unwraps known privilege and shell wrappers, returning the underlying
// executable or an empty string if none is found.
func ExtractBaseCommand(command string) string {
	return ExtractCommandBinary(command, "")
}

// ExtractCommandBinary unwraps wrappers (sudo, doas, gsudo, runas, su, env,
// shell interpreters sh, bash, zsh, cmd, powershell, pwsh), skips environment
// variable assignments (KEY=val), and returns the target executable or fallbackName.
func ExtractCommandBinary(command, fallbackName string) string {
	tokens := TokenizeShellCommand(command)
	if len(tokens) == 0 {
		return fallbackName
	}

	for i := 0; i < len(tokens); i++ {
		tok := tokens[i]
		if tok == "" || isEnvVarAssignment(tok) {
			continue
		}

		baseLower := normalizedWrapperName(tok)
		if baseLower == "env" {
			for i+1 < len(tokens) {
				next := tokens[i+1]
				if next == "--" {
					i++
					break
				}
				if next == "-u" || next == "--unset" || next == "-C" || next == "--chdir" {
					i += 2
					continue
				}
				if next == "-S" || next == "--split-string" {
					if i+2 < len(tokens) {
						if inner := ExtractCommandBinary(tokens[i+2], ""); inner != "" {
							return inner
						}
					}
					i += 2
					continue
				}
				if strings.HasPrefix(next, "-") {
					i++
					continue
				}
				break
			}
			continue
		}

		if baseLower == "su" {
			for i+1 < len(tokens) {
				next := tokens[i+1]
				if next == "--" {
					i++
					break
				}
				if next == "-c" || next == "--command" || (strings.HasPrefix(next, "-") && !strings.HasPrefix(next, "--") && strings.HasSuffix(next, "c")) {
					if i+2 < len(tokens) {
						if inner := ExtractCommandBinary(tokens[i+2], ""); inner != "" {
							return inner
						}
					}
					i += 2
					continue
				}
				if strings.HasPrefix(next, "--command=") {
					if inner := ExtractCommandBinary(strings.TrimPrefix(next, "--command="), ""); inner != "" {
						return inner
					}
					i++
					continue
				}
				if isPrivilegeFlagWithArg(baseLower, next) {
					i += 2
					continue
				}
				if strings.HasPrefix(next, "-") || hasSuCommandFlagAhead(tokens[i+2:]) {
					i++
					continue
				}
				break
			}
			continue
		}

		if isPrivilegeWrapper(baseLower) {
			for i+1 < len(tokens) {
				next := tokens[i+1]
				if next == "--" {
					i++
					break
				}
				if isPrivilegeFlagWithArg(baseLower, next) {
					i += 2
					continue
				}
				if strings.HasPrefix(next, "-") || (baseLower == "runas" && strings.HasPrefix(next, "/")) {
					i++
					continue
				}
				break
			}
			continue
		}

		if isShellWrapper(baseLower) {
			if inner := extractShellCommandTarget(baseLower, tokens[i+1:]); inner != "" {
				return inner
			}
		}

		return tok
	}

	return fallbackName
}

func normalizedWrapperName(tok string) string {
	base := tok
	if idx := strings.LastIndexAny(base, `/\`); idx >= 0 {
		base = base[idx+1:]
	}
	baseLower := strings.ToLower(base)
	return strings.TrimSuffix(baseLower, ".exe")
}

func hasSuCommandFlagAhead(rest []string) bool {
	for _, t := range rest {
		if t == "-c" || t == "--command" || strings.HasPrefix(t, "--command=") ||
			(strings.HasPrefix(t, "-") && !strings.HasPrefix(t, "--") && strings.HasSuffix(t, "c")) {
			return true
		}
	}
	return false
}

func isShellWrapper(baseLower string) bool {
	switch baseLower {
	case "sh", "bash", "zsh", "dash", "ksh", "cmd", "powershell", "pwsh":
		return true
	default:
		return false
	}
}

func extractShellCommandTarget(shell string, args []string) string {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch shell {
		case "cmd":
			lower := strings.ToLower(arg)
			if (lower == "/c" || lower == "/k") && i+1 < len(args) {
				inner := ExtractCommandBinary(args[i+1], "")
				if inner != "" && !isShellBuiltin(normalizedWrapperName(inner)) {
					return inner
				}
				return ""
			}
		case "powershell", "pwsh":
			lower := strings.ToLower(arg)
			if (lower == "-c" || lower == "-command" || lower == "--command" || lower == "/c" || lower == "/command") && i+1 < len(args) {
				inner := ExtractCommandBinary(args[i+1], "")
				if inner != "" && !isShellBuiltin(normalizedWrapperName(inner)) {
					return inner
				}
				return ""
			}
		default:
			if arg == "-c" || (strings.HasPrefix(arg, "-") && !strings.HasPrefix(arg, "--") && strings.HasSuffix(arg, "c")) {
				if i+1 < len(args) {
					inner := ExtractCommandBinary(args[i+1], "")
					if inner != "" && !isShellBuiltin(normalizedWrapperName(inner)) {
						return inner
					}
				}
				return ""
			}
			if !strings.HasPrefix(arg, "-") {
				return ""
			}
		}
	}
	return ""
}

func isShellBuiltin(name string) bool {
	switch name {
	case "echo", "printf", "true", "false", "cd", "pwd", "export", "unset", "set", "exit", "return", "test", "[", ":", "eval", "exec", "source", ".":
		return true
	default:
		return false
	}
}

func isEnvVarAssignment(tok string) bool {
	idx := strings.IndexByte(tok, '=')
	if idx <= 0 {
		return false
	}
	key := tok[:idx]
	for i := 0; i < len(key); i++ {
		if !isEnvKeyChar(key[i], i == 0) {
			return false
		}
	}
	return true
}

func isEnvKeyChar(c byte, first bool) bool {
	if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_' {
		return true
	}
	return !first && c >= '0' && c <= '9'
}

func isPrivilegeWrapper(baseLower string) bool {
	switch baseLower {
	case "sudo", "doas", "pkexec", "su", "runas", "gsudo":
		return true
	default:
		return false
	}
}

func isPrivilegeFlagWithArg(wrapper, flag string) bool {
	if strings.ContainsRune(flag, '=') {
		return false
	}
	switch wrapper {
	case "sudo":
		switch flag {
		case "-u", "--user", "-g", "--group", "-C", "--close-from", "-D", "--chdir", "-R", "--chroot", "-T", "--command-timeout", "-h", "--host", "-p", "--prompt", "-r", "--role", "-t", "--type", "-U", "--other-user":
			return true
		}
		if strings.HasPrefix(flag, "-") && !strings.HasPrefix(flag, "--") && len(flag) > 2 {
			switch flag[len(flag)-1] {
			case 'u', 'g', 'C', 'D', 'R', 'T', 'h', 'p', 'r', 't', 'U':
				return true
			}
		}
	case "doas":
		switch flag {
		case "-u", "-C", "-a":
			return true
		}
		if strings.HasPrefix(flag, "-") && !strings.HasPrefix(flag, "--") && len(flag) > 2 {
			switch flag[len(flag)-1] {
			case 'u', 'C', 'a':
				return true
			}
		}
	case "pkexec":
		if flag == "--user" {
			return true
		}
	case "su":
		switch flag {
		case "-s", "--shell", "-g", "--group", "-G", "--supp-group":
			return true
		}
	case "gsudo":
		switch flag {
		case "-u", "--user":
			return true
		}
	}
	return false
}
