package security

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// ConfirmDecision represents the outcome of a confirmation prompt.
type ConfirmDecision int

const (
	// ConfirmProceed means the user approved the action.
	ConfirmProceed ConfirmDecision = iota
	// ConfirmDeny means the user denied the action.
	ConfirmDeny
	// ConfirmAuto means --ci mode auto-proceeded (no prompt shown).
	ConfirmAuto
	// ConfirmError means --ci mode rejected (medium+ risk, untrusted).
	ConfirmError
)

// ConfirmConfig holds the parameters for a confirmation prompt.
type ConfirmConfig struct {
	ToolName   string
	TrustLevel string // "official" or "custom"
	RiskLevel  RiskLevel
	Command    string
	Privileges []string
	CI         bool
	Trusted    bool // config trust override for custom tools
	Reader     io.Reader // injectable for testing
}

// ConfirmAction determines whether to prompt and returns the decision.
//
// Decision matrix (from design.md):
//
//	Official tools: auto-proceed (no prompts)
//	Custom untrusted, CI: error and exit
//	Custom trusted, CI: auto-proceed if risk < High, error if risk = High
//	Custom untrusted, interactive: prompt if risk >= Medium
//	Custom trusted, interactive: prompt if risk = High, show info otherwise
func ConfirmAction(cfg ConfirmConfig) ConfirmDecision {
	// Official tools always auto-proceed.
	if cfg.TrustLevel == "official" {
		return ConfirmAuto
	}

	// Custom tools — CI mode.
	if cfg.CI {
		if !cfg.Trusted {
			return ConfirmError
		}
		// Trusted custom tool in CI: auto-proceed if risk < High.
		if cfg.RiskLevel < RiskHigh {
			return ConfirmAuto
		}
		return ConfirmError
	}

	// Custom tools — interactive mode.
	// High risk always requires confirmation regardless of trust.
	if cfg.RiskLevel == RiskHigh {
		return promptUser(cfg)
	}

	// Trusted custom tool with medium risk: show info, no prompt.
	if cfg.Trusted && cfg.RiskLevel == RiskMedium {
		fmt.Printf("  %s (%s) — Command: %s — Risk: %s\n",
			cfg.ToolName, cfg.TrustLevel, cfg.Command, cfg.RiskLevel)
		return ConfirmProceed
	}

	// Untrusted custom tool with medium risk: prompt.
	if !cfg.Trusted && cfg.RiskLevel == RiskMedium {
		return promptUser(cfg)
	}

	// Low risk: show info, proceed.
	fmt.Printf("  %s (%s) — Command: %s — Risk: %s\n",
		cfg.ToolName, cfg.TrustLevel, cfg.Command, cfg.RiskLevel)
	return ConfirmProceed
}

// promptUser displays the confirmation prompt and reads the user's response.
func promptUser(cfg ConfirmConfig) ConfirmDecision {
	privs := "none"
	if len(cfg.Privileges) > 0 {
		privs = strings.Join(cfg.Privileges, ", ")
	}

	fmt.Printf("\n  %s (%s)\n", cfg.ToolName, cfg.TrustLevel)
	fmt.Printf("  Command: %s\n", cfg.Command)
	fmt.Printf("  Privileges: %s\n", privs)
	fmt.Printf("  Risk: %s\n", cfg.RiskLevel)
	fmt.Printf("  Proceed? [y/N] ")

	reader := cfg.Reader
	if reader == nil {
		reader = os.Stdin
	}

	scanner := bufio.NewScanner(reader)
	if scanner.Scan() {
		input := strings.TrimSpace(strings.ToLower(scanner.Text()))
		if input == "y" || input == "yes" {
			return ConfirmProceed
		}
	}

	return ConfirmDeny
}
