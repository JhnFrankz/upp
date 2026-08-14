// Package security implements trust levels, risk classification, and
// confirmation prompts for tool execution safety.
package security

import "strings"

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

// HighRiskKeywords are substrings that immediately classify a command as high risk.
var HighRiskKeywords = []string{
	"sudo",
	"rm -rf",
	"rm -r /",
	"curl|sh",
	"curl | sh",
	"curl -fsSL",
	"wget|sh",
	"wget | sh",
	"eval",
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

// ClassifyCommand uses a hybrid approach to determine the risk level of a command.
// It checks keyword matching first, then pattern matching for chaining/piping.
func ClassifyCommand(cmd string) RiskLevel {
	lower := strings.ToLower(cmd)

	// 1. Keyword matching — high risk first (short-circuits).
	for _, kw := range HighRiskKeywords {
		if strings.Contains(lower, kw) {
			return RiskHigh
		}
	}

	// 2. Keyword matching — medium risk.
	for _, kw := range MediumRiskKeywords {
		if strings.Contains(lower, kw) {
			return RiskMedium
		}
	}

	// 3. Pattern matching — command chaining increases risk.
	if hasCommandChaining(cmd) {
		return RiskMedium
	}

	// 4. Pattern matching — pipe to shell is always high risk.
	if hasPipeToShell(cmd) {
		return RiskHigh
	}

	return RiskLow
}

// hasCommandChaining detects command chaining operators.
func hasCommandChaining(cmd string) bool {
	return strings.Contains(cmd, "&&") ||
		strings.Contains(cmd, "||") ||
		strings.Contains(cmd, ";")
}

// hasPipeToShell detects piping output to a shell interpreter,
// both spaced ("| sh", "| bash") and compact ("|sh", "|bash") variants.
func hasPipeToShell(cmd string) bool {
	lower := strings.ToLower(strings.TrimSpace(cmd))
	return strings.Contains(lower, "| sh") ||
		strings.Contains(lower, "| bash") ||
		strings.Contains(lower, "|sh") ||
		strings.Contains(lower, "|bash") ||
		strings.Contains(lower, "|zsh") ||
		strings.Contains(lower, "| zsh")
}
