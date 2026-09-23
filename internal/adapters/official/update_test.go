package official

import (
	"context"
	"errors"
	"runtime"
	"strings"
	"testing"

	"github.com/JhnFrankz/upp/internal/adapters"
)

// Shared fake keys for the update commands, defined once so every table row
// reuses the exact same strings the production adapters call through
// runCmd/runCmdArgs. A mismatch between key and production command is a test
// failure, never a silent fake miss.
const (
	aptUpdateCmd      = "sudo apt install --only-upgrade apt"
	pacmanUpdateCmd   = "sudo pacman -S --noconfirm pacman"
	brewUpdateCmd     = "brew update"
	npmUpdateCmd      = "npm update -g"
	pnpmUpdateCmd     = "pnpm update -g"
	pnpmPruneCmd      = "pnpm store prune"
	bunUpdateCmd      = "bun upgrade"
	opencodeUpdateCmd = "opencode update"
	wingetUpdateCmd   = "winget upgrade winget"
	scoopUpdateCmd    = "scoop update scoop"
	nvmInstallLtsCmd  = "bash -c 'source \"${NVM_DIR:-$HOME/.nvm}/nvm.sh\" >/dev/null 2>&1 && nvm install --lts'"
	uvSelfUpdateCmd   = "uv self update"
	uvToolUpgradeCmd  = "uv tool upgrade --all"
)

// failIfRun is a fake result that fails loudly: any row that keys a command
// with this result asserts "this command MUST NOT be executed". Dry-run rows
// use it for the update command — a dry run that executes the update command
// would hit the error and return Success=false.
var failIfRun = fakeResult{err: errors.New("command must not be executed")}

// updateCase is one table row for Update(): fakes drive the exec seam, setup
// prepares the environment (nvm), goos restricts a row to one GOOS ("" = any),
// dryRun is the argument passed to Update, want is the expected Result,
// wantErr requires a function-level error (tool not installed) and resultErr
// requires a non-nil Result.Error (command failure / stderr marker).
type updateCase struct {
	name      string
	newAdpt   func() adapters.Adapter
	fakes     execFakes
	setup     func(t *testing.T)
	goos      string
	dryRun    bool
	pkg       string
	want      adapters.Result
	wantErr   bool
	resultErr bool
}

