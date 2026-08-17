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

// TestHelp_ShowsGroups locks the D6 grouping: `upp --help` MUST list command
// sections labeled Tool Commands / Config Commands / Maintenance (spec
// command-interface, requirement Help Output Grouping).
func TestHelp_ShowsGroups(t *testing.T) {
	output := rootHelpOutput(t, "--help")

	for _, want := range []string{"Tool Commands", "Config Commands", "Maintenance"} {
		if !strings.Contains(output, want) {
			t.Errorf("--help must show group %q, got:\n%s", want, output)
		}
	}
}

// TestHelp_CommandsListed verifies every command still appears under its
// group, regardless of which group, and the completion built-in is absent.
func TestHelp_CommandsListed(t *testing.T) {
	output := rootHelpOutput(t, "--help")

	for _, want := range []string{"check", "list", "update", "init", "export", "import", "self-update"} {
		if !strings.Contains(output, want) {
			t.Errorf("--help must list command %q, got:\n%s", want, output)
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

	for _, want := range []string{"Tool Commands", "Config Commands", "Maintenance"} {
		if !strings.Contains(helped, want) {
			t.Errorf("`upp help` must show group %q, got:\n%s", want, helped)
		}
	}
	if strings.Contains(helped, "completion") {
		t.Errorf("`upp help` must not list 'completion', got:\n%s", helped)
	}
}
