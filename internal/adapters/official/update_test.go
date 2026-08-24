package official

import (
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
	aptUpdateCmd           = "sudo apt install --only-upgrade apt"
	brewUpdateCmd          = "brew update"
	npmUpdateCmd           = "npm update -g"
	pnpmUpdateCmd          = "pnpm update -g"
	pnpmPruneCmd           = "pnpm store prune 2>/dev/null"
	bunUpdateCmd           = "bun upgrade"
	ghLinuxUpdateCmd       = "sudo apt update -qq && sudo apt install -y gh"
	ghDarwinUpdateCmd      = "brew upgrade gh"
	ghWindowsUpdateCmd     = "winget upgrade gh --accept-source-agreements --accept-package-agreements"
	dockerLinuxUpdateCmd   = "sudo apt update -qq && sudo apt upgrade -y docker-ce docker-ce-cli containerd.io"
	dockerDarwinUpdateCmd  = "brew upgrade docker"
	dockerWindowsUpdateCmd = "winget upgrade Docker.DockerDesktop --accept-source-agreements --accept-package-agreements"
	goLinuxUpdateCmd       = "curl -fsSL https://go.dev/dl/$(curl -fsSL https://go.dev/VERSION?m=text | head -1).linux-amd64.tar.gz | sudo tar -C /usr/local -xzf -"
	goDarwinUpdateCmd      = "brew upgrade go"
	goWindowsUpdateCmd     = "winget upgrade GoLang.Go --accept-source-agreements --accept-package-agreements"
	opencodeUpdateCmd      = "curl -fsSL https://opencode.ai/install | bash"
	wingetUpdateCmd        = "winget upgrade winget"
	scoopUpdateCmd         = "scoop update *"
	nvmInstallStableCmd    = "bash -c 'source \"${NVM_DIR:-$HOME/.nvm}/nvm.sh\" >/dev/null 2>&1 && nvm install stable'"
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

		// --- gh (per-GOOS update command) ---
		{
			name:    "gh/not-installed-error",
			newAdpt: func() adapters.Adapter { return &GhAdapter{} },
			fakes:   execFakes{lookPath: map[string]bool{"gh": false}},
			wantErr: true,
		},
		{
			name:    "gh/dry-run-shortcut",
			newAdpt: func() adapters.Adapter { return &GhAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"gh": true},
				cmdArgs:  map[string]fakeResult{"gh": {stdout: "gh version 2.45.0 (2024-05-30)"}},
			},
			dryRun: true,
			want:   adapters.Result{Success: true, Before: "2.45.0", After: "2.45.0"},
		},
		{
			name:    "gh/update-command-error",
			newAdpt: func() adapters.Adapter { return &GhAdapter{} },
			goos:    "linux",
			fakes: execFakes{
				lookPath: map[string]bool{"gh": true},
				cmdArgs:  map[string]fakeResult{"gh": {stdout: "gh version 2.45.0 (2024-05-30)"}},
				shell:    map[string]fakeResult{ghLinuxUpdateCmd: {err: errors.New("apt: lock held")}},
			},
			want:      adapters.Result{Success: false, Before: "2.45.0", After: "2.45.0", Privileges: sudo},
			resultErr: true,
		},
		{
			name:    "gh/stderr-error-marker",
			newAdpt: func() adapters.Adapter { return &GhAdapter{} },
			goos:    "linux",
			fakes: execFakes{
				lookPath: map[string]bool{"gh": true},
				cmdArgs:  map[string]fakeResult{"gh": {stdout: "gh version 2.45.0 (2024-05-30)"}},
				shell:    map[string]fakeResult{ghLinuxUpdateCmd: {stderr: "E: Package gh has no installation candidate"}},
			},
			want:      adapters.Result{Success: false, Before: "2.45.0", After: "2.45.0", Privileges: sudo},
			resultErr: true,
		},
		{
			name:    "gh/success",
			newAdpt: func() adapters.Adapter { return &GhAdapter{} },
			goos:    "linux",
			fakes: execFakes{
				lookPath: map[string]bool{"gh": true},
				cmdArgs:  map[string]fakeResult{"gh": {stdout: "gh version 2.45.0 (2024-05-30)"}},
				shell:    map[string]fakeResult{ghLinuxUpdateCmd: {}},
			},
			want: adapters.Result{Success: true, Before: "2.45.0", After: "2.45.0", Privileges: sudo},
		},
		{
			name:    "gh/darwin-success",
			newAdpt: func() adapters.Adapter { return &GhAdapter{} },
			goos:    "darwin",
			fakes: execFakes{
				lookPath: map[string]bool{"gh": true},
				cmdArgs:  map[string]fakeResult{"gh": {stdout: "gh version 2.45.0 (2024-05-30)"}},
				shell:    map[string]fakeResult{ghDarwinUpdateCmd: {}},
			},
			want: adapters.Result{Success: true, Before: "2.45.0", After: "2.45.0"},
		},
		{
			name:    "gh/windows-success",
			newAdpt: func() adapters.Adapter { return &GhAdapter{} },
			goos:    "windows",
			fakes: execFakes{
				lookPath: map[string]bool{"gh": true},
				cmdArgs:  map[string]fakeResult{"gh": {stdout: "gh version 2.45.0 (2024-05-30)"}},
				shell:    map[string]fakeResult{ghWindowsUpdateCmd: {}},
			},
			want: adapters.Result{Success: true, Before: "2.45.0", After: "2.45.0"},
		},

		// --- docker (per-GOOS update command) ---
		{
			name:    "docker/not-installed-error",
			newAdpt: func() adapters.Adapter { return &DockerAdapter{} },
			fakes:   execFakes{lookPath: map[string]bool{"docker": false}},
			wantErr: true,
		},
		{
			name:    "docker/dry-run-shortcut",
			newAdpt: func() adapters.Adapter { return &DockerAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"docker": true},
				cmdArgs:  map[string]fakeResult{"docker": {stdout: "Docker version 26.1.4, build 5650f9b"}},
			},
			dryRun: true,
			want:   adapters.Result{Success: true, Before: "26.1.4", After: "26.1.4"},
		},
		{
			name:    "docker/update-command-error",
			newAdpt: func() adapters.Adapter { return &DockerAdapter{} },
			goos:    "linux",
			fakes: execFakes{
				lookPath: map[string]bool{"docker": true},
				cmdArgs:  map[string]fakeResult{"docker": {stdout: "Docker version 26.1.4, build 5650f9b"}},
				shell:    map[string]fakeResult{dockerLinuxUpdateCmd: {err: errors.New("apt: lock held")}},
			},
			want:      adapters.Result{Success: false, Before: "26.1.4", After: "26.1.4", Privileges: sudo},
			resultErr: true,
		},
		{
			name:    "docker/stderr-error-marker",
			newAdpt: func() adapters.Adapter { return &DockerAdapter{} },
			goos:    "linux",
			fakes: execFakes{
				lookPath: map[string]bool{"docker": true},
				cmdArgs:  map[string]fakeResult{"docker": {stdout: "Docker version 26.1.4, build 5650f9b"}},
				shell:    map[string]fakeResult{dockerLinuxUpdateCmd: {stderr: "E: Unable to locate package docker-ce"}},
			},
			want:      adapters.Result{Success: false, Before: "26.1.4", After: "26.1.4", Privileges: sudo},
			resultErr: true,
		},
		{
			name:    "docker/success",
			newAdpt: func() adapters.Adapter { return &DockerAdapter{} },
			goos:    "linux",
			fakes: execFakes{
				lookPath: map[string]bool{"docker": true},
				cmdArgs:  map[string]fakeResult{"docker": {stdout: "Docker version 26.1.4, build 5650f9b"}},
				shell:    map[string]fakeResult{dockerLinuxUpdateCmd: {}},
			},
			want: adapters.Result{Success: true, Before: "26.1.4", After: "26.1.4", Privileges: sudo},
		},
		{
			name:    "docker/darwin-success",
			newAdpt: func() adapters.Adapter { return &DockerAdapter{} },
			goos:    "darwin",
			fakes: execFakes{
				lookPath: map[string]bool{"docker": true},
				cmdArgs:  map[string]fakeResult{"docker": {stdout: "Docker version 26.1.4, build 5650f9b"}},
				shell:    map[string]fakeResult{dockerDarwinUpdateCmd: {}},
			},
			want: adapters.Result{Success: true, Before: "26.1.4", After: "26.1.4"},
		},
		{
			name:    "docker/windows-success",
			newAdpt: func() adapters.Adapter { return &DockerAdapter{} },
			goos:    "windows",
			fakes: execFakes{
				lookPath: map[string]bool{"docker": true},
				cmdArgs:  map[string]fakeResult{"docker": {stdout: "Docker version 26.1.4, build 5650f9b"}},
				shell:    map[string]fakeResult{dockerWindowsUpdateCmd: {}},
			},
			want: adapters.Result{Success: true, Before: "26.1.4", After: "26.1.4"},
		},

		// --- go (per-GOOS update command) ---
		{
			name:    "go/not-installed-error",
			newAdpt: func() adapters.Adapter { return &GoAdapter{} },
			fakes:   execFakes{lookPath: map[string]bool{"go": false}},
			wantErr: true,
		},
		{
			name:    "go/dry-run-shortcut",
			newAdpt: func() adapters.Adapter { return &GoAdapter{} },
			fakes: execFakes{
				lookPath: map[string]bool{"go": true},
				cmdArgs:  map[string]fakeResult{"go": {stdout: "go version go1.22.0 linux/amd64"}},
			},
			dryRun: true,
			want:   adapters.Result{Success: true, Before: "1.22.0", After: "1.22.0"},
		},
		{
			name:    "go/update-command-error",
			newAdpt: func() adapters.Adapter { return &GoAdapter{} },
			goos:    "linux",
			fakes: execFakes{
				lookPath: map[string]bool{"go": true},
				cmdArgs:  map[string]fakeResult{"go": {stdout: "go version go1.22.0 linux/amd64"}},
				shell:    map[string]fakeResult{goLinuxUpdateCmd: {err: errors.New("curl: connection failed")}},
			},
			want:      adapters.Result{Success: false, Before: "1.22.0", After: "1.22.0", Privileges: sudo},
			resultErr: true,
		},
		{
			name:    "go/stderr-error-marker",
			newAdpt: func() adapters.Adapter { return &GoAdapter{} },
			goos:    "linux",
			fakes: execFakes{
				lookPath: map[string]bool{"go": true},
				cmdArgs:  map[string]fakeResult{"go": {stdout: "go version go1.22.0 linux/amd64"}},
				shell:    map[string]fakeResult{goLinuxUpdateCmd: {stderr: "error: cannot write to /usr/local/go"}},
			},
			want:      adapters.Result{Success: false, Before: "1.22.0", After: "1.22.0", Privileges: sudo},
			resultErr: true,
		},
		{
			name:    "go/success",
			newAdpt: func() adapters.Adapter { return &GoAdapter{} },
			goos:    "linux",
			fakes: execFakes{
				lookPath: map[string]bool{"go": true},
				cmdArgs:  map[string]fakeResult{"go": {stdout: "go version go1.22.0 linux/amd64"}},
				shell:    map[string]fakeResult{goLinuxUpdateCmd: {}},
			},
			want: adapters.Result{Success: true, Before: "1.22.0", After: "1.22.0", Privileges: sudo},
		},
		{
			name:    "go/darwin-success",
			newAdpt: func() adapters.Adapter { return &GoAdapter{} },
			goos:    "darwin",
			fakes: execFakes{
				lookPath: map[string]bool{"go": true},
				cmdArgs:  map[string]fakeResult{"go": {stdout: "go version go1.22.0 linux/amd64"}},
				shell:    map[string]fakeResult{goDarwinUpdateCmd: {}},
			},
			want: adapters.Result{Success: true, Before: "1.22.0", After: "1.22.0"},
		},
		{
			name:    "go/windows-success",
			newAdpt: func() adapters.Adapter { return &GoAdapter{} },
			goos:    "windows",
			fakes: execFakes{
				lookPath: map[string]bool{"go": true},
				cmdArgs:  map[string]fakeResult{"go": {stdout: "go version go1.22.0 linux/amd64"}},
				shell:    map[string]fakeResult{goWindowsUpdateCmd: {}},
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
					nvmCurrentCmd:       {stdout: "v20.11.0"},
					nvmInstallStableCmd: failIfRun,
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
					nvmCurrentCmd:       {stdout: "v20.11.0"},
					nvmInstallStableCmd: {err: errors.New("nvm: curl failed")},
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
					nvmCurrentCmd:       {stdout: "v20.11.0"},
					nvmInstallStableCmd: {stderr: "error: version not found"}},
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
					nvmCurrentCmd:       {stdout: "v20.11.0"},
					nvmInstallStableCmd: {},
				},
			},
			want: adapters.Result{Success: true, Before: "v20.11.0", After: "v20.11.0"},
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

			got, err := tt.newAdpt().Update(tt.dryRun)
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

	result, err := (&PnpmAdapter{}).Update(false)
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