func equalPrivileges(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestUpdate covers Update() for all 12 official adapters hermetically:
// dry-run shortcut, not-installed error, command failure, stderr markers and
// success with before/after versions. No real subprocess ever runs; docker,
// gh and go are tested on the current GOOS branch only (GOOS is not mockable).
func TestUpdate(t *testing.T) {
	sudo := []string{"sudo"}

	tests := []updateCase{
		// --- apt (shell-based version + sudo) ---
		{
			name:    "apt/not-installed-error",
			newAdpt: func() adapters.Adapter { return &AptAdapter{} },
			fakes:   execFakes{lookPath: map[string]bool{"apt": false}},
			wantErr: true,
		},
		{
			name:    "apt/dry-run-shortcut",
			newAdpt: func() adapters.Adapter { return &AptAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"apt": true},
				shell: map[string]fakeResult{
					aptInstalledCmd: {stdout: "2.4.0"},
					aptUpdateCmd:    failIfRun,
				},
			},
			dryRun: true,
			want:   adapters.Result{Success: true, Before: "2.4.0", After: "2.4.0"},
		},
		{
			name:    "apt/update-command-error",
			newAdpt: func() adapters.Adapter { return &AptAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"apt": true},
				shell: map[string]fakeResult{
					aptInstalledCmd: {stdout: "2.4.0"},
					aptUpdateCmd:    {err: errors.New("sudo: command not found")},
				},
			},
			want:      adapters.Result{Success: false, Before: "2.4.0", After: "2.4.0", Privileges: sudo},
			resultErr: true,
		},
		{
			name:    "apt/stderr-error-marker",
			newAdpt: func() adapters.Adapter { return &AptAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"apt": true},
				shell: map[string]fakeResult{
					aptInstalledCmd: {stdout: "2.4.0"},
					aptUpdateCmd:    {stderr: "E: Unable to acquire the dpkg frontend lock"},
				},
			},
			want:      adapters.Result{Success: false, Before: "2.4.0", After: "2.4.0", Privileges: sudo},
			resultErr: true,
		},
		{
			name:    "apt/success",
			newAdpt: func() adapters.Adapter { return &AptAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"apt": true},
				shell: map[string]fakeResult{
					aptInstalledCmd: {stdout: "2.4.0"},
					aptUpdateCmd:    {stdout: "Reading package lists..."},
				},
			},
			want: adapters.Result{Success: true, Before: "2.4.0", After: "2.4.0", Privileges: sudo},
		},

		// --- pacman (shell-based version + sudo) ---
		{
			name:    "pacman/not-installed-error",
			newAdpt: func() adapters.Adapter { return &PacmanAdapter{} },
			fakes:   execFakes{lookPath: map[string]bool{"pacman": false}},
			wantErr: true,
		},
		{
			name:    "pacman/dry-run-shortcut",
			newAdpt: func() adapters.Adapter { return &PacmanAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"pacman": true},
				shell: map[string]fakeResult{
					pacmanInstalledCmd: {stdout: "6.1.0-1"},
					pacmanUpdateCmd:    failIfRun,
				},
			},
			dryRun: true,
			want:   adapters.Result{Success: true, Before: "6.1.0-1", After: "6.1.0-1"},
		},
		{
			name:    "pacman/update-command-error",
			newAdpt: func() adapters.Adapter { return &PacmanAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"pacman": true},
				shell: map[string]fakeResult{
					pacmanInstalledCmd: {stdout: "6.1.0-1"},
					pacmanUpdateCmd:    {err: errors.New("sudo: command not found")},
				},
			},
			want:      adapters.Result{Success: false, Before: "6.1.0-1", After: "6.1.0-1", Privileges: sudo},
			resultErr: true,
		},
		{
			name:    "pacman/stderr-marker-fails",
			newAdpt: func() adapters.Adapter { return &PacmanAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"pacman": true},
				shell: map[string]fakeResult{
					pacmanInstalledCmd: {stdout: "6.1.0-1"},
					pacmanUpdateCmd:    {stderr: "error: failed to commit transaction"},
				},
			},
			want:      adapters.Result{Success: false, Before: "6.1.0-1", After: "6.1.0-1", Privileges: sudo},
			resultErr: true,
		},
		{
			name:    "pacman/success",
			newAdpt: func() adapters.Adapter { return &PacmanAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"pacman": true},
				shell: map[string]fakeResult{
					pacmanInstalledCmd: {stdout: "6.1.0-1"},
					pacmanUpdateCmd:    {stdout: "resolving dependencies...\nupgrading pacman..."},
				},
			},
			want: adapters.Result{Success: true, Before: "6.1.0-1", After: "6.1.0-1", Privileges: sudo},
		},

		// --- brew (version extraction + shell update) ---
		{
			name:    "brew/not-installed-error",
			newAdpt: func() adapters.Adapter { return &BrewAdapter{} },
			fakes:   execFakes{lookPath: map[string]bool{"brew": false}},
			wantErr: true,
		},
		{
			name:    "brew/dry-run-shortcut",
			newAdpt: func() adapters.Adapter { return &BrewAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"brew": true},
				cmdArgs:  map[string]fakeResult{"brew": {stdout: "Homebrew 4.1.0"}},
				shell:    map[string]fakeResult{brewUpdateCmd: failIfRun},
			},
			dryRun: true,
			want:   adapters.Result{Success: true, Before: "4.1.0", After: "4.1.0"},
		},
		{
			name:    "brew/update-command-error",
			newAdpt: func() adapters.Adapter { return &BrewAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"brew": true},
				cmdArgs:  map[string]fakeResult{"brew": {stdout: "Homebrew 4.1.0"}},
				shell:    map[string]fakeResult{brewUpdateCmd: {err: errors.New("brew: network error")}},
			},
			want:      adapters.Result{Success: false, Before: "4.1.0", After: "4.1.0"},
			resultErr: true,
		},
		{
			name:    "brew/stderr-error-marker",
			newAdpt: func() adapters.Adapter { return &BrewAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"brew": true},
				cmdArgs:  map[string]fakeResult{"brew": {stdout: "Homebrew 4.1.0"}},
				shell:    map[string]fakeResult{brewUpdateCmd: {stderr: "Error: Permission denied @ apply2files"}},
			},
			want:      adapters.Result{Success: false, Before: "4.1.0", After: "4.1.0"},
			resultErr: true,
		},
		{
			name:    "brew/success",
			newAdpt: func() adapters.Adapter { return &BrewAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"brew": true},
				cmdArgs:  map[string]fakeResult{"brew": {stdout: "Homebrew 4.1.0"}},
				shell:    map[string]fakeResult{brewUpdateCmd: {}},
			},
			want: adapters.Result{Success: true, Before: "4.1.0", After: "4.1.0"},
		},

		// --- npm (version + outdated-aware update) ---
		{
			name:    "npm/not-installed-error",
			newAdpt: func() adapters.Adapter { return &NpmAdapter{} },
			fakes:   execFakes{lookPath: map[string]bool{"npm": false}},
			wantErr: true,
		},
		{
			name:    "npm/dry-run-shortcut",
			newAdpt: func() adapters.Adapter { return &NpmAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"npm": true},
				cmdArgs:  map[string]fakeResult{"npm": {stdout: "10.2.4"}},
				shell:    map[string]fakeResult{npmUpdateCmd: failIfRun},
			},
			dryRun: true,
			want:   adapters.Result{Success: true, Before: "10.2.4", After: "10.2.4"},
		},
		{
			name:    "npm/update-command-error",
			newAdpt: func() adapters.Adapter { return &NpmAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"npm": true},
				cmdArgs:  map[string]fakeResult{"npm": {stdout: "10.2.4"}},
				shell:    map[string]fakeResult{npmUpdateCmd: {err: errors.New("npm: ENOENT")}},
			},
			want:      adapters.Result{Success: false, Before: "10.2.4", After: "10.2.4"},
			resultErr: true,
		},
		{
			name:    "npm/stderr-error-marker",
			newAdpt: func() adapters.Adapter { return &NpmAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"npm": true},
				cmdArgs:  map[string]fakeResult{"npm": {stdout: "10.2.4"}},
				shell:    map[string]fakeResult{npmUpdateCmd: {stderr: "npm ERR! code EEXIST"}},
			},
			want:      adapters.Result{Success: false, Before: "10.2.4", After: "10.2.4"},
			resultErr: true,
		},
		{
			name:    "npm/success",
			newAdpt: func() adapters.Adapter { return &NpmAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"npm": true},
				cmdArgs:  map[string]fakeResult{"npm": {stdout: "10.2.4"}},
				shell:    map[string]fakeResult{npmUpdateCmd: {}},
			},
			want: adapters.Result{Success: true, Before: "10.2.4", After: "10.2.4"},
		},

		// --- pnpm (version + corruption recovery) ---
		{
			name:    "pnpm/not-installed-error",
			newAdpt: func() adapters.Adapter { return &PnpmAdapter{} },
			fakes:   execFakes{lookPath: map[string]bool{"pnpm": false}},
			wantErr: true,
		},
		{
			name:    "pnpm/dry-run-shortcut",
			newAdpt: func() adapters.Adapter { return &PnpmAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"pnpm": true},
				cmdArgs:  map[string]fakeResult{"pnpm": {stdout: "8.14.0"}},
				shell:    map[string]fakeResult{pnpmUpdateCmd: failIfRun},
			},
			dryRun: true,
			want:   adapters.Result{Success: true, Before: "8.14.0", After: "8.14.0"},
		},
		{
			name:    "pnpm/update-command-error",
			newAdpt: func() adapters.Adapter { return &PnpmAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"pnpm": true},
				cmdArgs:  map[string]fakeResult{"pnpm": {stdout: "8.14.0"}},
				shell:    map[string]fakeResult{pnpmUpdateCmd: {err: errors.New("pnpm: ETIMEDOUT")}},
			},
			want:      adapters.Result{Success: false, Before: "8.14.0", After: "8.14.0"},
			resultErr: true,
		},
		{
			name:    "pnpm/corruption-recovery-fails",
			newAdpt: func() adapters.Adapter { return &PnpmAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"pnpm": true},
				cmdArgs:  map[string]fakeResult{"pnpm": {stdout: "8.14.0"}},
				shell: map[string]fakeResult{
					pnpmUpdateCmd: {stderr: "ERR_PNPM_STORE_BROKEN corrupt store", err: errors.New("store broken")},
					pnpmPruneCmd:  {},
				},
			},
			want:      adapters.Result{Success: false, Before: "8.14.0", After: "8.14.0"},
			resultErr: true,
		},
		{
			name:    "pnpm/success",
			newAdpt: func() adapters.Adapter { return &PnpmAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"pnpm": true},
				cmdArgs:  map[string]fakeResult{"pnpm": {stdout: "8.14.0"}},
				shell:    map[string]fakeResult{pnpmUpdateCmd: {}},
			},
			want: adapters.Result{Success: true, Before: "8.14.0", After: "8.14.0"},
		},

		// --- bun (version + upgrade) ---
		{
			name:    "bun/not-installed-error",
			newAdpt: func() adapters.Adapter { return &BunAdapter{} },
			fakes:   execFakes{lookPath: map[string]bool{"bun": false}},
			wantErr: true,
		},
		{
			name:    "bun/dry-run-shortcut",
			newAdpt: func() adapters.Adapter { return &BunAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"bun": true},
				cmdArgs:  map[string]fakeResult{"bun": {stdout: "1.0.30"}},
				shell:    map[string]fakeResult{bunUpdateCmd: failIfRun},
			},
			dryRun: true,
			want:   adapters.Result{Success: true, Before: "1.0.30", After: "1.0.30"},
		},
		{
			name:    "bun/update-command-error",
			newAdpt: func() adapters.Adapter { return &BunAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"bun": true},
				cmdArgs:  map[string]fakeResult{"bun": {stdout: "1.0.30"}},
				shell:    map[string]fakeResult{bunUpdateCmd: {err: errors.New("bun upgrade: failed")}},
			},
			want:      adapters.Result{Success: false, Before: "1.0.30", After: "1.0.30"},
			resultErr: true,
		},
		{
			name:    "bun/stderr-error-marker",
			newAdpt: func() adapters.Adapter { return &BunAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"bun": true},
				cmdArgs:  map[string]fakeResult{"bun": {stdout: "1.0.30"}},
				shell:    map[string]fakeResult{bunUpdateCmd: {stderr: "error: Failed to download"}},
			},
			want:      adapters.Result{Success: false, Before: "1.0.30", After: "1.0.30"},
			resultErr: true,
		},
		{
			name:    "bun/success",
			newAdpt: func() adapters.Adapter { return &BunAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"bun": true},
				cmdArgs:  map[string]fakeResult{"bun": {stdout: "1.0.30"}},
				shell:    map[string]fakeResult{bunUpdateCmd: {}},
			},
			want: adapters.Result{Success: true, Before: "1.0.30", After: "1.0.30"},
		},

		// --- gh (delegated to resolving manager via PackageUpdater) ---
		{
			name:    "gh/not-installed-error",
			newAdpt: func() adapters.Adapter { return &GhAdapter{} },
			fakes:   execFakes{lookPath: map[string]bool{"gh": false}},
			wantErr: true,
		},
		{
			name:    "gh/linux-dry-run-delegates-to-apt",
			newAdpt: func() adapters.Adapter { return &GhAdapter{} },
			goos:    "linux",
			fakes: execFakes{
				lookPath: map[string]bool{"gh": true, "apt": true},
				cmdArgs:  map[string]fakeResult{"gh": {stdout: "gh version 2.45.0 (2024-05-30)"}},
				shell: map[string]fakeResult{
					aptInstalledCmd:                      {stdout: "2.4.0"},
					"sudo apt install --only-upgrade gh": failIfRun, // dry-run must never exec the update cmd
				},
			},
			dryRun: true,
			want:   adapters.Result{Success: true},
		},
		{
			name:    "gh/linux-update-delegates-to-apt-error",
			newAdpt: func() adapters.Adapter { return &GhAdapter{} },
			goos:    "linux",
			fakes: execFakes{
				lookPath: map[string]bool{"gh": true, "apt": true},
				cmdArgs: map[string]fakeResult{
					"gh":                  {stdout: "gh version 2.45.0 (2024-05-30)"},
					"apt-cache policy gh": {stdout: "gh:\n  Installed: 2.45.0\n  Candidate: 2.46.0\n"},
				},
				shell: map[string]fakeResult{
					"sudo apt install --only-upgrade gh": {err: errors.New("apt: lock held")},
				},
			},
			want:      adapters.Result{Success: false, Before: "2.45.0", After: "2.45.0", Privileges: sudo},
			resultErr: true,
		},
		{
			name:    "gh/linux-update-delegates-to-apt-stderr",
			newAdpt: func() adapters.Adapter { return &GhAdapter{} },
			goos:    "linux",
			fakes: execFakes{
				lookPath: map[string]bool{"gh": true, "apt": true},
				cmdArgs: map[string]fakeResult{
					"gh":                  {stdout: "gh version 2.45.0 (2024-05-30)"},
					"apt-cache policy gh": {stdout: "gh:\n  Installed: 2.45.0\n  Candidate: 2.46.0\n"},
				},
				shell: map[string]fakeResult{
					"sudo apt install --only-upgrade gh": {stderr: "E: Unable to acquire the dpkg frontend lock"},
				},
			},
			want:      adapters.Result{Success: false, Before: "2.45.0", After: "2.45.0", Privileges: sudo},
			resultErr: true,
		},
		{
			name:    "gh/linux-update-delegates-to-apt-success",
			newAdpt: func() adapters.Adapter { return &GhAdapter{} },
			goos:    "linux",
			fakes: execFakes{
				lookPath: map[string]bool{"gh": true, "apt": true},
				cmdArgs: map[string]fakeResult{
					"gh":                  {stdout: "gh version 2.45.0 (2024-05-30)"},
					"apt-cache policy gh": {stdout: "gh:\n  Installed: 2.45.0\n  Candidate: 2.46.0\n"},
				},
				shell: map[string]fakeResult{
					"sudo apt install --only-upgrade gh": {},
				},
			},
			want: adapters.Result{Success: true, Before: "2.45.0", After: "2.45.0", Privileges: sudo},
		},
		{
			name:    "gh/linux-update-delegates-to-pacman-success",
			newAdpt: func() adapters.Adapter { return &GhAdapter{} },
			goos:    "linux",
			fakes: execFakes{
				lookPath: map[string]bool{"gh": true, "apt": false, "pacman": true},
				cmdArgs: map[string]fakeResult{
					"gh":                   {stdout: "gh version 2.45.0 (2024-05-30)"},
					"pacman -Q github-cli": {stdout: "github-cli 2.45.0-1"},
				},
				shell: map[string]fakeResult{
					"sudo pacman -S --noconfirm github-cli": {},
				},
			},
			want: adapters.Result{Success: true, Before: "2.45.0-1", After: "2.45.0-1", Privileges: sudo},
		},
		{
			name:    "gh/macos-delegates-to-brew-success",
			newAdpt: func() adapters.Adapter { return &GhAdapter{} },
			goos:    "darwin",
			fakes: execFakes{
				lookPath: map[string]bool{"gh": true, "brew": true},
				cmdArgs: map[string]fakeResult{
					"gh":                      {stdout: "gh version 2.45.0 (2024-05-30)"},
					"brew outdated --json gh": {stdout: `[{"name":"gh","installed_versions":["2.45.0"],"current_version":"2.46.0"}]`},
				},
				shell: map[string]fakeResult{"brew upgrade gh": {}},
			},
			want: adapters.Result{Success: true, Before: "2.45.0", After: "2.45.0"},
		},
		{
			name:    "gh/windows-delegates-to-winget-success",
			newAdpt: func() adapters.Adapter { return &GhAdapter{} },
			goos:    "windows",
			fakes: execFakes{
				lookPath: map[string]bool{"gh": true, "winget": true},
				cmdArgs: map[string]fakeResult{
					"gh":             {stdout: "gh version 2.45.0 (2024-05-30)"},
					"winget upgrade": {stdout: "Name  Id  Version  Available  Source\n------\ngithub-cli  gh  2.45.0  2.46.0  winget\n"},
				},
				shell: map[string]fakeResult{"winget upgrade gh": {}},
			},
			want: adapters.Result{Success: true, Before: "2.45.0", After: "2.45.0"},
		},

		// --- docker (delegated to resolving manager via PackageUpdater) ---
		{
			name:    "docker/not-installed-error",
			newAdpt: func() adapters.Adapter { return &DockerAdapter{} },
			fakes:   execFakes{lookPath: map[string]bool{"docker": false}},
			wantErr: true,
		},
		{
			name:    "docker/linux-dry-run-delegates-to-apt",
			newAdpt: func() adapters.Adapter { return &DockerAdapter{} },
			goos:    "linux",
			fakes: execFakes{
				lookPath: map[string]bool{"docker": true, "apt": true},
				cmdArgs:  map[string]fakeResult{"docker": {stdout: "Docker version 26.1.4, build 5650f9b"}},
				shell: map[string]fakeResult{
					aptInstalledCmd: {stdout: "2.4.0"},
					"sudo apt install --only-upgrade docker-ce": failIfRun,
				},
			},
			dryRun: true,
			want:   adapters.Result{Success: true},
		},
		{
			name:    "docker/linux-update-delegates-to-apt-error",
			newAdpt: func() adapters.Adapter { return &DockerAdapter{} },
			goos:    "linux",
			fakes: execFakes{
				lookPath: map[string]bool{"docker": true, "apt": true},
				cmdArgs: map[string]fakeResult{
					"docker":                     {stdout: "Docker version 26.1.4, build 5650f9b"},
					"apt-cache policy docker-ce": {stdout: "docker-ce:\n  Installed: 26.1.4\n  Candidate: 26.1.4\n"},
				},
				shell: map[string]fakeResult{
					"sudo apt install --only-upgrade docker-ce": {err: errors.New("apt: lock held")},
				},
			},
			want:      adapters.Result{Success: false, Before: "26.1.4", After: "26.1.4", Privileges: sudo},
			resultErr: true,
		},
		{
			name:    "docker/linux-update-delegates-to-apt-stderr",
			newAdpt: func() adapters.Adapter { return &DockerAdapter{} },
			goos:    "linux",
			fakes: execFakes{
				lookPath: map[string]bool{"docker": true, "apt": true},
				cmdArgs: map[string]fakeResult{
					"docker":                     {stdout: "Docker version 26.1.4, build 5650f9b"},
					"apt-cache policy docker-ce": {stdout: "docker-ce:\n  Installed: 26.1.4\n  Candidate: 26.1.4\n"},
				},
				shell: map[string]fakeResult{
					"sudo apt install --only-upgrade docker-ce": {stderr: "E: Unable to acquire the dpkg frontend lock"},
				},
			},
			want:      adapters.Result{Success: false, Before: "26.1.4", After: "26.1.4", Privileges: sudo},
			resultErr: true,
		},
		{
			name:    "docker/linux-update-delegates-to-apt-success",
			newAdpt: func() adapters.Adapter { return &DockerAdapter{} },
			goos:    "linux",
			fakes: execFakes{
				lookPath: map[string]bool{"docker": true, "apt": true},
				cmdArgs: map[string]fakeResult{
					"docker":                     {stdout: "Docker version 26.1.4, build 5650f9b"},
					"apt-cache policy docker-ce": {stdout: "docker-ce:\n  Installed: 26.1.4\n  Candidate: 26.1.4\n"},
				},
				shell: map[string]fakeResult{
					"sudo apt install --only-upgrade docker-ce": {},
				},
			},
			want: adapters.Result{Success: true, Before: "26.1.4", After: "26.1.4", Privileges: sudo},
		},
		{
			name:    "docker/linux-update-delegates-to-pacman-success",
			newAdpt: func() adapters.Adapter { return &DockerAdapter{} },
			goos:    "linux",
			fakes: execFakes{
				lookPath: map[string]bool{"docker": true, "apt": false, "pacman": true},
				cmdArgs: map[string]fakeResult{
					"docker":           {stdout: "Docker version 26.1.4, build 5650f9b"},
					"pacman -Q docker": {stdout: "docker 26.1.4-1"},
				},
				shell: map[string]fakeResult{
					"sudo pacman -S --noconfirm docker": {},
				},
			},
			want: adapters.Result{Success: true, Before: "26.1.4-1", After: "26.1.4-1", Privileges: sudo},
		},
		{
			name:    "docker/macos-delegates-to-brew-success",
			newAdpt: func() adapters.Adapter { return &DockerAdapter{} },
			goos:    "darwin",
			fakes: execFakes{
				lookPath: map[string]bool{"docker": true, "brew": true},
				cmdArgs: map[string]fakeResult{
					"docker":                      {stdout: "Docker version 26.1.4, build 5650f9b"},
					"brew outdated --json docker": {stdout: `[{"name":"docker","installed_versions":["26.1.4"],"current_version":"26.1.5"}]`},
				},
				shell: map[string]fakeResult{"brew upgrade docker": {}},
			},
			want: adapters.Result{Success: true, Before: "26.1.4", After: "26.1.4"},
		},
		{
			name:    "docker/windows-delegates-to-winget-success",
			newAdpt: func() adapters.Adapter { return &DockerAdapter{} },
			goos:    "windows",
			fakes: execFakes{
				lookPath: map[string]bool{"docker": true, "winget": true},
				cmdArgs: map[string]fakeResult{
					"docker":         {stdout: "Docker version 26.1.4, build 5650f9b"},
					"winget upgrade": {stdout: "Name  Id  Version  Available  Source\n------\nDocker  Docker.Docker  26.1.4  26.1.5  winget\n"},
				},
				shell: map[string]fakeResult{"winget upgrade Docker.Docker": {}},
			},
			want: adapters.Result{Success: true, Before: "26.1.4", After: "26.1.4"},
		},

		// --- go (standalone on Linux; delegated on macOS/Windows) ---
		{
			name:    "go/not-installed-error",
			newAdpt: func() adapters.Adapter { return &GoAdapter{} },
			fakes:   execFakes{lookPath: map[string]bool{"go": false}},
			wantErr: true,
		},
		{
			name:    "go/linux-dry-run-shortcut",
			newAdpt: func() adapters.Adapter { return &GoAdapter{} },
			goos:    "linux",
			fakes: execFakes{
				lookPath: map[string]bool{"go": true},
				cmdArgs:  map[string]fakeResult{"go": {stdout: "go version go1.22.0 linux/amd64"}},
				shell:    map[string]fakeResult{goLinuxUpdateCmd: failIfRun},
			},
			dryRun: true,
			want:   adapters.Result{Success: true, Before: "1.22.0", After: "1.22.0"},
		},
		{
			name:    "go/linux-non-standard-path-error",
			newAdpt: func() adapters.Adapter { return &GoAdapter{} },
			goos:    "linux",
			fakes: execFakes{
				goBinaryPath: "/home/user/go/bin/go",
				lookPath:     map[string]bool{"go": true},
				cmdArgs:      map[string]fakeResult{"go": {stdout: "go version go1.22.0 linux/amd64"}},
			},
			want:      adapters.Result{Success: false, Before: "1.22.0", After: "1.22.0", Privileges: sudo},
			resultErr: true,
		},
		{
			name:    "go/linux-checksum-mismatch-aborts",
			newAdpt: func() adapters.Adapter { return &GoAdapter{} },
			goos:    "linux",
			fakes: execFakes{
				lookPath:      map[string]bool{"go": true},
				cmdArgs:       map[string]fakeResult{"go": {stdout: "go version go1.22.0 linux/amd64"}},
				goDownloadErr: errors.New("checksum mismatch for go1.22.1.linux-amd64.tar.gz: got 111, want 222"),
			},
			want:      adapters.Result{Success: false, Before: "1.22.0", After: "1.22.0", Privileges: sudo},
			resultErr: true,
		},
		{
			name:    "go/linux-update-command-error",
			newAdpt: func() adapters.Adapter { return &GoAdapter{} },
			goos:    "linux",
			fakes: execFakes{
				lookPath: map[string]bool{"go": true},
				cmdArgs: map[string]fakeResult{
					"go":                           {stdout: "go version go1.22.0 linux/amd64"},
					"sudo mv staged /usr/local/go": {err: errors.New("mv: permission denied")},
				},
			},
			want:      adapters.Result{Success: false, Before: "1.22.0", After: "1.22.0", Privileges: sudo},
			resultErr: true,
		},
		{
			name:    "go/linux-update-stderr-error-marker",
			newAdpt: func() adapters.Adapter { return &GoAdapter{} },
			goos:    "linux",
			fakes: execFakes{
				lookPath: map[string]bool{"go": true},
				cmdArgs: map[string]fakeResult{
					"go":                           {stdout: "go version go1.22.0 linux/amd64"},
					"sudo mv staged /usr/local/go": {stderr: "error: cannot write to /usr/local/go"},
				},
			},
			want:      adapters.Result{Success: false, Before: "1.22.0", After: "1.22.0", Privileges: sudo},
			resultErr: true,
		},
		{
			name:    "go/linux-swap-failure-rollback",
			newAdpt: func() adapters.Adapter { return &GoAdapter{} },
			goos:    "linux",
			fakes: execFakes{
				lookPath: map[string]bool{"go": true},
				cmdArgs: map[string]fakeResult{
					"go":                           {stdout: "go version go1.22.0 linux/amd64"},
					"sudo mv staged /usr/local/go": {err: errors.New("swap failed")},
				},
			},
			want:      adapters.Result{Success: false, Before: "1.22.0", After: "1.22.0", Privileges: sudo},
			resultErr: true,
		},
		{
			name:    "go/linux-update-standalone-success",
			newAdpt: func() adapters.Adapter { return &GoAdapter{} },
			goos:    "linux",
			fakes: execFakes{
				lookPath: map[string]bool{"go": true},
				cmdArgs:  map[string]fakeResult{"go": {stdout: "go version go1.22.0 linux/amd64"}},
			},
			want: adapters.Result{Success: true, Before: "1.22.0", After: "1.22.0", Privileges: sudo},
		},
		{
			name:    "go/linux-apt-delegation-success",
			newAdpt: func() adapters.Adapter { return &GoAdapter{} },
			goos:    "linux",
			fakes: execFakes{
				goBinaryPath: "/usr/bin/go",
				lookPath:     map[string]bool{"go": true, "apt": true},
				cmdArgs:      map[string]fakeResult{"go": {stdout: "go version go1.22.0 linux/amd64"}},
				shell:        map[string]fakeResult{"sudo apt install --only-upgrade golang-go": {}},
			},
			want: adapters.Result{Success: true, Before: "unknown", After: "unknown", Privileges: sudo},
		},
		{
			name:    "go/linux-pacman-delegation-success",
			newAdpt: func() adapters.Adapter { return &GoAdapter{} },
			goos:    "linux",
			fakes: execFakes{
				goBinaryPath: "/usr/bin/go",
				lookPath:     map[string]bool{"go": true, "pacman": true},
				cmdArgs:      map[string]fakeResult{"go": {stdout: "go version go1.22.0 linux/amd64"}},
				shell:        map[string]fakeResult{"sudo pacman -S --noconfirm go": {}},
			},
			want: adapters.Result{Success: true, Before: "unknown", After: "unknown", Privileges: sudo},
		},
		{
			name:    "go/macos-delegates-to-brew-success",
			newAdpt: func() adapters.Adapter { return &GoAdapter{} },
			goos:    "darwin",
			fakes: execFakes{
				lookPath: map[string]bool{"go": true, "brew": true},
				cmdArgs: map[string]fakeResult{
					"go":                          {stdout: "go version go1.22.0 linux/amd64"},
					"brew outdated --json golang": {stdout: `[{"name":"golang","installed_versions":["1.22.0"],"current_version":"1.22.1"}]`},
				},
				shell: map[string]fakeResult{"brew upgrade golang": {}},
			},
			want: adapters.Result{Success: true, Before: "1.22.0", After: "1.22.0"},
		},
		{
			name:    "go/windows-delegates-to-winget-success",
			newAdpt: func() adapters.Adapter { return &GoAdapter{} },
			goos:    "windows",
			fakes: execFakes{
				lookPath: map[string]bool{"go": true, "winget": true},
				cmdArgs: map[string]fakeResult{
					"go":             {stdout: "go version go1.22.0 linux/amd64"},
					"winget upgrade": {stdout: "Name  Id  Version  Available  Source\n------\nGo  GoLang.Go  1.22.0  1.22.1  winget\n"},
				},
				shell: map[string]fakeResult{"winget upgrade GoLang.Go": {}},
			},
			want: adapters.Result{Success: true, Before: "1.22.0", After: "1.22.0"},
		},

		// --- opencode (curl installer) ---
		{
			name:    "opencode/not-installed-error",
			newAdpt: func() adapters.Adapter { return &OpenCodeAdapter{} },
			fakes:   execFakes{lookPath: map[string]bool{"opencode": false}},
			wantErr: true,
		},
		{
			name:    "opencode/dry-run-shortcut",
			newAdpt: func() adapters.Adapter { return &OpenCodeAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"opencode": true},
				cmdArgs:  map[string]fakeResult{"opencode": {stdout: "opencode v0.3.6"}},
			},
			dryRun: true,
			want:   adapters.Result{Success: true, Before: "v0.3.6", After: "v0.3.6"},
		},
		{
			name:    "opencode/update-command-error",
			newAdpt: func() adapters.Adapter { return &OpenCodeAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"opencode": true},
				cmdArgs:  map[string]fakeResult{"opencode": {stdout: "opencode v0.3.6"}},
				shell:    map[string]fakeResult{opencodeUpdateCmd: {err: errors.New("curl: could not resolve host")}},
			},
			want:      adapters.Result{Success: false, Before: "v0.3.6", After: "v0.3.6"},
			resultErr: true,
		},
		{
			name:    "opencode/stderr-error-marker",
			newAdpt: func() adapters.Adapter { return &OpenCodeAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"opencode": true},
				cmdArgs:  map[string]fakeResult{"opencode": {stdout: "opencode v0.3.6"}},
				shell:    map[string]fakeResult{opencodeUpdateCmd: {stderr: "Error: could not find supported platform"}},
			},
			want:      adapters.Result{Success: false, Before: "v0.3.6", After: "v0.3.6"},
			resultErr: true,
		},
		{
			name:    "opencode/success",
			newAdpt: func() adapters.Adapter { return &OpenCodeAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"opencode": true},
				cmdArgs:  map[string]fakeResult{"opencode": {stdout: "opencode v0.3.6"}},
				shell:    map[string]fakeResult{opencodeUpdateCmd: {}},
			},
			want: adapters.Result{Success: true, Before: "v0.3.6", After: "v0.3.6"},
		},

		// --- winget (self-only: real versions + `winget upgrade winget`) ---
		// versions come from `winget --version` (faked per row); the update
		// command is keyed `winget upgrade winget` (WU2 re-key).
		{
			name:    "winget/not-installed-error",
			newAdpt: func() adapters.Adapter { return &WingetAdapter{} },
			fakes:   execFakes{lookPath: map[string]bool{"winget": false}},
			wantErr: true,
		},
		{
			name:    "winget/dry-run-shortcut",
			newAdpt: func() adapters.Adapter { return &WingetAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"winget": true},
				cmdArgs:  map[string]fakeResult{"winget --version": {stdout: "v1.8.2301"}},
				shell:    map[string]fakeResult{wingetUpdateCmd: failIfRun},
			},
			dryRun: true,
			want:   adapters.Result{Success: true, Before: "v1.8.2301", After: "v1.8.2301"},
		},
		{
			name:    "winget/update-command-error",
			newAdpt: func() adapters.Adapter { return &WingetAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"winget": true},
				cmdArgs:  map[string]fakeResult{"winget --version": {stdout: "v1.8.2301"}},
				shell:    map[string]fakeResult{wingetUpdateCmd: {err: errors.New("winget: no installed package found")}},
			},
			want:      adapters.Result{Success: false, Before: "v1.8.2301", After: "v1.8.2301"},
			resultErr: true,
		},
		{
			name:    "winget/stderr-error-marker",
			newAdpt: func() adapters.Adapter { return &WingetAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"winget": true},
				cmdArgs:  map[string]fakeResult{"winget --version": {stdout: "v1.8.2301"}},
				shell:    map[string]fakeResult{wingetUpdateCmd: {stderr: "Error: source is not valid"}},
			},
			want:      adapters.Result{Success: false, Before: "v1.8.2301", After: "v1.8.2301"},
			resultErr: true,
		},
		{
			name:    "winget/success",
			newAdpt: func() adapters.Adapter { return &WingetAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"winget": true},
				cmdArgs:  map[string]fakeResult{"winget --version": {stdout: "v1.8.2301"}},
				shell:    map[string]fakeResult{wingetUpdateCmd: {}},
			},
			want: adapters.Result{Success: true, Before: "v1.8.2301", After: "v1.8.2301"},
		},

		// --- scoop (unknown versions + update) ---
		{
			name:    "scoop/not-installed-error",
			newAdpt: func() adapters.Adapter { return &ScoopAdapter{} },
			fakes:   execFakes{lookPath: map[string]bool{"scoop": false}},
			wantErr: true,
		},
		{
			name:    "scoop/dry-run-shortcut",
			newAdpt: func() adapters.Adapter { return &ScoopAdapter{} },
			fakes:   execFakes{lookPath: map[string]bool{"scoop": true}},
			dryRun:  true,
			want:    adapters.Result{Success: true, Before: "unknown", After: "unknown"},
		},
		{
			name:    "scoop/update-command-error",
			newAdpt: func() adapters.Adapter { return &ScoopAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"scoop": true},
				shell:    map[string]fakeResult{scoopUpdateCmd: {err: errors.New("scoop: git not found")}},
			},
			want:      adapters.Result{Success: false, Before: "unknown", After: "unknown"},
			resultErr: true,
		},
		{
			name:    "scoop/stderr-error-marker",
			newAdpt: func() adapters.Adapter { return &ScoopAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"scoop": true},
				shell:    map[string]fakeResult{scoopUpdateCmd: {stderr: "ERROR 'ffmpeg' isn't installed correctly"}},
			},
			want:      adapters.Result{Success: false, Before: "unknown", After: "unknown"},
			resultErr: true,
		},
		{
			name:    "scoop/success",
			newAdpt: func() adapters.Adapter { return &ScoopAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"scoop": true},
				shell:    map[string]fakeResult{scoopUpdateCmd: {}},
			},
			want: adapters.Result{Success: true, Before: "unknown", After: "unknown"},
		},

		// --- nvm (env-driven detect + shell version) ---
		{
			name:    "nvm/not-installed-error",
			newAdpt: func() adapters.Adapter { return &NVMAdapter{} },
			setup:   nvmMissingSetup,
			fakes:   execFakes{},
			wantErr: true,
		},
		{
			name:    "nvm/dry-run-shortcut",
			newAdpt: func() adapters.Adapter { return &NVMAdapter{} },
			setup:   nvmInstalledSetup,
			fakes: execFakes{
				shell: map[string]fakeResult{
					nvmCurrentCmd:    {stdout: "v20.11.0"},
					nvmInstallLtsCmd: failIfRun,
				},
			},
			dryRun: true,
			want:   adapters.Result{Success: true, Before: "v20.11.0", After: "v20.11.0"},
		},
		{
			name:    "nvm/update-command-error",
			newAdpt: func() adapters.Adapter { return &NVMAdapter{} },
			setup:   nvmInstalledSetup,
			fakes: execFakes{
				shell: map[string]fakeResult{
					nvmCurrentCmd:    {stdout: "v20.11.0"},
					nvmInstallLtsCmd: {err: errors.New("nvm: curl failed")},
				},
			},
			want:      adapters.Result{Success: false, Before: "v20.11.0", After: "v20.11.0"},
			resultErr: true,
		},
		{
			name:    "nvm/stderr-error-marker",
			newAdpt: func() adapters.Adapter { return &NVMAdapter{} },
			setup:   nvmInstalledSetup,
			fakes: execFakes{
				shell: map[string]fakeResult{
					nvmCurrentCmd:    {stdout: "v20.11.0"},
					nvmInstallLtsCmd: {stderr: "error: version not found"}},
			},
			want:      adapters.Result{Success: false, Before: "v20.11.0", After: "v20.11.0"},
			resultErr: true,
		},
		{
			name:    "nvm/success",
			newAdpt: func() adapters.Adapter { return &NVMAdapter{} },
			setup:   nvmInstalledSetup,
			fakes: execFakes{
				shell: map[string]fakeResult{
					nvmCurrentCmd:    {stdout: "v20.11.0"},
					nvmInstallLtsCmd: {},
				},
			},
			want: adapters.Result{Success: true, Before: "v20.11.0", After: "v20.11.0"},
		},

		// --- uv (dual-scope: self update + tool upgrade) ---
		{
			name:    "uv/not-installed-error",
			newAdpt: func() adapters.Adapter { return &UvAdapter{} },
			fakes:   execFakes{lookPath: map[string]bool{"uv": false}},
			wantErr: true,
		},
		{
			name:    "uv/dry-run",
			newAdpt: func() adapters.Adapter { return &UvAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"uv": true},
				cmdArgs: map[string]fakeResult{
					"uv":           {stdout: "uv 0.12.10"},
					"uv --version": {stdout: "uv 0.12.10"},
				},
				shell: map[string]fakeResult{
					uvSelfUpdateCmd:  failIfRun,
					uvToolUpgradeCmd: failIfRun,
				},
			},
			dryRun: true,
			want:   adapters.Result{Success: true, Before: "0.12.10", After: "0.12.10"},
		},
		{
			name:    "uv/dry-run-shortcut",
			newAdpt: func() adapters.Adapter { return &UvAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"uv": true},
				cmdArgs: map[string]fakeResult{
					"uv":           {stdout: "uv 0.12.10"},
					"uv --version": {stdout: "uv 0.12.10"},
				},
				shell: map[string]fakeResult{
					uvSelfUpdateCmd:  failIfRun,
					uvToolUpgradeCmd: failIfRun,
				},
			},
			dryRun: true,
			want:   adapters.Result{Success: true, Before: "0.12.10", After: "0.12.10"},
		},
		{
			name:    "uv/live-standalone-success",
			newAdpt: func() adapters.Adapter { return &UvAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"uv": true},
				cmdArgs: map[string]fakeResult{
					"uv":           {stdout: "uv 0.12.10"},
					"uv --version": {stdout: "uv 0.12.10"},
				},
				shell: map[string]fakeResult{
					uvSelfUpdateCmd:  {},
					uvToolUpgradeCmd: {},
				},
			},
			want: adapters.Result{Success: true, Before: "0.12.10", After: "0.12.10"},
		},
		{
			name:    "uv/external-manager-bypass-success",
			newAdpt: func() adapters.Adapter { return &UvAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"uv": true},
				cmdArgs: map[string]fakeResult{
					"uv":           {stdout: "uv 0.12.10"},
					"uv --version": {stdout: "uv 0.12.10"},
				},
				shell: map[string]fakeResult{
					uvSelfUpdateCmd:  {stderr: "error: uv was installed through an external package manager", err: errors.New("exit status 2")},
					uvToolUpgradeCmd: {},
				},
			},
			want: adapters.Result{Success: true, Before: "0.12.10", After: "0.12.10"},
		},
		{
			name:    "uv/unexpected-self-update-error",
			newAdpt: func() adapters.Adapter { return &UvAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"uv": true},
				cmdArgs: map[string]fakeResult{
					"uv":           {stdout: "uv 0.12.10"},
					"uv --version": {stdout: "uv 0.12.10"},
				},
				shell: map[string]fakeResult{
					uvSelfUpdateCmd:  {stderr: "error: network failure", err: errors.New("exit status 1")},
					uvToolUpgradeCmd: failIfRun,
				},
			},
			want:      adapters.Result{Success: false, Before: "0.12.10", After: "0.12.10"},
			resultErr: true,
		},
		{
			name:    "uv/self-update-unexpected-failure",
			newAdpt: func() adapters.Adapter { return &UvAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"uv": true},
				cmdArgs: map[string]fakeResult{
					"uv":           {stdout: "uv 0.12.10"},
					"uv --version": {stdout: "uv 0.12.10"},
				},
				shell: map[string]fakeResult{
					uvSelfUpdateCmd:  {stderr: "error: network failure", err: errors.New("exit status 1")},
					uvToolUpgradeCmd: failIfRun,
				},
			},
			want:      adapters.Result{Success: false, Before: "0.12.10", After: "0.12.10"},
			resultErr: true,
		},
		{
			name:    "uv/tool-upgrade-error",
			newAdpt: func() adapters.Adapter { return &UvAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"uv": true},
				cmdArgs: map[string]fakeResult{
					"uv":           {stdout: "uv 0.12.10"},
					"uv --version": {stdout: "uv 0.12.10"},
				},
				shell: map[string]fakeResult{
					uvSelfUpdateCmd:  {},
					uvToolUpgradeCmd: {stderr: "error: failed to upgrade tool", err: errors.New("exit status 1")},
				},
			},
			want:      adapters.Result{Success: false, Before: "0.12.10", After: "0.12.10"},
			resultErr: true,
		},
		{
			name:    "uv/tool-upgrade-failure",
			newAdpt: func() adapters.Adapter { return &UvAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"uv": true},
				cmdArgs: map[string]fakeResult{
					"uv":           {stdout: "uv 0.12.10"},
					"uv --version": {stdout: "uv 0.12.10"},
				},
				shell: map[string]fakeResult{
					uvSelfUpdateCmd:  {},
					uvToolUpgradeCmd: {stderr: "error: failed to upgrade tool", err: errors.New("exit status 1")},
				},
			},
			want:      adapters.Result{Success: false, Before: "0.12.10", After: "0.12.10"},
			resultErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.goos != "" && tt.goos != runtime.GOOS {
				t.Skipf("row is for %s, running on %s", tt.goos, runtime.GOOS)
			}
			if tt.setup != nil {
				tt.setup(t)
			}
			setExecFakes(t, tt.fakes)

			got, err := tt.newAdpt().Update(context.Background(), tt.dryRun)
			if tt.wantErr {
				if err == nil {
					t.Fatal("Update() error = nil, want error when tool is not installed")
				}
				return
			}
			if err != nil {
				t.Fatalf("Update() unexpected error: %v", err)
			}
			if tt.resultErr != (got.Error != nil) {
				t.Errorf("Result.Error presence = %v, want %v (got error: %v)", got.Error != nil, tt.resultErr, got.Error)
			}
			if got.Success != tt.want.Success {
				t.Errorf("Result.Success = %v, want %v", got.Success, tt.want.Success)
			}
			if got.Before != tt.want.Before {
				t.Errorf("Result.Before = %q, want %q", got.Before, tt.want.Before)
			}
			if got.After != tt.want.After {
				t.Errorf("Result.After = %q, want %q", got.After, tt.want.After)
			}
			if !equalPrivileges(got.Privileges, tt.want.Privileges) {
				t.Errorf("Result.Privileges = %v, want %v", got.Privileges, tt.want.Privileges)
			}
		})
	}
}

