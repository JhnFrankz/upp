package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/JhnFrankz/upp/internal/output"
	"github.com/JhnFrankz/upp/internal/platform"
	"github.com/JhnFrankz/upp/internal/selfupdate"
)

// Production endpoint bases for self-update (U5 BaseURL resolution of
// the U3 gap): the API base serves the latest-release lookup, while
// release assets must come from a github.com web base — api.github.com
// answers 404 on /releases/download (verified 2026-08-12) and
// github.com redirects over HTTPS to the asset host, which the client's
// HTTPS-only redirect policy allows.
const (
	selfUpdateAPIBase = "https://api.github.com"
	selfUpdateWebBase = "https://github.com"
)

// NewSelfUpdateCommand creates the `upp self-update` command. It
// accepts no local flags (spec flag semantics: any unknown flag gets
// the default cobra rejection). Persistent flags: --ci denies the
// update; --only/--skip are ignored (they filter tools for
// update/check, not releases); --quiet never suppresses the confirm
// prompt or the deny message.
func NewSelfUpdateCommand(gf *GlobalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "self-update",
		Short: "Update the upp binary itself",
		Long: "Check for a newer upp release, verify its sha256 checksum, and replace the current binary after confirmation. " +
			"--only and --skip are ignored: they filter tools for update/check, not releases.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSelfUpdate(gf, cmd.Root().Version, cliDeps.selfUpdate)
		},
	}
}

// selfUpdateDeps carries the injectable seams for runSelfUpdate. The
// zero value uses production behavior: os.Stdin (real TTY detection and
// prompt input), platform.Detect, os.Executable, and the production
// GitHub client.
type selfUpdateDeps struct {
	stdin    io.Reader
	isTTY    func() bool
	detect   func() (platform.Platform, error)
	execPath func() (string, error)
	client   *selfupdate.Client
}

// runSelfUpdate implements `upp self-update` (design data flow (a);
// decisions D4/D7/D8): --ci denies unconditionally before anything
// else; development builds exit 0 with no network; Prepare runs
// detect → latest lookup → download → verify → extract; the TTY
// confirmation gate precedes the atomic replace; the user declining
// exits 0 with nothing modified. The CLI layer only orchestrates — all
// pipeline logic lives in internal/selfupdate.
func runSelfUpdate(gf *GlobalFlags, version string, deps selfUpdateDeps) error {
	if deps.isTTY == nil {
		deps.isTTY = stdinIsTTY
	}
	if deps.detect == nil {
		deps.detect = platform.Detect
	}
	if deps.execPath == nil {
		deps.execPath = os.Executable
	}

	// --ci denies always, before any work: never auto-proceed, never
	// hang, no network (spec Confirmation Gate + flag semantics).
	if gf.CI {
		return fmt.Errorf("%s: %w", "self-update denied in --ci mode; run upp self-update interactively to confirm", selfupdate.ErrDeniedCI)
	}

	current, err := selfupdate.Parse(version)
	if err != nil {
		return fmt.Errorf("cannot parse current version %q: %w", version, err)
	}
	// Development builds: informational exit 0, no update claim, no
	// network (spec R1).
	if current.Dev || current.Dirty {
		r := output.NewRenderer(os.Stdout, gf.Quiet)
		r.SelfUpdateDevBuild()
		return nil
	}

	p, err := deps.detect()
	if err != nil {
		return fmt.Errorf("cannot detect platform: %w", err)
	}

	c := deps.client
	if c == nil {
		c = selfupdate.NewClient(selfUpdateAPIBase, "")
		c.DownloadBaseURL = selfUpdateWebBase
	}

	rel, newPath, err := selfupdate.Prepare(c, current, p)
	switch {
	case errors.Is(err, selfupdate.ErrUpToDate):
		// Up to date: latest lookup happened (1 request), no download
		// (spec R1).
		r := output.NewRenderer(os.Stdout, gf.Quiet)
		r.SelfUpdateUpToDate(formatVersion(current))
		return nil
	case errors.Is(err, selfupdate.ErrUnsupportedPlatform):
		// Windows (and any other unmapped platform): clear
		// not-supported-yet refusal, nothing modified (spec R7).
		return fmt.Errorf("%s: %w", "self-update is not supported on this platform yet", err)
	case err != nil:
		// Network failures, checksum mismatches, etc.: propagate with
		// their clear package error (spec R2/R4).
		return err
	}

	// The path shown to the user and replaced: os.Executable resolved
	// through symlinks (Replace re-resolves internally; spec R6).
	execPath, err := deps.execPath()
	if err != nil {
		return fmt.Errorf("cannot locate the upp binary: %w", err)
	}
	target, err := filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("cannot resolve binary path %s: %w", execPath, err)
	}

	// Confirmation gate (design D8): TTY only. Non-TTY stdin never
	// hangs, never auto-proceeds, never silently skips.
	if !deps.isTTY() {
		return fmt.Errorf("%s: %w", "self-update requires an interactive terminal; run upp self-update in a terminal", selfupdate.ErrNotTTY)
	}

	r := output.NewRenderer(os.Stdout, gf.Quiet)
	if !confirmReplace(r, deps.stdin, formatVersion(current), rel.Tag, target) {
		return nil // declined: nothing modified, exit 0 (spec R7)
	}

	if err := selfupdate.Replace(execPath, newPath); err != nil {
		return err
	}
	r.SelfUpdateDone(formatVersion(current), rel.Tag)
	return nil
}

// formatVersion renders a parsed Version as its vX.Y.Z tag for display.
func formatVersion(v selfupdate.Version) string {
	return fmt.Sprintf("v%d.%d.%d", v.Tag[0], v.Tag[1], v.Tag[2])
}

// confirmReplace shows the dedicated self-update prompt (design D8 —
// NOT security.ConfirmAction, which auto-proceeds for official tools)
// and returns true only on an explicit y/yes answer. The prompt shows
// current → latest versions and the resolved binary path, and is never
// suppressed by --quiet.
func confirmReplace(r *output.Renderer, stdin io.Reader, current, latest, target string) bool {
	r.SelfUpdatePrompt(current, latest, target)
	reader := stdin
	if reader == nil {
		reader = os.Stdin
	}
	scanner := bufio.NewScanner(reader)
	if scanner.Scan() {
		switch strings.ToLower(strings.TrimSpace(scanner.Text())) {
		case "y", "yes":
			return true
		}
	}
	return false
}

// stdinIsTTY reports whether stdin is a character device (a terminal).
func stdinIsTTY() bool {
	fi, err := os.Stdin.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}
