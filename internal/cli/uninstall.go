package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/JhnFrankz/upp/internal/config"
	"github.com/JhnFrankz/upp/internal/output"
	"github.com/JhnFrankz/upp/internal/uninstall"
)

// UninstallFlags holds flags specific to the uninstall command.
type UninstallFlags struct {
	DryRun bool
}

// uninstallDeps carries the injectable seams for runUninstall.
type uninstallDeps struct {
	execPath  func() (string, error)
	configDir func() (string, error)
	cacheDir  func() (string, error)
	discover  func(execPath, configDir, cacheDir string) ([]uninstall.Target, error)
	remove    func(string) error
	removeAll func(string) error
}

// defaultCacheDir resolves the platform-appropriate cache directory for upp.
func defaultCacheDir() (string, error) {
	cacheRoot, err := os.UserCacheDir()
	if err != nil {
		home, herr := os.UserHomeDir()
		if herr != nil {
			return "", fmt.Errorf("cannot determine cache directory: %w", err)
		}
		return filepath.Join(home, ".cache", "upp"), nil
	}
	return filepath.Join(cacheRoot, "upp"), nil
}

// NewUninstallCommand creates the `upp uninstall` command.
func NewUninstallCommand(gf *GlobalFlags) *cobra.Command {
	flags := UninstallFlags{}

	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall upp and remove all binaries, configuration, and caches",
		Long: "Completely uninstall upp by removing the active binary, historical backups, " +
			"configuration directories (~/.config/upp), and cache directories (~/.cache/upp).\n" +
			"Zero-Sudo policy: upp never escalates privileges. If an unwritable path is encountered, " +
			"it performs best-effort deletion, emits remediation instructions, and exits with code 1.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUninstall(gf, flags, os.Stdout, cliDeps.uninstall)
		},
	}

	cmd.Flags().BoolVar(&flags.DryRun, "dry-run", false, "preview targets to be removed without deleting anything")

	return cmd
}

// runUninstall executes the uninstallation process or simulates it under --dry-run.
func runUninstall(gf *GlobalFlags, flags UninstallFlags, w io.Writer, deps uninstallDeps) error {
	if deps.execPath == nil {
		deps.execPath = os.Executable
	}
	if deps.configDir == nil {
		deps.configDir = config.ConfigDir
	}
	if deps.cacheDir == nil {
		deps.cacheDir = defaultCacheDir
	}
	if deps.discover == nil {
		deps.discover = uninstall.DiscoverTargets
	}
	if deps.remove == nil {
		deps.remove = os.Remove
	}
	if deps.removeAll == nil {
		deps.removeAll = os.RemoveAll
	}

	execPath, err := deps.execPath()
	if err != nil {
		return fmt.Errorf("cannot locate upp binary: %w", err)
	}

	cfgDir, _ := deps.configDir()
	cDir, _ := deps.cacheDir()

	targets, err := deps.discover(execPath, cfgDir, cDir)
	if err != nil {
		return fmt.Errorf("cannot discover uninstall targets: %w", err)
	}

	r := output.NewRenderer(w, gf.Quiet)

	if flags.DryRun {
		r.UninstallDryRunHeader()
		for _, t := range targets {
			r.UninstallDryRunTarget(string(t.Type), t.Path)
		}
		return nil
	}

	errs := uninstall.Execute(targets, deps.remove, deps.removeAll)

	// Report successful deletions
	failedMap := make(map[string]bool)
	for _, e := range errs {
		failedMap[e.Target.Path] = true
	}
	for _, t := range targets {
		if t.Exists && !failedMap[t.Path] {
			r.UninstallRemoved(string(t.Type), t.Path)
		}
	}

	if len(errs) > 0 {
		for _, e := range errs {
			r.Warning(fmt.Sprintf("could not remove %s (%s): %v. Run manual command: sudo rm -rf %s",
				e.Target.Type, e.Target.Path, e.Err, e.Target.Path))
		}
		return errors.New("uninstall completed with warnings; some files could not be removed due to permissions")
	}

	r.UninstallDone()
	return nil
}