// --- WU2: delegated update + ownership (spec Official Adapter Catalog /
// Resolved Owner Update Delegation) ---
//
// An owned tool (gh, docker, go) MUST delegate its Update() to its resolving
// manager rather than run its own hardcoded manager command. These cases pin
// that behavior hermetically: the owned tool's own update command key is
// replaced by the MANAGER's update command key, proving the tool never runs
// its own command on the delegated path. docker on Linux is the GOTCHA case
// (runtime.GOOS "linux" -> platform "linux"); go is the standalone case on
// Linux (nil owner -> own command still runs).

// TestUpdateDelegation proves the owned-tool Update() delegation contract:
// gh/docker delegate to their resolving manager (apt/brew/winget), and go on
// Linux has no resolving owner so its own (manual binary replace) command
// still runs. Each row drives the runtime.GOOS that the production adapter
// sees so the platform.NormalizeOS translation is exercised for real.
func TestUpdateDelegation(t *testing.T) {
	sudo := []string{"sudo"}

	tests := []updateCase{
		// gh on Linux owned by apt (Gated): delegated apt.UpdatePackage("gh") runs, so
		// the result carries gh's package versions + sudo. gh's own
		// "sudo apt install --only-upgrade gh" command is executed.
		{
			name:    "gh/linux-delegates-to-apt",
			newAdpt: func() adapters.Adapter { return &GhAdapter{} },
			goos:    "linux",
			fakes: execFakes{
				lookPath: map[string]bool{"gh": true, "apt": true},
				cmdArgs: map[string]fakeResult{
					"gh":                  {stdout: "gh version 2.45.0 (2024-05-30)"},
					"apt-cache policy gh": {stdout: "gh:\n  Installed: 2.45.0\n  Candidate: 2.46.0\n"},
				},
				shell: map[string]fakeResult{
					"sudo apt install --only-upgrade gh": {},
				},
			},
			want: adapters.Result{Success: true, Before: "2.45.0", After: "2.45.0", Privileges: sudo},
		},
		// gh on macOS owned by brew (AlwaysUpdate): delegated brew.UpdatePackage("gh")
		// runs, so the result carries gh's package versions.
		{
			name:    "gh/macos-delegates-to-brew",
			newAdpt: func() adapters.Adapter { return &GhAdapter{} },
			goos:    "darwin",
			fakes: execFakes{
				lookPath: map[string]bool{"gh": true, "brew": true},
				cmdArgs: map[string]fakeResult{
					"gh":                      {stdout: "gh version 2.45.0 (2024-05-30)"},
					"brew outdated --json gh": {stdout: `[{"name":"gh","installed_versions":["2.45.0"],"current_version":"2.46.0"}]`},
				},
				shell: map[string]fakeResult{"brew upgrade gh": {}},
			},
			want: adapters.Result{Success: true, Before: "2.45.0", After: "2.45.0"},
		},
		// docker on Linux owned by apt (Gated): delegated apt.UpdatePackage("docker-ce") runs,
		// carrying docker-ce's package versions + sudo.
		{
			name:    "docker/linux-delegates-to-apt",
			newAdpt: func() adapters.Adapter { return &DockerAdapter{} },
			goos:    "linux",
			fakes: execFakes{
				lookPath: map[string]bool{"docker": true, "apt": true},
				cmdArgs: map[string]fakeResult{
					"docker":                     {stdout: "Docker version 26.1.4, build 5650f9b"},
					"apt-cache policy docker-ce": {stdout: "docker-ce:\n  Installed: 26.1.4\n  Candidate: 26.1.4\n"},
				},
				shell: map[string]fakeResult{
					"sudo apt install --only-upgrade docker-ce": {},
				},
			},
			want: adapters.Result{Success: true, Before: "26.1.4", After: "26.1.4", Privileges: sudo},
		},
		// go on macOS owned by brew (AlwaysUpdate): delegated brew.UpdatePackage("golang")
		// runs, carrying golang's package versions.
		{
			name:    "go/macos-delegates-to-brew",
			newAdpt: func() adapters.Adapter { return &GoAdapter{} },
			goos:    "darwin",
			fakes: execFakes{
				lookPath: map[string]bool{"go": true, "brew": true},
				cmdArgs: map[string]fakeResult{
					"go":                          {stdout: "go version go1.22.0 linux/amd64"},
					"brew outdated --json golang": {stdout: `[{"name":"golang","installed_versions":["1.22.0"],"current_version":"1.22.1"}]`},
				},
				shell: map[string]fakeResult{"brew upgrade golang": {}},
			},
			want: adapters.Result{Success: true, Before: "1.22.0", After: "1.22.0"},
		},
		// go on Linux is standalone (no resolving owner) — its own manual
		// binary-replace command must still run.
		{
			name:    "go/linux-standalone-keeps-own-cmd",
			newAdpt: func() adapters.Adapter { return &GoAdapter{} },
			goos:    "linux",
			fakes: execFakes{
				lookPath: map[string]bool{"go": true},
				cmdArgs:  map[string]fakeResult{"go": {stdout: "go version go1.22.0 linux/amd64"}},
			},
			want: adapters.Result{Success: true, Before: "1.22.0", After: "1.22.0", Privileges: sudo},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.goos != "" && tt.goos != runtime.GOOS {
				t.Skipf("row is for %s, running on %s", tt.goos, runtime.GOOS)
			}
			if tt.setup != nil {
				tt.setup(t)
			}
			setExecFakes(t, tt.fakes)

			got, err := tt.newAdpt().Update(context.Background(), tt.dryRun)
			if err != nil {
				t.Fatalf("Update() unexpected error: %v", err)
			}
			if got.Success != tt.want.Success {
				t.Errorf("Result.Success = %v, want %v", got.Success, tt.want.Success)
			}
			if got.Before != tt.want.Before {
				t.Errorf("Result.Before = %q, want %q", got.Before, tt.want.Before)
			}
			if got.After != tt.want.After {
				t.Errorf("Result.After = %q, want %q", got.After, tt.want.After)
			}
			if !equalPrivileges(got.Privileges, tt.want.Privileges) {
				t.Errorf("Result.Privileges = %v, want %v", got.Privileges, tt.want.Privileges)
			}
		})
	}
}

