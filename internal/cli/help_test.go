package cli

import (
	"bytes"
	"strings"
	"testing"
)

// rootHelpOutput runs `upp` with the given args (--help / help) and returns
// the captured output. Cobra writes help through the root's out writer, so
// SetOut keeps this hermetic — no os.Stdout capture needed.
func rootHelpOutput(t *testing.T, args ...string) string {
	t.Helper()
	root, gf := BuildRoot()
	AddCommands(root, gf)
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs(args)
	if err := root.Execute(); err != nil {
		t.Fatalf("execute %v: %v", args, err)
	}
	return buf.String()
}

// TestHelp_ShowsGroups locks the 2-group layout: `upp --help` MUST list command
// sections labeled Commands and Maintenance (spec command-interface, requirement Help Output Grouping).
// Config Commands and Tool Commands must be absent.
func TestHelp_ShowsGroups(t *testing.T) {
	output := rootHelpOutput(t, "--help")

	for _, want := range []string{"Commands", "Maintenance"} {
		if !strings.Contains(output, want) {
			t.Errorf("--help must show group %q, got:\n%s", want, output)
		}
	}

	for _, unwanted := range []string{"Tool Commands", "Config Commands"} {
		if strings.Contains(output, unwanted) {
			t.Errorf("--help must not show legacy group %q, got:\n%s", unwanted, output)
		}
	}
}

// TestHelp_CommandsListed verifies every command still appears under its
// group, regardless of which group, and the completion built-in is absent.
func TestHelp_CommandsListed(t *testing.T) {
	output := rootHelpOutput(t, "--help")

	for _, want := range []string{"check", "list", "update", "init", "self-update"} {
		if !strings.Contains(output, want) {
			t.Errorf("--help must list command %q, got:\n%s", want, output)
		}
	}

	for _, unwanted := range []string{"export", "import"} {
		if strings.Contains(output, unwanted) {
			t.Errorf("--help must not list pruned command %q, got:\n%s", unwanted, output)
		}
	}

	if strings.Contains(output, "completion") {
		t.Errorf("--help must not list the hidden 'completion' command, got:\n%s", output)
	}
}

// TestHelp_EqualsHelpSubcommand locks the spec scenario "upp help" renders
// the same grouped output as `upp --help`.
func TestHelp_EqualsHelpSubcommand(t *testing.T) {
	helped := rootHelpOutput(t, "help")

	for _, want := range []string{"Commands", "Maintenance"} {
		if !strings.Contains(helped, want) {
			t.Errorf("`upp help` must show group %q, got:\n%s", want, helped)
		}
	}
	for _, unwanted := range []string{"Tool Commands", "Config Commands"} {
		if strings.Contains(helped, unwanted) {
			t.Errorf("`upp help` must not show legacy group %q, got:\n%s", unwanted, helped)
		}
	}
	if strings.Contains(helped, "completion") {
		t.Errorf("`upp help` must not list 'completion', got:\n%s", helped)
	}
}
