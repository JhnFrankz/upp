package security

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/JhnFrankz/upp/internal/adapters"
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
	// ConfirmError means --ci mode rejected the tool (medium untrusted or high risk).
	ConfirmError
)

// ConfirmConfig holds the parameters for a confirmation prompt.
type ConfirmConfig struct {
	ToolName   string
	TrustLevel adapters.TrustLevel // typed: official, custom-trusted, custom-untrusted
	RiskLevel  RiskLevel
	Command    string
	Privileges []string
	CI         bool
	Reader     io.Reader // injectable for testing
}

// ConfirmAction determines whether to prompt and returns the decision.
//
// Risk is evaluated before trust: the risk tier decides whether a prompt or
// error is required; trust only refines behavior within a tier and never
// bypasses the risk matrix.
//
// Decision matrix (from design.md D4):
//
//	Official tools: auto-proceed (no prompts)
//	CI: Low → Auto (any trust); Medium → Auto (trusted) / Error (untrusted);
//	    High → Error (trust does not waive it)
//	Interactive: High → prompt (any trust); Medium → info (trusted) / prompt
//	    (untrusted); Low → info
func ConfirmAction(cfg ConfirmConfig) ConfirmDecision {
	// Official tools always auto-proceed.
	if cfg.TrustLevel == adapters.TrustOfficial {
		return ConfirmAuto
	}

	switch cfg.RiskLevel {
	case RiskLow:
		// Low risk never requires confirmation; CI auto-proceeds without info.
		if cfg.CI {
			return ConfirmAuto
		}
		printInfo(cfg)
		return ConfirmProceed

	case RiskMedium:
		// CI: trusted auto-proceeds, untrusted errors.
		if cfg.CI {
			if cfg.TrustLevel == adapters.TrustCustomTrusted {
				return ConfirmAuto
			}
			return ConfirmError
		}
		// Interactive: trusted shows info, untrusted prompts.
		if cfg.TrustLevel == adapters.TrustCustomTrusted {
			printInfo(cfg)
			return ConfirmProceed
		}
		return promptUser(cfg)

	default: // RiskHigh
		// High risk requires confirmation; trust never waives it.
		// Deliberately a default, NOT `case RiskHigh:`: an unknown future
		// RiskLevel (e.g. RiskLevel(99)) would otherwise exit the switch
		// unhandled and return the zero-value ConfirmProceed (fail-open, R4-4).
		// With the default branch, unknown risk values fall into High = fail-closed.
		if cfg.CI {
			return ConfirmError
		}
		return promptUser(cfg)
	}
}

// printInfo shows the command details without asking for confirmation.
func printInfo(cfg ConfirmConfig) {
	fmt.Printf("  %s (%s) — Command: %s — Risk: %s\n",
		cfg.ToolName, cfg.TrustLevel, cfg.Command, cfg.RiskLevel)
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