// TestUpdatePackage covers the WU3 manager-group bulk privileged executor
// (design D3): apt/brew/winget each run the per-package update COMMAND for one
// owned package (e.g. `sudo apt install --only-upgrade gh`, `brew upgrade gh`,
// `winget upgrade gh`). This is NOT the manager's self-only Update() — it
// upgrades the owned package. Every row is hermetic (setExecFakes, no real
// subprocess). The before/after version is the PACKAGE's authentic version
// reported via CheckPackage.
func TestUpdatePackage(t *testing.T) {
	sudo := []string{"sudo"}

	tests := []updateCase{
		{
			name:    "apt/gh-updates-owned-package",
			newAdpt: func() adapters.Adapter { return &AptAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"apt": true},
				cmdArgs: map[string]fakeResult{
					"apt-cache policy gh": {stdout: "gh:\n  Installed: 2.45.0\n  Candidate: 2.46.0\n"},
				},
				shell: map[string]fakeResult{
					"sudo apt install --only-upgrade gh": {},
				},
			},
			want: adapters.Result{Success: true, Before: "2.45.0", After: "2.45.0", Privileges: sudo},
		},
		{
			name:    "apt/command-fails-structurally",
			newAdpt: func() adapters.Adapter { return &AptAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"apt": true},
				cmdArgs: map[string]fakeResult{
					"apt-cache policy gh": {stdout: "gh:\n  Installed: 2.45.0\n  Candidate: 2.46.0\n"},
				},
				shell: map[string]fakeResult{
					"sudo apt install --only-upgrade gh": {err: errors.New("sudo: command not found")},
				},
			},
			want: adapters.Result{Success: false, Before: "2.45.0", After: "2.45.0", Privileges: sudo},
		},
		{
			name:    "apt/stderr-marker-fails",
			newAdpt: func() adapters.Adapter { return &AptAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"apt": true},
				cmdArgs: map[string]fakeResult{
					"apt-cache policy gh": {stdout: "gh:\n  Installed: 2.45.0\n  Candidate: 2.46.0\n"},
				},
				shell: map[string]fakeResult{
					"sudo apt install --only-upgrade gh": {stderr: "E: Unable to locate package gh"},
				},
			},
			want: adapters.Result{Success: false, Before: "2.45.0", After: "2.45.0", Privileges: sudo},
		},
		{
			name:    "apt/check-package-error-fallback",
			newAdpt: func() adapters.Adapter { return &AptAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"apt": true},
				cmdArgs: map[string]fakeResult{
					"apt-cache policy gh": {err: errors.New("apt-cache failed")},
				},
				shell: map[string]fakeResult{
					"sudo apt install --only-upgrade gh": {},
				},
			},
			want: adapters.Result{Success: true, Before: "unknown", After: "unknown", Privileges: sudo},
		},
		{
			name:    "apt/not-installed-error",
			newAdpt: func() adapters.Adapter { return &AptAdapter{} },
			fakes:   execFakes{lookPath: map[string]bool{"apt": false}},
			wantErr: true,
		},
		{
			name:    "brew/gh-updates-owned-formula",
			newAdpt: func() adapters.Adapter { return &BrewAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"brew": true},
				cmdArgs: map[string]fakeResult{
					"brew outdated --json gh": {stdout: `[{"name":"gh","installed_versions":["2.45.0"],"current_version":"2.46.0"}]`},
				},
				shell: map[string]fakeResult{"brew upgrade gh": {}},
			},
			want: adapters.Result{Success: true, Before: "2.45.0", After: "2.45.0"},
		},
		{
			name:    "brew/command-fails",
			newAdpt: func() adapters.Adapter { return &BrewAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"brew": true},
				cmdArgs: map[string]fakeResult{
					"brew outdated --json gh": {stdout: `[{"name":"gh","installed_versions":["2.45.0"],"current_version":"2.46.0"}]`},
				},
				shell: map[string]fakeResult{"brew upgrade gh": {err: errors.New("brew: network error")}},
			},
			want: adapters.Result{Success: false, Before: "2.45.0", After: "2.45.0"},
		},
		{
			name:    "winget/gh-updates-owned-package",
			newAdpt: func() adapters.Adapter { return &WingetAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"winget": true},
				cmdArgs: map[string]fakeResult{
					"winget upgrade": {stdout: "Name  Id  Version  Available  Source\n------\ngithub-cli  gh  2.45.0  2.46.0  winget\n"},
				},
				shell: map[string]fakeResult{"winget upgrade gh": {}},
			},
			want: adapters.Result{Success: true, Before: "2.45.0", After: "2.45.0"},
		},
		{
			name:    "winget/command-fails",
			newAdpt: func() adapters.Adapter { return &WingetAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"winget": true},
				cmdArgs: map[string]fakeResult{
					"winget upgrade": {stdout: "Name  Id  Version  Available  Source\n------\ngithub-cli  gh  2.45.0  2.46.0  winget\n"},
				},
				shell: map[string]fakeResult{"winget upgrade gh": {err: errors.New("winget: package not found")}},
			},
			want: adapters.Result{Success: false, Before: "2.45.0", After: "2.45.0"},
		},

		// --- pacman: sudo pacman -S --noconfirm <pkg> ---
		{
			name:    "pacman/ripgrep-updates-owned-package",
			pkg:     "ripgrep",
			newAdpt: func() adapters.Adapter { return &PacmanAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"pacman": true, "vercmp": true},
				cmdArgs: map[string]fakeResult{
					"pacman -Q ripgrep":        {stdout: "ripgrep 14.1.0-1\n"},
					"pacman -Si ripgrep":       {stdout: "Repository : extra\nName : ripgrep\nVersion : 14.1.2-1\n"},
					"vercmp 14.1.2-1 14.1.0-1": {stdout: "1"},
				},
				shell: map[string]fakeResult{
					"sudo pacman -S --noconfirm ripgrep": {},
				},
			},
			want: adapters.Result{Success: true, Before: "14.1.0-1", After: "14.1.0-1", Privileges: sudo},
		},
		{
			name:    "pacman/package-command-fails",
			pkg:     "ripgrep",
			newAdpt: func() adapters.Adapter { return &PacmanAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"pacman": true, "vercmp": true},
				cmdArgs: map[string]fakeResult{
					"pacman -Q ripgrep":  {stdout: "ripgrep 14.1.0-1\n"},
					"pacman -Si ripgrep": {stdout: "Repository : extra\nName : ripgrep\nVersion : 14.1.2-1\n"},
				},
				shell: map[string]fakeResult{
					"sudo pacman -S --noconfirm ripgrep": {err: errors.New("exit status 1")},
				},
			},
			want: adapters.Result{Success: false, Before: "14.1.0-1", After: "14.1.0-1", Privileges: sudo},
		},
		{
			name:    "pacman/stderr-marker-fails",
			pkg:     "badpkg",
			newAdpt: func() adapters.Adapter { return &PacmanAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"pacman": true},
				shell: map[string]fakeResult{
					"sudo pacman -S --noconfirm badpkg": {stderr: "error: target not found: badpkg"},
				},
			},
			want: adapters.Result{Success: false, Before: "unknown", After: "unknown", Privileges: sudo},
		},
		{
			name:    "pacman/not-installed-error",
			pkg:     "ripgrep",
			newAdpt: func() adapters.Adapter { return &PacmanAdapter{} },
			fakes:   execFakes{lookPath: map[string]bool{"pacman": false}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setExecFakes(t, tt.fakes)

			updater, ok := tt.newAdpt().(adapters.PackageUpdater)
			if !ok {
				t.Fatalf("adapter %T does not implement PackageUpdater", tt.newAdpt())
			}
			pkg := tt.pkg
			if pkg == "" {
				pkg = "gh"
			}
			res, err := updater.UpdatePackage(context.Background(), pkg)
			if tt.wantErr {
				if err == nil {
					t.Fatal("UpdatePackage() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("UpdatePackage() unexpected error: %v", err)
			}
			if res.Success != tt.want.Success {
				t.Errorf("Result.Success = %v, want %v", res.Success, tt.want.Success)
			}
			if res.Before != tt.want.Before {
				t.Errorf("Result.Before = %q, want %q", res.Before, tt.want.Before)
			}
			if res.After != tt.want.After {
				t.Errorf("Result.After = %q, want %q", res.After, tt.want.After)
			}
			if !equalPrivileges(res.Privileges, tt.want.Privileges) {
				t.Errorf("Result.Privileges = %v, want %v", res.Privileges, tt.want.Privileges)
			}
		})
	}
}

