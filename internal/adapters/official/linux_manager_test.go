package official

import (
	"context"
	"runtime"
	"testing"

	"github.com/JhnFrankz/upp/internal/adapters"
)

// TestDynamicLinuxManager_Info tests dynamic package manager and package name
// resolution for gh and docker on Linux depending on whether apt or pacman is present.
func TestDynamicLinuxManager_Info(t *testing.T) {
	tests := []struct {
		name              string
		fakes             execFakes
		wantGhManager     string
		wantGhPkg         string
		wantDockerManager string
		wantDockerPkg     string
		wantOwnerType     string
	}{
		{
			name: "pacman fallback when apt absent and pacman present",
			fakes: execFakes{
				lookPath: map[string]bool{"apt": false, "pacman": true},
			},
			wantGhManager:     "pacman",
			wantGhPkg:         "github-cli",
			wantDockerManager: "pacman",
			wantDockerPkg:     "docker",
			wantOwnerType:     "*official.PacmanAdapter",
		},
		{
			name: "default to apt when apt is present (pacman also present)",
			fakes: execFakes{
				lookPath: map[string]bool{"apt": true, "pacman": true},
			},
			wantGhManager:     "apt",
			wantGhPkg:         "gh",
			wantDockerManager: "apt",
			wantDockerPkg:     "docker-ce",
			wantOwnerType:     "*official.AptAdapter",
		},
		{
			name: "default to apt when apt is present (pacman absent)",
			fakes: execFakes{
				lookPath: map[string]bool{"apt": true, "pacman": false},
			},
			wantGhManager:     "apt",
			wantGhPkg:         "gh",
			wantDockerManager: "apt",
			wantDockerPkg:     "docker-ce",
			wantOwnerType:     "*official.AptAdapter",
		},
		{
			name: "default to apt when neither apt nor pacman is present",
			fakes: execFakes{
				lookPath: map[string]bool{"apt": false, "pacman": false},
			},
			wantGhManager:     "apt",
			wantGhPkg:         "gh",
			wantDockerManager: "apt",
			wantDockerPkg:     "docker-ce",
			wantOwnerType:     "*official.AptAdapter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setExecFakes(t, tt.fakes)

			// gh assertions
			ghInfo := (&GhAdapter{}).Info()
			if got := ghInfo.Manager["linux"]; got != tt.wantGhManager {
				t.Errorf("gh.Info().Manager[\"linux\"] = %q, want %q", got, tt.wantGhManager)
			}
			if got := ghInfo.ManagerPackage["linux"]; got != tt.wantGhPkg {
				t.Errorf("gh.Info().ManagerPackage[\"linux\"] = %q, want %q", got, tt.wantGhPkg)
			}

			// docker assertions
			dockerInfo := (&DockerAdapter{}).Info()
			if got := dockerInfo.Manager["linux"]; got != tt.wantDockerManager {
				t.Errorf("docker.Info().Manager[\"linux\"] = %q, want %q", got, tt.wantDockerManager)
			}
			if got := dockerInfo.ManagerPackage["linux"]; got != tt.wantDockerPkg {
				t.Errorf("docker.Info().ManagerPackage[\"linux\"] = %q, want %q", got, tt.wantDockerPkg)
			}

			// ResolveOwner assertions
			ghOwner := ResolveOwner("gh", "linux")
			if ghOwner == nil {
				t.Fatalf("ResolveOwner(\"gh\", \"linux\") = nil, want %s", tt.wantOwnerType)
			}
			if ghOwner.Name() != tt.wantGhManager {
				t.Errorf("ResolveOwner(\"gh\", \"linux\").Name() = %q, want %q", ghOwner.Name(), tt.wantGhManager)
			}

			dockerOwner := ResolveOwner("docker", "linux")
			if dockerOwner == nil {
				t.Fatalf("ResolveOwner(\"docker\", \"linux\") = nil, want %s", tt.wantOwnerType)
			}
			if dockerOwner.Name() != tt.wantDockerManager {
				t.Errorf("ResolveOwner(\"docker\", \"linux\").Name() = %q, want %q", dockerOwner.Name(), tt.wantDockerManager)
			}

			if tt.wantGhManager == "pacman" {
				if _, ok := ghOwner.(*PacmanAdapter); !ok {
					t.Errorf("ghOwner type = %T, want *PacmanAdapter", ghOwner)
				}
				if _, ok := dockerOwner.(*PacmanAdapter); !ok {
					t.Errorf("dockerOwner type = %T, want *PacmanAdapter", dockerOwner)
				}
			} else {
				if _, ok := ghOwner.(*AptAdapter); !ok {
					t.Errorf("ghOwner type = %T, want *AptAdapter", ghOwner)
				}
				if _, ok := dockerOwner.(*AptAdapter); !ok {
					t.Errorf("dockerOwner type = %T, want *AptAdapter", dockerOwner)
				}
			}
		})
	}
}

