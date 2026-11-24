package adapters

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"slices"

	"golang.org/x/term"
)

var reSudoWord = regexp.MustCompile(`\bsudo\b`)

var (
	stdinIsTerminalFn = func() bool {
		return term.IsTerminal(int(os.Stdin.Fd()))
	}
	sudoInteractiveAuthFn = func(ctx context.Context) error {
		cmd := exec.CommandContext(ctx, "sudo", "-v")
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
)

// SetSudoTestSeams swaps TTY detection and interactive sudo auth for tests and
// returns a restore function.
func SetSudoTestSeams(isTTY func() bool, auth func(context.Context) error) func() {
	origTTY := stdinIsTerminalFn
	origAuth := sudoInteractiveAuthFn
	if isTTY != nil {
		stdinIsTerminalFn = isTTY
	}
	if auth != nil {
		sudoInteractiveAuthFn = auth
	}
	return func() {
		stdinIsTerminalFn = origTTY
		sudoInteractiveAuthFn = origAuth
	}
}

// CommandUsesSudo reports whether a shell command invokes sudo as a word token.
func CommandUsesSudo(command string) bool {
	return reSudoWord.MatchString(command)
}

// EnsureSudoAuth pre-authenticates sudo credentials on the foreground TTY
// before spawning a Setpgid background-pgrp child, preventing SIGTTIN hangs.
func EnsureSudoAuth(ctx context.Context) error {
	return EnsureSudoAuthForOS(ctx, runtimeGOOSFn())
}

// EnsureSudoAuthForOS behaves like EnsureSudoAuth for the specified OS name.
func EnsureSudoAuthForOS(ctx context.Context, goos string) error {
	if goos == "windows" || !stdinIsTerminalFn() {
		return nil
	}
	if err := sudoInteractiveAuthFn(ctx); err != nil {
		return fmt.Errorf("sudo authentication failed: %w", err)
	}
	return nil
}

// PrepareSudoArgs returns args with "-n" injected right after "sudo" on Unix
// unless non-interactive flags are already present, so background-pgrp children
// fail fast instead of attempting to read from /dev/tty.
func PrepareSudoArgs(name string, args []string) []string {
	return PrepareSudoArgsForOS(runtimeGOOSFn(), name, args)
}

// PrepareSudoArgsForOS behaves like PrepareSudoArgs for the specified OS name.
func PrepareSudoArgsForOS(goos, name string, args []string) []string {
	if goos == "windows" || name != "sudo" {
		return args
	}
	for _, a := range args {
		if a == "--" {
			break
		}
		if a == "-n" || a == "--non-interactive" {
			return args
		}
	}
	out := make([]string, 0, len(args)+1)
	out = append(out, "-n")
	out = append(out, args...)
	return slices.Clip(out)
}
