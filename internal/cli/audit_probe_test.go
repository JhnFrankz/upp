package cli

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/JhnFrankz/upp/internal/adapters"
)

// The probes exercise the security-classification path through runUpdate with
// an injected fake adapter (design D4): the fake's Command string drives
// ClassifyCommand (sudo → High, "&&" → Medium) and its Privileges flow into
// ConfirmConfig (update.go), while the updated flag proves whether Update()
// ever executed. ConfirmAction is string-driven, so the CI error / interactive
// deny / trusted proceed branches fire exactly as with real subprocesses;
// os.Stdin is /dev/null under `go test`, so interactive prompts deny, same as
// before. Real-subprocess proof of the security branches lives in
// internal/adapters tests and scripts/smoke-test.sh.

// TestProbe_TrustedCustomHighRisk_CI: trusted custom high-risk (sudo) must
// fail with a non-zero exit in --ci mode and never execute — trust does not
// waive the high-risk gate in non-interactive mode (security-model spec).
func TestProbe_TrustedCustomHighRisk_CI(t *testing.T) {
	fake := &fakeUpdateAdapter{
		name:       "evil-tool",
		policy:     adapters.PolicyAlwaysUpdate,
		trust:      adapters.TrustCustomTrusted,
		command:    "sudo rm -rf " + filepath.Join(t.TempDir(), "victim"),
		privileges: []string{"sudo"},
	}
	_, err := runUpdateWithFlags(t, fake, &GlobalFlags{CI: true}, &UpdateFlags{})
	if err == nil {
		t.Error("SECURITY BYPASS: high-risk trusted custom tool did not fail in --ci mode")
	}
	if fake.updated {
		t.Error("SECURITY BYPASS: high-risk trusted custom tool EXECUTED in --ci mode")
	}
}

// TestProbe_TrustedCustomHighRisk_Interactive: trusted custom high-risk
// (rm -rf via sudo) must prompt in interactive mode; with no stdin available
// the prompt denies and nothing executes.
func TestProbe_TrustedCustomHighRisk_Interactive(t *testing.T) {
	fake := &fakeUpdateAdapter{
		name:       "evil-tool",
		policy:     adapters.PolicyAlwaysUpdate,
		trust:      adapters.TrustCustomTrusted,
		command:    "sudo rm -rf " + filepath.Join(t.TempDir(), "victim"),
		privileges: []string{"sudo"},
	}
	_, err := runUpdateWithFlags(t, fake, &GlobalFlags{}, &UpdateFlags{})
	if err != nil {
		t.Fatalf("interactive update should not error on deny: %v", err)
	}
	if fake.updated {
		t.Error("SECURITY BYPASS: high-risk trusted custom tool EXECUTED interactively without prompt")
	}
}

// TestProbe_UntrustedCustomHighRisk_Interactive: untrusted custom high-risk
// (rm -rf via sudo) must prompt in interactive mode and never execute without
// confirmation.
func TestProbe_UntrustedCustomHighRisk_Interactive(t *testing.T) {
	fake := &fakeUpdateAdapter{
		name:       "evil-tool",
		policy:     adapters.PolicyAlwaysUpdate,
		trust:      adapters.TrustCustomUntrusted,
		command:    "sudo rm -rf " + filepath.Join(t.TempDir(), "victim"),
		privileges: []string{"sudo"},
	}
	_, err := runUpdateWithFlags(t, fake, &GlobalFlags{}, &UpdateFlags{})
	if err != nil {
		t.Fatalf("interactive update should not error on deny: %v", err)
	}
	if fake.updated {
		t.Error("SECURITY BYPASS: untrusted high-risk custom tool EXECUTED interactively without prompt")
	}
}

// TestProbe_TrustedLowRisk_Executes: a trusted low-risk custom tool must
// execute — the updated flag proves Update() actually ran.
func TestProbe_TrustedLowRisk_Executes(t *testing.T) {
	fake := &fakeUpdateAdapter{
		name:    "evil-tool",
		policy:  adapters.PolicyAlwaysUpdate,
		trust:   adapters.TrustCustomTrusted,
		command: "harmless-tool --version",
		result:  adapters.Result{Success: true, Before: "1.0.0", After: "1.0.0"},
	}
	_, err := runUpdateWithFlags(t, fake, &GlobalFlags{}, &UpdateFlags{})
	if err != nil {
		t.Fatalf("low-risk update should not error: %v", err)
	}
	if !fake.updated {
		t.Error("low-risk trusted custom tool should have executed (Update never called)")
	}
}

// TestProbe_QuietMediumRisk_StillPrompts: --quiet must NOT suppress the
// confirmation prompt for a medium-risk untrusted custom tool (ux-patterns:
// quiet affects detail, not prompts). The prompt is shown and, with no stdin
// available, denies execution.
func TestProbe_QuietMediumRisk_StillPrompts(t *testing.T) {
	fake := &fakeUpdateAdapter{
		name:    "evil-tool",
		policy:  adapters.PolicyAlwaysUpdate,
		trust:   adapters.TrustCustomUntrusted,
		command: "evil-tool --update && echo done",
	}
	out, err := runUpdateWithFlags(t, fake, &GlobalFlags{Quiet: true}, &UpdateFlags{})
	if err != nil {
		t.Fatalf("interactive update should not error on deny: %v", err)
	}
	if !strings.Contains(out, "Proceed? [y/N]") {
		t.Errorf("--quiet suppressed the confirmation prompt; output: %q", out)
	}
	if fake.updated {
		t.Error("SECURITY BYPASS: medium-risk untrusted tool EXECUTED under --quiet without confirmation")
	}
}
