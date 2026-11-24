package official

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JhnFrankz/upp/internal/adapters"
)

func TestWingetAndScoopCoalesceConcurrentQueries(t *testing.T) {
	t.Run("winget upgrade runs once across concurrent Check and CheckPackage calls", func(t *testing.T) {
		var upgradeCalls atomic.Int32
		setExecFakes(t, execFakes{
			lookPath: map[string]bool{"winget": true},
		})
		orig := runCmdArgsFn
		runCmdArgsFn = func(ctx context.Context, name string, args ...string) (string, string, error) {
			if name == "winget" && len(args) == 1 && args[0] == "--version" {
				return "v1.8.2301", "", nil
			}
			if name == "winget" && len(args) == 1 && args[0] == "upgrade" {
				upgradeCalls.Add(1)
				return "winget  Microsoft.AppInstaller  v1.8.2301  v1.8.2311  winget\ngh  GitHub.cli  2.45.0  2.46.0  winget\nDocker.Docker  Docker.Docker  4.28.0  4.29.0  winget\nGoLang.Go  GoLang.Go  1.22.0  1.22.1  winget\n", "", nil
			}
			return orig(ctx, name, args...)
		}

		w := &WingetAdapter{}
		var wg sync.WaitGroup
		wg.Add(4)
		go func() { defer wg.Done(); _, _ = w.Check(context.Background()) }()
		go func() { defer wg.Done(); _, _ = w.CheckPackage(context.Background(), "GitHub.cli") }()
		go func() { defer wg.Done(); _, _ = w.CheckPackage(context.Background(), "Docker.Docker") }()
		go func() { defer wg.Done(); _, _ = w.CheckPackage(context.Background(), "GoLang.Go") }()
		wg.Wait()

		if got := upgradeCalls.Load(); got != 1 {
			t.Errorf("winget upgrade invoked %d times, want 1", got)
		}
	})

	t.Run("scoop status runs once across concurrent Check and CheckPackage calls", func(t *testing.T) {
		var statusCalls atomic.Int32
		setExecFakes(t, execFakes{
			lookPath: map[string]bool{"scoop": true},
		})
		orig := runCmdArgsFn
		runCmdArgsFn = func(ctx context.Context, name string, args ...string) (string, string, error) {
			if name == "scoop" && len(args) == 1 && args[0] == "status" {
				statusCalls.Add(1)
				return "Name   Installed Version Latest Version Missing Dependencies Info\n----   ----------------- -------------- -------------------- ----\nscoop  0.4.0             0.4.1\ngh     2.45.0            2.46.0\n", "", nil
			}
			return orig(ctx, name, args...)
		}

		s := &ScoopAdapter{}
		var wg sync.WaitGroup
		wg.Add(3)
		go func() { defer wg.Done(); _, _ = s.Check(context.Background()) }()
		go func() { defer wg.Done(); _, _ = s.CheckPackage(context.Background(), "gh") }()
		go func() { defer wg.Done(); _, _ = s.CheckPackage(context.Background(), "docker") }()
		wg.Wait()

		if got := statusCalls.Load(); got != 1 {
			t.Errorf("scoop status invoked %d times, want 1", got)
		}
	})
}

func TestPathCachesInvalidateOnPathChange(t *testing.T) {
	t.Run("defaultLinuxManager invalidates when PATH changes", func(t *testing.T) {
		setExecFakes(t, execFakes{goos: "linux"})
		calls := 0
		lookPathFn = func(name string) bool {
			calls++
			return name == "pacman"
		}

		t.Setenv("PATH", "/bin/first")
		if got := defaultLinuxManager(); got != "pacman" {
			t.Fatalf("first defaultLinuxManager() = %q, want pacman", got)
		}
		if got := defaultLinuxManager(); got != "pacman" {
			t.Fatalf("cached defaultLinuxManager() = %q, want pacman", got)
		}
		callsAfterFirstPath := calls

		t.Setenv("PATH", "/bin/second")
		lookPathFn = func(name string) bool {
			calls++
			return name == "apt"
		}
		if got := defaultLinuxManager(); got != "apt" {
			t.Fatalf("after PATH change defaultLinuxManager() = %q, want apt", got)
		}
		if calls <= callsAfterFirstPath {
			t.Error("expected lookPathFn to be re-evaluated after PATH change")
		}
	})

	t.Run("GoAdapter.Info invalidates when PATH changes", func(t *testing.T) {
		setExecFakes(t, execFakes{
			goos:         "linux",
			goBinaryPath: "/usr/bin/go",
			lookPath:     map[string]bool{"go": true, "apt": false, "pacman": true},
		})
		t.Setenv("PATH", "/bin/first")
		info1 := (&GoAdapter{}).Info()
		if info1.Manager["linux"] != "pacman" {
			t.Fatalf("first GoAdapter.Info().Manager[linux] = %q, want pacman", info1.Manager["linux"])
		}

		t.Setenv("PATH", "/bin/second")
		lookPathFn = func(name string) bool {
			return name == "go" || name == "apt"
		}
		info2 := (&GoAdapter{}).Info()
		if info2.Manager["linux"] != "apt" {
			t.Fatalf("after PATH change GoAdapter.Info().Manager[linux] = %q, want apt", info2.Manager["linux"])
		}
	})
}

