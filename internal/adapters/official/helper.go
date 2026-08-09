// Package official provides built-in adapter implementations for all supported tools.
package official

import (
	"bytes"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// runCmd executes a shell command and returns stdout, stderr, and any error.
// The command runs via the platform's default shell.
func runCmd(command string) (stdout, stderr string, err error) {
	var stdoutBuf, stderrBuf bytes.Buffer

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", command)
	} else {
		cmd = exec.Command("sh", "-c", command)
	}

	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err = cmd.Run()
	return stdoutBuf.String(), stderrBuf.String(), err
}

// runCmdArgs executes a command with explicit arguments (no shell).
func runCmdArgs(name string, args ...string) (stdout, stderr string, err error) {
	var stdoutBuf, stderrBuf bytes.Buffer

	cmd := exec.Command(name, args...)
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err = cmd.Run()
	return stdoutBuf.String(), stderrBuf.String(), err
}

// lookPath checks if a command exists on PATH.
// Returns true if found, false otherwise.
func lookPath(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// extractVersion attempts to extract a version string from command output.
// It looks for semver-like patterns (v1.2.3, 1.2.3, 1.2.3-rc1, etc.).
func extractVersion(output string) string {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Try to find a version-like string in the line.
		version := findVersionInLine(line)
		if version != "" {
			return version
		}
	}
	return ""
}

// findVersionInLine scans a line for a version-like token.
func findVersionInLine(line string) string {
	fields := strings.Fields(line)
	for _, field := range fields {
		cleaned := strings.Trim(field, "(),:;")
		if isVersionLike(cleaned) {
			return cleaned
		}
	}
	return ""
}

// isVersionLike returns true if the string looks like a version number.
func isVersionLike(s string) bool {
	if s == "" {
		return false
	}
	// Strip leading letters (e.g., "go" in "go1.22.0").
	start := 0
	for start < len(s) && ((s[start] >= 'a' && s[start] <= 'z') || (s[start] >= 'A' && s[start] <= 'Z')) {
		start++
	}
	if start >= len(s) {
		return false
	}
	// Must start with a digit after stripping letters.
	if s[start] < '0' || s[start] > '9' {
		return false
	}
	// Must contain at least one dot.
	dotFound := false
	for i := start; i < len(s); i++ {
		c := s[i]
		if c == '.' {
			dotFound = true
		} else if (c < '0' || c > '9') && c != '.' && c != '-' && c != '+' {
			// Allow digits, dots, hyphens, plus — stop at any letter (pre-release tags)
			break
		}
	}
	return dotFound
}

// commandOutput runs a command and returns its trimmed stdout.
// Used for simple version commands where stderr is irrelevant.
func commandOutput(name string, args ...string) string {
	stdout, _, err := runCmdArgs(name, args...)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(stdout)
}

// shellOutput runs a shell command and returns its trimmed stdout.
func shellOutput(command string) string {
	stdout, _, err := runCmd(command)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(stdout)
}

// formatUpdateCmd returns a human-readable description of the update command.
func formatUpdateCmd(cmd string) string {
	return fmt.Sprintf("exec: %s", cmd)
}
