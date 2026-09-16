package cli

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JhnFrankz/upp/internal/config"
)

// initProbeMarker is a custom tool name seeded into a pre-existing config. It
// makes a destructive overwrite observable: `upp init` rewrites the config from
// detected tools alone, so a surviving marker proves the file was untouched
// while a replaced one proves the overwrite really happened.
const initProbeMarker = "keepme"

// runInitCmd executes `upp init` with the given args through the real CLI
// (root.Execute), serving stdin to fmt.Scanln prompts via withStdin. An empty
// stdin yields an immediate EOF, which is how a non-TTY invocation behaves.
// Returns the captured stdout plus the error from root.Execute().
func runInitCmd(t *testing.T, stdin string, args ...string) (string, error) {
	t.Helper()

	var out string
	var runErr error
	withStdin(t, stdin, func() {
		out = withCapturedStdout(func() {
			root, gf := BuildRoot()
			AddCommands(root, gf)
			root.SetArgs(append([]string{"init"}, args...))
			runErr = root.Execute()
		})
	})
	return out, runErr
}

// TestInitProbe_ConfigGateMatrix walks the whole `upp init` state machine
// (spec command-interface: `upp init`) as one visible matrix over
// {no config, existing config} x {interactive n, interactive y, interactive
// EOF, --ci}:
//
//   - No config: the wizard runs and creates the config, with or without --ci.
//     First-run state comes from explicit file existence, never applied
//     defaults (config-system).
//   - Existing config, interactive: the overwrite prompt appears and only an
//     explicit y/yes rewrites. Anything else — including EOF on a non-TTY —
//     cancels with exit 0 and leaves the file byte-for-byte unchanged.
//   - Existing config, --ci: DENIED before any work with ErrInitDeniedCI and a
//     non-zero exit. --ci can never answer the prompt, and the doctrine
//     (security.ConfirmAction, self-update Confirmation Gate) is to deny rather
//     than auto-proceed or silently skip.
func TestInitProbe_ConfigGateMatrix(t *testing.T) {
	tests := []struct {
		name string
		// preexisting seeds a config containing initProbeMarker before the run.
		preexisting bool
		// stdin is fed to the prompt; "" yields an immediate EOF (non-TTY).
		stdin string
		// args are passed after `init`.
		args []string
		// wantErrIs asserts the returned error wraps this sentinel.
		wantErrIs error
		// wantErrText asserts the returned error message contains this text.
		wantErrText string
		// wantPrompt asserts the interactive overwrite prompt appeared.
		wantPrompt bool
		// wantWrote asserts the "Config written to" confirmation appeared.
		wantWrote bool
		// wantUntouched asserts the seeded config is byte-for-byte unchanged.
		wantUntouched bool
		// wantMarkerReplaced asserts the seeded tool was overwritten.
		wantMarkerReplaced bool
	}{
		{
			name:      "first run, wizard creates",
			wantWrote: true,
		},
		{
			name:      "first run, --ci creates",
			args:      []string{"--ci"},
			wantWrote: true,
		},
		{
			name:          "existing config, interactive n preserves",
			preexisting:   true,
			stdin:         "n\n",
			wantPrompt:    true,
			wantUntouched: true,
		},
		{
			name:          "existing config, interactive EOF cancels",
			preexisting:   true,
			stdin:         "",
			wantPrompt:    true,
			wantUntouched: true,
		},
		{
			name:               "existing config, interactive y overwrites",
			preexisting:        true,
			stdin:              "y\n",
			wantPrompt:         true,
			wantWrote:          true,
			wantMarkerReplaced: true,
		},
		{
			name:          "existing config, --ci denies",
			preexisting:   true,
			args:          []string{"--ci"},
			wantErrIs:     ErrInitDeniedCI,
			wantErrText:   "--ci",
			wantUntouched: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := probeHome(t)
			cfgPath := filepath.Join(tmpDir, ".config", "upp", "config.toml")

			var before []byte
			if tc.preexisting {
				cfg := config.DefaultConfig()
				cfg.Custom[initProbeMarker] = config.CustomTool{
					Command: initProbeMarker + " --update",
					Trusted: true,
				}
				if err := config.Save(cfg); err != nil {
					t.Fatalf("seed config: %v", err)
				}
				var readErr error
				before, readErr = os.ReadFile(cfgPath)
				if readErr != nil {
					t.Fatalf("read seeded config: %v", readErr)
				}
			}

			out, runErr := runInitCmd(t, tc.stdin, tc.args...)

			if tc.wantErrIs != nil || tc.wantErrText != "" {
				if runErr == nil {
					t.Fatal("expected the command to fail, got a nil error")
				}
				if tc.wantErrIs != nil && !errors.Is(runErr, tc.wantErrIs) {
					t.Errorf("error should wrap %v, got: %v", tc.wantErrIs, runErr)
				}
				if tc.wantErrText != "" && !strings.Contains(runErr.Error(), tc.wantErrText) {
					t.Errorf("error message should contain %q, got: %v", tc.wantErrText, runErr)
				}
			} else if runErr != nil {
				t.Fatalf("unexpected error: %v", runErr)
			}

			if got := strings.Contains(out, "Overwrite with new detection?"); got != tc.wantPrompt {
				t.Errorf("prompt shown = %v, want %v; output: %q", got, tc.wantPrompt, out)
			}
			if got := strings.Contains(out, "Config written to"); got != tc.wantWrote {
				t.Errorf("write reported = %v, want %v; output: %q", got, tc.wantWrote, out)
			}

			after, err := os.ReadFile(cfgPath)
			if err != nil {
				t.Fatalf("config must exist after init: %v", err)
			}
			if tc.wantUntouched && string(before) != string(after) {
				t.Error("config must be byte-for-byte unchanged")
			}
			if tc.wantMarkerReplaced && strings.Contains(string(after), initProbeMarker) {
				t.Errorf("overwrite must replace the seeded %q tool; file still contains it", initProbeMarker)
			}
		})
	}
}