func TestCoalescedQuery_PanicAndDetachedCtx(t *testing.T) {
	t.Run("panic in fn clears inflight so subsequent call runs cleanly", func(t *testing.T) {
		var q coalescedQuery
		if _, err := q.run(context.Background(), func(context.Context) (string, error) {
			panic("boom")
		}); err == nil {
			t.Fatal("expected error from panicked coalescedQuery.run, got nil")
		}

		done := make(chan string, 1)
		go func() {
			out, err := q.run(context.Background(), func(context.Context) (string, error) {
				return "ok", nil
			})
			if err != nil {
				t.Errorf("subsequent call error = %v", err)
			}
			done <- out
		}()
		select {
		case got := <-done:
			if got != "ok" {
				t.Errorf("got %q, want ok", got)
			}
		case <-time.After(time.Second):
			t.Fatal("subsequent call deadlocked after panic in coalescedQuery.run")
		}
	})

	t.Run("caller A cancel does not abort caller B", func(t *testing.T) {
		var q coalescedQuery
		started := make(chan struct{})
		release := make(chan struct{})

		ctxA, cancelA := context.WithCancel(context.Background())
		errA := make(chan error, 1)
		go func() {
			_, err := q.run(ctxA, func(c context.Context) (string, error) {
				close(started)
				<-release
				if c.Err() != nil {
					return "", c.Err()
				}
				return "shared-result", nil
			})
			errA <- err
		}()

		<-started
		type resB struct {
			out string
			err error
		}
		chB := make(chan resB, 1)
		go func() {
			out, err := q.run(context.Background(), func(context.Context) (string, error) {
				return "unexpected-second-run", nil
			})
			chB <- resB{out, err}
		}()

		time.Sleep(10 * time.Millisecond)
		cancelA()
		if err := <-errA; !errors.Is(err, context.Canceled) {
			t.Fatalf("caller A err = %v, want context.Canceled", err)
		}

		close(release)
		gotB := <-chB
		if gotB.err != nil || gotB.out != "shared-result" {
			t.Fatalf("caller B got (%q, %v), want (shared-result, nil)", gotB.out, gotB.err)
		}
	})
}

func TestNVMCheck_UsesCheckTimeoutSeam(t *testing.T) {
	setExecFakes(t, execFakes{goos: "linux"})
	origRunCmd := runCmdFn
	origRunCmdCheck := runCmdCheckFn
	defer func() {
		runCmdFn = origRunCmd
		runCmdCheckFn = origRunCmdCheck
	}()

	runCmdFn = func(context.Context, string) (string, string, error) {
		t.Fatal("NVMAdapter.Check used runCmdFn (UpdateTimeout) instead of runCmdCheckFn (CheckTimeout)")
		return "", "", nil
	}
	checkCalls := 0
	runCmdCheckFn = func(ctx context.Context, command string) (string, string, error) {
		checkCalls++
		return "v22.0.0", "", nil
	}

	// Ensure Detect() returns true via temporary NVM_DIR.
	nvmDir := t.TempDir()
	t.Setenv("NVM_DIR", nvmDir)
	if err := os.WriteFile(filepath.Join(nvmDir, "nvm.sh"), nil, 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := (&NVMAdapter{}).Check(context.Background()); err != nil {
		t.Fatalf("Check error: %v", err)
	}
	if checkCalls != 2 {
		t.Errorf("runCmdCheckFn called %d times, want 2", checkCalls)
	}
}

func TestGoAdapter_UnmanagedInfoIsPolicyGated(t *testing.T) {
	tests := []struct {
		name    string
		fakes   execFakes
		wantMgr string
		wantCmd string
	}{
		{
			name: "linux non-standard path without apt/pacman",
			fakes: execFakes{
				goos:         "linux",
				goBinaryPath: "/opt/custom/go/bin/go",
				lookPath:     map[string]bool{"go": true},
			},
		},
		{
			name: "macos without brew",
			fakes: execFakes{
				goos:     "darwin",
				lookPath: map[string]bool{"go": true},
			},
		},
		{
			name: "windows without winget",
			fakes: execFakes{
				goos:     "windows",
				lookPath: map[string]bool{"go": true},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			setExecFakes(t, tc.fakes)
			info := (&GoAdapter{}).Info()
			if info.UpdatePolicy != adapters.PolicyGated {
				t.Errorf("UpdatePolicy = %v, want PolicyGated", info.UpdatePolicy)
			}
			if info.Command != tc.wantCmd {
				t.Errorf("Command = %q, want %q", info.Command, tc.wantCmd)
			}
		})
	}
}