func TestUpdatePackage_AuthenticVersionProgression(t *testing.T) {
	t.Run("apt/version-advances", func(t *testing.T) {
		callCount := 0
		setExecFakes(t, execFakes{
			lookPath: map[string]bool{"apt": true},
			shell: map[string]fakeResult{
				"sudo apt install --only-upgrade gh": {},
			},
		})
		origRunCmdArgs := runCmdArgsFn
		runCmdArgsFn = func(ctx context.Context, name string, args ...string) (string, string, error) {
			if name == "apt-cache" && len(args) == 2 && args[0] == "policy" && args[1] == "gh" {
				callCount++
				if callCount == 1 {
					return "gh:\n  Installed: 2.45.0\n  Candidate: 2.46.0\n", "", nil
				}
				return "gh:\n  Installed: 2.46.0\n  Candidate: 2.46.0\n", "", nil
			}
			return origRunCmdArgs(ctx, name, args...)
		}

		res, err := (&AptAdapter{}).UpdatePackage(context.Background(), "gh")
		if err != nil {
			t.Fatalf("UpdatePackage unexpected error: %v", err)
		}
		if !res.Success {
			t.Errorf("res.Success = false, want true")
		}
		if res.Before != "2.45.0" {
			t.Errorf("res.Before = %q, want 2.45.0", res.Before)
		}
		if res.After != "2.46.0" {
			t.Errorf("res.After = %q, want 2.46.0", res.After)
		}
	})

	t.Run("pacman/version-advances", func(t *testing.T) {
		callCount := 0
		setExecFakes(t, execFakes{
			lookPath: map[string]bool{"pacman": true, "vercmp": true},
			shell: map[string]fakeResult{
				"sudo pacman -S --noconfirm ripgrep": {},
			},
		})
		origRunCmdArgs := runCmdArgsFn
		runCmdArgsFn = func(ctx context.Context, name string, args ...string) (string, string, error) {
			if name == "pacman" && len(args) == 2 && args[0] == "-Q" && args[1] == "ripgrep" {
				callCount++
				if callCount == 1 {
					return "ripgrep 14.1.0-1\n", "", nil
				}
				return "ripgrep 14.1.2-1\n", "", nil
			}
			if name == "pacman" && len(args) == 2 && args[0] == "-Si" && args[1] == "ripgrep" {
				return "Repository : extra\nName : ripgrep\nVersion : 14.1.2-1\n", "", nil
			}
			return origRunCmdArgs(ctx, name, args...)
		}

		res, err := (&PacmanAdapter{}).UpdatePackage(context.Background(), "ripgrep")
		if err != nil {
			t.Fatalf("UpdatePackage unexpected error: %v", err)
		}
		if !res.Success {
			t.Errorf("res.Success = false, want true")
		}
		if res.Before != "14.1.0-1" {
			t.Errorf("res.Before = %q, want 14.1.0-1", res.Before)
		}
		if res.After != "14.1.2-1" {
			t.Errorf("res.After = %q, want 14.1.2-1", res.After)
		}
	})

	t.Run("winget/version-advances", func(t *testing.T) {
		callCount := 0
		setExecFakes(t, execFakes{
			lookPath: map[string]bool{"winget": true},
			shell: map[string]fakeResult{
				"winget upgrade gh": {},
			},
		})
		origRunCmdArgs := runCmdArgsFn
		runCmdArgsFn = func(ctx context.Context, name string, args ...string) (string, string, error) {
			if name == "winget" && len(args) == 1 && args[0] == "upgrade" {
				callCount++
				if callCount == 1 {
					return "Name  Id  Version  Available  Source\n------\ngithub-cli  gh  2.45.0  2.46.0  winget\n", "", nil
				}
				return "Name  Id  Version  Available  Source\n------\ngithub-cli  gh  2.46.0  2.46.0  winget\n", "", nil
			}
			return origRunCmdArgs(ctx, name, args...)
		}

		res, err := (&WingetAdapter{}).UpdatePackage(context.Background(), "gh")
		if err != nil {
			t.Fatalf("UpdatePackage unexpected error: %v", err)
		}
		if !res.Success {
			t.Errorf("res.Success = false, want true")
		}
		if res.Before != "2.45.0" {
			t.Errorf("res.Before = %q, want 2.45.0", res.Before)
		}
		if res.After != "2.46.0" {
			t.Errorf("res.After = %q, want 2.46.0", res.After)
		}
	})
}