// TestDynamicLinuxManager_CheckDelegation tests that gh and docker delegate Check()
// to pacman.CheckPackage on Linux when owned by pacman.
func TestDynamicLinuxManager_CheckDelegation(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Check delegation to pacman is Linux-specific")
	}

	t.Run("gh delegates Check to pacman CheckPackage", func(t *testing.T) {
		setExecFakes(t, execFakes{
			lookPath: map[string]bool{"gh": true, "apt": false, "pacman": true, "vercmp": true},
			cmdArgs: map[string]fakeResult{
				"pacman -Q github-cli":     {stdout: "github-cli 2.45.0-1"},
				"pacman -Si github-cli":    {stdout: "Repository : extra\nName : github-cli\nVersion : 2.46.0-1\n"},
				"vercmp 2.46.0-1 2.45.0-1": {stdout: "1"},
			},
		})

		info, err := (&GhAdapter{}).Check(context.Background())
		if err != nil {
			t.Fatalf("gh.Check() error = %v, want nil", err)
		}
		want := adapters.UpdateInfo{
			CurrentVersion:  "2.45.0-1",
			LatestVersion:   "2.46.0-1",
			UpdateAvailable: true,
		}
		if info != want {
			t.Errorf("gh.Check() = %+v, want %+v", info, want)
		}
	})

	t.Run("docker delegates Check to pacman CheckPackage", func(t *testing.T) {
		setExecFakes(t, execFakes{
			lookPath: map[string]bool{"docker": true, "apt": false, "pacman": true, "vercmp": true},
			cmdArgs: map[string]fakeResult{
				"pacman -Q docker":         {stdout: "docker 26.1.4-1"},
				"pacman -Si docker":        {stdout: "Repository : extra\nName : docker\nVersion : 26.2.0-1\n"},
				"vercmp 26.2.0-1 26.1.4-1": {stdout: "1"},
			},
		})

		info, err := (&DockerAdapter{}).Check(context.Background())
		if err != nil {
			t.Fatalf("docker.Check() error = %v, want nil", err)
		}
		want := adapters.UpdateInfo{
			CurrentVersion:  "26.1.4-1",
			LatestVersion:   "26.2.0-1",
			UpdateAvailable: true,
		}
		if info != want {
			t.Errorf("docker.Check() = %+v, want %+v", info, want)
		}
	})
}

// TestDynamicLinuxManager_UpdateDelegation tests that gh and docker delegate Update()
// to pacman.UpdatePackage on Linux when owned by pacman.
func TestDynamicLinuxManager_UpdateDelegation(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Update delegation to pacman is Linux-specific")
	}

	t.Run("gh delegates Update to pacman UpdatePackage (non-dry-run)", func(t *testing.T) {
		setExecFakes(t, execFakes{
			lookPath: map[string]bool{"gh": true, "apt": false, "pacman": true},
			cmdArgs: map[string]fakeResult{
				"gh":               {stdout: "gh version 2.45.0 (2024-05-30)"},
				"pacman -Q pacman": {stdout: "pacman 6.1.0-1"},
			},
			shell: map[string]fakeResult{
				"sudo pacman -S --noconfirm github-cli": {stdout: "upgrading github-cli..."},
			},
		})

		res, err := (&GhAdapter{}).Update(context.Background(), false)
		if err != nil {
			t.Fatalf("gh.Update(false) error = %v, want nil", err)
		}
		if !res.Success {
			t.Errorf("gh.Update(false) Success = false, want true")
		}
		if !equalPrivileges(res.Privileges, []string{"sudo"}) {
			t.Errorf("gh.Update(false) Privileges = %v, want [sudo]", res.Privileges)
		}
		if res.Before != "6.1.0-1" || res.After != "6.1.0-1" {
			t.Errorf("gh.Update(false) Before/After = %q/%q, want 6.1.0-1", res.Before, res.After)
		}
	})

	t.Run("docker delegates Update to pacman UpdatePackage (non-dry-run)", func(t *testing.T) {
		setExecFakes(t, execFakes{
			lookPath: map[string]bool{"docker": true, "apt": false, "pacman": true},
			cmdArgs: map[string]fakeResult{
				"docker":           {stdout: "Docker version 26.1.4, build 5650f9b"},
				"pacman -Q pacman": {stdout: "pacman 6.1.0-1"},
			},
			shell: map[string]fakeResult{
				"sudo pacman -S --noconfirm docker": {stdout: "upgrading docker..."},
			},
		})

		res, err := (&DockerAdapter{}).Update(context.Background(), false)
		if err != nil {
			t.Fatalf("docker.Update(false) error = %v, want nil", err)
		}
		if !res.Success {
			t.Errorf("docker.Update(false) Success = false, want true")
		}
		if !equalPrivileges(res.Privileges, []string{"sudo"}) {
			t.Errorf("docker.Update(false) Privileges = %v, want [sudo]", res.Privileges)
		}
		if res.Before != "6.1.0-1" || res.After != "6.1.0-1" {
			t.Errorf("docker.Update(false) Before/After = %q/%q, want 6.1.0-1", res.Before, res.After)
		}
	})

	t.Run("gh dry-run shortcut skips pacman update execution", func(t *testing.T) {
		setExecFakes(t, execFakes{
			lookPath: map[string]bool{"gh": true, "apt": false, "pacman": true},
			cmdArgs: map[string]fakeResult{
				"gh": {stdout: "gh version 2.45.0 (2024-05-30)"},
			},
			shell: map[string]fakeResult{
				"sudo pacman -S --noconfirm github-cli": failIfRun,
			},
		})

		res, err := (&GhAdapter{}).Update(context.Background(), true)
		if err != nil {
			t.Fatalf("gh.Update(true) error = %v, want nil", err)
		}
		if !res.Success {
			t.Errorf("gh.Update(true) Success = false, want true")
		}
	})

	t.Run("docker dry-run shortcut skips pacman update execution", func(t *testing.T) {
		setExecFakes(t, execFakes{
			lookPath: map[string]bool{"docker": true, "apt": false, "pacman": true},
			cmdArgs: map[string]fakeResult{
				"docker": {stdout: "Docker version 26.1.4, build 5650f9b"},
			},
			shell: map[string]fakeResult{
				"sudo pacman -S --noconfirm docker": failIfRun,
			},
		})

		res, err := (&DockerAdapter{}).Update(context.Background(), true)
		if err != nil {
			t.Fatalf("docker.Update(true) error = %v, want nil", err)
		}
		if !res.Success {
			t.Errorf("docker.Update(true) Success = false, want true")
		}
	})
}
