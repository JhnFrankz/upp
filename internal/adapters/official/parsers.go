package official

import (
	"encoding/json"
	"strings"
	"unicode/utf8"
)

// wingetSelfID is the manifest Id that identifies Windows Package Manager
// itself in `winget upgrade` (no args) output. winget's self-update package
// is the App Installer, whose Id is "Microsoft.AppInstaller"; the "winget"
// display label and the "winget" Source column are not unique enough to anchor
// on (the display label is field 0 and the Source column shares the same
// string), so the manifest Id is the unambiguous key.
const wingetSelfID = "Microsoft.AppInstaller"

// brewOutdatedEntry is one element of `brew outdated --json <pkg>` output.
type brewOutdatedEntry struct {
	Name              string   `json:"name"`
	InstalledVersions []string `json:"installed_versions"`
	CurrentVersion    string   `json:"current_version"`
}

type brewOutdatedV2 struct {
	Formulae []brewOutdatedEntry `json:"formulae"`
	Casks    []brewOutdatedEntry `json:"casks"`
}

// parseBrewOutdatedJSON is a PURE, fail-closed function that parses the JSON
// array produced by `brew outdated --json <pkg>`. brew emits an empty array
// `[]` when the formula is current (or not installed), so found=false and both
// versions empty means "no update". When the formula IS outdated, the single
// element carries the oldest installed version and the newest current_version.
// Both v1 JSON arrays and v2 JSON objects (with formulae or casks) are supported.
// Malformed JSON yields the empty fail-closed tuple (no error), matching the
// fail-closed helper convention — the caller distinguishes that from a real
// subprocess failure via commandOutputErr.
func parseBrewOutdatedJSON(out string) (current, latest string, found bool) {
	trimmed := strings.TrimSpace(out)
	if trimmed == "[]" || trimmed == "" {
		return "", "", false
	}
	var entries []brewOutdatedEntry
	if err := json.Unmarshal([]byte(trimmed), &entries); err != nil {
		var v2 brewOutdatedV2
		if errV2 := json.Unmarshal([]byte(trimmed), &v2); errV2 == nil {
			if len(v2.Formulae) > 0 {
				entries = v2.Formulae
			} else if len(v2.Casks) > 0 {
				entries = v2.Casks
			}
		}
		if len(entries) == 0 {
			return "", "", false
		}
	}
	if len(entries) == 0 {
		return "", "", false
	}
	e := entries[0]
	current = ""
	if len(e.InstalledVersions) > 0 {
		current = e.InstalledVersions[0]
	}
	found = current != e.CurrentVersion && e.CurrentVersion != ""
	return current, e.CurrentVersion, found
}

// nextField extracts the next whitespace-delimited token and returns the token and remaining string.
// It performs no heap allocations.
func nextField(s string) (field, rest string) {
	i := 0
	for i < len(s) && (s[i] == ' ' || s[i] == '\t' || s[i] == '\r' || s[i] == '\n') {
		i++
	}
	s = s[i:]
	if len(s) == 0 {
		return "", ""
	}
	j := 0
	for j < len(s) && s[j] != ' ' && s[j] != '\t' && s[j] != '\r' && s[j] != '\n' {
		j++
	}
	return s[:j], s[j:]
}

// parseWingetPackageUpgradeOutput scans the no-argument `winget upgrade`
// listing (the read-only form CheckPackage runs) for the row whose manifest Id
// matches pkgID and returns (current, latest, found). It
// is a PURE, fail-closed generalization of parseWingetUpgradeOutput: instead
// of anchoring on winget's self Id, it anchors on the supplied package Id, so
// CheckPackage can find any owned package's row (e.g. gh, GoLang.Go). An
// absent or unparseable row yields found=false (current, no update).
func parseWingetPackageUpgradeOutput(out, pkgID string) (current, latest string, found bool) {
	for len(out) > 0 {
		var line string
		line, out, _ = strings.Cut(out, "\n")
		rest := line
		for {
			var field string
			field, rest = nextField(rest)
			if field == "" {
				break
			}
			if strings.EqualFold(field, pkgID) ||
				(strings.EqualFold(pkgID, "gh") && strings.EqualFold(field, "GitHub.cli")) ||
				(strings.EqualFold(pkgID, "GitHub.cli") && strings.EqualFold(field, "gh")) {
				c, restAfterCurrent := nextField(rest)
				l, _ := nextField(restAfterCurrent)
				if c != "" && l != "" && isVersionLike(c) && isVersionLike(l) {
					return c, l, true
				}
				continue
			}
		}
	}
	return "", "", false
}

// parseWingetUpgradeOutput scans `winget upgrade` (no args) output for the
// winget self row and returns (current, latest, found). It is a PURE,
// fail-closed function: unparseable output or a missing self row yields
// found=false and no error, so an old winget (< 1.6, which lists no self row)
// reports "availability unavailable gracefully". The leading-v on a version
// (e.g. "v1.8.2311") is tolerated — the string is returned unchanged, since
// the winget versions genuinely carry the leading v.
func parseWingetUpgradeOutput(out string) (current, latest string, found bool) {
	return parseWingetPackageUpgradeOutput(out, wingetSelfID)
}

// parseScoopStatusOutput scans `scoop status` output for the scoop self row
// and returns (current, latest, found). It is a PURE, fail-closed function:
// an absent or unparseable scoop row yields found=false and no error, so when
// the `scoop status` output shape is unstable the caller falls back to
// current-only. The leading-v on a version (e.g. "v0.22.0") is tolerated —
// the string is returned unchanged, since scoop versions genuinely carry the
// leading v (winget parity). A leading WARN banner line (scoop prints
// "Scoop is out of date" to stderr, but a hermetic capture may concatenate it
// into stdout) is tolerated: the row is recognized ONLY when the "scoop"
// token is followed by two version-like fields, so a banner like
// "WARN Scoop is out of date." is skipped and the real table row is found.
// parseScoopPackageStatusOutput scans `scoop status` output for a package row
// and returns (current, latest, found). It is a PURE, fail-closed function:
// an absent or unparseable row yields found=false and no error.
func parseScoopPackageStatusOutput(out, pkg string) (current, latest string, found bool) {
	for len(out) > 0 {
		var line string
		line, out, _ = strings.Cut(out, "\n")
		rest := line
		for {
			var field string
			field, rest = nextField(rest)
			if field == "" {
				break
			}
			if !strings.EqualFold(field, pkg) {
				continue
			}
			c, restAfterCurrent := nextField(rest)
			l, _ := nextField(restAfterCurrent)
			if c == "" || l == "" {
				continue
			}
			if !isVersionLike(c) || !isVersionLike(l) {
				continue
			}
			return c, l, true
		}
	}
	return "", "", false
}

func parseScoopStatusOutput(out string) (current, latest string, found bool) {
	return parseScoopPackageStatusOutput(out, "scoop")
}

// truncate shortens a string to maxLen characters, appending "..." if truncated.
func truncate(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	if !utf8.ValidString(s) {
		s = strings.ToValidUTF8(s, "")
	}
	if len(s) <= maxLen {
		return s
	}
	// Rewind to the nearest valid UTF-8 rune start boundary to prevent splitting runes
	for maxLen > 0 && !utf8.RuneStart(s[maxLen]) {
		maxLen--
	}
	return s[:maxLen] + "..."
}