// TestPnpmCorruptionRecoveryMessage verifies the recovery failure path reports
// the recovery attempt in the error, so a corrupted-store failure is
// distinguishable from a plain update failure.
func TestPnpmCorruptionRecoveryMessage(t *testing.T) {
	setExecFakes(t, execFakes{
		lookPath: map[string]bool{"pnpm": true},
		cmdArgs:  map[string]fakeResult{"pnpm": {stdout: "8.14.0"}},
		shell: map[string]fakeResult{
			pnpmUpdateCmd: {stderr: "corrupt store detected", err: errors.New("store broken")},
			pnpmPruneCmd:  {},
		},
	})

	result, err := (&PnpmAdapter{}).Update(context.Background(), false)
	if err != nil {
		t.Fatalf("Update() unexpected error: %v", err)
	}
	if result.Error == nil {
		t.Fatal("Result.Error = nil, want recovery failure error")
	}
	if !strings.Contains(result.Error.Error(), "even after recovery") {
		t.Errorf("Result.Error = %q, want it to mention the recovery attempt", result.Error)
	}
}

// TestUpdateRunsDeclaredCommand is the D2 byte-equality pin (spec tool-adapter
// "Declared equals executed"): for every manager, the command its
// Update(false)/UpdatePackage(pkg) actually executes through the runCmdFn seam
// MUST be byte-identical to the command declared on ToolInfo
// (SelfUpdateCommand, or RenderPackageCommand(PackageUpdateCommand, pkg)).
// A recorder replaces the seam for the duration of each row; the declared
// constant must appear verbatim among the captured shell commands — no other
// call site can produce that string, so presence proves the update path ran
// exactly the declaration. This is the structural guarantee that plan/execution
// divergence (the pacman synthesis bug) cannot reappear.
func TestUpdateRunsDeclaredCommand(t *testing.T) {
	tests := []struct {
		name    string
		newAdpt func() adapters.Adapter
		pkg     string // "" → self-update row; otherwise the owned package
	}{
		{name: "apt/self", newAdpt: func() adapters.Adapter { return &AptAdapter{} }},
		{name: "apt/package", newAdpt: func() adapters.Adapter { return &AptAdapter{} }, pkg: "gh"},
		{name: "brew/self", newAdpt: func() adapters.Adapter { return &BrewAdapter{} }},
		{name: "brew/package", newAdpt: func() adapters.Adapter { return &BrewAdapter{} }, pkg: "gh"},
		{name: "winget/self", newAdpt: func() adapters.Adapter { return &WingetAdapter{} }},
		{name: "winget/package", newAdpt: func() adapters.Adapter { return &WingetAdapter{} }, pkg: "gh"},
		{name: "scoop/self", newAdpt: func() adapters.Adapter { return &ScoopAdapter{} }},
		{name: "pacman/self", newAdpt: func() adapters.Adapter { return &PacmanAdapter{} }},
		{name: "pacman/package", newAdpt: func() adapters.Adapter { return &PacmanAdapter{} }, pkg: "ripgrep"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Recorder seam: capture every shell command; version lookups are
			// no-ops (adapters tolerate empty versions on the update path).
			var captured []string
			origRunCmd := runCmdFn
			origRunCmdArgs := runCmdArgsFn
			origRunCmdArgsUpdate := runCmdArgsUpdateFn
			origLookPath := lookPathFn
			runCmdFn = func(ctx context.Context, command string) (string, string, error) {
				captured = append(captured, command)
				return "", "", nil
			}
			runCmdArgsFn = func(ctx context.Context, s string, strings ...string) (string, string, error) { return "", "", nil }
			runCmdArgsUpdateFn = func(ctx context.Context, name string, args ...string) (string, string, error) {
				cmd := name
				if len(args) > 0 {
					cmd = name + " " + strings.Join(args, " ")
				}
				captured = append(captured, cmd)
				return "", "", nil
			}
			lookPathFn = func(string) bool { return true }
			t.Cleanup(func() {
				runCmdFn = origRunCmd
				runCmdArgsFn = origRunCmdArgs
				runCmdArgsUpdateFn = origRunCmdArgsUpdate
				lookPathFn = origLookPath
			})

			info := tt.newAdpt().Info()
			wantCmd := info.SelfUpdateCommand
			if tt.pkg != "" {
				wantCmd = adapters.RenderPackageCommand(info.PackageUpdateCommand, tt.pkg)
			}

			var res adapters.Result
			var err error
			if tt.pkg == "" {
				res, err = tt.newAdpt().Update(context.Background(), false)
			} else {
				updater, ok := tt.newAdpt().(adapters.PackageUpdater)
				if !ok {
					t.Fatalf("adapter %T does not implement PackageUpdater", tt.newAdpt())
				}
				res, err = updater.UpdatePackage(context.Background(), tt.pkg)
			}
			if err != nil {
				t.Fatalf("update returned unexpected error: %v", err)
			}
			if !res.Success {
				t.Fatalf("update did not succeed: %v", res.Error)
			}

			found := false
			for _, cmd := range captured {
				if cmd == wantCmd {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("executed shell commands %v; declared command %q not executed byte-exactly",
					captured, wantCmd)
			}
		})
	}
}

