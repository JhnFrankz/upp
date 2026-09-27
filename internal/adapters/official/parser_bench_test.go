package official

import (
	"testing"
)

func BenchmarkParseAptPolicyOutput(b *testing.B) {
	fixture := `Installed: 2.34.1-1ubuntu1
Candidate: 2.35.0-1ubuntu1
Version table:
 *** 2.34.1-1ubuntu1 500
        500 http://archive.ubuntu.com/ubuntu jammy/main amd64 Packages
        100 /var/lib/dpkg/status
`
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parseAptPolicyOutput(fixture)
	}
}

func BenchmarkParseBrewOutdatedJSON(b *testing.B) {
	fixture := `[{"name":"gh","installed_versions":["2.40.0"],"current_version":"2.45.0"},{"name":"docker","installed_versions":["24.0.5"],"current_version":"25.0.0"}]`
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = parseBrewOutdatedJSON(fixture)
	}
}

func BenchmarkParsePacmanQOutput(b *testing.B) {
	fixture := "github-cli 2.45.0-1\n"
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = parsePacmanQOutput(fixture)
	}
}

func BenchmarkParsePacmanSiOutput(b *testing.B) {
	fixture := `Repository      : extra
Name            : github-cli
Version         : 2.45.0-1
Description     : The GitHub CLI tool
Architecture    : x86_64
URL             : https://cli.github.com/
Licenses        : MIT
Groups          : None
Provides        : None
Depends On      : glibc
Optional Deps   : None
Conflicts With  : None
Replaces        : None
Download Size   : 9.50 MiB
Installed Size  : 32.50 MiB
Packager        : Arch Linux
Build Date      : Mon 01 Jan 2026 12:00:00 PM UTC
Validated By    : SHA-256 Sum
`
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = parsePacmanSiOutput(fixture)
	}
}

func BenchmarkParseWingetUpgradeOutput(b *testing.B) {
	fixture := `Name                          Id                             Version        Available      Source
-------------------------------------------------------------------------------------------------
App Installer                 Microsoft.AppInstaller         1.22.11261.0   1.24.1101.0    winget
GitHub CLI                    GitHub.cli                     2.40.0         2.45.0         winget
Docker Desktop                Docker.DockerDesktop           4.28.0         4.29.0         winget
`
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = parseWingetUpgradeOutput(fixture)
	}
}

func BenchmarkParseWingetPackageUpgradeOutput(b *testing.B) {
	fixture := `Name                          Id                             Version        Available      Source
-------------------------------------------------------------------------------------------------
App Installer                 Microsoft.AppInstaller         1.22.11261.0   1.24.1101.0    winget
GitHub CLI                    GitHub.cli                     2.40.0         2.45.0         winget
Docker Desktop                Docker.DockerDesktop           4.28.0         4.29.0         winget
`
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = parseWingetPackageUpgradeOutput(fixture, "GitHub.cli")
	}
}

func BenchmarkParseScoopStatusOutput(b *testing.B) {
	fixture := `WARN Scoop is out of date.
Name       Installed  Latest
----       ---------  ------
scoop      1.0.0      1.2.0
git        2.43.0     2.44.0
`
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = parseScoopStatusOutput(fixture)
	}
}

func BenchmarkParseVersionFromHeader(b *testing.B) {
	fixture := "ToolName v3.2.1 (c) 2026\nSecond line\nThird line\n"
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = parseVersionFromHeader(fixture)
	}
}
