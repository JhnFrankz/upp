// Package security implements trust levels, risk classification, and
// confirmation prompts for tool execution safety.
package security

import (
	"strings"
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

// HighRiskKeywords are substrings that immediately classify a command as high risk.
var HighRiskKeywords = []string{
	"sudo",
	"doas",
	"pkexec",
	"runas",
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

// hasCommandChaining detects command chaining operators.
func hasCommandChaining(cmd string) bool {
	return strings.Contains(cmd, "&&") ||
		strings.Contains(cmd, "||") ||
		strings.Contains(cmd, ";")
}

var pipeInterpreters = []string{
	"sh", "bash", "zsh", "fish",
	"powershell", "pwsh",
	"python", "python3", "node", "ruby", "perl",
}

// hasPipeToShell detects piping output to a shell or script interpreter,
// both spaced ("| sh", "| python") and compact ("|sh", "|python") variants.
func hasPipeToShell(cmd string) bool {
	lower := strings.ToLower(strings.TrimSpace(cmd))
	for _, interp := range pipeInterpreters {
		if strings.Contains(lower, "| "+interp) || strings.Contains(lower, "|"+interp) {
			return true
		}
	}
	return false
}