// TestUpdatePackage_StructuredArgumentSecurity asserts that calling UpdatePackage
// with shell metacharacters passes them as a single literal argument rather than
// evaluating them via a shell.
func TestUpdatePackage_StructuredArgumentSecurity(t *testing.T) {
	payload := "my-pkg;touch /tmp/pwn"

	tests := []struct {
		name     string
		newAdpt  func() adapters.Adapter
		wantBin  string
		wantArgs []string
	}{
		{
			name:     "apt",
			newAdpt:  func() adapters.Adapter { return &AptAdapter{} },
			wantBin:  "sudo",
			wantArgs: []string{"apt", "install", "--only-upgrade", payload},
		},
		{
			name:     "pacman",
			newAdpt:  func() adapters.Adapter { return &PacmanAdapter{} },
			wantBin:  "sudo",
			wantArgs: []string{"pacman", "-S", "--noconfirm", payload},
		},
		{
			name:     "brew",
			newAdpt:  func() adapters.Adapter { return &BrewAdapter{} },
			wantBin:  "brew",
			wantArgs: []string{"upgrade", payload},
		},
		{
			name:     "winget",
			newAdpt:  func() adapters.Adapter { return &WingetAdapter{} },
			wantBin:  "winget",
			wantArgs: []string{"upgrade", payload},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var recordedBin string
			var recordedArgs []string

			origRunCmd := runCmdFn
			origRunCmdArgs := runCmdArgsFn
			origRunCmdArgsUpdate := runCmdArgsUpdateFn
			origLookPath := lookPathFn

			runCmdFn = func(ctx context.Context, command string) (string, string, error) {
				t.Fatalf("runCmd should NOT be called; evaluated via shell: %q", command)
				return "", "", nil
			}
			runCmdArgsFn = func(ctx context.Context, name string, args ...string) (string, string, error) {
				return "[]", "", nil
			}
			runCmdArgsUpdateFn = func(ctx context.Context, name string, args ...string) (string, string, error) {
				recordedBin = name
				recordedArgs = append([]string(nil), args...)
				return "", "", nil
			}
			lookPathFn = func(string) bool { return true }

			t.Cleanup(func() {
				runCmdFn = origRunCmd
				runCmdArgsFn = origRunCmdArgs
				runCmdArgsUpdateFn = origRunCmdArgsUpdate
				lookPathFn = origLookPath
			})

			updater, ok := tt.newAdpt().(adapters.PackageUpdater)
			if !ok {
				t.Fatalf("adapter %T does not implement PackageUpdater", tt.newAdpt())
			}

			_, err := updater.UpdatePackage(context.Background(), payload)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if recordedBin != tt.wantBin {
				t.Errorf("binary = %q, want %q", recordedBin, tt.wantBin)
			}
			if len(recordedArgs) != len(tt.wantArgs) {
				t.Fatalf("args length = %d (%v), want %d (%v)", len(recordedArgs), recordedArgs, len(tt.wantArgs), tt.wantArgs)
			}
			for i := range recordedArgs {
				if recordedArgs[i] != tt.wantArgs[i] {
					t.Errorf("arg[%d] = %q, want %q", i, recordedArgs[i], tt.wantArgs[i])
				}
			}
		})
	}
}

