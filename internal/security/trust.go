// Package security implements trust levels, risk classification, and
// confirmation prompts for tool execution safety.
package security

import (
	"regexp"
	"strings"

	"github.com/JhnFrankz/upp/internal/platform"
)

// TrustLevel represents how much the system trusts a tool adapter.
type TrustLevel int

const (
	// TrustCustomUntrusted is for custom adapters, untrusted by default.
	// It is the ZERO value on purpose: an unset TrustLevel MUST resolve to the
	// least-privileged level so unset trust fails closed. The zero value MUST
	// stay the least-privileged tier — never insert a new level before it.
	TrustCustomUntrusted TrustLevel = 0
	// TrustCustomTrusted is for custom adapters marked trusted=true in config.
	// It must never alias TrustOfficial: trust level MUST NOT bypass the risk matrix.
	TrustCustomTrusted TrustLevel = 1
	// TrustOfficial is for official, built-in adapters.
	TrustOfficial TrustLevel = 2
)

// String returns a human-readable trust label.
func (t TrustLevel) String() string {
	switch t {
	case TrustOfficial:
		return "official"
	case TrustCustomTrusted:
		return "custom-trusted"
	case TrustCustomUntrusted:
		return "custom-untrusted"
	default:
		return "unknown"
	}
}

// RiskLevel classifies how dangerous a command is.
type RiskLevel int

const (
	// RiskHigh is the ZERO value on purpose: an unset RiskLevel MUST resolve to
	// the most restrictive member so unclassified risk fails closed. The zero
	// value MUST stay the most restrictive tier — never insert a tier before it.
	RiskHigh RiskLevel = 0
	// RiskMedium may modify system state.
	RiskMedium RiskLevel = 1
	// RiskLow is non-destructive, no privileges required.
	RiskLow RiskLevel = 2
)

// String returns a human-readable risk label.
func (r RiskLevel) String() string {
	switch r {
	case RiskLow:
		return "LOW"
	case RiskMedium:
		return "MEDIUM"
	case RiskHigh:
		return "HIGH"
	default:
		return "UNKNOWN"
	}
}

var reEval = regexp.MustCompile(`(?i)\beval\b`)

// HighRiskKeywords are substrings that immediately classify a command as high risk.
var HighRiskKeywords = []string{
	"rm -rf",
	"rm -r /",
	"curl|sh",
	"curl | sh",
	"curl -fsSL",
	"wget|sh",
	"wget | sh",
	"rm -rf /",
}

// MediumRiskKeywords are substrings that classify a command as medium risk.
var MediumRiskKeywords = []string{
	"apt remove",
	"brew uninstall",
	"npm uninstall -g",
	"pnpm uninstall -g",
	"pip uninstall",
	"apt purge",
}

var (
	highRiskKeywordsLower   []string
	mediumRiskKeywordsLower []string
)

func init() {
	syncRiskKeywordsLower()
}

func syncRiskKeywordsLower() {
	highRiskKeywordsLower = make([]string, len(HighRiskKeywords))
	for i, kw := range HighRiskKeywords {
		highRiskKeywordsLower[i] = strings.ToLower(kw)
	}
	mediumRiskKeywordsLower = make([]string, len(MediumRiskKeywords))
	for i, kw := range MediumRiskKeywords {
		mediumRiskKeywordsLower[i] = strings.ToLower(kw)
	}
}

// ClassifyCommand uses a hybrid approach to determine the risk level of a command.
// It checks keyword matching first, then pattern matching for chaining/piping.
func ClassifyCommand(cmd string) RiskLevel {
	if platform.ContainsFold(cmd, "eval") && reEval.MatchString(cmd) {
		return RiskHigh
	}
	if len(DetectPrivileges(cmd)) > 0 {
		return RiskHigh
	}

	// 1. Keyword matching — high risk first (short-circuits).
	for _, kw := range highRiskKeywordsLower {
		if platform.ContainsFold(cmd, kw) {
			return RiskHigh
		}
	}

	// 2. Pattern matching — pipe to shell is always high risk.
	// Evaluated before medium-risk keywords so compound commands with medium-risk
	// verbs that pipe into a shell (e.g., npm uninstall -g x && curl ... | bash)
	// correctly evaluate to RiskHigh.
	if hasPipeToShell(cmd) {
		return RiskHigh
	}

	// 3. Keyword matching — medium risk.
	for _, kw := range mediumRiskKeywordsLower {
		if platform.ContainsFold(cmd, kw) {
			return RiskMedium
		}
	}

	// 4. Pattern matching — command chaining increases risk.
	if hasCommandChaining(cmd) {
		return RiskMedium
	}

	return RiskLow
}

// CheckNeedsConsent reports whether an adapter's declared check command is
// dangerous enough to require consent before it runs. A tool that declares no
// check command, or one classified RiskLow, needs no consent.
//
// A check command is arbitrary shell, so it is classified by its real risk
// exactly like an update command — never by the tool's trust level alone
// (spec security-model: custom check-command gate).
func CheckNeedsConsent(checkCmd string) bool {
	return checkCmd != "" && ClassifyCommand(checkCmd) != RiskLow
}

// hasCommandChaining detects command chaining operators and generic shell pipelines.
func hasCommandChaining(cmd string) bool {
	if !strings.ContainsAny(cmd, "&|;$`") {
		return false
	}
	return strings.Contains(cmd, "&&") ||
		strings.Contains(cmd, "||") ||
		strings.Contains(cmd, ";") ||
		strings.Contains(cmd, "$(") ||
		hasPipe(cmd) ||
		(strings.Contains(cmd, "`") && strings.Count(cmd, "`") >= 2)
}

// hasPipe detects whether a command contains a pipe operator (|),
// ignoring the logical OR operator (||).
func hasPipe(cmd string) bool {
	n := len(cmd)
	for i := 0; i < n; i++ {
		if cmd[i] != '|' {
			continue
		}
		if i+1 < n && cmd[i+1] == '|' {
			i++
			continue
		}
		return true
	}
	return false
}

var rePipeShell = regexp.MustCompile(
	`(?i)^(?:\s*(?:sudo|doas|pkexec)\s+)?` +
		`(?:\s*['"]?(?:[\w.:/\\ -]+[/\\])?env(?:\.exe)?['"]?(?:\s+-\S+)*\s+)?` +
		`['"]?(?:[\w.:/\\ -]+[/\\])?` +
		`(?:sh|bash|zsh|fish|dash|ksh|ash|powershell|pwsh|cmd|python3|python|node|ruby|perl|xargs)` +
		`(?:\.exe)?['"]?(?:$|[\s;&|<>'"])`,
)

// hasPipeToShell detects piping output to a shell or script interpreter,
// ensuring token/word boundaries after the interpreter name to avoid false
// positives on tools sharing prefixes (sha256sum, node_exporter), and handling
// multi-space pipes, path-qualified shells (/bin/sh, /usr/bin/env bash), execution
// targets like xargs, and shells like dash, ksh, ash, and cmd.
func hasPipeToShell(cmd string) bool {
	if !strings.ContainsRune(cmd, '|') {
		return false
	}
	n := len(cmd)
	for i := 0; i < n; i++ {
		if cmd[i] != '|' {
			continue
		}

		// Skip logical OR (||)
		if i+1 < n && cmd[i+1] == '|' {
			i++
			continue
		}

		start := i + 1
		// Skip bash |& (pipe stderr)
		if start < n && cmd[start] == '&' {
			start++
		}

		right := strings.TrimLeft(cmd[start:], " \t")
		if rePipeShell.MatchString(right) {
			return true
		}
	}
	return false
}
