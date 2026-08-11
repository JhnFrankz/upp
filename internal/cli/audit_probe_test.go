package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JhnFrankz/upp/internal/config"
)

// runUpdateCmd executes `upp update` with the given args and returns the
// captured stdout plus the error from root.Execute().
func runUpdateCmd(args ...string) (string, error) {
	var runErr error
	out := withCapturedStdout(func() {
		root, gf := BuildRoot()
		AddCommands(root, gf)
		root.SetArgs(append([]string{"update"}, args...))
		runErr = root.Execute()
	})
	return out, runErr
}

// Probe (converted from audit): trusted custom high-risk (sudo) must fail with
// a non-zero exit in --ci mode and never execute — trust does not waive the
// high-risk gate in non-interactive mode (security-model spec).
func TestProbe_TrustedCustomHighRisk_CI(t *testing.T) {
	marker := probeSetup(t, config.CustomTool{
		Command: "sudo rm -rf " + filepath.Join(t.TempDir(), "victim"),
		Trusted: true,
	})
	_, err := runUpdateCmd("--ci", "--only", "evil-tool")
	if err == nil {
		t.Error("SECURITY BYPASS: high-risk trusted custom tool did not fail in --ci mode")
	}
	if _, statErr := os.Stat(marker); statErr == nil {
		t.Error("SECURITY BYPASS: high-risk trusted custom tool EXECUTED in --ci mode")
	}
}

// Probe (converted from audit): trusted custom high-risk (rm -rf via sudo) must
// prompt in interactive mode; with no stdin available the prompt denies and
// nothing executes.
func TestProbe_TrustedCustomHighRisk_Interactive(t *testing.T) {
	marker := probeSetup(t, config.CustomTool{
		Command: "sudo rm -rf " + filepath.Join(t.TempDir(), "victim"),
		Trusted: true,
	})
	_, err := runUpdateCmd("--only", "evil-tool")
	if err != nil {
		t.Fatalf("interactive update should not error on deny: %v", err)
	}
	if _, statErr := os.Stat(marker); statErr == nil {
		t.Error("SECURITY BYPASS: high-risk trusted custom tool EXECUTED interactively without prompt")
	}
}

// Probe (converted from audit): untrusted custom high-risk (rm -rf via sudo)
// must prompt in interactive mode and never execute without confirmation.
func TestProbe_UntrustedCustomHighRisk_Interactive(t *testing.T) {
	marker := probeSetup(t, config.CustomTool{
		Command: "sudo rm -rf " + filepath.Join(t.TempDir(), "victim"),
		Trusted: false,
	})
	_, err := runUpdateCmd("--only", "evil-tool")
	if err != nil {
		t.Fatalf("interactive update should not error on deny: %v", err)
	}
	if _, statErr := os.Stat(marker); statErr == nil {
		t.Error("SECURITY BYPASS: untrusted high-risk custom tool EXECUTED interactively without prompt")
	}
}

// Probe (correct-pass): a trusted low-risk custom tool must execute — the
// marker proves the command actually ran.
func TestProbe_TrustedLowRisk_Executes(t *testing.T) {
	marker := probeSetup(t, config.CustomTool{
		Command: "harmless-tool --version",
		Trusted: true,
	})
	_, err := runUpdateCmd("--only", "evil-tool")
	if err != nil {
		t.Fatalf("low-risk update should not error: %v", err)
	}
	if _, statErr := os.Stat(marker); statErr != nil {
		t.Error("low-risk trusted custom tool should have executed (marker missing)")
	}
}

// Probe: --quiet must NOT suppress the confirmation prompt for a medium-risk
// untrusted custom tool (ux-patterns: quiet affects detail, not prompts).
// The prompt is shown and, with no stdin available, denies execution.
func TestProbe_QuietMediumRisk_StillPrompts(t *testing.T) {
	marker := probeSetup(t, config.CustomTool{
		Command: "evil-tool --update && echo done",
		Trusted: false,
	})
	out, err := runUpdateCmd("--quiet", "--only", "evil-tool")
	if err != nil {
		t.Fatalf("interactive update should not error on deny: %v", err)
	}
	if !strings.Contains(out, "Proceed? [y/N]") {
		t.Errorf("--quiet suppressed the confirmation prompt; output: %q", out)
	}
	if _, statErr := os.Stat(marker); statErr == nil {
		t.Error("SECURITY BYPASS: medium-risk untrusted tool EXECUTED under --quiet without confirmation")
	}
}
