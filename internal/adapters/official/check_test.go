package official

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/JhnFrankz/upp/internal/adapters"
)

// checkCase is one table row for Check(): fakes drive the exec seam, setup
// prepares the environment (nvm), want is the expected UpdateInfo.
type checkCase struct {
	name    string
	newAdpt func() adapters.Adapter
	fakes   execFakes
	setup   func(t *testing.T)
	want    adapters.UpdateInfo
	wantErr bool
	// wantErrContains, when set with wantErr, requires the error message to
	// contain the substring (structured tool/operation failure, task 3.2).
	wantErrContains string
	// wantDeadline requires errors.Is(err, context.DeadlineExceeded) — the
	// CLI timeoutErr mapping must survive the %w chain (D3).
	wantDeadline bool
	// exitCode, when non-nil, replaces the fakes' seam errors with a REAL
	// *exec.ExitError from an `sh -c exit N` child so exit-code
	// interpretation runs against a genuine ExitError, not a plain fake
	// (task 3.3). Skipped on windows like TestRunCmd_*.
	exitCode *int
}

const (
	aptInstalledCmd = "bash -o pipefail -c 'apt-cache policy apt 2>/dev/null | grep \"Installed:\" | awk \"{print \\$2}\"'"
	aptCandidateCmd = "bash -o pipefail -c 'apt-cache policy apt 2>/dev/null | grep \"Candidate:\" | awk \"{print \\$2}\"'"
	nvmCurrentCmd   = "bash -c 'source \"${NVM_DIR:-$HOME/.nvm}/nvm.sh\" >/dev/null 2>&1 && nvm current'"
	nvmRemoteCmd    = "bash -o pipefail -c 'source \"${NVM_DIR:-$HOME/.nvm}/nvm.sh\" >/dev/null 2>&1 && nvm ls-remote --lts | grep -E \"^[[:space:]]*v[0-9]\" | tail -1 | awk \"{print \\$1}\"'"
)

// exitErrFromChild runs `sh -c "exit N"` for real and returns its error — a
// genuine *exec.ExitError — so exit-code interpretation rows exercise a real
// child result (precedent: TestRunCmd_*, task 3.1).
func exitErrFromChild(t *testing.T, code int) error {
	t.Helper()
	err := exec.Command("sh", "-c", fmt.Sprintf("exit %d", code)).Run()
	if err == nil {
		t.Fatal("sh -c exit N unexpectedly succeeded")
	}
	return err
}

func nvmInstalledSetup(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "nvm.sh"), []byte("#!/bin/sh\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("NVM_DIR", dir)
	t.Setenv("HOME", t.TempDir())
}

func nvmMissingSetup(t *testing.T) {
	t.Helper()
	t.Setenv("NVM_DIR", "")
	t.Setenv("HOME", t.TempDir())
}

