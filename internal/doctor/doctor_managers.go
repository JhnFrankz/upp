package doctor

import (
	"fmt"
	"strings"

	"github.com/JhnFrankz/upp/internal/platform"
)

type pmCheck struct {
	name    string
	binName string
}

var pmCandidates = []pmCheck{
	{"brew", "brew"},
	{"apt", "apt"},
	{"pacman", "pacman"},
	{"dnf", "dnf"},
	{"zypper", "zypper"},
	{"winget", "winget"},
	{"scoop", "scoop"},
	{"choco", "choco"},
	{"nix", "nix"},
}

func checkPackageManagers(results []CheckResult, deps DoctorDeps) []CheckResult {
	found := make([]string, 0, 4)
	for _, pm := range pmCandidates {
		_, err := deps.LookPath(pm.binName)
		if err == nil {
			found = append(found, pm.name)
		} else if pm.name == "apt" {
			if _, gerr := deps.LookPath("apt-get"); gerr == nil {
				found = append(found, pm.name)
			}
		}
	}

	if len(found) == 0 {
		return append(results, CheckResult{
			Category: "Package Managers",
			Name:     "Package Managers",
			Status:   SeverityWarn,
			Message:  "No package manager found in PATH",
			FixHint:  "Install a supported package manager for your platform (e.g. brew, apt, pacman, winget, scoop).",
		})
	}

	if deps.Platform.OS == platform.OSLinux {
		hasBrew := false
		hasSystem := false
		for _, m := range found {
			if m == "brew" {
				hasBrew = true
			}
			if m == "apt" || m == "pacman" || m == "dnf" || m == "zypper" {
				hasSystem = true
			}
		}
		if hasBrew && hasSystem {
			return append(results, CheckResult{
				Category: "Package Managers",
				Name:     "Package Managers",
				Status:   SeverityWarn,
				Message:  fmt.Sprintf("Multiple package managers detected (%s) on Linux", strings.Join(found, ", ")),
				Detail:   "Both Homebrew (brew) and a system package manager are installed. Tools may be duplicated or managed inconsistently.",
				FixHint:  "Ensure tools are managed consistently to avoid duplicate installations or PATH conflicts.",
			})
		}
	}

	return append(results, CheckResult{
		Category: "Package Managers",
		Name:     "Package Managers",
		Status:   SeverityOK,
		Message:  fmt.Sprintf("Detected package manager(s): %s", strings.Join(found, ", ")),
	})
}