// TestUpdate_MigratedStructuredExecution asserts that official adapters migrated
// from runCmd (npm, pnpm, scoop, bun, uv, opencode) execute their update commands
// strictly via runCmdArgsUpdate and never invoke the shell runCmdFn seam.
func TestUpdate_MigratedStructuredExecution(t *testing.T) {
	tests := []struct {
		name      string
		newAdpt   func() adapters.Adapter
		wantCalls [][]string // list of [binary, arg1, arg2...] expected to runCmdArgsUpdate
	}{
		{
			name:      "npm",
			newAdpt:   func() adapters.Adapter { return &NpmAdapter{} },
			wantCalls: [][]string{{"npm", "update", "-g"}},
		},
		{
			name:      "pnpm",
			newAdpt:   func() adapters.Adapter { return &PnpmAdapter{} },
			wantCalls: [][]string{{"pnpm", "update", "-g"}},
		},
		{
			name:      "scoop",
			newAdpt:   func() adapters.Adapter { return &ScoopAdapter{} },
			wantCalls: [][]string{{"scoop", "update", "scoop"}},
		},
		{
			name:      "bun",
			newAdpt:   func() adapters.Adapter { return &BunAdapter{} },
			wantCalls: [][]string{{"bun", "upgrade"}},
		},
		{
			name:      "uv",
			newAdpt:   func() adapters.Adapter { return &UvAdapter{} },
			wantCalls: [][]string{{"uv", "self", "update"}, {"uv", "tool", "upgrade", "--all"}},
		},
		{
			name:      "opencode",
			newAdpt:   func() adapters.Adapter { return &OpenCodeAdapter{} },
			wantCalls: [][]string{{"opencode", "update"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var recordedCalls [][]string

			origRunCmd := runCmdFn
			origRunCmdArgsUpdate := runCmdArgsUpdateFn
			origRunCmdArgs := runCmdArgsFn
			origLookPath := lookPathFn

			runCmdFn = func(ctx context.Context, command string) (string, string, error) {
				t.Fatalf("runCmd should NOT be called; evaluated via shell: %q", command)
				return "", "", nil
			}
			runCmdArgsUpdateFn = func(ctx context.Context, name string, args ...string) (string, string, error) {
				call := append([]string{name}, args...)
				recordedCalls = append(recordedCalls, call)
				return "", "", nil
			}
			runCmdArgsFn = func(ctx context.Context, name string, args ...string) (string, string, error) {
				return "", "", nil
			}
			lookPathFn = func(string) bool { return true }

			t.Cleanup(func() {
				runCmdFn = origRunCmd
				runCmdArgsUpdateFn = origRunCmdArgsUpdate
				runCmdArgsFn = origRunCmdArgs
				lookPathFn = origLookPath
			})

			_, err := tt.newAdpt().Update(context.Background(), false)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(recordedCalls) != len(tt.wantCalls) {
				t.Fatalf("recorded calls = %v, want %v", recordedCalls, tt.wantCalls)
			}
			for i := range recordedCalls {
				if len(recordedCalls[i]) != len(tt.wantCalls[i]) {
					t.Fatalf("call[%d] len = %d (%v), want %d (%v)", i, len(recordedCalls[i]), recordedCalls[i], len(tt.wantCalls[i]), tt.wantCalls[i])
				}
				for j := range recordedCalls[i] {
					if recordedCalls[i][j] != tt.wantCalls[i][j] {
						t.Errorf("call[%d][%d] = %q, want %q", i, j, recordedCalls[i][j], tt.wantCalls[i][j])
					}
				}
			}
		})
	}

	t.Run("pnpm/corruption-recovery", func(t *testing.T) {
		var recordedCalls [][]string

		origRunCmd := runCmdFn
		origRunCmdArgsUpdate := runCmdArgsUpdateFn
		origRunCmdArgs := runCmdArgsFn
		origLookPath := lookPathFn

		runCmdFn = func(ctx context.Context, command string) (string, string, error) {
			t.Fatalf("runCmd should NOT be called; evaluated via shell: %q", command)
			return "", "", nil
		}
		attempts := 0
		runCmdArgsUpdateFn = func(ctx context.Context, name string, args ...string) (string, string, error) {
			call := append([]string{name}, args...)
			recordedCalls = append(recordedCalls, call)
			attempts++
			if attempts == 1 {
				return "", "corrupt store detected", errors.New("exit 1")
			}
			return "", "", nil
		}
		runCmdArgsFn = func(ctx context.Context, name string, args ...string) (string, string, error) {
			return "", "", nil
		}
		lookPathFn = func(string) bool { return true }

		t.Cleanup(func() {
			runCmdFn = origRunCmd
			runCmdArgsUpdateFn = origRunCmdArgsUpdate
			runCmdArgsFn = origRunCmdArgs
			lookPathFn = origLookPath
		})

		adpt := &PnpmAdapter{}
		res, err := adpt.Update(context.Background(), false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !res.Success {
			t.Fatalf("expected success after recovery, got error: %v", res.Error)
		}

		wantCalls := [][]string{
			{"pnpm", "update", "-g"},
			{"pnpm", "store", "prune"},
			{"pnpm", "update", "-g"},
		}
		if len(recordedCalls) != len(wantCalls) {
			t.Fatalf("recorded calls = %v, want %v", recordedCalls, wantCalls)
		}
		for i := range recordedCalls {
			if len(recordedCalls[i]) != len(wantCalls[i]) {
				t.Fatalf("call[%d] len = %d (%v), want %d (%v)", i, len(recordedCalls[i]), recordedCalls[i], len(wantCalls[i]), wantCalls[i])
			}
			for j := range recordedCalls[i] {
				if recordedCalls[i][j] != wantCalls[i][j] {
					t.Errorf("call[%d][%d] = %q, want %q", i, j, recordedCalls[i][j], wantCalls[i][j])
				}
			}
		}
	})
}

func TestGo_LinuxAtomicSwapAndRollback(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Go atomic swap is Linux-specific")
	}

	t.Run("success stages and executes backup, swap, and cleanup", func(t *testing.T) {
		var recordedCalls [][]string
		setExecFakes(t, execFakes{
			lookPath:     map[string]bool{"go": true},
			cmdArgs:      map[string]fakeResult{"go": {stdout: "go version go1.22.0 linux/amd64"}},
			recordedCmds: &recordedCalls,
		})

		adpt := &GoAdapter{}
		res, err := adpt.Update(context.Background(), false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !res.Success {
			t.Fatalf("expected success, got error: %v", res.Error)
		}

		if len(recordedCalls) != 3 {
			t.Fatalf("recorded calls count = %d, want 3; calls: %v", len(recordedCalls), recordedCalls)
		}
		if len(recordedCalls[0]) != 4 || recordedCalls[0][0] != "sudo" || recordedCalls[0][1] != "mv" || recordedCalls[0][2] != "/usr/local/go" || recordedCalls[0][3] != "/usr/local/go.bak" {
			t.Errorf("call 0 = %v, want [sudo mv /usr/local/go /usr/local/go.bak]", recordedCalls[0])
		}
		if len(recordedCalls[1]) != 4 || recordedCalls[1][0] != "sudo" || recordedCalls[1][1] != "mv" || recordedCalls[1][3] != "/usr/local/go" {
			t.Errorf("call 1 = %v, want [sudo mv <staged> /usr/local/go]", recordedCalls[1])
		}
		if len(recordedCalls[2]) != 4 || recordedCalls[2][0] != "sudo" || recordedCalls[2][1] != "rm" || recordedCalls[2][2] != "-rf" || recordedCalls[2][3] != "/usr/local/go.bak" {
			t.Errorf("call 2 = %v, want [sudo rm -rf /usr/local/go.bak]", recordedCalls[2])
		}
	})

	t.Run("swap failure triggers rollback to restore backup", func(t *testing.T) {
		var recordedCalls [][]string
		setExecFakes(t, execFakes{
			lookPath: map[string]bool{"go": true},
			cmdArgs: map[string]fakeResult{
				"go":                           {stdout: "go version go1.22.0 linux/amd64"},
				"sudo mv staged /usr/local/go": {err: errors.New("mv: disk full")},
			},
			recordedCmds: &recordedCalls,
		})

		adpt := &GoAdapter{}
		res, err := adpt.Update(context.Background(), false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Success {
			t.Fatal("expected failure on swap error, got success")
		}

		if len(recordedCalls) != 3 {
			t.Fatalf("recorded calls count = %d, want 3; calls: %v", len(recordedCalls), recordedCalls)
		}
		if len(recordedCalls[0]) != 4 || recordedCalls[0][2] != "/usr/local/go" || recordedCalls[0][3] != "/usr/local/go.bak" {
			t.Errorf("call 0 = %v, want backup", recordedCalls[0])
		}
		if len(recordedCalls[1]) != 4 || recordedCalls[1][3] != "/usr/local/go" {
			t.Errorf("call 1 = %v, want swap attempt", recordedCalls[1])
		}
		if len(recordedCalls[2]) != 4 || recordedCalls[2][0] != "sudo" || recordedCalls[2][1] != "mv" || recordedCalls[2][2] != "/usr/local/go.bak" || recordedCalls[2][3] != "/usr/local/go" {
			t.Errorf("call 2 = %v, want [sudo mv /usr/local/go.bak /usr/local/go]", recordedCalls[2])
		}
	})

	t.Run("checksum mismatch aborts before privileged execution", func(t *testing.T) {
		var recordedCalls [][]string
		setExecFakes(t, execFakes{
			lookPath:      map[string]bool{"go": true},
			cmdArgs:       map[string]fakeResult{"go": {stdout: "go version go1.22.0 linux/amd64"}},
			goDownloadErr: errors.New("checksum mismatch for archive"),
			recordedCmds:  &recordedCalls,
		})

		adpt := &GoAdapter{}
		res, err := adpt.Update(context.Background(), false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Success {
			t.Fatal("expected failure on checksum mismatch, got success")
		}
		if len(recordedCalls) != 0 {
			t.Errorf("expected 0 privileged commands on checksum mismatch, got %v", recordedCalls)
		}
	})

	t.Run("non-standard installation path aborts before privileged execution", func(t *testing.T) {
		var recordedCalls [][]string
		setExecFakes(t, execFakes{
			goBinaryPath: "/home/user/go/bin/go",
			lookPath:     map[string]bool{"go": true},
			cmdArgs:      map[string]fakeResult{"go": {stdout: "go version go1.22.0 linux/amd64"}},
			recordedCmds: &recordedCalls,
		})

		adpt := &GoAdapter{}
		res, err := adpt.Update(context.Background(), false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Success {
			t.Fatal("expected failure on non-standard path, got success")
		}
		if len(recordedCalls) != 0 {
			t.Errorf("expected 0 privileged commands on path guardrail, got %v", recordedCalls)
		}
	})
}