// TestCheck covers Check() for all 12 official adapters hermetically.
// Every row drives the exec seam; no real subprocess runs.
func TestCheck(t *testing.T) {
	tests := []checkCase{
		// --- apt (shell output parsing) ---
		{
			name:    "apt/update-available",
			newAdpt: func() adapters.Adapter { return &AptAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"apt": true},
				shell: map[string]fakeResult{
					aptInstalledCmd: {stdout: "2.4.0"},
					aptCandidateCmd: {stdout: "2.4.5"},
				},
			},
			want: adapters.UpdateInfo{CurrentVersion: "2.4.0", LatestVersion: "2.4.5", UpdateAvailable: true},
		},
		{
			name:    "apt/not-installed-version-none",
			newAdpt: func() adapters.Adapter { return &AptAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"apt": true},
				shell: map[string]fakeResult{
					aptInstalledCmd: {stdout: "(none)"},
					aptCandidateCmd: {stdout: "2.4.5"},
				},
			},
			want: adapters.UpdateInfo{CurrentVersion: "unknown", LatestVersion: "2.4.5", UpdateAvailable: false},
		},
		{
			name:    "apt/empty-output",
			newAdpt: func() adapters.Adapter { return &AptAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"apt": true},
				shell: map[string]fakeResult{
					aptInstalledCmd: {stdout: ""},
					aptCandidateCmd: {stdout: ""},
				},
			},
			want: adapters.UpdateInfo{CurrentVersion: "unknown", LatestVersion: "unknown", UpdateAvailable: false},
		},
		{
			name:    "apt/command-fails",
			newAdpt: func() adapters.Adapter { return &AptAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"apt": true},
				shell: map[string]fakeResult{
					aptInstalledCmd: {err: errors.New("apt-cache: command not found")},
					aptCandidateCmd: {err: errors.New("apt-cache: command not found")},
				},
			},
			wantErr:         true,
			wantErrContains: "apt check failed",
		},
		{
			name:    "apt/not-installed-error",
			newAdpt: func() adapters.Adapter { return &AptAdapter{} },
			fakes:   execFakes{lookPath: map[string]bool{"apt": false}},
			wantErr: true,
		},

		// --- brew (version extraction) ---
		// Every check row keys the update command with failIfRun so that if
		// Check() is ever (incorrectly) changed to invoke `brew update`, the
		// mutating command runs inside check and the row fails loudly.
		{
			name:    "brew/normal",
			newAdpt: func() adapters.Adapter { return &BrewAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"brew": true},
				cmdArgs:  map[string]fakeResult{"brew": {stdout: "Homebrew 4.1.0"}},
				shell:    map[string]fakeResult{brewUpdateCmd: failIfRun},
			},
			want: adapters.UpdateInfo{CurrentVersion: "4.1.0", LatestVersion: "4.1.0", UpdateAvailable: false},
		},
		{
			name:    "brew/multiline-output",
			newAdpt: func() adapters.Adapter { return &BrewAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"brew": true},
				cmdArgs:  map[string]fakeResult{"brew": {stdout: "Homebrew 4.1.0\nUpdating Homebrew...\n==> Auto-updated!"}},
				shell:    map[string]fakeResult{brewUpdateCmd: failIfRun},
			},
			want: adapters.UpdateInfo{CurrentVersion: "4.1.0", LatestVersion: "4.1.0", UpdateAvailable: false},
		},
		{
			name:    "brew/empty-output",
			newAdpt: func() adapters.Adapter { return &BrewAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"brew": true},
				cmdArgs:  map[string]fakeResult{"brew": {}},
				shell:    map[string]fakeResult{brewUpdateCmd: failIfRun},
			},
			want: adapters.UpdateInfo{CurrentVersion: "", LatestVersion: "", UpdateAvailable: false},
		},
		{
			name:    "brew/command-fails",
			newAdpt: func() adapters.Adapter { return &BrewAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"brew": true},
				cmdArgs:  map[string]fakeResult{"brew": {err: errors.New("brew: command not found")}},
				shell:    map[string]fakeResult{brewUpdateCmd: failIfRun},
			},
			want: adapters.UpdateInfo{CurrentVersion: "", LatestVersion: "", UpdateAvailable: false},
		},
		{
			name:    "brew/not-installed-error",
			newAdpt: func() adapters.Adapter { return &BrewAdapter{} },
			fakes:   execFakes{lookPath: map[string]bool{"brew": false}},
			wantErr: true,
		},

		// --- npm (version + outdated) ---
		{
			name:    "npm/update-available",
			newAdpt: func() adapters.Adapter { return &NpmAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"npm": true},
				cmdArgs: map[string]fakeResult{
					"npm":                       {stdout: "10.2.4"},
					"npm outdated -g --depth=0": {stdout: "npm  10.2.5  10.2.5  10.2.5"},
				},
			},
			want: adapters.UpdateInfo{CurrentVersion: "10.2.4", LatestVersion: "10.2.4", UpdateAvailable: true},
		},
		{
			name:    "npm/no-outdated",
			newAdpt: func() adapters.Adapter { return &NpmAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"npm": true},
				cmdArgs: map[string]fakeResult{
					"npm":                       {stdout: "10.2.4"},
					"npm outdated -g --depth=0": {stdout: ""},
				},
			},
			want: adapters.UpdateInfo{CurrentVersion: "10.2.4", LatestVersion: "10.2.4", UpdateAvailable: false},
		},
		{
			name:    "npm/empty-version-unknown",
			newAdpt: func() adapters.Adapter { return &NpmAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"npm": true},
				cmdArgs: map[string]fakeResult{
					"npm":                       {},
					"npm outdated -g --depth=0": {stdout: ""},
				},
			},
			want: adapters.UpdateInfo{CurrentVersion: "unknown", LatestVersion: "unknown", UpdateAvailable: false},
		},
		{
			name:    "npm/exit-1-outdated",
			newAdpt: func() adapters.Adapter { return &NpmAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"npm": true},
				cmdArgs: map[string]fakeResult{
					"npm":                       {stdout: "10.2.4"},
					"npm outdated -g --depth=0": {stdout: "npm  10.2.5  10.2.5  10.2.5", err: errors.New("sentinel")},
				},
			},
			exitCode: &one,
			want:     adapters.UpdateInfo{CurrentVersion: "10.2.4", LatestVersion: "10.2.4", UpdateAvailable: true},
		},
		{
			name:    "npm/exit-1-empty-output",
			newAdpt: func() adapters.Adapter { return &NpmAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"npm": true},
				cmdArgs: map[string]fakeResult{
					"npm":                       {stdout: "10.2.4"},
					"npm outdated -g --depth=0": {stdout: "", err: errors.New("sentinel")},
				},
			},
			exitCode:        &one,
			wantErr:         true,
			wantErrContains: "(exit 1)",
		},
		{
			name:    "npm/other-nonzero-exit",
			newAdpt: func() adapters.Adapter { return &NpmAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"npm": true},
				cmdArgs: map[string]fakeResult{
					"npm":                       {stdout: "10.2.4"},
					"npm outdated -g --depth=0": {stdout: "", err: errors.New("sentinel")},
				},
			},
			exitCode:        &two,
			wantErr:         true,
			wantErrContains: "(exit 2)",
		},
		{
			name:    "npm/deadline-exceeded",
			newAdpt: func() adapters.Adapter { return &NpmAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"npm": true},
				cmdArgs: map[string]fakeResult{
					"npm":                       {stdout: "10.2.4"},
					"npm outdated -g --depth=0": {err: context.DeadlineExceeded},
				},
			},
			wantErr:      true,
			wantDeadline: true,
		},
		{
			name:    "npm/not-installed-error",
			newAdpt: func() adapters.Adapter { return &NpmAdapter{} },
			fakes:   execFakes{lookPath: map[string]bool{"npm": false}},
			wantErr: true,
		},

		// --- pnpm (version + outdated table parsing) ---
		{
			name:    "pnpm/update-available",
			newAdpt: func() adapters.Adapter { return &PnpmAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"pnpm": true},
				cmdArgs: map[string]fakeResult{
					"pnpm":             {stdout: "8.14.0"},
					"pnpm outdated -g": {stdout: "├─ foo │ 1.0.0 │ 2.0.0 │"},
				},
			},
			want: adapters.UpdateInfo{CurrentVersion: "8.14.0", LatestVersion: "8.14.0", UpdateAvailable: true},
		},
		{
			name:    "pnpm/outdated-header-only",
			newAdpt: func() adapters.Adapter { return &PnpmAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"pnpm": true},
				cmdArgs: map[string]fakeResult{
					"pnpm":             {stdout: "8.14.0"},
					"pnpm outdated -g": {stdout: "Package │ Current │ Latest"},
				},
			},
			want: adapters.UpdateInfo{CurrentVersion: "8.14.0", LatestVersion: "8.14.0", UpdateAvailable: false},
		},
		{
			name:    "pnpm/empty-version-unknown",
			newAdpt: func() adapters.Adapter { return &PnpmAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"pnpm": true},
				cmdArgs: map[string]fakeResult{
					"pnpm":             {},
					"pnpm outdated -g": {stdout: ""},
				},
			},
			want: adapters.UpdateInfo{CurrentVersion: "unknown", LatestVersion: "unknown", UpdateAvailable: false},
		},
		{
			name:    "pnpm/exit-1-outdated",
			newAdpt: func() adapters.Adapter { return &PnpmAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"pnpm": true},
				cmdArgs: map[string]fakeResult{
					"pnpm":             {stdout: "8.14.0"},
					"pnpm outdated -g": {stdout: "├─ foo │ 1.0.0 │ 2.0.0 │", err: errors.New("sentinel")},
				},
			},
			exitCode: &one,
			want:     adapters.UpdateInfo{CurrentVersion: "8.14.0", LatestVersion: "8.14.0", UpdateAvailable: true},
		},
		{
			name:    "pnpm/exit-1-empty-output",
			newAdpt: func() adapters.Adapter { return &PnpmAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"pnpm": true},
				cmdArgs: map[string]fakeResult{
					"pnpm":             {stdout: "8.14.0"},
					"pnpm outdated -g": {stdout: "", err: errors.New("sentinel")},
				},
			},
			exitCode:        &one,
			wantErr:         true,
			wantErrContains: "(exit 1)",
		},
		{
			name:    "pnpm/other-nonzero-exit",
			newAdpt: func() adapters.Adapter { return &PnpmAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"pnpm": true},
				cmdArgs: map[string]fakeResult{
					"pnpm":             {stdout: "8.14.0"},
					"pnpm outdated -g": {stdout: "", err: errors.New("sentinel")},
				},
			},
			exitCode:        &two,
			wantErr:         true,
			wantErrContains: "(exit 2)",
		},
		{
			name:    "pnpm/not-installed-error",
			newAdpt: func() adapters.Adapter { return &PnpmAdapter{} },
			fakes:   execFakes{lookPath: map[string]bool{"pnpm": false}},
			wantErr: true,
		},

		// --- bun (version only) ---
		{
			name:    "bun/normal",
			newAdpt: func() adapters.Adapter { return &BunAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"bun": true},
				cmdArgs:  map[string]fakeResult{"bun": {stdout: "1.0.30"}},
			},
			want: adapters.UpdateInfo{CurrentVersion: "1.0.30", LatestVersion: "1.0.30", UpdateAvailable: false},
		},
		{
			name:    "bun/empty-version-unknown",
			newAdpt: func() adapters.Adapter { return &BunAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"bun": true},
				cmdArgs:  map[string]fakeResult{"bun": {}},
			},
			want: adapters.UpdateInfo{CurrentVersion: "unknown", LatestVersion: "unknown", UpdateAvailable: false},
		},
		{
			name:    "bun/not-installed-error",
			newAdpt: func() adapters.Adapter { return &BunAdapter{} },
			fakes:   execFakes{lookPath: map[string]bool{"bun": false}},
			wantErr: true,
		},

		// --- gh (version extraction) ---
		{
			name:    "gh/normal",
			newAdpt: func() adapters.Adapter { return &GhAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"gh": true},
				cmdArgs:  map[string]fakeResult{"gh": {stdout: "gh version 2.45.0 (2024-05-30)"}},
			},
			want: adapters.UpdateInfo{CurrentVersion: "2.45.0", LatestVersion: "2.45.0", UpdateAvailable: false},
		},
		{
			name:    "gh/empty-output",
			newAdpt: func() adapters.Adapter { return &GhAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"gh": true},
				cmdArgs:  map[string]fakeResult{"gh": {}},
			},
			want: adapters.UpdateInfo{CurrentVersion: "", LatestVersion: "", UpdateAvailable: false},
		},
		{
			name:    "gh/not-installed-error",
			newAdpt: func() adapters.Adapter { return &GhAdapter{} },
			fakes:   execFakes{lookPath: map[string]bool{"gh": false}},
			wantErr: true,
		},

		// --- docker (version extraction) ---
		{
			name:    "docker/normal",
			newAdpt: func() adapters.Adapter { return &DockerAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"docker": true},
				cmdArgs:  map[string]fakeResult{"docker": {stdout: "Docker version 26.1.4, build 5650f9b"}},
			},
			want: adapters.UpdateInfo{CurrentVersion: "26.1.4", LatestVersion: "26.1.4", UpdateAvailable: false},
		},
		{
			name:    "docker/empty-output",
			newAdpt: func() adapters.Adapter { return &DockerAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"docker": true},
				cmdArgs:  map[string]fakeResult{"docker": {}},
			},
			want: adapters.UpdateInfo{CurrentVersion: "", LatestVersion: "", UpdateAvailable: false},
		},
		{
			name:    "docker/not-installed-error",
			newAdpt: func() adapters.Adapter { return &DockerAdapter{} },
			fakes:   execFakes{lookPath: map[string]bool{"docker": false}},
			wantErr: true,
		},

		// --- go (go version parsing) ---
		{
			name:    "go/normal",
			newAdpt: func() adapters.Adapter { return &GoAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"go": true},
				cmdArgs:  map[string]fakeResult{"go": {stdout: "go version go1.22.0 linux/amd64"}},
			},
			want: adapters.UpdateInfo{CurrentVersion: "1.22.0", LatestVersion: "1.22.0", UpdateAvailable: false},
		},
		{
			name:    "go/empty-output",
			newAdpt: func() adapters.Adapter { return &GoAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"go": true},
				cmdArgs:  map[string]fakeResult{"go": {}},
			},
			want: adapters.UpdateInfo{CurrentVersion: "", LatestVersion: "", UpdateAvailable: false},
		},
		{
			name:    "go/not-installed-error",
			newAdpt: func() adapters.Adapter { return &GoAdapter{} },
			fakes:   execFakes{lookPath: map[string]bool{"go": false}},
			wantErr: true,
		},

		// --- opencode (version extraction) ---
		{
			name:    "opencode/normal",
			newAdpt: func() adapters.Adapter { return &OpenCodeAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"opencode": true},
				cmdArgs:  map[string]fakeResult{"opencode": {stdout: "opencode v0.3.6"}},
			},
			want: adapters.UpdateInfo{CurrentVersion: "v0.3.6", LatestVersion: "v0.3.6", UpdateAvailable: false},
		},
		{
			name:    "opencode/empty-output",
			newAdpt: func() adapters.Adapter { return &OpenCodeAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"opencode": true},
				cmdArgs:  map[string]fakeResult{"opencode": {}},
			},
			want: adapters.UpdateInfo{CurrentVersion: "", LatestVersion: "", UpdateAvailable: false},
		},
		{
			name:    "opencode/not-installed-error",
			newAdpt: func() adapters.Adapter { return &OpenCodeAdapter{} },
			fakes:   execFakes{lookPath: map[string]bool{"opencode": false}},
			wantErr: true,
		},

		// --- winget (unknown/unknown/available) ---
		{
			name:    "winget/always-unknown",
			newAdpt: func() adapters.Adapter { return &WingetAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"winget": true},
				cmdArgs:  map[string]fakeResult{"winget": {stdout: "Name  Version  Available"}},
			},
			want: adapters.UpdateInfo{CurrentVersion: "unknown", LatestVersion: "unknown", UpdateAvailable: true},
		},
		{
			name:    "winget/not-installed-error",
			newAdpt: func() adapters.Adapter { return &WingetAdapter{} },
			fakes:   execFakes{lookPath: map[string]bool{"winget": false}},
			wantErr: true,
		},

		// --- scoop (unknown/unknown/available) ---
		{
			name:    "scoop/always-unknown",
			newAdpt: func() adapters.Adapter { return &ScoopAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"scoop": true},
				cmdArgs:  map[string]fakeResult{"scoop": {stdout: "Name  Installed  Latest"}},
			},
			want: adapters.UpdateInfo{CurrentVersion: "unknown", LatestVersion: "unknown", UpdateAvailable: true},
		},
		{
			name:    "scoop/not-installed-error",
			newAdpt: func() adapters.Adapter { return &ScoopAdapter{} },
			fakes:   execFakes{lookPath: map[string]bool{"scoop": false}},
			wantErr: true,
		},

		// --- nvm (env-driven detect + shell version parsing) ---
		{
			name:    "nvm/update-available",
			newAdpt: func() adapters.Adapter { return &NVMAdapter{} },
			setup:   nvmInstalledSetup,
			fakes: execFakes{
				shell: map[string]fakeResult{
					nvmCurrentCmd: {stdout: "v20.11.0"},
					nvmRemoteCmd:  {stdout: "v22.0.0"},
				},
			},
			want: adapters.UpdateInfo{CurrentVersion: "v20.11.0", LatestVersion: "v22.0.0", UpdateAvailable: true},
		},
		{
			name:    "nvm/empty-current-unknown",
			newAdpt: func() adapters.Adapter { return &NVMAdapter{} },
			setup:   nvmInstalledSetup,
			fakes: execFakes{
				shell: map[string]fakeResult{
					nvmCurrentCmd: {stdout: ""},
					nvmRemoteCmd:  {stdout: "v22.0.0"},
				},
			},
			want: adapters.UpdateInfo{CurrentVersion: "unknown", LatestVersion: "v22.0.0", UpdateAvailable: false},
		},
		{
			name:    "nvm/same-version",
			newAdpt: func() adapters.Adapter { return &NVMAdapter{} },
			setup:   nvmInstalledSetup,
			fakes: execFakes{
				shell: map[string]fakeResult{
					nvmCurrentCmd: {stdout: "v20.11.0"},
					nvmRemoteCmd:  {stdout: "v20.11.0"},
				},
			},
			want: adapters.UpdateInfo{CurrentVersion: "v20.11.0", LatestVersion: "v20.11.0", UpdateAvailable: false},
		},
		// S2 D7: newer current must NOT be reported as an update (no phantom
		// downgrade) — semver comparison, not string inequality.
		{
			name:    "nvm/newer-current-no-downgrade",
			newAdpt: func() adapters.Adapter { return &NVMAdapter{} },
			setup:   nvmInstalledSetup,
			fakes: execFakes{
				shell: map[string]fakeResult{
					nvmCurrentCmd: {stdout: "v26.7.0"},
					nvmRemoteCmd:  {stdout: "v24.19.0"},
				},
			},
			want: adapters.UpdateInfo{CurrentVersion: "v26.7.0", LatestVersion: "v24.19.0", UpdateAvailable: false},
		},
		// S2 D7: equal versions with/without the v prefix parse to the same
		// version — no update.
		{
			name:    "nvm/equal-versions-v-prefix-tolerated",
			newAdpt: func() adapters.Adapter { return &NVMAdapter{} },
			setup:   nvmInstalledSetup,
			fakes: execFakes{
				shell: map[string]fakeResult{
					nvmCurrentCmd: {stdout: "v20.11.0"},
					nvmRemoteCmd:  {stdout: "20.11.0"},
				},
			},
			want: adapters.UpdateInfo{CurrentVersion: "v20.11.0", LatestVersion: "20.11.0", UpdateAvailable: false},
		},
		// S2 D7: an unparseable latest (nvm ls-remote alias like "stable")
		// is unknown — false, no error (adapter Unparseable scenario).
		{
			name:    "nvm/unparseable-stable-no-error",
			newAdpt: func() adapters.Adapter { return &NVMAdapter{} },
			setup:   nvmInstalledSetup,
			fakes: execFakes{
				shell: map[string]fakeResult{
					nvmCurrentCmd: {stdout: "v26.7.0"},
					nvmRemoteCmd:  {stdout: "stable"},
				},
			},
			want: adapters.UpdateInfo{CurrentVersion: "v26.7.0", LatestVersion: "stable", UpdateAvailable: false},
		},
		// S2 D7: a dev-ish current parses as Dev → fail-closed false.
		{
			name:    "nvm/current-dev-no-update",
			newAdpt: func() adapters.Adapter { return &NVMAdapter{} },
			setup:   nvmInstalledSetup,
			fakes: execFakes{
				shell: map[string]fakeResult{
					nvmCurrentCmd: {stdout: "dev"},
					nvmRemoteCmd:  {stdout: "v22.0.0"},
				},
			},
			want: adapters.UpdateInfo{CurrentVersion: "dev", LatestVersion: "v22.0.0", UpdateAvailable: false},
		},
		// Multi-LTS: ls-remote ends with an alias line ("lts/* -> lts/jod").
		// The latest version must come from the last real vX.Y.Z line, not the
		// alias line — otherwise a real update is silently missed (regression
		// guard for the tail -1 | awk extraction).
		{
			name:    "nvm/multi-lts-alias-line",
			newAdpt: func() adapters.Adapter { return &NVMAdapter{} },
			setup:   nvmInstalledSetup,
			fakes: execFakes{
				shell: map[string]fakeResult{
					nvmCurrentCmd: {stdout: "v20.11.0"},
					nvmRemoteCmd:  {stdout: "v22.3.0"},
				},
			},
			want: adapters.UpdateInfo{CurrentVersion: "v20.11.0", LatestVersion: "v22.3.0", UpdateAvailable: true},
		},
		{
			name:    "nvm/command-fails",
			newAdpt: func() adapters.Adapter { return &NVMAdapter{} },
			setup:   nvmInstalledSetup,
			fakes: execFakes{
				shell: map[string]fakeResult{
					nvmCurrentCmd: {err: errors.New("nvm: command not found")},
					nvmRemoteCmd:  {err: errors.New("nvm: command not found")},
				},
			},
			wantErr:         true,
			wantErrContains: "nvm check failed",
		},
		{
			name:    "nvm/not-installed-error",
			newAdpt: func() adapters.Adapter { return &NVMAdapter{} },
			setup:   nvmMissingSetup,
			fakes:   execFakes{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup(t)
			}
			setExecFakes(t, tt.fakes)
			if tt.exitCode != nil {
				if runtime.GOOS == "windows" {
					t.Skip("exit-code rows run a real sh child")
				}
				realErr := exitErrFromChild(t, *tt.exitCode)
				for key, r := range tt.fakes.shell {
					if r.err != nil {
						r.err = realErr
						tt.fakes.shell[key] = r
					}
				}
				for key, r := range tt.fakes.cmdArgs {
					if r.err != nil {
						r.err = realErr
						tt.fakes.cmdArgs[key] = r
					}
				}
			}

			got, err := tt.newAdpt().Check()
			if tt.wantErr {
				if err == nil {
					t.Fatal("Check() error = nil, want error")
				}
				if tt.wantErrContains != "" && !strings.Contains(err.Error(), tt.wantErrContains) {
					t.Errorf("Check() error = %q, want contains %q", err, tt.wantErrContains)
				}
				if tt.wantDeadline && !errors.Is(err, context.DeadlineExceeded) {
					t.Errorf("Check() error = %v, want errors.Is(err, context.DeadlineExceeded)", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Check() unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("Check() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

var (
	one = 1
	two = 2
)
