// Package version implements semver and git-describe version parsing
// and comparison.
package version

import (
	"fmt"
	"strconv"
	"strings"
)

// Version is a parsed upp version string in one of the shapes produced
// by `git describe --tags --always --dirty`: vX.Y.Z, vX.Y.Z-N-gHASH,
// vX.Y.Z-N-gHASH-dirty — or the literal "dev" fallback.
//
// Tag holds the 3-part numeric tag prefix; Dirty marks a build with
// uncommitted changes; Dev marks an untagged development build.
// Dev and Dirty builds are treated as development builds by the update
// flow (no update claim, no network), but still compare by tag prefix.
type Version struct {
	Tag   [3]int
	Dirty bool
	Dev   bool
}

// Parse parses a version string into a Version. Any shape outside the
// git-describe grammar above fails closed with an error.
func Parse(s string) (Version, error) {
	if s == "" {
		return Version{}, fmt.Errorf("version: empty version string")
	}
	if s == "dev" {
		return Version{Dev: true}, nil
	}
	if !strings.HasPrefix(s, "v") {
		return Version{}, fmt.Errorf("version: %q: missing \"v\" prefix", s)
	}

	body := strings.TrimPrefix(s, "v")
	tagPart, suffix, _ := strings.Cut(body, "-")
	tag, err := parseTag(tagPart)
	if err != nil {
		return Version{}, fmt.Errorf("version: %q: %w", s, err)
	}

	dirty := false
	// The leading "-" before the suffix was consumed by Cut above, so a
	// clean-tag dirty build arrives as suffix "dirty" while an untagged
	// dirty build arrives as "N-gHASH-dirty".
	if suffix == "dirty" {
		dirty = true
		suffix = ""
	} else if rest, ok := strings.CutSuffix(suffix, "-dirty"); ok {
		dirty = true
		suffix = rest
	}
	if suffix != "" {
		if err := parseDescribeSuffix(suffix); err != nil {
			return Version{}, fmt.Errorf("version: %q: %w", s, err)
		}
	}

	return Version{Tag: tag, Dirty: dirty}, nil
}

// parseTag parses the "X.Y.Z" prefix into three non-negative integers.
func parseTag(s string) ([3]int, error) {
	var tag [3]int
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return tag, fmt.Errorf("tag %q must be X.Y.Z", s)
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return tag, fmt.Errorf("tag %q: component %q is not a non-negative integer", s, p)
		}
		tag[i] = n
	}
	return tag, nil
}

// parseDescribeSuffix validates the "N-gHASH" part of an untagged
// git-describe build: a decimal commit count followed by a hex hash.
func parseDescribeSuffix(s string) error {
	i := strings.LastIndex(s, "-g")
	if i <= 0 {
		return fmt.Errorf("expected -N-gHASH describe suffix, got %q", s)
	}
	if _, err := strconv.Atoi(s[:i]); err != nil {
		return fmt.Errorf("expected decimal commit count in %q", s)
	}
	hash := s[i+2:]
	if hash == "" {
		return fmt.Errorf("empty commit hash in %q", s)
	}
	for _, r := range hash {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return fmt.Errorf("commit hash %q is not hexadecimal", hash)
		}
	}
	return nil
}

// Compare returns -1, 0, or +1 comparing v's tag prefix with o's
// numerically. Untagged -N-gHASH builds and dirty builds compare by
// their tag prefix only; "dev" has a zero tag and compares below any
// release. The update flow gates on Dev/Dirty before comparing.
func (v Version) Compare(o Version) int {
	for i := 0; i < 3; i++ {
		switch {
		case v.Tag[i] < o.Tag[i]:
			return -1
		case v.Tag[i] > o.Tag[i]:
			return 1
		}
	}
	return 0
}

// ExtractVersionFromString attempts to extract a version string from command output or a version header.
// It looks for semver-like patterns (v1.2.3, 1.2.3, 1.2.3-rc1, etc.) in each line.
// Returns empty string if no version token is found.
func ExtractVersionFromString(s string) string {
	lines := strings.Split(s, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		version := findVersionInLine(line)
		if version != "" {
			return version
		}
	}
	return ""
}

// ExtractVersionFromOutput extracts a version string from command output,
// falling back to the entire raw output string if no version token is identified.
func ExtractVersionFromOutput(s string) string {
	v := ExtractVersionFromString(s)
	if v != "" {
		return v
	}
	return s
}

// findVersionInLine scans a line for a version-like token.
func findVersionInLine(line string) string {
	fields := strings.Fields(line)
	for _, field := range fields {
		cleaned := strings.Trim(field, "(),:;")
		if IsVersionLike(cleaned) {
			return cleaned
		}
	}
	return ""
}

// IsVersionLike returns true if the string looks like a version number.
func IsVersionLike(s string) bool {
	if s == "" {
		return false
	}
	start := 0
	for start < len(s) && ((s[start] >= 'a' && s[start] <= 'z') || (s[start] >= 'A' && s[start] <= 'Z')) {
		start++
	}
	if start >= len(s) {
		return false
	}
	if s[start] < '0' || s[start] > '9' {
		return false
	}
	dotFound := false
	for i := start; i < len(s); i++ {
		c := s[i]
		if c == '.' {
			dotFound = true
		} else if (c < '0' || c > '9') && c != '.' && c != '-' && c != '+' {
			break
		}
	}
	return dotFound
}
